// Package monitoring provides lightweight in-process observability for LLM calls.
package monitoring

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/beihai0xff/snowy/internal/pkg/common"
	"github.com/beihai0xff/snowy/internal/pkg/config"
	"github.com/beihai0xff/snowy/internal/repo/llm"
)

const defaultMaxLLMRecords = 200

// LLMProviderConfig is a sanitized provider configuration exposed to the monitoring UI.
type LLMProviderConfig struct {
	Role             string `json:"role"`
	Provider         string `json:"provider"`
	ModelProvider    string `json:"model_provider,omitempty"`
	Model            string `json:"model"`
	BaseURL          string `json:"base_url,omitempty"`
	Timeout          string `json:"timeout,omitempty"`
	MaxRetries       int    `json:"max_retries"`
	Configured       bool   `json:"configured"`
	APIKeyConfigured bool   `json:"api_key_configured"`
}

// LLMPromptProfile describes a prompt engineering profile used by an LLM path.
type LLMPromptProfile struct {
	ID                 string         `json:"id"`
	Scene              string         `json:"scene"`
	Version            string         `json:"version"`
	Mode               string         `json:"mode"`
	Title              string         `json:"title"`
	SystemPE           string         `json:"system_pe"`
	UserPromptContract string         `json:"user_prompt_contract"`
	SuccessChecklist   []string       `json:"success_checklist"`
	GenerationParams   map[string]any `json:"generation_params,omitempty"`
	UpdatedAt          time.Time      `json:"updated_at"`
}

// LLMCallRecord is a single observed LLM call.
type LLMCallRecord struct {
	ID            string    `json:"id"`
	Role          string    `json:"role"`
	Provider      string    `json:"provider"`
	ModelProvider string    `json:"model_provider,omitempty"`
	Model         string    `json:"model"`
	BaseURL       string    `json:"base_url,omitempty"`
	Operation     string    `json:"operation"`
	Status        string    `json:"status"`
	LatencyMS     int64     `json:"latency_ms"`
	InputTokens   int       `json:"input_tokens"`
	OutputTokens  int       `json:"output_tokens"`
	MaxTokens     int       `json:"max_tokens"`
	Temperature   float64   `json:"temperature"`
	PromptChars   int       `json:"prompt_chars"`
	SystemPE      string    `json:"system_pe,omitempty"`
	UserPrompt    string    `json:"user_prompt,omitempty"`
	PromptPreview string    `json:"prompt_preview,omitempty"`
	UserID        string    `json:"user_id,omitempty"`
	FinishReason  string    `json:"finish_reason,omitempty"`
	Error         string    `json:"error,omitempty"`
	StartedAt     time.Time `json:"started_at"`
	FinishedAt    time.Time `json:"finished_at"`
}

// LLMSummary aggregates LLM call metrics.
type LLMSummary struct {
	TotalCalls        int        `json:"total_calls"`
	SuccessCalls      int        `json:"success_calls"`
	FailedCalls       int        `json:"failed_calls"`
	SuccessRate       float64    `json:"success_rate"`
	AvgLatencyMS      float64    `json:"avg_latency_ms"`
	P50LatencyMS      int64      `json:"p50_latency_ms"`
	P95LatencyMS      int64      `json:"p95_latency_ms"`
	MaxLatencyMS      int64      `json:"max_latency_ms"`
	TotalInputTokens  int        `json:"total_input_tokens"`
	TotalOutputTokens int        `json:"total_output_tokens"`
	AvgPromptChars    float64    `json:"avg_prompt_chars"`
	LastError         string     `json:"last_error,omitempty"`
	LastCallAt        *time.Time `json:"last_call_at,omitempty"`
}

// LLMGroupMetric aggregates metrics by provider or operation.
type LLMGroupMetric struct {
	Key               string  `json:"key"`
	TotalCalls        int     `json:"total_calls"`
	SuccessCalls      int     `json:"success_calls"`
	FailedCalls       int     `json:"failed_calls"`
	SuccessRate       float64 `json:"success_rate"`
	AvgLatencyMS      float64 `json:"avg_latency_ms"`
	P95LatencyMS      int64   `json:"p95_latency_ms"`
	TotalInputTokens  int     `json:"total_input_tokens"`
	TotalOutputTokens int     `json:"total_output_tokens"`
}

