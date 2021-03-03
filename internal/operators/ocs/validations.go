package ocs

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/alecthomas/units"
	"github.com/openshift/assisted-service/internal/operators/api"
	"github.com/openshift/assisted-service/models"
	"github.com/sirupsen/logrus"
)

//go:generate mockgen -source=validations.go -package=ocs -destination=mock_validations.go
type OCSValidator interface {
	ValidateRequirements(cluster *models.Cluster) (api.ValidationStatus, string)
	IdentifyDeploymentTypeForResource(cluster *models.Cluster, res resource) (ocsDeploymentType, error)
}

type ocsValidator struct {
	*Config
	log logrus.FieldLogger
}

func NewOCSValidator(log logrus.FieldLogger, cfg *Config) OCSValidator {
	return &ocsValidator{
		log:    log,
		Config: cfg,
	}
}

type Config struct {
	OCSMinimumCPUCount             int64  `envconfig:"OCS_MINIMUM_CPU_COUNT" default:"18"`
	OCSMinimumRAMGB                int64  `envconfig:"OCS_MINIMUM_RAM_GB" default:"51"` // disable deployment if less
	OCSRequiredDisk                int64  `envconfig:"OCS_MINIMUM_DISK" default:"3"`
	OCSRequiredDiskCPUCount        int64  `envconfig:"OCS_REQUIRED_DISK_CPU_COUNT" default:"2"` // each disk requires 2 cpus
	OCSRequiredDiskRAMGB           int64  `envconfig:"OCS_REQUIRED_DISK_RAM_GB" default:"5"`    // each disk requires 5GB ram
	OCSRequiredHosts               int64  `envconfig:"OCS_MINIMUM_HOST" default:"3"`
	OCSRequiredCPUCount            int64  `envconfig:"OCS_REQUIRED_CPU_COUNT" default:"24"`              // Standard Mode 8*3
	OCSRequiredRAMGB               int64  `envconfig:"OCS_REQUIRED_RAM_GB" default:"57"`                 // Standard Mode
	OCSRequiredCompactModeCPUCount int64  `envconfig:"OCS_REQUIRED_COMAPCT_MODE_CPU_COUNT" default:"30"` // Compact Mode cpu requirements for 3 nodes( 4*3(master)+6*3(OCS) cpus)
	OCSRequiredCompactModeRAMGB    int64  `envconfig:"OCS_REQUIRED_COMAPCT_MODE_RAM_GB" default:"81"`    //Compact Mode ram requirements (8*3(master)+57(OCS) GB)
	OCSMasterWorkerHosts           int    `envconfig:"OCS_REQUIRED_MASTER_WORKER_HOSTS" default:"6"`
	OCSMinimalDeployment           bool   `envconfig:"OCS_MINIMAL_DEPLOYMENT" default:"false"`
	OCSDisksAvailable              int64  `envconfig:"OCS_DISKS_AVAILABLE" default:"3"`
	OCSDeploymentType              string `envconfig:"OCS_DEPLOYMENT_TYPE" default:"None"`
}

type resourceAvailability struct {
	usableMemory  int64 // count the total available RAM on the cluster
	cpuCount      int64 //count the total CPUs on the cluster
	diskCount     int64 // count the total disks on all the hosts
	hostsWithDisk int64 // to determine total number of hosts with disks. OCS requires atleast 3 hosts with non-bootable disks
}

