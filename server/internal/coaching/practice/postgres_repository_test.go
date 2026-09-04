package practice

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestPostgresRepositoryListPlansReturnsEmptySliceWhenActorHasNoPlans(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT id,actor_id,scene_id,scene_version,role_id,practice_option_id,objective,status,created_at,updated_at FROM practice_plans WHERE actor_id=\\$1 ORDER BY created_at DESC").
		WithArgs("user-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "actor_id", "scene_id", "scene_version", "role_id", "practice_option_id", "objective", "status", "created_at", "updated_at"}))

	plans, err := NewPostgresPlanRepository(db).ListPlans(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListPlans() error = %v", err)
	}
	if plans == nil || len(plans) != 0 {
		t.Fatalf("ListPlans() = %#v, want non-nil empty slice", plans)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
