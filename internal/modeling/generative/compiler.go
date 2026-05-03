package generative

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	biologydomain "github.com/beihai0xff/snowy/internal/modeling/biology/domain"
	biologyservice "github.com/beihai0xff/snowy/internal/modeling/biology/service"
	physicsdomain "github.com/beihai0xff/snowy/internal/modeling/physics/domain"
	physicsservice "github.com/beihai0xff/snowy/internal/modeling/physics/service"
	"github.com/beihai0xff/snowy/internal/repo/llm"
	searchdomain "github.com/beihai0xff/snowy/internal/repo/search"
)

const defaultCompileTimeout = 120 * time.Second

type CompilerOption func(*compilerService)

type compilerService struct {
	searchSvc   searchdomain.Service
	physicsSvc  physicsservice.PhysicsService
	biologySvc  biologyservice.BiologyService
	primaryLLM  llm.Provider
	fallbackLLM llm.Provider
	repo        Repository
	validator   Validator
	now         func() time.Time
}

func NewCompilerService(
	searchSvc searchdomain.Service,
	physicsSvc physicsservice.PhysicsService,
	biologySvc biologyservice.BiologyService,
	repo Repository,
	opts ...CompilerOption,
) Service {
	svc := &compilerService{
		searchSvc:  searchSvc,
		physicsSvc: physicsSvc,
		biologySvc: biologySvc,
		repo:       repo,
		validator:  NewDefaultValidator(),
		now:        time.Now,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(svc)
		}
	}
	return svc
}

func WithLLMProviders(primary, fallback llm.Provider) CompilerOption {
	return func(s *compilerService) {
		s.primaryLLM = primary
		s.fallbackLLM = fallback
	}
}

func WithValidator(v Validator) CompilerOption {
	return func(s *compilerService) {
		if v != nil {
			s.validator = v
		}
	}
}

func WithNow(now func() time.Time) CompilerOption {
	return func(s *compilerService) {
		if now != nil {
			s.now = now
		}
	}
}

func (s *compilerService) Compile(ctx context.Context, req *CompileRequest) (*GenerativeModelPackage, error) {
	if req == nil {
		return nil, errors.New("compile request is nil")
	}
	req.Message = strings.TrimSpace(req.Message)
	if req.Message == "" {
		return nil, errors.New("message is empty")
	}
	if req.Domain == "" {
		req.Domain = DomainAuto
	}
	if req.GradeBand == "" {
		req.GradeBand = GradeBandHighSchool
	}
	if req.TargetMode == "" {
		req.TargetMode = TargetModeInteractive
	}

	domain := resolveDomain(req.Domain, req.Message)
	evidence := s.collectEvidence(ctx, req, domain)
	pkg, modelName, llmErr := s.compileWithLLM(ctx, req, domain, evidence)
	var validationErr error
	if llmErr == nil && pkg != nil {
		s.finalizePackage(req, pkg, domain, evidence, modelName, "success", "")
		report := s.validator.Validate(pkg)
		pkg.ValidationReport = report
		pkg.Confidence = clampConfidence(report.Confidence)
		if !report.FallbackRequired {
			return s.saveAndReturn(ctx, pkg)
		}
		validationErr = validationFailureError(report)
	}

	fallbackCause := llmErr
	if fallbackCause == nil {
		fallbackCause = validationErr
	}
	fallbackPkg, fallbackErr := s.compileFallback(ctx, req, domain, evidence, fallbackCause)
	if fallbackErr != nil {
		return nil, fallbackErr
	}
	return s.saveAndReturn(ctx, fallbackPkg)
}

func validationFailureError(report ModelValidationReport) error {
	if !report.FallbackRequired {
		return nil
	}
	parts := make([]string, 0, len(report.Checks)+1)
	if strings.TrimSpace(report.FallbackReason) != "" {
		parts = append(parts, report.FallbackReason)
	}
	for _, check := range report.Checks {
		if check.Status != "fail" {
			continue
		}
		msg := strings.TrimSpace(check.Message)
		if msg == "" {
			msg = check.Name
		}
		parts = append(parts, fmt.Sprintf("%s: %s", check.Name, msg))
	}
	if len(parts) == 0 {
		return errors.New("generated package did not pass validation")
	}
	return errors.New(strings.Join(parts, "; "))
}

func (s *compilerService) GetPackage(ctx context.Context, id string) (*GenerativeModelPackage, error) {
	if s.repo == nil {
		return nil, errors.New("generative package repository is nil")
	}
	uid, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return nil, fmt.Errorf("invalid package id: %w", err)
	}
	return s.repo.GetByID(ctx, uid)
}

func (s *compilerService) compileWithLLM(
	ctx context.Context,
	req *CompileRequest,
	domain string,
	evidence []EvidenceRef,
) (*GenerativeModelPackage, string, error) {
	providers := []llm.Provider{s.primaryLLM, s.fallbackLLM}
	failures := make([]string, 0, len(providers))
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		requestCtx, cancel := context.WithTimeout(ctx, defaultCompileTimeout)
		resp, err := provider.Generate(requestCtx, &llm.Request{
			Model: providerConfiguredModel(provider),
			Messages: []llm.Message{
				{Role: "system", Content: compileSystemPrompt()},
				{Role: "user", Content: buildCompileUserPrompt(req, domain, evidence)},
			},
			MaxTokens:   4096,
			Temperature: 0.2,
		})
		cancel()
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", provider.Name(), err))
			continue
		}
		pkg, err := decodePackageJSON(resp.Content)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", provider.Name(), err))
			continue
		}
		modelName := resp.Model
		if modelName == "" {
			modelName = provider.Name()
		}
		return pkg, modelName, nil
	}
	if len(failures) == 0 {
		return nil, "", errors.New("no llm provider configured")
	}
	return nil, "", errors.New(strings.Join(failures, "; "))
}

