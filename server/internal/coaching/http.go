// Package coaching 组装口语训练模块的传输层接口。
package coaching

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gangcaiyoule/ai-speak/server/internal/coaching/evaluation"
	"github.com/gangcaiyoule/ai-speak/server/internal/coaching/practice"
	"github.com/gangcaiyoule/ai-speak/server/internal/coaching/scene"
	"github.com/gangcaiyoule/ai-speak/server/internal/identity"
)

// HTTPHandler 通过 REST 传输层暴露场景、练习和评测接口。
type HTTPHandler struct {
	catalog     scene.CatalogReader
	auth        identity.AuthService
	plans       practice.PlanService
	sessions    practice.SessionService
	evaluations evaluation.Repository
}

var evaluationIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)

// NewHTTPHandler 创建使用内存 Repository 的本地 HTTP 处理器。
func NewHTTPHandler() *HTTPHandler {
	catalog := scene.NewCatalog()
	plans := practice.NewPlanService(practice.NewMemoryPlanRepository(), catalog)
	sessions := practice.NewSessionService(practice.NewMemorySessionRepository(), plans, catalog)
	return &HTTPHandler{catalog: catalog, plans: plans, sessions: sessions}
}
func NewHTTPHandlerWithDependencies(auth identity.AuthService, plans practice.PlanService, catalog scene.CatalogReader) *HTTPHandler {
	if plans == nil || catalog == nil {
		panic("coaching plan dependencies are required")
	}
	return NewHTTPHandlerWithAllDependencies(auth, plans, practice.NewSessionService(practice.NewMemorySessionRepository(), plans, catalog), catalog)
}

func NewHTTPHandlerWithAllDependencies(auth identity.AuthService, plans practice.PlanService, sessions practice.SessionService, catalog scene.CatalogReader) *HTTPHandler {
	if plans == nil || sessions == nil || catalog == nil {
		panic("coaching dependencies are required")
	}
	return &HTTPHandler{catalog: catalog, auth: auth, plans: plans, sessions: sessions}
}

// NewHTTPHandlerWithEvaluationRepository adds the report query repository to an existing handler.
func NewHTTPHandlerWithEvaluationRepository(auth identity.AuthService, plans practice.PlanService, sessions practice.SessionService, catalog scene.CatalogReader, reports evaluation.Repository) *HTTPHandler {
	h := NewHTTPHandlerWithAllDependencies(auth, plans, sessions, catalog)
	h.evaluations = reports
	return h
}

// RegisterRoutes 在指定路由器上注册口语训练接口。
func (h *HTTPHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/scenes", h.listScenes)
	mux.HandleFunc("GET /v1/scenes/{scene_id}", h.getScene)
	mux.HandleFunc("GET /v1/scenes/{scene_id}/roles", h.listRoles)
	mux.HandleFunc("POST /v1/practice-plans", h.createPlan)
	mux.HandleFunc("GET /v1/practice-plans", h.listPlans)
	mux.HandleFunc("GET /v1/practice-plans/{plan_id}", h.getPlan)
	mux.HandleFunc("POST /v1/practice-plans/{plan_id}/archive", h.archivePlan)
	mux.HandleFunc("POST /v1/practice-sessions", h.createSession)
	mux.HandleFunc("GET /v1/practice-sessions/resumable", h.getResumableSession)
	mux.HandleFunc("GET /v1/practice-sessions/{session_id}", h.getSession)
	mux.HandleFunc("GET /v1/practice-sessions/{session_id}/current-question", h.getCurrentQuestion)
	mux.HandleFunc("POST /v1/practice-sessions/{session_id}/activation", h.activateSession)
	mux.HandleFunc("POST /v1/practice-sessions/{session_id}/activate", h.activateSession)
	mux.HandleFunc("POST /v1/practice-sessions/{session_id}/text-answers", h.submitTextAnswer)
	mux.HandleFunc("POST /v1/practice-sessions/{session_id}/complete", h.completeSession)
	mux.HandleFunc("GET /v1/practice-sessions/{session_id}/evaluation", h.getSessionEvaluation)
	mux.HandleFunc("GET /v1/evaluation-reports/{report_id}", h.getEvaluationReport)
	mux.HandleFunc("GET /v1/evaluation-reports", h.listEvaluationReports)
}

func (h *HTTPHandler) getResumableSession(w http.ResponseWriter, r *http.Request) {
	actorID, ok := h.actorID(w, r)
	if !ok {
		return
	}
	session, err := h.sessions.GetLatestResumableSession(r.Context(), actorID)
	if errors.Is(err, practice.ErrSessionNotFound) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err != nil {
		h.writePracticeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"session": session})
}

func (h *HTTPHandler) getSessionEvaluation(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actorID(w, r)
	if !ok {
		return
	}
	if !evaluationIDPattern.MatchString(r.PathValue("session_id")) {
		h.writeEvaluationError(w, evaluation.ErrInvalidInput)
		return
	}
	if h.evaluations == nil {
		h.writeEvaluationError(w, errors.New("evaluation repository unavailable"))
		return
	}
	report, err := h.evaluations.FindBySession(r.Context(), actor, r.PathValue("session_id"))
	if err != nil {
		h.writeEvaluationError(w, err)
		return
	}
	if report.Status != evaluation.StatusReady {
		h.writePrivateJSON(w, http.StatusConflict, map[string]any{"report": report, "error": map[string]string{"code": "evaluation_report_not_ready", "message": "evaluation_report_not_ready"}})
		return
	}
	h.writePrivateJSON(w, http.StatusOK, map[string]any{"report": report})
}

