// Package user implements the user use case.
package user

import (
	"context"
	"strings"

	"github.com/hrk-m/spec-to-dev-workflow/sample-api/domain"
)

const (
	minLimit  = 1
	maxLimit  = 500
	minOffset = 0
)

// UserRepository defines the interface for user data access.
type UserRepository interface {
	ListUsers(ctx context.Context, q string, limit, offset int) ([]domain.User, int, error)
}

// Service handles user business logic.
type Service struct {
	repo UserRepository
}

// NewService returns a new Service instance.
func NewService(repo UserRepository) *Service {
	return &Service{repo: repo}
}

// ListUsers returns a paginated list of active users.
func (s *Service) ListUsers(ctx context.Context, q string, limit, offset int) ([]domain.User, int, error) {
	if limit < minLimit || limit > maxLimit {
		return nil, 0, domain.ErrBadParamInput
	}
	if offset < minOffset {
		return nil, 0, domain.ErrBadParamInput
	}

	q = strings.TrimSpace(q)

	users, total, err := s.repo.ListUsers(ctx, q, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	if users == nil {
		users = []domain.User{}
	}

	return users, total, nil
}
