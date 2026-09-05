package practice

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/gangcaiyoule/ai-speak/server/internal/coaching/scene"
)

var (
	ErrInvalidSession            = errors.New("invalid practice session")
	ErrSessionNotFound           = errors.New("practice session not found")
	ErrSessionNotActive          = errors.New("practice session is not active")
	ErrInvalidSessionTransition  = errors.New("invalid practice session state transition")
	ErrNoCurrentQuestion         = errors.New("practice session has no current question")
	ErrInvalidAnswer             = errors.New("invalid practice answer")
	ErrQuestionNotCurrent        = errors.New("practice question is not current")
	ErrAnswerAlreadySubmitted    = errors.New("practice answer already submitted")
	ErrSessionHasPendingQuestion = errors.New("practice session has pending questions")
)

const (
	SessionStatusDraft     = "DRAFT"
	SessionStatusActive    = "ACTIVE"
	SessionStatusCompleted = "COMPLETED"
)

// Session 是一次练习的不可变场景快照及其生命周期状态。
type Session struct {
	ID                string     `json:"id"`
	ActorID           string     `json:"actor_id"`
	PlanID            string     `json:"plan_id"`
	SceneID           string     `json:"scene_id"`
	SceneVersion      int        `json:"scene_version"`
	Status            string     `json:"status"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	Questions         []Question `json:"questions"`
	CurrentQuestionID *string    `json:"current_question_id"`
	CurrentQuestion   *Question  `json:"current_question"`
}

// Question 是创建会话时从 turn_blueprints 复制出的有序问题快照。
type Question struct {
	ID        string `json:"id"`
	SessionID string `json:"session_id"`
	Position  int    `json:"position"`
	Content   string `json:"content"`
}

// PracticeTurn 是用户针对某道问题提交的一次文本回答。
type PracticeTurn struct {
	ID         string    `json:"id"`
	SessionID  string    `json:"session_id"`
	QuestionID string    `json:"question_id"`
	ActorID    string    `json:"actor_id"`
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"created_at"`
}

// SubmitTextAnswerInput 是文本回答提交参数。
type SubmitTextAnswerInput struct {
	QuestionID string `json:"question_id"`
	Content    string `json:"content"`
}

type AnswerInput = SubmitTextAnswerInput

// CreateSessionInput 包含创建练习会话所需的数据。
type CreateSessionInput struct {
	PlanID string `json:"plan_id"`
}

// SessionRepository 是会话及问题快照的持久化边界。
// actorID 出现在所有读取和状态变更方法中，避免调用方遗漏越权约束。
type SessionRepository interface {
	CreateSession(context.Context, Session, []Question) (Session, error)
	FindSession(context.Context, string, string) (Session, error)
	FindLatestResumableSession(context.Context, string) (Session, error)
	ActivateSession(context.Context, string, string, time.Time) (Session, error)
	CompleteSession(context.Context, string, string, time.Time) (Session, error)
	SubmitTextAnswer(context.Context, string, string, SubmitTextAnswerInput, time.Time) (PracticeTurn, error)
}

