// Package mocks provides test doubles for the group package.
package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/hrk-m/spec-to-dev-workflow/sample-api/domain"
)

// MockGroupRepository is a testify mock for group.GroupRepository.
type MockGroupRepository struct {
	mock.Mock
}

func (m *MockGroupRepository) ListGroups(ctx context.Context, q string, limit, offset int) ([]domain.Group, int, error) {
	args := m.Called(ctx, q, limit, offset)
	return args.Get(0).([]domain.Group), args.Int(1), args.Error(2)
}

func (m *MockGroupRepository) GetByID(ctx context.Context, id uint64) (domain.Group, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(domain.Group), args.Error(1)
}

func (m *MockGroupRepository) ListGroupMembers(ctx context.Context, id uint64, limit, offset int, q string) ([]domain.GroupMember, int, error) {
	args := m.Called(ctx, id, limit, offset, q)
	members, _ := args.Get(0).([]domain.GroupMember)
	return members, args.Int(1), args.Error(2)
}

func (m *MockGroupRepository) Store(ctx context.Context, name, description string, userID uint64) (domain.Group, error) {
	args := m.Called(ctx, name, description, userID)
	return args.Get(0).(domain.Group), args.Error(1)
}

func (m *MockGroupRepository) Update(ctx context.Context, id uint64, name, description string, userID uint64) (*domain.Group, error) {
	args := m.Called(ctx, id, name, description, userID)
	g, _ := args.Get(0).(*domain.Group)
	return g, args.Error(1)
}

func (m *MockGroupRepository) Delete(ctx context.Context, id uint64, userID uint64) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *MockGroupRepository) ListNonGroupMembers(ctx context.Context, groupID uint64, limit, offset int, q string) ([]domain.User, int, error) {
	args := m.Called(ctx, groupID, limit, offset, q)
	return args.Get(0).([]domain.User), args.Int(1), args.Error(2)
}

func (m *MockGroupRepository) AddGroupMembers(ctx context.Context, groupID uint64, userIDs []uint64) ([]domain.User, error) {
	args := m.Called(ctx, groupID, userIDs)
	return args.Get(0).([]domain.User), args.Error(1)
}

func (m *MockGroupRepository) RemoveGroupMembers(ctx context.Context, groupID uint64, userIDs []uint64) error {
	args := m.Called(ctx, groupID, userIDs)
	return args.Error(0)
}
