// Package prompt centralizes versioned system prompts and user prompt renderers.
//
//nolint:goconst,lll // Prompt contracts intentionally contain repeated schema words and long textual constraints.
package prompt

import (
	"encoding/json"
	"strings"
	"time"
)

const maxTokens128K = 131072

const (
	SceneSearchDirect          = "search.direct"
	SceneModelingCompile       = "modeling.compile"
	SceneRenderGeneration      = "render.generation"
	SceneRegenerateClassifier  = "agent.regenerate_classifier"
	ScenePhysicsNativeAnalysis = "physics.native_analysis"
	SceneBiologyRender         = "biology.render"
)

const (
	VersionSearchDirect         = "search.direct.v3"
	VersionModelingCompile      = "modeling.compile.v8"
	VersionRenderGeneration     = "render.generation.v4"
	VersionRegenerateClassifier = "agent.regenerate_classifier.v2"
	VersionPhysicsNative        = "physics.native.v1"
	VersionBiologyRender        = "biology.render.v4"
)

// Profile describes an operator-visible prompt profile.
type Profile struct {
	ID                 string
	Scene              string
	Version            string
	Mode               string
	Title              string
	System             string
	UserPromptContract string
	SuccessChecklist   []string
	GenerationParams   map[string]any
	UpdatedAt          time.Time
}

// KnowledgeAnswerInput renders the search direct-answer user prompt.
type KnowledgeAnswerInput struct {
	Date     time.Time
	Question string
	Subject  string
	Grade    string
	Intent   string
	Entities []string
}

// ModelingCompileInput renders the generative model compiler user prompt.
type ModelingCompileInput struct {
	Domain     string
	GradeBand  string
	TargetMode string
	Message    string
	Context    any
	Evidence   any
}

// RenderInput renders the browser-visualization user prompt.
type RenderInput struct {
	SceneSpec any
	Mode      string
	Biology   bool
}

func KnowledgeAnswerSystem() string {
	return strings.TrimSpace(`你是一名专业、严谨、通用的高中阶段学科辅导专家，负责直接回答学生提出的知识点、概念辨析、题目理解与学习方法问题。

核心原则：
1. 先给结论，再给学科解释、必要公式/机制、易错点和下一步追问方向。
2. 面向高中生，表达准确、清晰、循序渐进；数学、物理、化学必须说明符号含义、单位和适用条件；生物必须突出结构、过程、变量、因果链和实验设计逻辑。
3. 不依赖外部检索结果，不声称答案来自内部系统、数据库或资料库。
4. 不编造教材页码、论文、链接、实验数据或引用；不确定内容必须明确标注，并给出可验证或继续追问方向。
5. 信息不足时，先指出缺失条件，再给通用分析框架和可能情形。
6. 不展示隐藏推理或冗长思维链；只展示适合学生学习的简洁推导步骤、解题流程或判断依据。
7. 默认中文回答，用户明确指定其他语言时跟随用户；语气专业、耐心、中立，避免品牌名、平台名、内部链路、供应商或实现细节。

回答结构：
- 结论：1-2 句话直接回答。
- 关键点：3-4 条解释核心知识。
- 必要步骤：按学科需要给出公式、过程或分析路径。
- 易错点/继续追问：指出 1-2 个边界条件或追问方向。
总长度通常控制在 600-900 中文字以内，除非用户明确要求详细展开。`)
}

