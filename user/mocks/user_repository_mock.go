// Package mocks provides test doubles for the user package.
package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/hrk-m/spec-to-dev-workflow/sample-api/domain"
)

// MockUserRepository is a testify mock for user.UserRepository.
type MockUserRepository struct {
	mock.Mock
}

// ListUsers returns a user list.
func (m *MockUserRepository) ListUsers(ctx context.Context, q string, limit, offset int) ([]domain.User, int, error) {
	args := m.Called(ctx, q, limit, offset)
	return args.Get(0).([]domain.User), args.Int(1), args.Error(2)
}

// GetByID returns a single user by ID.
func (m *MockUserRepository) GetByID(ctx context.Context, id uint64) (*domain.User, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*domain.User), args.Error(1)
}
