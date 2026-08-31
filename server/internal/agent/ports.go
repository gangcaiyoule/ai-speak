package agent

import "context"

// ThreadRepository 定义对话线程的持久化能力。
type ThreadRepository interface {
	// Create 保存新线程并返回持久化结果。
	Create(context.Context, Thread) (Thread, error)
	// FindByID 按线程标识读取线程。
	FindByID(context.Context, string) (Thread, error)
	// ListByActor 读取指定用户拥有的线程。
	ListByActor(context.Context, string) ([]Thread, error)
}

// MessageRepository 定义对话消息的持久化能力。
type MessageRepository interface {
	// Append 向线程追加消息。
	Append(context.Context, Message) (Message, error)
	// ListByThread 读取指定线程的全部消息。
	ListByThread(context.Context, string) ([]Message, error)
}

// TextGenerator 定义调用文本模型生成回复的能力。
type TextGenerator interface {
	// Generate 根据标准消息上下文生成文本回复。
	Generate(context.Context, GenerationRequest) (GenerationResponse, error)
}

// Service 定义 Agent 对话的应用用例。
type Service interface {
	// CreateThread 为指定用户创建对话线程。
	CreateThread(context.Context, string) (Thread, error)
	// GetThread 读取指定对话线程。
	GetThread(context.Context, string) (Thread, error)
	// ListThreads 读取指定用户的对话线程。
	ListThreads(context.Context, string) ([]Thread, error)
	// AppendMessage 向指定线程追加消息。
	AppendMessage(context.Context, Message) (Message, error)
	// StartRun 启动指定线程的 AI 生成任务。
	StartRun(context.Context, string) (Run, error)
}