// LLMDashboard is the response returned to the frontend monitoring dashboard.
type LLMDashboard struct {
	GeneratedAt    time.Time           `json:"generated_at"`
	Summary        LLMSummary          `json:"summary"`
	Providers      []LLMProviderConfig `json:"providers"`
	ByProvider     []LLMGroupMetric    `json:"by_provider"`
	ByOperation    []LLMGroupMetric    `json:"by_operation"`
	RecentCalls    []LLMCallRecord     `json:"recent_calls"`
	PromptProfiles []LLMPromptProfile  `json:"prompt_profiles"`
}

// LLMRecorder stores recent LLM calls in memory. It intentionally avoids secrets.
type LLMRecorder struct {
	mu             sync.RWMutex
	maxRecords     int
	records        []LLMCallRecord
	store          LLMCallRecordStore
	providers      []LLMProviderConfig
	promptProfiles []LLMPromptProfile
}

// RecorderOption configures an LLMRecorder.
type RecorderOption func(*LLMRecorder)

func WithMaxRecords(maxRecords int) RecorderOption {
	return func(r *LLMRecorder) {
		if maxRecords > 0 {
			r.maxRecords = maxRecords
		}
	}
}

func WithProviderConfigs(providers ...LLMProviderConfig) RecorderOption {
	return func(r *LLMRecorder) {
		r.providers = append([]LLMProviderConfig(nil), providers...)
	}
}

func WithPromptProfiles(profiles ...LLMPromptProfile) RecorderOption {
	return func(r *LLMRecorder) {
		r.promptProfiles = append([]LLMPromptProfile(nil), profiles...)
	}
}

func WithStore(store LLMCallRecordStore) RecorderOption {
	return func(r *LLMRecorder) {
		r.store = store
	}
}

// NewLLMRecorder creates an in-process recorder.
func NewLLMRecorder(opts ...RecorderOption) *LLMRecorder {
	r := &LLMRecorder{maxRecords: defaultMaxLLMRecords}
	for _, opt := range opts {
		opt(r)
	}

	return r
}

// Record stores one LLM call record.
func (r *LLMRecorder) Record(record LLMCallRecord) {
	if r == nil {
		return
	}

	if record.ID == "" {
		record.ID = uuid.NewString()
	}

	if record.StartedAt.IsZero() {
		record.StartedAt = time.Now()
	}

	if record.FinishedAt.IsZero() {
		record.FinishedAt = record.StartedAt
	}

	r.mu.Lock()
	r.records = append([]LLMCallRecord{record}, r.records...)
	if len(r.records) > r.maxRecords {
		r.records = r.records[:r.maxRecords]
	}
	store := r.store
	r.mu.Unlock()

	if store != nil {
		go func() { _ = store.Save(context.Background(), record) }()
	}
}

// Dashboard returns a snapshot for the monitoring UI.
func (r *LLMRecorder) Dashboard(filter ...LLMRecordFilter) LLMDashboard {
	if r == nil {
		return LLMDashboard{GeneratedAt: time.Now()}
	}

	r.mu.RLock()

	records := append([]LLMCallRecord(nil), r.records...)
	if records == nil {
		records = []LLMCallRecord{}
	}

	providers := append([]LLMProviderConfig(nil), r.providers...)
	if providers == nil {
		providers = []LLMProviderConfig{}
	}

	profiles := append([]LLMPromptProfile(nil), r.promptProfiles...)
	if profiles == nil {
		profiles = []LLMPromptProfile{}
	}

	store := r.store
	r.mu.RUnlock()

	if store != nil && len(filter) > 0 {
		if stored, err := store.List(context.Background(), filter[0]); err == nil {
			records = stored
		}
	}

	return LLMDashboard{
		GeneratedAt:    time.Now(),
		Summary:        summarize(records),
		Providers:      providers,
		ByProvider:     groupBy(records, func(r LLMCallRecord) string { return r.Provider + "/" + r.Model }),
		ByOperation:    groupBy(records, func(r LLMCallRecord) string { return r.Operation }),
		RecentCalls:    records,
		PromptProfiles: profiles,
	}
}

