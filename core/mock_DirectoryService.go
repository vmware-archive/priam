package core

import "github.com/stretchr/testify/mock"

type MockDirectoryService struct {
	mock.Mock
}

// AddEntity provides a mock function with given fields: ctx, entity
func (_m *MockDirectoryService) AddEntity(ctx *HttpContext, entity interface{}) error {
	ret := _m.Called(ctx, entity)

	var r0 error
	if rf, ok := ret.Get(0).(func(*HttpContext, interface{}) error); ok {
		r0 = rf(ctx, entity)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DisplayEntity provides a mock function with given fields: ctx, name
func (_m *MockDirectoryService) DisplayEntity(ctx *HttpContext, name string) {
	_m.Called(ctx, name)
}

// UpdateEntity provides a mock function with given fields: ctx, name, entity
func (_m *MockDirectoryService) UpdateEntity(ctx *HttpContext, name string, entity interface{}) {
	_m.Called(ctx, name, entity)
}

// DeleteEntity provides a mock function with given fields: ctx, name
func (_m *MockDirectoryService) DeleteEntity(ctx *HttpContext, name string) {
	_m.Called(ctx, name)
}

// ListEntities provides a mock function with given fields: ctx, count, filter
func (_m *MockDirectoryService) ListEntities(ctx *HttpContext, count int, filter string) {
	_m.Called(ctx, count, filter)
}

// LoadEntities provides a mock function with given fields: ctx, fileName
func (_m *MockDirectoryService) LoadEntities(ctx *HttpContext, fileName string) {
	_m.Called(ctx, fileName)
}
