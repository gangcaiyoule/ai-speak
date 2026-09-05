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

func TestPracticeSessionHTTPFlowRequiresAuthAndScopesSessionToActor(t *testing.T) {
	auth := identity.NewService(identity.NewMemoryRepository())
	login := func(email string) string {
		t.Helper()
		if _, err := auth.Register(t.Context(), identity.RegisterInput{Email: email, Password: "password123"}); err != nil {
			t.Fatal(err)
		}
		result, err := auth.Login(t.Context(), identity.LoginInput{Email: email, Password: "password123"})
		if err != nil {
			t.Fatal(err)
		}
		return result.Token
	}
	token1, token2 := login("session-one@example.com"), login("session-two@example.com")
	catalog := scene.NewCatalog()
	actor, err := auth.Authenticate(t.Context(), token1)
	if err != nil {
		t.Fatal(err)
	}
	planRepo := practice.NewMemoryPlanRepository()
	planService := practice.NewPlanService(planRepo, catalog)
	plan, err := planService.CreatePlan(t.Context(), actor.UserID, practice.CreatePlanInput{
		SceneID: "self-introduction", SceneVersion: 1, RoleID: "recruiter", PracticeOptionID: "self-introduction-full", Objective: "practice structure",
	})
	if err != nil {
		t.Fatal(err)
	}
	h := NewHTTPHandlerWithAllDependencies(auth, planService, practice.NewSessionService(practice.NewMemorySessionRepository(), planService, catalog), catalog)
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

	if got := request(http.MethodPost, "/v1/practice-sessions", "", `{"plan_id":"`+plan.ID+`"}`); got.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated create = %d", got.Code)
	}
	created := request(http.MethodPost, "/v1/practice-sessions", token1, `{"plan_id":"`+plan.ID+`"}`)
	if created.Code != http.StatusCreated {
		t.Fatalf("create = %d, body=%s", created.Code, created.Body.String())
	}
	var createdEnvelope struct{ Session practice.Session }
	if err := json.NewDecoder(created.Body).Decode(&createdEnvelope); err != nil {
		t.Fatal(err)
	}
	if createdEnvelope.Session.Status != practice.SessionStatusDraft || len(createdEnvelope.Session.Questions) != 1 {
		t.Fatalf("created session = %#v", createdEnvelope.Session)
	}

	sessionID := createdEnvelope.Session.ID
	if got := request(http.MethodGet, "/v1/practice-sessions/"+sessionID, token2, ""); got.Code != http.StatusNotFound {
		t.Fatalf("cross-user detail = %d, body=%s", got.Code, got.Body.String())
	}
	if got := request(http.MethodPost, "/v1/practice-sessions/"+sessionID+"/activation", token2, ""); got.Code != http.StatusNotFound {
		t.Fatalf("cross-user activation = %d, body=%s", got.Code, got.Body.String())
	}
	activated := request(http.MethodPost, "/v1/practice-sessions/"+sessionID+"/activation", token1, "")
	if activated.Code != http.StatusOK || !strings.Contains(activated.Body.String(), `"ACTIVE"`) || !strings.Contains(activated.Body.String(), "请介绍你的背景和最近的经历") {
		t.Fatalf("activation = %d, body=%s", activated.Code, activated.Body.String())
	}
	var activeEnvelope struct{ Session practice.Session }
	if err := json.NewDecoder(activated.Body).Decode(&activeEnvelope); err != nil {
		t.Fatal(err)
	}
	if current := request(http.MethodGet, "/v1/practice-sessions/"+sessionID+"/current-question", token1, ""); current.Code != http.StatusOK || !strings.Contains(current.Body.String(), "请介绍你的背景和最近的经历") {
		t.Fatalf("current question = %d, body=%s", current.Code, current.Body.String())
	}
	if repeated := request(http.MethodPost, "/v1/practice-sessions/"+sessionID+"/activation", token1, ""); repeated.Code != http.StatusConflict {
		t.Fatalf("repeated activation = %d, want %d", repeated.Code, http.StatusConflict)
	}
	answer := `{"question_id":"` + activeEnvelope.Session.CurrentQuestion.ID + `","content":"My recent experience is building this product."}`
	if crossUser := request(http.MethodPost, "/v1/practice-sessions/"+sessionID+"/text-answers", token2, answer); crossUser.Code != http.StatusNotFound {
		t.Fatalf("cross-user text answer = %d, body=%s", crossUser.Code, crossUser.Body.String())
	}
	if empty := request(http.MethodPost, "/v1/practice-sessions/"+sessionID+"/text-answers", token1, `{"question_id":"`+activeEnvelope.Session.CurrentQuestion.ID+`","content":"  "}`); empty.Code != http.StatusBadRequest {
		t.Fatalf("empty text answer = %d, body=%s", empty.Code, empty.Body.String())
	}
	if submitted := request(http.MethodPost, "/v1/practice-sessions/"+sessionID+"/text-answers", token1, answer); submitted.Code != http.StatusOK || !strings.Contains(submitted.Body.String(), `"turn"`) {
		t.Fatalf("text answer = %d, body=%s", submitted.Code, submitted.Body.String())
	}
	if repeatedAnswer := request(http.MethodPost, "/v1/practice-sessions/"+sessionID+"/text-answers", token1, answer); repeatedAnswer.Code != http.StatusConflict {
		t.Fatalf("repeated text answer = %d, body=%s", repeatedAnswer.Code, repeatedAnswer.Body.String())
	}
	if completed := request(http.MethodPost, "/v1/practice-sessions/"+sessionID+"/complete", token1, ""); completed.Code != http.StatusOK || !strings.Contains(completed.Body.String(), `"COMPLETED"`) {
		t.Fatalf("completion = %d, body=%s", completed.Code, completed.Body.String())
	}
	if current := request(http.MethodGet, "/v1/practice-sessions/"+sessionID+"/current-question", token1, ""); current.Code != http.StatusConflict {
		t.Fatalf("completed current question = %d, want %d", current.Code, http.StatusConflict)
	}
}