func summarize(records []LLMCallRecord) LLMSummary {
	var summary LLMSummary
	if len(records) == 0 {
		return summary
	}

	latencies := make([]int64, 0, len(records))

	var (
		latencyTotal     int64
		promptCharsTotal int
	)

	for _, record := range records {
		summary.TotalCalls++
		if record.Status == "success" {
			summary.SuccessCalls++
		} else {
			summary.FailedCalls++
			if summary.LastError == "" && strings.TrimSpace(record.Error) != "" {
				summary.LastError = record.Error
			}
		}

		latencies = append(latencies, record.LatencyMS)

		latencyTotal += record.LatencyMS
		if record.LatencyMS > summary.MaxLatencyMS {
			summary.MaxLatencyMS = record.LatencyMS
		}

		summary.TotalInputTokens += record.InputTokens
		summary.TotalOutputTokens += record.OutputTokens

		promptCharsTotal += record.PromptChars
		if summary.LastCallAt == nil || record.FinishedAt.After(*summary.LastCallAt) {
			lastCallAt := record.FinishedAt
			summary.LastCallAt = &lastCallAt
		}
	}

	summary.SuccessRate = ratio(summary.SuccessCalls, summary.TotalCalls)
	summary.AvgLatencyMS = float64(latencyTotal) / float64(summary.TotalCalls)
	summary.AvgPromptChars = float64(promptCharsTotal) / float64(summary.TotalCalls)
	summary.P50LatencyMS = percentile(latencies, 0.50)
	summary.P95LatencyMS = percentile(latencies, 0.95)

	return summary
}

func groupBy(records []LLMCallRecord, keyFn func(LLMCallRecord) string) []LLMGroupMetric {
	type acc struct {
		metric    LLMGroupMetric
		latencies []int64
		latency   int64
	}

	groups := map[string]*acc{}

	for _, record := range records {
		key := strings.TrimSpace(keyFn(record))
		if key == "" || key == "/" {
			key = "unknown"
		}

		item := groups[key]
		if item == nil {
			item = &acc{metric: LLMGroupMetric{Key: key}}
			groups[key] = item
		}

		item.metric.TotalCalls++
		if record.Status == "success" {
			item.metric.SuccessCalls++
		} else {
			item.metric.FailedCalls++
		}

		item.latencies = append(item.latencies, record.LatencyMS)
		item.latency += record.LatencyMS
		item.metric.TotalInputTokens += record.InputTokens
		item.metric.TotalOutputTokens += record.OutputTokens
	}

	metrics := make([]LLMGroupMetric, 0, len(groups))
	for _, item := range groups {
		item.metric.SuccessRate = ratio(item.metric.SuccessCalls, item.metric.TotalCalls)
		if item.metric.TotalCalls > 0 {
			item.metric.AvgLatencyMS = float64(item.latency) / float64(item.metric.TotalCalls)
		}

		item.metric.P95LatencyMS = percentile(item.latencies, 0.95)
		metrics = append(metrics, item.metric)
	}

	sort.Slice(metrics, func(i, j int) bool {
		if metrics[i].TotalCalls == metrics[j].TotalCalls {
			return metrics[i].Key < metrics[j].Key
		}

		return metrics[i].TotalCalls > metrics[j].TotalCalls
	})

	return metrics
}

func percentile(values []int64, p float64) int64 {
	if len(values) == 0 {
		return 0
	}

	sorted := append([]int64(nil), values...)
	slices.Sort(sorted)

	idx := max(int(float64(len(sorted)-1)*p), 0)

	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}

	return sorted[idx]
}

func ratio(part, total int) float64 {
	if total <= 0 {
		return 0
	}

	return float64(part) / float64(total)
}

// ObservedProvider wraps an LLM provider and records Generate/GenerateStream metrics.
type ObservedProvider struct {
	next     llm.Provider
	recorder *LLMRecorder
	role     string
}

