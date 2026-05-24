package search

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/beihai0xff/snowy/internal/prompt"
	"github.com/beihai0xff/snowy/internal/repo/embedding"
	"github.com/beihai0xff/snowy/internal/repo/llm"
)

const (
	defaultSearchOffset  = 0
	defaultSearchLimit   = 8
	maxRelatedTitleRunes = 48
	tagNewtonSecondLaw   = "牛顿第二定律"
	tagPhotosynthesis    = "光合作用"
)

type serviceImpl struct {
	repo          Repository
	parser        QueryParser
	ranker        ResultRanker
	embedding     embedding.Provider
	logs          LogRepository
	answerRecords AnswerRecordRepository
	feedback      FeedbackRepository
	llmChain      llm.Provider
}

// Option 配置 Search Service 的可选依赖。
type Option func(*serviceImpl)

// WithLLMProvider configures the ordered OpenAI-compatible model chain used for direct answers.
func WithLLMProvider(provider llm.Provider) Option {
	return func(s *serviceImpl) {
		s.llmChain = provider
	}
}

// WithAnswerRecordRepository enables durable answer_records persistence for search answers.
func WithAnswerRecordRepository(repo AnswerRecordRepository) Option {
	return func(s *serviceImpl) {
		s.answerRecords = repo
	}
}

// WithFeedbackRepository lets community reactions influence ranking and suggestions.
func WithFeedbackRepository(repo FeedbackRepository) Option {
	return func(s *serviceImpl) {
		s.feedback = repo
	}
}

// NewService 创建知识点直答服务实现。
func NewService(
	repo Repository,
	parser QueryParser,
	ranker ResultRanker,
	embeddingProvider embedding.Provider,
	logs LogRepository,
	opts ...Option,
) Service {
	if logs == nil {
		logs = noopLogRepository{}
	}

	svc := &serviceImpl{
		repo:      repo,
		parser:    parser,
		ranker:    ranker,
		embedding: embeddingProvider,
		logs:      logs,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(svc)
		}
	}

	return svc
}

func (s *serviceImpl) Query(ctx context.Context, q *Query) (*Response, error) {
	if q == nil || strings.TrimSpace(q.Text) == "" {
		return nil, errors.New("query text is empty")
	}

	start := time.Now()

	parsed, err := s.parseQuery(q.Text)
	if err != nil {
		return nil, fmt.Errorf("parse query: %w", err)
	}

	if s.hasLLMProvider() {
		archived := s.lookupBestArchivedAnswer(ctx, q)

		response, directErr := s.queryWithLLM(ctx, q, parsed)
		if directErr != nil {
			s.saveLog(ctx, q.Text, 0, start, nil)

			return nil, fmt.Errorf("llm direct answer: %w", directErr)
		}

		s.saveLog(ctx, q.Text, 1, start, nil)

		metadata := map[string]any{
			"intent": parsed.Intent,
			"mode":   "direct_answer",
		}

		if archived != nil {
			response = s.applyArchivedQualitySignals(ctx, response, archived)
			metadata["ranked_by_answer_id"] = archived.ID.String()
		}

		s.persistAnswerRecord(ctx, q, response, "llm_direct", providerConfiguredModel(s.llmChain), metadata)

		return response, nil
	}

	if err := s.validateDependencies(); err != nil {
		return nil, err
	}

	s.attachEmbedding(ctx, q, parsed)

	results, total, err := s.repo.Search(ctx, parsed, q.Filters, defaultSearchOffset, defaultSearchLimit)
	if err != nil {
		s.saveLog(ctx, q.Text, 0, start, nil)

		return nil, fmt.Errorf("search repository: %w", err)
	}

	ranked := s.ranker.Rank(ctx, results, parsed)
	response := assembleResponse(ranked)
	archived := s.lookupBestArchivedAnswer(ctx, q)
	metadata := map[string]any{
		"intent":       parsed.Intent,
		"result_count": total,
	}

	if archived != nil {
		response = s.applyArchivedQualitySignals(ctx, response, archived)
		metadata["ranked_by_answer_id"] = archived.ID.String()
	}

	s.saveLog(ctx, q.Text, int(total), start, ranked)
	s.persistAnswerRecord(ctx, q, response, "retrieval", "", metadata)

	return response, nil
}

