package evaluation

import (
	"context"
	"errors"
	"testing"
	"time"
)

type generatorRepo struct{ got Report }

func (r *generatorRepo) Create(_ context.Context, report Report) (Report, error) {
	r.got = report
	return report, nil
}
func (*generatorRepo) FindByID(context.Context, string, string) (Report, error) { return Report{}, nil }
func (*generatorRepo) FindBySession(context.Context, string, string) (Report, error) {
	return Report{}, nil
}
func (*generatorRepo) List(context.Context, HistoryFilter) (HistoryPage, error) {
	return HistoryPage{}, nil
}

type generatorScorer struct {
	report FormalReport
	err    error
}

func (s generatorScorer) Evaluate(context.Context, Transcript) (FormalReport, []FeedbackItem, error) {
	return s.report, nil, s.err
}

const generatorID = "11111111-1111-4111-8111-111111111111"
const generatorTurn = "22222222-2222-4222-8222-222222222222"

func generatorInput(answered bool) Transcript {
	q := ReportQuestion{ID: generatorID, Position: 1, Text: "Tell me about yourself."}
	if answered {
		q.Answer = &ReportAnswer{TurnID: generatorTurn, Transcript: "I lead projects."}
	}
	return Transcript{ActorID: generatorID, SessionID: generatorID, ReportID: generatorID, Version: 1, Questions: []ReportQuestion{q}}
}
func validGeneratorReport() FormalReport {
	score := 80.0
	return FormalReport{SchemaVersion: FormalReportSchemaVersion, SceneType: SceneInterview, PracticeExperience: "INTERVIEW", SceneCategory: "GENERAL", PracticeMode: "FULL", ScoreabilityStatus: ScoreabilityProvisional, Summary: "Good answer.", Questions: []ReportQuestion{{ID: generatorID, Position: 1, Text: "Tell me about yourself.", Answer: &ReportAnswer{TurnID: generatorTurn, Transcript: "I lead projects."}}}, Dimensions: []Dimension{{Key: "content", Score: &score, Scale: ScoreScalePercentage100, Coverage: 1, Confidence: .9, ReasonCodes: []string{}, EvidenceRefs: []string{generatorTurn}, Strengths: []Finding{}, Improvements: []Finding{}, Examples: []Finding{}}}, PriorityActions: []PriorityAction{}}
}
func TestGeneratorPersistsReadyReport(t *testing.T) {
	repo := &generatorRepo{}
	got, err := (Generator{Reports: repo, Scorer: generatorScorer{report: validGeneratorReport()}}).Generate(context.Background(), generatorInput(true))
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != StatusReady || repo.got.ID != generatorID || got.CompletedAt == nil {
		t.Fatalf("unexpected report: %#v", got)
	}
}
func TestGeneratorCreatesInsufficientReportWithoutScoring(t *testing.T) {
	repo := &generatorRepo{}
	got, err := (Generator{Reports: repo, Scorer: generatorScorer{err: errors.New("must not run")}}).Generate(context.Background(), generatorInput(false))
	if err != nil {
		t.Fatal(err)
	}
	if got.Result == nil || got.Result.ScoreabilityStatus != ScoreabilityInsufficient {
		t.Fatalf("unexpected report: %#v", got)
	}
}
func TestGeneratorDoesNotPersistInvalidScoring(t *testing.T) {
	repo := &generatorRepo{}
	report := validGeneratorReport()
	invalid := 101.0
	report.Dimensions[0].Score = &invalid
	if _, err := (Generator{Reports: repo, Scorer: generatorScorer{report: report}}).Generate(context.Background(), generatorInput(true)); !errors.Is(err, ErrInvalidTranscript) {
		t.Fatalf("error=%v", err)
	}
	if repo.got.ID != "" {
		t.Fatal("invalid report was persisted")
	}
	_ = time.Time{}
}