// WrapProvider returns an observed provider wrapper. Nil providers stay nil.
func WrapProvider(provider llm.Provider, recorder *LLMRecorder, role string) llm.Provider {
	if provider == nil || recorder == nil {
		return provider
	}

	return &ObservedProvider{next: provider, recorder: recorder, role: role}
}

func (p *ObservedProvider) Generate(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	if req == nil {
		err := errors.New("llm request is nil")
		p.record(ctx, req, nil, err, time.Now(), time.Now())

		return nil, err
	}

	start := time.Now()
	resp, err := p.next.Generate(ctx, req)
	p.record(ctx, req, resp, err, start, time.Now())

	return resp, err
}

func (p *ObservedProvider) GenerateStream(ctx context.Context, req *llm.Request, chunks chan<- llm.StreamChunk) error {
	if req == nil {
		err := errors.New("llm stream request is nil")
		p.record(ctx, req, nil, err, time.Now(), time.Now())

		return err
	}

	start := time.Now()
	err := p.next.GenerateStream(ctx, req, chunks)
	p.record(ctx, req, nil, err, start, time.Now())

	return err
}

func (p *ObservedProvider) HealthCheck(ctx context.Context) error { return p.next.HealthCheck(ctx) }
func (p *ObservedProvider) EstimateCost(ctx context.Context, req *llm.Request) (*llm.Cost, error) {
	return p.next.EstimateCost(ctx, req)
}
func (p *ObservedProvider) Name() string { return p.next.Name() }

func (p *ObservedProvider) ConfiguredModel() string {
	if configured, ok := p.next.(llm.ConfiguredProvider); ok {
		return configured.ConfiguredModel()
	}

	return ""
}

func (p *ObservedProvider) ConfiguredBaseURL() string {
	if configured, ok := p.next.(llm.ConfiguredProvider); ok {
		return configured.ConfiguredBaseURL()
	}

	return ""
}

func (p *ObservedProvider) ConfiguredModelProvider() string {
	if configured, ok := p.next.(llm.ConfiguredProvider); ok {
		return configured.ConfiguredModelProvider()
	}

	return ""
}

func (p *ObservedProvider) record(ctx context.Context, req *llm.Request, resp *llm.Response, callErr error, start, finish time.Time) {
	if p == nil || p.recorder == nil {
		return
	}

	status := "success"
	if callErr != nil {
		status = "failed"
	}

	model := ""
	maxTokens := 0
	temperature := 0.0
	messages := []llm.Message(nil)

	if req != nil {
		model = strings.TrimSpace(req.Model)
		maxTokens = req.MaxTokens
		temperature = req.Temperature
		messages = req.Messages
	}

	if model == "" {
		model = p.ConfiguredModel()
	}

	if resp != nil && strings.TrimSpace(resp.Model) != "" {
		model = strings.TrimSpace(resp.Model)
	}

	systemPE, userPrompt, promptChars := extractPrompts(messages)

	record := LLMCallRecord{
		ID:            uuid.NewString(),
		Role:          p.role,
		Provider:      p.next.Name(),
		ModelProvider: p.ConfiguredModelProvider(),
		Model:         model,
		BaseURL:       sanitizeBaseURL(p.ConfiguredBaseURL()),
		Operation:     inferOperation(systemPE, userPrompt),
		Status:        status,
		LatencyMS:     finish.Sub(start).Milliseconds(),
		MaxTokens:     maxTokens,
		Temperature:   temperature,
		PromptChars:   promptChars,
		UserID:        userIDFromContext(ctx),
		SystemPE:      truncate(systemPE, 2400),
		UserPrompt:    truncate(userPrompt, 2400),
		PromptPreview: truncate(joinPromptPreview(systemPE, userPrompt), 1200),
		StartedAt:     start,
		FinishedAt:    finish,
	}
	if resp != nil {
		record.InputTokens = resp.InputTokens
		record.OutputTokens = resp.OutputTokens
		record.FinishReason = resp.FinishReason
	}

	if callErr != nil {
		record.Error = truncate(callErr.Error(), 1200)
	}

	p.recorder.Record(record)
}

