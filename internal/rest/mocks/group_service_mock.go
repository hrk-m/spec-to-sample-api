// Package mocks provides test doubles for the rest package.
package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/hrk-m/spec-to-dev-workflow/sample-api/domain"
)

// MockGroupService is a testify mock for rest.GroupService.
type MockGroupService struct {
	mock.Mock
}

func (m *MockGroupService) ListGroups(ctx context.Context, q string, limit, offset int) ([]domain.Group, int, error) {
	args := m.Called(ctx, q, limit, offset)
	return args.Get(0).([]domain.Group), args.Int(1), args.Error(2)
}

func (m *MockGroupService) GetByID(ctx context.Context, id uint64) (domain.Group, []domain.Group, error) {
	args := m.Called(ctx, id)
	groups, _ := args.Get(1).([]domain.Group)
	return args.Get(0).(domain.Group), groups, args.Error(2)
}

func (m *MockGroupService) ListGroupMembers(ctx context.Context, id uint64, limit, offset int, q string) ([]domain.GroupMember, int, error) {
	args := m.Called(ctx, id, limit, offset, q)
	members, _ := args.Get(0).([]domain.GroupMember)
	return members, args.Int(1), args.Error(2)
}

func (m *MockGroupService) Store(ctx context.Context, name, description string, userID uint64) (domain.Group, error) {
	args := m.Called(ctx, name, description, userID)
	return args.Get(0).(domain.Group), args.Error(1)
}

func (m *MockGroupService) Update(ctx context.Context, id uint64, name, description string, userID uint64) (*domain.Group, error) {
	args := m.Called(ctx, id, name, description, userID)
	g, _ := args.Get(0).(*domain.Group)
	return g, args.Error(1)
}

func (m *MockGroupService) Delete(ctx context.Context, id uint64, userID uint64) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *MockGroupService) ListNonGroupMembers(ctx context.Context, groupID uint64, limit, offset int, q string) ([]domain.User, int, error) {
	args := m.Called(ctx, groupID, limit, offset, q)
	return args.Get(0).([]domain.User), args.Int(1), args.Error(2)
}

func (m *MockGroupService) AddGroupMembers(ctx context.Context, groupID uint64, userIDs []uint64) ([]domain.User, error) {
	args := m.Called(ctx, groupID, userIDs)
	return args.Get(0).([]domain.User), args.Error(1)
}

func (m *MockGroupService) RemoveGroupMembers(ctx context.Context, groupID uint64, userIDs []uint64) error {
	args := m.Called(ctx, groupID, userIDs)
	return args.Error(0)
}

func (m *MockGroupService) CreateSubGroup(ctx context.Context, parentGroupID, childGroupID uint64) (domain.GroupRelation, error) {
	args := m.Called(ctx, parentGroupID, childGroupID)
	return args.Get(0).(domain.GroupRelation), args.Error(1)
}

func (m *MockGroupService) DeleteSubGroup(ctx context.Context, parentGroupID, childGroupID uint64) error {
	args := m.Called(ctx, parentGroupID, childGroupID)
	return args.Error(0)
}
