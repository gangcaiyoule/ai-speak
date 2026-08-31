package agent

import ("encoding/json"; "net/http")

// HTTPHandler 通过 REST 传输层暴露 Agent 用例。
type HTTPHandler struct{ service Service }
// NewHTTPHandler 使用指定 Agent 服务创建 HTTP 处理器。
func NewHTTPHandler(service Service) *HTTPHandler { return &HTTPHandler{service: service} }
// RegisterRoutes 在指定路由器上注册 Agent 接口。
func (h *HTTPHandler) RegisterRoutes(mux *http.ServeMux) { mux.HandleFunc("POST /v1/agent-threads", h.notImplemented); mux.HandleFunc("GET /v1/agent-threads", h.notImplemented); mux.HandleFunc("GET /v1/agent-threads/{thread_id}", h.notImplemented); mux.HandleFunc("POST /v1/agent-threads/{thread_id}/messages", h.notImplemented); mux.HandleFunc("POST /v1/agent-threads/{thread_id}/runs", h.notImplemented); mux.HandleFunc("GET /v1/agent-runs/{run_id}", h.notImplemented) }
// notImplemented 返回 Agent 服务尚未实现的结构化错误。
func (h *HTTPHandler) notImplemented(w http.ResponseWriter, _ *http.Request) { w.Header().Set("Content-Type", "application/json"); w.WriteHeader(http.StatusNotImplemented); _ = json.NewEncoder(w).Encode(map[string]string{"code":"not_implemented", "message":"agent service is not implemented"}) }
