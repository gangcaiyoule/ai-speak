package practice

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type PostgresPlanRepository struct{ db *sql.DB }
type PostgresRepository = PostgresPlanRepository

func NewPostgresPlanRepository(db *sql.DB) *PostgresPlanRepository {
	return &PostgresPlanRepository{db: db}
}
func NewPostgresRepository(db *sql.DB) *PostgresPlanRepository { return NewPostgresPlanRepository(db) }

func (r *PostgresPlanRepository) CreatePlan(ctx context.Context, plan Plan) (Plan, error) {
	if r == nil || r.db == nil {
		return Plan{}, ErrInvalidPlan
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO practice_plans (id, actor_id, scene_id, scene_version, role_id, practice_option_id, objective, status, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, plan.ID, plan.ActorID, plan.SceneID, plan.SceneVersion, plan.RoleID, plan.PracticeOptionID, plan.Objective, plan.Status, plan.CreatedAt, plan.UpdatedAt)
	if err != nil {
		return Plan{}, fmt.Errorf("insert practice plan: %w", err)
	}
	return plan, nil
}

func (r *PostgresPlanRepository) ListPlans(ctx context.Context, actorID string) ([]Plan, error) {
	if r == nil || r.db == nil {
		return nil, ErrInvalidPlan
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id,actor_id,scene_id,scene_version,role_id,practice_option_id,objective,status,created_at,updated_at FROM practice_plans WHERE actor_id=$1 ORDER BY created_at DESC`, actorID)
	if err != nil {
		return nil, fmt.Errorf("list practice plans: %w", err)
	}
	defer rows.Close()
	plans := make([]Plan, 0)
	for rows.Next() {
		var plan Plan
		if err := rows.Scan(&plan.ID, &plan.ActorID, &plan.SceneID, &plan.SceneVersion, &plan.RoleID, &plan.PracticeOptionID, &plan.Objective, &plan.Status, &plan.CreatedAt, &plan.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan practice plan: %w", err)
		}
		plans = append(plans, plan)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read practice plans: %w", err)
	}
	return plans, nil
}

func (r *PostgresPlanRepository) FindPlan(ctx context.Context, actorID, id string) (Plan, error) {
	if r == nil || r.db == nil {
		return Plan{}, ErrPlanNotFound
	}
	var plan Plan
	err := r.db.QueryRowContext(ctx, `SELECT id,actor_id,scene_id,scene_version,role_id,practice_option_id,objective,status,created_at,updated_at FROM practice_plans WHERE id=$1 AND actor_id=$2`, id, actorID).Scan(&plan.ID, &plan.ActorID, &plan.SceneID, &plan.SceneVersion, &plan.RoleID, &plan.PracticeOptionID, &plan.Objective, &plan.Status, &plan.CreatedAt, &plan.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Plan{}, ErrPlanNotFound
	}
	if err != nil {
		return Plan{}, fmt.Errorf("find practice plan: %w", err)
	}
	return plan, nil
}

func (r *PostgresPlanRepository) ArchivePlan(ctx context.Context, actorID, id string, now time.Time) (Plan, error) {
	if r == nil || r.db == nil {
		return Plan{}, ErrPlanNotFound
	}
	var plan Plan
	err := r.db.QueryRowContext(ctx, `UPDATE practice_plans SET status='ARCHIVED', updated_at=$3 WHERE id=$1 AND actor_id=$2 AND status <> 'ARCHIVED' RETURNING id,actor_id,scene_id,scene_version,role_id,practice_option_id,objective,status,created_at,updated_at`, id, actorID, now).Scan(&plan.ID, &plan.ActorID, &plan.SceneID, &plan.SceneVersion, &plan.RoleID, &plan.PracticeOptionID, &plan.Objective, &plan.Status, &plan.CreatedAt, &plan.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		current, findErr := r.FindPlan(ctx, actorID, id)
		if errors.Is(findErr, ErrPlanNotFound) {
			return Plan{}, ErrPlanNotFound
		}
		if findErr != nil {
			return Plan{}, findErr
		}
		if current.Status == PlanStatusArchived {
			return Plan{}, ErrPlanArchived
		}
		return Plan{}, ErrPlanNotFound
	}
	if err != nil {
		return Plan{}, fmt.Errorf("archive practice plan: %w", err)
	}
	return plan, nil
}

// PostgresSessionRepository 将会话和问题快照持久化到 PostgreSQL。
type PostgresSessionRepository struct{ db *sql.DB }

func NewPostgresSessionRepository(db *sql.DB) *PostgresSessionRepository {
	return &PostgresSessionRepository{db: db}
}

type postgresSessionQueryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

const sessionColumns = `id,actor_id,plan_id,scene_id,scene_version,status,created_at,updated_at`

func (r *PostgresSessionRepository) CreateSession(ctx context.Context, session Session, questions []Question) (Session, error) {
	if err := contextError(ctx); err != nil {
		return Session{}, err
	}
	if r == nil || r.db == nil {
		return Session{}, ErrInvalidSession
	}
	if err := validateSessionSnapshot(session, questions); err != nil {
		return Session{}, err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Session{}, fmt.Errorf("begin practice session transaction: %w", err)
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO practice_sessions (id,actor_id,plan_id,scene_id,scene_version,status,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, session.ID, session.ActorID, session.PlanID, session.SceneID, session.SceneVersion, session.Status, session.CreatedAt, session.UpdatedAt)
	if err != nil {
		return Session{}, fmt.Errorf("insert practice session: %w", err)
	}
	for _, question := range questions {
		_, err = tx.ExecContext(ctx, `INSERT INTO practice_questions (id,session_id,position,content) VALUES ($1,$2,$3,$4)`, question.ID, question.SessionID, question.Position, question.Content)
		if err != nil {
			return Session{}, fmt.Errorf("insert practice question: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return Session{}, fmt.Errorf("commit practice session: %w", err)
	}
	return withQuestions(session, questions), nil
}

func (r *PostgresSessionRepository) FindSession(ctx context.Context, actorID, sessionID string) (Session, error) {
	if err := contextError(ctx); err != nil {
		return Session{}, err
	}
	if r == nil || r.db == nil {
		return Session{}, ErrSessionNotFound
	}
	session, err := findPostgresSession(ctx, r.db, actorID, sessionID)
	if err != nil {
		return Session{}, err
	}
	questions, err := listPostgresQuestions(ctx, r.db, actorID, sessionID)
	if err != nil {
		return Session{}, err
	}
	return withQuestions(session, questions), nil
}

func (r *PostgresSessionRepository) ActivateSession(ctx context.Context, actorID, sessionID string, now time.Time) (Session, error) {
	return r.transitionSession(ctx, actorID, sessionID, SessionStatusDraft, SessionStatusActive, now)
}

func (r *PostgresSessionRepository) SubmitAnswer(ctx context.Context, actorID, sessionID string, in SubmitAnswerInput, now time.Time) (Turn, Session, error) {
	if err := contextError(ctx); err != nil {
		return Turn{}, Session{}, err
	}
	if r == nil || r.db == nil {
		return Turn{}, Session{}, ErrSessionNotFound
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Turn{}, Session{}, fmt.Errorf("begin practice answer transaction: %w", err)
	}
	defer tx.Rollback()
	var status string
	err = tx.QueryRowContext(ctx, `SELECT status FROM practice_sessions WHERE id=$1 AND actor_id=$2 FOR UPDATE`, sessionID, actorID).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return Turn{}, Session{}, ErrSessionNotFound
	}
	if err != nil {
		return Turn{}, Session{}, fmt.Errorf("lock practice session: %w", err)
	}
	if status != SessionStatusActive {
		return Turn{}, Session{}, ErrSessionNotActive
	}
	var currentID string
	err = tx.QueryRowContext(ctx, `SELECT q.id FROM practice_questions q WHERE q.session_id=$1 AND NOT EXISTS (SELECT 1 FROM practice_turns t WHERE t.question_id=q.id) ORDER BY q.position LIMIT 1`, sessionID).Scan(&currentID)
	if errors.Is(err, sql.ErrNoRows) {
		return Turn{}, Session{}, ErrNoCurrentQuestion
	}
	if err != nil {
		return Turn{}, Session{}, fmt.Errorf("find current practice question: %w", err)
	}
	if currentID != in.QuestionID {
		return Turn{}, Session{}, ErrQuestionNotFound
	}
	turn := Turn{ID: uuid.NewString(), SessionID: sessionID, QuestionID: in.QuestionID, Content: in.Content, CreatedAt: now}
	if _, err = tx.ExecContext(ctx, `INSERT INTO practice_turns (id,session_id,question_id,content,created_at) VALUES ($1,$2,$3,$4,$5)`, turn.ID, turn.SessionID, turn.QuestionID, turn.Content, turn.CreatedAt); err != nil {
		return Turn{}, Session{}, fmt.Errorf("insert practice answer: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return Turn{}, Session{}, fmt.Errorf("commit practice answer: %w", err)
	}
	session, err := r.FindSession(ctx, actorID, sessionID)
	if err != nil {
		return Turn{}, Session{}, err
	}
	for index := range session.Questions {
		var answered bool
		if err := r.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM practice_turns WHERE question_id=$1)`, session.Questions[index].ID).Scan(&answered); err != nil {
			return Turn{}, Session{}, fmt.Errorf("read practice answer state: %w", err)
		}
		if answered {
			session.Questions[index].Status = "ANSWERED"
		}
	}
	return turn, withCurrentQuestion(session), nil
}