func (s *compilerService) finalizePackage(
	req *CompileRequest,
	pkg *GenerativeModelPackage,
	domain string,
	evidence []EvidenceRef,
	modelName string,
	status string,
	fallbackReason string,
) {
	if pkg.PackageID == uuid.Nil {
		pkg.PackageID = uuid.New()
	}
	pkg.UserID = req.UserID
	pkg.SessionID = req.SessionID
	pkg.Domain = domain
	pkg.Question = req.Message
	if pkg.CreatedAt.IsZero() {
		pkg.CreatedAt = s.now()
	}
	if len(pkg.EvidenceRefs) == 0 {
		pkg.EvidenceRefs = evidence
	}
	if pkg.LearningModel.Domain == "" {
		pkg.LearningModel.Domain = domain
	}
	if pkg.LearningModel.GradeBand == "" {
		pkg.LearningModel.GradeBand = req.GradeBand
	}
	if pkg.GenerativeModel.Domain == "" {
		pkg.GenerativeModel.Domain = domain
	}
	if pkg.GenerativeModel.GradeBand == "" {
		pkg.GenerativeModel.GradeBand = req.GradeBand
	}
	if pkg.ModelName == "" {
		pkg.ModelName = modelName
	}
	pkg.Status = status
	pkg.FallbackReason = fallbackReason
	pkg.Confidence = clampConfidence(pkg.Confidence)
}

func (s *compilerService) saveAndReturn(ctx context.Context, pkg *GenerativeModelPackage) (*GenerativeModelPackage, error) {
	if s.repo != nil {
		if err := s.repo.Save(ctx, pkg); err != nil {
			return nil, fmt.Errorf("save generative model package: %w", err)
		}
	}
	return pkg, nil
}

func (s *compilerService) compileFallback(
	ctx context.Context,
	req *CompileRequest,
	domain string,
	evidence []EvidenceRef,
	cause error,
) (*GenerativeModelPackage, error) {
	var pkg *GenerativeModelPackage
	var err error
	switch domain {
	case DomainPhysics:
		pkg, err = s.fallbackPhysics(ctx, req, evidence)
	case DomainBiology:
		pkg, err = s.fallbackBiology(ctx, req, evidence)
	default:
		pkg, err = s.fallbackPhysics(ctx, req, evidence)
	}
	if err != nil {
		return nil, err
	}
	fallbackReason := "llm generation unavailable"
	if cause != nil {
		fallbackReason = cause.Error()
	}
	s.finalizePackage(req, pkg, domain, evidence, "local-fallback", "fallback", fallbackReason)
	report := s.validator.Validate(pkg)
	report.FallbackRequired = true
	report.FallbackReason = fallbackReason
	if report.Confidence > 0.55 || report.Confidence == 0 {
		report.Confidence = 0.55
	}
	pkg.ValidationReport = report
	pkg.Confidence = report.Confidence
	pkg.FallbackReason = fallbackReason
	pkg.Warnings = append(pkg.Warnings, "当前结果由规则兜底生成；建议在模型服务恢复后重新推理。")
	pkg.RegenerationHints = append(pkg.RegenerationHints, RegenerationHint{Reason: "llm_fallback", Message: "模型服务失败或输出未通过校验，可点击重新生成触发大模型再推理。"})
	return pkg, nil
}

func (s *compilerService) fallbackPhysics(ctx context.Context, req *CompileRequest, evidence []EvidenceRef) (*GenerativeModelPackage, error) {
	if s.physicsSvc == nil {
		return nil, errors.New("physics service is nil")
	}
	model, err := s.physicsSvc.Analyze(ctx, req.Message, req.Context.UserNotes)
	if err != nil {
		return nil, fmt.Errorf("physics fallback analyze: %w", err)
	}
	return packageFromPhysics(req, model, evidence, s.now()), nil
}

func (s *compilerService) fallbackBiology(ctx context.Context, req *CompileRequest, evidence []EvidenceRef) (*GenerativeModelPackage, error) {
	if s.biologySvc == nil {
		return nil, errors.New("biology service is nil")
	}
	model, err := s.biologySvc.Analyze(ctx, req.Message, req.Context.UserNotes)
	if err != nil {
		return nil, fmt.Errorf("biology fallback analyze: %w", err)
	}
	return packageFromBiology(req, model, evidence, s.now()), nil
}

func (s *compilerService) collectEvidence(ctx context.Context, req *CompileRequest, domain string) []EvidenceRef {
	evidence := append([]EvidenceRef(nil), req.Context.Citations...)
	if len(evidence) > 0 || s.searchSvc == nil {
		return normalizeEvidence(evidence, req.Context.KnowledgeTags)
	}
	resp, err := s.searchSvc.Query(ctx, &searchdomain.Query{Text: req.Message, Filters: searchdomain.Filters{Subject: domain, Grade: req.GradeBand}})
	if err != nil || resp == nil {
		return normalizeEvidence(evidence, req.Context.KnowledgeTags)
	}
	for _, citation := range resp.Citations {
		evidence = append(evidence, EvidenceRef{DocID: citation.DocID, SourceType: citation.SourceType, Snippet: citation.Snippet, KnowledgeTags: resp.KnowledgeTags, Confidence: citation.Score})
	}
	return normalizeEvidence(evidence, resp.KnowledgeTags)
}

func normalizeEvidence(evidence []EvidenceRef, tags []string) []EvidenceRef {
	if len(evidence) == 0 {
		return []EvidenceRef{{DocID: "runtime-grounding", SourceType: "runtime", Snippet: "当前建模基于用户题干与高中阶段通用知识，缺少可追溯引用时需降低可信度。", KnowledgeTags: tags, Confidence: 0.35}}
	}
	for i := range evidence {
		if evidence[i].Confidence <= 0 {
			evidence[i].Confidence = 0.75
		}
		if len(evidence[i].KnowledgeTags) == 0 {
			evidence[i].KnowledgeTags = tags
		}
	}
	return evidence
}

func compileSystemPrompt() string {
	return strings.TrimSpace(`你是 Snowy v2 的生成式科学建模编译器。请根据用户问题和证据，输出一个面向高中生的 GenerativeModelPackage JSON。
硬性要求：
1. 只输出 JSON，不要 Markdown，不要代码块。
2. 不展示隐藏思维链；reasoning_trace 只写给学生看的简洁推理摘要和关键步骤。
3. domain 只能是 physics 或 biology。
4. 物理必须输出 simulation_logic，包含变量、单位、公式、渲染说明、交互计划和再推理条件。
5. 生物必须输出 visualization_graph，包含概念节点、关系边、过程阶段、实验变量、曲线解释或限制因素。
6. 所有知识性结论尽量绑定 evidence_refs；证据不足时在 warnings 和 validation_report 中标记低可信。
7. 输出字段使用 snake_case，结构必须匹配用户提供的契约。`)
}

