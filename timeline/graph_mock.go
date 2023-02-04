// Code generated by MockGen. DO NOT EDIT.
// Source: interfaces.go

// Package timeline is a generated GoMock package.
package timeline

import (
	context "context"
	gomock "github.com/golang/mock/gomock"
	address "github.com/msaldanha/setinstone/anticorp/address"
	graph "github.com/msaldanha/setinstone/anticorp/graph"
	reflect "reflect"
)

// MockGraph is a mock of Graph interface
type MockGraph struct {
	ctrl     *gomock.Controller
	recorder *MockGraphMockRecorder
}

// MockGraphMockRecorder is the mock recorder for MockGraph
type MockGraphMockRecorder struct {
	mock *MockGraph
}

// NewMockGraph creates a new mock instance
func NewMockGraph(ctrl *gomock.Controller) *MockGraph {
	mock := &MockGraph{ctrl: ctrl}
	mock.recorder = &MockGraphMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockGraph) EXPECT() *MockGraphMockRecorder {
	return m.recorder
}

// GetName mocks base method
func (m *MockGraph) GetName() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetName")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetName indicates an expected call of GetName
func (mr *MockGraphMockRecorder) GetName() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetName", reflect.TypeOf((*MockGraph)(nil).GetName))
}

// GetMetaData mocks base method
func (m *MockGraph) GetMetaData() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetMetaData")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetMetaData indicates an expected call of GetMetaData
func (mr *MockGraphMockRecorder) GetMetaData() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMetaData", reflect.TypeOf((*MockGraph)(nil).GetMetaData))
}

// Get mocks base method
func (m *MockGraph) Get(ctx context.Context, key string) (graph.Node, bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", ctx, key)
	ret0, _ := ret[0].(graph.Node)
	ret1, _ := ret[1].(bool)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// Get indicates an expected call of Get
func (mr *MockGraphMockRecorder) Get(ctx, key interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockGraph)(nil).Get), ctx, key)
}

// Append mocks base method
func (m *MockGraph) Append(ctx context.Context, keyRoot string, node graph.NodeData) (graph.Node, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Append", ctx, keyRoot, node)
	ret0, _ := ret[0].(graph.Node)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Append indicates an expected call of Append
func (mr *MockGraphMockRecorder) Append(ctx, keyRoot, node interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Append", reflect.TypeOf((*MockGraph)(nil).Append), ctx, keyRoot, node)
}

// GetIterator mocks base method
func (m *MockGraph) GetIterator(ctx context.Context, keyRoot, branch, from string) (*graph.Iterator, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetIterator", ctx, keyRoot, branch, from)
	ret0, _ := ret[0].(*graph.Iterator)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetIterator indicates an expected call of GetIterator
func (mr *MockGraphMockRecorder) GetIterator(ctx, keyRoot, branch, from interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetIterator", reflect.TypeOf((*MockGraph)(nil).GetIterator), ctx, keyRoot, branch, from)
}

// GetAddress mocks base method
func (m *MockGraph) GetAddress(ctx context.Context) *address.Address {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAddress", ctx)
	ret0, _ := ret[0].(*address.Address)
	return ret0
}

// GetAddress indicates an expected call of GetAddress
func (mr *MockGraphMockRecorder) GetAddress(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAddress", reflect.TypeOf((*MockGraph)(nil).GetAddress), ctx)
}

// Manage mocks base method
func (m *MockGraph) Manage(addr *address.Address) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Manage", addr)
	ret0, _ := ret[0].(error)
	return ret0
}

// Manage indicates an expected call of Manage
func (mr *MockGraphMockRecorder) Manage(addr interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Manage", reflect.TypeOf((*MockGraph)(nil).Manage), addr)
}

// MockIterator is a mock of Iterator interface
type MockIterator struct {
	ctrl     *gomock.Controller
	recorder *MockIteratorMockRecorder
}

// MockIteratorMockRecorder is the mock recorder for MockIterator
type MockIteratorMockRecorder struct {
	mock *MockIterator
}

// NewMockIterator creates a new mock instance
func NewMockIterator(ctrl *gomock.Controller) *MockIterator {
	mock := &MockIterator{ctrl: ctrl}
	mock.recorder = &MockIteratorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockIterator) EXPECT() *MockIteratorMockRecorder {
	return m.recorder
}

// Next mocks base method
func (m *MockIterator) Next(ctx context.Context) (graph.Node, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Next", ctx)
	ret0, _ := ret[0].(graph.Node)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Next indicates an expected call of Next
func (mr *MockIteratorMockRecorder) Next(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Next", reflect.TypeOf((*MockIterator)(nil).Next), ctx)
}

// HasNext mocks base method
func (m *MockIterator) HasNext() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "HasNext")
	ret0, _ := ret[0].(bool)
	return ret0
}

