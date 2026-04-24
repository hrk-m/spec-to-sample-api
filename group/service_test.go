package group_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/hrk-m/spec-to-dev-workflow/sample-api/domain"
	"github.com/hrk-m/spec-to-dev-workflow/sample-api/group"
	"github.com/hrk-m/spec-to-dev-workflow/sample-api/group/mocks"
)

func TestService_GetByID_OK(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	expected := domain.Group{ID: 1, Name: "dev-team", Description: "developers", MemberCount: 5}
	repo.On("GetByID", mock.Anything, uint64(1)).Return(expected, nil)

	result, err := svc.GetByID(context.Background(), 1)

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	repo.AssertExpectations(t)
}

func TestService_GetByID_NotFound(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	repo.On("GetByID", mock.Anything, uint64(9999)).
		Return(domain.Group{}, domain.ErrNotFound)

	_, err := svc.GetByID(context.Background(), 9999)

	assert.ErrorIs(t, err, domain.ErrNotFound)
	repo.AssertExpectations(t)
}

func TestService_GetByID_InvalidID(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	_, err := svc.GetByID(context.Background(), 0)

	assert.ErrorIs(t, err, domain.ErrBadParamInput)
	repo.AssertNotCalled(t, "GetByID")
}

func TestService_GetByID_RepositoryError(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	repo.On("GetByID", mock.Anything, uint64(1)).
		Return(domain.Group{}, domain.ErrInternalServerError)

	_, err := svc.GetByID(context.Background(), 1)

	assert.ErrorIs(t, err, domain.ErrInternalServerError)
	repo.AssertExpectations(t)
}

func TestService_ListGroupMembers_OK(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	groupResp := domain.Group{ID: 1, Name: "dev-team", Description: "developers", MemberCount: 2}
	repo.On("GetByID", mock.Anything, uint64(1)).Return(groupResp, nil)

	members := []domain.User{
		{ID: 1, UUID: "00000000-0000-0000-0000-000000000001", FirstName: "Taro", LastName: "Yamada"},
		{ID: 2, UUID: "00000000-0000-0000-0000-000000000002", FirstName: "Hanako", LastName: "Suzuki"},
	}
	repo.On("ListGroupMembers", mock.Anything, uint64(1), 500, 0, "").
		Return(members, 2, nil)

	result, total, err := svc.ListGroupMembers(context.Background(), 1, 500, 0, "")

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, 2, total)
	repo.AssertExpectations(t)
}

func TestService_ListGroupMembers_WithSearch(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	groupResp := domain.Group{ID: 1, Name: "dev-team", Description: "developers", MemberCount: 2}
	repo.On("GetByID", mock.Anything, uint64(1)).Return(groupResp, nil)

	members := []domain.User{
		{ID: 1, UUID: "00000000-0000-0000-0000-000000000001", FirstName: "Taro", LastName: "Yamada"},
	}
	repo.On("ListGroupMembers", mock.Anything, uint64(1), 500, 0, "Yamada").
		Return(members, 2, nil)

	result, total, err := svc.ListGroupMembers(context.Background(), 1, 500, 0, "Yamada")

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, 2, total)
	repo.AssertExpectations(t)
}

func TestService_ListGroupMembers_GroupNotFound(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	repo.On("GetByID", mock.Anything, uint64(9999)).
		Return(domain.Group{}, domain.ErrNotFound)

	_, _, err := svc.ListGroupMembers(context.Background(), 9999, 500, 0, "")

	assert.ErrorIs(t, err, domain.ErrNotFound)
	repo.AssertNotCalled(t, "ListGroupMembers")
}

func TestService_ListGroupMembers_InvalidID(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	_, _, err := svc.ListGroupMembers(context.Background(), 0, 500, 0, "")

	assert.ErrorIs(t, err, domain.ErrBadParamInput)
	repo.AssertNotCalled(t, "GetByID")
	repo.AssertNotCalled(t, "ListGroupMembers")
}

func TestService_ListGroupMembers_InvalidLimitTooLow(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	_, _, err := svc.ListGroupMembers(context.Background(), 1, 0, 0, "")

	assert.ErrorIs(t, err, domain.ErrBadParamInput)
	repo.AssertNotCalled(t, "GetByID")
}

