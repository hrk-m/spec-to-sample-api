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

func newGetByIDService(t *testing.T) (*group.Service, *mocks.MockGroupRepository, *mocks.MockGroupRelationRepository) {
	t.Helper()
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	relRepo := new(mocks.MockGroupRelationRepository)
	svc := group.NewServiceWithRelation(repo, userRepo, relRepo)
	return svc, repo, relRepo
}

func TestService_GetByID_WithSubgroups(t *testing.T) {
	svc, repo, relRepo := newGetByIDService(t)

	expected := domain.Group{ID: 1, Name: "dev-team", Description: "developers", MemberCount: 5}
	repo.On("GetByID", mock.Anything, uint64(1)).Return(expected, nil)

	children := []domain.Group{
		{ID: 2, Name: "Frontend Team", Description: "", MemberCount: 2},
		{ID: 3, Name: "Backend Team", Description: "", MemberCount: 3},
	}
	relRepo.On("ListChildren", mock.Anything, uint64(1)).Return(children, nil)

	result, subgroups, err := svc.GetByID(context.Background(), 1)

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	assert.Len(t, subgroups, 2)
	assert.Equal(t, uint64(2), subgroups[0].ID)
	repo.AssertExpectations(t)
	relRepo.AssertExpectations(t)
}

func TestService_GetByID_SubgroupsEmpty(t *testing.T) {
	svc, repo, relRepo := newGetByIDService(t)

	expected := domain.Group{ID: 1, Name: "dev-team", Description: "developers", MemberCount: 5}
	repo.On("GetByID", mock.Anything, uint64(1)).Return(expected, nil)
	relRepo.On("ListChildren", mock.Anything, uint64(1)).Return([]domain.Group{}, nil)

	result, subgroups, err := svc.GetByID(context.Background(), 1)

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	assert.NotNil(t, subgroups)
	assert.Empty(t, subgroups)
	repo.AssertExpectations(t)
	relRepo.AssertExpectations(t)
}

func TestService_GetByID_ListChildrenError(t *testing.T) {
	svc, repo, relRepo := newGetByIDService(t)

	expected := domain.Group{ID: 1, Name: "dev-team", Description: "developers", MemberCount: 5}
	repo.On("GetByID", mock.Anything, uint64(1)).Return(expected, nil)
	relRepo.On("ListChildren", mock.Anything, uint64(1)).Return([]domain.Group(nil), domain.ErrInternalServerError)

	_, _, err := svc.GetByID(context.Background(), 1)

	assert.ErrorIs(t, err, domain.ErrInternalServerError)
	repo.AssertExpectations(t)
	relRepo.AssertExpectations(t)
}

func TestService_GetByID_NotFound(t *testing.T) {
	svc, repo, _ := newGetByIDService(t)

	repo.On("GetByID", mock.Anything, uint64(9999)).
		Return(domain.Group{}, domain.ErrNotFound)

	_, _, err := svc.GetByID(context.Background(), 9999)

	assert.ErrorIs(t, err, domain.ErrNotFound)
	repo.AssertExpectations(t)
}

func TestService_GetByID_InvalidID(t *testing.T) {
	svc, repo, _ := newGetByIDService(t)

	_, _, err := svc.GetByID(context.Background(), 0)

	assert.ErrorIs(t, err, domain.ErrBadParamInput)
	repo.AssertNotCalled(t, "GetByID")
}

func TestService_GetByID_RepositoryError(t *testing.T) {
	svc, repo, _ := newGetByIDService(t)

	repo.On("GetByID", mock.Anything, uint64(1)).
		Return(domain.Group{}, domain.ErrInternalServerError)

	_, _, err := svc.GetByID(context.Background(), 1)

	assert.ErrorIs(t, err, domain.ErrInternalServerError)
	repo.AssertExpectations(t)
}

