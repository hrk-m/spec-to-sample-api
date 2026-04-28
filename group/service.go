// Package group implements the group use case.
package group

import (
	"context"
	"strings"

	"github.com/hrk-m/spec-to-dev-workflow/sample-api/domain"
)

const (
	minLimit  = 1
	maxLimit  = 500
	minOffset = 0

	minID          = 1
	minMemberLimit = 1
	maxMemberLimit = 500

	maxNameLength = 100

	maxComponentGroups = 10
	maxDepthNodes      = 5
)

// GroupRepository defines the interface for group data access.
type GroupRepository interface {
	ListGroups(ctx context.Context, q string, limit, offset int) ([]domain.Group, int, error)
	GetByID(ctx context.Context, id uint64) (domain.Group, error)
	ListGroupMembers(ctx context.Context, id uint64, limit, offset int, q string) ([]domain.User, int, error)
	Store(ctx context.Context, name, description string, userID uint64) (domain.Group, error)
	Update(ctx context.Context, id uint64, name, description string, userID uint64) (*domain.Group, error)
	Delete(ctx context.Context, id uint64, userID uint64) error
	ListNonGroupMembers(ctx context.Context, groupID uint64, limit, offset int, q string) ([]domain.User, int, error)
	AddGroupMembers(ctx context.Context, groupID uint64, userIDs []uint64) ([]domain.User, error)
	RemoveGroupMembers(ctx context.Context, groupID uint64, userIDs []uint64) error
}

// UserRepository defines the interface for user data access used by the group service.
type UserRepository interface {
	CountByIDs(ctx context.Context, ids []uint64) (int, error)
}

// GroupRelationRepository defines the interface for group relation data access.
type GroupRelationRepository interface {
	GetAncestorIDs(ctx context.Context, groupID uint64) ([]uint64, error)
	GetDescendantIDs(ctx context.Context, groupID uint64) ([]uint64, error)
	CountComponentGroups(ctx context.Context, groupID uint64) (int, error)
	MaxDepthInComponent(ctx context.Context, parentGroupID, childGroupID uint64) (int, error)
	CreateRelation(ctx context.Context, parentGroupID, childGroupID uint64) (domain.GroupRelation, error)
	ListChildren(ctx context.Context, parentGroupID uint64) ([]domain.Group, error)
	DeleteRelation(ctx context.Context, parentGroupID, childGroupID uint64) error
}

// Service handles group business logic.
type Service struct {
	repo        GroupRepository
	userRepo    UserRepository
	relationRepo GroupRelationRepository
}

// NewService returns a new Service instance.
func NewService(repo GroupRepository, userRepo UserRepository) *Service {
	return &Service{repo: repo, userRepo: userRepo}
}

// NewServiceWithRelation returns a new Service instance with a GroupRelationRepository.
func NewServiceWithRelation(repo GroupRepository, userRepo UserRepository, relationRepo GroupRelationRepository) *Service {
	return &Service{repo: repo, userRepo: userRepo, relationRepo: relationRepo}
}

// GetByID returns a group by its ID along with its direct child subgroups.
func (s *Service) GetByID(ctx context.Context, id uint64) (domain.Group, []domain.Group, error) {
	if id < minID {
		return domain.Group{}, nil, domain.ErrBadParamInput
	}

	grp, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return domain.Group{}, nil, err
	}

	if s.relationRepo == nil {
		return grp, []domain.Group{}, nil
	}

	children, err := s.relationRepo.ListChildren(ctx, id)
	if err != nil {
		return domain.Group{}, nil, domain.ErrInternalServerError
	}

	if children == nil {
		children = []domain.Group{}
	}

	return grp, children, nil
}

// ListGroupMembers returns a paginated list of members for a group.
func (s *Service) ListGroupMembers(ctx context.Context, id uint64, limit, offset int, q string) ([]domain.User, int, error) {
	if id < minID {
		return nil, 0, domain.ErrBadParamInput
	}
	if limit < minMemberLimit || limit > maxMemberLimit {
		return nil, 0, domain.ErrBadParamInput
	}

	// Check group existence first.
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return nil, 0, err
	}

	members, total, err := s.repo.ListGroupMembers(ctx, id, limit, offset, q)
	if err != nil {
		return nil, 0, err
	}

	if members == nil {
		members = []domain.User{}
	}

	return members, total, nil
}

// Store creates a new group after validating the name.
func (s *Service) Store(ctx context.Context, name, description string, userID uint64) (domain.Group, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > maxNameLength {
		return domain.Group{}, domain.ErrBadParamInput
	}

	return s.repo.Store(ctx, name, description, userID)
}

// Update updates a group's name and description by ID.
func (s *Service) Update(ctx context.Context, id uint64, name, description string, userID uint64) (*domain.Group, error) {
	if id < minID {
		return nil, domain.ErrBadParamInput
	}

	name = strings.TrimSpace(name)
	if name == "" || len(name) > maxNameLength {
		return nil, domain.ErrBadParamInput
	}

	return s.repo.Update(ctx, id, name, description, userID)
}

// Delete soft-deletes a group by ID.
func (s *Service) Delete(ctx context.Context, id uint64, userID uint64) error {
	if id < minID {
		return domain.ErrBadParamInput
	}

	return s.repo.Delete(ctx, id, userID)
}

