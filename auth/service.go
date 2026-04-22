// Package auth implements the authentication use case.
package auth

import (
	"context"

	"github.com/hrk-m/spec-to-dev-workflow/sample-api/domain"
)

// UserRepository defines the interface for user data access required by auth.
type UserRepository interface {
	GetByUUID(ctx context.Context, uuid string) (domain.User, error)
}

// Service handles authentication business logic.
type Service struct {
	repo UserRepository
}

// NewService returns a new Service instance.
func NewService(repo UserRepository) *Service {
	return &Service{repo: repo}
}

// GetByUUID returns the user with the given UUID.
func (s *Service) GetByUUID(ctx context.Context, uuid string) (domain.User, error) {
	return s.repo.GetByUUID(ctx, uuid)
}
