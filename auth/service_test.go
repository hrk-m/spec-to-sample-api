package auth_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/hrk-m/spec-to-dev-workflow/sample-api/auth"
	"github.com/hrk-m/spec-to-dev-workflow/sample-api/auth/mocks"
	"github.com/hrk-m/spec-to-dev-workflow/sample-api/domain"
)

func TestService_GetByUUID_Success(t *testing.T) {
	repo := new(mocks.MockUserRepository)
	svc := auth.NewService(repo)

	expected := domain.User{ID: 1, UUID: "test-uuid-1234", FirstName: "Taro", LastName: "Yamada"}
	repo.On("GetByUUID", mock.Anything, "test-uuid-1234").Return(expected, nil)

	result, err := svc.GetByUUID(context.Background(), "test-uuid-1234")

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	repo.AssertExpectations(t)
}

func TestService_GetByUUID_NotFound(t *testing.T) {
	repo := new(mocks.MockUserRepository)
	svc := auth.NewService(repo)

	repo.On("GetByUUID", mock.Anything, "nonexistent-uuid").Return(domain.User{}, domain.ErrNotFound)

	_, err := svc.GetByUUID(context.Background(), "nonexistent-uuid")

	assert.ErrorIs(t, err, domain.ErrNotFound)
	repo.AssertExpectations(t)
}
