// Package practice 定义口语练习会话相关契约。
package practice

import "context"

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
