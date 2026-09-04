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
	ErrInvalidSession           = errors.New("invalid practice session")
	ErrSessionNotFound          = errors.New("practice session not found")
	ErrSessionNotActive         = errors.New("practice session is not active")
	ErrInvalidSessionTransition = errors.New("invalid practice session state transition")
	ErrNoCurrentQuestion        = errors.New("practice session has no current question")
)

const (
	SessionStatusDraft     = "DRAFT"
	SessionStatusActive    = "ACTIVE"
	SessionStatusCompleted = "COMPLETED"
)

// Session 是一次练习的不可变场景快照及其生命周期状态。
type Session struct {
	ID              string     `json:"id"`
	ActorID         string     `json:"actor_id"`
	PlanID          string     `json:"plan_id"`
	SceneID         string     `json:"scene_id"`
	SceneVersion    int        `json:"scene_version"`
	Status          string     `json:"status"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	Questions       []Question `json:"questions"`
	CurrentQuestion *Question  `json:"current_question"`
}

// Question 是创建会话时从 turn_blueprints 复制出的有序问题快照。
type Question struct {
	ID        string `json:"id"`
	SessionID string `json:"session_id"`
	Position  int    `json:"position"`
	Content   string `json:"content"`
}

// CreateSessionInput 包含创建练习会话所需的数据。
type CreateSessionInput struct {
	PlanID string `json:"plan_id"`
}

// SessionRepository 是会话及问题快照的持久化边界。
// actorID 出现在所有读取和状态变更方法中，避免调用方遗漏越权约束。
type SessionRepository interface {
	CreateSession(context.Context, Session, []Question) (Session, error)
	FindSession(context.Context, string, string) (Session, error)
	ActivateSession(context.Context, string, string, time.Time) (Session, error)
	CompleteSession(context.Context, string, string, time.Time) (Session, error)
}

// SessionService 定义练习会话创建、激活、查看和完成用例。
type SessionService interface {
	CreateSession(context.Context, string, CreateSessionInput) (Session, error)
	GetSession(context.Context, string, string) (Session, error)
	GetCurrentQuestion(context.Context, string, string) (Question, error)
	ActivateSession(context.Context, string, string) (Session, error)
	CompleteSession(context.Context, string, string) (Session, error)
}

// Service 保留练习模块对外的通用服务名称。
type Service = SessionService

type sessionService struct {
	repo    SessionRepository
	plans   PlanService
	catalog scene.CatalogReader
	now     func() time.Time
}

func NewSessionService(repo SessionRepository, plans PlanService, catalog scene.CatalogReader) SessionService {
	if repo == nil || plans == nil || catalog == nil {
		panic("practice session dependencies are required")
	}
	return &sessionService{repo: repo, plans: plans, catalog: catalog, now: time.Now}
}

func NewPracticeSessionService(repo SessionRepository, plans PlanService, catalog scene.CatalogReader) SessionService {
	return NewSessionService(repo, plans, catalog)
}

func (s *sessionService) CreateSession(ctx context.Context, actorID string, in CreateSessionInput) (Session, error) {
	if strings.TrimSpace(actorID) == "" || strings.TrimSpace(in.PlanID) == "" {
		return Session{}, ErrInvalidSession
	}
	plan, err := s.plans.GetPlan(ctx, actorID, in.PlanID)
	if err != nil {
		return Session{}, err
	}
	if plan.Status != PlanStatusDraft && plan.Status != PlanStatusActive {
		if plan.Status == PlanStatusArchived {
			return Session{}, ErrPlanArchived
		}
		return Session{}, ErrInvalidPlan
	}
	detail, err := s.catalog.GetScene(ctx, plan.SceneID)
	if err != nil || detail.SceneVersion != plan.SceneVersion || len(detail.Prompt.TurnBlueprints) == 0 {
		return Session{}, ErrInvalidSession
	}

	now := s.now().UTC()
	session := Session{
		ID:           uuid.NewString(),
		ActorID:      actorID,
		PlanID:       plan.ID,
		SceneID:      plan.SceneID,
		SceneVersion: plan.SceneVersion,
		Status:       SessionStatusDraft,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	questions := make([]Question, 0, len(detail.Prompt.TurnBlueprints))
	for position, blueprint := range detail.Prompt.TurnBlueprints {
		if strings.TrimSpace(blueprint) == "" {
			return Session{}, ErrInvalidSession
		}
		questions = append(questions, Question{
			ID:        uuid.NewString(),
			SessionID: session.ID,
			Position:  position + 1,
			Content:   blueprint,
		})
	}
	return s.repo.CreateSession(ctx, session, questions)
}

func (s *sessionService) GetSession(ctx context.Context, actorID, sessionID string) (Session, error) {
	if strings.TrimSpace(actorID) == "" || strings.TrimSpace(sessionID) == "" {
		return Session{}, ErrSessionNotFound
	}
	session, err := s.repo.FindSession(ctx, actorID, sessionID)
	if err != nil {
		return Session{}, err
	}
	return withCurrentQuestion(session), nil
}

func (s *sessionService) GetCurrentQuestion(ctx context.Context, actorID, sessionID string) (Question, error) {
	session, err := s.GetSession(ctx, actorID, sessionID)
	if err != nil {
		return Question{}, err
	}
	if session.Status != SessionStatusActive {
		return Question{}, ErrSessionNotActive
	}
	if session.CurrentQuestion == nil {
		return Question{}, ErrNoCurrentQuestion
	}
	return *session.CurrentQuestion, nil
}

func (s *sessionService) ActivateSession(ctx context.Context, actorID, sessionID string) (Session, error) {
	if strings.TrimSpace(actorID) == "" || strings.TrimSpace(sessionID) == "" {
		return Session{}, ErrSessionNotFound
	}
	session, err := s.repo.ActivateSession(ctx, actorID, sessionID, s.now().UTC())
	if err != nil {
		return Session{}, err
	}
	return withCurrentQuestion(session), nil
}

func (s *sessionService) CompleteSession(ctx context.Context, actorID, sessionID string) (Session, error) {
	if strings.TrimSpace(actorID) == "" || strings.TrimSpace(sessionID) == "" {
		return Session{}, ErrSessionNotFound
	}
	session, err := s.repo.CompleteSession(ctx, actorID, sessionID, s.now().UTC())
	if err != nil {
		return Session{}, err
	}
	return withCurrentQuestion(session), nil
}

func withCurrentQuestion(session Session) Session {
	session.Questions = append([]Question(nil), session.Questions...)
	session.CurrentQuestion = nil
	if session.Status == SessionStatusActive && len(session.Questions) > 0 {
		question := session.Questions[0]
		session.CurrentQuestion = &question
	}
	return session
}

// MemorySessionRepository 是用于本地开发和单元测试的会话 Repository。
type MemorySessionRepository struct {
	mu        sync.RWMutex
	sessions  map[string]Session
	questions map[string][]Question
}

func NewMemorySessionRepository() *MemorySessionRepository {
	return &MemorySessionRepository{sessions: map[string]Session{}, questions: map[string][]Question{}}
}

func (r *MemorySessionRepository) CreateSession(ctx context.Context, session Session, questions []Question) (Session, error) {
	if err := contextError(ctx); err != nil {
		return Session{}, err
	}
	if err := validateSessionSnapshot(session, questions); err != nil {
		return Session{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.sessions[session.ID]; exists {
		return Session{}, ErrInvalidSession
	}
	r.sessions[session.ID] = session
	r.questions[session.ID] = cloneQuestions(questions)
	return withQuestions(session, questions), nil
}

func (r *MemorySessionRepository) FindSession(ctx context.Context, actorID, sessionID string) (Session, error) {
	if err := contextError(ctx); err != nil {
		return Session{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	session, ok := r.sessions[sessionID]
	if !ok || session.ActorID != actorID {
		return Session{}, ErrSessionNotFound
	}
	return withQuestions(session, r.questions[sessionID]), nil
}

func (r *MemorySessionRepository) ActivateSession(ctx context.Context, actorID, sessionID string, now time.Time) (Session, error) {
	return r.transition(ctx, actorID, sessionID, SessionStatusDraft, SessionStatusActive, now)
}

func (r *MemorySessionRepository) CompleteSession(ctx context.Context, actorID, sessionID string, now time.Time) (Session, error) {
	return r.transition(ctx, actorID, sessionID, SessionStatusActive, SessionStatusCompleted, now)
}

func (r *MemorySessionRepository) transition(ctx context.Context, actorID, sessionID, from, to string, now time.Time) (Session, error) {
	if err := contextError(ctx); err != nil {
		return Session{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	session, ok := r.sessions[sessionID]
	if !ok || session.ActorID != actorID {
		return Session{}, ErrSessionNotFound
	}
	if session.Status != from {
		return Session{}, ErrInvalidSessionTransition
	}
	session.Status = to
	session.UpdatedAt = now
	r.sessions[sessionID] = session
	return withQuestions(session, r.questions[sessionID]), nil
}

func validateSessionSnapshot(session Session, questions []Question) error {
	if strings.TrimSpace(session.ID) == "" || strings.TrimSpace(session.ActorID) == "" || strings.TrimSpace(session.PlanID) == "" || strings.TrimSpace(session.SceneID) == "" || session.SceneVersion < 1 || session.Status != SessionStatusDraft || len(questions) == 0 {
		return ErrInvalidSession
	}
	for position, question := range questions {
		if strings.TrimSpace(question.ID) == "" || question.SessionID != session.ID || question.Position != position+1 || strings.TrimSpace(question.Content) == "" {
			return ErrInvalidSession
		}
	}
	return nil
}

func withQuestions(session Session, questions []Question) Session {
	session.Questions = cloneQuestions(questions)
	session.CurrentQuestion = nil
	return session
}

func cloneQuestions(questions []Question) []Question {
	return append([]Question(nil), questions...)
}
