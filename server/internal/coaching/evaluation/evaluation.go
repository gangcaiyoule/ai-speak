// Package evaluation defines the review contracts shared by later storage,
// HTTP, and Flutter work. It deliberately has no transport dependency.
package evaluation

import (
	"context"
	"math"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

const FormalReportSchemaVersion = "evaluation-report/v2"

var (
	uuidPattern       = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)
	identifierPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
)

type Status string
const ( StatusQueued Status = "QUEUED"; StatusRunning Status = "RUNNING"; StatusReady Status = "READY"; StatusFailed Status = "FAILED" )
type SceneType string
const ( SceneIELTSSpeaking SceneType = "IELTS_SPEAKING"; SceneInterview SceneType = "INTERVIEW"; SceneOverseasDaily SceneType = "OVERSEAS_DAILY_LIFE"; SceneOverseasWorkplace SceneType = "OVERSEAS_WORKPLACE" )
type Scoreability string
const ( ScoreabilityProvisional Scoreability = "PROVISIONAL"; ScoreabilityInsufficient Scoreability = "INSUFFICIENT" )
type ScoreScale string
const ( ScoreScalePercentage100 ScoreScale = "PERCENTAGE_100"; ScoreScaleIELTSBand ScoreScale = "IELTS_BAND_9" )
type FeedbackCategory string
const ( FeedbackCorrection FeedbackCategory = "CORRECTION"; FeedbackStrength FeedbackCategory = "STRENGTH"; FeedbackRecommendedExpression FeedbackCategory = "RECOMMENDED_EXPRESSION" )
type RepracticeMode string
const ( RepracticeNone RepracticeMode = "NONE"; RepracticeSameQuestion RepracticeMode = "SAME_QUESTION" )

type Report struct { ID string `json:"id"`; SessionID string `json:"session_id"`; Status Status `json:"status"`; Summary string `json:"summary"`; Result *FormalReport `json:"result,omitempty"`; Items []FeedbackItem `json:"items"` }
type FormalReport struct { SchemaVersion string `json:"schema_version"`; SceneType SceneType `json:"scene_type"`; PracticeExperience string `json:"practice_experience"`; SceneCategory string `json:"scene_category"`; PracticeMode string `json:"practice_mode"`; ScoreabilityStatus Scoreability `json:"scoreability_status"`; Summary string `json:"summary"`; Questions []ReportQuestion `json:"questions"`; Dimensions []Dimension `json:"dimensions"`; PriorityActions []PriorityAction `json:"priority_actions"` }
type ReportQuestion struct { ID string `json:"question_id"`; Position int `json:"position"`; ParentQuestionID string `json:"parent_question_id,omitempty"`; Text string `json:"text"`; Answer *ReportAnswer `json:"answer"` }
type ReportAnswer struct { TurnID string `json:"turn_id"`; Transcript string `json:"transcript"` }
type Dimension struct { Key string `json:"key"`; Score *float64 `json:"score"`; Scale ScoreScale `json:"scale"`; Coverage float64 `json:"coverage"`; Confidence float64 `json:"confidence"`; ReasonCodes []string `json:"reason_codes"`; EvidenceRefs []string `json:"evidence_ref_ids"`; Strengths []Finding `json:"strengths"`; Improvements []Finding `json:"improvements"`; Examples []Finding `json:"recommended_examples"` }
type Finding struct { ID string `json:"finding_id"`; Message string `json:"message"`; Suggestion string `json:"suggestion,omitempty"`; Evidence []Evidence `json:"evidence"` }
type Evidence struct { EvidenceRefID string `json:"evidence_ref_id"`; TurnID string `json:"turn_id"`; StartUTF8Byte int `json:"start_utf8_byte"`; EndUTF8Byte int `json:"end_utf8_byte"`; OriginalExcerpt string `json:"original_excerpt"` }
type PriorityAction struct { DimensionKey string `json:"dimension_key"`; FindingID string `json:"finding_id"` }

// FeedbackItem is an evidence-backed review observation. RepracticeMode is
// consumed by the later same-question retry endpoint.
type FeedbackItem struct { ID string `json:"feedback_item_id"`; EvaluationID string `json:"evaluation_id"`; Position int `json:"position"`; Category FeedbackCategory `json:"category"`; Severity string `json:"severity,omitempty"`; Evidence Evidence `json:"evidence"`; Recommendation string `json:"recommendation"`; Correction string `json:"correction,omitempty"`; RepracticeMode RepracticeMode `json:"repractice_mode"`; CreatedAt time.Time `json:"created_at"` }
type RetryTurnRequest struct { FeedbackItemID string; IdempotencyKey string }