func (s *serviceImpl) lookupBestArchivedAnswer(ctx context.Context, q *Query) *AnswerRecord {
	if s.answerRecords == nil || s.feedback == nil || q == nil || strings.TrimSpace(q.Text) == "" {
		return nil
	}

	records, _, err := s.answerRecords.ListByQuery(ctx, strings.TrimSpace(q.Text), 0, 10)
	if err != nil {
		slog.WarnContext(ctx, "lookup archived answers failed", "error", err)

		return nil
	}

	var best *AnswerRecord

	bestScore := math.Inf(-1)

	for _, record := range records {
		if record == nil {
			continue
		}

		feedback, err := s.feedback.TargetFeedback(ctx, "answer", record.ID.String())
		if err != nil {
			slog.WarnContext(ctx, "lookup answer feedback failed", "answer_id", record.ID, "error", err)

			continue
		}

		score := communityScore(record.Confidence, feedback)
		if score > bestScore {
			bestScore = score
			best = record
		}
	}

	return best
}

func (s *serviceImpl) applyArchivedQualitySignals(ctx context.Context, resp *Response, record *AnswerRecord) *Response {
	if resp == nil || record == nil || s.feedback == nil {
		return resp
	}

	feedback, err := s.feedback.TargetFeedback(ctx, "answer", record.ID.String())
	if err != nil {
		return resp
	}

	if feedback.DislikeCount > feedback.LikeCount && strings.TrimSpace(record.AnswerSummary) != "" {
		resp.Answer = record.AnswerSummary + "\n\n> 社区质量信号提示：该历史答案存在较多点踩，请优先核验证据与适用条件。"
		resp.Confidence = math.Max(0.2, resp.Confidence-0.15)
	}

	resp.KnowledgeTags = appendUnique(resp.KnowledgeTags, "社区反馈排序")
	resp.RelatedQuestions = prependRelatedQuestion(resp.RelatedQuestions, RelatedQuestion{
		ID:    "community-reviewed-answer",
		Title: fmt.Sprintf("查看社区质量信号：%d 赞 / %d 踩", feedback.LikeCount, feedback.DislikeCount),
	})

	return resp
}

func communityScore(confidence float64, feedback FeedbackSummary) float64 {
	return confidence + float64(feedback.LikeCount)*0.08 - float64(feedback.DislikeCount)*0.12
}

func appendUnique(items []string, item string) []string {
	item = strings.TrimSpace(item)
	if item == "" {
		return items
	}

	if slices.Contains(items, item) {
		return items
	}

	return append(items, item)
}

func prependRelatedQuestion(items []RelatedQuestion, item RelatedQuestion) []RelatedQuestion {
	if strings.TrimSpace(item.ID) == "" {
		return items
	}

	out := make([]RelatedQuestion, 0, len(items)+1)

	out = append(out, item)
	for _, existing := range items {
		if existing.ID != item.ID {
			out = append(out, existing)
		}
	}

	return out
}

func (s *serviceImpl) parseQuery(raw string) (*ParsedQuery, error) {
	if s.parser == nil {
		cleaned := strings.TrimSpace(raw)

		return &ParsedQuery{
			Original: cleaned,
			Keywords: []string{cleaned},
			Entities: []string{cleaned},
			Intent:   IntentExplain,
		}, nil
	}

	return s.parser.Parse(raw)
}

func (s *serviceImpl) hasLLMProvider() bool {
	return s.llmChain != nil
}

func (s *serviceImpl) queryWithLLM(ctx context.Context, q *Query, parsed *ParsedQuery) (*Response, error) {
	if s.llmChain == nil {
		return nil, errors.New("no llm provider configured")
	}

	requestCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	response, err := s.llmChain.Generate(requestCtx, &llm.Request{
		Model: providerConfiguredModel(s.llmChain),
		Messages: []llm.Message{
			{Role: "system", Content: prompt.KnowledgeAnswerSystem()},
			{Role: "user", Content: buildKnowledgeAnswerUserPrompt(q, parsed)},
		},
		MaxTokens:   llm.MaxTokens128K,
		Temperature: 0.2,
	})
	if err != nil {
		return nil, err
	}

	answer := strings.TrimSpace(response.Content)
	if answer == "" {
		return nil, errors.New(s.llmChain.Name() + ": empty response")
	}

	return assembleLLMResponse(q, parsed, answer, s.llmChain.Name()), nil
}