func extractPrompts(messages []llm.Message) (string, string, int) {
	var (
		systemParts []string
		userParts   []string
	)

	chars := 0

	for _, message := range messages {
		content := strings.TrimSpace(message.Content)
		chars += len([]rune(content))

		switch strings.ToLower(strings.TrimSpace(message.Role)) {
		case "system":
			systemParts = append(systemParts, content)
		case "user":
			userParts = append(userParts, content)
		}
	}

	return strings.Join(systemParts, "\n\n"), strings.Join(userParts, "\n\n"), chars
}

func joinPromptPreview(systemPE, userPrompt string) string {
	parts := make([]string, 0, 2)
	if strings.TrimSpace(systemPE) != "" {
		parts = append(parts, "System PE:\n"+strings.TrimSpace(systemPE))
	}

	if strings.TrimSpace(userPrompt) != "" {
		parts = append(parts, "User Prompt:\n"+strings.TrimSpace(userPrompt))
	}

	return strings.Join(parts, "\n\n---\n")
}

func inferOperation(systemPE, userPrompt string) string {
	combined := strings.ToLower(systemPE + "\n" + userPrompt)
	switch {
	case strings.Contains(combined, "交互式科学可视化前端工程师") || strings.Contains(combined, "code_bundle") || strings.Contains(combined, "biology_*"):
		return "render_generation_pe"
	case strings.Contains(combined, "学科辅导专家") || strings.Contains(combined, "知识点问题") || strings.Contains(combined, "概念辨析"):
		return "knowledge_answer_pe"
	case strings.Contains(combined, "scene_spec"):
		return "scene_spec_render"
	default:
		return "llm_generate"
	}
}

func truncate(text string, limit int) string {
	text = strings.TrimSpace(text)

	if limit <= 0 {
		return ""
	}

	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}

	return string(runes[:limit]) + "…"
}

func sanitizeBaseURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}

	parsed.User = nil
	parsed.RawQuery = ""
	parsed.Fragment = ""

	return strings.TrimRight(parsed.String(), "/")
}

// ProviderConfigFromConfig converts provider config into a safe dashboard shape.
func ProviderConfigFromConfig(role string, cfg config.ModelProviderConfig) LLMProviderConfig {
	provider := strings.TrimSpace(cfg.Provider)
	model := cfg.EffectiveModel()
	baseURL := sanitizeBaseURL(cfg.EffectiveBaseURL())

	return LLMProviderConfig{
		Role:             role,
		Provider:         provider,
		ModelProvider:    strings.TrimSpace(cfg.ModelProvider),
		Model:            model,
		BaseURL:          baseURL,
		Timeout:          cfg.Timeout.String(),
		MaxRetries:       cfg.MaxRetries,
		Configured:       provider != "" && model != "" && baseURL != "",
		APIKeyConfigured: providerAPIKeyConfigured(role, provider, cfg.APIKey),
	}
}

func providerAPIKeyConfigured(role, provider, cfgKey string) bool {
	if strings.TrimSpace(cfgKey) != "" {
		return true
	}

	role = strings.ToUpper(strings.TrimSpace(role))
	provider = strings.ToLower(strings.TrimSpace(provider))
	keys := []string{fmt.Sprintf("SNOWY_LLM_%s_API_KEY", role)}

	switch provider {
	case "mimo", "xiaomi", "xiaomi-mimo":
		keys = append(keys, "MIMO_API_KEY", "XIAOMI_MIMO_API_KEY")
	case "openai":
		keys = append(keys, "OPENAI_API_KEY")
	case "google", "gemini":
		keys = append(keys, "GEMINI_API_KEY", "GOOGLE_API_KEY")
	}

	for _, key := range keys {
		if strings.TrimSpace(os.Getenv(key)) != "" {
			return true
		}
	}

	return false
}