func TestService_ListGroupMembers_InvalidLimitTooHigh(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	_, _, err := svc.ListGroupMembers(context.Background(), 1, 501, 0, "")

	assert.ErrorIs(t, err, domain.ErrBadParamInput)
	repo.AssertNotCalled(t, "GetByID")
}

func TestService_ListGroupMembers_RepositoryError(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	groupResp := domain.Group{ID: 1, Name: "dev-team", Description: "developers", MemberCount: 2}
	repo.On("GetByID", mock.Anything, uint64(1)).Return(groupResp, nil)
	repo.On("ListGroupMembers", mock.Anything, uint64(1), 500, 0, "").
		Return([]domain.User(nil), 0, domain.ErrInternalServerError)

	_, _, err := svc.ListGroupMembers(context.Background(), 1, 500, 0, "")

	assert.ErrorIs(t, err, domain.ErrInternalServerError)
	repo.AssertExpectations(t)
}

func TestService_ListGroupMembers_EmptyResult(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	groupResp := domain.Group{ID: 1, Name: "dev-team", Description: "developers", MemberCount: 0}
	repo.On("GetByID", mock.Anything, uint64(1)).Return(groupResp, nil)
	repo.On("ListGroupMembers", mock.Anything, uint64(1), 500, 0, "").
		Return([]domain.User(nil), 0, nil)

	result, total, err := svc.ListGroupMembers(context.Background(), 1, 500, 0, "")

	assert.NoError(t, err)
	assert.Empty(t, result)
	assert.NotNil(t, result)
	assert.Equal(t, 0, total)
	repo.AssertExpectations(t)
}

func TestService_ListGroups_OK(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	groups := []domain.Group{
		{ID: 1, Name: "group1", Description: "desc1", MemberCount: 1},
	}
	repo.On("ListGroups", mock.Anything, "", 500, 0).Return(groups, 1, nil)

	result, total, err := svc.ListGroups(context.Background(), "", 500, 0)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "group1", result[0].Name)
	assert.Equal(t, 1, total)
	repo.AssertExpectations(t)
}

func TestService_ListGroups_WithSearch(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	groups := []domain.Group{
		{ID: 2, Name: "dev-team", Description: "developers", MemberCount: 0},
	}
	repo.On("ListGroups", mock.Anything, "dev", 20, 0).Return(groups, 5, nil)

	result, total, err := svc.ListGroups(context.Background(), "dev", 20, 0)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "dev-team", result[0].Name)
	assert.Equal(t, 5, total)
	repo.AssertExpectations(t)
}

func TestService_ListGroups_WithOffset(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	groups := []domain.Group{
		{ID: 3, Name: "group3", Description: "desc3", MemberCount: 2},
	}
	repo.On("ListGroups", mock.Anything, "", 500, 500).Return(groups, 42, nil)

	result, total, err := svc.ListGroups(context.Background(), "", 500, 500)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, 42, total)
	repo.AssertExpectations(t)
}

func TestService_ListGroups_InvalidLimitTooLow(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	_, _, err := svc.ListGroups(context.Background(), "", 0, 0)

	assert.ErrorIs(t, err, domain.ErrBadParamInput)
	repo.AssertNotCalled(t, "ListGroups")
}

func TestService_ListGroups_InvalidLimitTooHigh(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	_, _, err := svc.ListGroups(context.Background(), "", 501, 0)

	assert.ErrorIs(t, err, domain.ErrBadParamInput)
	repo.AssertNotCalled(t, "ListGroups")
}

func TestService_ListGroups_InvalidOffsetNegative(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	_, _, err := svc.ListGroups(context.Background(), "", 500, -1)

	assert.ErrorIs(t, err, domain.ErrBadParamInput)
	repo.AssertNotCalled(t, "ListGroups")
}

func TestService_ListGroups_RepositoryError(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	repo.On("ListGroups", mock.Anything, "", 500, 0).
		Return([]domain.Group(nil), 0, domain.ErrInternalServerError)

	_, _, err := svc.ListGroups(context.Background(), "", 500, 0)

	assert.ErrorIs(t, err, domain.ErrInternalServerError)
	repo.AssertExpectations(t)
}