func providerConfiguredModel(provider llm.Provider) string {
	if configured, ok := provider.(llm.ConfiguredProvider); ok {
		return configured.ConfiguredModel()
	}

	return ""
}

func buildKnowledgeAnswerUserPrompt(q *Query, parsed *ParsedQuery) string {
	input := prompt.KnowledgeAnswerInput{
		Date:     time.Now(),
		Question: q.Text,
		Subject:  q.Filters.Subject,
		Grade:    q.Filters.Grade,
	}
	if parsed != nil {
		input.Intent = parsed.Intent
		input.Entities = append([]string(nil), parsed.Entities...)
	}

	return prompt.KnowledgeAnswerUser(input)
}

func assembleLLMResponse(q *Query, parsed *ParsedQuery, answer, providerName string) *Response {
	tags := buildKnowledgeTags(q, parsed, providerName)

	return enrichLearningSearchResponse(q, parsed, &Response{
		Answer:           answer,
		KnowledgeTags:    tags,
		Citations:        buildRuntimeCitations(q, parsed),
		RelatedQuestions: buildDirectRelatedQuestions(q, parsed),
		Confidence:       directAnswerConfidence(answer),
	})
}

func buildRuntimeCitations(q *Query, parsed *ParsedQuery) []Citation {
	text := "用户问题"
	if q != nil && strings.TrimSpace(q.Text) != "" {
		text = strings.TrimSpace(q.Text)
	}

	snippet := fmt.Sprintf("当前回答由大模型基于题干“%s”和高中阶段通用知识生成；若需更高可信度，请接入课本/题库索引或补充引用。", truncateRunes(text, 42))
	if parsed != nil && len(parsed.Entities) > 0 {
		snippet += " 识别关键词：" + strings.Join(parsed.Entities, "、") + "。"
	}

	return []Citation{{DocID: "llm-runtime-grounding", SourceType: "runtime_grounding", Snippet: snippet, Score: 0.55}}
}

func enrichLearningSearchResponse(q *Query, parsed *ParsedQuery, resp *Response) *Response {
	if resp == nil {
		return resp
	}

	if len(resp.KnowledgeTags) == 0 {
		resp.KnowledgeTags = inferKnowledgeTags(q, parsed, resp.Answer)
	}

	if len(resp.Misconceptions) == 0 {
		resp.Misconceptions = inferMisconceptions(q, parsed, resp.KnowledgeTags)
	}

	if len(resp.FormulaCards) == 0 {
		resp.FormulaCards = inferFormulaCards(q, parsed, resp.KnowledgeTags)
	}

	if len(resp.ExamMappings) == 0 {
		resp.ExamMappings = inferExamMappings(q, parsed, resp.KnowledgeTags)
	}

	if len(resp.NextActions) == 0 {
		resp.NextActions = inferNextActions(q, parsed, resp.KnowledgeTags, resp.Confidence)
	}

	if len(resp.RelatedQuestions) == 0 {
		resp.RelatedQuestions = buildDirectRelatedQuestions(q, parsed)
	}

	return resp
}

func inferKnowledgeTags(q *Query, parsed *ParsedQuery, answer string) []string {
	tags := make([]string, 0, 8)
	seen := map[string]struct{}{}

	add := func(tag string) {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			return
		}

		if _, ok := seen[tag]; ok {
			return
		}

		seen[tag] = struct{}{}
		tags = append(tags, tag)
	}
	if q != nil {
		add(q.Filters.Subject)
		add(q.Filters.Grade)
	}

	if parsed != nil {
		add(intentLabel(parsed.Intent))

		for _, entity := range parsed.Entities {
			add(entity)
		}
	}

	text := strings.ToLower(strings.TrimSpace(answer))
	for _, rule := range []struct{ keyword, tag string }{
		{"牛顿", tagNewtonSecondLaw},
		{"斜面", "斜面受力"},
		{"平抛", "平抛运动"},
		{"弹簧", "弹簧振子"},
		{tagPhotosynthesis, tagPhotosynthesis},
		{"光照", "限制因素"},
		{"酶", "酶活性"},
		{"遗传", "遗传规律"},
	} {
		if strings.Contains(text, strings.ToLower(rule.keyword)) {
			add(rule.tag)
		}
	}

	if len(tags) == 0 {
		add("知识点讲解")
	}

	if len(tags) > 8 {
		return tags[:8]
	}

	return tags
}

