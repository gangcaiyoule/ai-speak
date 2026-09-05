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
	for i := 0; i < len(active.Questions); i++ {
		current, err := service.GetCurrentQuestion(context.Background(), plan.ActorID, session.ID)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := service.SubmitTextAnswer(context.Background(), plan.ActorID, session.ID, SubmitTextAnswerInput{QuestionID: current.ID, Content: " answer "}); err != nil {
			t.Fatalf("SubmitTextAnswer() error = %v", err)
		}
	}

	completed, err := service.CompleteSession(context.Background(), plan.ActorID, session.ID)
	if err != nil {
		t.Fatalf("CompleteSession() error = %v", err)
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

func TestSessionServiceFindsLatestResumableSessionForActor(t *testing.T) {
	service, _, plan := newSessionTestService(t)
	clock := time.Now().UTC()
	service.(*sessionService).now = func() time.Time {
		clock = clock.Add(time.Second)
		return clock
	}
	first, err := service.CreateSession(context.Background(), plan.ActorID, CreateSessionInput{PlanID: plan.ID})
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.CreateSession(context.Background(), plan.ActorID, CreateSessionInput{PlanID: plan.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ActivateSession(context.Background(), plan.ActorID, second.ID); err != nil {
		t.Fatal(err)
	}
	got, err := service.GetLatestResumableSession(context.Background(), plan.ActorID)
	if err != nil || got.ID != second.ID || got.Status != SessionStatusActive || got.CurrentQuestion == nil {
		t.Fatalf("GetLatestResumableSession() = %#v, %v", got, err)
	}
	if _, err := service.GetLatestResumableSession(context.Background(), "another-user"); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("cross-user resumable session error = %v, want ErrSessionNotFound", err)
	}
	if first.ID == second.ID {
		t.Fatal("test requires distinct sessions")
	}
}

func TestSessionServiceSubmitsAnswersAdvancesQuestionsAndRejectsInvalidOperations(t *testing.T) {
	service, _, plan := newSessionTestService(t)
	session, err := service.CreateSession(context.Background(), plan.ActorID, CreateSessionInput{PlanID: plan.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SubmitTextAnswer(context.Background(), plan.ActorID, session.ID, SubmitTextAnswerInput{QuestionID: session.Questions[0].ID, Content: "answer"}); !errors.Is(err, ErrSessionNotActive) {
		t.Fatalf("draft answer error = %v", err)
	}
	active, err := service.ActivateSession(context.Background(), plan.ActorID, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SubmitTextAnswer(context.Background(), plan.ActorID, session.ID, SubmitTextAnswerInput{QuestionID: active.Questions[1].ID, Content: "answer"}); !errors.Is(err, ErrQuestionNotCurrent) {
		t.Fatalf("non-current answer error = %v", err)
	}
	if _, err := service.SubmitTextAnswer(context.Background(), "another-user", session.ID, SubmitTextAnswerInput{QuestionID: active.Questions[0].ID, Content: "answer"}); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("cross-user answer error = %v", err)
	}
	if _, err := service.SubmitTextAnswer(context.Background(), plan.ActorID, session.ID, SubmitTextAnswerInput{QuestionID: active.Questions[0].ID, Content: "   "}); !errors.Is(err, ErrInvalidAnswer) {
		t.Fatalf("blank answer error = %v", err)
	}
	for i := 0; i < len(active.Questions); i++ {
		current, err := service.GetCurrentQuestion(context.Background(), plan.ActorID, session.ID)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := service.SubmitTextAnswer(context.Background(), plan.ActorID, session.ID, SubmitTextAnswerInput{QuestionID: current.ID, Content: "answer"}); err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			if _, err := service.SubmitTextAnswer(context.Background(), plan.ActorID, session.ID, SubmitTextAnswerInput{QuestionID: current.ID, Content: "again"}); !errors.Is(err, ErrAnswerAlreadySubmitted) {
				t.Fatalf("duplicate answer error = %v", err)
			}
		}
	}
	if _, err := service.GetCurrentQuestion(context.Background(), plan.ActorID, session.ID); !errors.Is(err, ErrNoCurrentQuestion) {
		t.Fatalf("after final answer current error = %v", err)
	}
	completed, err := service.CompleteSession(context.Background(), plan.ActorID, session.ID)
	if err != nil || completed.Status != SessionStatusCompleted || completed.CurrentQuestionID != nil {
		t.Fatalf("completed session = %#v, %v", completed, err)
	}
	if _, err := service.SubmitTextAnswer(context.Background(), plan.ActorID, session.ID, SubmitTextAnswerInput{QuestionID: active.Questions[2].ID, Content: "again"}); !errors.Is(err, ErrSessionNotActive) {
		t.Fatalf("completed answer error = %v", err)
	}
	if _, err := service.CompleteSession(context.Background(), plan.ActorID, session.ID); !errors.Is(err, ErrInvalidSessionTransition) {
		t.Fatalf("duplicate completion error = %v", err)
	}
}

func TestSessionServiceRequiresAllQuestionsBeforeCompletion(t *testing.T) {
	service, _, plan := newSessionTestService(t)
	session, err := service.CreateSession(context.Background(), plan.ActorID, CreateSessionInput{PlanID: plan.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ActivateSession(context.Background(), plan.ActorID, session.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CompleteSession(context.Background(), plan.ActorID, session.ID); !errors.Is(err, ErrSessionHasPendingQuestion) {
		t.Fatalf("pending completion error = %v", err)
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