func KnowledgeAnswerUser(input KnowledgeAnswerInput) string {
	var builder strings.Builder
	builder.WriteString("请直接回答这个知识点问题，不要进行数据库检索，不要输出 JSON。\n")

	date := input.Date
	if date.IsZero() {
		date = time.Now()
	}

	builder.WriteString("当前日期：")
	builder.WriteString(date.Format("2006-01-02"))
	builder.WriteString("\n问题：")
	builder.WriteString(strings.TrimSpace(input.Question))
	builder.WriteString("\n")

	if strings.TrimSpace(input.Subject) != "" {
		builder.WriteString("学科：")
		builder.WriteString(input.Subject)
		builder.WriteString("\n")
	}

	if strings.TrimSpace(input.Grade) != "" {
		builder.WriteString("年级：")
		builder.WriteString(input.Grade)
		builder.WriteString("\n")
	}

	if strings.TrimSpace(input.Intent) != "" {
		builder.WriteString("问题意图：")
		builder.WriteString(input.Intent)
		builder.WriteString("\n")
	}

	if len(input.Entities) > 0 {
		builder.WriteString("识别到的关键词：")
		builder.WriteString(strings.Join(input.Entities, "、"))
		builder.WriteString("\n")
	}

	builder.WriteString("回答要具体但保持紧凑，不要只给定义；如果涉及公式，请说明符号含义和适用条件；不要伪造引用；总长度通常控制在 600-900 中文字以内。")

	return builder.String()
}

func ModelingCompileSystem() string {
	return strings.TrimSpace(`你是 Snowy 的版本化生成式科学建模编译器。请根据用户问题、上下文和证据，输出一个面向高中生的 GenerativeModelPackage JSON。

硬性要求：
1. 只输出合法 JSON，不要 Markdown，不要代码块，不要解释文字。
2. 不展示隐藏思维链；reasoning_trace 只写给学生看的简洁推理摘要、关键步骤和可信度。
3. domain 只能是 physics、biology 或 chemistry；字段必须使用 snake_case，并匹配用户提供的契约。
4. physics 必须输出 simulation_logic，包含变量、单位、公式、渲染说明、交互计划、本地重算变量和 LLM regenerate 条件。
5. biology 必须输出 visualization_graph，包含概念节点、关系边、过程阶段、实验变量、曲线解释或限制因素。
6. chemistry 必须输出 balanced equation / reaction type / species / animation / interaction controls；所有方程必须满足质量守恒，不确定时在 warnings 和 validation_report 中标注。
7. 区分本地 recompute 与 LLM regenerate：数值参数变化优先支持 local_recompute，结构性变化才进入 llm_regenerate。
8. 所有知识性结论尽量绑定 evidence_refs；证据不足时在 warnings 和 validation_report 中标记低可信。
9. validation_report 必须可审计，至少覆盖 schema/domain/evidence/safety 结果。`)
}

func ModelingCompileUser(input ModelingCompileInput) string {
	payload := map[string]any{
		"contract": map[string]any{
			"package_id":          "uuid string optional",
			"domain":              "physics|biology|chemistry",
			"question":            "string",
			"learning_model":      "object",
			"evidence_refs":       "array",
			"reasoning_trace":     "object",
			"generative_model":    "object",
			"simulation_logic":    "object|null",
			"visualization_graph": "object|null",
			"interaction_plan":    "object",
			"assessment_tasks":    "array",
			"validation_report":   "object",
			"regeneration_hints":  "array",
			"warnings":            "array",
			"confidence":          "number 0..1",
		},
		"domain":      input.Domain,
		"grade_band":  input.GradeBand,
		"target_mode": input.TargetMode,
		"message":     input.Message,
		"context":     input.Context,
		"evidence":    input.Evidence,
	}

	return mustJSON(payload)
}

