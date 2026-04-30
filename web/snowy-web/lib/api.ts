/**
 * Snowy API 客户端 — 封装所有后端接口调用。
 * 统一处理响应解包、错误映射。
 * 当前已禁用登录，无需 token 注入。
 */

const API_BASE = process.env.NEXT_PUBLIC_API_BASE || '/api/v1';

export interface APIResponse<T = unknown> {
  code: string;
  message: string;
  data?: T;
  request_id?: string;
}

export interface PageResponse<T = unknown> {
  total: number;
  page: number;
  page_size: number;
  items: T[];
}

// ── User ─────────────────────────────────────────────────

export interface User {
  id: string;
  google_id?: string;
  email?: string;
  phone?: string;
  nickname: string;
  role: string;
  avatar_url?: string;
  last_login_at: string;
  created_at: string;
  updated_at: string;
}

export interface HistoryItem {
  id: string;
  user_id: string;
  action_type: string;
  query: string;
  session_id?: string;
  created_at: string;
}

export interface Favorite {
  id: string;
  user_id: string;
  target_type: string;
  target_id: string;
  title: string;
  created_at: string;
}

export interface FavoriteReq {
  target_type: 'search' | 'physics' | 'biology';
  target_id: string;
  title: string;
}

// ── Recommendations ──────────────────────────────────────

export interface RecommendationItem {
  id: string;
  title: string;
  description: string;
  category: string;
  icon?: string;
}

export interface RecommendationsResp {
  hot_topics: RecommendationItem[];
  physics_models: RecommendationItem[];
  biology_topics: RecommendationItem[];
}

// ── Search ───────────────────────────────────────────────

export interface SearchFilters {
  subject?: string;
  grade?: string;
  chapter?: string;
  source?: string;
}

export interface SearchQueryReq {
  query: string;
  session_id?: string;
  filters?: SearchFilters;
}

export interface Citation {
  doc_id: string;
  source_type: string;
  snippet: string;
  score: number;
}

export interface RelatedQuestion {
  id: string;
  title: string;
}

export interface SearchResponse {
  answer: string;
  knowledge_tags: string[];
  citations: Citation[];
  related_questions: RelatedQuestion[];
  confidence: number;
}

// ── Physics / Render ─────────────────────────────────────

export interface PhysicsAnalyzeReq {
  question: string;
  context?: string;
  target_scene?: string;
}

export interface Condition {
  name: string;
  value: number;
  unit: string;
}

export interface DerivationStep {
  index: number;
  title: string;
  content: string;
}

export interface ParameterSchema {
  name: string;
  label: string;
  default: number;
  min: number;
  max: number;
  step: number;
  unit: string;
}

export interface SceneSpec {
  scene_type: string;
  title: string;
  summary?: string;
  render_mode?: 'html_iframe' | 'react_iframe';
  default_props?: Record<string, number>;
}

export interface RenderManifest {
  entry: string;
  framework: string;
  sandbox: string;
  render_mode: 'html_iframe' | 'react_iframe';
  mount_selector: string;
  dependencies?: string[];
  allowed_apis?: string[];
  blocked_apis?: string[];
  initial_props?: Record<string, number>;
}

export interface RenderArtifact {
  scene_type: string;
  render_mode: 'html_iframe' | 'react_iframe';
  render_manifest: RenderManifest;
  code_bundle: Record<string, string>;
  result_summary: string;
  warnings?: string[];
}

export interface PhysicsModel {
  model_type: string;
  conditions: Condition[];
  steps: DerivationStep[];
  result_summary: string;
  warnings?: string[];
  parameters?: ParameterSchema[];
  scene_spec?: SceneSpec;
}

export interface RenderGenerateReq {
  scene_spec: SceneSpec;
  context?: string;
  render_mode?: 'html_iframe' | 'react_iframe';
}

// ── Biology ──────────────────────────────────────────────

export interface BiologyAnalyzeReq {
  question: string;
  context?: string;
}

export interface Concept {
  name: string;
  type: string;
}

export interface Relation {
  source: string;
  target: string;
  type: string;
}

export interface ProcessStep {
  index: number;
  title: string;
  content: string;
}

export interface ExperimentVariables {
  independent: string[];
  dependent: string[];
  controlled: string[];
}

export interface DiagramNode {
  id: string;
  label: string;
  type: string;
}

export interface DiagramEdge {
  source: string;
  target: string;
  label: string;
}

export interface DiagramSpec {
  diagram_type: string;
  title: string;
  nodes: DiagramNode[];
  edges: DiagramEdge[];
}

export interface BiologyModel {
  topic: string;
  concepts: Concept[];
  relations: Relation[];
  process_steps: ProcessStep[];
  experiment_variables?: ExperimentVariables;
  diagram?: DiagramSpec;
  scene_spec?: SceneSpec;
  result_summary: string;
}

// ── Agent / Chat ─────────────────────────────────────────

export interface ChatReq {
  session_id?: string;
  message: string;
  mode?: 'search' | 'physics' | 'biology' | 'auto';
  filters?: { subject?: string; grade?: string };
}

export interface ChatResponse {
  mode: string;
  answer: string;
  citations?: Citation[];
  tool_calls?: { tool: string; status: string }[];
  structured_payload?: unknown;
  confidence: number;
  next_actions?: string[];
}

