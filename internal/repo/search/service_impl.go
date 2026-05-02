package search

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/beihai0xff/snowy/internal/repo/embedding"
	"github.com/beihai0xff/snowy/internal/repo/llm"
)

const (
	defaultSearchOffset  = 0
	defaultSearchLimit   = 8
	maxRelatedTitleRunes = 48
)

type serviceImpl struct {
	repo        Repository
	parser      QueryParser
	ranker      ResultRanker
	embedding   embedding.Provider
	logs        LogRepository
	primaryLLM  llm.Provider
	fallbackLLM llm.Provider
}

// Option 配置 Search Service 的可选依赖。
type Option func(*serviceImpl)

// WithLLMProviders 配置知识点问答的大模型直答 provider。
func WithLLMProviders(primary, fallback llm.Provider) Option {
	return func(s *serviceImpl) {
		s.primaryLLM = primary
		s.fallbackLLM = fallback
	}
}

// NewService 创建知识检索服务实现。
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
		response, directErr := s.queryWithLLM(ctx, q, parsed)
		if directErr != nil {
			response = fallbackResponse(q, parsed, fmt.Errorf("llm direct answer: %w", directErr))
		}
		s.saveLog(ctx, q.Text, 1, start, nil)

		return response, nil
	}

	if err := s.validateDependencies(); err != nil {
		return nil, err
	}

	s.attachEmbedding(ctx, q, parsed)

	results, total, err := s.repo.Search(ctx, parsed, q.Filters, defaultSearchOffset, defaultSearchLimit)
	if err != nil {
		response := fallbackResponse(q, parsed, fmt.Errorf("search repository: %w", err))
		s.saveLog(ctx, q.Text, 0, start, nil)

		return response, nil
	}

	ranked := s.ranker.Rank(ctx, results, parsed)
	response := assembleResponse(ranked)

	s.saveLog(ctx, q.Text, int(total), start, ranked)

	return response, nil
}

func (s *serviceImpl) parseQuery(raw string) (*ParsedQuery, error) {
	if s.parser == nil {
		cleaned := strings.TrimSpace(raw)
		return &ParsedQuery{Original: cleaned, Keywords: []string{cleaned}, Entities: []string{cleaned}, Intent: "explain"}, nil
	}

	return s.parser.Parse(raw)
}

func (s *serviceImpl) hasLLMProvider() bool {
	return s.primaryLLM != nil || s.fallbackLLM != nil
}

func (s *serviceImpl) queryWithLLM(ctx context.Context, q *Query, parsed *ParsedQuery) (*Response, error) {
	providers := []llm.Provider{s.primaryLLM, s.fallbackLLM}
	failures := make([]string, 0, len(providers))
	for _, provider := range providers {
		if provider == nil {
			continue
		}

		requestCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
		response, err := provider.Generate(requestCtx, &llm.Request{
			Model: providerConfiguredModel(provider),
			Messages: []llm.Message{
				{Role: "system", Content: knowledgeAnswerSystemPrompt()},
				{Role: "user", Content: buildKnowledgeAnswerUserPrompt(q, parsed)},
			},
			MaxTokens:   1024,
			Temperature: 0.2,
		})
		cancel()
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", provider.Name(), err))
			continue
		}

		answer := strings.TrimSpace(response.Content)
		if answer == "" {
			failures = append(failures, fmt.Sprintf("%s: empty response", provider.Name()))
			continue
		}

		return assembleLLMResponse(q, parsed, answer, provider.Name()), nil
	}

	if len(failures) == 0 {
		return nil, errors.New("no llm provider configured")
	}

	return nil, errors.New(strings.Join(failures, "; "))
}

func providerConfiguredModel(provider llm.Provider) string {
	if configured, ok := provider.(llm.ConfiguredProvider); ok {
		return configured.ConfiguredModel()
	}

	return ""
}

func knowledgeAnswerSystemPrompt() string {
	return strings.TrimSpace(`你是一名专业、严谨、通用的高中阶段学科辅导专家，负责直接回答学生提出的知识点、概念辨析、题目理解与学习方法问题。不依赖外部检索结果，也不要声称答案来自某个内部系统、数据库或资料库。

核心原则：
1. 直接回应用户问题；默认使用中文，用户明确指定其他语言时跟随用户。
2. 面向高中生，表达准确、清晰、循序渐进；先给结论，再解释关键概念、适用条件、公式/机制和典型例子。
3. 不编造教材页码、论文、链接、实验数据或“检索到的资料”；不确定的内容要明确标注不确定性，并给出可验证或继续追问的方向。
4. 题目信息不足时，先指出缺失条件，再给出通用分析框架、可能情形和下一步需要补充的信息。
5. 数学、物理、化学问题要保留必要公式、符号含义、单位和适用条件；生物问题要突出结构、过程、变量、因果链和实验设计逻辑。
6. 不展示隐藏推理或冗长思维链；可以展示面向学习者的简洁推导步骤、解题流程或判断依据。
7. 语气专业、耐心、中立；避免品牌名、平台名、内部链路、供应商或实现细节等无关信息。

回答结构应紧凑：
- 结论：1-2 句话直接回答。
- 关键点：3-4 条解释核心知识。
- 必要步骤：按学科需要给出简洁公式、过程或分析路径。
- 易错点/继续追问：指出 1-2 个边界条件或追问方向。
总长度通常控制在 600-900 中文字以内，除非用户明确要求详细展开。`)
}