func (h *HTTPHandler) getEvaluationReport(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actorID(w, r)
	if !ok {
		return
	}
	if h.evaluations == nil {
		h.writeEvaluationError(w, errors.New("evaluation repository unavailable"))
		return
	}
	if !evaluationIDPattern.MatchString(r.PathValue("report_id")) {
		h.writeEvaluationError(w, evaluation.ErrInvalidInput)
		return
	}
	report, err := h.evaluations.FindByID(r.Context(), actor, r.PathValue("report_id"))
	if err != nil {
		h.writeEvaluationError(w, err)
		return
	}
	if report.Status != evaluation.StatusReady {
		h.writePrivateJSON(w, http.StatusConflict, map[string]any{"report": report, "error": map[string]string{"code": "evaluation_report_not_ready", "message": "evaluation_report_not_ready"}})
		return
	}
	h.writePrivateJSON(w, http.StatusOK, map[string]any{"report": report})
}

func (h *HTTPHandler) listEvaluationReports(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actorID(w, r)
	if !ok {
		return
	}
	if h.evaluations == nil {
		h.writeEvaluationError(w, errors.New("evaluation repository unavailable"))
		return
	}
	q := r.URL.Query()
	limit := 20
	if raw := q.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 100 {
			h.writeEvaluationError(w, evaluation.ErrInvalidInput)
			return
		}
		limit = n
	}
	var cursor *evaluation.HistoryCursor
	if raw := q.Get("cursor"); raw != "" {
		c, err := evaluation.DecodeCursor(raw)
		if err != nil {
			h.writeEvaluationError(w, evaluation.ErrInvalidInput)
			return
		}
		cursor = &c
	}
	if len(q.Get("search")) > 200 {
		h.writeEvaluationError(w, evaluation.ErrInvalidInput)
		return
	}
	page, err := h.evaluations.List(r.Context(), evaluation.HistoryFilter{ActorID: actor, Limit: limit, Cursor: cursor, Search: q.Get("search")})
	if err != nil {
		h.writeEvaluationError(w, err)
		return
	}
	h.writePrivateJSON(w, http.StatusOK, page)
}

func (h *HTTPHandler) writeEvaluationError(w http.ResponseWriter, err error) {
	status, code := http.StatusInternalServerError, "internal_error"
	if errors.Is(err, evaluation.ErrInvalidInput) {
		status, code = http.StatusBadRequest, "invalid_request"
	}
	if errors.Is(err, evaluation.ErrNotFound) {
		status, code = http.StatusNotFound, "evaluation_report_not_found"
	}
	if status == http.StatusUnauthorized {
		w.Header().Set("WWW-Authenticate", "Bearer")
	}
	h.writePrivateJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": code}})
}

func (h *HTTPHandler) writePrivateJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, status, value)
}

func (h *HTTPHandler) createPlan(w http.ResponseWriter, r *http.Request) {
	actorID, ok := h.actorID(w, r)
	if !ok {
		return
	}
	var in practice.CreatePlanInput
	if !decodeCoachingJSON(w, r, &in) {
		return
	}
	plan, err := h.plans.CreatePlan(r.Context(), actorID, in)
	if err != nil {
		h.writePlanError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"plan": plan})
}
func (h *HTTPHandler) listPlans(w http.ResponseWriter, r *http.Request) {
	actorID, ok := h.actorID(w, r)
	if !ok {
		return
	}
	plans, err := h.plans.ListPlans(r.Context(), actorID)
	if err != nil {
		h.writePlanError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"plans": plans})
}
func (h *HTTPHandler) getPlan(w http.ResponseWriter, r *http.Request) {
	actorID, ok := h.actorID(w, r)
	if !ok {
		return
	}
	plan, err := h.plans.GetPlan(r.Context(), actorID, r.PathValue("plan_id"))
	if err != nil {
		h.writePlanError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"plan": plan})
}
func (h *HTTPHandler) archivePlan(w http.ResponseWriter, r *http.Request) {
	actorID, ok := h.actorID(w, r)
	if !ok {
		return
	}
	plan, err := h.plans.ArchivePlan(r.Context(), actorID, r.PathValue("plan_id"))
	if err != nil {
		h.writePlanError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"plan": plan})
}

func (h *HTTPHandler) createSession(w http.ResponseWriter, r *http.Request) {
	actorID, ok := h.actorID(w, r)
	if !ok {
		return
	}
	var in practice.CreateSessionInput
	if !decodeCoachingJSON(w, r, &in) {
		return
	}
	session, err := h.sessions.CreateSession(r.Context(), actorID, in)
	if err != nil {
		h.writePracticeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"session": session})
}