func (value FormalReport) Valid() bool {
	if value.SchemaVersion != FormalReportSchemaVersion || !validScene(value.SceneType) || !validIdentifier(value.PracticeExperience) || !validIdentifier(value.SceneCategory) || !validIdentifier(value.PracticeMode) || !validScoreability(value.ScoreabilityStatus) || !validText(value.Summary, 2048) || len(value.Questions) == 0 || len(value.Questions) > 128 || len(value.Dimensions) == 0 || len(value.Dimensions) > 8 || value.PriorityActions == nil || len(value.PriorityActions) > 5 { return false }
	questions, turns, dimensions, findings, improvements := map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]string{}
	for _, question := range value.Questions {
		if !validUUID(question.ID) || question.Position < 1 || !validText(question.Text, 16384) || questions[question.ID] || (question.ParentQuestionID != "" && question.ParentQuestionID == question.ID) { return false }
		questions[question.ID] = true
		if question.Answer != nil { if !validUUID(question.Answer.TurnID) || !validText(question.Answer.Transcript, 65536) || turns[question.Answer.TurnID] { return false }; turns[question.Answer.TurnID] = true }
	}
	for _, question := range value.Questions { if question.ParentQuestionID != "" && !questions[question.ParentQuestionID] { return false } }
	for _, dimension := range value.Dimensions {
		if !dimension.valid(value.ScoreabilityStatus) || dimensions[dimension.Key] { return false }; dimensions[dimension.Key] = true
		for _, group := range [][]Finding{dimension.Strengths, dimension.Improvements, dimension.Examples} { for _, finding := range group { if !finding.valid(turns) || findings[finding.ID] { return false }; findings[finding.ID] = true } }
		for _, finding := range dimension.Improvements { improvements[finding.ID] = dimension.Key }
	}
	actions := map[string]bool{}
	for _, action := range value.PriorityActions { key := action.DimensionKey + "\x00" + action.FindingID; if improvements[action.FindingID] != action.DimensionKey || actions[key] { return false }; actions[key] = true }
	return true
}

func (item FeedbackItem) Valid() bool {
	if !validUUID(item.ID) || !validUUID(item.EvaluationID) || item.Position < 1 || !item.Evidence.valid(nil) || !validText(item.Recommendation, 4096) || item.CreatedAt.IsZero() { return false }
	if item.Category == FeedbackStrength { return item.RepracticeMode == RepracticeNone && item.Correction == "" }
	return (item.Category == FeedbackCorrection || item.Category == FeedbackRecommendedExpression) && validText(item.Correction, 4096) && (item.RepracticeMode == RepracticeNone || item.RepracticeMode == RepracticeSameQuestion)
}
func (value Dimension) valid(scoreability Scoreability) bool {
	if !validIdentifier(value.Key) || (value.Scale != ScoreScalePercentage100 && value.Scale != ScoreScaleIELTSBand) || !ratio(value.Coverage) || !ratio(value.Confidence) || value.ReasonCodes == nil || value.EvidenceRefs == nil || value.Strengths == nil || value.Improvements == nil || value.Examples == nil || len(value.ReasonCodes) > 8 || len(value.EvidenceRefs) > 128 || (scoreability == ScoreabilityInsufficient && value.Score != nil) { return false }
	if value.Score != nil && (math.IsNaN(*value.Score) || math.IsInf(*value.Score, 0) || *value.Score < 0 || (value.Scale == ScoreScalePercentage100 && *value.Score > 100) || (value.Scale == ScoreScaleIELTSBand && (*value.Score > 9 || math.Mod(*value.Score*2, 1) != 0))) { return false }
	for _, code := range value.ReasonCodes { if !validIdentifier(code) { return false } }; for _, ref := range value.EvidenceRefs { if !validUUID(ref) { return false } }
	return len(value.Strengths) <= 5 && len(value.Improvements) <= 5 && len(value.Examples) <= 5
}
func (value Finding) valid(turns map[string]bool) bool { if !validIdentifier(value.ID) || !validText(value.Message, 2048) || (value.Suggestion != "" && !validText(value.Suggestion, 2048)) || value.Evidence == nil || len(value.Evidence) > 8 { return false }; for _, evidence := range value.Evidence { if !evidence.valid(turns) { return false } }; return true }
func (value Evidence) valid(turns map[string]bool) bool { return validUUID(value.EvidenceRefID) && validUUID(value.TurnID) && value.EvidenceRefID == value.TurnID && (turns == nil || turns[value.TurnID]) && value.StartUTF8Byte >= 0 && value.EndUTF8Byte > value.StartUTF8Byte && value.EndUTF8Byte-value.StartUTF8Byte == len(value.OriginalExcerpt) && validText(value.OriginalExcerpt, 16384) }
func validUUID(value string) bool { return uuidPattern.MatchString(value) }
func validIdentifier(value string) bool { return identifierPattern.MatchString(value) }
func validText(value string, maximum int) bool { return utf8.ValidString(value) && value == strings.TrimSpace(value) && value != "" && len(value) <= maximum && !strings.ContainsRune(value, '\x00') }
func ratio(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0 && value <= 1 }
func validScene(value SceneType) bool { return value == SceneIELTSSpeaking || value == SceneInterview || value == SceneOverseasDaily || value == SceneOverseasWorkplace }
func validScoreability(value Scoreability) bool { return value == ScoreabilityProvisional || value == ScoreabilityInsufficient }

type Repository interface { Create(context.Context, Report) (Report, error); FindByID(context.Context, string) (Report, error); FindBySession(context.Context, string) (Report, error) }
type Service interface { CreateForSession(context.Context, string) (Report, error); GetByID(context.Context, string) (Report, error); GetBySession(context.Context, string) (Report, error) }
