// Code generated by mockery v2.43.1. DO NOT EDIT.

package gpbft

import (
	time "time"

	mock "github.com/stretchr/testify/mock"
)

// MockHost is an autogenerated mock type for the Host type
type MockHost struct {
	mock.Mock
}

type MockHost_Expecter struct {
	mock *mock.Mock
}

func (_m *MockHost) EXPECT() *MockHost_Expecter {
	return &MockHost_Expecter{mock: &_m.Mock}
}

// Aggregate provides a mock function with given fields: pubKeys, sigs
func (_m *MockHost) Aggregate(pubKeys []PubKey, sigs [][]byte) ([]byte, error) {
	ret := _m.Called(pubKeys, sigs)

	if len(ret) == 0 {
		panic("no return value specified for Aggregate")
	}

	var r0 []byte
	var r1 error
	if rf, ok := ret.Get(0).(func([]PubKey, [][]byte) ([]byte, error)); ok {
		return rf(pubKeys, sigs)
	}
	if rf, ok := ret.Get(0).(func([]PubKey, [][]byte) []byte); ok {
		r0 = rf(pubKeys, sigs)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	if rf, ok := ret.Get(1).(func([]PubKey, [][]byte) error); ok {
		r1 = rf(pubKeys, sigs)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockHost_Aggregate_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Aggregate'
type MockHost_Aggregate_Call struct {
	*mock.Call
}

// Aggregate is a helper method to define mock.On call
//   - pubKeys []PubKey
//   - sigs [][]byte
func (_e *MockHost_Expecter) Aggregate(pubKeys interface{}, sigs interface{}) *MockHost_Aggregate_Call {
	return &MockHost_Aggregate_Call{Call: _e.mock.On("Aggregate", pubKeys, sigs)}
}

func (_c *MockHost_Aggregate_Call) Run(run func(pubKeys []PubKey, sigs [][]byte)) *MockHost_Aggregate_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].([]PubKey), args[1].([][]byte))
	})
	return _c
}

func (_c *MockHost_Aggregate_Call) Return(_a0 []byte, _a1 error) *MockHost_Aggregate_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockHost_Aggregate_Call) RunAndReturn(run func([]PubKey, [][]byte) ([]byte, error)) *MockHost_Aggregate_Call {
	_c.Call.Return(run)
	return _c
}

// GetChainForInstance provides a mock function with given fields: instance
func (_m *MockHost) GetChainForInstance(instance uint64) (ECChain, error) {
	ret := _m.Called(instance)

	if len(ret) == 0 {
		panic("no return value specified for GetChainForInstance")
	}

	var r0 ECChain
	var r1 error
	if rf, ok := ret.Get(0).(func(uint64) (ECChain, error)); ok {
		return rf(instance)
	}
	if rf, ok := ret.Get(0).(func(uint64) ECChain); ok {
		r0 = rf(instance)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(ECChain)
		}
	}

	if rf, ok := ret.Get(1).(func(uint64) error); ok {
		r1 = rf(instance)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockHost_GetChainForInstance_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetChainForInstance'
type MockHost_GetChainForInstance_Call struct {
	*mock.Call
}

// GetChainForInstance is a helper method to define mock.On call
//   - instance uint64
func (_e *MockHost_Expecter) GetChainForInstance(instance interface{}) *MockHost_GetChainForInstance_Call {
	return &MockHost_GetChainForInstance_Call{Call: _e.mock.On("GetChainForInstance", instance)}
}

func (_c *MockHost_GetChainForInstance_Call) Run(run func(instance uint64)) *MockHost_GetChainForInstance_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(uint64))
	})
	return _c
}

func (_c *MockHost_GetChainForInstance_Call) Return(chain ECChain, err error) *MockHost_GetChainForInstance_Call {
	_c.Call.Return(chain, err)
	return _c
}

func (_c *MockHost_GetChainForInstance_Call) RunAndReturn(run func(uint64) (ECChain, error)) *MockHost_GetChainForInstance_Call {
	_c.Call.Return(run)
	return _c
}

