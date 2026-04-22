// Package mocks provides test doubles for the auth package.
package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/hrk-m/spec-to-dev-workflow/sample-api/domain"
)

// MockUserRepository is a testify mock for auth.UserRepository.
type MockUserRepository struct {
	mock.Mock
}

// GetByUUID returns a user by UUID.
func (m *MockUserRepository) GetByUUID(ctx context.Context, uuid string) (domain.User, error) {
	args := m.Called(ctx, uuid)
	return args.Get(0).(domain.User), args.Error(1)
}
