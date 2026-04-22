package rest_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"github.com/hrk-m/spec-to-dev-workflow/sample-api/internal/rest"
)

func TestHealthCheck_OK(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectPing()

	e := echo.New()
	rest.RegisterHealthHandler(e, db)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "ok", result["status"])
	_, hasMessage := result["message"]
	assert.False(t, hasMessage, "message field should be omitted on success")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHealthCheck_DBUnavailable(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectPing().WillReturnError(sql.ErrConnDone)

	e := echo.New()
	rest.RegisterHealthHandler(e, db)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)

	var result map[string]string
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, "error", result["status"])
	assert.Equal(t, "db unavailable", result["message"])
	assert.NoError(t, mock.ExpectationsWereMet())
}
