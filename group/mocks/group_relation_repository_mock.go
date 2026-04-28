// Package mocks provides test doubles for the group package.
package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/hrk-m/spec-to-dev-workflow/sample-api/domain"
)

// MockGroupRelationRepository is a testify mock for group.GroupRelationRepository.
type MockGroupRelationRepository struct {
	mock.Mock
}

func (m *MockGroupRelationRepository) GetAncestorIDs(ctx context.Context, groupID uint64) ([]uint64, error) {
	args := m.Called(ctx, groupID)
	return args.Get(0).([]uint64), args.Error(1)
}

func (m *MockGroupRelationRepository) GetDescendantIDs(ctx context.Context, groupID uint64) ([]uint64, error) {
	args := m.Called(ctx, groupID)
	return args.Get(0).([]uint64), args.Error(1)
}

func (m *MockGroupRelationRepository) CountComponentGroups(ctx context.Context, groupID uint64) (int, error) {
	args := m.Called(ctx, groupID)
	return args.Int(0), args.Error(1)
}

func (m *MockGroupRelationRepository) MaxDepthInComponent(ctx context.Context, parentGroupID, childGroupID uint64) (int, error) {
	args := m.Called(ctx, parentGroupID, childGroupID)
	return args.Int(0), args.Error(1)
}

func (m *MockGroupRelationRepository) CreateRelation(ctx context.Context, parentGroupID, childGroupID uint64) (domain.GroupRelation, error) {
	args := m.Called(ctx, parentGroupID, childGroupID)
	return args.Get(0).(domain.GroupRelation), args.Error(1)
}

func (m *MockGroupRelationRepository) ListChildren(ctx context.Context, parentGroupID uint64) ([]domain.Group, error) {
	args := m.Called(ctx, parentGroupID)
	return args.Get(0).([]domain.Group), args.Error(1)
}
