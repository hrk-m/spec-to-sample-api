package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/hrk-m/spec-to-dev-workflow/sample-api/domain"
	"github.com/hrk-m/spec-to-dev-workflow/sample-api/internal/rest"
	"github.com/hrk-m/spec-to-dev-workflow/sample-api/internal/rest/mocks"
)

func TestGroupHandler_GetByID_OK(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id")
	c.SetParamNames("id")
	c.SetParamValues("1")

	svc := new(mocks.MockGroupService)
	resp := domain.Group{ID: 1, Name: "dev-team", Description: "developers", MemberCount: 5}
	svc.On("GetByID", mock.Anything, uint64(1)).Return(resp, nil)

	h := &rest.GroupHandler{Service: svc}
	err := h.GetByID(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var result domain.Group
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, uint64(1), result.ID)
	assert.Equal(t, "dev-team", result.Name)
	assert.Equal(t, 5, result.MemberCount)
	svc.AssertExpectations(t)
}

func TestGroupHandler_GetByID_InvalidID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups/abc", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id")
	c.SetParamNames("id")
	c.SetParamValues("abc")

	h := &rest.GroupHandler{Service: new(mocks.MockGroupService)}
	err := h.GetByID(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "given param is not valid", result["message"])
}

func TestGroupHandler_GetByID_ZeroID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups/0", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id")
	c.SetParamNames("id")
	c.SetParamValues("0")

	h := &rest.GroupHandler{Service: new(mocks.MockGroupService)}
	err := h.GetByID(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "given param is not valid", result["message"])
}

func TestGroupHandler_GetByID_NegativeID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups/-1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id")
	c.SetParamNames("id")
	c.SetParamValues("-1")

	h := &rest.GroupHandler{Service: new(mocks.MockGroupService)}
	err := h.GetByID(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "given param is not valid", result["message"])
}

func TestGroupHandler_GetByID_NotFound(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups/9999", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id")
	c.SetParamNames("id")
	c.SetParamValues("9999")

	svc := new(mocks.MockGroupService)
	svc.On("GetByID", mock.Anything, uint64(9999)).
		Return(domain.Group{}, domain.ErrNotFound)

	h := &rest.GroupHandler{Service: svc}
	err := h.GetByID(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "your requested item is not found", result["message"])
	svc.AssertExpectations(t)
}

func TestGroupHandler_GetByID_InternalError(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id")
	c.SetParamNames("id")
	c.SetParamValues("1")

	svc := new(mocks.MockGroupService)
	svc.On("GetByID", mock.Anything, uint64(1)).
		Return(domain.Group{}, domain.ErrInternalServerError)

	h := &rest.GroupHandler{Service: svc}
	err := h.GetByID(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "internal server error", result["message"])
	svc.AssertExpectations(t)
}

func TestGroupHandler_ListGroupMembers_OK(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups/1/members?limit=500&offset=0", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id/members")
	c.SetParamNames("id")
	c.SetParamValues("1")

	svc := new(mocks.MockGroupService)
	members := []domain.User{
		{ID: 1, UUID: "00000000-0000-0000-0000-000000000001", FirstName: "Taro", LastName: "Yamada"},
	}
	svc.On("ListGroupMembers", mock.Anything, uint64(1), 500, 0, "").
		Return(members, 1, nil)

	h := &rest.GroupHandler{Service: svc}
	err := h.ListGroupMembers(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var result struct {
		Members []domain.User `json:"members"`
		Total   int           `json:"total"`
	}
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Len(t, result.Members, 1)
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", result.Members[0].UUID)
	assert.Equal(t, 1, result.Total)
	svc.AssertExpectations(t)
}

func TestGroupHandler_ListGroupMembers_DefaultParams(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups/1/members", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id/members")
	c.SetParamNames("id")
	c.SetParamValues("1")

	svc := new(mocks.MockGroupService)
	svc.On("ListGroupMembers", mock.Anything, uint64(1), 500, 0, "").
		Return([]domain.User{}, 0, nil)

	h := &rest.GroupHandler{Service: svc}
	err := h.ListGroupMembers(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	svc.AssertExpectations(t)
}

func TestGroupHandler_ListGroupMembers_WithSearch(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups/1/members?q=Yamada", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id/members")
	c.SetParamNames("id")
	c.SetParamValues("1")

	svc := new(mocks.MockGroupService)
	members := []domain.User{
		{ID: 1, UUID: "00000000-0000-0000-0000-000000000001", FirstName: "Taro", LastName: "Yamada"},
	}
	svc.On("ListGroupMembers", mock.Anything, uint64(1), 500, 0, "Yamada").
		Return(members, 2, nil)

	h := &rest.GroupHandler{Service: svc}
	err := h.ListGroupMembers(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	svc.AssertExpectations(t)
}