func TestService_Store_OK(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	expected := domain.Group{ID: 1, Name: "Test Group", Description: "A test group", MemberCount: 1}
	repo.On("Store", mock.Anything, "Test Group", "A test group", uint64(10)).Return(expected, nil)

	result, err := svc.Store(context.Background(), "Test Group", "A test group", uint64(10))

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	repo.AssertExpectations(t)
}

func TestService_Store_TrimsName(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	expected := domain.Group{ID: 1, Name: "Trimmed", Description: "", MemberCount: 1}
	repo.On("Store", mock.Anything, "Trimmed", "", uint64(10)).Return(expected, nil)

	result, err := svc.Store(context.Background(), "  Trimmed  ", "", uint64(10))

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	repo.AssertExpectations(t)
}

func TestService_Store_UserIDPropagated(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	expected := domain.Group{ID: 2, Name: "Another Group", Description: "", MemberCount: 1}
	repo.On("Store", mock.Anything, "Another Group", "", uint64(42)).Return(expected, nil)

	result, err := svc.Store(context.Background(), "Another Group", "", uint64(42))

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	repo.AssertExpectations(t)
}

func TestService_Store_EmptyName(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	_, err := svc.Store(context.Background(), "", "desc", uint64(1))

	assert.ErrorIs(t, err, domain.ErrBadParamInput)
	repo.AssertNotCalled(t, "Store")
}

func TestService_Store_WhitespaceOnlyName(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	_, err := svc.Store(context.Background(), "   ", "desc", uint64(1))

	assert.ErrorIs(t, err, domain.ErrBadParamInput)
	repo.AssertNotCalled(t, "Store")
}

func TestService_Store_NameTooLong(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	longName := make([]byte, 101)
	for i := range longName {
		longName[i] = 'a'
	}

	_, err := svc.Store(context.Background(), string(longName), "desc", uint64(1))

	assert.ErrorIs(t, err, domain.ErrBadParamInput)
	repo.AssertNotCalled(t, "Store")
}

func TestService_Store_RepositoryError(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	repo.On("Store", mock.Anything, "Valid", "desc", uint64(1)).
		Return(domain.Group{}, domain.ErrInternalServerError)

	_, err := svc.Store(context.Background(), "Valid", "desc", uint64(1))

	assert.ErrorIs(t, err, domain.ErrInternalServerError)
	repo.AssertExpectations(t)
}

func TestService_Update_OK(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	expected := &domain.Group{ID: 1, Name: "Updated Group", Description: "New desc", MemberCount: 3}
	repo.On("Update", mock.Anything, uint64(1), "Updated Group", "New desc", uint64(10)).Return(expected, nil)

	result, err := svc.Update(context.Background(), 1, "Updated Group", "New desc", uint64(10))

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	repo.AssertExpectations(t)
}

func TestService_Update_TrimsName(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	expected := &domain.Group{ID: 1, Name: "Trimmed", Description: "", MemberCount: 0}
	repo.On("Update", mock.Anything, uint64(1), "Trimmed", "", uint64(10)).Return(expected, nil)

	result, err := svc.Update(context.Background(), 1, "  Trimmed  ", "", uint64(10))

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	repo.AssertExpectations(t)
}

func TestService_Update_UserIDPropagated(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	expected := &domain.Group{ID: 1, Name: "Group", Description: "", MemberCount: 0}
	repo.On("Update", mock.Anything, uint64(1), "Group", "", uint64(42)).Return(expected, nil)

	result, err := svc.Update(context.Background(), 1, "Group", "", uint64(42))

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	repo.AssertExpectations(t)
}

func TestService_Update_EmptyName(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	_, err := svc.Update(context.Background(), 1, "", "desc", uint64(1))

	assert.ErrorIs(t, err, domain.ErrBadParamInput)
	repo.AssertNotCalled(t, "Update")
}

func TestService_Update_WhitespaceOnlyName(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	_, err := svc.Update(context.Background(), 1, "   ", "desc", uint64(1))

	assert.ErrorIs(t, err, domain.ErrBadParamInput)
	repo.AssertNotCalled(t, "Update")
}