// GetCommitteeForInstance provides a mock function with given fields: instance
func (_m *MockHost) GetCommitteeForInstance(instance uint64) (*PowerTable, []byte, error) {
	ret := _m.Called(instance)

	if len(ret) == 0 {
		panic("no return value specified for GetCommitteeForInstance")
	}

	var r0 *PowerTable
	var r1 []byte
	var r2 error
	if rf, ok := ret.Get(0).(func(uint64) (*PowerTable, []byte, error)); ok {
		return rf(instance)
	}
	if rf, ok := ret.Get(0).(func(uint64) *PowerTable); ok {
		r0 = rf(instance)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*PowerTable)
		}
	}

	if rf, ok := ret.Get(1).(func(uint64) []byte); ok {
		r1 = rf(instance)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).([]byte)
		}
	}

	if rf, ok := ret.Get(2).(func(uint64) error); ok {
		r2 = rf(instance)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// MockHost_GetCommitteeForInstance_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetCommitteeForInstance'
type MockHost_GetCommitteeForInstance_Call struct {
	*mock.Call
}

// GetCommitteeForInstance is a helper method to define mock.On call
//   - instance uint64
func (_e *MockHost_Expecter) GetCommitteeForInstance(instance interface{}) *MockHost_GetCommitteeForInstance_Call {
	return &MockHost_GetCommitteeForInstance_Call{Call: _e.mock.On("GetCommitteeForInstance", instance)}
}

func (_c *MockHost_GetCommitteeForInstance_Call) Run(run func(instance uint64)) *MockHost_GetCommitteeForInstance_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(uint64))
	})
	return _c
}

func (_c *MockHost_GetCommitteeForInstance_Call) Return(power *PowerTable, beacon []byte, err error) *MockHost_GetCommitteeForInstance_Call {
	_c.Call.Return(power, beacon, err)
	return _c
}

func (_c *MockHost_GetCommitteeForInstance_Call) RunAndReturn(run func(uint64) (*PowerTable, []byte, error)) *MockHost_GetCommitteeForInstance_Call {
	_c.Call.Return(run)
	return _c
}

// ID provides a mock function with given fields:
func (_m *MockHost) ID() ActorID {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for ID")
	}

	var r0 ActorID
	if rf, ok := ret.Get(0).(func() ActorID); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(ActorID)
	}

	return r0
}

// MockHost_ID_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ID'
type MockHost_ID_Call struct {
	*mock.Call
}

// ID is a helper method to define mock.On call
func (_e *MockHost_Expecter) ID() *MockHost_ID_Call {
	return &MockHost_ID_Call{Call: _e.mock.On("ID")}
}

func (_c *MockHost_ID_Call) Run(run func()) *MockHost_ID_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockHost_ID_Call) Return(_a0 ActorID) *MockHost_ID_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockHost_ID_Call) RunAndReturn(run func() ActorID) *MockHost_ID_Call {
	_c.Call.Return(run)
	return _c
}

// MarshalPayloadForSigning provides a mock function with given fields: _a0
func (_m *MockHost) MarshalPayloadForSigning(_a0 *Payload) []byte {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for MarshalPayloadForSigning")
	}

	var r0 []byte
	if rf, ok := ret.Get(0).(func(*Payload) []byte); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	return r0
}

// MockHost_MarshalPayloadForSigning_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'MarshalPayloadForSigning'
type MockHost_MarshalPayloadForSigning_Call struct {
	*mock.Call
}

// MarshalPayloadForSigning is a helper method to define mock.On call
//   - _a0 *Payload
func (_e *MockHost_Expecter) MarshalPayloadForSigning(_a0 interface{}) *MockHost_MarshalPayloadForSigning_Call {
	return &MockHost_MarshalPayloadForSigning_Call{Call: _e.mock.On("MarshalPayloadForSigning", _a0)}
}

func (_c *MockHost_MarshalPayloadForSigning_Call) Run(run func(_a0 *Payload)) *MockHost_MarshalPayloadForSigning_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*Payload))
	})
	return _c
}

func (_c *MockHost_MarshalPayloadForSigning_Call) Return(_a0 []byte) *MockHost_MarshalPayloadForSigning_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockHost_MarshalPayloadForSigning_Call) RunAndReturn(run func(*Payload) []byte) *MockHost_MarshalPayloadForSigning_Call {
	_c.Call.Return(run)
	return _c
}

// NetworkName provides a mock function with given fields:
func (_m *MockHost) NetworkName() NetworkName {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for NetworkName")
	}

	var r0 NetworkName
	if rf, ok := ret.Get(0).(func() NetworkName); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(NetworkName)
	}

	return r0
}

// MockHost_NetworkName_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'NetworkName'
type MockHost_NetworkName_Call struct {
	*mock.Call
}

// NetworkName is a helper method to define mock.On call
func (_e *MockHost_Expecter) NetworkName() *MockHost_NetworkName_Call {
	return &MockHost_NetworkName_Call{Call: _e.mock.On("NetworkName")}
}

