package user_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/hrk-m/spec-to-dev-workflow/sample-api/domain"
	"github.com/hrk-m/spec-to-dev-workflow/sample-api/user"
	"github.com/hrk-m/spec-to-dev-workflow/sample-api/user/mocks"
)

func TestService_ListUsers_OK(t *testing.T) {
	repo := new(mocks.MockUserRepository)
	svc := user.NewService(repo)

	expected := []domain.User{
		{ID: 1, UUID: "550e8400-e29b-41d4-a716-446655440001", FirstName: "Taro", LastName: "Yamada"},
		{ID: 2, UUID: "550e8400-e29b-41d4-a716-446655440002", FirstName: "Hanako", LastName: "Suzuki"},
	}
	repo.On("ListUsers", mock.Anything, "Suzuki", 500, 0).Return(expected[1:2], 15, nil)

	result, total, err := svc.ListUsers(context.Background(), " Suzuki ", 500, 0)

	assert.NoError(t, err)
	assert.Equal(t, []domain.User{expected[1]}, result)
	assert.Equal(t, 15, total)
	repo.AssertExpectations(t)
}

func TestService_ListUsers_InvalidLimitTooLow(t *testing.T) {
	repo := new(mocks.MockUserRepository)
	svc := user.NewService(repo)

	_, _, err := svc.ListUsers(context.Background(), "", 0, 0)

	assert.ErrorIs(t, err, domain.ErrBadParamInput)
	repo.AssertNotCalled(t, "ListUsers")
}

func TestService_ListUsers_InvalidLimitTooHigh(t *testing.T) {
	repo := new(mocks.MockUserRepository)
	svc := user.NewService(repo)

	_, _, err := svc.ListUsers(context.Background(), "", 501, 0)

	assert.ErrorIs(t, err, domain.ErrBadParamInput)
	repo.AssertNotCalled(t, "ListUsers")
}

func TestService_ListUsers_InvalidOffset(t *testing.T) {
	repo := new(mocks.MockUserRepository)
	svc := user.NewService(repo)

	_, _, err := svc.ListUsers(context.Background(), "", 500, -1)

	assert.ErrorIs(t, err, domain.ErrBadParamInput)
	repo.AssertNotCalled(t, "ListUsers")
}

func TestService_ListUsers_RepositoryError(t *testing.T) {
	repo := new(mocks.MockUserRepository)
	svc := user.NewService(repo)

	repo.On("ListUsers", mock.Anything, "", 500, 0).
		Return([]domain.User(nil), 0, domain.ErrInternalServerError)

	_, _, err := svc.ListUsers(context.Background(), " ", 500, 0)

	assert.ErrorIs(t, err, domain.ErrInternalServerError)
	repo.AssertExpectations(t)
}

func TestService_ListUsers_EmptyResult(t *testing.T) {
	repo := new(mocks.MockUserRepository)
	svc := user.NewService(repo)

	repo.On("ListUsers", mock.Anything, "", 500, 0).
		Return([]domain.User(nil), 0, nil)

	result, total, err := svc.ListUsers(context.Background(), "", 500, 0)

	assert.NoError(t, err)
	assert.Empty(t, result)
	assert.NotNil(t, result)
	assert.Equal(t, 0, total)
	repo.AssertExpectations(t)
}

func TestService_GetUser_OK(t *testing.T) {
	repo := new(mocks.MockUserRepository)
	svc := user.NewService(repo)

	expected := &domain.User{ID: 1, UUID: "550e8400-e29b-41d4-a716-446655440001", FirstName: "Taro", LastName: "Yamada"}
	repo.On("GetByID", mock.Anything, uint64(1)).Return(expected, nil)

	result, err := svc.GetUser(context.Background(), uint64(1))

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	repo.AssertExpectations(t)
}

func TestService_GetUser_NotFound(t *testing.T) {
	repo := new(mocks.MockUserRepository)
	svc := user.NewService(repo)

	repo.On("GetByID", mock.Anything, uint64(9999)).Return((*domain.User)(nil), domain.ErrNotFound)

	_, err := svc.GetUser(context.Background(), uint64(9999))

	assert.ErrorIs(t, err, domain.ErrNotFound)
	repo.AssertExpectations(t)
}

func TestService_GetUser_RepositoryError(t *testing.T) {
	repo := new(mocks.MockUserRepository)
	svc := user.NewService(repo)

	repo.On("GetByID", mock.Anything, uint64(1)).Return((*domain.User)(nil), domain.ErrInternalServerError)

	_, err := svc.GetUser(context.Background(), uint64(1))

	assert.ErrorIs(t, err, domain.ErrInternalServerError)
	repo.AssertExpectations(t)
}