func (r *PostgresSessionRepository) CompleteSession(ctx context.Context, actorID, sessionID string, now time.Time) (Session, error) {
	return r.transitionSession(ctx, actorID, sessionID, SessionStatusActive, SessionStatusCompleted, now)
}

func (r *PostgresSessionRepository) transitionSession(ctx context.Context, actorID, sessionID, from, to string, now time.Time) (Session, error) {
	if err := contextError(ctx); err != nil {
		return Session{}, err
	}
	if r == nil || r.db == nil {
		return Session{}, ErrSessionNotFound
	}
	query := `UPDATE practice_sessions SET status=$3,updated_at=$4 WHERE id=$1 AND actor_id=$2 AND status=$5 RETURNING ` + sessionColumns
	session, err := scanPostgresSession(r.db.QueryRowContext(ctx, query, sessionID, actorID, to, now, from))
	if errors.Is(err, sql.ErrNoRows) {
		current, findErr := findPostgresSession(ctx, r.db, actorID, sessionID)
		if errors.Is(findErr, ErrSessionNotFound) {
			return Session{}, ErrSessionNotFound
		}
		if findErr != nil {
			return Session{}, findErr
		}
		if current.Status != from {
			return Session{}, ErrInvalidSessionTransition
		}
		return Session{}, ErrInvalidSessionTransition
	}
	if err != nil {
		return Session{}, fmt.Errorf("transition practice session: %w", err)
	}
	questions, err := listPostgresQuestions(ctx, r.db, actorID, sessionID)
	if err != nil {
		return Session{}, err
	}
	return withQuestions(session, questions), nil
}

