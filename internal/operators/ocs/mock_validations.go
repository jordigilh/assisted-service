// Code generated by MockGen. DO NOT EDIT.
// Source: validations.go

// Package ocs is a generated GoMock package.
package ocs

import (
	gomock "github.com/golang/mock/gomock"
	api "github.com/openshift/assisted-service/internal/operators/api"
	models "github.com/openshift/assisted-service/models"
	reflect "reflect"
)

// MockOCSValidator is a mock of OCSValidator interface
type MockOCSValidator struct {
	ctrl     *gomock.Controller
	recorder *MockOCSValidatorMockRecorder
}

// MockOCSValidatorMockRecorder is the mock recorder for MockOCSValidator
type MockOCSValidatorMockRecorder struct {
	mock *MockOCSValidator
}

// NewMockOCSValidator creates a new mock instance
func NewMockOCSValidator(ctrl *gomock.Controller) *MockOCSValidator {
	mock := &MockOCSValidator{ctrl: ctrl}
	mock.recorder = &MockOCSValidatorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockOCSValidator) EXPECT() *MockOCSValidatorMockRecorder {
	return m.recorder
}

// ValidateRequirements mocks base method
func (m *MockOCSValidator) ValidateRequirements(cluster *models.Cluster) (api.ValidationStatus, string) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ValidateRequirements", cluster)
	ret0, _ := ret[0].(api.ValidationStatus)
	ret1, _ := ret[1].(string)
	return ret0, ret1
}

// ValidateRequirements indicates an expected call of ValidateRequirements
func (mr *MockOCSValidatorMockRecorder) ValidateRequirements(cluster interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ValidateRequirements", reflect.TypeOf((*MockOCSValidator)(nil).ValidateRequirements), cluster)
}

// IdentifyDeploymentTypeForResource mocks base method
func (m *MockOCSValidator) IdentifyDeploymentTypeForResource(cluster *models.Cluster, res resource) (ocsDeploymentType, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IdentifyDeploymentTypeForResource", cluster, res)
	ret0, _ := ret[0].(ocsDeploymentType)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// IdentifyDeploymentTypeForResource indicates an expected call of IdentifyDeploymentTypeForResource
func (mr *MockOCSValidatorMockRecorder) IdentifyDeploymentTypeForResource(cluster, res interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IdentifyDeploymentTypeForResource", reflect.TypeOf((*MockOCSValidator)(nil).IdentifyDeploymentTypeForResource), cluster, res)
}
