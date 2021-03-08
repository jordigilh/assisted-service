package network

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/go-openapi/swag"
	"github.com/openshift/assisted-service/internal/common"
	"github.com/openshift/assisted-service/models"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/thoas/go-funk"
)

//go:generate mockgen -source=manifests_generator.go -package=network -destination=mock_manifests_generator.go

const dnsmasqManifestFileName = "dnsmasq-bootstrap-in-place.yaml"

type ManifestGeneratorAPI interface {
	AddDnsmasqForSingleNode(ctx context.Context, cluster *common.Cluster) (map[string][]byte, error)
	AddChronyManifest(ctx context.Context, cluster *common.Cluster) (map[string][]byte, error)
}

type manifestsGenerator struct {
	log logrus.FieldLogger
}

func NewManifestsGenerator(log logrus.FieldLogger) ManifestGeneratorAPI {
	return &manifestsGenerator{log: log}
}

func (m *manifestsGenerator) AddChronyManifest(ctx context.Context, cluster *common.Cluster) (map[string][]byte, error) {
	manifests := map[string][]byte{}
	for _, role := range []models.HostRole{models.HostRoleMaster, models.HostRoleWorker} {
		content, err := m.createChronyManifestContent(cluster, role)

		if err != nil {
			return nil, errors.Wrapf(err, "Failed to create chrony manifest content for role %s cluster id %s", role, *cluster.ID)
		}

		manifests[fmt.Sprintf("%ss-chrony-configuration.yaml", string(role))] = content
	}

	return manifests, nil
}

func (m *manifestsGenerator) AddDnsmasqForSingleNode(ctx context.Context, cluster *common.Cluster) (map[string][]byte, error) {
	manifests := map[string][]byte{}
	content, err := m.createDnsmasqForSingleNode(cluster)
	if err != nil {
		m.log.WithError(err).Errorf("Failed to create dnsmasq manifest")
		return nil, err
	}
	manifests[dnsmasqManifestFileName] = content
	return manifests, nil
}

func (m *manifestsGenerator) createDnsmasqForSingleNode(cluster *common.Cluster) ([]byte, error) {
	bootstrap := common.GetBootstrapHost(cluster)
	if bootstrap == nil {
		return nil, errors.Errorf("no bootstap host were found in cluter")
	}

	cidr := GetMachineCidrForUserManagedNetwork(cluster, m.log)
	cluster.MachineNetworkCidr = cidr
	hostIP, err := getMachineCIDRObj(bootstrap, cluster, "ip")
	if hostIP == "" || err != nil {
		msg := "failed to get ip for bootstrap in place dnsmasq manifest"
		if err != nil {
			msg = errors.Wrapf(err, msg).Error()
		}
		return nil, errors.Errorf(msg)
	}

	var manifestParams = map[string]string{
		"CLUSTER_NAME": cluster.Cluster.Name,
		"DNS_DOMAIN":   cluster.Cluster.BaseDNSDomain,
		"HOST_IP":      hostIP,
	}

	m.log.Infof("Creating dnsmasq manifest with values: cluster name: %q, domain - %q, host ip - %q",
		cluster.Cluster.Name, cluster.Cluster.BaseDNSDomain, hostIP)

	content, err := m.fillTemplate(manifestParams, snoDnsmasqConf)
	if err != nil {
		return nil, err
	}
	dnsmasqContent := base64.StdEncoding.EncodeToString(content)

	content, err = m.fillTemplate(manifestParams, forceDnsDispatcherScript)
	if err != nil {
		return nil, err
	}
	forceDnsDispatcherScriptContent := base64.StdEncoding.EncodeToString(content)

	manifestParams = map[string]string{
		"DNSMASQ_CONTENT":  dnsmasqContent,
		"FORCE_DNS_SCRIPT": forceDnsDispatcherScriptContent,
	}

	content, err = m.fillTemplate(manifestParams, dnsMachineConfigManifest)
	if err != nil {
		return nil, err
	}

	return content, nil
}

func (m *manifestsGenerator) fillTemplate(manifestParams map[string]string, templateData string) ([]byte, error) {
	tmpl, err := template.New("template").Parse(templateData)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create template")
	}
	buf := &bytes.Buffer{}
	if err = tmpl.Execute(buf, manifestParams); err != nil {
		m.log.WithError(err).Errorf("Failed to set manifest params %v to template", manifestParams)
		return nil, err
	}
	return buf.Bytes(), nil
}

