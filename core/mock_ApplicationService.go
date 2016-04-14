package core

import "github.com/stretchr/testify/mock"

type MockApplicationService struct {
	mock.Mock
}

// Display provides a mock function with given fields: ctx, name
func (_m *MockApplicationService) Display(ctx *HttpContext, name string) {
	_m.Called(ctx, name)
}

// Delete provides a mock function with given fields: ctx, name
func (_m *MockApplicationService) Delete(ctx *HttpContext, name string) {
	_m.Called(ctx, name)
}

// List provides a mock function with given fields: ctx, count, filter
func (_m *MockApplicationService) List(ctx *HttpContext, count int, filter string) {
	_m.Called(ctx, count, filter)
}

// Publish provides a mock function with given fields: ctx, manifestFile
func (_m *MockApplicationService) Publish(ctx *HttpContext, manifestFile string) {
	_m.Called(ctx, manifestFile)
}