func inferMisconceptions(q *Query, parsed *ParsedQuery, tags []string) []MisconceptionTip {
	text := searchContextText(q, parsed, tags)
	switch {
	case containsAny(text, "牛顿", "斜面", "受力", "force"):
		return []MisconceptionTip{
			{Type: "force_direction", Description: "把重力 mg 直接当成沿运动方向的合力。", Correction: "先建立坐标轴，把重力分解为沿斜面和垂直斜面的分量。"},
			{Type: "friction", Description: "摩擦力方向凭感觉判断。", Correction: "先判断相对运动或相对运动趋势，再确定摩擦方向。"},
		}
	case containsAny(text, "平抛", "projectile"):
		return []MisconceptionTip{
			{Type: "independence", Description: "认为水平速度会影响落地时间。", Correction: "忽略空气阻力时，落地时间由竖直方向高度和 g 决定。"},
			{Type: "sign", Description: "竖直方向位移方程符号混乱。", Correction: "先规定正方向，再统一使用同一套符号。"},
		}
	case containsAny(text, tagPhotosynthesis, "光照", "photosynthesis"):
		return []MisconceptionTip{
			{
				Type:        "limiting_factor",
				Description: "误以为光照越强有机物积累一定无限增加。",
				Correction:  "光照增强到一定程度后，CO₂ 浓度、温度或酶活性可能成为限制因素。",
			},
			{Type: "net_accumulation", Description: "忽略呼吸作用对净积累的消耗。", Correction: "净积累应同时考虑光合作用制造量和呼吸消耗量。"},
		}
	case containsAny(text, "酶", "enzyme"):
		return []MisconceptionTip{
			{Type: "optimal_condition", Description: "认为温度越高酶活性越强。", Correction: "酶通常有最适温度，过高会导致空间结构被破坏。"},
		}
	default:
		return []MisconceptionTip{{Type: "condition", Description: "只记结论而忽略适用条件。", Correction: "复述结论时同时说出前提、变量和限制条件。"}}
	}
}

func inferFormulaCards(q *Query, parsed *ParsedQuery, tags []string) []FormulaCard {
	text := searchContextText(q, parsed, tags)
	switch {
	case containsAny(text, "牛顿", "受力", "斜面", "force"):
		return []FormulaCard{
			{
				Name:       tagNewtonSecondLaw,
				Expression: "ΣF = ma",
				Variables:  []string{"ΣF：合外力", "m：质量", "a：加速度"},
				AppliesTo:  []string{"惯性参考系", "研究对象受力可明确"},
				Limits:     []string{"先做受力分析", "注意方向与正负号"},
			},
		}
	case containsAny(text, "平抛", "projectile"):
		return []FormulaCard{
			{
				Name:       "平抛运动分解",
				Expression: "x = v0·t,  y = h - 1/2·g·t²",
				Variables:  []string{"v0：水平初速度", "h：抛出高度", "g：重力加速度"},
				AppliesTo:  []string{"忽略空气阻力", "初速度水平"},
				Limits:     []string{"水平与竖直方向独立处理"},
			},
		}
	case containsAny(text, "弹簧", "振子", "spring"):
		return []FormulaCard{
			{
				Name:       "弹簧振子周期",
				Expression: "T = 2π√(m/k)",
				Variables:  []string{"m：振子质量", "k：劲度系数"},
				AppliesTo:  []string{"小振幅近似", "理想弹簧"},
				Limits:     []string{"阻尼较大或非线性弹簧需重新建模"},
			},
		}
	case containsAny(text, tagPhotosynthesis, "光照", "photosynthesis"):
		return []FormulaCard{
			{
				Name:       "净有机物积累",
				Expression: "净积累 ≈ 光合作用制造量 - 呼吸消耗量",
				Variables:  []string{"光照强度", "CO₂ 浓度", "温度", "酶活性"},
				AppliesTo:  []string{"高中阶段定性分析"},
				Limits:     []string{"平台期通常说明出现新的限制因素"},
			},
		}
	default:
		return nil
	}
}

