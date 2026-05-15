'use client';

import React, { Suspense, useCallback, useEffect, useMemo, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { Alert, Button, Collapse, Empty, Select, Space, Tag, Typography, message } from 'antd';
import {
  BookOutlined,
  BranchesOutlined,
  BulbOutlined,
  CheckCircleFilled,
  ExclamationCircleFilled,
  ExperimentOutlined,
  FunctionOutlined,
  ReloadOutlined,
  RightOutlined,
  SearchOutlined,
  StarOutlined,
} from '@ant-design/icons';
import { api, type Citation, type FavoriteReq, type SearchResponse } from '@/lib/api';
import MarkdownText from '@/components/common/MarkdownText';
import ReactionBar from '@/components/common/ReactionBar';

const { Title, Text } = Typography;

const askExamples = [
  '为什么平抛运动水平方向是匀速？',
  '光合作用中光照强度如何影响有机物积累？',
  '牛顿第二定律的适用条件是什么？',
];

const sourceLabel: Record<string, string> = {
  textbook: '课本',
  exam:     '考纲',
  exercise: '题库',
  lecture:  '讲义',
  paper:    '文献',
};

function confidenceLevel(value: number): { tone: 'ok' | 'warn' | 'err'; label: string } {
  if (value >= 0.8) return { tone: 'ok',   label: '高可信' };
  if (value >= 0.5) return { tone: 'warn', label: '需核验' };
  return { tone: 'err', label: '低可信' };
}

function inferModelingSubject(text: string, selectedSubject?: string): 'physics' | 'biology' {
  if (selectedSubject === 'biology' || selectedSubject === 'physics') return selectedSubject;
  if (/光合|细胞|酶|遗传|突触|神经|生态|呼吸|膜|DNA|RNA|蛋白质/.test(text)) return 'biology';
  return 'physics';
}

function CitationsList({ citations, onJump }: { citations: Citation[]; onJump?: (index: number) => void }) {
  if (!citations.length) {
    return <Text type="secondary" style={{ fontSize: 13 }}>暂无引用</Text>;
  }
  return (
    <div>
      {citations.map((citation, index) => (
        <div
          key={`${citation.doc_id}-${index}`}
          className="snowy-citation-row"
          onClick={() => onJump?.(index)}
          role="button"
          tabIndex={0}
          onKeyDown={(event) => { if (event.key === 'Enter') onJump?.(index); }}
        >
          <span className="snowy-citation-num">{index + 1}</span>
          <div style={{ minWidth: 0 }}>
            <div style={{ fontSize: 13, fontWeight: 500, marginBottom: 4, color: 'var(--color-text)' }}>
              <Tag bordered={false} color="default" style={{ marginRight: 6, background: 'var(--color-bg-subtle)', color: 'var(--color-text-muted)' }}>
                {sourceLabel[citation.source_type] || citation.source_type}
              </Tag>
              {citation.doc_id}
            </div>
            <Text type="secondary" style={{ fontSize: 12, display: '-webkit-box', WebkitLineClamp: 2, WebkitBoxOrient: 'vertical', overflow: 'hidden' }}>
              {citation.snippet}
            </Text>
          </div>
          <span className="snowy-citation-score">{Math.round((citation.score || 0) * 100)}%</span>
        </div>
      ))}
    </div>
  );
}

function AskPageInner() {
  const searchParams = useSearchParams();
  const router = useRouter();
  const [query, setQuery] = useState(searchParams.get('q') || '');
  const [result, setResult] = useState<SearchResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [errorText, setErrorText] = useState<string | null>(null);
  const [subject, setSubject] = useState<string | undefined>();
  const [grade, setGrade] = useState<string | undefined>();

  const runSearch = useCallback(async (text: string) => {
    const trimmed = text.trim();
    if (!trimmed) return;
    setLoading(true);
    setErrorText(null);
    try {
      const res = await api.searchQuery({ query: trimmed, filters: { subject, grade } });
      setResult(res.data ?? null);
    } catch (error) {
      const msg = error instanceof Error ? error.message : '检索失败，请稍后重试';
      setErrorText(msg);
      setResult(null);
      message.error(msg);
    } finally {
      setLoading(false);
    }
  }, [subject, grade]);

  useEffect(() => {
    const q = searchParams.get('q');
    if (q) {
      setQuery(q);
      void runSearch(q);
    }
  }, [searchParams, runSearch]);

  const confidence = result ? confidenceLevel(result.confidence) : null;
  const evidenceScore = useMemo(() => {
    if (!result?.citations?.length) return 0;
    const avg = result.citations.reduce((sum, c) => sum + (c.score || 0), 0) / result.citations.length;
    return Math.round(avg * 100);
  }, [result]);

  const inferredSubject = useMemo(() => inferModelingSubject(query, subject), [query, subject]);

  const onFavorite = async () => {
    if (!result) return;
    const req: FavoriteReq = {
      target_type: result.answer_id ? 'answer' : 'search',
      target_id:   result.answer_id || query,
      title:       query,
      metadata_json: {
        answer_id: result.answer_id,
        query,
        answer_summary: result.answer.slice(0, 600),
        confidence: result.confidence,
        knowledge_tags: result.knowledge_tags || [],
        citations: (result.citations || []).slice(0, 5),
      },
    };
    try {
      await api.addFavorite(req);
      message.success('已加入收藏');
    } catch (error) {
      message.error(error instanceof Error ? error.message : '收藏失败');
    }
  };

  const scrollToCitations = (index: number) => {
    const el = document.getElementById(`snowy-citation-${index}`);
    if (el && 'scrollIntoView' in el) el.scrollIntoView({ behavior: 'smooth', block: 'center' });
  };

  return (
    <div className="snowy-page">
      {/* 顶部搜索条（吸顶） */}
      <div className="snowy-ask-bar">
        <form
          className="snowy-search"
          onSubmit={(event) => { event.preventDefault(); void runSearch(query); }}
          role="search"
        >
          <span className="snowy-search__icon"><SearchOutlined /></span>
          <input
            className="snowy-search__input"
            type="search"
            value={query}
            placeholder="再问一个问题，或修改当前问题…"
            onChange={(event) => setQuery(event.target.value)}
            aria-label="输入你的问题"
          />
          <button type="submit" className="snowy-search__submit">
            {loading ? <span className="snowy-spinner" style={{ borderTopColor: '#fff', width: 14, height: 14 }} /> : <RightOutlined />}
            <span>提问</span>
          </button>
        </form>
        <Space size={8} style={{ marginTop: 8 }} wrap>
          <Select
            placeholder="学科"
            allowClear
            size="small"
            style={{ width: 110 }}
            value={subject}
            onChange={setSubject}
            options={[
              { value: 'physics',   label: '物理' },
              { value: 'biology',   label: '生物' },
              { value: 'chemistry', label: '化学' },
              { value: 'math',      label: '数学' },
            ]}
          />
          <Select
            placeholder="年级"
            allowClear
            size="small"
            style={{ width: 110 }}
            value={grade}
            onChange={setGrade}
            options={[
              { value: 'high_school_1', label: '高一' },
              { value: 'high_school_2', label: '高二' },
              { value: 'high_school_3', label: '高三' },
            ]}
          />
          <Text type="secondary" style={{ fontSize: 13 }}>每个答案都会先查证据、再生成结论</Text>
        </Space>
      </div>

      {/* 错误诊断 */}
      {errorText && !loading && (
        <Alert
          type="error"
          showIcon
          message="检索失败"
          description={errorText}
          action={<Button size="small" icon={<ReloadOutlined />} onClick={() => void runSearch(query)}>重试</Button>}
          style={{ marginBottom: 16, borderRadius: 12 }}
        />
      )}

      {/* 加载态 */}
      {loading && (
        <div className="snowy-loading-card">
          <span className="snowy-spinner" />
          <span>正在检索证据并组织答案…</span>
        </div>
      )}

      {/* 空状态 */}
      {!loading && !result && !errorText && (
        <div className="snowy-loading-card" style={{ padding: 56 }}>
          <Empty
            image={Empty.PRESENTED_IMAGE_SIMPLE}
            description={(
              <Space direction="vertical" size={12} align="center">
                <Text>输入一个问题，Snowy 会先查证据，再给答案。</Text>
                <Space wrap size={8}>
                  {askExamples.map((q) => (
                    <button key={q} className="snowy-chip" type="button" onClick={() => { setQuery(q); void runSearch(q); }}>
                      试试：{q}
                    </button>
                  ))}
                </Space>
              </Space>
            )}
          />
        </div>
      )}

      {/* 答案 + 证据双列 */}
      {!loading && result && (
        <div className="snowy-ask-layout">
          {/* 左列：答案 */}
          <article className="snowy-answer">
            <div className={`snowy-confidence-bar ${confidence?.tone === 'warn' ? 'snowy-confidence-bar--warn' : confidence?.tone === 'err' ? 'snowy-confidence-bar--err' : ''}`}>
              {confidence?.tone === 'ok' && <CheckCircleFilled style={{ color: 'var(--color-success)', fontSize: 18 }} />}
              {confidence?.tone !== 'ok' && <ExclamationCircleFilled style={{ color: confidence?.tone === 'err' ? 'var(--color-danger)' : 'var(--color-warning)', fontSize: 18 }} />}
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ fontWeight: 600, fontSize: 14 }}>
                  {confidence?.label}（{Math.round(result.confidence * 100)}%）
                  {result.citations?.length ? `，基于 ${result.citations.length} 条引用` : ''}
                </div>
                {result.confidence < 0.5 && (
                  <Text type="secondary" style={{ fontSize: 12 }}>建议补充题干、限定学科年级，或进入推演舱让 AI 重新分析。</Text>
                )}
              </div>
              <Space size={8} wrap>
                <ReactionBar targetType="answer" targetID={result.answer_id || query} size="small" showUsers={false} />
                <Button size="small" icon={<StarOutlined />} onClick={onFavorite}>收藏</Button>
              </Space>
            </div>

            <Title level={2} style={{ marginTop: 8, marginBottom: 16, fontSize: 24, fontWeight: 700, letterSpacing: '-0.01em' }}>
              {query}
            </Title>

            <MarkdownText content={result.answer} />

            {result.knowledge_tags && result.knowledge_tags.length > 0 && (
              <Space wrap style={{ margin: '16px 0 24px' }} size={6}>
                {result.knowledge_tags.map((tag) => (
                  <Tag key={tag} color="default" bordered={false} style={{ background: 'var(--color-primary-soft)', color: 'var(--color-primary)' }}>
                    {tag}
                  </Tag>
                ))}
              </Space>
            )}

            {/* 行动栏：跳推演 / 跳生物图谱 */}
            <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', margin: '24px 0' }}>
              <Button
                type="primary"
                icon={<ExperimentOutlined />}
                onClick={() => router.push(`/modeling?type=${inferredSubject}&q=${encodeURIComponent(query)}`)}
              >
                用这个问题做推演
              </Button>
              <Button
                icon={<BranchesOutlined />}
                onClick={() => router.push(`/modeling?type=biology&q=${encodeURIComponent(query)}`)}
              >
                看生物图谱
              </Button>
              <Button
                icon={<BookOutlined />}
                onClick={() => router.push('/learning')}
              >
                查看我的学习
              </Button>
            </div>

            {/* 手风琴：公式 / 易错点 / 题型 */}
            <Collapse
              accordion
              ghost
              style={{ marginTop: 24 }}
              items={[
                ...(result.formula_cards && result.formula_cards.length > 0 ? [{
                  key: 'formula',
                  label: <Space><FunctionOutlined /> 公式与规律卡（{result.formula_cards.length}）</Space>,
                  children: (
                    <Space direction="vertical" size={16} style={{ width: '100%' }}>
                      {result.formula_cards.map((card, index) => (
                        <div key={`${card.name}-${index}`} style={{ padding: 16, background: 'var(--color-bg-subtle)', borderRadius: 8, borderLeft: '3px solid var(--color-primary)' }}>
                          <div style={{ fontWeight: 600, marginBottom: 4 }}>{card.name}</div>
                          <Text code style={{ fontSize: 15 }}>{card.expression}</Text>
                          <div style={{ marginTop: 8, fontSize: 13, lineHeight: 1.7, color: 'var(--color-text-muted)' }}>
                            {card.variables && card.variables.length > 0 && <div>变量：{card.variables.join('；')}</div>}
                            {card.applies_to && card.applies_to.length > 0 && <div>适用：{card.applies_to.join('；')}</div>}
                            {card.limits && card.limits.length > 0 && <div style={{ color: 'var(--color-warning)' }}>边界：{card.limits.join('；')}</div>}
                          </div>
                        </div>
                      ))}
                    </Space>
                  ),
                }] : []),
                ...(result.misconceptions && result.misconceptions.length > 0 ? [{
                  key: 'misconceptions',
                  label: <Space><BulbOutlined /> 易错点提醒（{result.misconceptions.length}）</Space>,
                  children: (
                    <Space direction="vertical" size={12} style={{ width: '100%' }}>
                      {result.misconceptions.map((tip, index) => (
                        <div key={`${tip.type}-${index}`} style={{ padding: 12, background: 'var(--color-warning-soft)', borderLeft: '3px solid var(--color-warning)', borderRadius: 8 }}>
                          <Space size={6} style={{ marginBottom: 4 }}><Tag color="orange" bordered={false}>{tip.type}</Tag></Space>
                          <div style={{ fontSize: 14, lineHeight: 1.7 }}>{tip.description}</div>
                          {tip.correction && <div style={{ fontSize: 13, color: 'var(--color-text-muted)', marginTop: 4 }}>纠偏：{tip.correction}</div>}
                        </div>
                      ))}
                    </Space>
                  ),
                }] : []),
                ...(result.exam_mappings && result.exam_mappings.length > 0 ? [{
                  key: 'exam',
                  label: <Space>📝 高考题型映射（{result.exam_mappings.length}）</Space>,
                  children: (
                    <Space direction="vertical" size={12} style={{ width: '100%' }}>
                      {result.exam_mappings.map((exam, index) => (
                        <div key={`${exam.question_type}-${index}`} style={{ padding: 12, borderRadius: 8, border: '1px solid var(--color-divider)' }}>
                          <Space size={6} style={{ marginBottom: 4 }}>
                            <Tag color="gold" bordered={false}>{exam.question_type}</Tag>
                            {exam.knowledge && exam.knowledge.map((k) => <Tag key={k} bordered={false}>{k}</Tag>)}
                          </Space>
                          <div style={{ fontSize: 14 }}>{exam.focus}</div>
                          {exam.practice_hint && <div style={{ fontSize: 13, color: 'var(--color-text-muted)', marginTop: 4 }}>{exam.practice_hint}</div>}
                        </div>
                      ))}
                    </Space>
                  ),
                }] : []),
                ...(result.next_actions && result.next_actions.length > 0 ? [{
                  key: 'next',
                  label: <Space>🎯 下一步建议（{result.next_actions.length}）</Space>,
                  children: (
                    <Space direction="vertical" size={8} style={{ width: '100%' }}>
                      {result.next_actions.map((action, index) => (
                        <div key={`${action.type}-${index}`} style={{ padding: 12, background: 'var(--color-bg-subtle)', borderRadius: 8 }}>
                          <div style={{ fontWeight: 500, marginBottom: 2 }}>{action.label}</div>
                          {action.description && <div style={{ fontSize: 13, color: 'var(--color-text-muted)' }}>{action.description}</div>}
                          {action.tags && action.tags.length > 0 && <Space wrap size={4} style={{ marginTop: 6 }}>{action.tags.map((t) => <Tag key={t} bordered={false}>{t}</Tag>)}</Space>}
                        </div>
                      ))}
                    </Space>
                  ),
                }] : []),
              ]}
            />
          </article>

          {/* 右列：证据 + 相关问题 */}
          <aside className="snowy-ask-aside">
            <Space direction="vertical" size={20} style={{ width: '100%' }}>
              <section>
                <div className="snowy-section-title">
                  <h2>证据 · {result.citations?.length || 0} 条</h2>
                  <Text type="secondary" style={{ fontSize: 13 }}>平均 {evidenceScore}%</Text>
                </div>
                <div id="snowy-citations-anchor">
                  <CitationsList citations={result.citations || []} onJump={scrollToCitations} />
                </div>
              </section>

              {result.related_questions && result.related_questions.length > 0 && (
                <section>
                  <div className="snowy-section-title">
                    <h2>相关问题</h2>
                  </div>
                  <div style={{ background: 'var(--color-bg-card)', border: '1px solid var(--color-border)', borderRadius: 12 }}>
                    {result.related_questions.map((rq) => (
                      <div
                        key={rq.id || rq.title}
                        className="snowy-related-item"
                        onClick={() => { setQuery(rq.title); void runSearch(rq.title); }}
                        role="button"
                        tabIndex={0}
                        onKeyDown={(event) => { if (event.key === 'Enter') { setQuery(rq.title); void runSearch(rq.title); } }}
                      >
                        {rq.title}
                      </div>
                    ))}
                  </div>
                </section>
              )}
            </Space>
          </aside>
        </div>
      )}
    </div>
  );
}

export default function AskPage() {
  return (
    <Suspense fallback={<div className="snowy-loading-card"><span className="snowy-spinner" /><span>加载中…</span></div>}>
      <AskPageInner />
    </Suspense>
  );
}