func buildCompileUserPrompt(req *CompileRequest, domain string, evidence []EvidenceRef) string {
	payload := map[string]any{
		"contract": map[string]any{
			"package_id":          "uuid string optional",
			"domain":              "physics|biology",
			"question":            "string",
			"learning_model":      "object",
			"evidence_refs":       "array",
			"reasoning_trace":     "object",
			"generative_model":    "object",
			"simulation_logic":    "object|null for biology",
			"visualization_graph": "object|null for physics",
			"interaction_plan":    "object",
			"assessment_tasks":    "array",
			"validation_report":   "object",
			"regeneration_hints":  "array",
			"warnings":            "array",
			"confidence":          "number 0..1",
		},
		"domain":      domain,
		"grade_band":  req.GradeBand,
		"target_mode": req.TargetMode,
		"message":     req.Message,
		"context":     req.Context,
		"evidence":    evidence,
	}
	b, _ := json.Marshal(payload)
	return string(b)
}

func decodePackageJSON(content string) (*GenerativeModelPackage, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, errors.New("empty llm response")
	}
	content = extractJSONPayload(stripJSONFence(content))

	var raw map[string]any
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return nil, fmt.Errorf("decode generative package json: %w", err)
	}
	normalizePackageRaw(raw)
	normalized, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("normalize generative package json: %w", err)
	}

	var pkg GenerativeModelPackage
	if err := json.Unmarshal(normalized, &pkg); err != nil {
		return nil, fmt.Errorf("decode generative package json: %w", err)
	}
	return &pkg, nil
}

func extractJSONPayload(content string) string {
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "{") && strings.HasSuffix(content, "}") {
		return content
	}
	start := strings.IndexByte(content, '{')
	if start < 0 {
		return content
	}
	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(content); i++ {
		c := content[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if c == '\\' {
				escaped = true
				continue
			}
			if c == '"' {
				inString = false
			}
			continue
		}
		switch c {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return strings.TrimSpace(content[start : i+1])
			}
		}
	}
	return content
}

func normalizePackageRaw(raw map[string]any) {
	normalizeUUIDField(raw, "package_id")
	normalizeUUIDField(raw, "session_id")
	if evidence, ok := raw["evidence_refs"]; ok {
		raw["evidence_refs"] = normalizeEvidenceList(evidence)
	}
	if warnings, ok := raw["warnings"]; ok {
		raw["warnings"] = normalizeStringList(warnings)
	}
	if hints, ok := raw["regeneration_hints"]; ok {
		raw["regeneration_hints"] = normalizeRegenerationHints(hints)
	}
	if tasks, ok := raw["assessment_tasks"]; ok {
		raw["assessment_tasks"] = normalizeAssessmentTasks(tasks)
	}
	if trace, ok := raw["reasoning_trace"].(map[string]any); ok {
		normalizeReasoningTrace(trace)
	}
	if report, ok := raw["validation_report"].(map[string]any); ok {
		normalizeValidationReport(report)
	}
	if learning, ok := raw["learning_model"].(map[string]any); ok {
		normalizeLearningModel(learning, raw)
	}
	if model, ok := raw["generative_model"].(map[string]any); ok {
		normalizeLearningModel(model, raw)
		if variables, ok := model["variables"]; ok {
			model["variables"] = normalizeVariableList(variables)
		}
	}
	if sim, ok := raw["simulation_logic"].(map[string]any); ok {
		normalizeSimulationLogic(sim)
		mergeSimulationInteractionPlan(raw, sim)
	}
	if _, ok := raw["interaction_plan"]; !ok {
		raw["interaction_plan"] = map[string]any{}
	}
	if plan, ok := raw["interaction_plan"].(map[string]any); ok {
		normalizeInteractionPlan(plan)
		ensureInteractionPlanFromSimulation(plan, raw)
	}
}

func mergeSimulationInteractionPlan(raw map[string]any, sim map[string]any) {
	simPlan, ok := sim["interaction_plan"].(map[string]any)
	if !ok || len(simPlan) == 0 {
		return
	}
	plan, ok := raw["interaction_plan"].(map[string]any)
	if !ok || plan == nil {
		plan = map[string]any{}
		raw["interaction_plan"] = plan
	}
	if _, ok := plan["controls"]; !ok {
		for _, alias := range []string{"controls", "user_controls", "elements", "parameters"} {
			if controls, exists := simPlan[alias]; exists {
				plan["controls"] = controls
				break
			}
		}
	}
	if _, ok := plan["feedback_rules"]; !ok {
		if update, ok := simPlan["real_time_update"]; ok {
			plan["feedback_rules"] = []any{fmt.Sprint(update)}
		}
	}
}

func normalizeLearningModel(model map[string]any, raw map[string]any) {
	if _, ok := model["learning_goal"]; !ok {
		switch {
		case model["goal"] != nil:
			model["learning_goal"] = model["goal"]
		case model["objective"] != nil:
			model["learning_goal"] = model["objective"]
		case model["objectives"] != nil:
			model["learning_goal"] = strings.Join(anyToStrings(model["objectives"]), "；")
		case raw["reasoning_trace"] != nil:
			if trace, ok := raw["reasoning_trace"].(map[string]any); ok {
				model["learning_goal"] = trace["summary"]
			}
		}
		if strings.TrimSpace(fmt.Sprint(model["learning_goal"])) == "" {
			model["learning_goal"] = "理解并解释当前生成式科学模型的核心变量关系。"
		}
	}
	model["learning_goal"] = firstNonEmptyString(model["learning_goal"], "理解并解释当前生成式科学模型的核心变量关系。")
	if _, ok := model["domain"]; !ok {
		if domain, ok := raw["domain"]; ok {
			model["domain"] = domain
		}
	}
	if _, ok := model["grade_band"]; !ok {
		model["grade_band"] = GradeBandHighSchool
	}
	if _, ok := model["topic"]; !ok {
		if modelType := strings.TrimSpace(fmt.Sprint(model["model_type"])); modelType != "" {
			model["topic"] = modelType
		} else if description := strings.TrimSpace(fmt.Sprint(model["description"])); description != "" {
			model["topic"] = description
		} else {
			model["topic"] = "generated_model"
		}
	}
	model["topic"] = firstNonEmptyString(model["topic"], "generated_model")
	if tags, ok := model["knowledge_tags"]; ok {
		model["knowledge_tags"] = normalizeStringList(tags)
	}
}

