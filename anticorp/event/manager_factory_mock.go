// Code generated by MockGen. DO NOT EDIT.
// Source: manager_factory.go

// Package event is a generated GoMock package.
package event

import (
	gomock "github.com/golang/mock/gomock"
	address "github.com/msaldanha/setinstone/anticorp/address"
	zap "go.uber.org/zap"
	reflect "reflect"
)

// MockManagerFactory is a mock of ManagerFactory interface
type MockManagerFactory struct {
	ctrl     *gomock.Controller
	recorder *MockManagerFactoryMockRecorder
}

// MockManagerFactoryMockRecorder is the mock recorder for MockManagerFactory
type MockManagerFactoryMockRecorder struct {
	mock *MockManagerFactory
}

// NewMockManagerFactory creates a new mock instance
func NewMockManagerFactory(ctrl *gomock.Controller) *MockManagerFactory {
	mock := &MockManagerFactory{ctrl: ctrl}
	mock.recorder = &MockManagerFactoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockManagerFactory) EXPECT() *MockManagerFactoryMockRecorder {
	return m.recorder
}

// Build mocks base method
func (m *MockManagerFactory) Build(nameSpace string, signerAddr, managedAddr *address.Address, logger *zap.Logger) (Manager, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Build", nameSpace, signerAddr, managedAddr, logger)
	ret0, _ := ret[0].(Manager)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Build indicates an expected call of Build
func (mr *MockManagerFactoryMockRecorder) Build(nameSpace, signerAddr, managedAddr, logger interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Build", reflect.TypeOf((*MockManagerFactory)(nil).Build), nameSpace, signerAddr, managedAddr, logger)
}
