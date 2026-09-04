package coaching

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gangcaiyoule/ai-speak/server/internal/coaching/practice"
	"github.com/gangcaiyoule/ai-speak/server/internal/coaching/scene"
	"github.com/gangcaiyoule/ai-speak/server/internal/identity"
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

func TestPracticePlanHTTPFlowRequiresAuthAndScopesPlansToActor(t *testing.T) {
	auth := identity.NewService(identity.NewMemoryRepository())
	registerAndLogin := func(email string) string {
		if _, err := auth.Register(t.Context(), identity.RegisterInput{Email: email, Password: "password123"}); err != nil {
			t.Fatal(err)
		}
		result, err := auth.Login(t.Context(), identity.LoginInput{Email: email, Password: "password123"})
		if err != nil {
			t.Fatal(err)
		}
		return result.Token
	}
	token1, token2 := registerAndLogin("one@example.com"), registerAndLogin("two@example.com")
	h := NewHTTPHandlerWithDependencies(auth, practice.NewPlanService(practice.NewMemoryPlanRepository(), scene.NewCatalog()), scene.NewCatalog())
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	request := func(method, path, token, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec
	}
	if got := request(http.MethodPost, "/v1/practice-plans", "", `{"scene_id":"self-introduction"}`); got.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated create = %d", got.Code)
	}
	created := request(http.MethodPost, "/v1/practice-plans", token1, `{"scene_id":"self-introduction","scene_version":1,"role_id":"recruiter","practice_option_id":"self-introduction-full","objective":"Improve structure"}`)
	if created.Code != http.StatusCreated {
		t.Fatalf("create = %d, body=%s", created.Code, created.Body.String())
	}
	var envelope struct{ Plan practice.Plan }
	if err := json.NewDecoder(created.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	if got := request(http.MethodGet, "/v1/practice-plans/"+envelope.Plan.ID, token2, ""); got.Code != http.StatusNotFound {
		t.Fatalf("cross-user detail = %d", got.Code)
	}
	if got := request(http.MethodGet, "/v1/practice-plans", token2, ""); got.Code != http.StatusOK || strings.Contains(got.Body.String(), envelope.Plan.ID) {
		t.Fatalf("cross-user list = %d, %s", got.Code, got.Body.String())
	}
	archived := request(http.MethodPost, "/v1/practice-plans/"+envelope.Plan.ID+"/archive", token1, "")
	if archived.Code != http.StatusOK || !strings.Contains(archived.Body.String(), `"ARCHIVED"`) {
		t.Fatalf("archive = %d, %s", archived.Code, archived.Body.String())
	}
}

func TestPracticePlanHTTPRejectsInvalidJSON(t *testing.T) {
	auth := identity.NewService(identity.NewMemoryRepository())
	_, err := auth.Register(t.Context(), identity.RegisterInput{Email: "user@example.com", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	login, err := auth.Login(t.Context(), identity.LoginInput{Email: "user@example.com", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	h := NewHTTPHandlerWithDependencies(auth, practice.NewPlanService(practice.NewMemoryPlanRepository(), scene.NewCatalog()), scene.NewCatalog())
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	req := httptest.NewRequest(http.MethodPost, "/v1/practice-plans", strings.NewReader(`{"scene_id":"self-introduction","scene_version":1,"role_id":"recruiter","practice_option_id":"bad","objective":"x"}`))
	req.Header.Set("Authorization", "Bearer "+login.Token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid config = %d", rec.Code)
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
