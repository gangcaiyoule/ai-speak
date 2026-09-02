// Package scene 定义口语练习场景目录及其读取能力。
package scene

import (
	"context"
	"errors"
)

var ErrSceneNotFound = errors.New("scene not found")

type PracticeExperience string

const (
	PracticeExperienceInterview     PracticeExperience = "INTERVIEW"
	PracticeExperienceIELTSSpeaking PracticeExperience = "IELTS_SPEAKING"
	PracticeExperienceWorkplace     PracticeExperience = "WORKPLACE"
	PracticeExperienceLifeAndTravel PracticeExperience = "LIFE_AND_TRAVEL"
)

type SceneCategory string

const (
	SceneCategoryInterviewRecruiter    SceneCategory = "INTERVIEW_RECRUITER"
	SceneCategoryInterviewBehavioral   SceneCategory = "INTERVIEW_BEHAVIORAL"
	SceneCategoryInterviewProfessional SceneCategory = "INTERVIEW_PROFESSIONAL"
	SceneCategoryWorkplaceGeneral      SceneCategory = "WORKPLACE_GENERAL"
	SceneCategoryLifeDaily             SceneCategory = "LIFE_DAILY"
)

type PracticeMode string

const (
	PracticeModeFullSimulation PracticeMode = "FULL_SIMULATION"
	PracticeModeFocus          PracticeMode = "FOCUS"
)

type Scene struct {
	ID                 string             `json:"scene_id"`
	PracticeExperience PracticeExperience `json:"practice_experience"`
	SceneCategory      SceneCategory      `json:"scene_category"`
	Name               string             `json:"name"`
	SceneVersion       int                `json:"scene_version"`
	Status             string             `json:"status"`
	Prompt             ScenePrompt        `json:"prompt"`
	Roles              []RoleDefinition   `json:"roles"`
	PracticeOptions    []PracticeOption   `json:"practice_options"`
}

type ScenePrompt struct {
	PublicSceneBrief string   `json:"public_scene_brief"`
	PracticeGoal     string   `json:"practice_goal"`
	UserRole         string   `json:"user_role"`
	AIRole           string   `json:"ai_role"`
	PersonaSummary   string   `json:"persona_summary"`
	FocusAreas       []string `json:"focus_areas"`
	TurnBlueprints   []string `json:"turn_blueprints"`
}

type RoleDefinition struct {
	ID                 string              `json:"role_definition_id"`
	SceneID            string              `json:"scene_id"`
	RoleType           string              `json:"role_type"`
	DisplayName        string              `json:"display_name"`
	Responsibilities   string              `json:"responsibilities"`
	Style              string              `json:"style"`
	PracticeObjectives []PracticeObjective `json:"practice_objectives"`
}

type PracticeObjective struct {
	ObjectiveID string `json:"objective_id"`
	Description string `json:"description"`
}

type PracticeOption struct {
	ID                       string       `json:"practice_option_id"`
	SceneID                  string       `json:"scene_id"`
	PracticeMode             PracticeMode `json:"practice_mode"`
	DisplayName              string       `json:"display_name"`
	SuggestedDurationSeconds int          `json:"suggested_duration_seconds"`
	TurnPolicyRef            string       `json:"turn_policy_ref"`
	SessionPolicyRef         string       `json:"session_policy_ref"`
	EvaluationPolicyRef      string       `json:"evaluation_policy_ref"`
	RoleDefinitionID         *string      `json:"role_definition_id,omitempty"`
}

type CatalogReader interface {
	ListScenes(context.Context) ([]Scene, error)
	GetScene(context.Context, string) (Scene, error)
	ListRoles(context.Context, string) ([]RoleDefinition, error)
}

type Catalog struct{ scenes []Scene }

