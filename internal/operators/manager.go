package operators

import (
	"container/list"
	"context"
	"fmt"

	"github.com/openshift/assisted-service/internal/common"
	"github.com/openshift/assisted-service/internal/operators/api"
	"github.com/openshift/assisted-service/models"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/thoas/go-funk"
)

// Manager is responsible for performing operations against additional operators
type Manager struct {
	log                logrus.FieldLogger
	olmOperators       map[string]api.Operator
	monitoredOperators map[string]*models.MonitoredOperator
}

// API defines Operator management operation
//go:generate mockgen -package=operators -destination=mock_operators_api.go . API
type API interface {
	// ValidateCluster validates cluster requirements
	ValidateCluster(ctx context.Context, cluster *common.Cluster) ([]api.ValidationResult, error)
	// ValidateHost validates host requirements
	ValidateHost(ctx context.Context, cluster *common.Cluster, host *models.Host) ([]api.ValidationResult, error)
	// GenerateManifests generates manifests for all enabled operators.
	// Returns map assigning manifest content to its desired file name
	GenerateManifests(cluster *common.Cluster) (map[string]string, error)
	// AnyOLMOperatorEnabled checks whether any OLM operator has been enabled for the given cluster
	AnyOLMOperatorEnabled(cluster *common.Cluster) bool
	// UpdateDependencies amends the list of requested additional operators with any missing dependencies
	UpdateDependencies(cluster *common.Cluster) error
	// GetMonitoredOperatorsList returns the monitored operators available by the manager.
	GetMonitoredOperatorsList() map[string]*models.MonitoredOperator
	// GetOperatorByName the manager's supported operator object by name.
	GetOperatorByName(operatorName string) (*models.MonitoredOperator, error)
	// GetSupportedOperatorsByType returns the manager's supported operator objects by type.
	GetSupportedOperatorsByType(operatorType models.OperatorType) []*models.MonitoredOperator
	// GetSupportedOperators returns a list of OLM operators that are supported
	GetSupportedOperators() []string
	// GetOperatorProperties provides description of properties of an operator
	GetOperatorProperties(operatorName string) (models.OperatorProperties, error)
	// GetCPURequirementForWorker provides worker CPU requirements for the operator
	GetCPURequirementForRole(cluster *common.Cluster, role models.HostRole) (int64, error)
	// GetMemoryRequirementForWorker provides worker memory requirements for the operator in Bytes
	GetMemoryRequirementForRole(cluster *common.Cluster, role models.HostRole) (int64, error)
}

// GenerateManifests generates manifests for all enabled operators.
// Returns map assigning manifest content to its desired file name
func (mgr *Manager) GenerateManifests(cluster *common.Cluster) (map[string]string, error) {
	// TODO: cluster should already contain up-to-date list of operators - implemented here for now to replicate
	// the original behaviour
	err := mgr.UpdateDependencies(cluster)
	if err != nil {
		return nil, err
	}

	operatorManifests := make(map[string]string)

	// Generate manifests for all the generic operators
	for _, clusterOperator := range cluster.MonitoredOperators {
		if clusterOperator.OperatorType != models.OperatorTypeOlm {
			continue
		}

		operator := mgr.olmOperators[clusterOperator.Name]
		if operator != nil {
			manifests, err := operator.GenerateManifests(cluster)
			if err != nil {
				mgr.log.Error(fmt.Sprintf("Cannot generate %s manifests due to ", clusterOperator.Name), err)
				return nil, err
			}
			if manifests != nil {
				for k, v := range manifests.Files {
					operatorManifests[k] = v
				}
			}
		}
	}

	return operatorManifests, nil
}

// AnyOLMOperatorEnabled checks whether any OLM operator has been enabled for the given cluster
func (mgr *Manager) AnyOLMOperatorEnabled(cluster *common.Cluster) bool {
	for _, operator := range mgr.olmOperators {
		if IsEnabled(cluster.MonitoredOperators, operator.GetName()) {
			return true
		}
	}
	return false
}

