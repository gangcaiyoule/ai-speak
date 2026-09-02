package evaluation

import (
	"testing"
	"time"
)

const (
	testID1 = "11111111-1111-4111-8111-111111111111"
	testID2 = "22222222-2222-4222-8222-222222222222"
	testID3 = "33333333-3333-4333-8333-333333333333"
)

func validReport() FormalReport {
	score := 82.0
	return FormalReport{
		SchemaVersion: FormalReportSchemaVersion, SceneType: SceneInterview,
		PracticeExperience: "INTERVIEW", SceneCategory: "INTERVIEW_RECRUITER", PracticeMode: "FULL_SIMULATION",
		ScoreabilityStatus: ScoreabilityProvisional, Summary: "Clear answer with one improvement.",
		Questions: []ReportQuestion{{ID: testID1, Position: 1, Text: "Tell me about yourself.", Answer: &ReportAnswer{TurnID: testID2, Transcript: "I lead backend projects."}}},
		Dimensions: []Dimension{{Key: "content", Score: &score, Scale: ScoreScalePercentage100, Coverage: 1, Confidence: 0.9, ReasonCodes: []string{}, EvidenceRefs: []string{testID2}, Strengths: []Finding{}, Improvements: []Finding{{ID: "content.clarity", Message: "Add a measurable result.", Suggestion: "State the result first.", Evidence: []Evidence{{EvidenceRefID: testID2, TurnID: testID2, StartUTF8Byte: 0, EndUTF8Byte: 1, OriginalExcerpt: "I"}}}}, Examples: []Finding{}}},
		PriorityActions: []PriorityAction{{DimensionKey: "content", FindingID: "content.clarity"}},
	}
}

func TestFormalReportValid(t *testing.T) {
	if !validReport().Valid() { t.Fatal("valid report rejected") }
}

func TestFormalReportRejectsInvalidEvidenceAndInsufficientScore(t *testing.T) {
	report := validReport()
	report.Dimensions[0].Improvements[0].Evidence[0].TurnID = testID3
	if report.Valid() { t.Fatal("report accepted evidence for an unanswered turn") }
	report = validReport()
	report.ScoreabilityStatus = ScoreabilityInsufficient
	if report.Valid() { t.Fatal("insufficient report accepted a score") }
}

func TestFeedbackItemValid(t *testing.T) {
	item := FeedbackItem{ID: testID1, EvaluationID: testID2, Position: 1, Category: FeedbackCorrection, Evidence: Evidence{EvidenceRefID: testID3, TurnID: testID3, StartUTF8Byte: 0, EndUTF8Byte: 1, OriginalExcerpt: "I"}, Recommendation: "Use a measurable result.", Correction: "I reduced deployment time by 20%.", RepracticeMode: RepracticeSameQuestion, CreatedAt: time.Now()}
	if !item.Valid() { t.Fatal("valid correction feedback rejected") }
	item.Category, item.RepracticeMode = FeedbackStrength, RepracticeSameQuestion
	if item.Valid() { t.Fatal("strength feedback accepted a retry mode") }
}