func TestService_Update_NameTooLong(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	longName := make([]byte, 101)
	for i := range longName {
		longName[i] = 'a'
	}

	_, err := svc.Update(context.Background(), 1, string(longName), "desc", uint64(1))

	assert.ErrorIs(t, err, domain.ErrBadParamInput)
	repo.AssertNotCalled(t, "Update")
}

func TestService_Update_InvalidID(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	_, err := svc.Update(context.Background(), 0, "Valid", "desc", uint64(1))

	assert.ErrorIs(t, err, domain.ErrBadParamInput)
	repo.AssertNotCalled(t, "Update")
}

func TestService_Update_RepositoryError(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	repo.On("Update", mock.Anything, uint64(1), "Valid", "desc", uint64(1)).
		Return((*domain.Group)(nil), domain.ErrInternalServerError)

	_, err := svc.Update(context.Background(), 1, "Valid", "desc", uint64(1))

	assert.ErrorIs(t, err, domain.ErrInternalServerError)
	repo.AssertExpectations(t)
}

func TestService_Delete_InvalidID(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	err := svc.Delete(context.Background(), 0, uint64(1))

	assert.ErrorIs(t, err, domain.ErrBadParamInput)
	repo.AssertNotCalled(t, "Delete")
}

// Case #7: Normal - repository.Delete succeeds with userID correctly passed.
func TestService_Delete_OK(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	repo.On("Delete", mock.Anything, uint64(1), uint64(42)).Return(nil)

	err := svc.Delete(context.Background(), 1, uint64(42))

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

// Case #8: Error - repository.Delete returns ErrNotFound.
func TestService_Delete_NotFound(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	repo.On("Delete", mock.Anything, uint64(9999), uint64(1)).Return(domain.ErrNotFound)

	err := svc.Delete(context.Background(), 9999, uint64(1))

	assert.ErrorIs(t, err, domain.ErrNotFound)
	repo.AssertExpectations(t)
}

// Case #9: Error - repository.Delete returns DB error.
func TestService_Delete_RepositoryError(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	repo.On("Delete", mock.Anything, uint64(1), uint64(1)).Return(domain.ErrInternalServerError)

	err := svc.Delete(context.Background(), 1, uint64(1))

	assert.ErrorIs(t, err, domain.ErrInternalServerError)
	repo.AssertExpectations(t)
}

func TestService_ListNonGroupMembers_OK(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	groupResp := domain.Group{ID: 1, Name: "dev-team", Description: "developers", MemberCount: 1}
	repo.On("GetByID", mock.Anything, uint64(1)).Return(groupResp, nil)

	users := []domain.User{
		{ID: 2, UUID: "00000000-0000-0000-0000-000000000002", FirstName: "Hanako", LastName: "Suzuki"},
		{ID: 3, UUID: "00000000-0000-0000-0000-000000000003", FirstName: "Jiro", LastName: "Tanaka"},
	}
	repo.On("ListNonGroupMembers", mock.Anything, uint64(1), 500, 0, "").
		Return(users, 2, nil)

	result, total, err := svc.ListNonGroupMembers(context.Background(), uint64(1), 500, 0, "")

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, 2, total)
	repo.AssertExpectations(t)
}

func TestService_ListNonGroupMembers_WithSearch(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	groupResp := domain.Group{ID: 1, Name: "dev-team", Description: "developers", MemberCount: 1}
	repo.On("GetByID", mock.Anything, uint64(1)).Return(groupResp, nil)

	users := []domain.User{
		{ID: 3, UUID: "00000000-0000-0000-0000-000000000003", FirstName: "Jiro", LastName: "Tanaka"},
	}
	repo.On("ListNonGroupMembers", mock.Anything, uint64(1), 500, 0, "Tanaka").
		Return(users, 5, nil)

	result, total, err := svc.ListNonGroupMembers(context.Background(), uint64(1), 500, 0, "Tanaka")

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, 5, total)
	repo.AssertExpectations(t)
}