func TestService_ListGroupMembers_OK(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	groupResp := domain.Group{ID: 1, Name: "dev-team", Description: "developers", MemberCount: 2}
	repo.On("GetByID", mock.Anything, uint64(1)).Return(groupResp, nil)

	src1 := []domain.SourceGroup{{GroupID: 1, GroupName: "dev-team"}}
	members := []domain.GroupMember{
		{ID: 1, UUID: "00000000-0000-0000-0000-000000000001", FirstName: "Taro", LastName: "Yamada", SourceGroups: src1},
		{ID: 2, UUID: "00000000-0000-0000-0000-000000000002", FirstName: "Hanako", LastName: "Suzuki", SourceGroups: src1},
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

	src := []domain.SourceGroup{{GroupID: 1, GroupName: "dev-team"}}
	members := []domain.GroupMember{
		{ID: 1, UUID: "00000000-0000-0000-0000-000000000001", FirstName: "Taro", LastName: "Yamada", SourceGroups: src},
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
		Return([]domain.GroupMember(nil), 0, domain.ErrInternalServerError)

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
		Return([]domain.GroupMember(nil), 0, nil)

	result, total, err := svc.ListGroupMembers(context.Background(), 1, 500, 0, "")

	assert.NoError(t, err)
	assert.Empty(t, result)
	assert.NotNil(t, result)
	assert.Equal(t, 0, total)
	repo.AssertExpectations(t)
}

// --- ListGroupMembers PRD tests (#14-#23) ---

// #14: 正常系 — 親 + 全子孫を再帰で集約した結果を返す（3 階層）。
func TestService_ListGroupMembers_MultiLevel(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	groupResp := domain.Group{ID: 1, Name: "Engineering", Description: "", MemberCount: 3}
	repo.On("GetByID", mock.Anything, uint64(1)).Return(groupResp, nil)

	members := []domain.GroupMember{
		{ID: 1, UUID: "uuid-1", FirstName: "Taro", LastName: "Yamada", SourceGroups: []domain.SourceGroup{{GroupID: 1, GroupName: "Engineering"}}},
		{ID: 2, UUID: "uuid-2", FirstName: "Hanako", LastName: "Suzuki", SourceGroups: []domain.SourceGroup{{GroupID: 2, GroupName: "Frontend Team"}}},
		{ID: 3, UUID: "uuid-3", FirstName: "Jiro", LastName: "Tanaka", SourceGroups: []domain.SourceGroup{{GroupID: 3, GroupName: "Backend Team"}}},
	}
	repo.On("ListGroupMembers", mock.Anything, uint64(1), 500, 0, "").
		Return(members, 3, nil)

	result, total, err := svc.ListGroupMembers(context.Background(), 1, 500, 0, "")

	assert.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Equal(t, 3, total)
	assert.Equal(t, uint64(1), result[0].SourceGroups[0].GroupID)
	assert.Equal(t, uint64(2), result[1].SourceGroups[0].GroupID)
	repo.AssertExpectations(t)
}

// #15: 正常系 — 親優先の重複排除が働く。
func TestService_ListGroupMembers_DuplicateUserParentPriority(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	groupResp := domain.Group{ID: 1, Name: "Engineering", Description: "", MemberCount: 1}
	repo.On("GetByID", mock.Anything, uint64(1)).Return(groupResp, nil)

	// User 5 belongs to both parent(1) and child(2); both source groups are returned.
	dupSources := []domain.SourceGroup{{GroupID: 1, GroupName: "Engineering"}, {GroupID: 2, GroupName: "SubGroup"}}
	members := []domain.GroupMember{
		{ID: 5, UUID: "uuid-5", FirstName: "Duplicate", LastName: "User", SourceGroups: dupSources},
	}
	repo.On("ListGroupMembers", mock.Anything, uint64(1), 500, 0, "").
		Return(members, 1, nil)

	result, total, err := svc.ListGroupMembers(context.Background(), 1, 500, 0, "")

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, 1, total)
	assert.Equal(t, uint64(1), result[0].SourceGroups[0].GroupID)
	repo.AssertExpectations(t)
}

// #16: 正常系 — 子孫由来は最浅祖先の group_id を source として採用。
func TestService_ListGroupMembers_ShallowAncestorSource(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	groupResp := domain.Group{ID: 1, Name: "Root", Description: "", MemberCount: 1}
	repo.On("GetByID", mock.Anything, uint64(1)).Return(groupResp, nil)

	// User only in grandchild(3); source is child(2) which is the shallowest ancestor's root_child_id.
	members := []domain.GroupMember{
		{ID: 7, UUID: "uuid-7", FirstName: "Deep", LastName: "Member", SourceGroups: []domain.SourceGroup{{GroupID: 2, GroupName: "Child Group"}}},
	}
	repo.On("ListGroupMembers", mock.Anything, uint64(1), 500, 0, "").
		Return(members, 1, nil)

	result, total, err := svc.ListGroupMembers(context.Background(), 1, 500, 0, "")

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, 1, total)
	assert.Equal(t, uint64(2), result[0].SourceGroups[0].GroupID)
	assert.Equal(t, "Child Group", result[0].SourceGroups[0].GroupName)
	repo.AssertExpectations(t)
}

