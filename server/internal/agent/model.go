// Package agent 定义 AI 对话相关的领域契约。
package agent

// Thread 表示一条可持续追加消息的 AI 对话线程。
type Thread struct {
	ID      string `json:"id"`
	ActorID string `json:"actor_id"`
}

// Message 表示对话线程中的一条用户或助手消息。
type Message struct {
	ID       string `json:"id"`
	ThreadID string `json:"thread_id"`
	Role     string `json:"role"`
	Content  string `json:"content"`
}

// Run 表示一次基于线程上下文执行的 AI 生成任务。
type Run struct {
	ID       string `json:"id"`
	ThreadID string `json:"thread_id"`
	Status   string `json:"status"`
}

// GenerationRequest 表示提交给文本生成器的标准请求。
type GenerationRequest struct {
	Messages []Message `json:"messages"`
}

// GenerationResponse 表示文本生成器返回的标准结果。
type GenerationResponse struct {
	Content string `json:"content"`
}