func normalizeSimulationLogic(sim map[string]any) {
	if _, ok := sim["simulation_type"]; !ok {
		if typ, ok := sim["type"]; ok {
			sim["simulation_type"] = typ
		} else if modelType, ok := sim["model_type"]; ok {
			sim["simulation_type"] = modelType
		} else {
			sim["simulation_type"] = "generated_physics_model"
		}
	}
	sim["simulation_type"] = firstNonEmptyString(sim["simulation_type"], "generated_physics_model")
	if _, ok := sim["runtime"]; !ok {
		sim["runtime"] = "safe_math_dsl"
	}
	if assumptions, ok := sim["assumptions"]; ok {
		sim["assumptions"] = normalizeStringList(assumptions)
	}
	if render, ok := sim["rendering_instructions"]; ok {
		if _, exists := sim["render_instructions"]; !exists {
			sim["render_instructions"] = render
		}
	}
	if render, ok := sim["render_instructions"]; ok {
		sim["render_instructions"] = normalizeRenderInstructions(render)
	} else {
		sim["render_instructions"] = defaultRenderInstructions(nil)
	}
	if variables, ok := sim["variables"]; ok {
		sim["variables"] = normalizeVariableList(variables)
	}
	if _, exists := sim["formulas"]; !exists {
		for _, alias := range []string{"formula", "equation", "equations"} {
			if formula, ok := sim[alias]; ok {
				sim["formulas"] = formula
				break
			}
		}
	}
	if formulas, ok := sim["formulas"]; ok {
		sim["formulas"] = normalizeFormulaList(formulas)
	}
	if _, ok := sim["local_recompute_allowed"]; !ok {
		sim["local_recompute_allowed"] = true
	}
	if _, ok := sim["state_variables"]; !ok {
		sim["state_variables"] = variableNames(sim["variables"])
	}
	if regenerate, ok := sim["re_reasoning_conditions"]; ok {
		if _, exists := sim["regenerate_when"]; !exists {
			sim["regenerate_when"] = normalizeStringList(regenerate)
		}
	}
	if regenerate, ok := sim["regenerate_when"]; ok {
		sim["regenerate_when"] = normalizeStringList(regenerate)
	}
}

func normalizeRenderInstructions(value any) any {
	switch render := value.(type) {
	case string:
		return defaultRenderInstructions([]any{strings.TrimSpace(render)})
	case []any:
		return defaultRenderInstructions(normalizeStringList(render))
	case map[string]any:
		if _, ok := render["coordinate_system"]; !ok {
			render["coordinate_system"] = "2d_cartesian"
		}
		if layers, ok := render["layers"]; ok {
			render["layers"] = normalizeStringList(layers)
		} else {
			render["layers"] = []any{"trajectory", "vector", "curve"}
		}
		if annotations, ok := render["annotations"]; ok {
			render["annotations"] = normalizeStringList(annotations)
		}
		return render
	default:
		return defaultRenderInstructions(nil)
	}
}

func defaultRenderInstructions(annotations any) map[string]any {
	out := map[string]any{
		"coordinate_system": "2d_cartesian",
		"layers":            []any{"trajectory", "vector", "curve"},
	}
	if annotations != nil {
		out["annotations"] = normalizeStringList(annotations)
	}
	return out
}

func normalizeUUIDField(raw map[string]any, field string) {
	value, ok := raw[field]
	if !ok || value == nil {
		delete(raw, field)
		return
	}
	text, ok := value.(string)
	if !ok {
		return
	}
	text = strings.TrimSpace(text)
	if text == "" {
		delete(raw, field)
		return
	}
	if _, err := uuid.Parse(text); err != nil {
		delete(raw, field)
		return
	}
	raw[field] = text
}

func normalizeEvidenceList(value any) any {
	items, ok := value.([]any)
	if !ok {
		return value
	}
	out := make([]any, 0, len(items))
	for i, item := range items {
		switch evidence := item.(type) {
		case string:
			snippet := strings.TrimSpace(evidence)
			if snippet == "" {
				continue
			}
			out = append(out, map[string]any{
				"doc_id":      fmt.Sprintf("llm-evidence-%d", i+1),
				"source_type": "llm_grounding",
				"snippet":     snippet,
				"confidence":  0.55,
			})
		case map[string]any:
			if _, ok := evidence["doc_id"]; !ok {
				evidence["doc_id"] = fmt.Sprintf("llm-evidence-%d", i+1)
			}
			if _, ok := evidence["source_type"]; !ok {
				evidence["source_type"] = "llm_grounding"
			}
			if _, ok := evidence["snippet"]; !ok {
				if title, ok := evidence["title"]; ok {
					evidence["snippet"] = title
				} else {
					evidence["snippet"] = "大模型基于输入证据生成的引用片段"
				}
			}
			if _, ok := evidence["confidence"]; !ok {
				evidence["confidence"] = 0.55
			}
			out = append(out, evidence)
		default:
			out = append(out, item)
		}
	}
	return out
}

func normalizeVariableList(value any) any {
	items, ok := value.([]any)
	if !ok {
		return value
	}
	out := make([]any, 0, len(items))
	for i, item := range items {
		variable, ok := item.(map[string]any)
		if !ok {
			out = append(out, item)
			continue
		}
		name := strings.TrimSpace(fmt.Sprint(variable["name"]))
		if name == "" {
			name = strings.TrimSpace(fmt.Sprint(variable["symbol"]))
		}
		if name == "" {
			name = fmt.Sprintf("variable_%d", i+1)
		}
		variable["name"] = name
		if _, ok := variable["label"]; !ok {
			if description, ok := variable["description"]; ok {
				variable["label"] = description
			} else {
				variable["label"] = labelForVariable(name)
			}
		} else {
			variable["label"] = firstNonEmptyString(variable["label"], labelForVariable(name))
		}
		def := defaultValueForVariable(name, numberOrDefault(variable["default"], 0))
		variable["default"] = def
		if _, ok := variable["min"]; !ok {
			variable["min"] = defaultMinForVariable(name, def)
		}
		if _, ok := variable["max"]; !ok {
			variable["max"] = defaultMaxForVariable(name, def)
		}
		if numberOrDefault(variable["max"], 0) < numberOrDefault(variable["min"], 0) {
			minValue := numberOrDefault(variable["max"], 0)
			maxValue := numberOrDefault(variable["min"], 0)
			variable["min"] = minValue
			variable["max"] = maxValue
		}
		if numberOrDefault(variable["max"], 0) == numberOrDefault(variable["min"], 0) {
			variable["max"] = numberOrDefault(variable["min"], 0) + 1
		}
		if _, ok := variable["step"]; !ok {
			variable["step"] = defaultStepForVariable(name)
		}
		out = append(out, variable)
	}
	return out
}