func TestService_ListNonGroupMembers_EmptyResult(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	groupResp := domain.Group{ID: 1, Name: "dev-team", Description: "developers", MemberCount: 3}
	repo.On("GetByID", mock.Anything, uint64(1)).Return(groupResp, nil)

	repo.On("ListNonGroupMembers", mock.Anything, uint64(1), 500, 0, "").
		Return([]domain.User(nil), 0, nil)

	result, total, err := svc.ListNonGroupMembers(context.Background(), uint64(1), 500, 0, "")

	assert.NoError(t, err)
	assert.Empty(t, result)
	assert.NotNil(t, result)
	assert.Equal(t, 0, total)
	repo.AssertExpectations(t)
}

func TestService_ListNonGroupMembers_GroupNotFound(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	repo.On("GetByID", mock.Anything, uint64(9999)).
		Return(domain.Group{}, domain.ErrNotFound)

	_, _, err := svc.ListNonGroupMembers(context.Background(), uint64(9999), 500, 0, "")

	assert.ErrorIs(t, err, domain.ErrNotFound)
	repo.AssertNotCalled(t, "ListNonGroupMembers")
}

func TestService_ListNonGroupMembers_InvalidID(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	_, _, err := svc.ListNonGroupMembers(context.Background(), uint64(0), 500, 0, "")

	assert.ErrorIs(t, err, domain.ErrBadParamInput)
	repo.AssertNotCalled(t, "GetByID")
	repo.AssertNotCalled(t, "ListNonGroupMembers")
}

func TestService_ListNonGroupMembers_InvalidLimitTooLow(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	_, _, err := svc.ListNonGroupMembers(context.Background(), uint64(1), 0, 0, "")

	assert.ErrorIs(t, err, domain.ErrBadParamInput)
	repo.AssertNotCalled(t, "GetByID")
}

func TestService_ListNonGroupMembers_InvalidLimitTooHigh(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	_, _, err := svc.ListNonGroupMembers(context.Background(), uint64(1), 501, 0, "")

	assert.ErrorIs(t, err, domain.ErrBadParamInput)
	repo.AssertNotCalled(t, "GetByID")
}

func TestService_ListNonGroupMembers_RepositoryError(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	groupResp := domain.Group{ID: 1, Name: "dev-team", Description: "developers", MemberCount: 1}
	repo.On("GetByID", mock.Anything, uint64(1)).Return(groupResp, nil)

	repo.On("ListNonGroupMembers", mock.Anything, uint64(1), 500, 0, "").
		Return([]domain.User(nil), 0, domain.ErrInternalServerError)

	_, _, err := svc.ListNonGroupMembers(context.Background(), uint64(1), 500, 0, "")

	assert.ErrorIs(t, err, domain.ErrInternalServerError)
	repo.AssertExpectations(t)
}

func TestService_AddGroupMembers_OK(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	groupResp := domain.Group{ID: 1, Name: "dev-team", Description: "developers", MemberCount: 1}
	repo.On("GetByID", mock.Anything, uint64(1)).Return(groupResp, nil)

	userRepo.On("CountByIDs", mock.Anything, []uint64{2, 3}).Return(2, nil)

	user2 := domain.User{ID: 2, UUID: "00000000-0000-0000-0000-000000000002", FirstName: "Hanako", LastName: "Suzuki"}
	user3 := domain.User{ID: 3, UUID: "00000000-0000-0000-0000-000000000003", FirstName: "Jiro", LastName: "Tanaka"}
	added := []domain.User{user2, user3}
	repo.On("AddGroupMembers", mock.Anything, uint64(1), []uint64{2, 3}).Return(added, nil)

	result, err := svc.AddGroupMembers(context.Background(), 1, []uint64{2, 3})

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	repo.AssertExpectations(t)
	userRepo.AssertExpectations(t)
}

func TestService_AddGroupMembers_GroupNotFound(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	repo.On("GetByID", mock.Anything, uint64(9999)).
		Return(domain.Group{}, domain.ErrNotFound)

	_, err := svc.AddGroupMembers(context.Background(), 9999, []uint64{1})

	assert.ErrorIs(t, err, domain.ErrNotFound)
	userRepo.AssertNotCalled(t, "CountByIDs")
	repo.AssertNotCalled(t, "AddGroupMembers")
}

