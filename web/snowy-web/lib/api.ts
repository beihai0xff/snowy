/**
 * Snowy API 客户端 — 封装所有后端接口调用。
 * 统一处理 token 注入、响应解包、错误映射。
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

// ── Auth ─────────────────────────────────────────────────

export interface GoogleLoginReq {
  id_token: string;
}

export interface LoginResp {
  access_token: string;
  refresh_token: string;
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

// ── Physics ──────────────────────────────────────────────

export interface PhysicsAnalyzeReq {
  question: string;
  context?: string;
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

export interface AxisSpec {
  label: string;
  unit: string;
}

export interface SeriesSpec {
  name: string;
  data: number[][];
}

export interface ChartSpec {
  chart_type: string;
  title: string;
  x_axis: AxisSpec;
  y_axis: AxisSpec;
  series: SeriesSpec[];
}

export interface PhysicsModel {
  model_type: string;
  conditions: Condition[];
  steps: DerivationStep[];
  result_summary: string;
  warnings?: string[];
  chart?: ChartSpec;
  parameters?: ParameterSchema[];
}

export interface PhysicsSimulateReq {
  model_type: string;
  parameters: Record<string, number>;
}

export interface ComputeResult {
  values: Record<string, number>;
  chart: ChartSpec;
  warnings?: string[];
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

// ── HTTP helpers ─────────────────────────────────────────

function getToken(): string | null {
  if (typeof window === 'undefined') return null;
  return localStorage.getItem('snowy_access_token');
}

async function request<T>(
  path: string,
  options: RequestInit = {},
): Promise<APIResponse<T>> {
  const token = getToken();
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string>),
  };
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers,
  });

  const json: APIResponse<T> = await res.json();
  if (json.code !== 'OK') {
    throw new Error(json.message || 'API Error');
  }
  return json;
}

// ── API functions ────────────────────────────────────────

export const api = {
  // Auth
  googleLogin: (data: GoogleLoginReq) =>
    request<LoginResp>('/auth/google/callback', { method: 'POST', body: JSON.stringify(data) }),

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

  // Physics
  physicsAnalyze: (data: PhysicsAnalyzeReq) =>
    request<PhysicsModel>('/modeling/physics/analyze', { method: 'POST', body: JSON.stringify(data) }),

  physicsSimulate: (data: PhysicsSimulateReq) =>
    request<ComputeResult>('/modeling/physics/simulate', { method: 'POST', body: JSON.stringify(data) }),

  // Biology
  biologyAnalyze: (data: BiologyAnalyzeReq) =>
    request<BiologyModel>('/modeling/biology/analyze', { method: 'POST', body: JSON.stringify(data) }),

  // Agent Chat
  agentChat: (data: ChatReq) =>
    request<ChatResponse>('/agent/chat', { method: 'POST', body: JSON.stringify(data) }),

  createSession: (mode: string) =>
    request<SessionResp>('/agent/sessions', { method: 'POST', body: JSON.stringify({ mode }) }),
};