func (o *ocsValidator) IdentifyDeploymentTypeForResource(cluster *models.Cluster, res resource) (ocsDeploymentType, error) {
	hosts := cluster.Hosts

	resourceAvailability, insufficientHosts, _, err := o.gatherResourceAvailability(cluster)
	if err != nil {
		return none, err
	}
	if len(insufficientHosts) > 0 {
		var msg []string
		for _, h := range insufficientHosts {
			if h.isMemory && res == memory {
				msg = append(msg, h.status)
			} else if h.isCPU && res == cpu {
				msg = append(msg, h.status)
			}
		}
		if len(msg) > 0 {
			return none, fmt.Errorf("Insufficient resources: %s", strings.Join(msg, ","))
		}
	}
	if int64(len(hosts)) == o.OCSRequiredHosts { // for 3 hosts
		return compact, nil
	}

	if res == memory && resourceAvailability.usableMemory < gbToBytes(o.OCSMinimumRAMGB) {
		return none, fmt.Errorf("Insufficient memory to deploy OCS in %s. A minimum usable memory of %dGB is needed", minimal, o.OCSMinimumRAMGB)
	}
	if res == cpu && resourceAvailability.cpuCount < o.OCSMinimumCPUCount {
		return none, fmt.Errorf("Insufficient CPU count to deploy OCS in %s. A minimum of %d cores is needed", minimal, o.OCSMinimumCPUCount)
	}
	if (resourceAvailability.cpuCount >= o.OCSMinimumCPUCount && resourceAvailability.cpuCount < o.OCSRequiredCPUCount) ||
		(resourceAvailability.usableMemory >= gbToBytes(o.OCSMinimumRAMGB) && resourceAvailability.usableMemory < gbToBytes(o.OCSRequiredRAMGB)) {
		return minimal, nil
	}
	return standard, nil
}

func (o *ocsValidator) gatherResourceAvailability(cluster *models.Cluster) (*resourceAvailability, []insufficientHost, bool, error) {
	hosts := cluster.Hosts
	if int64(len(hosts)) < o.OCSRequiredHosts {
		return nil, nil, false, errors.New("Insufficient hosts to deploy OCS. A minimum of 3 hosts is required to deploy OCS. ")
	}
	r := resourceAvailability{}
	var insufficientHosts []insufficientHost
	if int64(len(hosts)) == o.OCSRequiredHosts { // for only 3 hosts, we need to install OCS in compact mode
		var mc int // counter of master hosts or hosts with autoassign in the cluster
		for _, host := range hosts {
			inventoryMissing, err := o.nodeResources(host, &r, &insufficientHosts)
			if err != nil {
				o.log.Fatal("Error occured while calculating Node requirements ", err)
				return nil, nil, false, fmt.Errorf("error occured while calculating node requirements: %W", err)
			}
			if inventoryMissing {
				return nil, nil, true, fmt.Errorf("Missing Inventory in some of the hosts")
			}
			if host.Role == models.HostRoleMaster || host.Role == models.HostRoleAutoAssign {
				mc++
			}
		}
		if mc != 3 {
			return nil, nil, false, fmt.Errorf("Insufficient hosts to deploy OCS: Exactly 3 master hosts are required to deploy OCS in compact mode")
		}
		return &r, insufficientHosts, false, nil
	}
	if len(hosts) < o.OCSMasterWorkerHosts { // not supporting OCS installation for 2 Workers and 3 Masters
		return nil, nil, false, fmt.Errorf("Not supporting OCS Installation for 3 Masters and 2 Workers")
	}
	for _, host := range hosts { // if the worker nodes >=3 , install OCS on all the worker nodes if they satisfy OCS requirements
		/* If the Role is set to Auto-assign for a host, it is not possible to determine whether the node will end up as a master or worker node.
		As OCS will use only worker nodes for non-compact deployments, the OCS validations cannot be performed as it cannot know which nodes will be worker nodes.
		We ignore the role check for a cluster of 3 nodes as they will all be master nodes. OCS validations will proceed as for a compact deployment.
		*/
		if host.Role == models.HostRoleAutoAssign {
			return nil, nil, false, fmt.Errorf("All host roles must be assigned to enable OCS.")
		}
		if host.Role == models.HostRoleWorker {
			inventoryMissing, err := o.nodeResources(host, &r, &insufficientHosts)
			if err != nil {
				o.log.Fatal("Error occured while calculating node requirements ", err)
				return nil, nil, false, fmt.Errorf("error occured while calculating node requirements: %W", err)
			}
			if inventoryMissing {
				return nil, nil, true, fmt.Errorf("Missing Inventory in some of the hosts")
			}
		}
	}
	return &r, insufficientHosts, false, nil
}

