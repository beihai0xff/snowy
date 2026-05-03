// Package generative implements Snowy v2 large-model-first modeling contracts.
package generative

import (
	"time"

	"github.com/google/uuid"
)

const (
	DomainAuto    = "auto"
	DomainPhysics = "physics"
	DomainBiology = "biology"

	GradeBandHighSchool = "high_school"

	TargetModeInteractive = "interactive_model"
	TargetModeReview      = "review"
	TargetModeExplain     = "explain"
)

// CompileRequest is the public v2 modeling compile request.
type CompileRequest struct {
	SessionID  uuid.UUID      `json:"session_id,omitempty"`
	UserID     uuid.UUID      `json:"-"`
	Message    string         `json:"message"`
	Domain     string         `json:"domain,omitempty"`
	GradeBand  string         `json:"grade_band,omitempty"`
	TargetMode string         `json:"target_mode,omitempty"`
	Context    CompileContext `json:"context,omitempty"`
}

// CompileContext carries grounding and page context from search/modeling flows.
type CompileContext struct {
	Citations     []EvidenceRef `json:"citations,omitempty"`
	KnowledgeTags []string      `json:"knowledge_tags,omitempty"`
	SourcePage    string        `json:"source_page,omitempty"`
	UserNotes     string        `json:"user_notes,omitempty"`
}

// GenerativeModelPackage is the persisted v2 modeling artifact.
type GenerativeModelPackage struct {
	PackageID          uuid.UUID                    `json:"package_id"`
	SessionID          uuid.UUID                    `json:"session_id,omitempty"`
	UserID             uuid.UUID                    `json:"-"`
	Domain             string                       `json:"domain"`
	Question           string                       `json:"question"`
	LearningModel      LearningModelSpec            `json:"learning_model"`
	EvidenceRefs       []EvidenceRef                `json:"evidence_refs"`
	ReasoningTrace     ReasoningTrace               `json:"reasoning_trace"`
	GenerativeModel    GenerativeModelSpec          `json:"generative_model"`
	SimulationLogic    *DynamicSimulationSpec       `json:"simulation_logic,omitempty"`
	VisualizationGraph *GenerativeVisualizationSpec `json:"visualization_graph,omitempty"`
	InteractionPlan    InteractionPlan              `json:"interaction_plan"`
	AssessmentTasks    []AssessmentTask             `json:"assessment_tasks"`
	ValidationReport   ModelValidationReport        `json:"validation_report"`
	RegenerationHints  []RegenerationHint           `json:"regeneration_hints,omitempty"`
	Warnings           []string                     `json:"warnings,omitempty"`
	Confidence         float64                      `json:"confidence"`
	ModelName          string                       `json:"model_name,omitempty"`
	Status             string                       `json:"status"`
	FallbackReason     string                       `json:"fallback_reason,omitempty"`
	CreatedAt          time.Time                    `json:"created_at"`
}

type LearningModelSpec struct {
	ID            string   `json:"id,omitempty"`
	Domain        string   `json:"domain"`
	GradeBand     string   `json:"grade_band"`
	Topic         string   `json:"topic"`
	LearningGoal  string   `json:"learning_goal"`
	KnowledgeTags []string `json:"knowledge_tags,omitempty"`
	Difficulty    string   `json:"difficulty,omitempty"`
}

type EvidenceRef struct {
	DocID         string   `json:"doc_id"`
	SourceType    string   `json:"source_type"`
	Title         string   `json:"title,omitempty"`
	Chapter       string   `json:"chapter,omitempty"`
	Snippet       string   `json:"snippet"`
	KnowledgeTags []string `json:"knowledge_tags,omitempty"`
	Confidence    float64  `json:"confidence"`
}

type ReasoningTrace struct {
	Summary      string   `json:"summary"`
	EvidenceUsed []string `json:"evidence_used,omitempty"`
	Assumptions  []string `json:"assumptions,omitempty"`
	KeySteps     []string `json:"key_steps,omitempty"`
	Confidence   float64  `json:"confidence"`
}

type GenerativeModelSpec struct {
	ID            string         `json:"id,omitempty"`
	Domain        string         `json:"domain"`
	GradeBand     string         `json:"grade_band"`
	Topic         string         `json:"topic"`
	LearningGoal  string         `json:"learning_goal"`
	KnowledgeTags []string       `json:"knowledge_tags,omitempty"`
	Entities      []EntitySpec   `json:"entities,omitempty"`
	Variables     []VariableSpec `json:"variables,omitempty"`
	Relations     []RelationSpec `json:"relations,omitempty"`
}

type EntitySpec struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type VariableSpec struct {
	Name    string  `json:"name"`
	Label   string  `json:"label"`
	Unit    string  `json:"unit,omitempty"`
	Default float64 `json:"default"`
	Min     float64 `json:"min"`
	Max     float64 `json:"max"`
	Step    float64 `json:"step,omitempty"`
}

