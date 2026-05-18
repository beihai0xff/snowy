'use client';

import React, { Suspense, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import {
  Alert,
  Button,
  Collapse,
  Empty,
  Input,
  List,
  Segmented,
  Space,
  Tag,
  Typography,
  message,
} from 'antd';
import {
  BookOutlined,
  BranchesOutlined,
  CheckCircleFilled,
  CloseCircleFilled,
  ExperimentOutlined,
  PlayCircleOutlined,
  ReloadOutlined,
  RightOutlined,
  StarOutlined,
  StopOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons';
import { api, type EvidenceRef, type GenerativeModelPackage, type VariableSpec } from '@/lib/api';
import GenerativePhysics3DCanvas from '@/components/generative/GenerativePhysics3DCanvas';
import GenerativeBiologyGraph from '@/components/generative/GenerativeBiologyGraph';
import GenerativeChemistryCanvas from '@/components/chemistry/GenerativeChemistryCanvas';
import SkeletonPreview from '@/components/common/SkeletonPreview';
import InteractionPlanPanel from '@/components/generative/InteractionPlanPanel';
import ValidationReportPanel from '@/components/generative/ValidationReportPanel';
import ReactionBar from '@/components/common/ReactionBar';
import CurriculumBadge from '@/components/learning/CurriculumBadge';
import { normalizedSubject, type SubjectKey } from '@/lib/curriculum';

const { Title, Paragraph, Text } = Typography;
const { TextArea } = Input;

type ModelingSubject = SubjectKey;
type Stage = 'idle' | 'grounding' | 'reasoning' | 'validating' | 'rendering' | 'done' | 'error';

const stageText: Record<Stage, string> = {
  idle:        '等待输入',
  grounding:   '检索证据',
  reasoning:   'AI 推理',
  validating:  '校验',
  rendering:   '渲染',
  done:        '完成',
  error:       '失败',
};

const stageOrder: Stage[] = ['grounding', 'reasoning', 'validating', 'rendering', 'done'];

const subjectExamples: Record<ModelingSubject, string[]> = {
  physics: [
    '平抛运动如何影响水平位移？初速度 20m/s，高度 20m',
    '牛顿第二定律在斜面题中怎么用？',
    '弹簧振子简谐运动中周期和质量、劲度系数有什么关系？',
  ],
  biology: [
    '光照强度如何影响有机物积累？',
    '遗传分离定律如何推导子代表现型比例？',
    '酶活性受温度影响的实验变量如何设计？',
  ],
  chemistry: [
    '2NaOH + H2SO4 如何配平并理解中和反应？',
    '电解水时为什么氢气和氧气体积比是 2:1？',
    '铁与硫酸铜反应中电子如何转移？',
  ],
};

function normalizeSubject(value: string | null): ModelingSubject {
  return normalizedSubject(value);
}

function subjectPlaceholder(subject: ModelingSubject): string {
  if (subject === 'biology') return '输入生物问题，例如：光合作用平台期怎么形成？';
  if (subject === 'chemistry') return '输入化学反应，例如：2NaOH + H2SO4 如何配平？';
  return '输入物理题目，例如：平抛运动怎样命中目标区？';
}

function subjectCanvasTitle(subject: ModelingSubject): string {
  if (subject === 'biology') return '生物图谱画布';
  if (subject === 'chemistry') return '化学反应画布';
  return '物理推演画布';
}

function initialValues(pkg: GenerativeModelPackage | null): Record<string, number> {
  const vars: VariableSpec[] = pkg?.simulation_logic?.variables || pkg?.generative_model.variables || [];
  return vars.reduce<Record<string, number>>((acc, variable) => {
    acc[variable.name] = variable.default;
    return acc;
  }, {});
}

function stageStatus(current: Stage, target: Stage): 'pending' | 'active' | 'done' {
  if (current === 'error') return target === stageOrder[stageOrder.length - 1] ? 'pending' : 'pending';
  if (current === 'done') return 'done';
  const idxCurrent = stageOrder.indexOf(current);
  const idxTarget = stageOrder.indexOf(target);
  if (idxTarget < idxCurrent) return 'done';
  if (idxTarget === idxCurrent) return 'active';
  return 'pending';
}

function ModelingPageInner() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [subject, setSubject] = useState<ModelingSubject>(normalizeSubject(searchParams.get('type') || searchParams.get('subject')));
  const [question, setQuestion] = useState(searchParams.get('q') || '');
  const [context, setContext] = useState('');
  const [pkg, setPkg] = useState<GenerativeModelPackage | null>(null);
  const [values, setValues] = useState<Record<string, number>>({});
  const [loading, setLoading] = useState(false);
  const [stage, setStage] = useState<Stage>('idle');
  const [errorText, setErrorText] = useState<string | null>(null);
  const [evidenceOpen, setEvidenceOpen] = useState(false);
  const loadedPackageRef = useRef<string | null>(null);
  const abortRef = useRef<AbortController | null>(null);

  const examples = subjectExamples[subject];
  const confidence = pkg?.validation_report?.confidence ?? pkg?.confidence ?? 0;

  const evidenceTags = useMemo(() => {
    const tags = new Set<string>();
    pkg?.evidence_refs?.forEach((ref) => ref.knowledge_tags?.forEach((tag) => tags.add(tag)));
    pkg?.learning_model?.knowledge_tags?.forEach((tag) => tags.add(tag));
    return Array.from(tags).slice(0, 8);
  }, [pkg]);

  const validationChecks = useMemo(() => {
    const report = pkg?.validation_report;
    if (!report) return [] as ReadonlyArray<readonly [string, boolean]>;
    return [
      ['Schema',   report.schema_valid] as const,
      ['Evidence', report.evidence_valid] as const,
      ['Domain',   report.domain_valid] as const,
      ['Safety',   report.safety_valid] as const,
    ];
  }, [pkg]);

  const handleCompile = useCallback(async (nextQuestion?: string, nextSubject?: ModelingSubject) => {
    const text = (nextQuestion || question).trim();
    const selectedSubject = nextSubject || subject;
    if (!text) return;
    abortRef.current?.abort();
    const controller = new AbortController();
    abortRef.current = controller;
    setLoading(true);
    setErrorText(null);
    setPkg(null);
    setValues({});
    setStage('grounding');
    try {
      let compileGrounding: { citations: EvidenceRef[]; knowledge_tags: string[] } = { citations: [], knowledge_tags: [] };
      try {
        const searchRes = await api.searchQuery({ query: text, filters: { subject: selectedSubject, grade: 'high_school' } }, { signal: controller.signal });
        compileGrounding = {
          citations: (searchRes.data?.citations || []).map((citation) => ({
            doc_id:         citation.doc_id,
            source_type:    citation.source_type,
            snippet:        citation.snippet,
            confidence:     citation.score,
            knowledge_tags: searchRes.data?.knowledge_tags || [],
          })),
          knowledge_tags: searchRes.data?.knowledge_tags || [],
        };
      } catch (err) {
        if ((err as { name?: string })?.name === 'AbortError') throw err;
        compileGrounding = { citations: [], knowledge_tags: [] };
      }
      if (controller.signal.aborted) return;
      setStage('reasoning');
      const res = await api.modelingCompile({
        message: text,
        domain: selectedSubject,
        grade_band: 'high_school',
        target_mode: 'interactive_model',
        context: {
          source_page: 'modeling-v6',
          user_notes: context || undefined,
          citations: compileGrounding.citations,
          knowledge_tags: compileGrounding.knowledge_tags,
        },
      }, { signal: controller.signal });
      if (controller.signal.aborted) return;
      setStage('validating');
      const data = res.data ?? null;
      setPkg(data);
      setValues(initialValues(data));
      setStage('rendering');
      if (data?.package_id) {
        const params = new URLSearchParams(searchParams.toString());
        params.set('package_id', data.package_id);
        router.replace(`/modeling?${params.toString()}`);
      }
      window.setTimeout(() => {
        if (!controller.signal.aborted) setStage('done');
      }, 150);
    } catch (error) {
      if ((error as { name?: string })?.name === 'AbortError') {
        setStage('idle');
        return;
      }
      const msg = error instanceof Error ? error.message : '推演失败';
      setErrorText(msg);
      setStage('error');
      message.error(msg);
    } finally {
      if (abortRef.current === controller) abortRef.current = null;
      setLoading(false);
    }
  }, [context, question, subject, router, searchParams]);

  const handleCancel = useCallback(() => {
    abortRef.current?.abort();
  }, []);

  useEffect(() => () => abortRef.current?.abort(), []);

  useEffect(() => {
    const packageID = searchParams.get('package_id');
    if (packageID) {
      if (loadedPackageRef.current === packageID) return;
      loadedPackageRef.current = packageID;
      setLoading(true);
      setErrorText(null);
      void api.getModelingPackage(packageID)
        .then((res) => {
          const data = res.data ?? null;
          setPkg(data);
          setValues(initialValues(data));
          if (data?.question) setQuestion(data.question);
          if (data?.domain === 'biology' || data?.domain === 'physics' || data?.domain === 'chemistry') setSubject(data.domain);
          setStage(data ? 'done' : 'idle');
        })
        .catch((error) => {
          const msg = error instanceof Error ? error.message : '模型包加载失败';
          setErrorText(msg);
          setStage('error');
          message.error(msg);
        })
        .finally(() => setLoading(false));
      return;
    }

    loadedPackageRef.current = null;
    const q = searchParams.get('q');
    const nextSubject = normalizeSubject(searchParams.get('type') || searchParams.get('subject'));
    setSubject(nextSubject);
    if (q) {
      setQuestion(q);
      void handleCompile(q, nextSubject);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchParams]);

  const handleFavorite = async () => {
    try {
      await api.addFavorite({
        target_type: pkg?.package_id ? 'model_package' : subject,
        target_id:   pkg?.package_id || question,
        title:       pkg?.learning_model.learning_goal || question || 'Snowy 推演任务',
        metadata_json: pkg ? {
          package_id:     pkg.package_id,
          domain:         pkg.domain,
          question:       pkg.question,
          learning_goal:  pkg.learning_model.learning_goal,
          knowledge_tags: pkg.learning_model.knowledge_tags || [],
          confidence:     pkg.confidence,
          status:         pkg.status,
          model_name:     pkg.model_name,
        } : { subject, question },
      });
      message.success('已加入收藏');
    } catch (error) {
      message.error(error instanceof Error ? error.message : '收藏失败');
    }
  };

  const handleRegenerate = async () => {
    if (!question.trim()) return;
    await handleCompile(question, subject);
  };

  return (
    <div className="snowy-page">
      <div className="snowy-page-heading">
        <h1>推演 · 看公式动起来 / 用图理流程</h1>
        <p>把问题变成一个能调参数、看动画、对照证据的模型。</p>
      </div>

      {/* 工具栏 */}
      <div className="snowy-modeling-toolbar">
        <Segmented
          value={subject}
          onChange={(value) => {
            const next = value as ModelingSubject;
            setSubject(next);
            setPkg(null);
            setValues({});
            setStage('idle');
          }}
          options={[
            { label: <Space size={4}><ExperimentOutlined />物理</Space>, value: 'physics' },
            { label: <Space size={4}><BranchesOutlined />生物</Space>,   value: 'biology' },
            { label: <Space size={4}><ExperimentOutlined />化学</Space>, value: 'chemistry' },
          ]}
        />
        <Input
          size="large"
          placeholder={subjectPlaceholder(subject)}
          value={question}
          onChange={(event) => setQuestion(event.target.value)}
          onPressEnter={() => void handleCompile()}
          allowClear
        />
        <Button icon={<StarOutlined />} disabled={!question.trim() && !pkg} onClick={handleFavorite}>收藏</Button>
        {loading ? (
          <Button danger icon={<StopOutlined />} onClick={handleCancel}>取消</Button>
        ) : (
          <Button type="primary" icon={<ThunderboltOutlined />} onClick={() => void handleCompile()}>
            开始推演
          </Button>
        )}
      </div>

      {/* 上下文输入（可选） */}
      <TextArea
        placeholder="补充上下文（可选）：实验条件、想观察的变量、希望挑战的目标区…"
        value={context}
        onChange={(event) => setContext(event.target.value)}
        autoSize={{ minRows: 1, maxRows: 3 }}
        style={{ marginBottom: 16 }}
      />

      {/* 阶段进度条 */}
      {(loading || pkg) && (
        <div className="snowy-stage-strip">
          {stageOrder.map((item) => {
            const status = stageStatus(stage, item);
            return (
              <div key={item} className={`snowy-stage-pill ${status === 'done' ? 'is-done' : status === 'active' ? 'is-active' : ''}`}>
                <span className="snowy-stage-pill__dot" />
                {stageText[item]}
              </div>
            );
          })}
        </div>
      )}

      {/* 示例 chip（仅在未生成时显示） */}
      {!pkg && !loading && (
        <Space wrap size={6} style={{ marginBottom: 16 }}>
          <Text type="secondary" style={{ fontSize: 13 }}>试试：</Text>
          {examples.map((example) => (
            <button
              key={example}
              type="button"
              className="snowy-chip"
              onClick={() => { setQuestion(example); void handleCompile(example); }}
            >
              {example.length > 22 ? `${example.slice(0, 22)}…` : example}
            </button>
          ))}
        </Space>
      )}

      {/* 错误 */}
      {errorText && !loading && (
        <Alert
          type="error"
          showIcon
          message="推演失败"
          description={errorText}
          action={<Button size="small" icon={<ReloadOutlined />} onClick={handleRegenerate}>重试</Button>}
          style={{ marginBottom: 16, borderRadius: 12 }}
        />
      )}
      {pkg?.warnings && pkg.warnings.length > 0 && (
        <Alert type="warning" showIcon message="生成提示" description={pkg.warnings.join('；')} style={{ marginBottom: 16, borderRadius: 12 }} />
      )}

      {/* 证据折叠条 */}
      {pkg && (
        <>
          <div
            className={`snowy-evidence-bar ${evidenceOpen ? 'is-open' : ''}`}
            onClick={() => setEvidenceOpen((v) => !v)}
            role="button"
            tabIndex={0}
            onKeyDown={(event) => { if (event.key === 'Enter') setEvidenceOpen((v) => !v); }}
          >
            <span className="snowy-evidence-bar__icon"><BookOutlined /></span>
            <div className="snowy-evidence-bar__main">
              <div style={{ fontSize: 14, fontWeight: 500, color: 'var(--color-text)' }}>
                基于 {pkg.evidence_refs?.length || 0} 条课本/考纲证据 · 置信度 {Math.round(confidence * 100)}%
                {pkg.learning_model.knowledge_tags && pkg.learning_model.knowledge_tags.length > 0 && (
                  <span style={{ color: 'var(--color-text-muted)', fontWeight: 400, marginLeft: 8 }}>
                    · {pkg.learning_model.knowledge_tags.slice(0, 3).join('、')}
                  </span>
                )}
              </div>
              <div style={{ fontSize: 12, color: 'var(--color-text-muted)' }}>
                <Space size={8} wrap>
                  <span>{evidenceOpen ? '点击收起' : '点击展开查看完整证据列表'}</span>
                  <CurriculumBadge subject={normalizedSubject(pkg.domain)} tags={evidenceTags} compact />
                </Space>
              </div>
            </div>
            <RightOutlined className="snowy-evidence-bar__caret" />
          </div>
          {evidenceOpen && (
            <div className="snowy-evidence-detail">
              <Space wrap style={{ marginBottom: 12 }}>
                {evidenceTags.map((tag) => (
                  <Tag key={tag} color="default" bordered={false} style={{ background: 'var(--color-primary-soft)', color: 'var(--color-primary)' }}>{tag}</Tag>
                ))}
              </Space>
              <List
                size="small"
                dataSource={pkg.evidence_refs || []}
                renderItem={(item, idx) => (
                  <List.Item style={{ padding: '8px 0', borderBottom: '1px solid var(--color-divider)' }}>
                    <Space align="start" size={8} style={{ width: '100%' }}>
                      <span className="snowy-citation-num">{idx + 1}</span>
                      <div style={{ flex: 1, minWidth: 0 }}>
                        <Tag bordered={false} style={{ background: 'var(--color-bg-subtle)', color: 'var(--color-text-muted)' }}>{item.source_type}</Tag>
                        <Text type="secondary" style={{ fontSize: 13 }}>{item.snippet}</Text>
                      </div>
                      <Text type="success" style={{ fontSize: 12 }}>{Math.round((item.confidence || 0) * 100)}%</Text>
                    </Space>
                  </List.Item>
                )}
                locale={{ emptyText: '暂无证据片段' }}
              />
            </div>
          )}
        </>
      )}

      {/* 两栏：画布 + 教练 */}
      <div className="snowy-modeling-grid">
        {/* 左：画布 */}
        <div className="snowy-canvas">
          <div className="snowy-canvas__head">
            <div>
              <div style={{ fontSize: 13, color: 'var(--color-text-muted)' }}>
                {pkg ? (pkg.learning_model.topic || '当前模型') : '模型画布'}
              </div>
              <div style={{ fontSize: 16, fontWeight: 600 }}>
                {pkg?.learning_model.learning_goal || subjectCanvasTitle(subject)}
              </div>
            </div>
            <Space wrap size={6}>
              {pkg && (
                <>
                  <Tag color={confidence >= 0.8 ? 'green' : confidence >= 0.55 ? 'orange' : 'red'} bordered={false}>
                    可信 {Math.round(confidence * 100)}%
                  </Tag>
                  <ReactionBar targetType="model_package" targetID={pkg.package_id} size="small" showUsers={false} />
                </>
              )}
            </Space>
          </div>

          {loading && !pkg && (
            <SkeletonPreview height={420} label="正在编译模型包…绑定证据、生成结构化交互模型" />
          )}

          {!loading && !pkg && (
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description={(
                <Space direction="vertical" align="center" size={12}>
                  <Text type="secondary">输入问题后，Snowy 会先绑定证据，再生成结构化的可交互模型。</Text>
                  <Space wrap size={6}>
                    {examples.slice(0, 2).map((example) => (
                      <Button key={example} icon={<PlayCircleOutlined />} onClick={() => { setQuestion(example); void handleCompile(example); }}>
                        试试：{example.slice(0, 16)}…
                      </Button>
                    ))}
                  </Space>
                </Space>
              )}
              style={{ padding: '80px 0' }}
            />
          )}

          {pkg && pkg.domain === 'biology' && <GenerativeBiologyGraph spec={pkg.visualization_graph} values={values} />}
          {pkg && pkg.domain === 'chemistry' && <GenerativeChemistryCanvas pkg={pkg} values={values} />}
          {pkg && pkg.domain !== 'biology' && pkg.domain !== 'chemistry' && <GenerativePhysics3DCanvas spec={pkg.simulation_logic} values={values} />}
        </div>

        {/* 右：AI 教练 */}
        <aside className="snowy-coach">
          {!pkg ? (
            <div className="snowy-loading-card" style={{ padding: 24 }}>
              <Text type="secondary" style={{ fontSize: 13 }}>右侧将显示 AI 教练讲解、参数滑块、校验灯与微练习。</Text>
            </div>
          ) : (
            <>
              <section style={{ padding: 16, background: 'var(--color-bg-card)', border: '1px solid var(--color-border)', borderRadius: 12 }}>
                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 12 }}>
                  <Title level={4} style={{ margin: 0, fontSize: 14, fontWeight: 600 }}>AI 教练</Title>
                  <Button size="small" icon={<ReloadOutlined />} onClick={handleRegenerate}>再推</Button>
                </div>
                <Paragraph style={{ marginBottom: 12, fontSize: 14, lineHeight: 1.7 }}>{pkg.reasoning_trace.summary}</Paragraph>
                {(pkg.reasoning_trace.key_steps || []).slice(0, 6).map((step, idx) => (
                  <div className="snowy-reasoning-step" key={idx}>
                    <span className="snowy-reasoning-step__num">{idx + 1}</span>
                    <span className="snowy-reasoning-step__text">{step}</span>
                  </div>
                ))}
              </section>

              {validationChecks.length > 0 && (
                <section style={{ padding: 16, background: 'var(--color-bg-card)', border: '1px solid var(--color-border)', borderRadius: 12 }}>
                  <Title level={4} style={{ margin: '0 0 12px', fontSize: 14, fontWeight: 600 }}>模型校验</Title>
                  <div className="snowy-check-grid">
                    {validationChecks.map(([label, ok]) => (
                      <div key={label} className={`snowy-check-item ${ok ? '' : 'snowy-check-item--fail'}`}>
                        {ok ? <CheckCircleFilled style={{ fontSize: 16 }} /> : <CloseCircleFilled style={{ fontSize: 16 }} />}
                        <span>{label}</span>
                      </div>
                    ))}
                  </div>
                </section>
              )}

              <InteractionPlanPanel pkg={pkg} values={values} subject={normalizedSubject(pkg.domain)} onChange={(name, value) => setValues((prev) => ({ ...prev, [name]: value }))} onRegenerate={() => void handleCompile()} />

              <Collapse
                ghost
                items={[
                  {
                    key: 'validation',
                    label: '完整校验报告',
                    children: <ValidationReportPanel report={pkg.validation_report} />,
                  },
                  ...(pkg.assessment_tasks && pkg.assessment_tasks.length > 0 ? [{
                    key: 'practice',
                    label: `微练习（${pkg.assessment_tasks.length}）`,
                    children: (
                      <Space direction="vertical" size={10} style={{ width: '100%' }}>
                        {pkg.assessment_tasks.slice(0, 3).map((task, idx) => (
                          <div key={idx} style={{ padding: 10, borderRadius: 8, background: 'var(--color-bg-subtle)' }}>
                            <Tag color="gold" bordered={false}>{task.task_type}</Tag>
                            <div style={{ fontSize: 13, marginTop: 6, lineHeight: 1.6 }}>{task.question}</div>
                            {task.next_action && <div style={{ fontSize: 12, color: 'var(--color-text-muted)', marginTop: 4 }}>下一步：{task.next_action}</div>}
                          </div>
                        ))}
                      </Space>
                    ),
                  }] : []),
                  ...((pkg.regeneration_hints || []).length > 0 ? [{
                    key: 'hints',
                    label: '再推理建议',
                    children: <Text type="secondary" style={{ fontSize: 13 }}>{(pkg.regeneration_hints || []).map((hint) => hint.message).join('；')}</Text>,
                  }] : []),
                ]}
              />
            </>
          )}
        </aside>
      </div>
    </div>
  );
}

export default function ModelingPage() {
  return (
    <Suspense fallback={<div className="snowy-loading-card" style={{ margin: '40px auto', maxWidth: 360 }}><span className="snowy-spinner" /><span>加载中…</span></div>}>
      <ModelingPageInner />
    </Suspense>
  );
}
