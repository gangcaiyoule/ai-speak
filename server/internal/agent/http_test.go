package agent

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHTTPHandlerRegisterRoutes 验证 Agent 路由能够返回结构化占位响应。
func TestHTTPHandlerRegisterRoutes(t *testing.T) {
	mux := http.NewServeMux()
	NewHTTPHandler(StubService{}).RegisterRoutes(mux)
	request := httptest.NewRequest(http.MethodPost, "/v1/agent-threads", nil)
	recording := httptest.NewRecorder()
	mux.ServeHTTP(recording, request)
	if recording.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want %d", recording.Code, http.StatusNotImplemented)
	}
}