func buildKnowledgeAnswerUserPrompt(q *Query, parsed *ParsedQuery) string {
	var builder strings.Builder
	builder.WriteString("请直接回答这个知识点问题，不要进行数据库检索，不要输出 JSON。\n")
	builder.WriteString("当前日期：")
	builder.WriteString(time.Now().Format("2006-01-02"))
	builder.WriteString("\n")
	builder.WriteString("问题：")
	builder.WriteString(strings.TrimSpace(q.Text))
	builder.WriteString("\n")
	if strings.TrimSpace(q.Filters.Subject) != "" {
		builder.WriteString("学科：")
		builder.WriteString(q.Filters.Subject)
		builder.WriteString("\n")
	}
	if strings.TrimSpace(q.Filters.Grade) != "" {
		builder.WriteString("年级：")
		builder.WriteString(q.Filters.Grade)
		builder.WriteString("\n")
	}
	if parsed != nil {
		if parsed.Intent != "" {
			builder.WriteString("问题意图：")
			builder.WriteString(parsed.Intent)
			builder.WriteString("\n")
		}
		if len(parsed.Entities) > 0 {
			builder.WriteString("识别到的关键词：")
			builder.WriteString(strings.Join(parsed.Entities, "、"))
			builder.WriteString("\n")
		}
	}
	builder.WriteString("回答要具体但保持紧凑，不要只给定义；如果涉及公式，请说明符号含义和适用条件；总长度通常控制在 600-900 中文字以内。")

	return builder.String()
}

func assembleLLMResponse(q *Query, parsed *ParsedQuery, answer, providerName string) *Response {
	return &Response{
		Answer:           answer,
		KnowledgeTags:    buildKnowledgeTags(q, parsed, providerName),
		Citations:        []Citation{},
		RelatedQuestions: buildDirectRelatedQuestions(q, parsed),
		Confidence:       directAnswerConfidence(answer),
	}
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
	case "definition":
		return "概念定义"
	case "reason":
		return "原因解释"
	case "method":
		return "方法步骤"
	case "explain":
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
		{ID: "llm-direct-summary", Title: fmt.Sprintf("用一句话总结：%s", truncateRunes(text, 28))},
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
	if parsed != nil && parsed.Intent == "definition" {
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

func fallbackResponse(q *Query, parsed *ParsedQuery, cause error) *Response {
	text := "这个问题"
	if q != nil && strings.TrimSpace(q.Text) != "" {
		text = strings.TrimSpace(q.Text)
	}

	tags := []string{"本地兜底", "知识检索"}
	if q != nil && strings.TrimSpace(q.Filters.Subject) != "" {
		tags = append(tags, q.Filters.Subject)
	}
	if parsed != nil && parsed.Intent != "" {
		tags = append(tags, parsed.Intent)
	}

	answer := fmt.Sprintf("当前大模型知识问答暂时不可用。你可以围绕“%s”补充学科、章节、已知条件，或稍后重试。", text)
	if cause != nil && strings.Contains(cause.Error(), "search repository") {
		answer = fmt.Sprintf("当前知识索引暂时不可用或没有命中结果。你可以围绕“%s”补充学科、章节、已知条件，或直接跳转到物理/生物建模继续分析。", text)
	}
	if cause != nil {
		answer += " 诊断信息：" + cause.Error()
	}

	return &Response{
		Answer:        answer,
		KnowledgeTags: tags,
		Citations: []Citation{
			{
				DocID:      "local-fallback",
				SourceType: "fallback",
				Snippet:    "OpenSearch 无可用命中时返回的本地兜底说明。",
				Score:      0.15,
			},
		},
		RelatedQuestions: []RelatedQuestion{
			{ID: "physics-modeling", Title: "用物理建模继续分析这个问题"},
			{ID: "biology-modeling", Title: "用生物建模继续分析这个问题"},
		},
		Confidence: 0.15,
	}
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

func assembleResponse(results []Result) *Response {
	if len(results) == 0 {
		return &Response{
			Answer:           "未找到直接匹配结果，建议补充更具体的学科、章节或关键词后再试。",
			KnowledgeTags:    nil,
			Citations:        nil,
			RelatedQuestions: nil,
			Confidence:       0.15,
		}
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

	return &Response{
		Answer:           answer,
		KnowledgeTags:    tags,
		Citations:        citations,
		RelatedQuestions: related,
		Confidence:       math.Min(0.99, math.Max(0.2, topScore(results))),
	}
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
