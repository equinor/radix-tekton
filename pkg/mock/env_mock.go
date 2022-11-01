// Code generated by MockGen. DO NOT EDIT.
// Source: ./pkg/models/env/env.go

// Package models is a generated GoMock package.
package models

import (
	reflect "reflect"

	v1 "github.com/equinor/radix-operator/pkg/apis/radix/v1"
	gomock "github.com/golang/mock/gomock"
	logrus "github.com/sirupsen/logrus"
)

// MockEnv is a mock of Env interface.
type MockEnv struct {
	ctrl     *gomock.Controller
	recorder *MockEnvMockRecorder
}

// MockEnvMockRecorder is the mock recorder for MockEnv.
type MockEnvMockRecorder struct {
	mock *MockEnv
}

// NewMockEnv creates a new mock instance.
func NewMockEnv(ctrl *gomock.Controller) *MockEnv {
	mock := &MockEnv{ctrl: ctrl}
	mock.recorder = &MockEnvMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockEnv) EXPECT() *MockEnvMockRecorder {
	return m.recorder
}

// GetAppName mocks base method.
func (m *MockEnv) GetAppName() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAppName")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetAppName indicates an expected call of GetAppName.
func (mr *MockEnvMockRecorder) GetAppName() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAppName", reflect.TypeOf((*MockEnv)(nil).GetAppName))
}

// GetAppNamespace mocks base method.
func (m *MockEnv) GetAppNamespace() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAppNamespace")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetAppNamespace indicates an expected call of GetAppNamespace.
func (mr *MockEnvMockRecorder) GetAppNamespace() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAppNamespace", reflect.TypeOf((*MockEnv)(nil).GetAppNamespace))
}

// GetBranch mocks base method.
func (m *MockEnv) GetBranch() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBranch")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetBranch indicates an expected call of GetBranch.
func (mr *MockEnvMockRecorder) GetBranch() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBranch", reflect.TypeOf((*MockEnv)(nil).GetBranch))
}

// GetGitConfigMapName mocks base method.
func (m *MockEnv) GetGitConfigMapName() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetGitConfigMapName")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetGitConfigMapName indicates an expected call of GetGitConfigMapName.
func (mr *MockEnvMockRecorder) GetGitConfigMapName() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetGitConfigMapName", reflect.TypeOf((*MockEnv)(nil).GetGitConfigMapName))
}

// GetGitRepositoryWorkspace mocks base method.
func (m *MockEnv) GetGitRepositoryWorkspace() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetGitRepositoryWorkspace")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetGitRepositoryWorkspace indicates an expected call of GetGitRepositoryWorkspace.
func (mr *MockEnvMockRecorder) GetGitRepositoryWorkspace() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetGitRepositoryWorkspace", reflect.TypeOf((*MockEnv)(nil).GetGitRepositoryWorkspace))
}

// GetLogLevel mocks base method.
func (m *MockEnv) GetLogLevel() logrus.Level {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLogLevel")
	ret0, _ := ret[0].(logrus.Level)
	return ret0
}

// GetLogLevel indicates an expected call of GetLogLevel.
func (mr *MockEnvMockRecorder) GetLogLevel() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLogLevel", reflect.TypeOf((*MockEnv)(nil).GetLogLevel))
}

// GetPipelinesAction mocks base method.
func (m *MockEnv) GetPipelinesAction() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPipelinesAction")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetPipelinesAction indicates an expected call of GetPipelinesAction.
func (mr *MockEnvMockRecorder) GetPipelinesAction() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPipelinesAction", reflect.TypeOf((*MockEnv)(nil).GetPipelinesAction))
}

// GetRadixConfigFileName mocks base method.
func (m *MockEnv) GetRadixConfigFileName() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRadixConfigFileName")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetRadixConfigFileName indicates an expected call of GetRadixConfigFileName.
func (mr *MockEnvMockRecorder) GetRadixConfigFileName() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRadixConfigFileName", reflect.TypeOf((*MockEnv)(nil).GetRadixConfigFileName))
}

// GetRadixConfigMapName mocks base method.
func (m *MockEnv) GetRadixConfigMapName() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRadixConfigMapName")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetRadixConfigMapName indicates an expected call of GetRadixConfigMapName.
func (mr *MockEnvMockRecorder) GetRadixConfigMapName() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRadixConfigMapName", reflect.TypeOf((*MockEnv)(nil).GetRadixConfigMapName))
}

// GetRadixDeployToEnvironment mocks base method.
func (m *MockEnv) GetRadixDeployToEnvironment() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRadixDeployToEnvironment")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetRadixDeployToEnvironment indicates an expected call of GetRadixDeployToEnvironment.
func (mr *MockEnvMockRecorder) GetRadixDeployToEnvironment() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRadixDeployToEnvironment", reflect.TypeOf((*MockEnv)(nil).GetRadixDeployToEnvironment))
}

// GetRadixImageTag mocks base method.
func (m *MockEnv) GetRadixImageTag() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRadixImageTag")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetRadixImageTag indicates an expected call of GetRadixImageTag.
func (mr *MockEnvMockRecorder) GetRadixImageTag() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRadixImageTag", reflect.TypeOf((*MockEnv)(nil).GetRadixImageTag))
}

// GetRadixPipelineJobName mocks base method.
func (m *MockEnv) GetRadixPipelineJobName() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRadixPipelineJobName")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetRadixPipelineJobName indicates an expected call of GetRadixPipelineJobName.
func (mr *MockEnvMockRecorder) GetRadixPipelineJobName() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRadixPipelineJobName", reflect.TypeOf((*MockEnv)(nil).GetRadixPipelineJobName))
}

// GetRadixPipelineType mocks base method.
func (m *MockEnv) GetRadixPipelineType() v1.RadixPipelineType {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRadixPipelineType")
	ret0, _ := ret[0].(v1.RadixPipelineType)
	return ret0
}

// GetRadixPipelineType indicates an expected call of GetRadixPipelineType.
func (mr *MockEnvMockRecorder) GetRadixPipelineType() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRadixPipelineType", reflect.TypeOf((*MockEnv)(nil).GetRadixPipelineType))
}

// GetRadixPromoteDeployment mocks base method.
func (m *MockEnv) GetRadixPromoteDeployment() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRadixPromoteDeployment")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetRadixPromoteDeployment indicates an expected call of GetRadixPromoteDeployment.
func (mr *MockEnvMockRecorder) GetRadixPromoteDeployment() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRadixPromoteDeployment", reflect.TypeOf((*MockEnv)(nil).GetRadixPromoteDeployment))
}

// GetRadixPromoteFromEnvironment mocks base method.
func (m *MockEnv) GetRadixPromoteFromEnvironment() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRadixPromoteFromEnvironment")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetRadixPromoteFromEnvironment indicates an expected call of GetRadixPromoteFromEnvironment.
func (mr *MockEnvMockRecorder) GetRadixPromoteFromEnvironment() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRadixPromoteFromEnvironment", reflect.TypeOf((*MockEnv)(nil).GetRadixPromoteFromEnvironment))
}

// GetWebhookCommitId mocks base method.
func (m *MockEnv) GetWebhookCommitId() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetWebhookCommitId")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetWebhookCommitId indicates an expected call of GetWebhookCommitId.
func (mr *MockEnvMockRecorder) GetWebhookCommitId() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetWebhookCommitId", reflect.TypeOf((*MockEnv)(nil).GetWebhookCommitId))
}
