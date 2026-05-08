package rest_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hrk-m/spec-to-dev-workflow/sample-api/domain"
	"github.com/hrk-m/spec-to-dev-workflow/sample-api/internal/rest"
)

const (
	accessLogStatusOK    = "ok"
	accessLogMsgNotFound = "not found"
	accessLogStatusKey   = "status"
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

	user := domain.User{ID: 1, UUID: "test-uuid-5678", FirstName: testFirstNameTaro, LastName: testLastNameYamada}
	c.Set("authUser", user)

	mw := rest.AccessLogMiddleware(logger)
	handler := mw(func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{accessLogStatusKey: accessLogStatusOK})
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
		return c.JSON(http.StatusOK, map[string]string{accessLogStatusKey: accessLogStatusOK})
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
		return c.JSON(http.StatusNotFound, rest.ResponseError{Message: accessLogMsgNotFound})
	})

	err := handler(c)
	require.NoError(t, err)

	var logEntry map[string]interface{}
	require.NoError(t, json.NewDecoder(buf).Decode(&logEntry))

	status, ok := logEntry[accessLogStatusKey].(float64)
	assert.True(t, ok, "status should be a number")
	assert.Equal(t, float64(http.StatusNotFound), status)
}

func TestAccessLogMiddleware_5xxUsesErrorLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := newTestLogger(buf)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups/30/members", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	user := domain.User{ID: 1, UUID: "test-uuid-1234", FirstName: testFirstNameTaro, LastName: testLastNameYamada}
	c.Set("authUser", user)

	mw := rest.AccessLogMiddleware(logger)
	handler := mw(func(c echo.Context) error {
		return c.JSON(http.StatusInternalServerError, rest.ResponseError{Message: "internal server error"})
	})

	err := handler(c)
	require.NoError(t, err)

	var logEntry map[string]interface{}
	require.NoError(t, json.NewDecoder(buf).Decode(&logEntry))

	assert.Equal(t, "ERROR", logEntry["level"])
	status, ok := logEntry[accessLogStatusKey].(float64)
	assert.True(t, ok)
	assert.Equal(t, float64(http.StatusInternalServerError), status)
}

func TestAccessLogMiddleware_5xxWithHandlerError(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := newTestLogger(buf)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups/30/members", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := rest.AccessLogMiddleware(logger)
	handler := mw(func(c echo.Context) error {
		c.Response().WriteHeader(http.StatusInternalServerError)
		return domain.ErrInternalServerError
	})

	_ = handler(c)

	var logEntry map[string]interface{}
	require.NoError(t, json.NewDecoder(buf).Decode(&logEntry))

	assert.Equal(t, "ERROR", logEntry["level"])
	assert.Equal(t, domain.ErrInternalServerError.Error(), logEntry["error"])
}

func TestAccessLogMiddleware_ErrorReturnWithoutWriteHeader(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := newTestLogger(buf)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups/30/members", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := rest.AccessLogMiddleware(logger)
	handler := mw(func(c echo.Context) error {
		// handler returns error without calling WriteHeader — status remains 0
		return domain.ErrInternalServerError
	})

	_ = handler(c)

	var logEntry map[string]interface{}
	require.NoError(t, json.NewDecoder(buf).Decode(&logEntry))

	// (a) log level must be ERROR
	assert.Equal(t, "ERROR", logEntry["level"])

	// (b) status field must be 500
	status, ok := logEntry[accessLogStatusKey].(float64)
	assert.True(t, ok, "status should be a number")
	assert.Equal(t, float64(http.StatusInternalServerError), status)

	// (c) error field must be present
	assert.Equal(t, domain.ErrInternalServerError.Error(), logEntry["error"])
}

