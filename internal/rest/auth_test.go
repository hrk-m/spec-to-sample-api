package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/hrk-m/spec-to-dev-workflow/sample-api/domain"
	"github.com/hrk-m/spec-to-dev-workflow/sample-api/internal/rest"
	"github.com/hrk-m/spec-to-dev-workflow/sample-api/internal/rest/mocks"
)

const testUUID = "test-uuid-1234"

func TestAuthMiddleware_Development_ValidUUID(t *testing.T) {
	t.Setenv("DEV_USER_UUID", testUUID)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	repo := new(mocks.MockAuthService)
	user := domain.User{ID: 1, UUID: testUUID, FirstName: "Taro", LastName: "Yamada"}
	repo.On("GetByUUID", mock.Anything, testUUID).Return(user, nil)

	mw := rest.AuthMiddleware("development", repo)
	handler := mw(func(c echo.Context) error {
		authUser, ok := c.Get("authUser").(domain.User)
		assert.True(t, ok)
		assert.Equal(t, user, authUser)
		return c.JSON(http.StatusOK, authUser)
	})

	err := handler(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	repo.AssertExpectations(t)
}

func TestAuthMiddleware_Development_NonexistentUUID(t *testing.T) {
	t.Setenv("DEV_USER_UUID", "nonexistent-uuid")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	repo := new(mocks.MockAuthService)
	repo.On("GetByUUID", mock.Anything, "nonexistent-uuid").Return(domain.User{}, domain.ErrNotFound)

	mw := rest.AuthMiddleware("development", repo)
	handler := mw(func(c echo.Context) error {
		return c.JSON(http.StatusOK, nil)
	})

	err := handler(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
	repo.AssertExpectations(t)
}

func TestAuthHandler_GetMe_OK(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	user := domain.User{ID: 1, UUID: testUUID, FirstName: "Taro", LastName: "Yamada"}
	c.Set("authUser", user)

	h := &rest.AuthHandler{}
	err := h.GetMe(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var result domain.User
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, user.ID, result.ID)
	assert.Equal(t, user.UUID, result.UUID)
	assert.Equal(t, user.FirstName, result.FirstName)
	assert.Equal(t, user.LastName, result.LastName)
}

func TestAuthHandler_GetMe_NoAuthUser(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := &rest.AuthHandler{}
	err := h.GetMe(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var result rest.ResponseError
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "Unauthorized", result.Message)
}
