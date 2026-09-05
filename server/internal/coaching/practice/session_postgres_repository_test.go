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

func sessionRowColumns() []string {
	return []string{"id", "actor_id", "plan_id", "scene_id", "scene_version", "status", "created_at", "updated_at", "current_question_id"}
}

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
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO practice_sessions (id,actor_id,plan_id,scene_id,scene_version,status,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`)).WithArgs(session.ID, session.ActorID, session.PlanID, session.SceneID, session.SceneVersion, session.Status, session.CreatedAt, session.UpdatedAt).WillReturnResult(sqlmock.NewResult(0, 1))
	for _, question := range questions {
		mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO practice_questions (id,session_id,position,content) VALUES ($1,$2,$3,$4)`)).WithArgs(question.ID, question.SessionID, question.Position, question.Content).WillReturnResult(sqlmock.NewResult(0, 1))
	}
	mock.ExpectCommit()
	got, err := NewPostgresSessionRepository(db).CreateSession(context.Background(), session, questions)
	if err != nil || len(got.Questions) != 2 {
		t.Fatalf("CreateSession() = %#v, %v", got, err)
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
	sessionID, planID := uuid.NewString(), uuid.NewString()
	createdAt := time.Now().UTC().Truncate(time.Microsecond)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT `+sessionColumns+` FROM practice_sessions WHERE id=$1 AND actor_id=$2`)).WithArgs(sessionID, "user-1").WillReturnRows(sqlmock.NewRows(sessionRowColumns()).AddRow(sessionID, "user-1", planID, "scene", 3, SessionStatusActive, createdAt, createdAt, nil))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT q.id,q.session_id,q.position,q.content FROM practice_questions q JOIN practice_sessions s ON s.id=q.session_id WHERE q.session_id=$1 AND s.actor_id=$2 ORDER BY q.position ASC`)).WithArgs(sessionID, "user-1").WillReturnRows(sqlmock.NewRows([]string{"id", "session_id", "position", "content"}).AddRow(uuid.NewString(), sessionID, 1, "first").AddRow(uuid.NewString(), sessionID, 2, "second"))
	got, err := NewPostgresSessionRepository(db).FindSession(context.Background(), "user-1", sessionID)
	if err != nil || len(got.Questions) != 2 {
		t.Fatalf("FindSession() = %#v, %v", got, err)
	}
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT `+sessionColumns+` FROM practice_sessions WHERE id=$1 AND actor_id=$2`)).WithArgs(sessionID, "user-2").WillReturnRows(sqlmock.NewRows(sessionRowColumns()))
	if _, err := NewPostgresSessionRepository(db).FindSession(context.Background(), "user-2", sessionID); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("cross-user error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresSessionRepositoryFindsLatestResumableSession(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	sessionID, planID, questionID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	now := time.Now().UTC().Truncate(time.Microsecond)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT `+sessionColumns+` FROM practice_sessions WHERE actor_id=$1 AND status IN ($2,$3) ORDER BY updated_at DESC,id DESC LIMIT 1`)).
		WithArgs("user-1", SessionStatusActive, SessionStatusDraft).
		WillReturnRows(sqlmock.NewRows(sessionRowColumns()).AddRow(sessionID, "user-1", planID, "scene", 1, SessionStatusActive, now, now, questionID))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT q.id,q.session_id,q.position,q.content FROM practice_questions q JOIN practice_sessions s ON s.id=q.session_id WHERE q.session_id=$1 AND s.actor_id=$2 ORDER BY q.position ASC`)).
		WithArgs(sessionID, "user-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "session_id", "position", "content"}).AddRow(questionID, sessionID, 1, "first"))
	got, err := NewPostgresSessionRepository(db).FindLatestResumableSession(context.Background(), "user-1")
	if err != nil || got.ID != sessionID || got.CurrentQuestionID == nil || *got.CurrentQuestionID != questionID {
		t.Fatalf("FindLatestResumableSession() = %#v, %v", got, err)
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
	mock.ExpectQuery(regexp.QuoteMeta(`UPDATE practice_sessions SET status=$3,updated_at=$4,current_question_id=(SELECT id FROM practice_questions WHERE session_id=$1 ORDER BY position ASC LIMIT 1) WHERE id=$1 AND actor_id=$2 AND status=$5 RETURNING `+sessionColumns)).WithArgs(sessionID, "user-1", SessionStatusActive, sqlmock.AnyArg(), SessionStatusDraft).WillReturnRows(sqlmock.NewRows(sessionRowColumns()))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT `+sessionColumns+` FROM practice_sessions WHERE id=$1 AND actor_id=$2`)).WithArgs(sessionID, "user-1").WillReturnRows(sqlmock.NewRows(sessionRowColumns()).AddRow(sessionID, "user-1", uuid.NewString(), "scene", 1, SessionStatusActive, createdAt, createdAt, uuid.NewString()))
	if _, err := NewPostgresSessionRepository(db).ActivateSession(context.Background(), "user-1", sessionID, createdAt.Add(time.Minute)); !errors.Is(err, ErrInvalidSessionTransition) {
		t.Fatalf("ActivateSession() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresSessionRepositoryTransitionsCompletedSessionWithoutPendingQuestion(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	sessionID, planID, questionID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	now := time.Now().UTC().Truncate(time.Microsecond)
	mock.ExpectQuery(regexp.QuoteMeta(`UPDATE practice_sessions SET status=$3,updated_at=$4 WHERE id=$1 AND actor_id=$2 AND status=$5 AND current_question_id IS NULL RETURNING `+sessionColumns)).WithArgs(sessionID, "user-1", SessionStatusCompleted, sqlmock.AnyArg(), SessionStatusActive).WillReturnRows(sqlmock.NewRows(sessionRowColumns()).AddRow(sessionID, "user-1", planID, "scene", 1, SessionStatusCompleted, now, now.Add(time.Minute), nil))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT q.id,q.session_id,q.position,q.content FROM practice_questions q JOIN practice_sessions s ON s.id=q.session_id WHERE q.session_id=$1 AND s.actor_id=$2 ORDER BY q.position ASC`)).WithArgs(sessionID, "user-1").WillReturnRows(sqlmock.NewRows([]string{"id", "session_id", "position", "content"}).AddRow(questionID, sessionID, 1, "first"))
	got, err := NewPostgresSessionRepository(db).CompleteSession(context.Background(), "user-1", sessionID, now.Add(time.Minute))
	if err != nil || got.Status != SessionStatusCompleted {
		t.Fatalf("CompleteSession() = %#v, %v", got, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresSessionRepositorySavesAnswerAndClearsLastQuestion(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	sessionID, questionID := uuid.NewString(), uuid.NewString()
	now := time.Now().UTC().Truncate(time.Microsecond)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT status,current_question_id FROM practice_sessions WHERE id=$1 AND actor_id=$2 FOR UPDATE`)).WithArgs(sessionID, "user-1").WillReturnRows(sqlmock.NewRows([]string{"status", "current_question_id"}).AddRow(SessionStatusActive, questionID))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT EXISTS (SELECT 1 FROM practice_turns WHERE session_id=$1 AND question_id=$2)`)).WithArgs(sessionID, questionID).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO practice_turns (id,session_id,question_id,actor_id,content,created_at) VALUES ($1,$2,$3,$4,$5,$6)`)).WithArgs(sqlmock.AnyArg(), sessionID, questionID, "user-1", "trimmed", now).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT q.id FROM practice_questions q WHERE q.session_id=$1 AND q.position > (SELECT position FROM practice_questions WHERE id=$2 AND session_id=$1) AND NOT EXISTS (SELECT 1 FROM practice_turns t WHERE t.session_id=$1 AND t.question_id=q.id) ORDER BY q.position ASC LIMIT 1`)).WithArgs(sessionID, questionID).WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE practice_sessions SET current_question_id=$2,updated_at=$3 WHERE id=$1 AND actor_id=$4`)).WithArgs(sessionID, nil, now, "user-1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	turn, err := NewPostgresSessionRepository(db).SubmitTextAnswer(context.Background(), "user-1", sessionID, SubmitTextAnswerInput{QuestionID: questionID, Content: "  trimmed  "}, now)
	if err != nil || turn.Content != "trimmed" {
		t.Fatalf("SubmitTextAnswer() = %#v, %v", turn, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
