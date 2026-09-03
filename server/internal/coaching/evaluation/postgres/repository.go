package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gangcaiyoule/ai-speak/server/internal/coaching/evaluation"
)

type Repository struct{ db *sql.DB }

func New(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Create(ctx context.Context, report evaluation.Report) (evaluation.Report, error) {
	if r == nil || r.db == nil || !validReport(report) {
		return evaluation.Report{}, evaluation.ErrInvalidInput
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return evaluation.Report{}, fmt.Errorf("begin report transaction: %w", err)
	}
	defer tx.Rollback()
	if err = insertReport(ctx, tx, report); err != nil {
		return evaluation.Report{}, err
	}
	if err = insertQuestions(ctx, tx, report); err != nil {
		return evaluation.Report{}, err
	}
	if err = insertDimensions(ctx, tx, report); err != nil {
		return evaluation.Report{}, err
	}
	if err = insertFeedback(ctx, tx, report); err != nil {
		return evaluation.Report{}, err
	}
	if err = tx.Commit(); err != nil {
		return evaluation.Report{}, fmt.Errorf("commit report: %w", err)
	}
	return r.FindByID(ctx, report.ActorID, report.ID)
}

func validReport(report evaluation.Report) bool {
	return report.ActorID != "" && report.ID != "" && report.SessionID != "" && report.Version > 0 && report.Status == evaluation.StatusReady && report.CompletedAt != nil && report.Result != nil && report.Result.Valid()
}

func insertReport(ctx context.Context, tx *sql.Tx, report evaluation.Report) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO evaluation_reports
        (id, actor_id, session_id, status, schema_version, version, scene_type, practice_experience,
         scene_category, practice_mode, scoreability_status, summary, completed_at, created_at, updated_at)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$14)`, report.ID, report.ActorID,
		report.SessionID, report.Status, report.Result.SchemaVersion, report.Version, report.Result.SceneType,
		report.Result.PracticeExperience, report.Result.SceneCategory, report.Result.PracticeMode,
		report.Result.ScoreabilityStatus, report.Summary, report.CompletedAt, report.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert report: %w", err)
	}
	return nil
}

func insertQuestions(ctx context.Context, tx *sql.Tx, report evaluation.Report) error {
	for _, q := range report.Result.Questions {
		_, err := tx.ExecContext(ctx, `INSERT INTO evaluation_report_questions
            (id,report_id,source_question_id,parent_question_id,position,text) VALUES ($1,$2,$1,NULLIF($3,''),$4,$5)`, q.ID, report.ID, q.ParentQuestionID, q.Position, q.Text)
		if err != nil {
			return fmt.Errorf("insert report question: %w", err)
		}
		if q.Answer != nil {
			if _, err = tx.ExecContext(ctx, `INSERT INTO evaluation_report_answers
            (id,report_question_id,source_turn_id,transcript) VALUES ($1,$2,$1,$3)`, q.Answer.TurnID, q.ID, q.Answer.Transcript); err != nil {
				return fmt.Errorf("insert report answer: %w", err)
			}
		}
	}
	return nil
}

func insertDimensions(ctx context.Context, tx *sql.Tx, report evaluation.Report) error {
	for position, d := range report.Result.Dimensions {
		codes, _ := json.Marshal(d.ReasonCodes)
		var dimensionID string
		err := tx.QueryRowContext(ctx, `INSERT INTO evaluation_dimensions
            (id,report_id,dimension_key,score,scale,coverage,confidence,reason_codes,position)
            VALUES (md5($1 || ':' || $2)::uuid,$1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`, report.ID, d.Key, d.Score, d.Scale, d.Coverage, d.Confidence, codes, position+1).Scan(&dimensionID)
		if err != nil {
			return fmt.Errorf("insert dimension: %w", err)
		}
		groups := []struct {
			kind   string
			values []evaluation.Finding
		}{{"STRENGTH", d.Strengths}, {"IMPROVEMENT", d.Improvements}, {"RECOMMENDED_EXAMPLE", d.Examples}}
		for _, group := range groups {
			for i, finding := range group.values {
				if err = insertFinding(ctx, tx, report.ID, dimensionID, group.kind, i+1, finding); err != nil {
					return err
				}
			}
		}
	}
	for i, action := range report.Result.PriorityActions {
		_, err := tx.ExecContext(ctx, `INSERT INTO evaluation_priority_actions
        (report_id,dimension_id,finding_id,position) VALUES ($1,md5($1 || ':' || $2)::uuid,md5($1 || ':' || $3)::uuid,$4)`, report.ID, action.DimensionKey, action.FindingID, i+1)
		if err != nil {
			return fmt.Errorf("insert priority action: %w", err)
		}
	}
	return nil
}

func insertFinding(ctx context.Context, tx *sql.Tx, reportID, dimensionID, kind string, position int, finding evaluation.Finding) error {
	var findingID string
	err := tx.QueryRowContext(ctx, `INSERT INTO evaluation_findings
        (id,report_id,dimension_id,finding_key,kind,message,suggestion,position)
        VALUES (md5($1 || ':' || $2)::uuid,$1,$3,$2,$4,$5,NULLIF($6,''),$7) RETURNING id`, reportID, finding.ID, dimensionID, kind, finding.Message, finding.Suggestion, position).Scan(&findingID)
	if err != nil {
		return fmt.Errorf("insert finding: %w", err)
	}
	for i, evidence := range finding.Evidence {
		if err = insertEvidence(ctx, tx, reportID, evidence); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO evaluation_finding_evidence (finding_id,evidence_id,position) VALUES ($1,$2,$3)`, findingID, evidence.EvidenceRefID, i+1); err != nil {
			return fmt.Errorf("link finding evidence: %w", err)
		}
	}
	return nil
}