type RelationSpec struct {
	Source      string `json:"source"`
	Target      string `json:"target"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	Condition   string `json:"condition,omitempty"`
}

type DynamicSimulationSpec struct {
	SimulationType        string             `json:"simulation_type"`
	Runtime               string             `json:"runtime"`
	Assumptions           []string           `json:"assumptions,omitempty"`
	StateVariables        []string           `json:"state_variables,omitempty"`
	Variables             []VariableSpec     `json:"variables,omitempty"`
	Formulas              []FormulaSpec      `json:"formulas,omitempty"`
	Vectors               []VectorSpec       `json:"vectors,omitempty"`
	Curves                []CurveSpec        `json:"curves,omitempty"`
	Outcomes              []OutcomeSpec      `json:"outcomes,omitempty"`
	RenderInstructions    RenderInstructions `json:"render_instructions"`
	LocalRecomputeAllowed bool               `json:"local_recompute_allowed"`
	RegenerateWhen        []string           `json:"regenerate_when,omitempty"`
}

type FormulaSpec struct {
	ID      string `json:"id"`
	Expr    string `json:"expr"`
	Meaning string `json:"meaning"`
}

type VectorSpec struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Origin  string `json:"origin,omitempty"`
	XExpr   string `json:"x_expr,omitempty"`
	YExpr   string `json:"y_expr,omitempty"`
	Meaning string `json:"meaning,omitempty"`
}

type CurveSpec struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	XLabel  string `json:"x_label,omitempty"`
	YLabel  string `json:"y_label,omitempty"`
	YExpr   string `json:"y_expr,omitempty"`
	Meaning string `json:"meaning,omitempty"`
}

type OutcomeSpec struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Expr        string `json:"expr,omitempty"`
	Unit        string `json:"unit,omitempty"`
	Description string `json:"description,omitempty"`
}

type RenderInstructions struct {
	CoordinateSystem string   `json:"coordinate_system,omitempty"`
	Layers           []string `json:"layers,omitempty"`
	Annotations      []string `json:"annotations,omitempty"`
}

type GenerativeVisualizationSpec struct {
	VisualizationType   string               `json:"visualization_type"`
	Topic               string               `json:"topic"`
	Nodes               []VisualizationNode  `json:"nodes,omitempty"`
	Edges               []VisualizationEdge  `json:"edges,omitempty"`
	ProcessSteps        []VisualizationStep  `json:"process_steps,omitempty"`
	ExperimentVariables *ExperimentVariables `json:"experiment_variables,omitempty"`
	CurveExplanation    string               `json:"curve_explanation,omitempty"`
	LimitingFactors     []string             `json:"limiting_factors,omitempty"`
	MechanismStages     []MechanismStage     `json:"mechanism_stages,omitempty"`
	VariableEffects     []VariableEffect     `json:"variable_effects,omitempty"`
}

type VisualizationNode struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Type  string `json:"type"`
}

type VisualizationEdge struct {
	Source      string `json:"source"`
	Target      string `json:"target"`
	Relation    string `json:"relation"`
	Condition   string `json:"condition,omitempty"`
	EvidenceRef string `json:"evidence_ref,omitempty"`
}

type VisualizationStep struct {
	Index  int      `json:"index"`
	Title  string   `json:"title"`
	Input  []string `json:"input,omitempty"`
	Output []string `json:"output,omitempty"`
	Detail string   `json:"detail,omitempty"`
}

type ExperimentVariables struct {
	Independent []string `json:"independent,omitempty"`
	Dependent   []string `json:"dependent,omitempty"`
	Controlled  []string `json:"controlled,omitempty"`
}

type MechanismStage struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Inputs      []string `json:"inputs,omitempty"`
	Outputs     []string `json:"outputs,omitempty"`
}

type VariableEffect struct {
	Variable  string `json:"variable"`
	Effect    string `json:"effect"`
	Condition string `json:"condition,omitempty"`
	Evidence  string `json:"evidence,omitempty"`
}

type InteractionPlan struct {
	Controls           []ControlSpec      `json:"controls,omitempty"`
	Challenge          *ChallengeSpec     `json:"challenge,omitempty"`
	FeedbackRules      []FeedbackRule     `json:"feedback_rules,omitempty"`
	RegenerationPolicy RegenerationPolicy `json:"regeneration_policy"`
}

type ControlSpec struct {
	Variable string `json:"variable"`
	Control  string `json:"control"`
	Label    string `json:"label"`
}

type ChallengeSpec struct {
	Goal                   string `json:"goal"`
	SuccessCondition       string `json:"success_condition,omitempty"`
	FeedbackGeneratedByLLM bool   `json:"feedback_generated_by_llm"`
}

type FeedbackRule struct {
	When    string `json:"when"`
	Message string `json:"message,omitempty"`
	Action  string `json:"action,omitempty"`
}

type RegenerationPolicy struct {
	LocalRecompute []string `json:"local_recompute,omitempty"`
	LLMRegenerate  []string `json:"llm_regenerate,omitempty"`
}

type AssessmentTask struct {
	TaskType          string   `json:"task_type"`
	Question          string   `json:"question"`
	ExpectedKeyPoints []string `json:"expected_key_points,omitempty"`
	FeedbackRule      string   `json:"feedback_rule,omitempty"`
	MisconceptionType string   `json:"misconception_type,omitempty"`
	NextAction        string   `json:"next_action,omitempty"`
}

type ModelValidationReport struct {
	SchemaValid      bool              `json:"schema_valid"`
	EvidenceValid    bool              `json:"evidence_valid"`
	DomainValid      bool              `json:"domain_valid"`
	SafetyValid      bool              `json:"safety_valid"`
	Checks           []ValidationCheck `json:"checks,omitempty"`
	Confidence       float64           `json:"confidence"`
	FallbackRequired bool              `json:"fallback_required"`
	FallbackReason   string            `json:"fallback_reason,omitempty"`
	RetryCount       int               `json:"retry_count,omitempty"`
}

type ValidationCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type RegenerationHint struct {
	Reason  string `json:"reason"`
	Message string `json:"message"`
}