func TestGroupHandler_ListGroupMembers_InvalidID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups/abc/members", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id/members")
	c.SetParamNames("id")
	c.SetParamValues("abc")

	h := &rest.GroupHandler{Service: new(mocks.MockGroupService)}
	err := h.ListGroupMembers(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "given param is not valid", result["message"])
}

func TestGroupHandler_ListGroupMembers_InvalidLimit(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups/1/members?limit=501", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id/members")
	c.SetParamNames("id")
	c.SetParamValues("1")

	h := &rest.GroupHandler{Service: new(mocks.MockGroupService)}
	err := h.ListGroupMembers(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "given param is not valid", result["message"])
}

func TestGroupHandler_ListGroupMembers_InvalidLimitZero(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups/1/members?limit=0", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id/members")
	c.SetParamNames("id")
	c.SetParamValues("1")

	h := &rest.GroupHandler{Service: new(mocks.MockGroupService)}
	err := h.ListGroupMembers(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "given param is not valid", result["message"])
}

func TestGroupHandler_ListGroupMembers_InvalidOffset(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups/1/members?offset=-1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id/members")
	c.SetParamNames("id")
	c.SetParamValues("1")

	h := &rest.GroupHandler{Service: new(mocks.MockGroupService)}
	err := h.ListGroupMembers(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "given param is not valid", result["message"])
}

func TestGroupHandler_ListGroupMembers_GroupNotFound(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups/9999/members", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id/members")
	c.SetParamNames("id")
	c.SetParamValues("9999")

	svc := new(mocks.MockGroupService)
	svc.On("ListGroupMembers", mock.Anything, uint64(9999), 500, 0, "").
		Return([]domain.User(nil), 0, domain.ErrNotFound)

	h := &rest.GroupHandler{Service: svc}
	err := h.ListGroupMembers(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "your requested item is not found", result["message"])
	svc.AssertExpectations(t)
}

func TestGroupHandler_ListGroupMembers_InternalError(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups/1/members", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id/members")
	c.SetParamNames("id")
	c.SetParamValues("1")

	svc := new(mocks.MockGroupService)
	svc.On("ListGroupMembers", mock.Anything, uint64(1), 500, 0, "").
		Return([]domain.User(nil), 0, domain.ErrInternalServerError)

	h := &rest.GroupHandler{Service: svc}
	err := h.ListGroupMembers(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "internal server error", result["message"])
	svc.AssertExpectations(t)
}

func TestGroupHandler_ListGroupMembers_LimitUpperBound(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups/1/members?limit=500", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id/members")
	c.SetParamNames("id")
	c.SetParamValues("1")

	svc := new(mocks.MockGroupService)
	svc.On("ListGroupMembers", mock.Anything, uint64(1), 500, 0, "").
		Return([]domain.User{}, 0, nil)

	h := &rest.GroupHandler{Service: svc}
	err := h.ListGroupMembers(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	svc.AssertExpectations(t)
}

func TestGroupHandler_ListGroupMembers_OffsetZero(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups/1/members?offset=0", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id/members")
	c.SetParamNames("id")
	c.SetParamValues("1")

	svc := new(mocks.MockGroupService)
	svc.On("ListGroupMembers", mock.Anything, uint64(1), 500, 0, "").
		Return([]domain.User{}, 0, nil)

	h := &rest.GroupHandler{Service: svc}
	err := h.ListGroupMembers(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	svc.AssertExpectations(t)
}