// SessionService 定义练习会话创建、激活、查看和完成用例。
type SessionService interface {
	CreateSession(context.Context, string, CreateSessionInput) (Session, error)
	GetSession(context.Context, string, string) (Session, error)
	GetLatestResumableSession(context.Context, string) (Session, error)
	GetCurrentQuestion(context.Context, string, string) (Question, error)
	ActivateSession(context.Context, string, string) (Session, error)
	SubmitTextAnswer(context.Context, string, string, SubmitTextAnswerInput) (PracticeTurn, error)
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

func (s *sessionService) GetLatestResumableSession(ctx context.Context, actorID string) (Session, error) {
	if strings.TrimSpace(actorID) == "" {
		return Session{}, ErrSessionNotFound
	}
	session, err := s.repo.FindLatestResumableSession(ctx, actorID)
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

func (s *sessionService) SubmitTextAnswer(ctx context.Context, actorID, sessionID string, in SubmitTextAnswerInput) (PracticeTurn, error) {
	if strings.TrimSpace(actorID) == "" || strings.TrimSpace(sessionID) == "" {
		return PracticeTurn{}, ErrSessionNotFound
	}
	questionID := strings.TrimSpace(in.QuestionID)
	content := strings.TrimSpace(in.Content)
	if questionID == "" || content == "" {
		return PracticeTurn{}, ErrInvalidAnswer
	}
	return s.repo.SubmitTextAnswer(ctx, actorID, sessionID, SubmitTextAnswerInput{QuestionID: questionID, Content: content}, s.now().UTC())
}

func (s *sessionService) SubmitAnswer(ctx context.Context, actorID, sessionID string, in SubmitTextAnswerInput) (PracticeTurn, error) {
	return s.SubmitTextAnswer(ctx, actorID, sessionID, in)
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
	if session.Status != SessionStatusActive {
		session.CurrentQuestionID = nil
		return session
	}
	if session.CurrentQuestionID != nil {
		for _, question := range session.Questions {
			if question.ID == *session.CurrentQuestionID {
				value := question
				session.CurrentQuestion = &value
				break
			}
		}
	}
	return session
}

// MemorySessionRepository 是用于本地开发和单元测试的会话 Repository。
type MemorySessionRepository struct {
	mu        sync.RWMutex
	sessions  map[string]Session
	questions map[string][]Question
	turns     map[string]map[string]PracticeTurn
}

func NewMemorySessionRepository() *MemorySessionRepository {
	return &MemorySessionRepository{sessions: map[string]Session{}, questions: map[string][]Question{}, turns: map[string]map[string]PracticeTurn{}}
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
	r.turns[session.ID] = map[string]PracticeTurn{}
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

func (r *MemorySessionRepository) FindLatestResumableSession(ctx context.Context, actorID string) (Session, error) {
	if err := contextError(ctx); err != nil {
		return Session{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	var selected *Session
	for _, candidate := range r.sessions {
		if candidate.ActorID != actorID || (candidate.Status != SessionStatusDraft && candidate.Status != SessionStatusActive) {
			continue
		}
		if selected == nil || candidate.UpdatedAt.After(selected.UpdatedAt) || (candidate.UpdatedAt.Equal(selected.UpdatedAt) && candidate.ID > selected.ID) {
			value := candidate
			selected = &value
		}
	}
	if selected == nil {
		return Session{}, ErrSessionNotFound
	}
	return withQuestions(*selected, r.questions[selected.ID]), nil
}

func (r *MemorySessionRepository) ActivateSession(ctx context.Context, actorID, sessionID string, now time.Time) (Session, error) {
	return r.transition(ctx, actorID, sessionID, SessionStatusDraft, SessionStatusActive, now)
}

func (r *MemorySessionRepository) CompleteSession(ctx context.Context, actorID, sessionID string, now time.Time) (Session, error) {
	if err := contextError(ctx); err != nil {
		return Session{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	session, ok := r.sessions[sessionID]
	if !ok || session.ActorID != actorID {
		return Session{}, ErrSessionNotFound
	}
	if session.Status != SessionStatusActive {
		return Session{}, ErrInvalidSessionTransition
	}
	if session.CurrentQuestionID != nil {
		return Session{}, ErrSessionHasPendingQuestion
	}
	session.Status = SessionStatusCompleted
	session.UpdatedAt = now
	r.sessions[sessionID] = session
	return withQuestions(session, r.questions[sessionID]), nil
}

func (r *MemorySessionRepository) SubmitTextAnswer(ctx context.Context, actorID, sessionID string, in SubmitTextAnswerInput, now time.Time) (PracticeTurn, error) {
	if err := contextError(ctx); err != nil {
		return PracticeTurn{}, err
	}
	questionID := strings.TrimSpace(in.QuestionID)
	content := strings.TrimSpace(in.Content)
	if questionID == "" || content == "" {
		return PracticeTurn{}, ErrInvalidAnswer
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	session, ok := r.sessions[sessionID]
	if !ok || session.ActorID != actorID {
		return PracticeTurn{}, ErrSessionNotFound
	}
	if session.Status != SessionStatusActive {
		return PracticeTurn{}, ErrSessionNotActive
	}
	if _, exists := r.turns[sessionID][questionID]; exists {
		return PracticeTurn{}, ErrAnswerAlreadySubmitted
	}
	if session.CurrentQuestionID == nil {
		return PracticeTurn{}, ErrNoCurrentQuestion
	}
	if *session.CurrentQuestionID != questionID {
		return PracticeTurn{}, ErrQuestionNotCurrent
	}
	turn := PracticeTurn{ID: uuid.NewString(), SessionID: sessionID, QuestionID: questionID, ActorID: actorID, Content: content, CreatedAt: now}
	r.turns[sessionID][questionID] = turn
	next := nextUnansweredQuestion(r.questions[sessionID], r.turns[sessionID], questionID)
	session.CurrentQuestionID = next
	session.UpdatedAt = now
	r.sessions[sessionID] = session
	return turn, nil
}

func (r *MemorySessionRepository) ListTurns(ctx context.Context, actorID, sessionID string) ([]PracticeTurn, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	session, ok := r.sessions[sessionID]
	if !ok || session.ActorID != actorID {
		return nil, ErrSessionNotFound
	}
	turns := make([]PracticeTurn, 0, len(r.turns[sessionID]))
	for _, turn := range r.turns[sessionID] {
		turns = append(turns, turn)
	}
	sort.Slice(turns, func(i, j int) bool { return turns[i].CreatedAt.Before(turns[j].CreatedAt) })
	return turns, nil
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
	if to == SessionStatusActive && session.CurrentQuestionID == nil {
		if questions := r.questions[sessionID]; len(questions) > 0 {
			id := questions[0].ID
			session.CurrentQuestionID = &id
		}
	}
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

func nextUnansweredQuestion(questions []Question, turns map[string]PracticeTurn, answeredID string) *string {
	position := 0
	for _, question := range questions {
		if question.ID == answeredID {
			position = question.Position
			break
		}
	}
	for _, question := range questions {
		if question.Position > position {
			if _, answered := turns[question.ID]; !answered {
				id := question.ID
				return &id
			}
		}
	}
	return nil
}