// #17: 正常系 — q フィルター適用。
func TestService_ListGroupMembers_QFilter(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	groupResp := domain.Group{ID: 1, Name: "Engineering", Description: "", MemberCount: 5}
	repo.On("GetByID", mock.Anything, uint64(1)).Return(groupResp, nil)

	members := []domain.GroupMember{
		{ID: 2, UUID: "uuid-2", FirstName: "Hanako", LastName: "Sato", SourceGroups: []domain.SourceGroup{{GroupID: 1, GroupName: "Engineering"}}},
	}
	repo.On("ListGroupMembers", mock.Anything, uint64(1), 500, 0, "Sato").
		Return(members, 1, nil)

	result, total, err := svc.ListGroupMembers(context.Background(), 1, 500, 0, "Sato")

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, 1, total)
	assert.Equal(t, "Sato", result[0].LastName)
	repo.AssertExpectations(t)
}

// #18: 正常系 — サブグループが空でも親直属メンバーが返る。
func TestService_ListGroupMembers_NoDescendants(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	groupResp := domain.Group{ID: 1, Name: "Solo", Description: "", MemberCount: 2}
	repo.On("GetByID", mock.Anything, uint64(1)).Return(groupResp, nil)

	members := []domain.GroupMember{
		{ID: 1, UUID: "uuid-1", FirstName: "A", LastName: "User", SourceGroups: []domain.SourceGroup{{GroupID: 1, GroupName: "Solo"}}},
		{ID: 2, UUID: "uuid-2", FirstName: "B", LastName: "User", SourceGroups: []domain.SourceGroup{{GroupID: 1, GroupName: "Solo"}}},
	}
	repo.On("ListGroupMembers", mock.Anything, uint64(1), 500, 0, "").
		Return(members, 2, nil)

	result, total, err := svc.ListGroupMembers(context.Background(), 1, 500, 0, "")

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, 2, total)
	for _, m := range result {
		assert.Equal(t, uint64(1), m.SourceGroups[0].GroupID)
	}
	repo.AssertExpectations(t)
}

// #19: 正常系 — メンバーが 0 人のグループ。
func TestService_ListGroupMembers_ZeroMembers(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	groupResp := domain.Group{ID: 1, Name: "Empty", Description: "", MemberCount: 0}
	repo.On("GetByID", mock.Anything, uint64(1)).Return(groupResp, nil)
	repo.On("ListGroupMembers", mock.Anything, uint64(1), 500, 0, "").
		Return([]domain.GroupMember(nil), 0, nil)

	result, total, err := svc.ListGroupMembers(context.Background(), 1, 500, 0, "")

	assert.NoError(t, err)
	assert.Empty(t, result)
	assert.NotNil(t, result)
	assert.Equal(t, 0, total)
	repo.AssertExpectations(t)
}