func NewCatalog() *Catalog {
	return &Catalog{scenes: []Scene{
		{
			ID: "self-introduction", PracticeExperience: PracticeExperienceInterview,
			SceneCategory: SceneCategoryInterviewRecruiter, Name: "英文自我介绍", SceneVersion: 1, Status: "active",
			Prompt:          ScenePrompt{PublicSceneBrief: "练习用英文清晰介绍自己的经历。", PracticeGoal: "在两分钟内完成结构清晰的自我介绍。", UserRole: "求职者", AIRole: "面试官", PersonaSummary: "友好但会追问细节的招聘面试官。", FocusAreas: []string{"结构清晰", "重点突出"}, TurnBlueprints: []string{"请介绍你的背景和最近的经历。"}},
			Roles:           []RoleDefinition{{ID: "recruiter", SceneID: "self-introduction", RoleType: "INTERVIEWER", DisplayName: "招聘面试官", Responsibilities: "了解你的背景并判断岗位匹配度。", Style: "友好、结构化", PracticeObjectives: []PracticeObjective{{ObjectiveID: "clarity", Description: "清晰表达个人经历。"}}}},
			PracticeOptions: []PracticeOption{{ID: "self-introduction-full", SceneID: "self-introduction", PracticeMode: PracticeModeFullSimulation, DisplayName: "完整模拟", SuggestedDurationSeconds: 600, TurnPolicyRef: "interview.full.turn.v1", SessionPolicyRef: "interview.full.session.v1", EvaluationPolicyRef: "interview.full.evaluation.v1"}, {ID: "self-introduction-focus", SceneID: "self-introduction", PracticeMode: PracticeModeFocus, DisplayName: "重点练习", SuggestedDurationSeconds: 300, TurnPolicyRef: "interview.focus.turn.v1", SessionPolicyRef: "interview.focus.session.v1", EvaluationPolicyRef: "interview.focus.evaluation.v1", RoleDefinitionID: stringPtr("recruiter")}},
		},
		{
			ID: "project-deep-dive", PracticeExperience: PracticeExperienceInterview,
			SceneCategory: SceneCategoryInterviewProfessional, Name: "项目经历深挖", SceneVersion: 1, Status: "active",
			Prompt:          ScenePrompt{PublicSceneBrief: "围绕一个真实项目讨论决策与结果。", PracticeGoal: "清楚解释技术取舍、个人贡献和项目结果。", UserRole: "候选人", AIRole: "技术面试官", PersonaSummary: "重视证据、会追问技术细节的面试官。", FocusAreas: []string{"技术取舍", "个人贡献", "结果量化"}, TurnBlueprints: []string{"请介绍一个你负责的项目。"}},
			Roles:           []RoleDefinition{{ID: "technical-interviewer", SceneID: "project-deep-dive", RoleType: "TECHNICAL_INTERVIEWER", DisplayName: "技术面试官", Responsibilities: "追问项目中的工程决策和个人贡献。", Style: "直接、重视证据", PracticeObjectives: []PracticeObjective{{ObjectiveID: "tradeoffs", Description: "解释技术取舍。"}}}},
			PracticeOptions: []PracticeOption{{ID: "project-deep-dive-full", SceneID: "project-deep-dive", PracticeMode: PracticeModeFullSimulation, DisplayName: "完整模拟", SuggestedDurationSeconds: 900, TurnPolicyRef: "interview.full.turn.v1", SessionPolicyRef: "interview.full.session.v1", EvaluationPolicyRef: "interview.full.evaluation.v1"}, {ID: "project-deep-dive-focus", SceneID: "project-deep-dive", PracticeMode: PracticeModeFocus, DisplayName: "重点练习", SuggestedDurationSeconds: 420, TurnPolicyRef: "interview.focus.turn.v1", SessionPolicyRef: "interview.focus.session.v1", EvaluationPolicyRef: "interview.focus.evaluation.v1", RoleDefinitionID: stringPtr("technical-interviewer")}},
		},
	}}
}

func stringPtr(value string) *string { return &value }

func (c *Catalog) ListScenes(ctx context.Context) ([]Scene, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	result := make([]Scene, 0, len(c.scenes))
	for _, value := range c.scenes {
		result = append(result, cloneScene(value))
	}
	return result, nil
}

func (c *Catalog) GetScene(ctx context.Context, sceneID string) (Scene, error) {
	if err := contextError(ctx); err != nil {
		return Scene{}, err
	}
	for _, value := range c.scenes {
		if value.ID == sceneID && value.Status == "active" {
			return cloneScene(value), nil
		}
	}
	return Scene{}, ErrSceneNotFound
}

func (c *Catalog) ListRoles(ctx context.Context, sceneID string) ([]RoleDefinition, error) {
	value, err := c.GetScene(ctx, sceneID)
	if err != nil {
		return nil, err
	}
	return append([]RoleDefinition(nil), value.Roles...), nil
}

func contextError(ctx context.Context) error {
	if ctx == nil {
		return errors.New("scene context is required")
	}
	return ctx.Err()
}

func cloneScene(source Scene) Scene {
	result := source
	result.Prompt.FocusAreas = append([]string(nil), source.Prompt.FocusAreas...)
	result.Roles = append([]RoleDefinition(nil), source.Roles...)
	result.PracticeOptions = append([]PracticeOption(nil), source.PracticeOptions...)
	return result
}