func insertEvidence(ctx context.Context, tx *sql.Tx, reportID string, evidence evaluation.Evidence) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO evaluation_evidence
        (id,report_id,source_turn_id,start_utf8_byte,end_utf8_byte,original_excerpt)
        VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT (id) DO NOTHING`, evidence.EvidenceRefID, reportID, evidence.TurnID, evidence.StartUTF8Byte, evidence.EndUTF8Byte, evidence.OriginalExcerpt)
	if err != nil {
		return fmt.Errorf("insert evidence: %w", err)
	}
	return nil
}

func insertFeedback(ctx context.Context, tx *sql.Tx, report evaluation.Report) error {
	for _, item := range report.Items {
		if !item.Valid() || item.EvaluationID != report.ID {
			return evaluation.ErrInvalidInput
		}
		if err := insertEvidence(ctx, tx, report.ID, item.Evidence); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO evaluation_feedback_items
            (id,report_id,evidence_id,position,category,severity,recommendation,correction,repractice_mode,created_at)
            VALUES ($1,$2,$3,$4,$5,NULLIF($6,''),$7,NULLIF($8,''),$9,$10)`, item.ID, report.ID, item.Evidence.EvidenceRefID, item.Position, item.Category, item.Severity, item.Recommendation, item.Correction, item.RepracticeMode, item.CreatedAt)
		if err != nil {
			return fmt.Errorf("insert feedback item: %w", err)
		}
	}
	return nil
}

func (r *Repository) FindByID(ctx context.Context, actorID, id string) (evaluation.Report, error) {
	return r.find(ctx, actorID, "r.id = $2", id)
}
func (r *Repository) FindBySession(ctx context.Context, actorID, sessionID string) (evaluation.Report, error) {
	return r.find(ctx, actorID, "r.session_id = $2", sessionID)
}