// #21: 境界値 — id=0（最小境界外）。
func TestService_ListGroupMembers_IDZero(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	_, _, err := svc.ListGroupMembers(context.Background(), 0, 500, 0, "")

	assert.ErrorIs(t, err, domain.ErrBadParamInput)
	repo.AssertNotCalled(t, "GetByID")
	repo.AssertNotCalled(t, "ListGroupMembers")
}

// #22: 境界値 — limit=0 / limit=501。
func TestService_ListGroupMembers_InvalidLimitBoundaries(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	_, _, errLow := svc.ListGroupMembers(context.Background(), 1, 0, 0, "")
	assert.ErrorIs(t, errLow, domain.ErrBadParamInput)

	_, _, errHigh := svc.ListGroupMembers(context.Background(), 1, 501, 0, "")
	assert.ErrorIs(t, errHigh, domain.ErrBadParamInput)

	repo.AssertNotCalled(t, "GetByID")
}

// #23: 例外処理 — repository が DB エラーを返す。
func TestService_ListGroupMembers_DBError(t *testing.T) {
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	svc := group.NewService(repo, userRepo)

	groupResp := domain.Group{ID: 1, Name: "dev-team", Description: "", MemberCount: 0}
	repo.On("GetByID", mock.Anything, uint64(1)).Return(groupResp, nil)
	repo.On("ListGroupMembers", mock.Anything, uint64(1), 500, 0, "").
		Return([]domain.GroupMember(nil), 0, domain.ErrInternalServerError)

	_, _, err := svc.ListGroupMembers(context.Background(), 1, 500, 0, "")

	assert.ErrorIs(t, err, domain.ErrInternalServerError)
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

// --- CreateSubGroup tests (#10-#21) ---

func newSubGroupService(t *testing.T) (*group.Service, *mocks.MockGroupRepository, *mocks.MockGroupRelationRepository) {
	t.Helper()
	repo := new(mocks.MockGroupRepository)
	userRepo := new(mocks.MockUserRepository)
	relRepo := new(mocks.MockGroupRelationRepository)
	svc := group.NewServiceWithRelation(repo, userRepo, relRepo)
	return svc, repo, relRepo
}

// #10: 正常系 — 有効な parentGroupID + childGroupID で登録成功。
func TestService_CreateSubGroup_OK(t *testing.T) {
	svc, repo, relRepo := newSubGroupService(t)

	parent := domain.Group{ID: 1, Name: "parent", Description: "", MemberCount: 0}
	child := domain.Group{ID: 2, Name: "child", Description: "", MemberCount: 0}
	repo.On("GetByID", mock.Anything, uint64(1)).Return(parent, nil)
	repo.On("GetByID", mock.Anything, uint64(2)).Return(child, nil)
	relRepo.On("GetAncestorIDs", mock.Anything, uint64(1)).Return([]uint64{}, nil)
	relRepo.On("GetDescendantIDs", mock.Anything, uint64(2)).Return([]uint64{}, nil)
	relRepo.On("CountComponentGroups", mock.Anything, uint64(1)).Return(2, nil)
	relRepo.On("MaxDepthInComponent", mock.Anything, uint64(1), uint64(2)).Return(2, nil)
	expected := domain.GroupRelation{ParentGroupID: 1, ChildGroupID: 2}
	relRepo.On("CreateRelation", mock.Anything, uint64(1), uint64(2)).Return(expected, nil)

	result, err := svc.CreateSubGroup(context.Background(), 1, 2)

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	repo.AssertExpectations(t)
	relRepo.AssertExpectations(t)
}

// #11: 異常系 — child_group_id が 0。
func TestService_CreateSubGroup_ChildIDZero(t *testing.T) {
	svc, _, _ := newSubGroupService(t)

	_, err := svc.CreateSubGroup(context.Background(), 1, 0)

	assert.ErrorIs(t, err, domain.ErrBadParamInput)
}

// #12: 異常系 — 自己ループ（parent == child）。
func TestService_CreateSubGroup_SelfLoop(t *testing.T) {
	svc, _, _ := newSubGroupService(t)

	_, err := svc.CreateSubGroup(context.Background(), 1, 1)

	assert.ErrorIs(t, err, domain.ErrBadParamInput)
}

// #13: 異常系 — parent_group_id が DB に存在しない。
func TestService_CreateSubGroup_ParentNotFound(t *testing.T) {
	svc, repo, _ := newSubGroupService(t)

	repo.On("GetByID", mock.Anything, uint64(9999)).Return(domain.Group{}, domain.ErrNotFound)

	_, err := svc.CreateSubGroup(context.Background(), 9999, 2)

	assert.ErrorIs(t, err, domain.ErrNotFound)
	repo.AssertExpectations(t)
}

// #14: 異常系 — child_group_id が DB に存在しない。
func TestService_CreateSubGroup_ChildNotFound(t *testing.T) {
	svc, repo, _ := newSubGroupService(t)

	parent := domain.Group{ID: 1, Name: "parent", Description: "", MemberCount: 0}
	repo.On("GetByID", mock.Anything, uint64(1)).Return(parent, nil)
	repo.On("GetByID", mock.Anything, uint64(9999)).Return(domain.Group{}, domain.ErrNotFound)

	_, err := svc.CreateSubGroup(context.Background(), 1, 9999)

	assert.ErrorIs(t, err, domain.ErrNotFound)
	repo.AssertExpectations(t)
}

// #15: 分岐条件 — 循環参照が検出される（child が parent の祖先）。
func TestService_CreateSubGroup_CycleDetected(t *testing.T) {
	svc, repo, relRepo := newSubGroupService(t)

	// parent=2, child=1, 既存: 1→2 なので child=1 の子孫に parent=2 が含まれる
	parent := domain.Group{ID: 2, Name: "B", Description: "", MemberCount: 0}
	child := domain.Group{ID: 1, Name: "A", Description: "", MemberCount: 0}
	repo.On("GetByID", mock.Anything, uint64(2)).Return(parent, nil)
	repo.On("GetByID", mock.Anything, uint64(1)).Return(child, nil)
	// parent(2) の祖先に child(1) が含まれる
	relRepo.On("GetAncestorIDs", mock.Anything, uint64(2)).Return([]uint64{1}, nil)
	relRepo.On("GetDescendantIDs", mock.Anything, uint64(1)).Return([]uint64{2}, nil)

	_, err := svc.CreateSubGroup(context.Background(), 2, 1)

	assert.ErrorIs(t, err, domain.ErrBadParamInput)
	repo.AssertExpectations(t)
	relRepo.AssertExpectations(t)
}

// #16: 境界値 — ツリーグループ数が 9（追加後 10）→ 成功。
func TestService_CreateSubGroup_ComponentSizeBoundary_OK(t *testing.T) {
	svc, repo, relRepo := newSubGroupService(t)

	parent := domain.Group{ID: 1, Name: "parent", Description: "", MemberCount: 0}
	child := domain.Group{ID: 10, Name: "child", Description: "", MemberCount: 0}
	repo.On("GetByID", mock.Anything, uint64(1)).Return(parent, nil)
	repo.On("GetByID", mock.Anything, uint64(10)).Return(child, nil)
	relRepo.On("GetAncestorIDs", mock.Anything, uint64(1)).Return([]uint64{}, nil)
	relRepo.On("GetDescendantIDs", mock.Anything, uint64(10)).Return([]uint64{}, nil)
	// 現在 9 グループ → 追加後 10 で OK
	relRepo.On("CountComponentGroups", mock.Anything, uint64(1)).Return(9, nil)
	relRepo.On("MaxDepthInComponent", mock.Anything, uint64(1), uint64(10)).Return(2, nil)
	expected := domain.GroupRelation{ParentGroupID: 1, ChildGroupID: 10}
	relRepo.On("CreateRelation", mock.Anything, uint64(1), uint64(10)).Return(expected, nil)

	result, err := svc.CreateSubGroup(context.Background(), 1, 10)

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}

// #17: 境界値 — ツリーグループ数が 10（追加後 11）→ ErrBadParamInput。
func TestService_CreateSubGroup_ComponentSizeExceeded(t *testing.T) {
	svc, repo, relRepo := newSubGroupService(t)

	parent := domain.Group{ID: 1, Name: "parent", Description: "", MemberCount: 0}
	child := domain.Group{ID: 11, Name: "child", Description: "", MemberCount: 0}
	repo.On("GetByID", mock.Anything, uint64(1)).Return(parent, nil)
	repo.On("GetByID", mock.Anything, uint64(11)).Return(child, nil)
	relRepo.On("GetAncestorIDs", mock.Anything, uint64(1)).Return([]uint64{}, nil)
	relRepo.On("GetDescendantIDs", mock.Anything, uint64(11)).Return([]uint64{}, nil)
	// 現在 10 グループ → 追加後 11 で NG
	relRepo.On("CountComponentGroups", mock.Anything, uint64(1)).Return(10, nil)

	_, err := svc.CreateSubGroup(context.Background(), 1, 11)

	assert.ErrorIs(t, err, domain.ErrBadParamInput)
}

// #18: 境界値 — 階層深度が 4 ノード（追加後 5 ノード）→ 成功。
func TestService_CreateSubGroup_DepthBoundary_OK(t *testing.T) {
	svc, repo, relRepo := newSubGroupService(t)

	parent := domain.Group{ID: 1, Name: "parent", Description: "", MemberCount: 0}
	child := domain.Group{ID: 2, Name: "child", Description: "", MemberCount: 0}
	repo.On("GetByID", mock.Anything, uint64(1)).Return(parent, nil)
	repo.On("GetByID", mock.Anything, uint64(2)).Return(child, nil)
	relRepo.On("GetAncestorIDs", mock.Anything, uint64(1)).Return([]uint64{}, nil)
	relRepo.On("GetDescendantIDs", mock.Anything, uint64(2)).Return([]uint64{}, nil)
	relRepo.On("CountComponentGroups", mock.Anything, uint64(1)).Return(3, nil)
	// 追加後の最大深度 5 ノード → OK
	relRepo.On("MaxDepthInComponent", mock.Anything, uint64(1), uint64(2)).Return(5, nil)
	expected := domain.GroupRelation{ParentGroupID: 1, ChildGroupID: 2}
	relRepo.On("CreateRelation", mock.Anything, uint64(1), uint64(2)).Return(expected, nil)

	result, err := svc.CreateSubGroup(context.Background(), 1, 2)

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}

// #19: 境界値 — 階層深度が 5 ノード（追加後 6 ノード）→ ErrBadParamInput。
func TestService_CreateSubGroup_DepthExceeded(t *testing.T) {
	svc, repo, relRepo := newSubGroupService(t)

	parent := domain.Group{ID: 1, Name: "parent", Description: "", MemberCount: 0}
	child := domain.Group{ID: 2, Name: "child", Description: "", MemberCount: 0}
	repo.On("GetByID", mock.Anything, uint64(1)).Return(parent, nil)
	repo.On("GetByID", mock.Anything, uint64(2)).Return(child, nil)
	relRepo.On("GetAncestorIDs", mock.Anything, uint64(1)).Return([]uint64{}, nil)
	relRepo.On("GetDescendantIDs", mock.Anything, uint64(2)).Return([]uint64{}, nil)
	relRepo.On("CountComponentGroups", mock.Anything, uint64(1)).Return(3, nil)
	// 追加後の最大深度 6 ノード → NG
	relRepo.On("MaxDepthInComponent", mock.Anything, uint64(1), uint64(2)).Return(6, nil)

	_, err := svc.CreateSubGroup(context.Background(), 1, 2)

	assert.ErrorIs(t, err, domain.ErrBadParamInput)
}

// #20: 異常系 — repository が ErrConflict を返す（重複登録）。
func TestService_CreateSubGroup_Conflict(t *testing.T) {
	svc, repo, relRepo := newSubGroupService(t)

	parent := domain.Group{ID: 1, Name: "parent", Description: "", MemberCount: 0}
	child := domain.Group{ID: 2, Name: "child", Description: "", MemberCount: 0}
	repo.On("GetByID", mock.Anything, uint64(1)).Return(parent, nil)
	repo.On("GetByID", mock.Anything, uint64(2)).Return(child, nil)
	relRepo.On("GetAncestorIDs", mock.Anything, uint64(1)).Return([]uint64{}, nil)
	relRepo.On("GetDescendantIDs", mock.Anything, uint64(2)).Return([]uint64{}, nil)
	relRepo.On("CountComponentGroups", mock.Anything, uint64(1)).Return(2, nil)
	relRepo.On("MaxDepthInComponent", mock.Anything, uint64(1), uint64(2)).Return(2, nil)
	relRepo.On("CreateRelation", mock.Anything, uint64(1), uint64(2)).Return(domain.GroupRelation{}, domain.ErrConflict)

	_, err := svc.CreateSubGroup(context.Background(), 1, 2)

	assert.ErrorIs(t, err, domain.ErrConflict)
}

// #21: 例外処理 — repository が DB エラーを返す。
func TestService_CreateSubGroup_DBError(t *testing.T) {
	svc, repo, relRepo := newSubGroupService(t)

	parent := domain.Group{ID: 1, Name: "parent", Description: "", MemberCount: 0}
	child := domain.Group{ID: 2, Name: "child", Description: "", MemberCount: 0}
	repo.On("GetByID", mock.Anything, uint64(1)).Return(parent, nil)
	repo.On("GetByID", mock.Anything, uint64(2)).Return(child, nil)
	relRepo.On("GetAncestorIDs", mock.Anything, uint64(1)).Return([]uint64{}, nil)
	relRepo.On("GetDescendantIDs", mock.Anything, uint64(2)).Return([]uint64{}, nil)
	relRepo.On("CountComponentGroups", mock.Anything, uint64(1)).Return(2, nil)
	relRepo.On("MaxDepthInComponent", mock.Anything, uint64(1), uint64(2)).Return(2, nil)
	relRepo.On("CreateRelation", mock.Anything, uint64(1), uint64(2)).Return(domain.GroupRelation{}, domain.ErrInternalServerError)

	_, err := svc.CreateSubGroup(context.Background(), 1, 2)

	assert.ErrorIs(t, err, domain.ErrInternalServerError)
}

// --- DeleteSubGroup tests (#9-#11) ---

// #9: 正常系 — 存在する親子関係を削除する。
func TestService_DeleteSubGroup_OK(t *testing.T) {
	svc, _, relRepo := newSubGroupService(t)

	relRepo.On("DeleteRelation", mock.Anything, uint64(1), uint64(2)).Return(nil)

	err := svc.DeleteSubGroup(context.Background(), 1, 2)

	assert.NoError(t, err)
	relRepo.AssertExpectations(t)
}

// #10: 異常系 — 対象の親子関係が存在しない（RowsAffected=0）。
func TestService_DeleteSubGroup_NotFound(t *testing.T) {
	svc, _, relRepo := newSubGroupService(t)

	relRepo.On("DeleteRelation", mock.Anything, uint64(1), uint64(2)).Return(domain.ErrNotFound)

	err := svc.DeleteSubGroup(context.Background(), 1, 2)

	assert.ErrorIs(t, err, domain.ErrNotFound)
	relRepo.AssertExpectations(t)
}

// #11: 例外処理 — repository が DB エラーを返す。
func TestService_DeleteSubGroup_DBError(t *testing.T) {
	svc, _, relRepo := newSubGroupService(t)

	relRepo.On("DeleteRelation", mock.Anything, uint64(1), uint64(2)).Return(domain.ErrInternalServerError)

	err := svc.DeleteSubGroup(context.Background(), 1, 2)

	assert.ErrorIs(t, err, domain.ErrInternalServerError)
	relRepo.AssertExpectations(t)
}