// HasNext indicates an expected call of HasNext
func (mr *MockIteratorMockRecorder) HasNext() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HasNext", reflect.TypeOf((*MockIterator)(nil).HasNext))
}

// MockTimeline is a mock of Timeline interface
type MockTimeline struct {
	ctrl     *gomock.Controller
	recorder *MockTimelineMockRecorder
}

// MockTimelineMockRecorder is the mock recorder for MockTimeline
type MockTimelineMockRecorder struct {
	mock *MockTimeline
}

// NewMockTimeline creates a new mock instance
func NewMockTimeline(ctrl *gomock.Controller) *MockTimeline {
	mock := &MockTimeline{ctrl: ctrl}
	mock.recorder = &MockTimelineMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockTimeline) EXPECT() *MockTimelineMockRecorder {
	return m.recorder
}

// AppendPost mocks base method
func (m *MockTimeline) AppendPost(ctx context.Context, post PostItem, keyRoot, connector string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AppendPost", ctx, post, keyRoot, connector)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// AppendPost indicates an expected call of AppendPost
func (mr *MockTimelineMockRecorder) AppendPost(ctx, post, keyRoot, connector interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AppendPost", reflect.TypeOf((*MockTimeline)(nil).AppendPost), ctx, post, keyRoot, connector)
}

// AppendReference mocks base method
func (m *MockTimeline) AppendReference(ctx context.Context, ref ReferenceItem, keyRoot, connector string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AppendReference", ctx, ref, keyRoot, connector)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// AppendReference indicates an expected call of AppendReference
func (mr *MockTimelineMockRecorder) AppendReference(ctx, ref, keyRoot, connector interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AppendReference", reflect.TypeOf((*MockTimeline)(nil).AppendReference), ctx, ref, keyRoot, connector)
}

// AddReceivedReference mocks base method
func (m *MockTimeline) AddReceivedReference(ctx context.Context, refKey string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AddReceivedReference", ctx, refKey)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// AddReceivedReference indicates an expected call of AddReceivedReference
func (mr *MockTimelineMockRecorder) AddReceivedReference(ctx, refKey interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddReceivedReference", reflect.TypeOf((*MockTimeline)(nil).AddReceivedReference), ctx, refKey)
}

// Get mocks base method
func (m *MockTimeline) Get(ctx context.Context, key string) (Item, bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", ctx, key)
	ret0, _ := ret[0].(Item)
	ret1, _ := ret[1].(bool)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// Get indicates an expected call of Get
func (mr *MockTimelineMockRecorder) Get(ctx, key interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockTimeline)(nil).Get), ctx, key)
}

// GetFrom mocks base method
func (m *MockTimeline) GetFrom(ctx context.Context, keyRoot, connector, keyFrom, keyTo string, count int) ([]Item, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetFrom", ctx, keyRoot, connector, keyFrom, keyTo, count)
	ret0, _ := ret[0].([]Item)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetFrom indicates an expected call of GetFrom
func (mr *MockTimelineMockRecorder) GetFrom(ctx, keyRoot, connector, keyFrom, keyTo, count interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetFrom", reflect.TypeOf((*MockTimeline)(nil).GetFrom), ctx, keyRoot, connector, keyFrom, keyTo, count)
}
