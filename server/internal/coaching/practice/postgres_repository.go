package practice

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
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