func (h *HTTPHandler) getSession(w http.ResponseWriter, r *http.Request) {
	actorID, ok := h.actorID(w, r)
	if !ok {
		return
	}
	session, err := h.sessions.GetSession(r.Context(), actorID, r.PathValue("session_id"))
	if err != nil {
		h.writePracticeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"session": session})
}

func (h *HTTPHandler) getCurrentQuestion(w http.ResponseWriter, r *http.Request) {
	actorID, ok := h.actorID(w, r)
	if !ok {
		return
	}
	question, err := h.sessions.GetCurrentQuestion(r.Context(), actorID, r.PathValue("session_id"))
	if err != nil {
		h.writePracticeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"question": question})
}

func (h *HTTPHandler) activateSession(w http.ResponseWriter, r *http.Request) {
	actorID, ok := h.actorID(w, r)
	if !ok {
		return
	}
	session, err := h.sessions.ActivateSession(r.Context(), actorID, r.PathValue("session_id"))
	if err != nil {
		h.writePracticeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"session": session})
}

func (h *HTTPHandler) submitTextAnswer(w http.ResponseWriter, r *http.Request) {
	actorID, ok := h.actorID(w, r)
	if !ok {
		return
	}
	var in practice.SubmitTextAnswerInput
	if !decodeCoachingJSON(w, r, &in) {
		return
	}
	turn, err := h.sessions.SubmitTextAnswer(r.Context(), actorID, r.PathValue("session_id"), in)
	if err != nil {
		h.writePracticeError(w, err)
		return
	}
	session, err := h.sessions.GetSession(r.Context(), actorID, r.PathValue("session_id"))
	if err != nil {
		h.writePracticeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"turn": turn, "session": session})
}

func (h *HTTPHandler) completeSession(w http.ResponseWriter, r *http.Request) {
	actorID, ok := h.actorID(w, r)
	if !ok {
		return
	}
	session, err := h.sessions.CompleteSession(r.Context(), actorID, r.PathValue("session_id"))
	if err != nil {
		h.writePracticeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"session": session})
}
func (h *HTTPHandler) actorID(w http.ResponseWriter, r *http.Request) (string, bool) {
	if h.auth == nil {
		h.writePlanError(w, identity.ErrUnauthorized)
		return "", false
	}
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		h.writePlanError(w, identity.ErrUnauthorized)
		return "", false
	}
	actor, err := h.auth.Authenticate(r.Context(), parts[1])
	if err != nil {
		h.writePlanError(w, identity.ErrUnauthorized)
		return "", false
	}
	return actor.UserID, true
}

func (h *HTTPHandler) writePlanError(w http.ResponseWriter, err error) {
	h.writeCoachingError(w, err)
}

func (h *HTTPHandler) writePracticeError(w http.ResponseWriter, err error) {
	h.writeCoachingError(w, err)
}

func (h *HTTPHandler) writeCoachingError(w http.ResponseWriter, err error) {
	status, code := http.StatusInternalServerError, "internal_error"
	switch {
	case errors.Is(err, identity.ErrUnauthorized):
		status, code = http.StatusUnauthorized, "authentication_required"
	case errors.Is(err, practice.ErrInvalidPlan):
		status, code = http.StatusBadRequest, "invalid_request"
	case errors.Is(err, practice.ErrPlanNotFound):
		status, code = http.StatusNotFound, "practice_plan_not_found"
	case errors.Is(err, practice.ErrPlanArchived):
		status, code = http.StatusConflict, "practice_plan_archived"
	case errors.Is(err, practice.ErrInvalidSession):
		status, code = http.StatusBadRequest, "invalid_request"
	case errors.Is(err, practice.ErrSessionNotFound):
		status, code = http.StatusNotFound, "practice_session_not_found"
	case errors.Is(err, practice.ErrSessionNotActive):
		status, code = http.StatusConflict, "practice_session_not_active"
	case errors.Is(err, practice.ErrInvalidSessionTransition):
		status, code = http.StatusConflict, "invalid_state_transition"
	case errors.Is(err, practice.ErrNoCurrentQuestion):
		status, code = http.StatusConflict, "practice_question_not_available"
	case errors.Is(err, practice.ErrInvalidAnswer):
		status, code = http.StatusBadRequest, "invalid_answer"
	case errors.Is(err, practice.ErrQuestionNotCurrent):
		status, code = http.StatusConflict, "question_not_current"
	case errors.Is(err, practice.ErrAnswerAlreadySubmitted):
		status, code = http.StatusConflict, "answer_already_submitted"
	case errors.Is(err, practice.ErrSessionHasPendingQuestion):
		status, code = http.StatusConflict, "practice_session_has_pending_questions"
	}
	if status == http.StatusUnauthorized {
		w.Header().Set("WWW-Authenticate", "Bearer")
	}
	writeJSON(w, status, map[string]any{"error": map[string]any{"code": code, "message": code}})
}

func decodeCoachingJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		h := map[string]string{"code": "invalid_request", "message": "invalid_request"}
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": h})
		return false
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	decoder.DisallowUnknownFields()
	if decoder.Decode(target) != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": map[string]string{"code": "invalid_request", "message": "invalid_request"}})
		return false
	}
	return true
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
