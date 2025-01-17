// Code generated by mockery v0.0.0-dev. DO NOT EDIT.

package toolchain

import (
	filter "github.com/odahu/odahu-flow/packages/operator/pkg/utils/filter"
	mock "github.com/stretchr/testify/mock"

	training "github.com/odahu/odahu-flow/packages/operator/pkg/apis/training"
)

// MockToolchainService is an autogenerated mock type for the Service type
type MockToolchainService struct {
	mock.Mock
}

// CreateToolchainIntegration provides a mock function with given fields: md
func (_m *MockToolchainService) CreateToolchainIntegration(md *training.ToolchainIntegration) error {
	ret := _m.Called(md)

	var r0 error
	if rf, ok := ret.Get(0).(func(*training.ToolchainIntegration) error); ok {
		r0 = rf(md)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteToolchainIntegration provides a mock function with given fields: name
func (_m *MockToolchainService) DeleteToolchainIntegration(name string) error {
	ret := _m.Called(name)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(name)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetToolchainIntegration provides a mock function with given fields: name
func (_m *MockToolchainService) GetToolchainIntegration(name string) (*training.ToolchainIntegration, error) {
	ret := _m.Called(name)

	var r0 *training.ToolchainIntegration
	if rf, ok := ret.Get(0).(func(string) *training.ToolchainIntegration); ok {
		r0 = rf(name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*training.ToolchainIntegration)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetToolchainIntegrationList provides a mock function with given fields: options
func (_m *MockToolchainService) GetToolchainIntegrationList(options ...filter.ListOption) ([]training.ToolchainIntegration, error) {
	_va := make([]interface{}, len(options))
	for _i := range options {
		_va[_i] = options[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 []training.ToolchainIntegration
	if rf, ok := ret.Get(0).(func(...filter.ListOption) []training.ToolchainIntegration); ok {
		r0 = rf(options...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]training.ToolchainIntegration)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(...filter.ListOption) error); ok {
		r1 = rf(options...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UpdateToolchainIntegration provides a mock function with given fields: md
func (_m *MockToolchainService) UpdateToolchainIntegration(md *training.ToolchainIntegration) error {
	ret := _m.Called(md)

	var r0 error
	if rf, ok := ret.Get(0).(func(*training.ToolchainIntegration) error); ok {
		r0 = rf(md)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