func TestGroupHandler_ListGroups_OK(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups?q=dev&limit=20&offset=0", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	svc := new(mocks.MockGroupService)
	groups := []domain.Group{
		{ID: 1, Name: "dev-team", Description: "developers", MemberCount: 1},
	}
	svc.On("ListGroups", mock.Anything, "dev", 20, 0).Return(groups, 42, nil)

	h := &rest.GroupHandler{Service: svc}
	err := h.ListGroups(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var result struct {
		Groups []domain.Group `json:"groups"`
		Total  int            `json:"total"`
	}
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Len(t, result.Groups, 1)
	assert.Equal(t, "dev-team", result.Groups[0].Name)
	assert.Equal(t, 42, result.Total)
	svc.AssertExpectations(t)
}

func TestGroupHandler_ListGroups_DefaultParams(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	svc := new(mocks.MockGroupService)
	svc.On("ListGroups", mock.Anything, "", 500, 0).Return([]domain.Group{}, 0, nil)

	h := &rest.GroupHandler{Service: svc}
	err := h.ListGroups(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	svc.AssertExpectations(t)
}

func TestGroupHandler_ListGroups_WithOffset(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups?limit=500&offset=500", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	svc := new(mocks.MockGroupService)
	svc.On("ListGroups", mock.Anything, "", 500, 500).Return([]domain.Group{}, 42, nil)

	h := &rest.GroupHandler{Service: svc}
	err := h.ListGroups(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var result struct {
		Groups []domain.Group `json:"groups"`
		Total  int            `json:"total"`
	}
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, 42, result.Total)
	svc.AssertExpectations(t)
}

func TestGroupHandler_ListGroups_InvalidLimit(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups?limit=abc", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := &rest.GroupHandler{Service: new(mocks.MockGroupService)}
	err := h.ListGroups(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "given param is not valid", result["message"])
}

func TestGroupHandler_ListGroups_LimitTooHigh(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups?limit=501", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := &rest.GroupHandler{Service: new(mocks.MockGroupService)}
	err := h.ListGroups(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "given param is not valid", result["message"])
}

func TestGroupHandler_ListGroups_LimitZero(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups?limit=0", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := &rest.GroupHandler{Service: new(mocks.MockGroupService)}
	err := h.ListGroups(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "given param is not valid", result["message"])
}

func TestGroupHandler_ListGroups_LimitMax(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups?limit=500", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	svc := new(mocks.MockGroupService)
	svc.On("ListGroups", mock.Anything, "", 500, 0).Return([]domain.Group{}, 0, nil)

	h := &rest.GroupHandler{Service: svc}
	err := h.ListGroups(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	svc.AssertExpectations(t)
}

func TestGroupHandler_ListGroups_InvalidOffset(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups?offset=abc", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := &rest.GroupHandler{Service: new(mocks.MockGroupService)}
	err := h.ListGroups(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "given param is not valid", result["message"])
}

func TestGroupHandler_ListGroups_NegativeOffset(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups?offset=-1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := &rest.GroupHandler{Service: new(mocks.MockGroupService)}
	err := h.ListGroups(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "given param is not valid", result["message"])
}

func TestGroupHandler_ListGroups_OffsetZero(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups?offset=0", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	svc := new(mocks.MockGroupService)
	svc.On("ListGroups", mock.Anything, "", 500, 0).Return([]domain.Group{}, 0, nil)

	h := &rest.GroupHandler{Service: svc}
	err := h.ListGroups(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	svc.AssertExpectations(t)
}

func TestGroupHandler_ListGroups_EmptyResult(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	svc := new(mocks.MockGroupService)
	svc.On("ListGroups", mock.Anything, "", 500, 0).Return([]domain.Group{}, 0, nil)

	h := &rest.GroupHandler{Service: svc}
	err := h.ListGroups(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var result struct {
		Groups []domain.Group `json:"groups"`
		Total  int            `json:"total"`
	}
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Empty(t, result.Groups)
	assert.Equal(t, 0, result.Total)
	svc.AssertExpectations(t)
}

func TestGroupHandler_ListGroups_ServiceInternalError(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	svc := new(mocks.MockGroupService)
	svc.On("ListGroups", mock.Anything, "", 500, 0).
		Return([]domain.Group(nil), 0, domain.ErrInternalServerError)

	h := &rest.GroupHandler{Service: svc}
	err := h.ListGroups(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "internal server error", result["message"])
	svc.AssertExpectations(t)
}

func TestGroupHandler_Store_OK(t *testing.T) {
	e := echo.New()
	body := strings.NewReader(`{"name":"Test","description":"Desc"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/groups", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("authUser", domain.User{ID: 1, FirstName: "Taro", LastName: "Yamada"})

	svc := new(mocks.MockGroupService)
	resp := domain.Group{ID: 1, Name: "Test", Description: "Desc", MemberCount: 1}
	svc.On("Store", mock.Anything, "Test", "Desc", uint64(1)).Return(resp, nil)

	h := &rest.GroupHandler{Service: svc}
	err := h.Store(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var result domain.Group
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, uint64(1), result.ID)
	assert.Equal(t, "Test", result.Name)
	assert.Equal(t, "Desc", result.Description)
	assert.Equal(t, 1, result.MemberCount)
	svc.AssertExpectations(t)
}

func TestGroupHandler_Store_BindError(t *testing.T) {
	e := echo.New()
	body := strings.NewReader("invalid json")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/groups", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	svc := new(mocks.MockGroupService)
	h := &rest.GroupHandler{Service: svc}
	err := h.Store(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.NotEmpty(t, result["message"])
	svc.AssertNotCalled(t, "Store")
}

func TestGroupHandler_Store_Unauthorized(t *testing.T) {
	e := echo.New()
	body := strings.NewReader(`{"name":"Test","description":"Desc"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/groups", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// authUser not set in context

	h := &rest.GroupHandler{Service: new(mocks.MockGroupService)}
	err := h.Store(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "Unauthorized", result["message"])
}

func TestGroupHandler_Store_BadParam(t *testing.T) {
	e := echo.New()
	body := strings.NewReader(`{"name":"","description":"Desc"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/groups", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("authUser", domain.User{ID: 1, FirstName: "Taro", LastName: "Yamada"})

	svc := new(mocks.MockGroupService)
	svc.On("Store", mock.Anything, "", "Desc", uint64(1)).
		Return(domain.Group{}, domain.ErrBadParamInput)

	h := &rest.GroupHandler{Service: svc}
	err := h.Store(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "given param is not valid", result["message"])
	svc.AssertExpectations(t)
}

func TestGroupHandler_Store_InternalError(t *testing.T) {
	e := echo.New()
	body := strings.NewReader(`{"name":"Test","description":"Desc"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/groups", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("authUser", domain.User{ID: 1, FirstName: "Taro", LastName: "Yamada"})

	svc := new(mocks.MockGroupService)
	svc.On("Store", mock.Anything, "Test", "Desc", uint64(1)).
		Return(domain.Group{}, domain.ErrInternalServerError)

	h := &rest.GroupHandler{Service: svc}
	err := h.Store(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "internal server error", result["message"])
	svc.AssertExpectations(t)
}

func TestGroupHandler_Update_OK(t *testing.T) {
	e := echo.New()
	body := strings.NewReader(`{"name":"Updated","description":"New Desc"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/groups/1", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id")
	c.SetParamNames("id")
	c.SetParamValues("1")
	c.Set("authUser", domain.User{ID: 1, FirstName: "Taro", LastName: "Yamada"})

	svc := new(mocks.MockGroupService)
	resp := &domain.Group{ID: 1, Name: "Updated", Description: "New Desc", MemberCount: 5}
	svc.On("Update", mock.Anything, uint64(1), "Updated", "New Desc", uint64(1)).Return(resp, nil)

	h := &rest.GroupHandler{Service: svc}
	err := h.Update(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var result domain.Group
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, uint64(1), result.ID)
	assert.Equal(t, "Updated", result.Name)
	assert.Equal(t, "New Desc", result.Description)
	assert.Equal(t, 5, result.MemberCount)
	svc.AssertExpectations(t)
}

func TestGroupHandler_Update_Unauthorized(t *testing.T) {
	e := echo.New()
	body := strings.NewReader(`{"name":"Updated","description":"New Desc"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/groups/1", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id")
	c.SetParamNames("id")
	c.SetParamValues("1")
	// authUser not set in context

	h := &rest.GroupHandler{Service: new(mocks.MockGroupService)}
	err := h.Update(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "Unauthorized", result["message"])
}

func TestGroupHandler_Update_InvalidIDString(t *testing.T) {
	e := echo.New()
	body := strings.NewReader(`{"name":"Updated","description":"Desc"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/groups/abc", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id")
	c.SetParamNames("id")
	c.SetParamValues("abc")

	h := &rest.GroupHandler{Service: new(mocks.MockGroupService)}
	err := h.Update(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "given param is not valid", result["message"])
}

func TestGroupHandler_Update_ZeroID(t *testing.T) {
	e := echo.New()
	body := strings.NewReader(`{"name":"Updated","description":"Desc"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/groups/0", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id")
	c.SetParamNames("id")
	c.SetParamValues("0")

	h := &rest.GroupHandler{Service: new(mocks.MockGroupService)}
	err := h.Update(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "given param is not valid", result["message"])
}

func TestGroupHandler_Update_ServiceBadParam(t *testing.T) {
	e := echo.New()
	body := strings.NewReader(`{"name":"","description":"Desc"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/groups/1", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id")
	c.SetParamNames("id")
	c.SetParamValues("1")
	c.Set("authUser", domain.User{ID: 1, FirstName: "Taro", LastName: "Yamada"})

	svc := new(mocks.MockGroupService)
	svc.On("Update", mock.Anything, uint64(1), "", "Desc", uint64(1)).
		Return((*domain.Group)(nil), domain.ErrBadParamInput)

	h := &rest.GroupHandler{Service: svc}
	err := h.Update(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "given param is not valid", result["message"])
	svc.AssertExpectations(t)
}

func TestGroupHandler_Update_ServiceNotFound(t *testing.T) {
	e := echo.New()
	body := strings.NewReader(`{"name":"Updated","description":"Desc"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/groups/9999", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id")
	c.SetParamNames("id")
	c.SetParamValues("9999")
	c.Set("authUser", domain.User{ID: 1, FirstName: "Taro", LastName: "Yamada"})

	svc := new(mocks.MockGroupService)
	svc.On("Update", mock.Anything, uint64(9999), "Updated", "Desc", uint64(1)).
		Return((*domain.Group)(nil), domain.ErrNotFound)

	h := &rest.GroupHandler{Service: svc}
	err := h.Update(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "your requested item is not found", result["message"])
	svc.AssertExpectations(t)
}

func TestGroupHandler_Update_ServiceInternalError(t *testing.T) {
	e := echo.New()
	body := strings.NewReader(`{"name":"Updated","description":"Desc"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/groups/1", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id")
	c.SetParamNames("id")
	c.SetParamValues("1")
	c.Set("authUser", domain.User{ID: 1, FirstName: "Taro", LastName: "Yamada"})

	svc := new(mocks.MockGroupService)
	svc.On("Update", mock.Anything, uint64(1), "Updated", "Desc", uint64(1)).
		Return((*domain.Group)(nil), domain.ErrInternalServerError)

	h := &rest.GroupHandler{Service: svc}
	err := h.Update(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "internal server error", result["message"])
	svc.AssertExpectations(t)
}

// Case #1: Normal - authUser set + valid id -> service.Delete succeeds -> 204 No Content.
func TestGroupHandler_Delete_OK(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/groups/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id")
	c.SetParamNames("id")
	c.SetParamValues("1")
	c.Set("authUser", domain.User{ID: 42, FirstName: "Taro", LastName: "Yamada"})

	svc := new(mocks.MockGroupService)
	svc.On("Delete", mock.Anything, uint64(1), uint64(42)).Return(nil)

	h := &rest.GroupHandler{Service: svc}
	err := h.Delete(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Empty(t, rec.Body.String())
	svc.AssertExpectations(t)
}

// Case #2: Error - authUser type assertion fails -> 401.
func TestGroupHandler_Delete_Unauthorized(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/groups/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id")
	c.SetParamNames("id")
	c.SetParamValues("1")
	// authUser not set in context

	h := &rest.GroupHandler{Service: new(mocks.MockGroupService)}
	err := h.Delete(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "Unauthorized", result["message"])
}

// Case #3: Error - id is a string -> 400.
func TestGroupHandler_Delete_InvalidIDString(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/groups/abc", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id")
	c.SetParamNames("id")
	c.SetParamValues("abc")

	h := &rest.GroupHandler{Service: new(mocks.MockGroupService)}
	err := h.Delete(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "given param is not valid", result["message"])
}

// Case #4: Error - id = 0 -> 400.
func TestGroupHandler_Delete_ZeroID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/groups/0", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id")
	c.SetParamNames("id")
	c.SetParamValues("0")

	h := &rest.GroupHandler{Service: new(mocks.MockGroupService)}
	err := h.Delete(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "given param is not valid", result["message"])
}

// Case #5: Error - service returns ErrNotFound -> 404.
func TestGroupHandler_Delete_ServiceNotFound(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/groups/9999", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id")
	c.SetParamNames("id")
	c.SetParamValues("9999")
	c.Set("authUser", domain.User{ID: 1, FirstName: "Taro", LastName: "Yamada"})

	svc := new(mocks.MockGroupService)
	svc.On("Delete", mock.Anything, uint64(9999), uint64(1)).Return(domain.ErrNotFound)

	h := &rest.GroupHandler{Service: svc}
	err := h.Delete(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "your requested item is not found", result["message"])
	svc.AssertExpectations(t)
}

// Case #6: Error - service returns ErrInternalServerError -> 500.
func TestGroupHandler_Delete_ServiceInternalError(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/groups/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id")
	c.SetParamNames("id")
	c.SetParamValues("1")
	c.Set("authUser", domain.User{ID: 1, FirstName: "Taro", LastName: "Yamada"})

	svc := new(mocks.MockGroupService)
	svc.On("Delete", mock.Anything, uint64(1), uint64(1)).Return(domain.ErrInternalServerError)

	h := &rest.GroupHandler{Service: svc}
	err := h.Delete(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "internal server error", result["message"])
	svc.AssertExpectations(t)
}

func TestGroupHandler_ListNonGroupMembers_OK(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups/1/non-members", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id/non-members")
	c.SetParamNames("id")
	c.SetParamValues("1")

	svc := new(mocks.MockGroupService)
	users := []domain.User{
		{ID: 2, UUID: "00000000-0000-0000-0000-000000000002", FirstName: "Hanako", LastName: "Suzuki"},
	}
	svc.On("ListNonGroupMembers", mock.Anything, uint64(1), 500, 0, "").Return(users, 1, nil)

	h := &rest.GroupHandler{Service: svc}
	err := h.ListNonGroupMembers(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var result struct {
		Users []domain.User `json:"users"`
		Total int           `json:"total"`
	}
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Len(t, result.Users, 1)
	assert.Equal(t, "00000000-0000-0000-0000-000000000002", result.Users[0].UUID)
	assert.Equal(t, 1, result.Total)
	svc.AssertExpectations(t)
}

func TestGroupHandler_ListNonGroupMembers_WithQuery(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups/1/non-members?q=Suzuki", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id/non-members")
	c.SetParamNames("id")
	c.SetParamValues("1")

	svc := new(mocks.MockGroupService)
	users := []domain.User{
		{ID: 2, UUID: "00000000-0000-0000-0000-000000000002", FirstName: "Hanako", LastName: "Suzuki"},
	}
	svc.On("ListNonGroupMembers", mock.Anything, uint64(1), 500, 0, "Suzuki").Return(users, 5, nil)

	h := &rest.GroupHandler{Service: svc}
	err := h.ListNonGroupMembers(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	svc.AssertExpectations(t)
}

func TestGroupHandler_ListNonGroupMembers_EmptyResult(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups/1/non-members", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id/non-members")
	c.SetParamNames("id")
	c.SetParamValues("1")

	svc := new(mocks.MockGroupService)
	svc.On("ListNonGroupMembers", mock.Anything, uint64(1), 500, 0, "").Return([]domain.User{}, 0, nil)

	h := &rest.GroupHandler{Service: svc}
	err := h.ListNonGroupMembers(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var result struct {
		Users []domain.User `json:"users"`
		Total int           `json:"total"`
	}
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Empty(t, result.Users)
	assert.Equal(t, 0, result.Total)
	svc.AssertExpectations(t)
}

func TestGroupHandler_ListNonGroupMembers_InvalidID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups/abc/non-members", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id/non-members")
	c.SetParamNames("id")
	c.SetParamValues("abc")

	h := &rest.GroupHandler{Service: new(mocks.MockGroupService)}
	err := h.ListNonGroupMembers(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "given param is not valid", result["message"])
}

func TestGroupHandler_ListNonGroupMembers_ZeroID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups/0/non-members", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id/non-members")
	c.SetParamNames("id")
	c.SetParamValues("0")

	h := &rest.GroupHandler{Service: new(mocks.MockGroupService)}
	err := h.ListNonGroupMembers(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "given param is not valid", result["message"])
}

func TestGroupHandler_ListNonGroupMembers_InvalidLimit(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups/1/non-members?limit=abc", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id/non-members")
	c.SetParamNames("id")
	c.SetParamValues("1")

	h := &rest.GroupHandler{Service: new(mocks.MockGroupService)}
	err := h.ListNonGroupMembers(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "given param is not valid", result["message"])
}

func TestGroupHandler_ListNonGroupMembers_InvalidOffset(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups/1/non-members?offset=-1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id/non-members")
	c.SetParamNames("id")
	c.SetParamValues("1")

	h := &rest.GroupHandler{Service: new(mocks.MockGroupService)}
	err := h.ListNonGroupMembers(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "given param is not valid", result["message"])
}

func TestGroupHandler_ListNonGroupMembers_GroupNotFound(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups/9999/non-members", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id/non-members")
	c.SetParamNames("id")
	c.SetParamValues("9999")

	svc := new(mocks.MockGroupService)
	svc.On("ListNonGroupMembers", mock.Anything, uint64(9999), 500, 0, "").
		Return([]domain.User(nil), 0, domain.ErrNotFound)

	h := &rest.GroupHandler{Service: svc}
	err := h.ListNonGroupMembers(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "your requested item is not found", result["message"])
	svc.AssertExpectations(t)
}

func TestGroupHandler_ListNonGroupMembers_InternalError(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups/1/non-members", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id/non-members")
	c.SetParamNames("id")
	c.SetParamValues("1")

	svc := new(mocks.MockGroupService)
	svc.On("ListNonGroupMembers", mock.Anything, uint64(1), 500, 0, "").
		Return([]domain.User(nil), 0, domain.ErrInternalServerError)

	h := &rest.GroupHandler{Service: svc}
	err := h.ListNonGroupMembers(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "internal server error", result["message"])
	svc.AssertExpectations(t)
}

func TestGroupHandler_AddGroupMembers_OK(t *testing.T) {
	e := echo.New()
	body := strings.NewReader(`{"user_ids":[2,3]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/groups/1/members", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id/members")
	c.SetParamNames("id")
	c.SetParamValues("1")

	svc := new(mocks.MockGroupService)
	added := []domain.User{
		{ID: 2, UUID: "00000000-0000-0000-0000-000000000002", FirstName: "Hanako", LastName: "Suzuki"},
		{ID: 3, UUID: "00000000-0000-0000-0000-000000000003", FirstName: "Jiro", LastName: "Tanaka"},
	}
	svc.On("AddGroupMembers", mock.Anything, uint64(1), []uint64{2, 3}).Return(added, nil)

	h := &rest.GroupHandler{Service: svc}
	err := h.AddGroupMembers(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var result struct {
		Members []domain.User `json:"members"`
	}
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Len(t, result.Members, 2)
	assert.Equal(t, "00000000-0000-0000-0000-000000000002", result.Members[0].UUID)
	assert.Equal(t, "00000000-0000-0000-0000-000000000003", result.Members[1].UUID)
	svc.AssertExpectations(t)
}

func TestGroupHandler_AddGroupMembers_InvalidID(t *testing.T) {
	e := echo.New()
	body := strings.NewReader(`{"user_ids":[1]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/groups/abc/members", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id/members")
	c.SetParamNames("id")
	c.SetParamValues("abc")

	h := &rest.GroupHandler{Service: new(mocks.MockGroupService)}
	err := h.AddGroupMembers(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "given param is not valid", result["message"])
}

func TestGroupHandler_AddGroupMembers_ZeroID(t *testing.T) {
	e := echo.New()
	body := strings.NewReader(`{"user_ids":[1]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/groups/0/members", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id/members")
	c.SetParamNames("id")
	c.SetParamValues("0")

	h := &rest.GroupHandler{Service: new(mocks.MockGroupService)}
	err := h.AddGroupMembers(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGroupHandler_AddGroupMembers_EmptyUserIDs(t *testing.T) {
	e := echo.New()
	body := strings.NewReader(`{"user_ids":[]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/groups/1/members", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id/members")
	c.SetParamNames("id")
	c.SetParamValues("1")

	h := &rest.GroupHandler{Service: new(mocks.MockGroupService)}
	err := h.AddGroupMembers(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "given param is not valid", result["message"])
}

func TestGroupHandler_AddGroupMembers_GroupNotFound(t *testing.T) {
	e := echo.New()
	body := strings.NewReader(`{"user_ids":[1]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/groups/9999/members", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id/members")
	c.SetParamNames("id")
	c.SetParamValues("9999")

	svc := new(mocks.MockGroupService)
	svc.On("AddGroupMembers", mock.Anything, uint64(9999), []uint64{1}).
		Return([]domain.User(nil), domain.ErrNotFound)

	h := &rest.GroupHandler{Service: svc}
	err := h.AddGroupMembers(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "your requested item is not found", result["message"])
	svc.AssertExpectations(t)
}

func TestGroupHandler_AddGroupMembers_AlreadyMember(t *testing.T) {
	e := echo.New()
	body := strings.NewReader(`{"user_ids":[1]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/groups/1/members", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id/members")
	c.SetParamNames("id")
	c.SetParamValues("1")

	svc := new(mocks.MockGroupService)
	svc.On("AddGroupMembers", mock.Anything, uint64(1), []uint64{1}).
		Return([]domain.User(nil), domain.ErrConflict)

	h := &rest.GroupHandler{Service: svc}
	err := h.AddGroupMembers(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusConflict, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "your item already exist", result["message"])
	svc.AssertExpectations(t)
}

func TestGroupHandler_AddGroupMembers_InternalError(t *testing.T) {
	e := echo.New()
	body := strings.NewReader(`{"user_ids":[2]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/groups/1/members", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id/members")
	c.SetParamNames("id")
	c.SetParamValues("1")

	svc := new(mocks.MockGroupService)
	svc.On("AddGroupMembers", mock.Anything, uint64(1), []uint64{2}).
		Return([]domain.User(nil), domain.ErrInternalServerError)

	h := &rest.GroupHandler{Service: svc}
	err := h.AddGroupMembers(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "internal server error", result["message"])
	svc.AssertExpectations(t)
}

func TestGroupHandler_DeleteGroupMembers_OK(t *testing.T) {
	e := echo.New()
	body := strings.NewReader(`{"user_ids":[2,3]}`)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/groups/1/members", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id/members")
	c.SetParamNames("id")
	c.SetParamValues("1")

	svc := new(mocks.MockGroupService)
	svc.On("RemoveGroupMembers", mock.Anything, uint64(1), []uint64{2, 3}).Return(nil)

	h := &rest.GroupHandler{Service: svc}
	err := h.DeleteGroupMembers(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, rec.Code)
	svc.AssertExpectations(t)
}

func TestGroupHandler_DeleteGroupMembers_InvalidID_NotInteger(t *testing.T) {
	e := echo.New()
	body := strings.NewReader(`{"user_ids":[1]}`)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/groups/abc/members", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id/members")
	c.SetParamNames("id")
	c.SetParamValues("abc")

	h := &rest.GroupHandler{Service: new(mocks.MockGroupService)}
	err := h.DeleteGroupMembers(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "given param is not valid", result["message"])
}

func TestGroupHandler_DeleteGroupMembers_InvalidID_Zero(t *testing.T) {
	e := echo.New()
	body := strings.NewReader(`{"user_ids":[1]}`)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/groups/0/members", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id/members")
	c.SetParamNames("id")
	c.SetParamValues("0")

	h := &rest.GroupHandler{Service: new(mocks.MockGroupService)}
	err := h.DeleteGroupMembers(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "given param is not valid", result["message"])
}

func TestGroupHandler_DeleteGroupMembers_InvalidID_Negative(t *testing.T) {
	e := echo.New()
	body := strings.NewReader(`{"user_ids":[1]}`)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/groups/-1/members", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id/members")
	c.SetParamNames("id")
	c.SetParamValues("-1")

	h := &rest.GroupHandler{Service: new(mocks.MockGroupService)}
	err := h.DeleteGroupMembers(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "given param is not valid", result["message"])
}

func TestGroupHandler_DeleteGroupMembers_EmptyUserIDs(t *testing.T) {
	e := echo.New()
	body := strings.NewReader(`{"user_ids":[]}`)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/groups/1/members", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id/members")
	c.SetParamNames("id")
	c.SetParamValues("1")

	h := &rest.GroupHandler{Service: new(mocks.MockGroupService)}
	err := h.DeleteGroupMembers(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "given param is not valid", result["message"])
}

func TestGroupHandler_DeleteGroupMembers_GroupNotFound(t *testing.T) {
	e := echo.New()
	body := strings.NewReader(`{"user_ids":[1]}`)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/groups/9999/members", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id/members")
	c.SetParamNames("id")
	c.SetParamValues("9999")

	svc := new(mocks.MockGroupService)
	svc.On("RemoveGroupMembers", mock.Anything, uint64(9999), []uint64{1}).
		Return(domain.ErrNotFound)

	h := &rest.GroupHandler{Service: svc}
	err := h.DeleteGroupMembers(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "your requested item is not found", result["message"])
	svc.AssertExpectations(t)
}

func TestGroupHandler_DeleteGroupMembers_NonMemberUserID(t *testing.T) {
	e := echo.New()
	body := strings.NewReader(`{"user_ids":[9999]}`)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/groups/1/members", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id/members")
	c.SetParamNames("id")
	c.SetParamValues("1")

	svc := new(mocks.MockGroupService)
	svc.On("RemoveGroupMembers", mock.Anything, uint64(1), []uint64{9999}).
		Return(domain.ErrNotFound)

	h := &rest.GroupHandler{Service: svc}
	err := h.DeleteGroupMembers(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "your requested item is not found", result["message"])
	svc.AssertExpectations(t)
}

func TestGroupHandler_DeleteGroupMembers_InternalError(t *testing.T) {
	e := echo.New()
	body := strings.NewReader(`{"user_ids":[2]}`)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/groups/1/members", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/groups/:id/members")
	c.SetParamNames("id")
	c.SetParamValues("1")

	svc := new(mocks.MockGroupService)
	svc.On("RemoveGroupMembers", mock.Anything, uint64(1), []uint64{2}).
		Return(domain.ErrInternalServerError)

	h := &rest.GroupHandler{Service: svc}
	err := h.DeleteGroupMembers(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "internal server error", result["message"])
	svc.AssertExpectations(t)
}
