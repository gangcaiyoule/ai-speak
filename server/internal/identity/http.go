package identity

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

// HTTPHandler 通过 REST 暴露身份相关用例。
type HTTPHandler struct{ service AuthService }

// NewHTTPHandler 使用认证服务创建 HTTP 处理器。
func NewHTTPHandler(service AuthService) *HTTPHandler { return &HTTPHandler{service: service} }

// RegisterRoutes 注册注册、登录、退出和当前用户路由。
func (h *HTTPHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/auth/register", h.register)
	mux.HandleFunc("POST /v1/auth/login", h.login)
	mux.HandleFunc("POST /v1/auth/logout", h.logout)
	mux.HandleFunc("GET /v1/me", h.currentUser)
}

func (h *HTTPHandler) register(w http.ResponseWriter, r *http.Request) {
	var input RegisterInput
	if !decodeJSON(w, r, &input) {
		return
	}
	session, err := h.service.Register(r.Context(), input)
	if err != nil {
		writeIdentityError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, session)
}

func (h *HTTPHandler) login(w http.ResponseWriter, r *http.Request) {
	var input LoginInput
	if !decodeJSON(w, r, &input) {
		return
	}
	session, err := h.service.Login(r.Context(), input)
	if err != nil {
		writeIdentityError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, session)
}

func (h *HTTPHandler) logout(w http.ResponseWriter, r *http.Request) {
	token, ok := bearerToken(r)
	if !ok {
		writeIdentityError(w, ErrUnauthorized)
		return
	}
	if err := h.service.Logout(r.Context(), token); err != nil {
		writeIdentityError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *HTTPHandler) currentUser(w http.ResponseWriter, r *http.Request) {
	token, ok := bearerToken(r)
	if !ok {
		writeIdentityError(w, ErrUnauthorized)
		return
	}
	user, err := h.service.CurrentUser(r.Context(), token)
	if err != nil {
		writeIdentityError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func bearerToken(r *http.Request) (string, bool) {
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || !strings.HasPrefix(parts[1], "sess_") {
		return "", false
	}
	return parts[1], true
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeIdentityError(w, ErrInvalidRequest)
		return false
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	if err := decoder.Decode(target); err != nil {
		writeIdentityError(w, ErrInvalidRequest)
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		panic(err)
	}
}

func writeIdentityError(w http.ResponseWriter, err error) {
	status, code := http.StatusInternalServerError, "internal_error"
	switch {
	case errors.Is(err, ErrInvalidRequest):
		status, code = http.StatusBadRequest, "invalid_request"
	case errors.Is(err, ErrConflict):
		status, code = http.StatusConflict, "registration_unavailable"
	case errors.Is(err, ErrInvalidCredentials):
		status, code = http.StatusUnauthorized, "invalid_credentials"
	case errors.Is(err, ErrUnauthorized):
		status, code = http.StatusUnauthorized, "authentication_required"
	}
	writeJSON(w, status, map[string]string{"code": code, "message": code})
}