func numberOrDefault(value any, fallback float64) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case json.Number:
		parsed, err := v.Float64()
		if err == nil {
			return parsed
		}
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func boolOrDefault(value any, fallback bool) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(v))
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func labelForVariable(name string) string {
	switch normalizeVariableKey(name) {
	case "m", "mass":
		return "质量"
	case "k", "stiffness", "spring_constant":
		return "劲度系数"
	case "period", "T":
		return "周期"
	case "t", "time":
		return "时间"
	case "g":
		return "重力加速度"
	case "amplitude", "A":
		return "振幅"
	case "v0":
		return "初速度"
	default:
		return name
	}
}

func defaultMinForVariable(name string, def float64) float64 {
	key := normalizeVariableKey(name)
	switch key {
	case "g":
		return 9.8
	case "t", "time", "period", "T":
		return 0.1
	case "m", "mass":
		return 0.1
	case "k", "stiffness", "spring_constant":
		return 1
	case "amplitude", "A":
		return 0.01
	}
	if def > 0 {
		return 0
	}
	return def - 10
}

func defaultMaxForVariable(name string, def float64) float64 {
	key := normalizeVariableKey(name)
	switch key {
	case "g":
		return 9.8
	case "t", "time":
		return 10
	case "period", "T":
		return 20
	case "m", "mass":
		return 10
	case "k", "stiffness", "spring_constant":
		return 1000
	case "amplitude", "A":
		return 2
	}
	if def > 0 {
		return def * 3
	}
	return def + 10
}

func defaultStepForVariable(name string) float64 {
	key := normalizeVariableKey(name)
	switch key {
	case "t", "time", "period", "T", "m", "mass", "amplitude", "A":
		return 0.1
	case "g":
		return 0
	case "k", "stiffness", "spring_constant":
		return 10
	default:
		return 1
	}
}

func defaultValueForVariable(name string, current float64) float64 {
	if current > 0 {
		return current
	}
	key := normalizeVariableKey(name)
	switch key {
	case "g":
		return 9.8
	case "t", "time":
		return 1
	case "period", "T":
		return 2
	case "m", "mass":
		return 1
	case "k", "stiffness", "spring_constant":
		return 100
	case "amplitude", "A":
		return 0.5
	default:
		return current
	}
}

func normalizeVariableKey(name string) string {
	key := strings.TrimSpace(name)
	key = strings.TrimPrefix(key, "var_")
	return strings.ToLower(key)
}

func normalizeStringList(value any) any {
	switch v := value.(type) {
	case string:
		if text := strings.TrimSpace(v); text != "" {
			return []any{text}
		}
		return []any{}
	case []string:
		out := make([]any, 0, len(v))
		for _, item := range v {
			if text := strings.TrimSpace(item); text != "" {
				out = append(out, text)
			}
		}
		return out
	}
	items, ok := value.([]any)
	if !ok {
		return value
	}
	out := make([]any, 0, len(items))
	for _, item := range items {
		if text := strings.TrimSpace(fmt.Sprint(item)); text != "" {
			out = append(out, text)
		}
	}
	return out
}

func normalizeReasoningTrace(trace map[string]any) {
	for _, field := range []string{"evidence_used", "assumptions", "key_steps"} {
		if value, ok := trace[field]; ok {
			trace[field] = normalizeStringList(value)
		}
	}
	if _, ok := trace["summary"]; !ok {
		trace["summary"] = "基于题干和证据生成可交互科学模型。"
	}
	if confidence, ok := trace["confidence"]; ok {
		trace["confidence"] = numberOrDefault(confidence, 0.75)
	}
}

func normalizeValidationReport(report map[string]any) {
	if checks, ok := report["checks"]; ok {
		report["checks"] = normalizeValidationChecks(checks)
	}
	for _, field := range []string{"schema_valid", "evidence_valid", "domain_valid", "safety_valid", "fallback_required"} {
		if value, ok := report[field]; ok {
			report[field] = boolOrDefault(value, false)
		}
	}
	if confidence, ok := report["confidence"]; ok {
		report["confidence"] = numberOrDefault(confidence, 0)
	}
}

func normalizeValidationChecks(value any) any {
	items, ok := value.([]any)
	if !ok {
		return value
	}
	out := make([]any, 0, len(items))
	for i, item := range items {
		switch check := item.(type) {
		case string:
			name := strings.TrimSpace(check)
			if name == "" {
				continue
			}
			out = append(out, map[string]any{"name": name, "status": "pass"})
		case map[string]any:
			if _, ok := check["name"]; !ok {
				check["name"] = fmt.Sprintf("check_%d", i+1)
			}
			if _, ok := check["status"]; !ok {
				check["status"] = "pass"
			}
			out = append(out, check)
		default:
			out = append(out, item)
		}
	}
	return out
}

func normalizeInteractionPlan(plan map[string]any) {
	if _, ok := plan["controls"]; !ok {
		for _, alias := range []string{"user_controls", "elements", "parameters"} {
			if controls, ok := plan[alias]; ok {
				plan["controls"] = controls
				break
			}
		}
	}
	if controls, ok := plan["controls"]; ok {
		plan["controls"] = normalizeControls(controls)
	}
	if _, ok := plan["challenge"]; !ok {
		if steps, ok := plan["steps"]; ok {
			joined := strings.Join(anyToStrings(steps), "；")
			if strings.TrimSpace(joined) != "" {
				plan["challenge"] = joined
			}
		}
	}
	if challenge, ok := plan["challenge"]; ok {
		plan["challenge"] = normalizeChallenge(challenge)
	}
	if _, ok := plan["feedback_rules"]; !ok {
		if update, ok := plan["real_time_update"]; ok {
			plan["feedback_rules"] = []any{fmt.Sprint(update)}
		}
	}
	if rules, ok := plan["feedback_rules"]; ok {
		plan["feedback_rules"] = normalizeFeedbackRules(rules)
	}
	if policy, ok := plan["regeneration_policy"].(map[string]any); ok {
		if local, ok := policy["local_recompute"]; ok {
			policy["local_recompute"] = normalizeStringList(local)
		}
		if llm, ok := policy["llm_regenerate"]; ok {
			policy["llm_regenerate"] = normalizeStringList(llm)
		}
	} else {
		plan["regeneration_policy"] = map[string]any{}
	}
}