// ValidateHost validates host requirements
func (mgr *Manager) ValidateHost(ctx context.Context, cluster *common.Cluster, host *models.Host) ([]api.ValidationResult, error) {
	// TODO: cluster should already contain up-to-date list of operators - implemented here for now to replicate
	// the original behaviour
	err := mgr.UpdateDependencies(cluster)
	if err != nil {
		return nil, err
	}

	results := make([]api.ValidationResult, 0, len(mgr.olmOperators))

	// To track operators that are disabled or not present in the cluster configuration, but have to be present
	// in the validation results and marked as valid.
	pendingOperators := make(map[string]struct{})
	for k := range mgr.olmOperators {
		pendingOperators[k] = struct{}{}
	}

	for _, clusterOperator := range cluster.MonitoredOperators {
		if clusterOperator.OperatorType != models.OperatorTypeOlm {
			continue
		}

		operator := mgr.olmOperators[clusterOperator.Name]
		var result api.ValidationResult
		if operator != nil {
			result, err = operator.ValidateHost(ctx, cluster, host)
			if err != nil {
				return nil, err
			}
			delete(pendingOperators, clusterOperator.Name)
			results = append(results, result)
		}
	}
	// Add successful validation result for disabled operators
	for OpName := range pendingOperators {
		operator := mgr.olmOperators[OpName]
		result := api.ValidationResult{
			Status:       api.Success,
			ValidationId: operator.GetHostValidationID(),
			Reasons: []string{
				fmt.Sprintf("%s is disabled", OpName),
			},
		}
		results = append(results, result)
	}
	return results, nil
}

// ValidateCluster validates cluster requirements
func (mgr *Manager) ValidateCluster(ctx context.Context, cluster *common.Cluster) ([]api.ValidationResult, error) {
	// TODO: cluster should already contain up-to-date list of operators - implemented here for now to replicate
	// the original behaviour
	err := mgr.UpdateDependencies(cluster)
	if err != nil {
		return nil, err
	}

	results := make([]api.ValidationResult, 0, len(mgr.olmOperators))

	pendingOperators := make(map[string]struct{})
	for k := range mgr.olmOperators {
		pendingOperators[k] = struct{}{}
	}

	for _, clusterOperator := range cluster.MonitoredOperators {
		if clusterOperator.OperatorType != models.OperatorTypeOlm {
			continue
		}

		operator := mgr.olmOperators[clusterOperator.Name]
		var result api.ValidationResult
		if operator != nil {
			result, err = operator.ValidateCluster(ctx, cluster)
			if err != nil {
				return nil, err
			}
			delete(pendingOperators, clusterOperator.Name)
			results = append(results, result)
		}
	}
	// Add successful validation result for disabled operators
	for opName := range pendingOperators {
		operator := mgr.olmOperators[opName]
		result := api.ValidationResult{
			Status:       api.Success,
			ValidationId: operator.GetClusterValidationID(),
			Reasons: []string{
				fmt.Sprintf("%s is disabled", opName),
			},
		}
		results = append(results, result)
	}
	return results, nil
}

// UpdateDependencies amends the list of requested additional operators with any missing dependencies
func (mgr *Manager) UpdateDependencies(cluster *common.Cluster) error {
	operators, err := mgr.resolveDependencies(cluster.MonitoredOperators)
	if err != nil {
		return err
	}

	cluster.MonitoredOperators = operators
	return nil
}

// GetSupportedOperators returns a list of OLM operators that are supported
func (mgr *Manager) GetSupportedOperators() []string {
	keys := make([]string, 0, len(mgr.olmOperators))
	for k := range mgr.olmOperators {
		keys = append(keys, k)
	}
	return keys
}

// GetOperatorProperties provides description of properties of an operator
func (mgr *Manager) GetOperatorProperties(operatorName string) (models.OperatorProperties, error) {
	if operator, ok := mgr.olmOperators[operatorName]; ok {
		return operator.GetProperties(), nil
	}
	return nil, errors.Errorf("Operator %s not found", operatorName)
}

func (mgr *Manager) resolveDependencies(operators []*models.MonitoredOperator) ([]*models.MonitoredOperator, error) {
	allDependentOperators := mgr.getDependencies(operators)

	inputOperatorNames := make([]string, len(operators))
	for _, inputOperator := range operators {
		inputOperatorNames = append(inputOperatorNames, inputOperator.Name)
	}

	for operatorName := range allDependentOperators {
		if funk.Contains(inputOperatorNames, operatorName) {
			continue
		}

		operator, err := mgr.GetOperatorByName(operatorName)
		if err != nil {
			return nil, err
		}

		operators = append(operators, operator)
	}

	return operators, nil
}

func (mgr *Manager) getDependencies(operators []*models.MonitoredOperator) map[string]bool {
	fifo := list.New()
	visited := make(map[string]bool)
	for _, op := range operators {
		if op.OperatorType != models.OperatorTypeOlm {
			continue
		}

		visited[op.Name] = true
		for _, dep := range mgr.olmOperators[op.Name].GetDependencies() {
			fifo.PushBack(dep)
		}
	}
	for fifo.Len() > 0 {
		first := fifo.Front()
		op := first.Value.(string)
		for _, dep := range mgr.olmOperators[op].GetDependencies() {
			if !visited[dep] {
				fifo.PushBack(dep)
			}
		}
		visited[op] = true
		fifo.Remove(first)
	}

	return visited
}