func TestAccessLogMiddleware_4xxUsesInfoLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := newTestLogger(buf)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/999", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := rest.AccessLogMiddleware(logger)
	handler := mw(func(c echo.Context) error {
		return c.JSON(http.StatusNotFound, rest.ResponseError{Message: accessLogMsgNotFound})
	})

	err := handler(c)
	require.NoError(t, err)

	var logEntry map[string]interface{}
	require.NoError(t, json.NewDecoder(buf).Decode(&logEntry))

	assert.Equal(t, "INFO", logEntry["level"])
	_, hasError := logEntry["error"]
	assert.False(t, hasError, "error field should not be present for non-5xx responses")
}

func TestAccessLogMiddleware_AllowListHeaderFilter(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := newTestLogger(buf)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	// 許可リスト内ヘッダー: Referer / Sec-Ch-Ua / Sec-Ch-Ua-Mobile / Sec-Ch-Ua-Platform
	req.Header.Set("Referer", "http://localhost:3000/")
	req.Header.Set("Sec-Ch-Ua", `"Chromium";v="146"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"macOS"`)
	// 許可リスト外ヘッダー
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Authorization", "Bearer secret-token")
	req.Header.Set("Cookie", "session=abc123")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Accept-Language", "ja,en;q=0.9")
	req.Header.Set("Connection", "keep-alive")
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

	// 許可リスト内のヘッダーが含まれる
	assert.Contains(t, header, "Referer")
	assert.Contains(t, header, "Sec-Ch-Ua")
	assert.Contains(t, header, "Sec-Ch-Ua-Mobile")
	assert.Contains(t, header, "Sec-Ch-Ua-Platform")

	// 許可リスト外のヘッダーが含まれない
	assert.NotContains(t, header, "User-Agent")
	assert.NotContains(t, header, "Authorization")
	assert.NotContains(t, header, "Cookie")
	assert.NotContains(t, header, "Accept")
	assert.NotContains(t, header, "Sec-Fetch-Site")
	assert.NotContains(t, header, "Sec-Fetch-Mode")
	assert.NotContains(t, header, "Sec-Fetch-Dest")
	assert.NotContains(t, header, "Accept-Encoding")
	assert.NotContains(t, header, "Accept-Language")
	assert.NotContains(t, header, "Connection")
}

// #6: request_body 記録（Content-Type: application/json + body あり）
func TestAccessLogMiddleware_RequestBodyRecorded(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := newTestLogger(buf)

	e := echo.New()
	body := `{"email":"user@example.com","name":"Alice"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := rest.AccessLogMiddleware(logger)
	handler := mw(func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{accessLogStatusKey: accessLogStatusOK})
	})

	err := handler(c)
	require.NoError(t, err)

	var logEntry map[string]any
	require.NoError(t, json.NewDecoder(buf).Decode(&logEntry))

	reqBody, ok := logEntry["request_body"].(map[string]any)
	require.True(t, ok, "request_body should be an object")
	assert.Equal(t, "user@example.com", reqBody["email"])
	assert.Equal(t, "Alice", reqBody["name"])
}

// #7: request_body マスク（機微キーが [REDACTED] に置換される・ネスト再帰・大文字小文字無視）
func TestAccessLogMiddleware_RequestBodyMasked(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := newTestLogger(buf)

	e := echo.New()
	// 全マスク対象キーを含む（大文字小文字混在・ネスト・配列）
	body := `{
		"email": "user@example.com",
		"password": "p@ssw0rd",
		"Token": "tok123",
		"access_token": "at123",
		"refresh_token": "rt123",
		"api_key": "ak123",
		"SECRET": "s3cr3t",
		"Authorization": "Bearer xyz",
		"nested": {
			"password": "nestedpass"
		},
		"items": [
			{"token": "itemtok"}
		]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := rest.AccessLogMiddleware(logger)
	handler := mw(func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{accessLogStatusKey: accessLogStatusOK})
	})

	err := handler(c)
	require.NoError(t, err)

	var logEntry map[string]any
	require.NoError(t, json.NewDecoder(buf).Decode(&logEntry))

	reqBody, ok := logEntry["request_body"].(map[string]any)
	require.True(t, ok, "request_body should be an object")

	// 非機微フィールドはそのまま
	assert.Equal(t, "user@example.com", reqBody["email"])
	// 機微フィールドはすべて [REDACTED]
	assert.Equal(t, "[REDACTED]", reqBody["password"])
	assert.Equal(t, "[REDACTED]", reqBody["Token"])
	assert.Equal(t, "[REDACTED]", reqBody["access_token"])
	assert.Equal(t, "[REDACTED]", reqBody["refresh_token"])
	assert.Equal(t, "[REDACTED]", reqBody["api_key"])
	assert.Equal(t, "[REDACTED]", reqBody["SECRET"])
	assert.Equal(t, "[REDACTED]", reqBody["Authorization"])
	// ネスト内の機微フィールド
	nested, ok := reqBody["nested"].(map[string]any)
	require.True(t, ok, "nested should be an object")
	assert.Equal(t, "[REDACTED]", nested["password"])
	// 配列内オブジェクトの機微フィールド
	items, ok := reqBody["items"].([]any)
	require.True(t, ok, "items should be an array")
	require.Len(t, items, 1)
	item, ok := items[0].(map[string]any)
	require.True(t, ok, "item should be an object")
	assert.Equal(t, "[REDACTED]", item["token"])
}