func findPostgresSession(ctx context.Context, q postgresSessionQueryer, actorID, sessionID string) (Session, error) {
	query := `SELECT ` + sessionColumns + ` FROM practice_sessions WHERE id=$1 AND actor_id=$2`
	session, err := scanPostgresSession(q.QueryRowContext(ctx, query, sessionID, actorID))
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, ErrSessionNotFound
	}
	if err != nil {
		return Session{}, fmt.Errorf("find practice session: %w", err)
	}
	return session, nil
}

func scanPostgresSession(row *sql.Row) (Session, error) {
	var session Session
	err := row.Scan(&session.ID, &session.ActorID, &session.PlanID, &session.SceneID, &session.SceneVersion, &session.Status, &session.CreatedAt, &session.UpdatedAt)
	return session, err
}

func listPostgresQuestions(ctx context.Context, q postgresSessionQueryer, actorID, sessionID string) ([]Question, error) {
	rows, err := q.QueryContext(ctx, `SELECT q.id,q.session_id,q.position,q.content FROM practice_questions q JOIN practice_sessions s ON s.id=q.session_id WHERE q.session_id=$1 AND s.actor_id=$2 ORDER BY q.position ASC`, sessionID, actorID)
	if err != nil {
		return nil, fmt.Errorf("list practice questions: %w", err)
	}
	defer rows.Close()
	questions := make([]Question, 0)
	for rows.Next() {
		var question Question
		if err := rows.Scan(&question.ID, &question.SessionID, &question.Position, &question.Content); err != nil {
			return nil, fmt.Errorf("scan practice question: %w", err)
		}
		questions = append(questions, question)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read practice questions: %w", err)
	}
	return questions, nil
}