func (r *Repository) find(ctx context.Context, actorID, predicate, value string) (evaluation.Report, error) {
	if r == nil || r.db == nil || actorID == "" || value == "" {
		return evaluation.Report{}, evaluation.ErrInvalidInput
	}
	query := `SELECT r.id,r.actor_id,r.session_id,r.version,r.status,r.summary,r.schema_version,r.scene_type,
        r.practice_experience,r.scene_category,r.practice_mode,r.scoreability_status,r.completed_at,r.created_at
        FROM evaluation_reports r WHERE r.actor_id=$1 AND ` + predicate + ` ORDER BY r.version DESC LIMIT 1`
	var out evaluation.Report
	var formal evaluation.FormalReport
	var completed time.Time
	err := r.db.QueryRowContext(ctx, query, actorID, value).Scan(&out.ID, &out.ActorID, &out.SessionID, &out.Version, &out.Status, &out.Summary, &formal.SchemaVersion, &formal.SceneType, &formal.PracticeExperience, &formal.SceneCategory, &formal.PracticeMode, &formal.ScoreabilityStatus, &completed, &out.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return evaluation.Report{}, evaluation.ErrNotFound
	}
	if err != nil {
		return evaluation.Report{}, fmt.Errorf("find report: %w", err)
	}
	out.CompletedAt = &completed
	out.Result = &formal
	out.Result.Summary = out.Summary
	if err = loadQuestions(ctx, r.db, out.ID, out.Result); err != nil {
		return evaluation.Report{}, err
	}
	if err = loadDimensions(ctx, r.db, out.ID, out.Result); err != nil {
		return evaluation.Report{}, err
	}
	if err = loadFeedback(ctx, r.db, out.ID, &out); err != nil {
		return evaluation.Report{}, err
	}
	return out, nil
}

type queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func loadQuestions(ctx context.Context, q queryer, reportID string, formal *evaluation.FormalReport) error {
	rows, err := q.QueryContext(ctx, `SELECT q.source_question_id,q.position,COALESCE(q.parent_question_id::text,''),q.text,COALESCE(a.source_turn_id::text,''),COALESCE(a.transcript,'') FROM evaluation_report_questions q LEFT JOIN evaluation_report_answers a ON a.report_question_id=q.id WHERE q.report_id=$1 ORDER BY q.position`, reportID)
	if err != nil {
		return fmt.Errorf("load questions: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item evaluation.ReportQuestion
		var turn, transcript string
		if err = rows.Scan(&item.ID, &item.Position, &item.ParentQuestionID, &item.Text, &turn, &transcript); err != nil {
			return err
		}
		if turn != "" {
			item.Answer = &evaluation.ReportAnswer{TurnID: turn, Transcript: transcript}
		}
		formal.Questions = append(formal.Questions, item)
	}
	return rows.Err()
}

func loadDimensions(ctx context.Context, q queryer, reportID string, formal *evaluation.FormalReport) error {
	rows, err := q.QueryContext(ctx, `SELECT dimension_key,score,scale,coverage,confidence,reason_codes FROM evaluation_dimensions WHERE report_id=$1 ORDER BY position`, reportID)
	if err != nil {
		return fmt.Errorf("load dimensions: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var d evaluation.Dimension
		var score sql.NullFloat64
		var codes []byte
		if err = rows.Scan(&d.Key, &score, &d.Scale, &d.Coverage, &d.Confidence, &codes); err != nil {
			return err
		}
		if score.Valid {
			d.Score = &score.Float64
		}
		if err = json.Unmarshal(codes, &d.ReasonCodes); err != nil {
			return err
		}
		d.EvidenceRefs = []string{}
		d.Strengths = []evaluation.Finding{}
		d.Improvements = []evaluation.Finding{}
		d.Examples = []evaluation.Finding{}
		formal.Dimensions = append(formal.Dimensions, d)
	}
	if err = rows.Err(); err != nil {
		return err
	}
	if err = rows.Close(); err != nil {
		return err
	}
	indexes := make(map[string]int, len(formal.Dimensions))
	for i := range formal.Dimensions {
		indexes[formal.Dimensions[i].Key] = i
	}
	if err = loadFindings(ctx, q, reportID, formal, indexes); err != nil {
		return err
	}
	formal.PriorityActions = []evaluation.PriorityAction{}
	actions, err := q.QueryContext(ctx, `SELECT d.dimension_key,f.finding_key
        FROM evaluation_priority_actions a
        JOIN evaluation_dimensions d ON d.id=a.dimension_id
        JOIN evaluation_findings f ON f.id=a.finding_id
        WHERE a.report_id=$1 ORDER BY a.position`, reportID)
	if err != nil {
		return fmt.Errorf("load priority actions: %w", err)
	}
	defer actions.Close()
	for actions.Next() {
		var action evaluation.PriorityAction
		if err = actions.Scan(&action.DimensionKey, &action.FindingID); err != nil {
			return err
		}
		formal.PriorityActions = append(formal.PriorityActions, action)
	}
	return actions.Err()
}

func loadFindings(ctx context.Context, q queryer, reportID string, formal *evaluation.FormalReport, indexes map[string]int) error {
	rows, err := q.QueryContext(ctx, `SELECT d.dimension_key,f.finding_key,f.kind,f.message,
        COALESCE(f.suggestion,''),e.id,e.source_turn_id,e.start_utf8_byte,e.end_utf8_byte,e.original_excerpt
        FROM evaluation_findings f
        JOIN evaluation_dimensions d ON d.id=f.dimension_id
        LEFT JOIN evaluation_finding_evidence fe ON fe.finding_id=f.id
        LEFT JOIN evaluation_evidence e ON e.id=fe.evidence_id
        WHERE f.report_id=$1 ORDER BY d.position,f.kind,f.position,fe.position`, reportID)
	if err != nil {
		return fmt.Errorf("load findings: %w", err)
	}
	defer rows.Close()
	type location struct{ dimension, group, finding int }
	locations := map[string]location{}
	refs := map[string]map[string]bool{}
	for rows.Next() {
		var dimensionKey, findingKey, kind, message, suggestion string
		var evidenceID, turnID, excerpt sql.NullString
		var start, end sql.NullInt64
		if err = rows.Scan(&dimensionKey, &findingKey, &kind, &message, &suggestion, &evidenceID, &turnID, &start, &end, &excerpt); err != nil {
			return err
		}
		dimensionIndex, ok := indexes[dimensionKey]
		if !ok {
			return fmt.Errorf("finding references unknown dimension %q", dimensionKey)
		}
		groups := findingGroups(&formal.Dimensions[dimensionIndex])
		groupIndex := findingGroupIndex(kind)
		if groupIndex < 0 {
			return fmt.Errorf("unknown finding kind %q", kind)
		}
		key := dimensionKey + "\x00" + findingKey
		place, ok := locations[key]
		if !ok {
			*groups[groupIndex] = append(*groups[groupIndex], evaluation.Finding{ID: findingKey, Message: message, Suggestion: suggestion, Evidence: []evaluation.Evidence{}})
			place = location{dimensionIndex, groupIndex, len(*groups[groupIndex]) - 1}
			locations[key] = place
		}
		if evidenceID.Valid {
			finding := &(*findingGroups(&formal.Dimensions[place.dimension])[place.group])[place.finding]
			finding.Evidence = append(finding.Evidence, evaluation.Evidence{EvidenceRefID: evidenceID.String, TurnID: turnID.String, StartUTF8Byte: int(start.Int64), EndUTF8Byte: int(end.Int64), OriginalExcerpt: excerpt.String})
			if refs[dimensionKey] == nil {
				refs[dimensionKey] = map[string]bool{}
			}
			if !refs[dimensionKey][evidenceID.String] {
				formal.Dimensions[dimensionIndex].EvidenceRefs = append(formal.Dimensions[dimensionIndex].EvidenceRefs, evidenceID.String)
				refs[dimensionKey][evidenceID.String] = true
			}
		}
	}
	return rows.Err()
}

func findingGroups(dimension *evaluation.Dimension) [3]*[]evaluation.Finding {
	return [3]*[]evaluation.Finding{&dimension.Strengths, &dimension.Improvements, &dimension.Examples}
}

func findingGroupIndex(kind string) int {
	switch kind {
	case "STRENGTH":
		return 0
	case "IMPROVEMENT":
		return 1
	case "RECOMMENDED_EXAMPLE":
		return 2
	default:
		return -1
	}
}

func loadFeedback(ctx context.Context, q queryer, reportID string, out *evaluation.Report) error {
	rows, err := q.QueryContext(ctx, `SELECT f.id,f.position,f.category,COALESCE(f.severity,''),f.recommendation,COALESCE(f.correction,''),f.repractice_mode,f.created_at,e.id,e.source_turn_id,e.start_utf8_byte,e.end_utf8_byte,e.original_excerpt FROM evaluation_feedback_items f JOIN evaluation_evidence e ON e.id=f.evidence_id WHERE f.report_id=$1 ORDER BY f.position`, reportID)
	if err != nil {
		return fmt.Errorf("load feedback: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item evaluation.FeedbackItem
		item.EvaluationID = reportID
		if err = rows.Scan(&item.ID, &item.Position, &item.Category, &item.Severity, &item.Recommendation, &item.Correction, &item.RepracticeMode, &item.CreatedAt, &item.Evidence.EvidenceRefID, &item.Evidence.TurnID, &item.Evidence.StartUTF8Byte, &item.Evidence.EndUTF8Byte, &item.Evidence.OriginalExcerpt); err != nil {
			return err
		}
		out.Items = append(out.Items, item)
	}
	if out.Items == nil {
		out.Items = []evaluation.FeedbackItem{}
	}
	return rows.Err()
}

func (r *Repository) List(ctx context.Context, filter evaluation.HistoryFilter) (evaluation.HistoryPage, error) {
	if r == nil || r.db == nil || filter.ActorID == "" || filter.Limit < 1 || filter.Limit > 100 || len(filter.Search) > 200 {
		return evaluation.HistoryPage{}, evaluation.ErrInvalidInput
	}
	args := []any{filter.ActorID, filter.Limit + 1}
	where := `actor_id=$1 AND status='READY'`
	if filter.Cursor != nil {
		args = append(args, filter.Cursor.CompletedAt, filter.Cursor.ID)
		where += fmt.Sprintf(" AND (completed_at,id)<($%d,$%d)", len(args)-1, len(args))
	}
	if search := strings.TrimSpace(filter.Search); search != "" {
		args = append(args, "%"+strings.ReplaceAll(strings.ReplaceAll(search, "\\", "\\\\"), "%", "\\%")+"%")
		where += fmt.Sprintf(" AND summary ILIKE $%d ESCAPE '\\'", len(args))
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id,actor_id,session_id,version,status,summary,completed_at,created_at FROM evaluation_reports WHERE `+where+` ORDER BY completed_at DESC,id DESC LIMIT $2`, args...)
	if err != nil {
		return evaluation.HistoryPage{}, fmt.Errorf("list reports: %w", err)
	}
	defer rows.Close()
	page := evaluation.HistoryPage{Reports: []evaluation.Report{}}
	for rows.Next() {
		var item evaluation.Report
		var completed time.Time
		if err = rows.Scan(&item.ID, &item.ActorID, &item.SessionID, &item.Version, &item.Status, &item.Summary, &completed, &item.CreatedAt); err != nil {
			return evaluation.HistoryPage{}, err
		}
		item.CompletedAt = &completed
		item.Items = []evaluation.FeedbackItem{}
		page.Reports = append(page.Reports, item)
	}
	if err = rows.Err(); err != nil {
		return evaluation.HistoryPage{}, err
	}
	if len(page.Reports) > filter.Limit {
		last := page.Reports[filter.Limit-1]
		page.Reports = page.Reports[:filter.Limit]
		page.NextCursor, err = evaluation.EncodeCursor(evaluation.HistoryCursor{CompletedAt: *last.CompletedAt, ID: last.ID})
	}
	return page, err
}
