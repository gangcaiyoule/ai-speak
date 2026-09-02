// Package coaching 组装口语训练模块的传输层接口。
package coaching

import (
	"encoding/json"
	"net/http"

	"github.com/gangcaiyoule/ai-speak/server/internal/coaching/scene"
)

// HTTPHandler 通过 REST 传输层暴露场景、练习和评测接口。
type HTTPHandler struct{ catalog scene.CatalogReader }

// NewHTTPHandler 创建口语训练模块的空 HTTP 处理器。
func NewHTTPHandler() *HTTPHandler { return &HTTPHandler{catalog: scene.NewCatalog()} }

// RegisterRoutes 在指定路由器上注册口语训练接口。
func (h *HTTPHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/scenes", h.listScenes)
	mux.HandleFunc("GET /v1/scenes/{scene_id}", h.getScene)
	mux.HandleFunc("GET /v1/scenes/{scene_id}/roles", h.listRoles)
	mux.HandleFunc("POST /v1/practice-sessions", h.notImplemented)
	mux.HandleFunc("GET /v1/practice-sessions/{session_id}", h.notImplemented)
	mux.HandleFunc("POST /v1/practice-sessions/{session_id}/activation", h.notImplemented)
	mux.HandleFunc("POST /v1/practice-sessions/{session_id}/text-answers", h.notImplemented)
	mux.HandleFunc("POST /v1/practice-sessions/{session_id}/complete", h.notImplemented)
	mux.HandleFunc("GET /v1/practice-sessions/{session_id}/evaluation", h.notImplemented)
	mux.HandleFunc("GET /v1/evaluation-reports/{report_id}", h.notImplemented)
}

func (h *HTTPHandler) listScenes(w http.ResponseWriter, r *http.Request) {
	scenes, err := h.catalog.ListScenes(r.Context())
	if err != nil {
		h.writeSceneError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"scenes": scenes})
}

func (h *HTTPHandler) getScene(w http.ResponseWriter, r *http.Request) {
	value, err := h.catalog.GetScene(r.Context(), r.PathValue("scene_id"))
	if err != nil {
		h.writeSceneError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (h *HTTPHandler) listRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := h.catalog.ListRoles(r.Context(), r.PathValue("scene_id"))
	if err != nil {
		h.writeSceneError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"roles": roles})
}

func (h *HTTPHandler) writeSceneError(w http.ResponseWriter, err error) {
	if err == scene.ErrSceneNotFound {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"error": map[string]string{
				"code":    "scene_not_found",
				"message": "Scene was not found.",
			},
		})
		return
	}
	writeJSON(w, http.StatusInternalServerError, map[string]any{
		"error": map[string]string{
			"code":    "internal_error",
			"message": "Internal server error.",
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		return
	}
}

// notImplemented 返回口语训练服务尚未实现的结构化错误。
func (h *HTTPHandler) notImplemented(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	_ = json.NewEncoder(w).Encode(map[string]string{"code": "not_implemented", "message": "coaching service is not implemented"})
}
