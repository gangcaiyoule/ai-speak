package identity

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

type HTTPHandler struct{ service AuthService }

func NewHTTPHandler(s AuthService) *HTTPHandler { return &HTTPHandler{service: s} }
func (h *HTTPHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/auth/register", h.register)
	mux.HandleFunc("POST /v1/auth/login", h.login)
	mux.HandleFunc("POST /v1/auth/logout", h.logout)
	mux.HandleFunc("GET /v1/me", h.currentUser)
}
func (h *HTTPHandler) register(w http.ResponseWriter, r *http.Request) {
	var in RegisterInput
	if !decodeJSON(w, r, &in) {
		return
	}
	u, e := h.service.Register(r.Context(), in)
	if e != nil {
		writeErr(w, e)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]User{"user": u})
}
func (h *HTTPHandler) login(w http.ResponseWriter, r *http.Request) {
	var in LoginInput
	if !decodeJSON(w, r, &in) {
		return
	}
	v, e := h.service.Login(r.Context(), in)
	if e != nil {
		writeErr(w, e)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, v)
}
func (h *HTTPHandler) logout(w http.ResponseWriter, r *http.Request) {
	t, ok := bearerToken(r)
	if !ok {
		writeErr(w, ErrUnauthorized)
		return
	}
	a, e := h.service.Authenticate(r.Context(), t)
	if e != nil {
		writeErr(w, e)
		return
	}
	if e = h.service.Logout(r.Context(), a.SessionID); e != nil {
		writeErr(w, e)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *HTTPHandler) currentUser(w http.ResponseWriter, r *http.Request) {
	t, ok := bearerToken(r)
	if !ok {
		writeErr(w, ErrUnauthorized)
		return
	}
	a, e := h.service.Authenticate(r.Context(), t)
	if e != nil {
		writeErr(w, e)
		return
	}
	u, e := h.service.CurrentUser(r.Context(), a.UserID)
	if e != nil {
		writeErr(w, e)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]User{"user": u})
}
func bearerToken(r *http.Request) (string, bool) {
	p := strings.Fields(r.Header.Get("Authorization"))
	if len(p) != 2 || !strings.EqualFold(p[0], "Bearer") || !validToken(p[1]) {
		return "", false
	}
	return p[1], true
}
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeErr(w, ErrInvalidRequest)
		return false
	}
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	d.DisallowUnknownFields()
	if d.Decode(v) != nil {
		writeErr(w, ErrInvalidRequest)
		return false
	}
	return true
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func writeErr(w http.ResponseWriter, e error) {
	status, code := http.StatusInternalServerError, "internal_error"
	switch {
	case errors.Is(e, ErrInvalidRequest):
		status, code = http.StatusBadRequest, "invalid_request"
	case errors.Is(e, ErrConflict):
		status, code = http.StatusConflict, "account_registration_unavailable"
	case errors.Is(e, ErrInvalidCredentials):
		status, code = http.StatusUnauthorized, "invalid_credentials"
	case errors.Is(e, ErrUnauthorized):
		status, code = http.StatusUnauthorized, "authentication_required"
	}
	if status == http.StatusUnauthorized {
		w.Header().Set("WWW-Authenticate", "Bearer")
	}
	writeJSON(w, status, map[string]any{"error": map[string]any{"code": code, "message": code, "retryable": status >= 500}})
}