func ensureInteractionPlanFromSimulation(plan map[string]any, raw map[string]any) {
	sim, ok := raw["simulation_logic"].(map[string]any)
	if !ok {
		return
	}
	if controls, ok := plan["controls"].([]any); !ok || len(controls) == 0 {
		if generated := controlsFromVariables(sim["variables"]); len(generated) > 0 {
			plan["controls"] = generated
		}
	}
	if _, ok := plan["challenge"]; !ok {
		plan["challenge"] = map[string]any{
			"goal":                      "调节参数，观察模型结果变化，并用公式解释趋势。",
			"success_condition":         "能说出关键变量与结果之间的定性关系。",
			"feedback_generated_by_llm": true,
		}
	}
	if rules, ok := plan["feedback_rules"].([]any); !ok || len(rules) == 0 {
		plan["feedback_rules"] = []any{map[string]any{"when": "parameter_changed", "message": "观察数值、曲线和公式项如何随参数同步变化。"}}
	}
	policy, ok := plan["regeneration_policy"].(map[string]any)
	if !ok || policy == nil {
		policy = map[string]any{}
		plan["regeneration_policy"] = policy
	}
	if local, ok := policy["local_recompute"].([]any); !ok || len(local) == 0 {
		policy["local_recompute"] = variableNames(sim["variables"])
	}
	if llm, ok := policy["llm_regenerate"].([]any); !ok || len(llm) == 0 {
		policy["llm_regenerate"] = []any{"改变模型假设", "加入阻尼/外力等新因素", "学生解释与模型冲突"}
	}
}

func controlsFromVariables(value any) []any {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]any, 0, len(items))
	for _, item := range items {
		variable, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name := strings.TrimSpace(fmt.Sprint(variable["name"]))
		if name == "" {
			continue
		}
		out = append(out, map[string]any{
			"variable": name,
			"control":  "slider",
			"label":    firstNonEmptyString(variable["label"], labelForVariable(name)),
		})
	}
	return out
}

func normalizeChallenge(value any) any {
	switch challenge := value.(type) {
	case string:
		goal := strings.TrimSpace(challenge)
		if goal == "" {
			goal = "完成模型观察并解释变量关系。"
		}
		return map[string]any{"goal": goal, "feedback_generated_by_llm": true}
	case map[string]any:
		if _, ok := challenge["goal"]; !ok {
			challenge["goal"] = "完成模型观察并解释变量关系。"
		}
		if _, ok := challenge["feedback_generated_by_llm"]; !ok {
			challenge["feedback_generated_by_llm"] = true
		}
		return challenge
	default:
		return value
	}
}

func normalizeRegenerationHints(value any) any {
	items, ok := value.([]any)
	if !ok {
		return value
	}
	out := make([]any, 0, len(items))
	for i, item := range items {
		switch hint := item.(type) {
		case string:
			message := strings.TrimSpace(hint)
			if message == "" {
				continue
			}
			out = append(out, map[string]any{"reason": fmt.Sprintf("hint_%d", i+1), "message": message})
		case map[string]any:
			if _, ok := hint["reason"]; !ok {
				hint["reason"] = fmt.Sprintf("hint_%d", i+1)
			}
			if _, ok := hint["message"]; !ok {
				if reason, ok := hint["reason"]; ok {
					hint["message"] = fmt.Sprint(reason)
				} else {
					hint["message"] = "可重新触发大模型推理修正模型。"
				}
			}
			out = append(out, hint)
		default:
			out = append(out, item)
		}
	}
	return out
}

func normalizeAssessmentTasks(value any) any {
	items, ok := value.([]any)
	if !ok {
		return value
	}
	out := make([]any, 0, len(items))
	for i, item := range items {
		switch task := item.(type) {
		case string:
			question := strings.TrimSpace(task)
			if question == "" {
				continue
			}
			out = append(out, map[string]any{"task_type": "reflection", "question": question})
		case map[string]any:
			if _, ok := task["task_type"]; !ok {
				task["task_type"] = "reflection"
			}
			if _, ok := task["question"]; !ok {
				task["question"] = fmt.Sprintf("请完成第 %d 个学习检查。", i+1)
			}
			if points, ok := task["expected_key_points"]; ok {
				task["expected_key_points"] = normalizeStringList(points)
			}
			out = append(out, task)
		default:
			out = append(out, item)
		}
	}
	return out
}

func normalizeControls(value any) any {
	items, ok := value.([]any)
	if !ok {
		return value
	}
	out := make([]any, 0, len(items))
	for i, item := range items {
		switch control := item.(type) {
		case string:
			variable := strings.TrimSpace(control)
			if variable == "" {
				continue
			}
			out = append(out, map[string]any{"variable": variable, "control": "slider", "label": variable})
		case map[string]any:
			if _, ok := control["variable"]; !ok {
				control["variable"] = fmt.Sprintf("variable_%d", i+1)
			}
			if _, ok := control["control"]; !ok {
				control["control"] = "slider"
			}
			if _, ok := control["label"]; !ok {
				control["label"] = fmt.Sprint(control["variable"])
			}
			out = append(out, control)
		default:
			out = append(out, item)
		}
	}
	return out
}

func normalizeFeedbackRules(value any) any {
	items, ok := value.([]any)
	if !ok {
		return value
	}
	out := make([]any, 0, len(items))
	for i, item := range items {
		switch rule := item.(type) {
		case string:
			message := strings.TrimSpace(rule)
			if message == "" {
				continue
			}
			out = append(out, map[string]any{"when": fmt.Sprintf("rule_%d", i+1), "message": message})
		case map[string]any:
			if _, ok := rule["when"]; !ok {
				rule["when"] = fmt.Sprintf("rule_%d", i+1)
			}
			out = append(out, rule)
		default:
			out = append(out, item)
		}
	}
	return out
}

func normalizeFormulaList(value any) any {
	switch v := value.(type) {
	case string:
		return normalizeFormulaList([]any{v})
	case map[string]any:
		return normalizeFormulaList([]any{v})
	}
	items, ok := value.([]any)
	if !ok {
		return value
	}
	out := make([]any, 0, len(items))
	for i, item := range items {
		switch formula := item.(type) {
		case string:
			expr := strings.TrimSpace(formula)
			if expr == "" {
				continue
			}
			out = append(out, map[string]any{
				"id":      fmt.Sprintf("formula_%d", i+1),
				"expr":    expr,
				"meaning": "由大模型生成的公式关系",
			})
		case map[string]any:
			if _, ok := formula["id"]; !ok {
				formula["id"] = fmt.Sprintf("formula_%d", i+1)
			}
			if _, ok := formula["expr"]; !ok {
				if expression, ok := formula["expression"]; ok {
					formula["expr"] = expression
				} else if expression, ok := formula["formula"]; ok {
					formula["expr"] = expression
				} else if expression, ok := formula["equation"]; ok {
					formula["expr"] = expression
				}
			}
			formula["expr"] = firstNonEmptyString(formula["expr"], "result = f(variables)")
			if _, ok := formula["meaning"]; !ok {
				formula["meaning"] = "由大模型生成的公式关系"
			}
			out = append(out, formula)
		default:
			out = append(out, item)
		}
	}
	return out
}