func RenderSystem() string {
	return strings.TrimSpace(
		`你是一名专业的交互式科学可视化前端工程师。任务是根据 scene_spec 生成一个可在浏览器 iframe srcDoc 中独立运行的原生 HTML/CSS/JavaScript 教学演示页面。不展示隐藏推理，不要输出 markdown，不要输出解释文字；最终只输出符合约定的 JSON。

总体目标：
- 生成离线可运行、无需网络、无需外部依赖、适合 iframe sandbox 的单文件交互式演示。
- 优先保证 JSON 合法、code_bundle 完整、代码可执行和教学信息准确；不限制 index.html/code_bundle 字符数，不要为了压缩而省略必要实现或使用占位符。
- 页面应具备专业的科普/课堂演示质感：清晰结构、动态视觉、关键标签、参数反馈、状态提示和可理解的过程呈现。

必须严格输出这个 JSON schema：
{
  "scene_type": "biology_concept_flow",
  "render_mode": "html_iframe",
  "render_manifest": {
    "entry": "index.html",
    "framework": "vanilla",
    "sandbox": "iframe",
    "render_mode": "html_iframe",
    "mount_selector": "#snowy-preview-root",
    "dependencies": ["native-html", "canvas"],
    "allowed_apis": ["requestAnimationFrame", "setTimeout", "postMessage", "CanvasRenderingContext2D"],
    "blocked_apis": ["fetch", "XMLHttpRequest", "localStorage", "sessionStorage", "document.cookie", "WebSocket", "navigator.sendBeacon"],
    "initial_props": {"concept_count":4,"relation_count":3,"animation_speed":1}
  },
  "code_bundle": {"index.html":"<!doctype html>..."},
  "result_summary": "..."
}

硬性要求：
1. code_bundle 必须是 JSON object，至少包含 index.html；不要把 code_bundle 输出成字符串；不要输出 markdown、解释文字、省略号或“待实现”占位符。
2. index.html 必须是完整 HTML，包含 id="snowy-preview-root" 的根节点；所有 CSS/JS 内联；禁止外链脚本、CDN、动态依赖和任何网络访问。
3. 禁止在 index.html 的代码、注释、字符串、HTML 属性中出现这些危险片段：fetch(、XMLHttpRequest、localStorage、sessionStorage、indexedDB、document.cookie、WebSocket、navigator.sendBeacon、<script src=、import(。
4. JS 必须监听 window message：event.data.type === 'snowy:update-props' 时合并 props 并重绘；支持数值 prop view_dimension=3 或 2 切换视图；支持 animation_speed 调整动画倍率。
5. 渲染 ready 后必须调用 parent.postMessage({source:'snowy-preview',type:'preview',status:'ready'}, '*')；异常时发送 parent.postMessage({source:'snowy-preview',type:'preview',status:'error',message:String(error)}, '*')。
6. 视觉要求：暗色或高对比舞台、动态图例、参数 HUD、阶段说明面板、发光箭头或粒子流、平滑动画；避免静态黑白示意图或信息密度过低的画面。
7. physics_* 场景由宿主应用的本地物理引擎承载；不要为 physics_* 生成可执行前端代码。本生成链路主要服务 biology_* 等非物理可视化场景。
8. biology_* 场景可以使用 Canvas 2D 或 WebGL；必须包含粒子/流动路径、阶段切换、概念标签、过程箭头、解释面板、播放/暂停或自动动画。
9. 输出前自检：JSON 必须包含 code_bundle.index.html；biology_* 的 index.html 中必须能找到 snowy:update-props、snowy-preview ready、CanvasRenderingContext2D 或 getContext('2d')、particle/flow/stage/label 等可视化语义。
10. JSON 字符串中的换行和引号必须合法转义；代码包长度不设上限，如果代码较长，继续完整输出，不要截断 code_bundle。`,
	)
}

func RenderUser(input RenderInput) string {
	payload := mustJSON(input.SceneSpec)

	prompt := "scene_spec=" + payload + "\nrender_mode=" + strings.TrimSpace(
		input.Mode,
	) + "\n只输出 JSON，不要 markdown；不要省略 code_bundle，不要用占位符；代码包长度不设上限，必须完整输出。"
	if input.Biology {
		prompt += "\n\nbiology_* 成功标准：code_bundle.index.html 必须是完整单文件 HTML；必须监听 snowy:update-props；必须发送 snowy-preview ready；必须包含 CanvasRenderingContext2D 或 getContext('2d')；必须包含 particle/flow/stage/label 等可视化语义；代码包长度不设上限，代码较长也要完整输出。"
	}

	return prompt
}