// #8: request_body 非 JSON / 空（フィールド自体が出力されない）
func TestAccessLogMiddleware_RequestBodyNotRecordedForNonJSON(t *testing.T) {
	t.Run("text/plain content-type", func(t *testing.T) {
		buf := &bytes.Buffer{}
		logger := newTestLogger(buf)

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader("plain text body"))
		req.Header.Set("Content-Type", "text/plain")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		mw := rest.AccessLogMiddleware(logger)
		handler := mw(func(c echo.Context) error {
			return c.JSON(http.StatusOK, map[string]string{accessLogStatusKey: accessLogStatusOK})
		})

		err := handler(c)
		require.NoError(t, err)

		var logEntry map[string]any
		require.NoError(t, json.NewDecoder(buf).Decode(&logEntry))

		_, hasRequestBody := logEntry["request_body"]
		assert.False(t, hasRequestBody, "request_body should not be present for non-JSON content-type")
	})

	t.Run("GET request without body", func(t *testing.T) {
		buf := &bytes.Buffer{}
		logger := newTestLogger(buf)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users/1", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		mw := rest.AccessLogMiddleware(logger)
		handler := mw(func(c echo.Context) error {
			return c.JSON(http.StatusOK, map[string]string{accessLogStatusKey: accessLogStatusOK})
		})

		err := handler(c)
		require.NoError(t, err)

		var logEntry map[string]any
		require.NoError(t, json.NewDecoder(buf).Decode(&logEntry))

		_, hasRequestBody := logEntry["request_body"]
		assert.False(t, hasRequestBody, "request_body should not be present for requests without body")
	})
}

// #9: request_body 4KB 超（_truncated: true, size_bytes: N）
func TestAccessLogMiddleware_RequestBodyTruncated(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := newTestLogger(buf)

	e := echo.New()
	// 5KB の JSON を作成
	largeValue := strings.Repeat("a", 5000)
	body := fmt.Sprintf(`{"data":"%s"}`, largeValue)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.ContentLength = int64(len(body))
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := rest.AccessLogMiddleware(logger)
	handler := mw(func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{accessLogStatusKey: accessLogStatusOK})
	})

	err := handler(c)
	require.NoError(t, err)

	var logEntry map[string]any
	require.NoError(t, json.NewDecoder(buf).Decode(&logEntry))

	reqBody, ok := logEntry["request_body"].(map[string]any)
	require.True(t, ok, "request_body should be an object")
	assert.Equal(t, true, reqBody["_truncated"])
	sizeBytes, ok := reqBody["size_bytes"].(float64)
	assert.True(t, ok, "size_bytes should be a number")
	assert.Greater(t, sizeBytes, float64(4096))
}

