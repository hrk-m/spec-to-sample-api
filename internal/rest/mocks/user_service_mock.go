// Package mocks provides test doubles for the rest package.
package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/hrk-m/spec-to-dev-workflow/sample-api/domain"
)

// MockUserService is a testify mock for rest.UserService.
type MockUserService struct {
	mock.Mock
}

// ListUsers returns a user list.
func (m *MockUserService) ListUsers(ctx context.Context, q string, limit, offset int) ([]domain.User, int, error) {
	args := m.Called(ctx, q, limit, offset)
	return args.Get(0).([]domain.User), args.Int(1), args.Error(2)
}

// GetUser returns a single user by ID.
func (m *MockUserService) GetUser(ctx context.Context, id uint64) (*domain.User, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*domain.User), args.Error(1)
}
