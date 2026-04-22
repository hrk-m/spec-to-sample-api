// Package mocks provides test doubles for the group package.
package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// MockUserRepository is a testify mock for group.UserRepository.
type MockUserRepository struct {
	mock.Mock
}

// CountByIDs returns the count of existing users for the given IDs.
func (m *MockUserRepository) CountByIDs(ctx context.Context, ids []uint64) (int, error) {
	args := m.Called(ctx, ids)
	return args.Int(0), args.Error(1)
}
