package practice

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/gangcaiyoule/ai-speak/server/internal/coaching/scene"
)

func TestSessionServiceCreatesOrderedQuestionSnapshotsAndScopesByActor(t *testing.T) {
	service, _, plan := newSessionTestService(t)

	session, err := service.CreateSession(context.Background(), plan.ActorID, CreateSessionInput{PlanID: plan.ID})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if session.ActorID != plan.ActorID || session.PlanID != plan.ID || session.SceneID != plan.SceneID || session.SceneVersion != plan.SceneVersion || session.Status != SessionStatusDraft {
		t.Fatalf("unexpected session = %#v", session)
	}
	if len(session.Questions) != 3 || session.CurrentQuestion != nil {
		t.Fatalf("created session questions/current = %#v/%#v", session.Questions, session.CurrentQuestion)
	}
	want := []string{"第一道问题", "第二道问题", "第三道问题"}
	for i, question := range session.Questions {
		if question.Position != i+1 || question.Content != want[i] || question.SessionID != session.ID {
			t.Fatalf("question[%d] = %#v, want position/content/session snapshot", i, question)
		}
	}

	if _, err := service.GetSession(context.Background(), "another-user", session.ID); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("cross-user GetSession() error = %v, want ErrSessionNotFound", err)
	}
	if _, err := service.ActivateSession(context.Background(), "another-user", session.ID); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("cross-user ActivateSession() error = %v, want ErrSessionNotFound", err)
	}
}

func TestSessionServiceEnforcesDraftActiveCompletedLifecycleAndCurrentQuestion(t *testing.T) {
	service, _, plan := newSessionTestService(t)
	session, err := service.CreateSession(context.Background(), plan.ActorID, CreateSessionInput{PlanID: plan.ID})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := service.CompleteSession(context.Background(), plan.ActorID, session.ID); !errors.Is(err, ErrInvalidSessionTransition) {
		t.Fatalf("complete draft error = %v, want ErrInvalidSessionTransition", err)
	}
	active, err := service.ActivateSession(context.Background(), plan.ActorID, session.ID)
	if err != nil {
		t.Fatalf("ActivateSession() error = %v", err)
	}
	if active.Status != SessionStatusActive || active.CurrentQuestion == nil || active.CurrentQuestion.Content != "第一道问题" {
		t.Fatalf("activated session = %#v", active)
	}
	current, err := service.GetCurrentQuestion(context.Background(), plan.ActorID, session.ID)
	if err != nil || current.Position != 1 || current.Content != "第一道问题" {
		t.Fatalf("GetCurrentQuestion() = %#v, %v", current, err)
	}
	if _, err := service.ActivateSession(context.Background(), plan.ActorID, session.ID); !errors.Is(err, ErrInvalidSessionTransition) {
		t.Fatalf("activate active error = %v, want ErrInvalidSessionTransition", err)
	}

	if _, err := service.CompleteSession(context.Background(), plan.ActorID, session.ID); !errors.Is(err, ErrSessionHasPendingQuestions) {
		t.Fatalf("complete active session with pending questions error = %v, want ErrSessionHasPendingQuestions", err)
	}
	for _, question := range active.Questions {
		if _, _, err := service.SubmitAnswer(context.Background(), plan.ActorID, session.ID, SubmitAnswerInput{QuestionID: question.ID, Content: "answer"}); err != nil {
			t.Fatalf("submit answer %q: %v", question.ID, err)
		}
	}
	completed, err := service.CompleteSession(context.Background(), plan.ActorID, session.ID)
	if err != nil {
		t.Fatalf("CompleteSession() after all answers error = %v", err)
	}
	if completed.Status != SessionStatusCompleted || completed.CurrentQuestion != nil {
		t.Fatalf("completed session = %#v", completed)
	}
	if _, err := service.GetCurrentQuestion(context.Background(), plan.ActorID, session.ID); !errors.Is(err, ErrSessionNotActive) {
		t.Fatalf("completed current question error = %v, want ErrSessionNotActive", err)
	}
	if _, err := service.CompleteSession(context.Background(), plan.ActorID, session.ID); !errors.Is(err, ErrInvalidSessionTransition) {
		t.Fatalf("complete completed error = %v, want ErrInvalidSessionTransition", err)
	}
}

