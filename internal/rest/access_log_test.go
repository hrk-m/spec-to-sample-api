package rest_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hrk-m/spec-to-dev-workflow/sample-api/domain"
	"github.com/hrk-m/spec-to-dev-workflow/sample-api/internal/rest"
)

func newTestLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(buf, nil))
}

func TestAccessLogMiddleware_AuthenticatedRequest(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := newTestLogger(buf)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/123", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	user := domain.User{ID: 1, UUID: "test-uuid-5678", FirstName: "Taro", LastName: "Yamada"}
	c.Set("authUser", user)

	mw := rest.AccessLogMiddleware(logger)
	handler := mw(func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	err := handler(c)
	require.NoError(t, err)

	var logEntry map[string]interface{}
	require.NoError(t, json.NewDecoder(buf).Decode(&logEntry))

	assert.Equal(t, "test-uuid-5678", logEntry["login_user"])
	assert.Equal(t, "test-uuid-5678", rec.Header().Get("X-Login-User"))
}

func TestAccessLogMiddleware_NoAuthUser(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := newTestLogger(buf)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// authUser をセットしない

	mw := rest.AccessLogMiddleware(logger)
	handler := mw(func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	err := handler(c)
	require.NoError(t, err)

	var logEntry map[string]interface{}
	require.NoError(t, json.NewDecoder(buf).Decode(&logEntry))

	assert.Equal(t, "", logEntry["login_user"])
}

func TestAccessLogMiddleware_Latency(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := newTestLogger(buf)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := rest.AccessLogMiddleware(logger)
	handler := mw(func(c echo.Context) error {
		return c.JSON(http.StatusOK, nil)
	})

	err := handler(c)
	require.NoError(t, err)

	var logEntry map[string]interface{}
	require.NoError(t, json.NewDecoder(buf).Decode(&logEntry))

	latency, ok := logEntry["latency_s"].(float64)
	assert.True(t, ok, "latency_s should be float64")
	assert.GreaterOrEqual(t, latency, float64(0))
}

func TestAccessLogMiddleware_StatusCode(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := newTestLogger(buf)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/999", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := rest.AccessLogMiddleware(logger)
	handler := mw(func(c echo.Context) error {
		return c.JSON(http.StatusNotFound, rest.ResponseError{Message: "not found"})
	})

	err := handler(c)
	require.NoError(t, err)

	var logEntry map[string]interface{}
	require.NoError(t, json.NewDecoder(buf).Decode(&logEntry))

	status, ok := logEntry["status"].(float64)
	assert.True(t, ok, "status should be a number")
	assert.Equal(t, float64(http.StatusNotFound), status)
}

func TestAccessLogMiddleware_AuthorizationHeaderMasked(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := newTestLogger(buf)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := rest.AccessLogMiddleware(logger)
	handler := mw(func(c echo.Context) error {
		return c.JSON(http.StatusOK, nil)
	})

	err := handler(c)
	require.NoError(t, err)

	var logEntry map[string]interface{}
	require.NoError(t, json.NewDecoder(buf).Decode(&logEntry))

	header, ok := logEntry["header"].(map[string]interface{})
	require.True(t, ok, "header should be an object")

	authValues, ok := header["Authorization"].([]interface{})
	require.True(t, ok, "header.Authorization should be an array")
	require.Len(t, authValues, 1)
	assert.Equal(t, "[REDACTED]", authValues[0])

	// Content-Type はそのまま出力される
	ctValues, ok := header["Content-Type"].([]interface{})
	require.True(t, ok, "header.Content-Type should be an array")
	require.Len(t, ctValues, 1)
	assert.Equal(t, "application/json", ctValues[0])
}