func (m *manifestsGenerator) createChronyManifestContent(c *common.Cluster, role models.HostRole) ([]byte, error) {
	sources := make([]string, 0)

	for _, host := range c.Hosts {
		if swag.StringValue(host.Status) == models.HostStatusDisabled || host.NtpSources == "" {
			continue
		}

		var ntpSources []*models.NtpSource
		if err := json.Unmarshal([]byte(host.NtpSources), &ntpSources); err != nil {
			m.log.Errorln(err)
			return nil, errors.Wrapf(err, "Failed to unmarshal %s", host.NtpSources)
		}

		for _, source := range ntpSources {
			if source.SourceState == models.SourceStateSynced {
				if !funk.Contains(sources, source.SourceName) {
					sources = append(sources, source.SourceName)
				}
			}
		}
	}

	content := defaultChronyConf[:]
	for _, source := range sources {
		content += fmt.Sprintf("\nserver %s iburst", source)
	}
	var manifestParams = map[string]string{
		"CHRONY_CONTENT": base64.StdEncoding.EncodeToString([]byte(content)),
		"ROLE":           string(role),
	}
	return m.fillTemplate(manifestParams, ntpMachineConfigManifest)
}

//AddOLMOperators is responsible of handling the manifest generation for the OLM operators in the given cluster
// func (m *ManifestsGenerator) AddOLMOperators(ctx context.Context, cluster *common.Cluster) error {
// 	manifests, err := m.operatorsAPI.GenerateManifests(cluster)
// 	if err != nil {
// 		return errors.Wrapf(err, "Failed to generate the OLM operators' manifests in cluster id %s", cluster.ID)
// 	}
// 	for filename, content := range manifests {
// 		err = m.createManifests(ctx, cluster, filename, []byte(content))
// 		if err != nil {
// 			return err
// 		}
// 	}

// 	return nil
// }

const defaultChronyConf = `
driftfile /var/lib/chrony/drift
makestep 1.0 3
rtcsync
logdir /var/log/chrony`

const ntpMachineConfigManifest = `
apiVersion: machineconfiguration.openshift.io/v1
kind: MachineConfig
metadata:
  labels:
    machineconfiguration.openshift.io/role: {{.ROLE}}
  name: {{.ROLE}}s-chrony-configuration
spec:
  config:
    ignition:
      config: {}
      security:
        tls: {}
      timeouts: {}
      version: 2.2.0
    networkd: {}
    passwd: {}
    storage:
      files:
      - contents:
          source: data:text/plain;charset=utf-8;base64,{{.CHRONY_CONTENT}}
          verification: {}
        filesystem: root
        mode: 420
        path: /etc/chrony.conf
  osImageURL: ""
`

const snoDnsmasqConf = `
address=/apps.{{.CLUSTER_NAME}}.{{.DNS_DOMAIN}}/{{.HOST_IP}}
address=/api-int.{{.CLUSTER_NAME}}.{{.DNS_DOMAIN}}/{{.HOST_IP}}
`

const forceDnsDispatcherScript = `
export IP="{{.HOST_IP}}"
if [ "$2" = "dhcp4-change" ] || [ "$2" = "dhcp6-change" ] || [ "$2" = "up" ] || [ "$2" = "connectivity-change" ]; then
    if ! grep -q "$IP" /etc/resolv.conf; then
      sed -i "s/{{.CLUSTER_NAME}}.{{.DNS_DOMAIN}}//" /etc/resolv.conf
      sed -i "s/search /search {{.CLUSTER_NAME}}.{{.DNS_DOMAIN}} /" /etc/resolv.conf
      sed -i "0,/nameserver/s/nameserver/nameserver $IP\nnameserver/" /etc/resolv.conf
    fi
fi
`

const dnsMachineConfigManifest = `
apiVersion: machineconfiguration.openshift.io/v1
kind: MachineConfig
metadata:
  labels:
    machineconfiguration.openshift.io/role: master
  name: 99-master-dnsmasq-configuration
spec:
  config:
    ignition:
      config: {}
      security:
        tls: {}
      timeouts: {}
      version: 2.2.0
    networkd: {}
    passwd: {}
    storage:
      files:
      - contents:
          source: data:text/plain;charset=utf-8;base64,{{.DNSMASQ_CONTENT}}
          verification: {}
        filesystem: root
        mode: 420
        path: /etc/dnsmasq.d/single-node.conf
      - contents:
          source: data:text/plain;charset=utf-8;base64,{{.FORCE_DNS_SCRIPT}}
          verification: {}
        filesystem: root
        mode: 365
        path: /etc/NetworkManager/dispatcher.d/forcedns
    systemd:
      units:
      - name: dnsmasq.service
        enabled: true
        contents: |
         [Unit]
         Description=Run dnsmasq to provide local dns for Singe Node OpenShift
         Before=kubelet.service crio.service
         After=network.target
         
         [Service]
         ExecStart=/usr/sbin/dnsmasq -k
         
         [Install]
         WantedBy=multi-user.target
`