func TestSessionServiceRejectsArchivedPlan(t *testing.T) {
	service, planRepo, plan := newSessionTestService(t)
	if _, err := planRepo.ArchivePlan(context.Background(), plan.ActorID, plan.ID, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateSession(context.Background(), plan.ActorID, CreateSessionInput{PlanID: plan.ID}); !errors.Is(err, ErrPlanArchived) {
		t.Fatalf("CreateSession() archived plan error = %v, want ErrPlanArchived", err)
	}
}

func TestSessionServiceSubmitsOnlyCurrentQuestionAndAdvances(t *testing.T) {
	service, _, plan := newSessionTestService(t)
	session, err := service.CreateSession(context.Background(), plan.ActorID, CreateSessionInput{PlanID: plan.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = service.SubmitAnswer(context.Background(), plan.ActorID, session.ID, SubmitAnswerInput{QuestionID: session.Questions[0].ID, Content: "answer"}); !errors.Is(err, ErrSessionNotActive) {
		t.Fatalf("submit draft error = %v, want ErrSessionNotActive", err)
	}
	if _, err = service.ActivateSession(context.Background(), plan.ActorID, session.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err = service.SubmitAnswer(context.Background(), plan.ActorID, session.ID, SubmitAnswerInput{QuestionID: session.Questions[1].ID, Content: "answer"}); !errors.Is(err, ErrQuestionNotFound) {
		t.Fatalf("submit non-current error = %v, want ErrQuestionNotFound", err)
	}
	turn, updated, err := service.SubmitAnswer(context.Background(), plan.ActorID, session.ID, SubmitAnswerInput{QuestionID: session.Questions[0].ID, Content: "  clear answer  "})
	if err != nil || turn.Content != "clear answer" || updated.CurrentQuestion == nil || updated.CurrentQuestion.Position != 2 {
		t.Fatalf("submit current = %#v, %#v, %v", turn, updated, err)
	}
	if _, _, err = service.SubmitAnswer(context.Background(), plan.ActorID, session.ID, SubmitAnswerInput{QuestionID: session.Questions[0].ID, Content: "again"}); !errors.Is(err, ErrQuestionAlreadyAnswered) {
		t.Fatalf("repeat answer error = %v, want ErrQuestionAlreadyAnswered", err)
	}
}

type sessionTestCatalog struct {
	detail scene.Scene
}

func (c *sessionTestCatalog) ListScenes(context.Context) ([]scene.Scene, error) {
	return []scene.Scene{c.detail}, nil
}

func (c *sessionTestCatalog) GetScene(context.Context, string) (scene.Scene, error) {
	return c.detail, nil
}

func (c *sessionTestCatalog) ListRoles(context.Context, string) ([]scene.RoleDefinition, error) {
	return c.detail.Roles, nil
}

func newSessionTestService(t *testing.T) (SessionService, *MemoryPlanRepository, Plan) {
	t.Helper()
	planRepo := NewMemoryPlanRepository()
	plan := Plan{ID: uuid.NewString(), ActorID: "user-1", SceneID: "test-scene", SceneVersion: 7, RoleID: "coach", PracticeOptionID: "full", Objective: "practice", Status: PlanStatusDraft, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if _, err := planRepo.CreatePlan(context.Background(), plan); err != nil {
		t.Fatal(err)
	}
	catalog := &sessionTestCatalog{detail: scene.Scene{ID: plan.SceneID, SceneVersion: plan.SceneVersion, Status: "active", Prompt: scene.ScenePrompt{TurnBlueprints: []string{"第一道问题", "第二道问题", "第三道问题"}}}}
	planService := NewPlanService(planRepo, catalog)
	return NewSessionService(NewMemorySessionRepository(), planService, catalog), planRepo, plan
}