export interface SessionResp {
  id: string;
  mode: string;
  status: string;
  created_at: string;
}

export interface SSEMessage {
  event: string;
  data: unknown;
}

export interface AgentChatStreamOptions {
  signal?: AbortSignal;
  onOpen?: () => void;
  onError?: (error: Error) => void;
  onDone?: () => void;
}

export interface AgentPhysicsPayload {
  analysis?: PhysicsModel;
  render_artifact?: RenderArtifact;
}

export interface AgentBiologyPayload {
  analysis?: BiologyModel;
  render_artifact?: RenderArtifact;
}

// ── HTTP helpers ─────────────────────────────────────────

async function request<T>(
  path: string,
  options: RequestInit = {},
): Promise<APIResponse<T>> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string>),
  };

  let res: Response;
  try {
    res = await fetch(`${API_BASE}${path}`, {
      ...options,
      headers,
    });
  } catch (error) {
    throw new Error(error instanceof Error ? `网络请求失败：${error.message}` : '网络请求失败');
  }

  const text = await res.text();
  let json: APIResponse<T> | null = null;
  try {
    json = text ? JSON.parse(text) as APIResponse<T> : null;
  } catch {
    json = null;
  }

  if (!res.ok) {
    const message = json?.message || text || `HTTP ${res.status}`;
    throw new Error(`请求失败：${message}`);
  }

  if (!json) {
    throw new Error(text ? `响应不是合法 JSON：${text.slice(0, 160)}` : '响应为空');
  }

  if (json.code !== 'OK') {
    throw new Error(json.message || 'API Error');
  }
  return json;
}

// ── API functions ────────────────────────────────────────

export const api = {
  // User
  getProfile: () => request<User>('/user/profile'),

  getHistory: () => request<PageResponse<HistoryItem>>('/history'),

  listFavorites: () => request<PageResponse<Favorite>>('/favorites'),

  addFavorite: (data: FavoriteReq) =>
    request<Favorite>('/favorites', { method: 'POST', body: JSON.stringify(data) }),

  // Recommendations
  getRecommendations: () => request<RecommendationsResp>('/recommendations'),

  // Search
  searchQuery: (data: SearchQueryReq) =>
    request<SearchResponse>('/search/query', { method: 'POST', body: JSON.stringify(data) }),

  // Physics / Render
  physicsAnalyze: (data: PhysicsAnalyzeReq) =>
    request<PhysicsModel>('/modeling/physics/analyze', { method: 'POST', body: JSON.stringify(data) }),

  renderGenerate: (data: RenderGenerateReq) =>
    request<RenderArtifact>('/modeling/render/generate', { method: 'POST', body: JSON.stringify(data) }),

  // Biology
  biologyAnalyze: (data: BiologyAnalyzeReq) =>
    request<BiologyModel>('/modeling/biology/analyze', { method: 'POST', body: JSON.stringify(data) }),

  // Agent Chat
  agentChat: (data: ChatReq) =>
    request<ChatResponse>('/agent/chat', { method: 'POST', body: JSON.stringify(data) }),

  createSession: (mode: string) =>
    request<SessionResp>('/agent/sessions', { method: 'POST', body: JSON.stringify({ mode }) }),
};

export async function agentChatStream(
  data: ChatReq,
  onEvent: (event: SSEMessage) => void,
  options: AgentChatStreamOptions = {},
): Promise<void> {
  try {
    const res = await fetch(`${API_BASE}/agent/chat`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Accept: 'text/event-stream',
      },
      body: JSON.stringify(data),
      signal: options.signal,
    });

    if (!res.ok || !res.body) {
      const text = await res.text().catch(() => '');
      throw new Error(text || `HTTP ${res.status}`);
    }

    options.onOpen?.();

    const reader = res.body.getReader();
    const decoder = new TextDecoder();
    let buffer = '';

    const flush = (chunk: string) => {
      const lines = chunk.replace(/\r\n/g, '\n').split('\n');
      let eventName = 'message';
      const dataLines: string[] = [];

      for (const line of lines) {
        if (!line || line.startsWith(':')) continue;
        if (line.startsWith('event:')) {
          eventName = line.slice(6).trim();
        } else if (line.startsWith('data:')) {
          dataLines.push(line.slice(5).trimStart());
        }
      }

      const raw = dataLines.join('\n');
      let payload: unknown = raw;
      try {
        payload = raw ? JSON.parse(raw) : null;
      } catch {
        payload = raw;
      }

      onEvent({ event: eventName, data: payload });
    };

    while (true) {
      const { done, value } = await reader.read();
      buffer += decoder.decode(value || new Uint8Array(), { stream: !done });

      let normalizedBuffer = buffer.replace(/\r\n/g, '\n');
      let idx = normalizedBuffer.indexOf('\n\n');
      while (idx >= 0) {
        const chunk = normalizedBuffer.slice(0, idx).trim();
        normalizedBuffer = normalizedBuffer.slice(idx + 2);
        if (chunk) flush(chunk);
        idx = normalizedBuffer.indexOf('\n\n');
      }
      buffer = normalizedBuffer;

      if (done) {
        const rest = buffer.trim();
        if (rest) flush(rest);
        options.onDone?.();
        break;
      }
    }
  } catch (error) {
    const normalized = error instanceof Error ? error : new Error('SSE stream failed');
    options.onError?.(normalized);
    throw normalized;
  }
}
