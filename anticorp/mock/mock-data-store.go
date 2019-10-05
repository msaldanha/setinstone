// Code generated by MockGen. DO NOT EDIT.
// Source: data-store.go

// Package mock is a generated GoMock package.
package mock

import (
	"context"
	"github.com/golang/mock/gomock"
	x "github.com/msaldanha/setinstone/anticorp/datastore"
	"io"
	"reflect"
)

// MockDataStore is a mock of DataStore interface
type MockDataStore struct {
	ctrl     *gomock.Controller
	recorder *MockDataStoreMockRecorder
}

// MockDataStoreMockRecorder is the mock recorder for MockDataStore
type MockDataStoreMockRecorder struct {
	mock *MockDataStore
}

// NewMockDataStore creates a new mock instance
func NewMockDataStore(ctrl *gomock.Controller) *MockDataStore {
	mock := &MockDataStore{ctrl: ctrl}
	mock.recorder = &MockDataStoreMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockDataStore) EXPECT() *MockDataStoreMockRecorder {
	return m.recorder
}

// AddFile mocks base method
func (m *MockDataStore) AddFile(ctx context.Context, path string) (x.Link, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AddFile", ctx, path)
	ret0, _ := ret[0].(x.Link)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// AddFile indicates an expected call of AddFile
func (mr *MockDataStoreMockRecorder) AddFile(ctx, path interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddFile", reflect.TypeOf((*MockDataStore)(nil).AddFile), ctx, path)
}

// AddBytes mocks base method
func (m *MockDataStore) AddBytes(ctx context.Context, name string, bytes []byte) (x.Link, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AddBytes", ctx, name, bytes)
	ret0, _ := ret[0].(x.Link)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// AddBytes indicates an expected call of AddBytes
func (mr *MockDataStoreMockRecorder) AddBytes(ctx, name, bytes interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddBytes", reflect.TypeOf((*MockDataStore)(nil).AddBytes), ctx, name, bytes)
}

// Get mocks base method
func (m *MockDataStore) Get(ctx context.Context, hash string) (io.Reader, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", ctx, hash)
	ret0, _ := ret[0].(io.Reader)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get
func (mr *MockDataStoreMockRecorder) Get(ctx, hash interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockDataStore)(nil).Get), ctx, hash)
}

// Ls mocks base method
func (m *MockDataStore) Ls(ctx context.Context, hash string) ([]x.Link, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Ls", ctx, hash)
	ret0, _ := ret[0].([]x.Link)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Ls indicates an expected call of Ls
func (mr *MockDataStoreMockRecorder) Ls(ctx, hash interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Ls", reflect.TypeOf((*MockDataStore)(nil).Ls), ctx, hash)
}

// Exists mocks base method
func (m *MockDataStore) Exists(ctx context.Context, hash string) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Exists", ctx, hash)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Exists indicates an expected call of Exists
func (mr *MockDataStoreMockRecorder) Exists(ctx, hash interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Exists", reflect.TypeOf((*MockDataStore)(nil).Exists), ctx, hash)
}
