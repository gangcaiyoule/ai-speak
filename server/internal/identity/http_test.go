package identity

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHTTPHandlerRegisterRoutes 验证身份路由能够返回结构化占位响应。
func TestHTTPHandlerRegisterRoutes(t *testing.T) {
	mux := http.NewServeMux()
	NewHTTPHandler(StubAuthService{}).RegisterRoutes(mux)
	request := httptest.NewRequest(http.MethodPost, "/v1/auth/login", nil)
	recording := httptest.NewRecorder()
	mux.ServeHTTP(recording, request)
	if recording.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want %d", recording.Code, http.StatusNotImplemented)
	}
}
