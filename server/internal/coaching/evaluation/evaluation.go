// Package evaluation 定义练习评测相关契约。
package evaluation

import "context"

// Report 表示一次练习会话的评测报告。
type Report struct {
	ID        string         `json:"id"`
	SessionID string         `json:"session_id"`
	Status    string         `json:"status"`
	Summary   string         `json:"summary"`
	Items     []FeedbackItem `json:"items"`
}

// FeedbackItem 表示评测报告中的一条具体反馈。
type FeedbackItem struct {
	ID       string `json:"id"`
	Category string `json:"category"`
	Comment  string `json:"comment"`
}

// Repository 定义评测报告的持久化能力。
type Repository interface {
	// Create 保存新评测报告。
	Create(context.Context, Report) (Report, error)
	// FindByID 按报告标识读取评测报告。
	FindByID(context.Context, string) (Report, error)
	// FindBySession 按练习会话标识读取评测报告。
	FindBySession(context.Context, string) (Report, error)
}

// Service 定义练习评测的应用用例。
type Service interface {
	// CreateForSession 为指定练习会话创建评测。
	CreateForSession(context.Context, string) (Report, error)
	// GetByID 读取指定评测报告。
	GetByID(context.Context, string) (Report, error)
	// GetBySession 读取指定练习会话的评测报告。
	GetBySession(context.Context, string) (Report, error)
}