// ValidateOCSRequirements is used to validate min requirements of OCS
func (o *ocsValidator) ValidateRequirements(cluster *models.Cluster) (api.ValidationStatus, string) {
	var status string
	hosts := cluster.Hosts

	resourceAvailability, insufficientHosts, pendingStatus, err := o.gatherResourceAvailability(cluster)
	if err != nil {
		if pendingStatus {
			return api.Pending, err.Error()
		}
		if u := errors.Unwrap(err); u != nil {
			return api.Failure, u.Error()
		}
		return api.Failure, err.Error()
	}

	if len(insufficientHosts) > 0 {
		for _, h := range insufficientHosts {
			status = status + h.status + ".\n"
		}
		o.log.Info("Validate Requirements status ", status)
		return api.Failure, status
	}

	// total disks excluding boot disk must be a multiple of 3
	if resourceAvailability.diskCount%3 != 0 {
		status = "Total disks on the cluster must be a multiple of 3"
		o.log.Info(status)
		return api.Failure, status
	}

	// this will be used to set count of StorageDevices in StorageCluster manifest
	o.OCSDisksAvailable = resourceAvailability.diskCount
	canDeployOCS, status := o.validate(hosts, resourceAvailability)

	o.log.Info(status)

	if canDeployOCS {
		return api.Success, status
	}
	return api.Failure, status
}

type insufficientHost struct {
	status          string
	isMemory, isCPU bool
}

func (o *ocsValidator) nodeResources(host *models.Host, resourceAvailability *resourceAvailability, insufficientHosts *[]insufficientHost) (bool, error) {
	var inventory models.Inventory
	// if inventory is empty
	if host.Inventory == "" {
		o.log.Info("Empty Inventory of host with hostID ", *host.ID)
		return true, nil // to indicate that inventory is empty and the ValidationStatus must be Pending
	}
	if err := json.Unmarshal([]byte(host.Inventory), &inventory); err != nil {
		o.log.Errorf("Failed to get inventory from host with id %s", host.ID)
		return false, err
	}

	disks := getValidDiskCount(inventory.Disks)

	if disks > 1 { // OCS must use the non-boot disks
		requiredDiskCPU := (disks - 1) * o.OCSRequiredDiskCPUCount
		requiredDiskRAM := (disks - 1) * o.OCSRequiredDiskRAMGB

		resourceAvailability.diskCount += disks - 1 // not counting the boot disk
		resourceAvailability.hostsWithDisk++

		if inventory.CPU.Count < requiredDiskCPU || inventory.Memory.UsableBytes < gbToBytes(requiredDiskRAM) {
			status := fmt.Sprint("Insufficient resources on host with host ID ", *host.ID, " to deploy OCS. The hosts has ", disks, " disks that require ", requiredDiskCPU, " CPUs, ", requiredDiskRAM, " RAMGB")

			i := insufficientHost{}
			if inventory.CPU.Count < requiredDiskCPU {
				i.isCPU = true
			}
			if inventory.Memory.UsableBytes < gbToBytes(requiredDiskRAM) {
				i.isMemory = true
			}
			i.status = status
			*insufficientHosts = append(*insufficientHosts, i)
			o.log.Info(status)
			return false, nil
		}
		resourceAvailability.cpuCount += (inventory.CPU.Count - requiredDiskCPU)                         // cpus excluding the cpus required for disks
		resourceAvailability.usableMemory += (inventory.Memory.UsableBytes - gbToBytes(requiredDiskRAM)) // ram excluding the ram required for disks
		return false, nil
	}
	resourceAvailability.cpuCount += inventory.CPU.Count
	resourceAvailability.usableMemory += inventory.Memory.UsableBytes
	return false, nil
}