func findOperator(operators []*models.MonitoredOperator, operatorName string) *models.MonitoredOperator {
	for _, operator := range operators {
		if operator.Name == operatorName {
			return operator
		}
	}
	return nil
}

func IsEnabled(operators []*models.MonitoredOperator, operatorName string) bool {
	return findOperator(operators, operatorName) != nil
}

func (mgr *Manager) GetMonitoredOperatorsList() map[string]*models.MonitoredOperator {
	return mgr.monitoredOperators
}

func (mgr *Manager) GetOperatorByName(operatorName string) (*models.MonitoredOperator, error) {
	operator, ok := mgr.monitoredOperators[operatorName]
	if !ok {
		return nil, fmt.Errorf("Operator %s isn't supported", operatorName)
	}

	return &models.MonitoredOperator{
		Name:           operator.Name,
		OperatorType:   operator.OperatorType,
		TimeoutSeconds: operator.TimeoutSeconds,
	}, nil
}

func (mgr *Manager) GetSupportedOperatorsByType(operatorType models.OperatorType) []*models.MonitoredOperator {
	operators := make([]*models.MonitoredOperator, 0)

	for _, operator := range mgr.GetMonitoredOperatorsList() {
		if operator.OperatorType == operatorType {
			operator, _ = mgr.GetOperatorByName(operator.Name)
			operators = append(operators, operator)
		}
	}

	return operators
}

// GetMemoryRequirementForRole returns the amount of usable memory required in Bytes in the host to be able to install all the enabled operators and their dependencies.
// The value is determined by the sum of each of the enabled operators.
func (mgr *Manager) GetMemoryRequirementForRole(cluster *common.Cluster, role models.HostRole) (int64, error) {
	switch role {
	case models.HostRoleMaster:
		m, err := mgr.getMemoryRequirementForMaster(cluster)
		if err != nil {
			return 0, err
		}
		return m, nil
	case models.HostRoleWorker:
		m, err := mgr.getMemoryRequirementForWorker(cluster)
		if err != nil {
			return 0, err
		}
		return m, nil
	default:
		return 0, nil

	}
}

func (mgr *Manager) getMemoryRequirementForMaster(cluster *common.Cluster) (int64, error) {

	var t int64
	for _, o := range cluster.MonitoredOperators {
		if o.OperatorType != models.OperatorTypeOlm {
			continue
		}
		m, err := mgr.olmOperators[o.Name].GetMemoryRequirementForMaster(cluster)
		if err != nil {
			return 0, err
		}
		t += m
	}
	return t, nil
}

func (mgr *Manager) getMemoryRequirementForWorker(cluster *common.Cluster) (int64, error) {

	var t int64
	for _, o := range cluster.MonitoredOperators {
		if o.OperatorType != models.OperatorTypeOlm {
			continue
		}
		m, err := mgr.olmOperators[o.Name].GetMemoryRequirementForWorker(cluster)
		if err != nil {
			return 0, err
		}
		t += m
	}
	return t, nil
}

// GetCPURequirementForRole returns the CPU core count available the host must have to install all the enabled operators and their dependencies.
// The value is determined by the sum of each of the enabled operators.
func (mgr *Manager) GetCPURequirementForRole(cluster *common.Cluster, role models.HostRole) (int64, error) {

	switch role {
	case models.HostRoleMaster:
		m, err := mgr.getCPURequirementForMaster(cluster)
		if err != nil {
			return 0, err
		}
		return m, nil
	case models.HostRoleWorker:
		m, err := mgr.getCPURequirementForWorker(cluster)
		if err != nil {
			return 0, err
		}
		return m, nil
	default:
		return 0, nil
	}
}

func (mgr *Manager) getCPURequirementForMaster(cluster *common.Cluster) (int64, error) {

	var t int64
	for _, o := range cluster.MonitoredOperators {
		if o.OperatorType != models.OperatorTypeOlm {
			continue
		}
		m, err := mgr.olmOperators[o.Name].GetCPURequirementForMaster(cluster)
		if err != nil {
			return 0, err
		}
		t += m
	}
	return t, nil
}

func (mgr *Manager) getCPURequirementForWorker(cluster *common.Cluster) (int64, error) {

	var t int64
	for _, o := range cluster.MonitoredOperators {
		if o.OperatorType != models.OperatorTypeOlm {
			continue
		}
		m, err := mgr.olmOperators[o.Name].GetCPURequirementForWorker(cluster)
		if err != nil {
			return 0, err
		}
		t += m
	}
	return t, nil
}
