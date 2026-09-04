package practice

import (
	"context"
	"errors"
	"testing"

	"github.com/gangcaiyoule/ai-speak/server/internal/coaching/scene"
)

func TestServiceCreatesAndReadsOwnPlan(t *testing.T) {
	service := NewPlanService(NewMemoryPlanRepository(), scene.NewCatalog())
	created, err := service.CreatePlan(context.Background(), "user-1", CreatePlanInput{
		SceneID:          "self-introduction",
		SceneVersion:     1,
		RoleID:           "recruiter",
		PracticeOptionID: "self-introduction-focus",
		Objective:        "Improve concise answers",
	})
	if err != nil {
		t.Fatalf("CreatePlan() error = %v", err)
	}
	if created.ActorID != "user-1" || created.Status != PlanStatusActive || created.Objective != "Improve concise answers" {
		t.Fatalf("unexpected plan = %#v", created)
	}
	got, err := service.GetPlan(context.Background(), "user-1", created.ID)
	if err != nil || got.ID != created.ID {
		t.Fatalf("GetPlan() = %#v, %v", got, err)
	}
	plans, err := service.ListPlans(context.Background(), "user-1")
	if err != nil || len(plans) != 1 || plans[0].ID != created.ID {
		t.Fatalf("ListPlans() = %#v, %v", plans, err)
	}
}

func TestServiceRejectsInvalidPlanAndUnknownSceneConfiguration(t *testing.T) {
	service := NewPlanService(NewMemoryPlanRepository(), scene.NewCatalog())
	cases := []CreatePlanInput{
		{SceneID: "", SceneVersion: 1, RoleID: "recruiter", PracticeOptionID: "self-introduction-focus", Objective: "goal"},
		{SceneID: "self-introduction", SceneVersion: 2, RoleID: "recruiter", PracticeOptionID: "self-introduction-focus", Objective: "goal"},
		{SceneID: "self-introduction", SceneVersion: 1, RoleID: "missing", PracticeOptionID: "self-introduction-focus", Objective: "goal"},
		{SceneID: "self-introduction", SceneVersion: 1, RoleID: "recruiter", PracticeOptionID: "missing", Objective: "goal"},
		{SceneID: "self-introduction", SceneVersion: 1, RoleID: "recruiter", PracticeOptionID: "self-introduction-focus", Objective: ""},
	}
	for _, input := range cases {
		if _, err := service.CreatePlan(context.Background(), "user-1", input); !errors.Is(err, ErrInvalidPlan) {
			t.Errorf("CreatePlan(%#v) error = %v, want ErrInvalidPlan", input, err)
		}
	}
}

func TestServiceDoesNotExposeAnotherUsersPlan(t *testing.T) {
	service := NewPlanService(NewMemoryPlanRepository(), scene.NewCatalog())
	plan, err := service.CreatePlan(context.Background(), "user-1", validPlanInput())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.GetPlan(context.Background(), "user-2", plan.ID); !errors.Is(err, ErrPlanNotFound) {
		t.Fatalf("GetPlan() error = %v, want ErrPlanNotFound", err)
	}
	plans, err := service.ListPlans(context.Background(), "user-2")
	if err != nil || len(plans) != 0 {
		t.Fatalf("ListPlans() = %#v, %v", plans, err)
	}
}

func TestArchivedPlanCannotStartNewSession(t *testing.T) {
	service := NewPlanService(NewMemoryPlanRepository(), scene.NewCatalog())
	plan, err := service.CreatePlan(context.Background(), "user-1", validPlanInput())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.ArchivePlan(context.Background(), "user-1", plan.ID); err != nil {
		t.Fatalf("ArchivePlan() error = %v", err)
	}
	archived, err := service.GetPlan(context.Background(), "user-1", plan.ID)
	if err != nil || archived.Status != PlanStatusArchived {
		t.Fatalf("archived plan = %#v, %v", archived, err)
	}
	if err = service.EnsureCanCreateSession(context.Background(), "user-1", plan.ID); !errors.Is(err, ErrPlanArchived) {
		t.Fatalf("EnsureCanCreateSession() error = %v, want ErrPlanArchived", err)
	}
}

func validPlanInput() CreatePlanInput {
	return CreatePlanInput{
		SceneID:          "self-introduction",
		SceneVersion:     1,
		RoleID:           "recruiter",
		PracticeOptionID: "self-introduction-full",
		Objective:        "Speak with a clear structure",
	}
}