func gbToBytes(gb int64) int64 {
	return gb * int64(units.GB)
}

// used to validate resource requirements for OCS excluding disk requirements and set a status message
func (o *ocsValidator) validate(hosts []*models.Host, resourceAvailability *resourceAvailability) (bool, string) {
	var TotalCPUs int64
	var TotalRAM int64
	var status string
	if int64(len(hosts)) == o.OCSRequiredHosts { // for 3 hosts
		TotalCPUs = o.OCSRequiredCompactModeCPUCount
		TotalRAM = o.OCSRequiredCompactModeRAMGB
		if resourceAvailability.cpuCount < TotalCPUs || resourceAvailability.usableMemory < gbToBytes(TotalRAM) || resourceAvailability.diskCount < o.OCSRequiredDisk || resourceAvailability.hostsWithDisk < o.OCSRequiredHosts { // check for master nodes requirements
			status = o.setStatusInsufficientResources(resourceAvailability, compact)
			return false, status
		}
		o.OCSDeploymentType = "Compact"
		status = "OCS Requirements for Compact Mode are satisfied"
		return true, status
	}
	TotalCPUs = o.OCSMinimumCPUCount
	TotalRAM = o.OCSMinimumRAMGB
	if resourceAvailability.diskCount < o.OCSRequiredDisk || resourceAvailability.cpuCount < TotalCPUs || resourceAvailability.usableMemory < gbToBytes(TotalRAM) || resourceAvailability.hostsWithDisk < o.OCSRequiredHosts { // check for worker nodes requirements
		status = o.setStatusInsufficientResources(resourceAvailability, minimal)
		return false, status
	}

	TotalCPUs = o.OCSRequiredCPUCount
	TotalRAM = o.OCSRequiredRAMGB
	if resourceAvailability.cpuCount < TotalCPUs || resourceAvailability.usableMemory < gbToBytes(TotalRAM) { // conditions for minimal deployment
		status = "Requirements for OCS Minimal Deployment are satisfied"
		o.OCSDeploymentType = "Minimal"
		o.OCSMinimalDeployment = true
		return true, status
	}

	status = "OCS Requirements for Standard Deployment are satisfied"
	o.OCSDeploymentType = "Standard"
	return true, status
}

func (o *ocsValidator) setStatusInsufficientResources(resourceAvailability *resourceAvailability, mode ocsDeploymentType) string {
	var TotalCPUs int64
	var TotalRAMGB int64
	if mode == compact {
		TotalCPUs = o.OCSRequiredCompactModeCPUCount
		TotalRAMGB = o.OCSRequiredCompactModeRAMGB
	} else {
		TotalCPUs = o.OCSMinimumCPUCount
		TotalRAMGB = o.OCSMinimumRAMGB
	}
	status := fmt.Sprint("Insufficient Resources to deploy OCS in ", mode, ". A minimum of ")
	if resourceAvailability.cpuCount < TotalCPUs {
		status = status + fmt.Sprint(TotalCPUs, " CPUs, excluding disk CPU resources ")
	}
	if resourceAvailability.usableMemory < gbToBytes(TotalRAMGB) {
		status = status + fmt.Sprint(TotalRAMGB, " RAM, excluding disk RAM resources ")
	}
	if resourceAvailability.diskCount < o.OCSRequiredDisk {
		status = status + fmt.Sprint(o.OCSRequiredDisk, " Disks, ")
	}
	if resourceAvailability.hostsWithDisk < o.OCSRequiredHosts {
		status = status + fmt.Sprint(o.OCSRequiredHosts, " Hosts with disks, ")
	}
	status = status + "is required."

	return status

}

//getValidDiskCount count all disks of drive type ssd or hdd
func getValidDiskCount(disks []*models.Disk) int64 {
	var countDisks int64
	for _, disk := range disks {
		if disk.DriveType == ssdDrive || disk.DriveType == hdDrive {
			countDisks++
		}
	}
	return countDisks
}
