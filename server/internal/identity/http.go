package identity

import (
	"encoding/json"
	"net/http"
)

// HTTPHandler 通过 REST 传输层暴露身份相关用例。
type HTTPHandler struct {
	service AuthService
}

// NewHTTPHandler 使用指定的身份服务创建 HTTP 处理器。
func NewHTTPHandler(service AuthService) *HTTPHandler { return &HTTPHandler{service: service} }

// RegisterRoutes 在指定路由器上注册身份接口。
func (h *HTTPHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/auth/register", h.notImplemented)
	mux.HandleFunc("POST /v1/auth/login", h.notImplemented)
	mux.HandleFunc("POST /v1/auth/logout", h.notImplemented)
	mux.HandleFunc("GET /v1/me", h.notImplemented)
}

// notImplemented 返回架构阶段身份服务尚未实现的结构化错误。
func (h *HTTPHandler) notImplemented(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusNotImplemented)
	_ = json.NewEncoder(writer).Encode(map[string]string{
		"code":    "not_implemented",
		"message": "identity service is not implemented",
	})
}