func (_c *MockHost_NetworkName_Call) Run(run func()) *MockHost_NetworkName_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockHost_NetworkName_Call) Return(_a0 NetworkName) *MockHost_NetworkName_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockHost_NetworkName_Call) RunAndReturn(run func() NetworkName) *MockHost_NetworkName_Call {
	_c.Call.Return(run)
	return _c
}

// ReceiveDecision provides a mock function with given fields: decision
func (_m *MockHost) ReceiveDecision(decision *Justification) time.Time {
	ret := _m.Called(decision)

	if len(ret) == 0 {
		panic("no return value specified for ReceiveDecision")
	}

	var r0 time.Time
	if rf, ok := ret.Get(0).(func(*Justification) time.Time); ok {
		r0 = rf(decision)
	} else {
		r0 = ret.Get(0).(time.Time)
	}

	return r0
}

// MockHost_ReceiveDecision_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ReceiveDecision'
type MockHost_ReceiveDecision_Call struct {
	*mock.Call
}

// ReceiveDecision is a helper method to define mock.On call
//   - decision *Justification
func (_e *MockHost_Expecter) ReceiveDecision(decision interface{}) *MockHost_ReceiveDecision_Call {
	return &MockHost_ReceiveDecision_Call{Call: _e.mock.On("ReceiveDecision", decision)}
}

func (_c *MockHost_ReceiveDecision_Call) Run(run func(decision *Justification)) *MockHost_ReceiveDecision_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*Justification))
	})
	return _c
}

func (_c *MockHost_ReceiveDecision_Call) Return(_a0 time.Time) *MockHost_ReceiveDecision_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockHost_ReceiveDecision_Call) RunAndReturn(run func(*Justification) time.Time) *MockHost_ReceiveDecision_Call {
	_c.Call.Return(run)
	return _c
}

// RequestBroadcast provides a mock function with given fields: mb
func (_m *MockHost) RequestBroadcast(mb *MessageBuilder) {
	_m.Called(mb)
}

// MockHost_RequestBroadcast_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'RequestBroadcast'
type MockHost_RequestBroadcast_Call struct {
	*mock.Call
}

// RequestBroadcast is a helper method to define mock.On call
//   - mb *MessageBuilder
func (_e *MockHost_Expecter) RequestBroadcast(mb interface{}) *MockHost_RequestBroadcast_Call {
	return &MockHost_RequestBroadcast_Call{Call: _e.mock.On("RequestBroadcast", mb)}
}

func (_c *MockHost_RequestBroadcast_Call) Run(run func(mb *MessageBuilder)) *MockHost_RequestBroadcast_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*MessageBuilder))
	})
	return _c
}

func (_c *MockHost_RequestBroadcast_Call) Return() *MockHost_RequestBroadcast_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockHost_RequestBroadcast_Call) RunAndReturn(run func(*MessageBuilder)) *MockHost_RequestBroadcast_Call {
	_c.Call.Return(run)
	return _c
}

// SetAlarm provides a mock function with given fields: at
func (_m *MockHost) SetAlarm(at time.Time) {
	_m.Called(at)
}

// MockHost_SetAlarm_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SetAlarm'
type MockHost_SetAlarm_Call struct {
	*mock.Call
}

// SetAlarm is a helper method to define mock.On call
//   - at time.Time
func (_e *MockHost_Expecter) SetAlarm(at interface{}) *MockHost_SetAlarm_Call {
	return &MockHost_SetAlarm_Call{Call: _e.mock.On("SetAlarm", at)}
}

func (_c *MockHost_SetAlarm_Call) Run(run func(at time.Time)) *MockHost_SetAlarm_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(time.Time))
	})
	return _c
}

func (_c *MockHost_SetAlarm_Call) Return() *MockHost_SetAlarm_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockHost_SetAlarm_Call) RunAndReturn(run func(time.Time)) *MockHost_SetAlarm_Call {
	_c.Call.Return(run)
	return _c
}

// Sign provides a mock function with given fields: sender, msg
func (_m *MockHost) Sign(sender PubKey, msg []byte) ([]byte, error) {
	ret := _m.Called(sender, msg)

	if len(ret) == 0 {
		panic("no return value specified for Sign")
	}

	var r0 []byte
	var r1 error
	if rf, ok := ret.Get(0).(func(PubKey, []byte) ([]byte, error)); ok {
		return rf(sender, msg)
	}
	if rf, ok := ret.Get(0).(func(PubKey, []byte) []byte); ok {
		r0 = rf(sender, msg)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	if rf, ok := ret.Get(1).(func(PubKey, []byte) error); ok {
		r1 = rf(sender, msg)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockHost_Sign_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Sign'
type MockHost_Sign_Call struct {
	*mock.Call
}

// Sign is a helper method to define mock.On call
//   - sender PubKey
//   - msg []byte
func (_e *MockHost_Expecter) Sign(sender interface{}, msg interface{}) *MockHost_Sign_Call {
	return &MockHost_Sign_Call{Call: _e.mock.On("Sign", sender, msg)}
}

func (_c *MockHost_Sign_Call) Run(run func(sender PubKey, msg []byte)) *MockHost_Sign_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(PubKey), args[1].([]byte))
	})
	return _c
}

