package ocs

import (
	"context"
	"fmt"
	"math"

	"github.com/docker/go-units"
	"github.com/kelseyhightower/envconfig"
	"github.com/openshift/assisted-service/internal/common"
	"github.com/openshift/assisted-service/internal/operators/api"
	"github.com/openshift/assisted-service/internal/operators/lso"
	"github.com/openshift/assisted-service/models"
	"github.com/sirupsen/logrus"
)

// ocsOperator is an OCS OLM operator plugin; it implements api.Operator
type ocsOperator struct {
	log                logrus.FieldLogger
	ocsValidatorConfig Config
	ocsValidator       OCSValidator
}

var Operator models.MonitoredOperator = models.MonitoredOperator{
	Name:           "ocs",
	OperatorType:   models.OperatorTypeOlm,
	TimeoutSeconds: 30 * 60,
}

// NewOcsOperator creates new OCSOperator
func NewOcsOperator(log logrus.FieldLogger) *ocsOperator {
	cfg := Config{}
	err := envconfig.Process("myapp", &cfg)
	if err != nil {
		log.Fatal(err.Error())
	}
	validator := NewOCSValidator(log.WithField("pkg", "ocs-operator-state"), &cfg)
	return NewOcsOperatorWithConfig(log, cfg, validator)
}

// NewOcsOperatorWithConfig creates new OCSOperator with given configuration and validator
func NewOcsOperatorWithConfig(log logrus.FieldLogger, config Config, validator OCSValidator) *ocsOperator {
	return &ocsOperator{
		log:                log,
		ocsValidatorConfig: config,
		ocsValidator:       validator,
	}
}

// GetName reports the name of an operator this Operator manages
func (o *ocsOperator) GetName() string {
	return Operator.Name
}

// GetDependencies provides a list of dependencies of the Operator
func (o *ocsOperator) GetDependencies() []string {
	return []string{lso.Operator.Name}
}

// GetClusterValidationID returns cluster validation ID for the Operator
func (o *ocsOperator) GetClusterValidationID() string {
	return string(models.ClusterValidationIDOcsRequirementsSatisfied)
}

// GetHostValidationID returns host validation ID for the Operator
func (o *ocsOperator) GetHostValidationID() string {
	return string(models.HostValidationIDOcsRequirementsSatisfied)
}

// ValidateCluster verifies whether this operator is valid for given cluster
func (o *ocsOperator) ValidateCluster(_ context.Context, cluster *common.Cluster) (api.ValidationResult, error) {
	status, message := o.ocsValidator.ValidateRequirements(&cluster.Cluster)

	return api.ValidationResult{Status: status, ValidationId: o.GetClusterValidationID(), Reasons: []string{message}}, nil
}

// ValidateHost verifies whether this operator is valid for given host
func (o *ocsOperator) ValidateHost(context.Context, *common.Cluster, *models.Host) (api.ValidationResult, error) {
	return api.ValidationResult{Status: api.Success, ValidationId: o.GetHostValidationID(), Reasons: []string{}}, nil
}

// GenerateManifests generates manifests for the operator
func (o *ocsOperator) GenerateManifests(cluster *common.Cluster) (*api.Manifests, error) {
	manifests, err := Manifests(o.ocsValidatorConfig.OCSMinimalDeployment, o.ocsValidatorConfig.OCSDisksAvailable, len(cluster.Cluster.Hosts))
	return &api.Manifests{Files: manifests}, err
}

// GetCPURequirementForWorker provides worker CPU requirements for the operator
func (o *ocsOperator) GetCPURequirementForWorker(cluster *common.Cluster) (int64, error) {
	return o.getPerHostCPURequirement(cluster)
}

// GetCPURequirementForMaster provides master CPU requirements for the operator
func (o *ocsOperator) GetCPURequirementForMaster(cluster *common.Cluster) (int64, error) {
	return o.getPerHostCPURequirement(cluster)
}

// GetMemoryRequirementForWorker provides worker memory requirements for the operator in MB
func (o *ocsOperator) GetMemoryRequirementForWorker(cluster *common.Cluster) (int64, error) {
	return o.getPerHostMemoryRequirement(cluster)
}

// GetMemoryRequirementForMaster provides master memory requirements for the operator
func (o *ocsOperator) GetMemoryRequirementForMaster(cluster *common.Cluster) (int64, error) {
	return o.getPerHostMemoryRequirement(cluster)
}

// getHostMemoryRequirement returns the minimum amount of memory in Bytes the host must have to install the OCS operator
func (o *ocsOperator) getPerHostMemoryRequirement(cluster *common.Cluster) (int64, error) {
	depType, err := o.ocsValidator.IdentifyDeploymentTypeForResource(&cluster.Cluster, memory)
	if err != nil {
		return 0, err
	}
	var mem, count int64

	switch depType {
	case compact:
		mem = o.ocsValidatorConfig.OCSRequiredCompactModeRAMGB * int64(units.GB)
		count = 3 // in compact mode we deploy only with 3 masters
	case minimal, standard:
		mem = o.ocsValidatorConfig.OCSMinimumRAMGB * int64(units.GB)
		count = o.getWorkerHostsCount(cluster) // OCS only deploys only in workers when not in compact mode.
	default:
		return 0, fmt.Errorf("invalid OCS deployment type %v", depType)
	}
	if count == 0 {
		return 0, fmt.Errorf("unable to find hosts with valid inventory")
	}
	return mem / count, nil
}

// getHostCPURequirement returns the minimum amount of CPU core count the cluster must have to install the OCS operator
func (o *ocsOperator) getPerHostCPURequirement(cluster *common.Cluster) (int64, error) {
	depType, err := o.ocsValidator.IdentifyDeploymentTypeForResource(&cluster.Cluster, cpu)
	if err != nil {
		return 0, err
	}
	var cpu, count int64
	switch depType {
	case compact:
		cpu = o.ocsValidatorConfig.OCSRequiredCompactModeCPUCount
		count = 3 // in compact mode we deploy only with 3 masters
	case minimal, standard:
		cpu = o.ocsValidatorConfig.OCSMinimumCPUCount
		count = o.getWorkerHostsCount(cluster) // OCS only deploys in workers when not in compact mode.
	default:
		return 0, fmt.Errorf("invalid OCS deployment type %v", depType)
	}
	if count == 0 {
		return 0, fmt.Errorf("unable to find workers with valid inventory")
	}
	t := float64(cpu / count)
	r := math.Round(t)
	if r < t { // round up the number of cores
		return int64(t) + 1, nil
	}
	return int64(r), nil
}

//getWorkerHostsCount returns the number of worker hosts or hosts with role type autoassign
func (o *ocsOperator) getWorkerHostsCount(cluster *common.Cluster) int64 {
	var hosts int64
	for _, h := range cluster.Hosts {
		if h.Role == models.HostRoleWorker {
			hosts++
		}
	}
	return hosts
}

// GetProperties provides description of operator properties: none required
func (o *ocsOperator) GetProperties() models.OperatorProperties {
	return models.OperatorProperties{}
}