func inferExamMappings(q *Query, parsed *ParsedQuery, tags []string) []ExamMapping {
	text := searchContextText(q, parsed, tags)
	switch {
	case containsAny(text, "牛顿", "受力", "斜面", "force"):
		return []ExamMapping{
			{
				QuestionType: "计算题",
				Focus:        "受力分析后列 ΣF=ma",
				PracticeHint: "先画受力图，再分解到选定坐标轴。",
				Knowledge:    []string{"受力分析", tagNewtonSecondLaw},
			},
			{QuestionType: "图像/判断题", Focus: "摩擦方向、临界条件与加速度方向", PracticeHint: "比较重力分力和最大静摩擦力。"},
		}
	case containsAny(text, "平抛", "projectile"):
		return []ExamMapping{
			{
				QuestionType: "计算题",
				Focus:        "由竖直方向求时间，再求水平位移",
				PracticeHint: "把未知量拆到 x/y 两个方向。",
				Knowledge:    []string{"运动的合成与分解", "匀变速运动"},
			},
		}
	case containsAny(text, tagPhotosynthesis, "光照", "photosynthesis"):
		return []ExamMapping{
			{
				QuestionType: "曲线题",
				Focus:        "解释上升段、平台期和限制因素",
				PracticeHint: "逐段判断哪个因素限制净光合速率。",
				Knowledge:    []string{tagPhotosynthesis, "限制因素"},
			},
			{QuestionType: "实验题", Focus: "自变量、因变量和控制变量", PracticeHint: "确认只有自变量被主动改变。"},
		}
	default:
		return []ExamMapping{{QuestionType: "概念理解题", Focus: "说清定义、适用条件和易混点", PracticeHint: "用自己的话解释并举一个例子。"}}
	}
}

func inferNextActions(q *Query, parsed *ParsedQuery, tags []string, confidence float64) []LearningAction {
	text := searchContextText(q, parsed, tags)

	actions := []LearningAction{
		{
			Type:        "review",
			Label:       "用一句话反思",
			Description: "总结当前知识点的适用条件和一个易错点。",
			Target:      "learning/reflection",
			Tags:        []string{"反思"},
		},
	}
	if confidence < 0.5 {
		actions = append(
			actions,
			LearningAction{
				Type:        "clarify",
				Label:       "补充条件后重试",
				Description: "当前可信度较低，建议补充章节、题干数值或实验条件。",
				Target:      "search",
				Tags:        []string{"低可信"},
			},
		)
	}

	switch {
	case containsAny(text, "牛顿", "受力", "平抛", "弹簧", "运动", "force", "projectile"):
		actions = append(
			actions,
			LearningAction{
				Type:        "modeling",
				Label:       "进入物理仿真实验室",
				Description: "把公式、变量和轨迹放到统一建模工作台中交互观察。",
				Target:      "physics",
				Tags:        []string{"物理仿真", "参数调节"},
			},
		)
	case containsAny(text, tagPhotosynthesis, "光照", "酶", "遗传", "生态", "biology"):
		actions = append(
			actions,
			LearningAction{
				Type:        "modeling",
				Label:       "进入生物可视化",
				Description: "生成概念关系、过程阶段和实验变量卡片。",
				Target:      "biology",
				Tags:        []string{"生物可视化", "实验变量"},
			},
		)
	default:
		actions = append(
			actions,
			LearningAction{
				Type:        "practice",
				Label:       "做一道微练习",
				Description: "用一个小题检查是否真正理解。",
				Target:      "learning/practice",
				Tags:        []string{"微练习"},
			},
		)
	}

	return actions
}

func searchContextText(q *Query, parsed *ParsedQuery, tags []string) string {
	parts := make([]string, 0, 8)
	if q != nil {
		parts = append(parts, q.Text, q.Filters.Subject, q.Filters.Grade, q.Filters.Chapter)
	}

	if parsed != nil {
		parts = append(
			parts,
			parsed.Original,
			strings.Join(parsed.Keywords, " "),
			strings.Join(parsed.Entities, " "),
			parsed.Intent,
		)
	}

	parts = append(parts, strings.Join(tags, " "))

	return strings.ToLower(strings.Join(parts, " "))
}