func TestService_AddGroupMembers_UserNotFound(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	groupResp := domain.Group{ID: 1, Name: "dev-team", Description: "developers", MemberCount: 0}
	repo.On("GetByID", mock.Anything, uint64(1)).Return(groupResp, nil)

	// CountByIDs returns fewer than requested — at least one user does not exist.
	userRepo.On("CountByIDs", mock.Anything, []uint64{9999}).Return(0, nil)

	_, err := svc.AddGroupMembers(context.Background(), 1, []uint64{9999})

	assert.ErrorIs(t, err, domain.ErrNotFound)
	repo.AssertNotCalled(t, "AddGroupMembers")
	userRepo.AssertExpectations(t)
}

func TestService_AddGroupMembers_AlreadyMember(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	groupResp := domain.Group{ID: 1, Name: "dev-team", Description: "developers", MemberCount: 1}
	repo.On("GetByID", mock.Anything, uint64(1)).Return(groupResp, nil)

	userRepo.On("CountByIDs", mock.Anything, []uint64{1}).Return(1, nil)

	repo.On("AddGroupMembers", mock.Anything, uint64(1), []uint64{1}).Return([]domain.User(nil), domain.ErrConflict)

	_, err := svc.AddGroupMembers(context.Background(), 1, []uint64{1})

	assert.ErrorIs(t, err, domain.ErrConflict)
	repo.AssertExpectations(t)
	userRepo.AssertExpectations(t)
}

func TestService_AddGroupMembers_CountByIDsError(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	groupResp := domain.Group{ID: 1, Name: "dev-team", Description: "developers", MemberCount: 0}
	repo.On("GetByID", mock.Anything, uint64(1)).Return(groupResp, nil)

	userRepo.On("CountByIDs", mock.Anything, []uint64{2}).Return(0, domain.ErrInternalServerError)

	_, err := svc.AddGroupMembers(context.Background(), 1, []uint64{2})

	assert.ErrorIs(t, err, domain.ErrInternalServerError)
	repo.AssertNotCalled(t, "AddGroupMembers")
	userRepo.AssertExpectations(t)
}

func TestService_AddGroupMembers_RepositoryError(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	groupResp := domain.Group{ID: 1, Name: "dev-team", Description: "developers", MemberCount: 0}
	repo.On("GetByID", mock.Anything, uint64(1)).Return(groupResp, nil)

	userRepo.On("CountByIDs", mock.Anything, []uint64{2}).Return(1, nil)

	repo.On("AddGroupMembers", mock.Anything, uint64(1), []uint64{2}).Return([]domain.User(nil), domain.ErrInternalServerError)

	_, err := svc.AddGroupMembers(context.Background(), 1, []uint64{2})

	assert.ErrorIs(t, err, domain.ErrInternalServerError)
	repo.AssertExpectations(t)
	userRepo.AssertExpectations(t)
}

func TestService_AddGroupMembers_DuplicateUserIDs(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	groupResp := domain.Group{ID: 1, Name: "dev-team", Description: "developers", MemberCount: 0}
	repo.On("GetByID", mock.Anything, uint64(1)).Return(groupResp, nil)

	// After deduplication [2, 2] becomes [2], so CountByIDs is called with [2] and returns 1.
	userRepo.On("CountByIDs", mock.Anything, []uint64{2}).Return(1, nil)

	user2 := domain.User{ID: 2, FirstName: "Hanako", LastName: "Suzuki"}
	repo.On("AddGroupMembers", mock.Anything, uint64(1), []uint64{2}).Return([]domain.User{user2}, nil)

	result, err := svc.AddGroupMembers(context.Background(), 1, []uint64{2, 2})

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	repo.AssertExpectations(t)
	userRepo.AssertExpectations(t)
}