func anyToStrings(value any) []string {
	switch v := value.(type) {
	case nil:
		return nil
	case string:
		if text := strings.TrimSpace(v); text != "" {
			return []string{text}
		}
		return nil
	case []string:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if text := strings.TrimSpace(item); text != "" {
				out = append(out, text)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if text := strings.TrimSpace(fmt.Sprint(item)); text != "" {
				out = append(out, text)
			}
		}
		return out
	default:
		if text := strings.TrimSpace(fmt.Sprint(v)); text != "" {
			return []string{text}
		}
		return nil
	}
}

func firstNonEmptyString(value any, fallback string) string {
	for _, item := range anyToStrings(value) {
		if strings.TrimSpace(item) != "" {
			return item
		}
	}
	return fallback
}

func variableNames(value any) []any {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]any, 0, len(items))
	for _, item := range items {
		if variable, ok := item.(map[string]any); ok {
			if name := strings.TrimSpace(fmt.Sprint(variable["name"])); name != "" {
				out = append(out, name)
			}
		}
	}
	return out
}

var fencedJSON = regexp.MustCompile("(?s)```(?:json)?\\s*(.*?)\\s*```")

func stripJSONFence(content string) string {
	if match := fencedJSON.FindStringSubmatch(content); len(match) == 2 {
		return strings.TrimSpace(match[1])
	}
	return content
}

func resolveDomain(domain, text string) string {
	domain = strings.ToLower(strings.TrimSpace(domain))
	if domain == DomainPhysics || domain == DomainBiology {
		return domain
	}
	lower := strings.ToLower(text)
	if strings.Contains(lower, "光合") || strings.Contains(lower, "细胞") || strings.Contains(lower, "遗传") || strings.Contains(lower, "生态") || strings.Contains(lower, "酶") || strings.Contains(lower, "biology") {
		return DomainBiology
	}
	return DomainPhysics
}

func providerConfiguredModel(provider llm.Provider) string {
	if configured, ok := provider.(llm.ConfiguredProvider); ok {
		return configured.ConfiguredModel()
	}
	return ""
}

func clampConfidence(v float64) float64 {
	if v <= 0 {
		return 0.8
	}
	if v > 0.99 {
		return 0.99
	}
	if v < 0.01 {
		return 0.01
	}
	return v
}

func packageFromPhysics(req *CompileRequest, model *physicsdomain.PhysicsModel, evidence []EvidenceRef, now time.Time) *GenerativeModelPackage {
	variables := make([]VariableSpec, 0, len(model.Parameters))
	controls := make([]ControlSpec, 0, len(model.Parameters))
	local := make([]string, 0, len(model.Parameters))
	for _, p := range model.Parameters {
		variables = append(variables, VariableSpec{Name: p.Name, Label: p.Label, Unit: p.Unit, Default: p.Default, Min: p.Min, Max: p.Max, Step: p.Step})
		controls = append(controls, ControlSpec{Variable: p.Name, Control: "slider", Label: p.Label})
		local = append(local, p.Name)
	}
	formulas := formulasForPhysics(model.ModelType)
	assumptions := []string{"高中阶段近似模型", "忽略未在题干中出现的次要因素"}
	if len(model.Warnings) > 0 {
		assumptions = append(assumptions, model.Warnings...)
	}
	return &GenerativeModelPackage{
		PackageID: uuid.New(), Domain: DomainPhysics, Question: req.Message, CreatedAt: now, EvidenceRefs: evidence, Confidence: 0.55,
		LearningModel:   LearningModelSpec{Domain: DomainPhysics, GradeBand: req.GradeBand, Topic: string(model.ModelType), LearningGoal: "通过大模型生成式仿真理解" + string(model.ModelType), KnowledgeTags: []string{string(model.ModelType)}, Difficulty: "medium"},
		ReasoningTrace:  ReasoningTrace{Summary: model.ResultSummary, EvidenceUsed: evidenceIDs(evidence), Assumptions: assumptions, KeySteps: derivationTitles(model.Steps), Confidence: 0.55},
		GenerativeModel: GenerativeModelSpec{ID: "fallback_physics", Domain: DomainPhysics, GradeBand: req.GradeBand, Topic: string(model.ModelType), LearningGoal: model.ResultSummary, KnowledgeTags: []string{string(model.ModelType)}, Entities: []EntitySpec{{ID: "object", Name: "研究对象", Type: "object"}}, Variables: variables, Relations: relationsForPhysics(model.ModelType)},
		SimulationLogic: &DynamicSimulationSpec{SimulationType: string(model.ModelType), Runtime: "safe_math_dsl", Assumptions: assumptions, StateVariables: []string{"t", "x", "v"}, Variables: variables, Formulas: formulas, RenderInstructions: RenderInstructions{CoordinateSystem: "2d_cartesian", Layers: []string{"trajectory", "vector", "curve", "target_zone"}, Annotations: []string{"由生成式模型包驱动展示"}}, LocalRecomputeAllowed: true, RegenerateWhen: []string{"改变模型假设", "新增受力或介质", "要求新的学习目标"}},
		InteractionPlan: InteractionPlan{Controls: controls, Challenge: &ChallengeSpec{Goal: "调节参数并解释结果变化", SuccessCondition: "能说出主要变量关系", FeedbackGeneratedByLLM: true}, FeedbackRules: []FeedbackRule{{When: "parameter_changed", Message: "观察轨迹、曲线和公式项如何同步变化。"}}, RegenerationPolicy: RegenerationPolicy{LocalRecompute: local, LLMRegenerate: []string{"new_force", "new_medium", "new_learning_goal", "conflicting_student_explanation"}}},
		AssessmentTasks: []AssessmentTask{{TaskType: "reflection", Question: "请用一句话总结当前模型中最关键的变量关系。", ExpectedKeyPoints: []string{"变量关系", "适用条件"}}},
	}
}

