package coaching

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHTTPHandlerRegisterRoutes 验证场景目录路由返回真实场景数据。
func TestHTTPHandlerRegisterRoutes(t *testing.T) {
	mux := http.NewServeMux()
	NewHTTPHandler().RegisterRoutes(mux)
	request := httptest.NewRequest(http.MethodGet, "/v1/scenes", nil)
	recording := httptest.NewRecorder()
	mux.ServeHTTP(recording, request)
	if recording.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recording.Code, http.StatusOK)
	}
	if recording.Body.Len() == 0 {
		t.Fatal("scene response body is empty")
	}
}

func TestHTTPHandlerSceneDetailAndRoles(t *testing.T) {
	mux := http.NewServeMux()
	NewHTTPHandler().RegisterRoutes(mux)

	detail := httptest.NewRecorder()
	mux.ServeHTTP(detail, httptest.NewRequest(http.MethodGet, "/v1/scenes/self-introduction", nil))
	if detail.Code != http.StatusOK {
		t.Fatalf("detail status = %d, want %d", detail.Code, http.StatusOK)
	}

	roles := httptest.NewRecorder()
	mux.ServeHTTP(roles, httptest.NewRequest(http.MethodGet, "/v1/scenes/self-introduction/roles", nil))
	if roles.Code != http.StatusOK {
		t.Fatalf("roles status = %d, want %d", roles.Code, http.StatusOK)
	}

	notFound := httptest.NewRecorder()
	mux.ServeHTTP(notFound, httptest.NewRequest(http.MethodGet, "/v1/scenes/missing", nil))
	if notFound.Code != http.StatusNotFound {
		t.Fatalf("missing status = %d, want %d", notFound.Code, http.StatusNotFound)
	}
}
