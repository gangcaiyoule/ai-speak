// Package coaching 组装口语训练模块的传输层接口。
package coaching

import (
	"encoding/json"
	"net/http"
)

// HTTPHandler 通过 REST 传输层暴露场景、练习和评测接口。
type HTTPHandler struct{}

// NewHTTPHandler 创建口语训练模块的空 HTTP 处理器。
func NewHTTPHandler() *HTTPHandler { return &HTTPHandler{} }

// RegisterRoutes 在指定路由器上注册口语训练接口。
func (h *HTTPHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/scenes", h.notImplemented)
	mux.HandleFunc("POST /v1/practice-sessions", h.notImplemented)
	mux.HandleFunc("GET /v1/practice-sessions/{session_id}", h.notImplemented)
	mux.HandleFunc("POST /v1/practice-sessions/{session_id}/activation", h.notImplemented)
	mux.HandleFunc("POST /v1/practice-sessions/{session_id}/text-answers", h.notImplemented)
	mux.HandleFunc("POST /v1/practice-sessions/{session_id}/complete", h.notImplemented)
	mux.HandleFunc("GET /v1/practice-sessions/{session_id}/evaluation", h.notImplemented)
	mux.HandleFunc("GET /v1/evaluation-reports/{report_id}", h.notImplemented)
}

// notImplemented 返回口语训练服务尚未实现的结构化错误。
func (h *HTTPHandler) notImplemented(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	_ = json.NewEncoder(w).Encode(map[string]string{"code": "not_implemented", "message": "coaching service is not implemented"})
}