func buildKnowledgeTags(q *Query, parsed *ParsedQuery, providerName string) []string {
	tags := make([]string, 0, 8)
	seen := map[string]struct{}{}
	add := func(tag string) {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			return
		}

		if _, ok := seen[tag]; ok {
			return
		}

		seen[tag] = struct{}{}
		tags = append(tags, tag)
	}

	add("大模型直答")

	if providerName != "" {
		add(providerName)
	}

	if q != nil {
		add(q.Filters.Subject)
		add(q.Filters.Grade)
	}

	if parsed != nil {
		add(intentLabel(parsed.Intent))

		for _, entity := range parsed.Entities {
			add(entity)

			if len(tags) >= 8 {
				return tags
			}
		}
	}

	return tags
}

func intentLabel(intent string) string {
	switch intent {
	case IntentDefinition:
		return "概念定义"
	case IntentReason:
		return "原因解释"
	case IntentMethod:
		return "方法步骤"
	case IntentExplain:
		return "知识点讲解"
	default:
		return intent
	}
}

func buildDirectRelatedQuestions(q *Query, parsed *ParsedQuery) []RelatedQuestion {
	text := "这个知识点"
	if q != nil && strings.TrimSpace(q.Text) != "" {
		text = strings.TrimSpace(q.Text)
	}

	questions := []RelatedQuestion{
		{ID: "llm-direct-summary", Title: "用一句话总结：" + truncateRunes(text, 28)},
		{ID: "llm-direct-mistakes", Title: "这个知识点有哪些常见易错点？"},
		{ID: "llm-direct-example", Title: "给我一道相关例题并逐步讲解"},
	}

	if q != nil {
		switch strings.TrimSpace(q.Filters.Subject) {
		case "physics":
			questions = append(questions, RelatedQuestion{ID: "physics-modeling", Title: "用物理建模把这个问题可视化"})
		case "biology":
			questions = append(questions, RelatedQuestion{ID: "biology-modeling", Title: "用生物建模生成概念图"})
		}
	}

	if parsed != nil && parsed.Intent == IntentDefinition {
		questions = append(questions, RelatedQuestion{ID: "llm-direct-compare", Title: "把这个概念和相近概念做对比"})
	}

	return questions
}

func directAnswerConfidence(answer string) float64 {
	lower := strings.ToLower(answer)
	if strings.Contains(answer, "不确定") || strings.Contains(answer, "无法判断") || strings.Contains(lower, "uncertain") {
		return 0.62
	}

	return 0.82
}

func truncateRunes(text string, limit int) string {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) <= limit {
		return string(runes)
	}

	return string(runes[:limit]) + "…"
}

func (s *serviceImpl) saveLog(ctx context.Context, queryText string, total int, start time.Time, results []Result) {
	if s.logs == nil {
		return
	}

	_ = s.logs.SaveLog(ctx, &Log{
		QueryText:   queryText,
		ResultCount: total,
		LatencyMS:   int(time.Since(start).Milliseconds()),
		TopScore:    topScore(results),
	})
}

func (s *serviceImpl) persistAnswerRecord(
	ctx context.Context,
	q *Query,
	resp *Response,
	source string,
	modelName string,
	metadata map[string]any,
) {
	if s.answerRecords == nil || q == nil || resp == nil {
		return
	}

	uid := q.UserID
	if uid == uuid.Nil {
		uid = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	}

	record := &AnswerRecord{
		UserID:        uid,
		SessionID:     q.SessionID,
		Query:         strings.TrimSpace(q.Text),
		AnswerSummary: truncateRunes(resp.Answer, 1200),
		KnowledgeTags: append([]string(nil), resp.KnowledgeTags...),
		Citations:     append([]Citation(nil), resp.Citations...),
		Confidence:    resp.Confidence,
		Source:        strings.TrimSpace(source),
		ModelName:     strings.TrimSpace(modelName),
		Metadata:      metadata,
	}
	if record.Source == "" {
		record.Source = "unknown"
	}

	if err := s.answerRecords.Save(ctx, record); err != nil {
		slog.WarnContext(ctx, "persist answer record failed", "error", err)

		return
	}

	resp.AnswerID = record.ID.String()
}

