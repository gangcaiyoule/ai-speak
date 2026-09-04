package evaluation

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	ErrInvalidTranscript = errors.New("invalid evaluation transcript")
	ErrInsufficientData  = errors.New("insufficient transcript data")
)

type Transcript struct {
	ActorID   string
	SessionID string
	ReportID  string
	Version   int
	Questions []ReportQuestion
}
type Scorer interface {
	Evaluate(context.Context, Transcript) (FormalReport, []FeedbackItem, error)
}
type Generator struct {
	Reports Repository
	Scorer  Scorer
	Now     func() time.Time
}

func (g Generator) Generate(ctx context.Context, input Transcript) (Report, error) {
	if g.Reports == nil || g.Scorer == nil || input.ActorID == "" || input.SessionID == "" || input.ReportID == "" || input.Version < 1 || len(input.Questions) == 0 {
		return Report{}, ErrInvalidTranscript
	}
	answered := 0
	for _, q := range input.Questions {
		if q.Answer != nil {
			answered++
		}
	}
	now := time.Now
	if g.Now != nil {
		now = g.Now
	}
	if answered == 0 {
		return g.Reports.Create(ctx, insufficientReport(input, now()))
	}
	formal, items, err := g.Scorer.Evaluate(ctx, input)
	if err != nil {
		return Report{ID: input.ReportID, ActorID: input.ActorID, SessionID: input.SessionID, Version: input.Version, Status: StatusFailed, CreatedAt: now()}, fmt.Errorf("score transcript: %w", err)
	}
	if !formal.Valid() || len(items) > 128 {
		return Report{}, fmt.Errorf("score transcript: %w", ErrInvalidTranscript)
	}
	for _, item := range items {
		if item.EvaluationID != input.ReportID || !item.Valid() {
			return Report{}, fmt.Errorf("score transcript feedback: %w", ErrInvalidTranscript)
		}
	}
	completed := now()
	report := Report{ID: input.ReportID, ActorID: input.ActorID, SessionID: input.SessionID, Version: input.Version, Status: StatusReady, Summary: formal.Summary, Result: &formal, Items: items, CompletedAt: &completed, CreatedAt: completed}
	return g.Reports.Create(ctx, report)
}

func insufficientReport(input Transcript, now time.Time) Report {
	formal := FormalReport{SchemaVersion: FormalReportSchemaVersion, SceneType: SceneInterview, PracticeExperience: "UNKNOWN", SceneCategory: "UNKNOWN", PracticeMode: "UNKNOWN", ScoreabilityStatus: ScoreabilityInsufficient, Summary: "Not enough answered turns to produce reliable scores.", Questions: input.Questions, Dimensions: []Dimension{{Key: "overall", Scale: ScoreScalePercentage100, Coverage: 0, Confidence: 0, ReasonCodes: []string{"INSUFFICIENT_DATA"}, EvidenceRefs: []string{}, Strengths: []Finding{}, Improvements: []Finding{}, Examples: []Finding{}}}, PriorityActions: []PriorityAction{}}
	return Report{ID: input.ReportID, ActorID: input.ActorID, SessionID: input.SessionID, Version: input.Version, Status: StatusReady, Summary: formal.Summary, Result: &formal, Items: []FeedbackItem{}, CompletedAt: &now, CreatedAt: now}
}
