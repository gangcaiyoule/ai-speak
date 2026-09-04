// Package practice 定义口语练习会话相关契约。
package practice

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/gangcaiyoule/ai-speak/server/internal/coaching/scene"
)

var (
	ErrInvalidPlan  = errors.New("invalid practice plan")
	ErrPlanNotFound = errors.New("practice plan not found")
	ErrPlanArchived = errors.New("practice plan is archived")
)

const (
	PlanStatusDraft    = "DRAFT"
	PlanStatusActive   = "ACTIVE"
	PlanStatusArchived = "ARCHIVED"
)

// Plan 是用户确认场景配置后保存的练习计划。
type Plan struct {
	ID               string    `json:"id"`
	ActorID          string    `json:"actor_id"`
	SceneID          string    `json:"scene_id"`
	SceneVersion     int       `json:"scene_version"`
	RoleID           string    `json:"role_id"`
	PracticeOptionID string    `json:"practice_option_id"`
	Objective        string    `json:"objective"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type CreatePlanInput struct {
	SceneID          string `json:"scene_id"`
	SceneVersion     int    `json:"scene_version"`
	RoleID           string `json:"role_id"`
	PracticeOptionID string `json:"practice_option_id"`
	Objective        string `json:"objective"`
}

// PlanRepository 是练习计划的持久化边界；方法都带 actorID，防止调用方忘记权限条件。
type PlanRepository interface {
	CreatePlan(context.Context, Plan) (Plan, error)
	ListPlans(context.Context, string) ([]Plan, error)
	FindPlan(context.Context, string, string) (Plan, error)
	ArchivePlan(context.Context, string, string, time.Time) (Plan, error)
}

type PlanService interface {
	CreatePlan(context.Context, string, CreatePlanInput) (Plan, error)
	ListPlans(context.Context, string) ([]Plan, error)
	GetPlan(context.Context, string, string) (Plan, error)
	ArchivePlan(context.Context, string, string) (Plan, error)
	EnsureCanCreateSession(context.Context, string, string) error
	CanCreateSession(context.Context, string, string) error
}

type planService struct {
	repo    PlanRepository
	catalog scene.CatalogReader
	now     func() time.Time
}

func NewPlanService(repo PlanRepository, catalog scene.CatalogReader) PlanService {
	if repo == nil || catalog == nil {
		panic("practice plan dependencies are required")
	}
	return &planService{repo: repo, catalog: catalog, now: time.Now}
}
func NewService(repo PlanRepository, catalog scene.CatalogReader) PlanService {
	return NewPlanService(repo, catalog)
}

func (s *planService) CreatePlan(ctx context.Context, actorID string, in CreatePlanInput) (Plan, error) {
	if strings.TrimSpace(actorID) == "" || strings.TrimSpace(in.SceneID) == "" || in.SceneVersion < 1 || strings.TrimSpace(in.RoleID) == "" || strings.TrimSpace(in.PracticeOptionID) == "" || strings.TrimSpace(in.Objective) == "" {
		return Plan{}, ErrInvalidPlan
	}
	detail, err := s.catalog.GetScene(ctx, in.SceneID)
	if err != nil || detail.SceneVersion != in.SceneVersion {
		return Plan{}, ErrInvalidPlan
	}
	var roleOK, optionOK bool
	for _, role := range detail.Roles {
		if role.ID == in.RoleID {
			roleOK = true
			break
		}
	}
	for _, option := range detail.PracticeOptions {
		if option.ID == in.PracticeOptionID && (option.RoleDefinitionID == nil || *option.RoleDefinitionID == in.RoleID) {
			optionOK = true
			break
		}
	}
	if !roleOK || !optionOK {
		return Plan{}, ErrInvalidPlan
	}
	now := s.now().UTC()
	return s.repo.CreatePlan(ctx, Plan{ID: newPlanID(), ActorID: actorID, SceneID: in.SceneID, SceneVersion: in.SceneVersion, RoleID: in.RoleID, PracticeOptionID: in.PracticeOptionID, Objective: strings.TrimSpace(in.Objective), Status: PlanStatusDraft, CreatedAt: now, UpdatedAt: now})
}
func (s *planService) ListPlans(ctx context.Context, actorID string) ([]Plan, error) {
	if strings.TrimSpace(actorID) == "" {
		return nil, ErrInvalidPlan
	}
	return s.repo.ListPlans(ctx, actorID)
}
func (s *planService) GetPlan(ctx context.Context, actorID, id string) (Plan, error) {
	if strings.TrimSpace(actorID) == "" || strings.TrimSpace(id) == "" {
		return Plan{}, ErrPlanNotFound
	}
	return s.repo.FindPlan(ctx, actorID, id)
}
func (s *planService) ArchivePlan(ctx context.Context, actorID, id string) (Plan, error) {
	if strings.TrimSpace(actorID) == "" || strings.TrimSpace(id) == "" {
		return Plan{}, ErrPlanNotFound
	}
	return s.repo.ArchivePlan(ctx, actorID, id, s.now().UTC())
}
func (s *planService) EnsureCanCreateSession(ctx context.Context, actorID, id string) error {
	plan, err := s.GetPlan(ctx, actorID, id)
	if err != nil {
		return err
	}
	if plan.Status == PlanStatusArchived {
		return ErrPlanArchived
	}
	return nil
}
func (s *planService) CanCreateSession(ctx context.Context, actorID, id string) error {
	return s.EnsureCanCreateSession(ctx, actorID, id)
}

func newPlanID() string {
	return uuid.NewString()
}

type MemoryPlanRepository struct {
	mu    sync.RWMutex
	plans map[string]Plan
}

func NewMemoryPlanRepository() *MemoryPlanRepository {
	return &MemoryPlanRepository{plans: map[string]Plan{}}
}

type MemoryRepository = MemoryPlanRepository

func NewMemoryRepository() *MemoryPlanRepository { return NewMemoryPlanRepository() }
func (r *MemoryPlanRepository) CreatePlan(ctx context.Context, plan Plan) (Plan, error) {
	if err := contextError(ctx); err != nil {
		return Plan{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.plans[plan.ID] = plan
	return plan, nil
}
func (r *MemoryPlanRepository) ListPlans(ctx context.Context, actorID string) ([]Plan, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Plan, 0)
	for _, plan := range r.plans {
		if plan.ActorID == actorID {
			out = append(out, plan)
		}
	}
	return out, nil
}
func (r *MemoryPlanRepository) FindPlan(ctx context.Context, actorID, id string) (Plan, error) {
	if err := contextError(ctx); err != nil {
		return Plan{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	plan, ok := r.plans[id]
	if !ok || plan.ActorID != actorID {
		return Plan{}, ErrPlanNotFound
	}
	return plan, nil
}
func (r *MemoryPlanRepository) ArchivePlan(ctx context.Context, actorID, id string, now time.Time) (Plan, error) {
	if err := contextError(ctx); err != nil {
		return Plan{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	plan, ok := r.plans[id]
	if !ok || plan.ActorID != actorID {
		return Plan{}, ErrPlanNotFound
	}
	if plan.Status == PlanStatusArchived {
		return Plan{}, ErrPlanArchived
	}
	plan.Status = PlanStatusArchived
	plan.UpdatedAt = now
	r.plans[id] = plan
	return plan, nil
}

func contextError(ctx context.Context) error {
	if ctx == nil {
		return errors.New("context is required")
	}
	return ctx.Err()
}

// Session 表示一次完整的口语练习会话。
type Session struct {
	ID      string `json:"id"`
	ActorID string `json:"actor_id"`
	SceneID string `json:"scene_id"`
	Status  string `json:"status"`
}

// Question 表示练习会话中的一道问题。
type Question struct {
	ID        string `json:"id"`
	SessionID string `json:"session_id"`
	Content   string `json:"content"`
}

// Turn 表示用户针对一道问题完成的一次回答回合。
type Turn struct {
	ID         string `json:"id"`
	QuestionID string `json:"question_id"`
	Answer     string `json:"answer"`
}

// CreateSessionInput 包含创建练习会话所需的数据。
type CreateSessionInput struct {
	ActorID string `json:"actor_id"`
	SceneID string `json:"scene_id"`
}

// SubmitAnswerInput 包含提交文字回答所需的数据。
type SubmitAnswerInput struct {
	SessionID  string `json:"session_id"`
	QuestionID string `json:"question_id"`
	Content    string `json:"content"`
}

// Repository 定义练习会话的持久化能力。
type Repository interface {
	// Create 保存新练习会话。
	Create(context.Context, Session) (Session, error)
	// FindByID 按会话标识读取练习会话。
	FindByID(context.Context, string) (Session, error)
	// SaveTurn 保存一个用户回答回合。
	SaveTurn(context.Context, Turn) (Turn, error)
}

// Service 定义练习会话的应用用例。
type Service interface {
	// CreateSession 创建一次练习会话。
	CreateSession(context.Context, CreateSessionInput) (Session, error)
	// GetSession 读取指定练习会话。
	GetSession(context.Context, string) (Session, error)
	// SubmitAnswer 提交一次文字回答。
	SubmitAnswer(context.Context, SubmitAnswerInput) (Turn, error)
	// CompleteSession 完成指定练习会话。
	CompleteSession(context.Context, string) error
}