func assembleResponse(results []Result) *Response {
	if len(results) == 0 {
		return enrichLearningSearchResponse(nil, nil, &Response{
			Answer:           "未找到直接匹配结果，建议补充更具体的学科、章节或关键词后再试。",
			KnowledgeTags:    nil,
			Citations:        nil,
			RelatedQuestions: nil,
			Confidence:       0.15,
		})
	}

	citations := make([]Citation, 0, minInt(len(results), 3))
	tagsSeen := map[string]struct{}{}
	tags := make([]string, 0, 6)
	related := make([]RelatedQuestion, 0, minInt(len(results), 3))

	snippets := make([]string, 0, minInt(len(results), 2))
	for i, result := range results {
		collectPrimaryResult(i, result, &citations, &related)
		collectSnippet(i, result, &snippets)
		tags = collectTags(result.Tags, tagsSeen, tags)
	}

	answer := strings.Join(snippets, "\n")
	if answer == "" {
		answer = "已检索到相关材料，但缺少足够片段用于生成摘要。"
	}

	return enrichLearningSearchResponse(nil, nil, &Response{
		Answer:           answer,
		KnowledgeTags:    tags,
		Citations:        citations,
		RelatedQuestions: related,
		Confidence:       math.Min(0.99, math.Max(0.2, topScore(results))),
	})
}

func buildRelatedTitle(result Result) string {
	parts := make([]string, 0, 3)

	for _, part := range []string{result.Subject, result.Chapter, strings.TrimSpace(result.Snippet)} {
		if part == "" {
			continue
		}

		parts = append(parts, part)
		if len(parts) == 3 {
			break
		}
	}

	title := strings.Join(parts, " · ")
	if len([]rune(title)) > maxRelatedTitleRunes {
		title = string([]rune(title)[:maxRelatedTitleRunes]) + "…"
	}

	return title
}

func topScore(results []Result) float64 {
	if len(results) == 0 {
		return 0
	}

	if results[0].Score <= 0 {
		return 0.35
	}

	return math.Min(0.99, results[0].Score)
}

func containsAny(text string, keywords ...string) bool {
	for _, keyword := range keywords {
		if strings.Contains(text, strings.ToLower(keyword)) || strings.Contains(text, keyword) {
			return true
		}
	}

	return false
}

func minInt(a, b int) int {
	if a < b {
		return a
	}

	return b
}

type noopLogRepository struct{}

func (noopLogRepository) SaveLog(context.Context, *Log) error { return nil }

func (s *serviceImpl) validateDependencies() error {
	if s.repo == nil {
		return errors.New("search repository is nil")
	}

	if s.parser == nil {
		return errors.New("search parser is nil")
	}

	if s.ranker == nil {
		return errors.New("search ranker is nil")
	}

	return nil
}

func (s *serviceImpl) attachEmbedding(ctx context.Context, q *Query, parsed *ParsedQuery) {
	if s.embedding == nil {
		return
	}

	vectors, err := s.embedding.Embed(ctx, []string{q.Text})
	if err != nil || len(vectors) == 0 {
		return
	}

	parsed.Embedding = vectors[0]
}

func collectPrimaryResult(index int, result Result, citations *[]Citation, related *[]RelatedQuestion) {
	if index >= 3 {
		return
	}

	*citations = append(*citations, Citation{
		DocID:      result.DocID,
		SourceType: result.SourceType,
		Snippet:    result.Snippet,
		Score:      result.Score,
	})
	*related = append(*related, RelatedQuestion{
		ID:    result.DocID,
		Title: buildRelatedTitle(result),
	})
}

func collectSnippet(index int, result Result, snippets *[]string) {
	if index >= 2 || strings.TrimSpace(result.Snippet) == "" {
		return
	}

	*snippets = append(*snippets, strings.TrimSpace(result.Snippet))
}

func collectTags(resultTags []string, tagsSeen map[string]struct{}, tags []string) []string {
	for _, tag := range resultTags {
		if _, ok := tagsSeen[tag]; ok || strings.TrimSpace(tag) == "" {
			continue
		}

		tagsSeen[tag] = struct{}{}

		tags = append(tags, tag)
		if len(tags) == 6 {
			return tags
		}
	}

	return tags
}