func RegenerateClassifierSystem() string {
	return strings.TrimSpace(`你是教育对话改写分类器。用户已经看到一个交互式演示，正在追问。请判断追问意图。

只允许输出 JSON，不要 markdown，不要解释文字：
{"action":"recompute","target_vars":{"v0":30},"reason":"用户只改变初速度"}

action 只能取：
- recompute：仅修改演示变量值、参数、数值条件，例如“如果 v0 变成 30 呢”“把温度调到 80℃”。
- regenerate：希望换例子、换解法、换场景、加入新假设、改变模型结构，例如“换成天体公转的例子”“加入空气阻力”。
- new：与当前演示无关的新主题。

要求：
1. action 必须是 recompute、regenerate 或 new。
2. recompute 时尽量抽取 target_vars；无法可靠抽取也可返回空 object。
3. regenerate 时 reason 用一句话概括结构性变化。
4. new 时 target_vars 为空，reason 为空或一句简短说明。
5. 不展示隐藏推理。`)
}

func DefaultProfiles(now time.Time) []Profile {
	if now.IsZero() {
		now = time.Now()
	}

	return []Profile{
		{
			ID:                 VersionSearchDirect,
			Scene:              SceneSearchDirect,
			Version:            VersionSearchDirect,
			Mode:               "knowledge_answer_pe",
			Title:              "知识点直答 PE",
			System:             KnowledgeAnswerSystem(),
			UserPromptContract: "注入当前日期、用户问题、学科/年级筛选、解析到的意图与关键词；要求直接回答，不输出 JSON，涉及公式需说明符号含义、单位和适用条件。",
			SuccessChecklist:   []string{"结论明确且适合高中生", "公式、单位、适用条件清楚", "不伪造引用或内部来源声明", "包含易错点和下一步追问"},
			GenerationParams:   map[string]any{"temperature": 0.35, "max_tokens": maxTokens128K},
			UpdatedAt:          now,
		},
		{
			ID:                 VersionRenderGeneration,
			Scene:              SceneBiologyRender,
			Version:            VersionBiologyRender,
			Mode:               "render_generation_pe",
			Title:              "生物可视化演示 PE",
			System:             RenderSystem(),
			UserPromptContract: "传入 scene_spec 和 render_mode；强调只输出 JSON、不使用 markdown、不省略 code_bundle、不输出占位符，代码包长度不设上限。",
			SuccessChecklist: []string{
				"code_bundle.index.html 完整可运行",
				"遵循预览通信协议并发送 ready 状态",
				"Canvas 2D/WebGL 离线渲染，无外链依赖",
				"体现 particle/flow/stage/label 等动态可视化语义",
			},
			GenerationParams: map[string]any{"temperature": 0.15, "max_tokens": maxTokens128K},
			UpdatedAt:        now,
		},
		{
			ID:                 VersionPhysicsNative,
			Scene:              ScenePhysicsNativeAnalysis,
			Version:            VersionPhysicsNative,
			Mode:               "native_physics_engine",
			Title:              "物理原生引擎解析 PE",
			System:             "你是一名专业、严谨的高中物理建模辅导专家。物理题目由本地 Rapier 3D 引擎确定性完成仿真与渲染；大模型只负责题意解析、参数抽取、步骤讲解和 scene_spec 组织，不生成可执行前端代码。回答应突出物理规律、变量关系、单位、适用条件和可视化参数含义。",
			UserPromptContract: "输入题干和会话上下文；输出模型类型、条件、参数、推导步骤、讲解与 scene_spec；禁止生成前端代码。",
			SuccessChecklist: []string{
				"scene_spec 可驱动 force_3d / projectile / motion 等预览",
				"参数包含质量、力、速度、角度、时间、重力等可调项",
				"解释中明确 F=ma、运动分解或对应物理规律",
				"解析失败时向调用方返回明确失败原因，不伪造可视化结果",
			},
			GenerationParams: map[string]any{"runtime": "rapier3d", "llm_code_generation": false},
			UpdatedAt:        now,
		},
	}
}

func mustJSON(value any) string {
	body, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}

	return string(body)
}
