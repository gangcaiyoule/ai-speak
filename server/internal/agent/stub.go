package agent

import ("context"; "errors")

var errNotImplemented = errors.New("agent service is not implemented")

// StubService 是架构阶段使用的 Agent 服务空实现。
type StubService struct{}
// CreateThread 返回“未实现”占位错误。
func (StubService) CreateThread(context.Context, string) (Thread, error) { return Thread{}, errNotImplemented }
// GetThread 返回“未实现”占位错误。
func (StubService) GetThread(context.Context, string) (Thread, error) { return Thread{}, errNotImplemented }
// ListThreads 返回“未实现”占位错误。
func (StubService) ListThreads(context.Context, string) ([]Thread, error) { return nil, errNotImplemented }
// AppendMessage 返回“未实现”占位错误。
func (StubService) AppendMessage(context.Context, Message) (Message, error) { return Message{}, errNotImplemented }
// StartRun 返回“未实现”占位错误。
func (StubService) StartRun(context.Context, string) (Run, error) { return Run{}, errNotImplemented }

// StubTextGenerator 是架构阶段使用的文本生成器空实现。
type StubTextGenerator struct{}
// Generate 返回“未实现”占位错误。
func (StubTextGenerator) Generate(context.Context, GenerationRequest) (GenerationResponse, error) { return GenerationResponse{}, errNotImplemented }
