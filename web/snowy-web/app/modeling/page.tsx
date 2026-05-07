'use client';

import React, { Suspense, useCallback, useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'next/navigation';
import {
  Alert,
  Button,
  Card,
  Empty,
  Input,
  List,
  Progress,
  Segmented,
  Space,
  Spin,
  Tag,
  Typography,
  message,
} from 'antd';
import {
  BranchesOutlined,
  CheckCircleOutlined,
  CompassOutlined,
  ExperimentOutlined,
  PlayCircleOutlined,
  RadarChartOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
  StarOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons';
import { api, type EvidenceRef, type GenerativeModelPackage, type VariableSpec } from '@/lib/api';
import GenerativePhysicsCanvas from '@/components/generative/GenerativePhysicsCanvas';
import GenerativeBiologyGraph from '@/components/generative/GenerativeBiologyGraph';
import InteractionPlanPanel from '@/components/generative/InteractionPlanPanel';
import ValidationReportPanel from '@/components/generative/ValidationReportPanel';

const { Title, Paragraph, Text } = Typography;
const { TextArea } = Input;

type ModelingSubject = 'physics' | 'biology';
type Stage = 'idle' | 'grounding' | 'reasoning' | 'validating' | 'rendering' | 'done' | 'error';

const stageText: Record<Stage, string> = {
  idle: '等待输入',
  grounding: '检索证据',
  reasoning: 'AI 推理',
  validating: '领域校验',
  rendering: '通用渲染',
  done: '任务完成',
  error: '失败',
};

const stagePercent: Record<Stage, number> = {
  idle: 0,
  grounding: 20,
  reasoning: 55,
  validating: 78,
  rendering: 92,
  done: 100,
  error: 100,
};

const stageOrder: Stage[] = ['grounding', 'reasoning', 'validating', 'rendering', 'done'];

const subjectExamples: Record<ModelingSubject, string[]> = {
  physics: [
    '平抛运动如何影响水平位移？初速度20m/s，高度20m',
    '牛顿第二定律在斜面题中怎么用？',
    '弹簧振子简谐运动中周期和质量、劲度系数有什么关系？',
  ],
  biology: [
    '光照强度如何影响有机物积累？',
    '遗传分离定律如何推导子代表现型比例？',
    '酶活性受温度影响的实验变量如何设计？',
  ],
};

function normalizeSubject(value: string | null): ModelingSubject {
  return value === 'biology' ? 'biology' : 'physics';
}

function initialValues(pkg: GenerativeModelPackage | null): Record<string, number> {
  const vars: VariableSpec[] = pkg?.simulation_logic?.variables || pkg?.generative_model.variables || [];
  return vars.reduce<Record<string, number>>((acc, variable) => {
    acc[variable.name] = variable.default;
    return acc;
  }, {});
}

function ModelingPageInner() {
  const searchParams = useSearchParams();
  const [subject, setSubject] = useState<ModelingSubject>(normalizeSubject(searchParams.get('type') || searchParams.get('subject')));
  const [question, setQuestion] = useState(searchParams.get('q') || '');
  const [context, setContext] = useState('');
  const [pkg, setPkg] = useState<GenerativeModelPackage | null>(null);
  const [values, setValues] = useState<Record<string, number>>({});
  const [loading, setLoading] = useState(false);
  const [stage, setStage] = useState<Stage>('idle');
  const [errorText, setErrorText] = useState<string | null>(null);

  const examples = subjectExamples[subject];
  const isPhysics = subject === 'physics';
  const confidence = pkg?.validation_report?.confidence ?? pkg?.confidence ?? 0;

  const evidenceTags = useMemo(() => {
    const tags = new Set<string>();
    pkg?.evidence_refs?.forEach((ref) => ref.knowledge_tags?.forEach((tag) => tags.add(tag)));
    pkg?.learning_model?.knowledge_tags?.forEach((tag) => tags.add(tag));
    return Array.from(tags).slice(0, 8);
  }, [pkg]);

  const validationChecks = useMemo(() => {
    const report = pkg?.validation_report;
    if (!report) return [];
    return [
      ['Schema', report.schema_valid],
      ['Evidence', report.evidence_valid],
      ['Domain', report.domain_valid],
      ['Safety', report.safety_valid],
    ] as const;
  }, [pkg]);

  const handleCompile = useCallback(async (nextQuestion?: string, nextSubject?: ModelingSubject) => {
    const text = (nextQuestion || question).trim();
    const selectedSubject = nextSubject || subject;
    if (!text) return;
    setLoading(true);
    setErrorText(null);
    setPkg(null);
    setValues({});
    setStage('grounding');
    try {
      window.setTimeout(() => setStage((prev) => (prev === 'grounding' ? 'reasoning' : prev)), 150);
      let compileGrounding: { citations: EvidenceRef[]; knowledge_tags: string[] } = { citations: [], knowledge_tags: [] };
      try {
        const searchRes = await api.searchQuery({ query: text, filters: { subject: selectedSubject, grade: 'high_school' } });
        compileGrounding = {
          citations: (searchRes.data?.citations || []).map((citation) => ({
            doc_id: citation.doc_id,
            source_type: citation.source_type,
            snippet: citation.snippet,
            confidence: citation.score,
            knowledge_tags: searchRes.data?.knowledge_tags || [],
          })),
          knowledge_tags: searchRes.data?.knowledge_tags || [],
        };
      } catch {
        compileGrounding = { citations: [], knowledge_tags: [] };
      }
      const res = await api.modelingCompile({
        message: text,
        domain: selectedSubject,
        grade_band: 'high_school',
        target_mode: 'interactive_model',
        context: {
          source_page: 'modeling-v4',
          user_notes: context || undefined,
          citations: compileGrounding.citations,
          knowledge_tags: compileGrounding.knowledge_tags,
        },
      });
      setStage('validating');
      const data = res.data ?? null;
      setPkg(data);
      setValues(initialValues(data));
      setStage('rendering');
      window.setTimeout(() => setStage('done'), 120);
    } catch (error) {
      const msg = error instanceof Error ? error.message : '生成式建模失败';
      setErrorText(msg);
      setStage('error');
      message.error(msg);
    } finally {
      setLoading(false);
    }
  }, [context, question, subject]);

  useEffect(() => {
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
      await api.addFavorite({ target_type: subject, target_id: pkg?.package_id || question, title: question || pkg?.learning_model.learning_goal || 'Snowy v4 模型包' });
      message.success('收藏成功');
    } catch (error) {
      message.error(error instanceof Error ? error.message : '收藏失败');
    }
  };

  const handleRegenerate = async () => {
    if (!question.trim()) return;
    await handleCompile(question, subject);
  };

  const renderCanvas = () => {
    if (!pkg) {
      return (
        <Empty
          description={(
            <Space direction="vertical" align="center">
              <Text type="secondary">输入问题后，Snowy v4 会先绑定证据，再生成结构化模型包。</Text>
              <Space wrap>
                {examples.slice(0, 2).map((example) => (
                  <Button key={example} icon={<PlayCircleOutlined />} onClick={() => { setQuestion(example); void handleCompile(example); }}>
                    试试：{example.slice(0, 18)}...
                  </Button>
                ))}
              </Space>
            </Space>
          )}
          style={{ padding: '80px 0' }}
        />
      );
    }
    if (pkg.domain === 'biology') return <GenerativeBiologyGraph spec={pkg.visualization_graph} />;
    return <GenerativePhysicsCanvas spec={pkg.simulation_logic} values={values} />;
  };

  return (
    <div className="snowy-page">
      <Space direction="vertical" size="middle" style={{ width: '100%' }}>
        <Card className="snowy-glass-strong">
          <Space direction="vertical" size="middle" style={{ width: '100%' }}>
            <div className="snowy-page-heading" style={{ marginBottom: 0 }}>
              <span className="snowy-kicker"><RadarChartOutlined /> Science Modeling Cockpit</span>
              <Title level={1}>科学建模舱</Title>
              <Paragraph>
                v4 三栏工作台：左侧锁定问题与证据，中间渲染模型画布，右侧由 AI 教练管理参数、挑战、微练习和校验报告。
              </Paragraph>
            </div>

            <div className="snowy-stage-strip">
              {stageOrder.map((item) => (
                <div className={`snowy-stage-pill ${stage === item || (stage === 'done' && item === 'done') ? 'is-active' : ''}`} key={item}>
                  {stageText[item]}
                </div>
              ))}
            </div>

            <Space wrap>
              <Segmented
                value={subject}
                options={[
                  { label: '物理仿真任务', value: 'physics', icon: <ExperimentOutlined /> },
                  { label: '生物可视化任务', value: 'biology', icon: <BranchesOutlined /> },
                ]}
                onChange={(value) => { setSubject(value as ModelingSubject); setPkg(null); setValues({}); setStage('idle'); }}
              />
              <Tag color={stage === 'error' ? 'red' : stage === 'done' ? 'green' : loading ? 'cyan' : 'default'}>阶段：{stageText[stage]}</Tag>
              {pkg && <Tag color={confidence >= 0.8 ? 'green' : confidence >= 0.55 ? 'orange' : 'red'}>可信度：{Math.round(confidence * 100)}%</Tag>}
              {pkg?.status && <Tag>{pkg.status}</Tag>}
            </Space>

            <TextArea rows={2} placeholder={isPhysics ? '输入物理题目或建模目标，例如：平抛运动怎样命中目标区？' : '输入生物问题、过程或实验题，例如：光照强度如何影响有机物积累？'} value={question} onChange={(event) => setQuestion(event.target.value)} />
            <TextArea rows={1} placeholder="补充上下文（可选）：实验条件、题干补充、想观察的变量、希望挑战的目标区..." value={context} onChange={(event) => setContext(event.target.value)} />
            <Space wrap>
              <Button type="primary" size="large" icon={<ThunderboltOutlined />} loading={loading} onClick={() => void handleCompile()}>生成 v4 模型包</Button>
              <Button disabled={!question.trim() || loading} icon={<ReloadOutlined />} onClick={handleRegenerate}>再推理</Button>
              <Button disabled={!question.trim()} icon={<StarOutlined />} onClick={handleFavorite}>收藏任务</Button>
              {examples.map((example) => <Button key={example} size="small" disabled={loading} onClick={() => { setQuestion(example); void handleCompile(example); }}>{example.slice(0, 14)}...</Button>)}
            </Space>
            {loading && <Progress percent={stagePercent[stage]} showInfo={false} status="active" strokeColor={{ '0%': '#22d3ee', '100%': '#34d399' }} />}
          </Space>
        </Card>

        {errorText && <Alert type="error" showIcon message="建模失败" description={errorText} action={<Button size="small" onClick={handleRegenerate}>重试</Button>} />}
        {pkg?.warnings && pkg.warnings.length > 0 && <Alert type="warning" showIcon message="生成提示" description={pkg.warnings.join('；')} />}

        <div className="snowy-modeling-grid">
          <Card title={<Space><SafetyCertificateOutlined /> 问题与证据</Space>} className="snowy-glass" styles={{ body: { minHeight: 620 } }}>
            {pkg ? (
              <Space direction="vertical" style={{ width: '100%' }}>
                <Paragraph strong>{pkg.question}</Paragraph>
                <Space wrap>{evidenceTags.map((tag) => <Tag key={tag} color="cyan">{tag}</Tag>)}</Space>
                <div className="snowy-stat-row" style={{ gridTemplateColumns: 'repeat(2, minmax(0, 1fr))' }}>
                  <div className="snowy-stat"><strong>{pkg.evidence_refs?.length || 0}</strong><span>证据片段</span></div>
                  <div className="snowy-stat"><strong>{Math.round(confidence * 100)}%</strong><span>校验可信度</span></div>
                </div>
                <List
                  size="small"
                  dataSource={pkg.evidence_refs || []}
                  renderItem={(item) => <List.Item><Text type="secondary">[{item.source_type}] {item.snippet}</Text></List.Item>}
                />
              </Space>
            ) : <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="生成后展示引用证据、知识标签与可信度" />}
          </Card>

          <Card
            title={<Space><ExperimentOutlined /> {pkg?.learning_model.learning_goal || '模型画布'}</Space>}
            extra={pkg && <Tag color={pkg.domain === 'biology' ? 'gold' : 'green'}>{pkg.domain}</Tag>}
            className="snowy-glass"
            styles={{ body: { minHeight: 620 } }}
          >
            {loading && !pkg ? <div style={{ padding: 120, textAlign: 'center' }}><Spin size="large" tip="AI 正在生成模型包..." /></div> : renderCanvas()}
          </Card>

          <Space direction="vertical" style={{ width: '100%' }} size="middle">
            {pkg ? (
              <>
                <Card size="small" title={<Space><CompassOutlined /> AI 教练</Space>} className="snowy-glass">
                  <Paragraph>{pkg.reasoning_trace.summary}</Paragraph>
                  <List size="small" dataSource={pkg.reasoning_trace.key_steps || []} renderItem={(item) => <List.Item><Text type="secondary">{item}</Text></List.Item>} />
                </Card>

                {validationChecks.length > 0 && (
                  <Card size="small" title={<Space><CheckCircleOutlined /> 校验灯</Space>} className="snowy-glass">
                    <Space wrap>
                      {validationChecks.map(([label, ok]) => <Tag key={label} color={ok ? 'green' : 'red'}>{label}</Tag>)}
                    </Space>
                  </Card>
                )}

                <InteractionPlanPanel pkg={pkg} values={values} onChange={(name, value) => setValues((prev) => ({ ...prev, [name]: value }))} />
                <ValidationReportPanel report={pkg.validation_report} />

                {pkg.assessment_tasks && pkg.assessment_tasks.length > 0 && (
                  <Card size="small" title="微练习队列" className="snowy-glass">
                    <List
                      size="small"
                      dataSource={pkg.assessment_tasks.slice(0, 3)}
                      renderItem={(task) => (
                        <List.Item>
                          <Space direction="vertical" size={2}>
                            <Tag color="gold">{task.task_type}</Tag>
                            <Text>{task.question}</Text>
                            {task.next_action && <Text type="secondary">下一步：{task.next_action}</Text>}
                          </Space>
                        </List.Item>
                      )}
                    />
                  </Card>
                )}

                {(pkg.regeneration_hints || []).length > 0 && <Alert type="info" showIcon message="再推理建议" description={(pkg.regeneration_hints || []).map((hint) => hint.message).join('；')} />}
              </>
            ) : <Card className="snowy-glass"><Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="右侧将展示 AI 教练、参数、挑战、微练习和校验报告" /></Card>}
          </Space>
        </div>
      </Space>
    </div>
  );
}

export default function ModelingPage() {
  return (
    <Suspense fallback={<Spin />}>
      <ModelingPageInner />
    </Suspense>
  );
}