// DefaultPromptProfiles returns safe prompt engineering profiles for the dashboard.
func DefaultPromptProfiles(now time.Time) []LLMPromptProfile {
	if now.IsZero() {
		now = time.Now()
	}

	return []LLMPromptProfile{
		{
			ID:      "knowledge-answer-direct-v2",
			Scene:   "search",
			Version: "v2-direct-llm",
			Mode:    "knowledge_answer_pe",
			Title:   "知识点直答 PE",
			SystemPE: strings.TrimSpace(
				`你是一名专业、严谨、通用的高中阶段学科辅导专家，负责直接回答学生提出的知识点、概念辨析、题目理解与学习方法问题。不依赖外部检索结果，也不要声称答案来自某个内部系统、数据库或资料库。
回答原则：先给结论，再解释关键概念、适用条件、公式/机制和典型例子；不编造教材页码、论文、链接、实验数据或“检索到的资料”；信息不足时说明缺失条件并给通用分析框架；不展示隐藏推理；语气专业、耐心、中立，避免品牌名、平台名、内部链路、供应商或实现细节等无关信息。`,
			),
			UserPromptContract: "注入当前日期、用户问题、学科/年级筛选、解析到的意图与关键词；要求直接回答，不输出 JSON，涉及公式需说明符号含义、单位和适用条件。",
			SuccessChecklist: []string{
				"结论明确且适合高中生",
				"公式、单位、适用条件清楚",
				"不伪造引用或内部来源声明",
				"包含易错点和下一步追问",
			},
			GenerationParams: map[string]any{"temperature": 0.35, "max_tokens": llm.MaxTokens128K},
			UpdatedAt:        now,
		},
		{
			ID:      "biology-render-visual-v3",
			Scene:   "biology_render",
			Version: "v3-visual-demo",
			Mode:    "render_generation_pe",
			Title:   "生物可视化演示 PE",
			SystemPE: strings.TrimSpace(
				`你是一名专业的交互式科学可视化前端工程师。目标是根据 biology_* scene_spec 生成可在无网络 iframe sandbox 中运行的原生 HTML/CSS/JavaScript 教学演示页。
输出必须是合法 JSON，code_bundle.index.html 必须完整；禁止 fetch、XMLHttpRequest、localStorage、WebSocket、外链脚本和动态 import；必须遵循指定的预览通信协议并发送 ready/error 状态。视觉要求：高对比舞台、渐变/霓虹高光、粒子/流动路径、阶段切换、概念标签、过程箭头、解释面板、播放/暂停或自动动画。`,
			),
			UserPromptContract: "传入 scene_spec 和 render_mode；强调只输出 JSON、不使用 markdown、不省略 code_bundle、不输出占位符，代码包长度不设上限。",
			SuccessChecklist: []string{
				"code_bundle.index.html 完整可运行",
				"遵循预览通信协议并发送 ready 状态",
				"Canvas 2D/WebGL 离线渲染，无外链依赖",
				"体现 particle/flow/stage/label 等动态可视化语义",
			},
			GenerationParams: map[string]any{"temperature": 0.15, "max_tokens": llm.MaxTokens128K},
			UpdatedAt:        now,
		},
		{
			ID:      "physics-native-rapier-v1",
			Scene:   "physics",
			Version: "v1-native-engine",
			Mode:    "native_physics_engine",
			Title:   "物理原生引擎解析 PE",
			SystemPE: strings.TrimSpace(
				`你是一名专业、严谨的高中物理建模辅导专家。物理题目由本地 Rapier 3D 引擎确定性完成仿真与渲染；大模型只负责题意解析、参数抽取、步骤讲解和 scene_spec 组织，不生成可执行前端代码。回答应突出物理规律、变量关系、单位、适用条件和可视化参数含义。`,
			),
			UserPromptContract: "输入题干和会话上下文；输出模型类型、条件、参数、推导步骤、讲解与 scene_spec；禁止生成前端代码。规则解析作为兜底能力。",
			SuccessChecklist: []string{
				"scene_spec 可驱动 force_3d / projectile / motion 等预览",
				"参数包含质量、力、速度、角度、时间、重力等可调项",
				"解释中明确 F=ma、运动分解或对应物理规律",
				"解析失败时规则兜底仍可显示物理仿真",
			},
			GenerationParams: map[string]any{"runtime": "rapier3d", "llm_code_generation": false},
			UpdatedAt:        now,
		},
	}
}

func userIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	return strings.TrimSpace(common.UserIDFromContext(ctx))
}