// #10: request_body パース失敗（JSON として解釈できない場合 _parse_error: true）
func TestAccessLogMiddleware_RequestBodyParseError(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := newTestLogger(buf)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader("{invalid json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := rest.AccessLogMiddleware(logger)
	handler := mw(func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{accessLogStatusKey: accessLogStatusOK})
	})

	err := handler(c)
	require.NoError(t, err)

	var logEntry map[string]any
	require.NoError(t, json.NewDecoder(buf).Decode(&logEntry))

	reqBody, ok := logEntry["request_body"].(map[string]any)
	require.True(t, ok, "request_body should be an object")
	assert.Equal(t, true, reqBody["_parse_error"])
}

// #11: request_body 取得後にハンドラー側で body を再読できる
func TestAccessLogMiddleware_RequestBodyCanBeReadByHandler(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := newTestLogger(buf)

	e := echo.New()
	body := `{"email":"user@example.com","password":"p@ssw0rd"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	var handlerReceivedBody string
	mw := rest.AccessLogMiddleware(logger)
	handler := mw(func(c echo.Context) error {
		// ハンドラー側でボディを再度読む（Bind 相当）
		data, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return err
		}
		handlerReceivedBody = string(data)
		return c.JSON(http.StatusOK, map[string]string{accessLogStatusKey: accessLogStatusOK})
	})

	err := handler(c)
	require.NoError(t, err)

	// ハンドラーが元のボディと同じ内容を読めることを確認
	assert.Equal(t, body, handlerReceivedBody)

	// ログにも request_body が記録され、password はマスクされていること
	var logEntry map[string]any
	require.NoError(t, json.NewDecoder(buf).Decode(&logEntry))

	reqBody, ok := logEntry["request_body"].(map[string]any)
	require.True(t, ok, "request_body should be an object")
	assert.Equal(t, "user@example.com", reqBody["email"])
	assert.Equal(t, "[REDACTED]", reqBody["password"])
}

func TestAccessLogMiddleware_AllowListCaseInsensitive(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := newTestLogger(buf)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	// 小文字で送っても許可リストに含まれるヘッダーは記録される
	req.Header["referer"] = []string{"http://localhost:3000/"}
	req.Header["sec-ch-ua"] = []string{`"Chromium";v="146"`}
	req.Header["sec-ch-ua-mobile"] = []string{"?0"}
	req.Header["sec-ch-ua-platform"] = []string{`"macOS"`}
	// 許可リスト外
	req.Header["user-agent"] = []string{"Mozilla/5.0"}
	req.Header["authorization"] = []string{"Bearer token"}
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

	// 大文字小文字を問わず許可リスト内のヘッダーは含まれる
	foundReferer := false
	foundSecChUa := false
	foundSecChUaMobile := false
	foundSecChUaPlatform := false
	foundUserAgent := false
	foundAuth := false
	for k := range header {
		switch strings.ToLower(k) {
		case "referer":
			foundReferer = true
		case "sec-ch-ua":
			foundSecChUa = true
		case "sec-ch-ua-mobile":
			foundSecChUaMobile = true
		case "sec-ch-ua-platform":
			foundSecChUaPlatform = true
		case "user-agent":
			foundUserAgent = true
		case "authorization":
			foundAuth = true
		}
	}
	assert.True(t, foundReferer, "referer should be in header")
	assert.True(t, foundSecChUa, "sec-ch-ua should be in header")
	assert.True(t, foundSecChUaMobile, "sec-ch-ua-mobile should be in header")
	assert.True(t, foundSecChUaPlatform, "sec-ch-ua-platform should be in header")
	assert.False(t, foundUserAgent, "user-agent should not be in header")
	assert.False(t, foundAuth, "authorization should not be in header")
}
