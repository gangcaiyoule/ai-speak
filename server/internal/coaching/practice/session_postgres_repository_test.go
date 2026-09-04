package practice

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

func TestPostgresSessionRepositoryCreatesSessionAndQuestionsAtomically(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	now := time.Now().UTC().Truncate(time.Microsecond)
	session := Session{ID: uuid.NewString(), ActorID: "user-1", PlanID: uuid.NewString(), SceneID: "self-introduction", SceneVersion: 1, Status: SessionStatusDraft, CreatedAt: now, UpdatedAt: now}
	questions := []Question{{ID: uuid.NewString(), SessionID: session.ID, Position: 1, Content: "first"}, {ID: uuid.NewString(), SessionID: session.ID, Position: 2, Content: "second"}}
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO practice_sessions (id,actor_id,plan_id,scene_id,scene_version,status,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`)).
		WithArgs(session.ID, session.ActorID, session.PlanID, session.SceneID, session.SceneVersion, session.Status, session.CreatedAt, session.UpdatedAt).
		WillReturnResult(sqlmock.NewResult(0, 1))
	for _, question := range questions {
		mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO practice_questions (id,session_id,position,content) VALUES ($1,$2,$3,$4)`)).
			WithArgs(question.ID, question.SessionID, question.Position, question.Content).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}
	mock.ExpectCommit()

	got, err := NewPostgresSessionRepository(db).CreateSession(context.Background(), session, questions)
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if got.Status != SessionStatusDraft || len(got.Questions) != 2 || got.Questions[0].Content != "first" || got.Questions[1].Position != 2 {
		t.Fatalf("CreateSession() = %#v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresSessionRepositoryFindScopesByActorAndLoadsQuestionOrder(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	sessionID := uuid.NewString()
	planID := uuid.NewString()
	createdAt := time.Now().UTC().Truncate(time.Microsecond)
	updatedAt := createdAt.Add(time.Minute)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT `+sessionColumns+` FROM practice_sessions WHERE id=$1 AND actor_id=$2`)).
		WithArgs(sessionID, "user-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "actor_id", "plan_id", "scene_id", "scene_version", "status", "created_at", "updated_at"}).
			AddRow(sessionID, "user-1", planID, "scene", 3, SessionStatusActive, createdAt, updatedAt))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT q.id,q.session_id,q.position,q.content FROM practice_questions q JOIN practice_sessions s ON s.id=q.session_id WHERE q.session_id=$1 AND s.actor_id=$2 ORDER BY q.position ASC`)).
		WithArgs(sessionID, "user-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "session_id", "position", "content"}).
			AddRow(uuid.NewString(), sessionID, 1, "first").
			AddRow(uuid.NewString(), sessionID, 2, "second"))

	got, err := NewPostgresSessionRepository(db).FindSession(context.Background(), "user-1", sessionID)
	if err != nil {
		t.Fatalf("FindSession() error = %v", err)
	}
	if got.ActorID != "user-1" || got.Status != SessionStatusActive || len(got.Questions) != 2 || got.Questions[0].Position != 1 || got.Questions[1].Content != "second" {
		t.Fatalf("FindSession() = %#v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT `+sessionColumns+` FROM practice_sessions WHERE id=$1 AND actor_id=$2`)).
		WithArgs(sessionID, "user-2").
		WillReturnRows(sqlmock.NewRows([]string{"id", "actor_id", "plan_id", "scene_id", "scene_version", "status", "created_at", "updated_at"}))
	if _, err := NewPostgresSessionRepository(db).FindSession(context.Background(), "user-2", sessionID); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("cross-user FindSession() error = %v, want ErrSessionNotFound", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresSessionRepositoryRejectsActivationWhenNotDraft(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	sessionID := uuid.NewString()
	createdAt := time.Now().UTC().Truncate(time.Microsecond)
	mock.ExpectQuery(regexp.QuoteMeta(`UPDATE practice_sessions SET status=$3,updated_at=$4 WHERE id=$1 AND actor_id=$2 AND status=$5 RETURNING `+sessionColumns)).
		WithArgs(sessionID, "user-1", SessionStatusActive, sqlmock.AnyArg(), SessionStatusDraft).
		WillReturnRows(sqlmock.NewRows([]string{"id", "actor_id", "plan_id", "scene_id", "scene_version", "status", "created_at", "updated_at"}))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT `+sessionColumns+` FROM practice_sessions WHERE id=$1 AND actor_id=$2`)).
		WithArgs(sessionID, "user-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "actor_id", "plan_id", "scene_id", "scene_version", "status", "created_at", "updated_at"}).
			AddRow(sessionID, "user-1", uuid.NewString(), "scene", 1, SessionStatusActive, createdAt, createdAt))

	if _, err := NewPostgresSessionRepository(db).ActivateSession(context.Background(), "user-1", sessionID, createdAt.Add(time.Minute)); !errors.Is(err, ErrInvalidSessionTransition) {
		t.Fatalf("ActivateSession() error = %v, want ErrInvalidSessionTransition", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresSessionRepositoryTransitionsActiveSessionToCompleted(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	sessionID := uuid.NewString()
	planID := uuid.NewString()
	questionID := uuid.NewString()
	now := time.Now().UTC().Truncate(time.Microsecond)
	mock.ExpectQuery(regexp.QuoteMeta(`UPDATE practice_sessions SET status=$3,updated_at=$4 WHERE id=$1 AND actor_id=$2 AND status=$5 RETURNING `+sessionColumns)).
		WithArgs(sessionID, "user-1", SessionStatusCompleted, sqlmock.AnyArg(), SessionStatusActive).
		WillReturnRows(sqlmock.NewRows([]string{"id", "actor_id", "plan_id", "scene_id", "scene_version", "status", "created_at", "updated_at"}).
			AddRow(sessionID, "user-1", planID, "scene", 1, SessionStatusCompleted, now, now.Add(time.Minute)))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT q.id,q.session_id,q.position,q.content FROM practice_questions q JOIN practice_sessions s ON s.id=q.session_id WHERE q.session_id=$1 AND s.actor_id=$2 ORDER BY q.position ASC`)).
		WithArgs(sessionID, "user-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "session_id", "position", "content"}).AddRow(questionID, sessionID, 1, "first"))

	got, err := NewPostgresSessionRepository(db).CompleteSession(context.Background(), "user-1", sessionID, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("CompleteSession() error = %v", err)
	}
	if got.Status != SessionStatusCompleted || len(got.Questions) != 1 || got.Questions[0].ID != questionID {
		t.Fatalf("CompleteSession() = %#v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