func TestService_RemoveGroupMembers_SingleMember(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	groupResp := domain.Group{ID: 1, Name: "dev-team", Description: "developers", MemberCount: 1}
	repo.On("GetByID", mock.Anything, uint64(1)).Return(groupResp, nil)
	repo.On("RemoveGroupMembers", mock.Anything, uint64(1), []uint64{2}).Return(nil)

	err := svc.RemoveGroupMembers(context.Background(), 1, []uint64{2})

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestService_RemoveGroupMembers_BulkDelete(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	groupResp := domain.Group{ID: 1, Name: "dev-team", Description: "developers", MemberCount: 3}
	repo.On("GetByID", mock.Anything, uint64(1)).Return(groupResp, nil)
	repo.On("RemoveGroupMembers", mock.Anything, uint64(1), []uint64{2, 3, 4}).Return(nil)

	err := svc.RemoveGroupMembers(context.Background(), 1, []uint64{2, 3, 4})

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestService_RemoveGroupMembers_GroupNotFound(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	repo.On("GetByID", mock.Anything, uint64(9999)).
		Return(domain.Group{}, domain.ErrNotFound)

	err := svc.RemoveGroupMembers(context.Background(), 9999, []uint64{1})

	assert.ErrorIs(t, err, domain.ErrNotFound)
	repo.AssertNotCalled(t, "RemoveGroupMembers")
}

func TestService_RemoveGroupMembers_NonMemberIncluded(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	groupResp := domain.Group{ID: 1, Name: "dev-team", Description: "developers", MemberCount: 1}
	repo.On("GetByID", mock.Anything, uint64(1)).Return(groupResp, nil)
	repo.On("RemoveGroupMembers", mock.Anything, uint64(1), []uint64{9999}).Return(domain.ErrNotFound)

	err := svc.RemoveGroupMembers(context.Background(), 1, []uint64{9999})

	assert.ErrorIs(t, err, domain.ErrNotFound)
	repo.AssertExpectations(t)
}

func TestService_RemoveGroupMembers_DBError(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	groupResp := domain.Group{ID: 1, Name: "dev-team", Description: "developers", MemberCount: 1}
	repo.On("GetByID", mock.Anything, uint64(1)).Return(groupResp, nil)
	repo.On("RemoveGroupMembers", mock.Anything, uint64(1), []uint64{2}).Return(domain.ErrInternalServerError)

	err := svc.RemoveGroupMembers(context.Background(), 1, []uint64{2})

	assert.ErrorIs(t, err, domain.ErrInternalServerError)
	repo.AssertExpectations(t)
}

func TestService_RemoveGroupMembers_SingleItemUserIDs(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	groupResp := domain.Group{ID: 1, Name: "dev-team", Description: "developers", MemberCount: 1}
	repo.On("GetByID", mock.Anything, uint64(1)).Return(groupResp, nil)
	repo.On("RemoveGroupMembers", mock.Anything, uint64(1), []uint64{5}).Return(nil)

	err := svc.RemoveGroupMembers(context.Background(), 1, []uint64{5})

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestService_RemoveGroupMembers_DuplicateUserIDs(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	groupResp := domain.Group{ID: 1, Name: "dev-team", Description: "developers", MemberCount: 1}
	repo.On("GetByID", mock.Anything, uint64(1)).Return(groupResp, nil)
	// After deduplication [2, 2] becomes [2], so RemoveGroupMembers is called with [2].
	repo.On("RemoveGroupMembers", mock.Anything, uint64(1), []uint64{2}).Return(nil)

	err := svc.RemoveGroupMembers(context.Background(), 1, []uint64{2, 2})

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestService_RemoveGroupMembers_IsolatedViaInterface(t *testing.T) {
	// Verify that Service is isolated from real DB via the GroupRepository interface.
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	groupResp := domain.Group{ID: 1, Name: "dev-team", Description: "developers", MemberCount: 2}
	repo.On("GetByID", mock.Anything, uint64(1)).Return(groupResp, nil)
	repo.On("RemoveGroupMembers", mock.Anything, uint64(1), []uint64{2, 3}).Return(nil)

	err := svc.RemoveGroupMembers(context.Background(), 1, []uint64{2, 3})

	assert.NoError(t, err)
	// Verify only the mock was called, not a real DB.
	repo.AssertExpectations(t)
	userRepo.AssertNotCalled(t, "CountByIDs")
}