func packageFromBiology(req *CompileRequest, model *biologydomain.BiologyModel, evidence []EvidenceRef, now time.Time) *GenerativeModelPackage {
	nodes := make([]VisualizationNode, 0, len(model.Concepts))
	nameToID := map[string]string{}
	for i, c := range model.Concepts {
		id := fmt.Sprintf("n%d", i+1)
		nodes = append(nodes, VisualizationNode{ID: id, Label: c.Name, Type: c.Type})
		nameToID[c.Name] = id
	}
	edges := make([]VisualizationEdge, 0, len(model.Relations))
	for _, r := range model.Relations {
		source, target := nameToID[r.Source], nameToID[r.Target]
		if source != "" && target != "" {
			edges = append(edges, VisualizationEdge{Source: source, Target: target, Relation: r.Type})
		}
	}
	steps := make([]VisualizationStep, 0, len(model.ProcessSteps))
	for _, step := range model.ProcessSteps {
		steps = append(steps, VisualizationStep{Index: step.Index, Title: step.Title, Detail: step.Content})
	}
	vars := &ExperimentVariables{}
	if model.ExperimentVariables != nil {
		vars = &ExperimentVariables{Independent: model.ExperimentVariables.Independent, Dependent: model.ExperimentVariables.Dependent, Controlled: model.ExperimentVariables.Controlled}
	}
	return &GenerativeModelPackage{
		PackageID: uuid.New(), Domain: DomainBiology, Question: req.Message, CreatedAt: now, EvidenceRefs: evidence, Confidence: 0.55,
		LearningModel:      LearningModelSpec{Domain: DomainBiology, GradeBand: req.GradeBand, Topic: model.Topic, LearningGoal: "通过生成式可视化理解" + model.Topic, KnowledgeTags: []string{model.Topic}, Difficulty: "medium"},
		ReasoningTrace:     ReasoningTrace{Summary: model.ResultSummary, EvidenceUsed: evidenceIDs(evidence), Assumptions: []string{"高中生物范围内解释"}, KeySteps: stepTitlesBiology(model.ProcessSteps), Confidence: 0.55},
		GenerativeModel:    GenerativeModelSpec{ID: "fallback_biology", Domain: DomainBiology, GradeBand: req.GradeBand, Topic: model.Topic, LearningGoal: model.ResultSummary, KnowledgeTags: []string{model.Topic}, Entities: entitiesFromConcepts(model.Concepts), Relations: relationSpecsFromBiology(model.Relations)},
		VisualizationGraph: &GenerativeVisualizationSpec{VisualizationType: "generated_biology_process_graph", Topic: model.Topic, Nodes: nodes, Edges: edges, ProcessSteps: steps, ExperimentVariables: vars, CurveExplanation: "观察变量改变时，重点判断限制因素和结果变化趋势。", LimitingFactors: vars.Controlled},
		InteractionPlan:    InteractionPlan{Controls: []ControlSpec{{Variable: "animation_speed", Control: "slider", Label: "动画速度"}}, Challenge: &ChallengeSpec{Goal: "指出自变量、因变量和至少一个控制变量", FeedbackGeneratedByLLM: true}, FeedbackRules: []FeedbackRule{{When: "variable_confusion", Action: "trigger_llm_explanation"}}, RegenerationPolicy: RegenerationPolicy{LocalRecompute: []string{"animation_speed"}, LLMRegenerate: []string{"new_factor", "new_experiment_design", "conflicting_student_explanation"}}},
		AssessmentTasks:    []AssessmentTask{{TaskType: "micro_quiz", Question: "本题中的自变量、因变量和控制变量分别是什么？", ExpectedKeyPoints: append(append(vars.Independent, vars.Dependent...), vars.Controlled...)}},
	}
}

func formulasForPhysics(modelType physicsdomain.ModelType) []FormulaSpec {
	switch modelType {
	case physicsdomain.ModelProjectileMotion:
		return []FormulaSpec{{ID: "x_t", Expr: "x = v0 * cos(angle) * t", Meaning: "水平位移"}, {ID: "y_t", Expr: "y = v0 * sin(angle) * t - 0.5 * g * t^2", Meaning: "竖直位移"}}
	case physicsdomain.ModelNewtonSecondLaw:
		return []FormulaSpec{{ID: "newton_2", Expr: "F = m * a", Meaning: "牛顿第二定律"}}
	case physicsdomain.ModelUniformAcceleration:
		return []FormulaSpec{{ID: "x_t", Expr: "x = x0 + v0 * t + 0.5 * a * t^2", Meaning: "匀变速位移"}, {ID: "v_t", Expr: "v = v0 + a * t", Meaning: "速度变化"}}
	default:
		return []FormulaSpec{{ID: "model_relation", Expr: "result = f(variables)", Meaning: "由题意生成的变量关系"}}
	}
}

func relationsForPhysics(modelType physicsdomain.ModelType) []RelationSpec {
	if modelType == physicsdomain.ModelProjectileMotion {
		return []RelationSpec{{Source: "v0", Target: "x", Type: "influences", Description: "初速度影响水平位移"}, {Source: "g", Target: "t", Type: "constrains", Description: "重力加速度影响落地时间"}}
	}
	return []RelationSpec{{Source: "variables", Target: "result", Type: "determines", Description: "变量共同决定模型结果"}}
}

func derivationTitles(steps []physicsdomain.DerivationStep) []string {
	out := make([]string, 0, len(steps))
	for _, step := range steps {
		out = append(out, step.Title)
	}
	return out
}

func stepTitlesBiology(steps []biologydomain.ProcessStep) []string {
	out := make([]string, 0, len(steps))
	for _, step := range steps {
		out = append(out, step.Title)
	}
	return out
}

func evidenceIDs(evidence []EvidenceRef) []string {
	out := make([]string, 0, len(evidence))
	for _, e := range evidence {
		if e.DocID != "" {
			out = append(out, e.DocID)
		}
	}
	return out
}

func entitiesFromConcepts(concepts []biologydomain.Concept) []EntitySpec {
	out := make([]EntitySpec, 0, len(concepts))
	for i, c := range concepts {
		out = append(out, EntitySpec{ID: fmt.Sprintf("c%d", i+1), Name: c.Name, Type: c.Type})
	}
	return out
}

func relationSpecsFromBiology(relations []biologydomain.Relation) []RelationSpec {
	out := make([]RelationSpec, 0, len(relations))
	for _, r := range relations {
		out = append(out, RelationSpec{Source: r.Source, Target: r.Target, Type: r.Type})
	}
	return out
}