func (_c *MockHost_Sign_Call) Return(_a0 []byte, _a1 error) *MockHost_Sign_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockHost_Sign_Call) RunAndReturn(run func(PubKey, []byte) ([]byte, error)) *MockHost_Sign_Call {
	_c.Call.Return(run)
	return _c
}

// Time provides a mock function with given fields:
func (_m *MockHost) Time() time.Time {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Time")
	}

	var r0 time.Time
	if rf, ok := ret.Get(0).(func() time.Time); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(time.Time)
	}

	return r0
}

// MockHost_Time_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Time'
type MockHost_Time_Call struct {
	*mock.Call
}

// Time is a helper method to define mock.On call
func (_e *MockHost_Expecter) Time() *MockHost_Time_Call {
	return &MockHost_Time_Call{Call: _e.mock.On("Time")}
}

func (_c *MockHost_Time_Call) Run(run func()) *MockHost_Time_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockHost_Time_Call) Return(_a0 time.Time) *MockHost_Time_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockHost_Time_Call) RunAndReturn(run func() time.Time) *MockHost_Time_Call {
	_c.Call.Return(run)
	return _c
}

// Verify provides a mock function with given fields: pubKey, msg, sig
func (_m *MockHost) Verify(pubKey PubKey, msg []byte, sig []byte) error {
	ret := _m.Called(pubKey, msg, sig)

	if len(ret) == 0 {
		panic("no return value specified for Verify")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(PubKey, []byte, []byte) error); ok {
		r0 = rf(pubKey, msg, sig)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockHost_Verify_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Verify'
type MockHost_Verify_Call struct {
	*mock.Call
}

// Verify is a helper method to define mock.On call
//   - pubKey PubKey
//   - msg []byte
//   - sig []byte
func (_e *MockHost_Expecter) Verify(pubKey interface{}, msg interface{}, sig interface{}) *MockHost_Verify_Call {
	return &MockHost_Verify_Call{Call: _e.mock.On("Verify", pubKey, msg, sig)}
}

func (_c *MockHost_Verify_Call) Run(run func(pubKey PubKey, msg []byte, sig []byte)) *MockHost_Verify_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(PubKey), args[1].([]byte), args[2].([]byte))
	})
	return _c
}

func (_c *MockHost_Verify_Call) Return(_a0 error) *MockHost_Verify_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockHost_Verify_Call) RunAndReturn(run func(PubKey, []byte, []byte) error) *MockHost_Verify_Call {
	_c.Call.Return(run)
	return _c
}

// VerifyAggregate provides a mock function with given fields: payload, aggSig, signers
func (_m *MockHost) VerifyAggregate(payload []byte, aggSig []byte, signers []PubKey) error {
	ret := _m.Called(payload, aggSig, signers)

	if len(ret) == 0 {
		panic("no return value specified for VerifyAggregate")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func([]byte, []byte, []PubKey) error); ok {
		r0 = rf(payload, aggSig, signers)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockHost_VerifyAggregate_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'VerifyAggregate'
type MockHost_VerifyAggregate_Call struct {
	*mock.Call
}

// VerifyAggregate is a helper method to define mock.On call
//   - payload []byte
//   - aggSig []byte
//   - signers []PubKey
func (_e *MockHost_Expecter) VerifyAggregate(payload interface{}, aggSig interface{}, signers interface{}) *MockHost_VerifyAggregate_Call {
	return &MockHost_VerifyAggregate_Call{Call: _e.mock.On("VerifyAggregate", payload, aggSig, signers)}
}

func (_c *MockHost_VerifyAggregate_Call) Run(run func(payload []byte, aggSig []byte, signers []PubKey)) *MockHost_VerifyAggregate_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].([]byte), args[1].([]byte), args[2].([]PubKey))
	})
	return _c
}

func (_c *MockHost_VerifyAggregate_Call) Return(_a0 error) *MockHost_VerifyAggregate_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockHost_VerifyAggregate_Call) RunAndReturn(run func([]byte, []byte, []PubKey) error) *MockHost_VerifyAggregate_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockHost creates a new instance of MockHost. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockHost(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockHost {
	mock := &MockHost{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