// ListGroups returns a paginated list of groups filtered by q keyword.
func (s *Service) ListGroups(ctx context.Context, q string, limit, offset int) ([]domain.Group, int, error) {
	if limit < minLimit || limit > maxLimit {
		return nil, 0, domain.ErrBadParamInput
	}
	if offset < minOffset {
		return nil, 0, domain.ErrBadParamInput
	}

	return s.repo.ListGroups(ctx, q, limit, offset)
}

// ListNonGroupMembers returns a paginated list of users not in the given group.
func (s *Service) ListNonGroupMembers(ctx context.Context, groupID uint64, limit, offset int, q string) ([]domain.User, int, error) {
	if groupID < minID {
		return nil, 0, domain.ErrBadParamInput
	}
	if limit < minMemberLimit || limit > maxMemberLimit {
		return nil, 0, domain.ErrBadParamInput
	}

	// Check group existence first.
	if _, err := s.repo.GetByID(ctx, groupID); err != nil {
		return nil, 0, err
	}

	users, total, err := s.repo.ListNonGroupMembers(ctx, groupID, limit, offset, q)
	if err != nil {
		return nil, 0, err
	}

	if users == nil {
		users = []domain.User{}
	}

	return users, total, nil
}

// AddGroupMembers adds users to a group and returns the added members.
func (s *Service) AddGroupMembers(ctx context.Context, groupID uint64, userIDs []uint64) ([]domain.User, error) {
	// Deduplicate userIDs so that COUNT(DISTINCT id) comparison is accurate.
	userIDs = deduplicateUint64(userIDs)

	// Check group existence.
	if _, err := s.repo.GetByID(ctx, groupID); err != nil {
		return nil, err
	}

	// Check all users exist with a single COUNT query.
	count, err := s.userRepo.CountByIDs(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	if count != len(userIDs) {
		return nil, domain.ErrNotFound
	}

	return s.repo.AddGroupMembers(ctx, groupID, userIDs)
}

// RemoveGroupMembers removes the given users from a group.
func (s *Service) RemoveGroupMembers(ctx context.Context, groupID uint64, userIDs []uint64) error {
	// Deduplicate userIDs so that COUNT comparison is accurate.
	userIDs = deduplicateUint64(userIDs)

	// Check group existence.
	if _, err := s.repo.GetByID(ctx, groupID); err != nil {
		return err
	}

	return s.repo.RemoveGroupMembers(ctx, groupID, userIDs)
}

// CreateSubGroup adds child as a subgroup of parent after validating constraints.
func (s *Service) CreateSubGroup(ctx context.Context, parentGroupID, childGroupID uint64) (domain.GroupRelation, error) {
	if childGroupID < minID {
		return domain.GroupRelation{}, domain.ErrBadParamInput
	}

	if parentGroupID == childGroupID {
		return domain.GroupRelation{}, domain.ErrBadParamInput
	}

	// Verify parent exists.
	if _, err := s.repo.GetByID(ctx, parentGroupID); err != nil {
		return domain.GroupRelation{}, err
	}

	// Verify child exists.
	if _, err := s.repo.GetByID(ctx, childGroupID); err != nil {
		return domain.GroupRelation{}, err
	}

	// Cycle detection: adding parent→child would create a cycle if child is already an ancestor of parent.
	// Equivalently: parent's ancestor set must not contain child, and child's descendant set must not contain parent.
	ancestorIDs, err := s.relationRepo.GetAncestorIDs(ctx, parentGroupID)
	if err != nil {
		return domain.GroupRelation{}, err
	}

	descendantIDs, err := s.relationRepo.GetDescendantIDs(ctx, childGroupID)
	if err != nil {
		return domain.GroupRelation{}, err
	}

	for _, id := range ancestorIDs {
		if id == childGroupID {
			return domain.GroupRelation{}, domain.ErrBadParamInput
		}
	}

	for _, id := range descendantIDs {
		if id == parentGroupID {
			return domain.GroupRelation{}, domain.ErrBadParamInput
		}
	}

	// Connected component size check (after adding: +1 for child's component).
	componentCount, err := s.relationRepo.CountComponentGroups(ctx, parentGroupID)
	if err != nil {
		return domain.GroupRelation{}, err
	}

	if componentCount+1 > maxComponentGroups {
		return domain.GroupRelation{}, domain.ErrBadParamInput
	}

	// Depth check.
	maxDepth, err := s.relationRepo.MaxDepthInComponent(ctx, parentGroupID, childGroupID)
	if err != nil {
		return domain.GroupRelation{}, err
	}

	if maxDepth > maxDepthNodes {
		return domain.GroupRelation{}, domain.ErrBadParamInput
	}

	return s.relationRepo.CreateRelation(ctx, parentGroupID, childGroupID)
}

// DeleteSubGroup removes the parent-child relation between the given groups.
func (s *Service) DeleteSubGroup(ctx context.Context, parentGroupID, childGroupID uint64) error {
	return s.relationRepo.DeleteRelation(ctx, parentGroupID, childGroupID)
}

// deduplicateUint64 returns a new slice with duplicate values removed, preserving order.
func deduplicateUint64(ids []uint64) []uint64 {
	seen := make(map[uint64]struct{}, len(ids))
	result := make([]uint64, 0, len(ids))

	for _, id := range ids {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			result = append(result, id)
		}
	}

	return result
}
