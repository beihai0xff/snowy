'use client';

import React, { Suspense, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useSearchParams } from 'next/navigation';
import {
  Alert,
  Button,
  Card,
  Col,
  Empty,
  Input,
  List,
  Progress,
  Row,
  Segmented,
  Slider,
  Space,
  Spin,
  Steps,
  Table,
  Tag,
  Typography,
  message,
} from 'antd';
import {
  BranchesOutlined,
  ExperimentOutlined,
  PlayCircleOutlined,
  ReloadOutlined,
  StarOutlined,
  StopOutlined,
} from '@ant-design/icons';
import {
  agentChatStream,
  api,
  type AgentPhysicsPayload,
  type BiologyModel,
  type ChatReq,
  type PhysicsModel,
  type RenderArtifact,
} from '@/lib/api';
import BiologyDiagram from '@/components/biology/BiologyDiagram';
import RenderPreviewSandbox from '@/components/common/RenderPreviewSandbox';

const { Title, Paragraph, Text } = Typography;
const { TextArea } = Input;

type ModelingSubject = 'physics' | 'biology';
type StreamStage = 'idle' | 'connecting' | 'thinking' | 'analyzing' | 'generating' | 'validating' | 'previewing' | 'done' | 'error' | 'aborted';
type PreviewStatus = 'idle' | 'loading' | 'ready' | 'updated' | 'error' | 'timeout';

const stageText: Record<StreamStage, string> = {
  idle: '等待输入',
  connecting: '连接 Agent',
  thinking: 'Agent 思考中',
  analyzing: '解析题目',
  generating: '生成预览配置',
  validating: '配置参数',
  previewing: '挂载演示',
  done: '完成',
  error: '失败',
  aborted: '已取消',
};

const stagePercent: Record<StreamStage, number> = {
  idle: 0,
  connecting: 10,
  thinking: 25,
  analyzing: 40,
  generating: 65,
  validating: 78,
  previewing: 90,
  done: 100,
  error: 100,
  aborted: 100,
};

const subjectExamples: Record<ModelingSubject, string[]> = {
  physics: [
    '质量2kg的物体受到6N水平力，展示 Rapier 3D 受力模型',
    '卫星绕地球做轨道运动，展示天体运动 3D 模型',
    '弹簧振子简谐运动，展示回复力和能量变化',
    '两个小球弹性碰撞，展示动量和能量变化',
    '生成一个3D空间中的抛体轨迹示意，初速度30m/s，角度35度',
  ],
  biology: [
    '光照强度对光合作用有机物积累的影响',
    '细胞膜的结构如何决定选择透过性？',
    '神经冲动在突触处如何传递？',
  ],
};

const PLAYBACK_SPEED_PRESETS = [0.25, 0.5, 1, 1.5, 2, 4];
const DEFAULT_PREVIEW_PROPS: Record<string, number> = { animation_speed: 1 };

function clampPlaybackSpeed(value: number | undefined): number {
  if (!Number.isFinite(value)) return 1;
  return Math.min(4, Math.max(0.1, value || 1));
}

function withDefaultAnimationSpeed(props: Record<string, number> = {}): Record<string, number> {
  return { ...props, animation_speed: clampPlaybackSpeed(props.animation_speed ?? 1) };
}

function PlaybackSpeedControl({
  value,
  disabled,
  hint,
  onChange,
}: {
  value: number;
  disabled?: boolean;
  hint?: string;
  onChange: (value: number) => void;
}) {
  const speed = clampPlaybackSpeed(value);
  return (
    <Card size="small" title="播放速率" extra={<Tag color="blue">{speed.toFixed(2)}x</Tag>}>
      <Space direction="vertical" style={{ width: '100%' }} size="small">
        <Segmented
          block
          disabled={disabled}
          value={PLAYBACK_SPEED_PRESETS.includes(speed) ? speed : 'custom'}
          options={[
            ...PLAYBACK_SPEED_PRESETS.map((item) => ({ label: `${item}x`, value: item })),
            { label: '自定义', value: 'custom', disabled: true },
          ]}
          onChange={(next) => {
            if (typeof next === 'number') onChange(next);
          }}
        />
        <Slider
          min={0.1}
          max={4}
          step={0.05}
          disabled={disabled}
          value={speed}
          tooltip={{ formatter: (next) => `${(next ?? speed).toFixed(2)}x` }}
          onChange={(next) => onChange(clampPlaybackSpeed(next))}
        />
        <Text type="secondary">{hint || '倍率会实时同步到当前动态演示；暂停/继续仍由预览区按钮控制。'}</Text>
      </Space>
    </Card>
  );
}

function normalizeSubject(input: string | null): ModelingSubject {
  return input === 'biology' ? 'biology' : 'physics';
}

function buildInitialProps(model: PhysicsModel): Record<string, number> {
  const merged: Record<string, number> = { ...(model.scene_spec?.default_props || {}) };
  model.parameters?.forEach((item) => {
    if (merged[item.name] === undefined) merged[item.name] = item.default;
  });
  return withDefaultAnimationSpeed(merged);
}

function ModelingPageInner() {
  const searchParams = useSearchParams();
  const abortRef = useRef<AbortController | null>(null);
  const [subject, setSubject] = useState<ModelingSubject>(normalizeSubject(searchParams.get('type') || searchParams.get('subject')));
  const [question, setQuestion] = useState(searchParams.get('q') || '');
  const [context, setContext] = useState('');

  const [physicsAnalysis, setPhysicsAnalysis] = useState<PhysicsModel | null>(null);
  const [biologyResult, setBiologyResult] = useState<BiologyModel | null>(null);
  const [artifact, setArtifact] = useState<RenderArtifact | null>(null);
  const [previewProps, setPreviewProps] = useState<Record<string, number>>(DEFAULT_PREVIEW_PROPS);

  const [loading, setLoading] = useState(false);
  const [stage, setStage] = useState<StreamStage>('idle');
  const [previewStatus, setPreviewStatus] = useState<PreviewStatus>('idle');
  const [errorText, setErrorText] = useState<string | null>(null);
  const [previewError, setPreviewError] = useState<string | null>(null);
  const [streamingText, setStreamingText] = useState('');
  const [startedAt, setStartedAt] = useState<number | null>(null);
  const [finishedAt, setFinishedAt] = useState<number | null>(null);

  const examples = subjectExamples[subject];
  const hasResult = Boolean(physicsAnalysis || biologyResult || artifact);
  const isPhysics = subject === 'physics';
  const isPhysics3DArtifact = artifact?.scene_type?.startsWith('physics_') === true && artifact.scene_type.includes('3d');
  const currentViewDimension = (previewProps.view_dimension ?? artifact?.render_manifest.initial_props?.view_dimension ?? 3) >= 2.5 ? '3d' : '2d';

  const elapsedText = useMemo(() => {
    if (!startedAt) return '0.0s';
    const end = finishedAt || Date.now();
    return `${Math.max(0, (end - startedAt) / 1000).toFixed(1)}s`;
  }, [finishedAt, startedAt]);

  const resetResult = useCallback(() => {
    abortRef.current?.abort();
    abortRef.current = null;
    setPhysicsAnalysis(null);
    setBiologyResult(null);
    setArtifact(null);
    setPreviewProps(DEFAULT_PREVIEW_PROPS);
    setPreviewStatus('idle');
    setErrorText(null);
    setPreviewError(null);
    setStreamingText('');
    setStage('idle');
    setStartedAt(null);
    setFinishedAt(null);
  }, []);

  useEffect(() => () => abortRef.current?.abort(), []);

  useEffect(() => {
    const q = searchParams.get('q');
    const nextSubject = normalizeSubject(searchParams.get('type') || searchParams.get('subject'));
    setSubject(nextSubject);
    if (q) {
      setQuestion(q);
      void handleAnalyze(q, nextSubject);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchParams]);

  const handlePreviewStatusChange = useCallback((status: Exclude<PreviewStatus, 'idle'>, detail?: string) => {
    setPreviewStatus(status);
    setPreviewError(status === 'error' || status === 'timeout' ? detail || '预览运行失败' : null);
    if (status === 'ready' || status === 'updated') {
      setStage((prev) => (prev === 'previewing' || prev === 'generating' ? 'done' : prev));
    }
  }, []);

  const handlePhysicsAnalyze = useCallback(async (text: string) => {
    const controller = new AbortController();
    abortRef.current = controller;
    const payload: ChatReq = {
      message: text,
      mode: 'physics',
      filters: context ? { subject: 'physics' } : undefined,
    };

    await agentChatStream(payload, (event) => {
      if (event.event === 'thinking') {
        setStage('thinking');
        return;
      }
      if (event.event === 'heartbeat') {
        const data = event.data as { tool?: string } | undefined;
        if (data?.tool === 'PhysicsAnalyzeTool') setStage('analyzing');
        else if (data?.tool === 'RenderCodeTool') setStage('generating');
        return;
      }
      if (event.event === 'content') {
        const data = event.data as { content?: string } | string;
        const content = typeof data === 'string' ? data : data?.content || '';
        if (content) setStreamingText(content);
        return;
      }
      if (event.event === 'tool_call') {
        const call = event.data as { tool?: string; status?: string };
        if (call.tool === 'PhysicsAnalyzeTool') setStage(call.status === 'success' ? 'validating' : 'analyzing');
        if (call.tool === 'RenderCodeTool') setStage(call.status === 'success' ? 'previewing' : 'generating');
        return;
      }
      if (event.event === 'render_code') {
        setArtifact(event.data as RenderArtifact);
        setPreviewStatus('loading');
        setStage('previewing');
        return;
      }
      if (event.event === 'preview') {
        const data = event.data as { status?: string; message?: string } | undefined;
        if (data?.status === 'error') {
          setPreviewStatus('error');
          setPreviewError(data.message || '预览挂载失败');
        } else if (data?.status === 'ready') {
          setPreviewStatus('ready');
        }
        return;
      }
      if (event.event === 'done') {
        const data = event.data as { structured_payload?: AgentPhysicsPayload; answer?: string };
        const payloadData = data.structured_payload;
        if (payloadData?.analysis) {
          setPhysicsAnalysis(payloadData.analysis);
          setPreviewProps(buildInitialProps(payloadData.analysis));
        }
        if (payloadData?.render_artifact) setArtifact(payloadData.render_artifact);
        if (data.answer) setStreamingText(data.answer);
        setStage('done');
        setFinishedAt(Date.now());
      }
    }, {
      signal: controller.signal,
      onOpen: () => setStage('thinking'),
      onError: (error) => {
        if (error.name !== 'AbortError') {
          setStage('error');
          setErrorText(error.message);
        }
      },
    });
  }, [context]);

  const handleBiologyAnalyze = useCallback(async (text: string) => {
    setStage('analyzing');
    const res = await api.biologyAnalyze({ question: text, context: context || undefined });
    const model = res.data ?? null;
    setBiologyResult(model);
    if (!model) return;
    setPreviewProps(withDefaultAnimationSpeed(model.scene_spec?.default_props || {}));
    setStreamingText(model.result_summary);
    if (model.scene_spec) {
      setStage('generating');
      setPreviewStatus('loading');
      try {
        const renderRes = await api.renderGenerate({ scene_spec: model.scene_spec, render_mode: model.scene_spec.render_mode || 'html_iframe' });
        setArtifact(renderRes.data ?? null);
        setStage('previewing');
      } catch (error) {
        setPreviewError(error instanceof Error ? error.message : '生物动态演示生成失败');
        setPreviewStatus('error');
      }
    }
    setStage('done');
    setFinishedAt(Date.now());
  }, [context]);

  const handleAnalyze = useCallback(async (nextQuestion?: string, nextSubject?: ModelingSubject) => {
    const text = (nextQuestion || question).trim();
    const selectedSubject = nextSubject || subject;
    if (!text) return;

    abortRef.current?.abort();
    setLoading(true);
    setStartedAt(Date.now());
    setFinishedAt(null);
    setStage('connecting');
    setErrorText(null);
    setPreviewError(null);
    setPreviewStatus('idle');
    setStreamingText('');
    setPhysicsAnalysis(null);
    setBiologyResult(null);
    setArtifact(null);
    setPreviewProps(DEFAULT_PREVIEW_PROPS);

    try {
      if (selectedSubject === 'physics') await handlePhysicsAnalyze(text);
      else await handleBiologyAnalyze(text);
    } catch (error) {
      if (error instanceof Error && error.name === 'AbortError') return;
      const messageText = error instanceof Error ? error.message : '建模失败';
      setStage('error');
      setErrorText(messageText);
      message.error(messageText);
    } finally {
      setLoading(false);
      abortRef.current = null;
    }
  }, [question, subject, handlePhysicsAnalyze, handleBiologyAnalyze]);

  const cancelAnalyze = () => {
    abortRef.current?.abort();
    abortRef.current = null;
    setLoading(false);
    setStage('aborted');
  };

  const handleFavorite = async () => {
    try {
      await api.addFavorite({ target_type: subject, target_id: question, title: question });
      message.success('收藏成功');
    } catch (error) {
      message.error(error instanceof Error ? error.message : '收藏失败');
    }
  };

  const renderPhysicsSidePanel = () => {
    if (!physicsAnalysis) return null;
    return (
      <Space direction="vertical" style={{ width: '100%' }} size="middle">
        <PlaybackSpeedControl
          value={previewProps.animation_speed ?? 1}
          onChange={(value) => setPreviewProps((prev) => ({ ...prev, animation_speed: value }))}
        />
        <Card size="small" title="参数" extra={<Tag color="green">{physicsAnalysis.model_type}</Tag>}>
          {physicsAnalysis.parameters && physicsAnalysis.parameters.length > 0 ? (
            <Space direction="vertical" style={{ width: '100%' }}>
              {isPhysics3DArtifact && (
                <div>
                  <Text type="secondary">视图模式</Text>
                  <Segmented
                    block
                    value={currentViewDimension}
                    options={[{ label: '3D', value: '3d' }, { label: '2D', value: '2d' }]}
                    onChange={(value) => setPreviewProps((prev) => ({ ...prev, view_dimension: value === '3d' ? 3 : 2 }))}
                    style={{ marginTop: 8 }}
                  />
                </div>
              )}
              {physicsAnalysis.parameters.map((parameter) => (
                <div key={parameter.name}>
                  <Text>{parameter.label} <Text type="secondary">{previewProps[parameter.name] ?? parameter.default} {parameter.unit}</Text></Text>
                  <Slider
                    min={parameter.min}
                    max={parameter.max}
                    step={parameter.step}
                    value={previewProps[parameter.name] ?? parameter.default}
                    onChange={(value) => setPreviewProps((prev) => ({ ...prev, [parameter.name]: value }))}
                  />
                </div>
              ))}
            </Space>
          ) : (
            <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无可调参数" />
          )}
        </Card>

        <Card size="small" title="摘要">
          <Paragraph style={{ marginBottom: 8 }}>{physicsAnalysis.scene_spec?.summary || physicsAnalysis.result_summary}</Paragraph>
          {physicsAnalysis.conditions.length > 0 && (
            <Table
              size="small"
              pagination={false}
              dataSource={physicsAnalysis.conditions.map((item, index) => ({ ...item, key: index }))}
              columns={[
                { title: '量', dataIndex: 'name' },
                { title: '值', dataIndex: 'value' },
                { title: '单位', dataIndex: 'unit' },
              ]}
            />
          )}
        </Card>
      </Space>
    );
  };

  const renderBiologySidePanel = () => {
    if (!biologyResult) return null;
    return (
      <Space direction="vertical" style={{ width: '100%' }} size="middle">
        <PlaybackSpeedControl
          value={previewProps.animation_speed ?? 1}
          disabled={!artifact}
          hint={artifact ? '倍率会通过 postMessage 同步到生物动态演示。' : '生成动态演示后生效；概念图谱不受播放速率影响。'}
          onChange={(value) => setPreviewProps((prev) => ({ ...prev, animation_speed: value }))}
        />
        <Card size="small" title="摘要" extra={<Tag color="purple">{biologyResult.topic}</Tag>}>
          <Paragraph>{biologyResult.result_summary}</Paragraph>
          {biologyResult.concepts.length > 0 && (
            <Space wrap>
              {biologyResult.concepts.slice(0, 10).map((concept) => <Tag key={`${concept.type}-${concept.name}`}>{concept.name}</Tag>)}
            </Space>
          )}
        </Card>
        {biologyResult.process_steps.length > 0 && (
          <Card size="small" title="过程">
            <Steps
              size="small"
              direction="vertical"
              current={biologyResult.process_steps.length}
              items={biologyResult.process_steps.slice(0, 4).map((step) => ({
                title: step.title,
                description: <Text type="secondary">{step.content}</Text>,
              }))}
            />
          </Card>
        )}
      </Space>
    );
  };

  const renderMainResult = () => {
    if (!hasResult) {
      return (
        <Empty
          description={(
            <Space direction="vertical" align="center">
              <Text type="secondary">选择学科并输入问题，建模结果会优先在这里大屏展示。</Text>
              <Space wrap>
                {examples.slice(0, 2).map((example) => (
                  <Button key={example} icon={<PlayCircleOutlined />} onClick={() => { setQuestion(example); void handleAnalyze(example); }}>
                    试试：{example.slice(0, 16)}...
                  </Button>
                ))}
              </Space>
            </Space>
          )}
          style={{ padding: '80px 0' }}
        />
      );
    }

    if (artifact) {
      return <RenderPreviewSandbox artifact={artifact} propsData={previewProps} onStatusChange={handlePreviewStatusChange} />;
    }

    if (biologyResult?.diagram) {
      return <BiologyDiagram spec={biologyResult.diagram} />;
    }

    return <Empty description="当前结果没有动态预览，已在右侧展示结构化摘要。" />;
  };

  return (
    <div style={{ maxWidth: 1440, margin: '0 auto' }}>
      <Space direction="vertical" size="middle" style={{ width: '100%' }}>
        <Card styles={{ body: { padding: 16 } }}>
          <Row gutter={[16, 12]} align="middle">
            <Col xs={24} lg={6}>
              <Title level={3} style={{ marginBottom: 4 }}>
                {isPhysics ? <ExperimentOutlined /> : <BranchesOutlined />} 统一建模
              </Title>
              <Text type="secondary">物理与生物建模已合并，结果区域优先大屏展示。</Text>
            </Col>
            <Col xs={24} lg={18}>
              <Space direction="vertical" style={{ width: '100%' }} size="small">
                <Space wrap>
                  <Segmented
                    value={subject}
                    options={[
                      { label: '物理建模', value: 'physics', icon: <ExperimentOutlined /> },
                      { label: '生物建模', value: 'biology', icon: <BranchesOutlined /> },
                    ]}
                    onChange={(value) => {
                      const next = value as ModelingSubject;
                      setSubject(next);
                      resetResult();
                    }}
                  />
                  <Tag color={stage === 'error' ? 'red' : stage === 'done' ? 'green' : loading ? 'blue' : 'default'}>阶段：{stageText[stage]}</Tag>
                  <Tag color={previewStatus === 'error' ? 'red' : previewStatus === 'ready' || previewStatus === 'updated' ? 'green' : 'default'}>预览：{previewStatus}</Tag>
                  <Tag>耗时：{elapsedText}</Tag>
                </Space>
                <TextArea
                  placeholder={isPhysics ? '输入物理题目，如：质量2kg的物体受到6N水平力...' : '输入生物问题，如：光照强度对光合作用有机物积累的影响...'}
                  rows={2}
                  value={question}
                  onChange={(event) => setQuestion(event.target.value)}
                />
                <TextArea
                  placeholder="补充上下文（可选，默认收起为较少展示信息）"
                  rows={1}
                  value={context}
                  onChange={(event) => setContext(event.target.value)}
                />
                <Space wrap>
                  <Button type="primary" size="large" loading={loading} onClick={() => void handleAnalyze()}>
                    开始建模
                  </Button>
                  <Button disabled={!loading} icon={<StopOutlined />} onClick={cancelAnalyze}>取消</Button>
                  <Button disabled={!question.trim()} icon={<StarOutlined />} onClick={handleFavorite}>收藏</Button>
                  {examples.map((example) => (
                    <Button
                      key={example}
                      size="small"
                      icon={<PlayCircleOutlined />}
                      disabled={loading}
                      onClick={() => { setQuestion(example); void handleAnalyze(example); }}
                    >
                      {example.slice(0, 14)}...
                    </Button>
                  ))}
                </Space>
                {loading && <Progress percent={stagePercent[stage]} showInfo={false} status="active" />}
              </Space>
            </Col>
          </Row>
        </Card>

        {(errorText || previewError) && (
          <Alert
            type={errorText ? 'error' : 'warning'}
            showIcon
            message={errorText ? '建模失败' : '预览提示'}
            description={errorText || previewError}
            action={<Button size="small" icon={<ReloadOutlined />} onClick={() => void handleAnalyze()}>重试</Button>}
          />
        )}

        <Row gutter={[16, 16]} align="top">
          <Col xs={24} xl={18}>
            <Card
              title={isPhysics ? '物理仿真 / 动态预览' : '生物动态演示 / 概念图谱'}
              extra={artifact && <Tag color="green">{artifact.render_manifest.framework}</Tag>}
              styles={{ body: { minHeight: 680 } }}
            >
              {loading && !artifact && !biologyResult ? (
                <div style={{ minHeight: 560, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                  <Spin size="large" tip={isPhysics ? '正在解析并初始化 Rapier 3D...' : '正在解析并生成动态演示...'} />
                </div>
              ) : renderMainResult()}
            </Card>
          </Col>
          <Col xs={24} xl={6}>
            {isPhysics ? renderPhysicsSidePanel() : renderBiologySidePanel()}
            {streamingText && (
              <Card size="small" title="讲解" style={{ marginTop: 16 }}>
                <Paragraph style={{ marginBottom: 0 }}>{streamingText}</Paragraph>
              </Card>
            )}
            {biologyResult?.diagram && artifact && (
              <Card size="small" title="概念图谱" style={{ marginTop: 16 }}>
                <BiologyDiagram spec={biologyResult.diagram} />
              </Card>
            )}
            {physicsAnalysis?.steps && physicsAnalysis.steps.length > 0 && (
              <Card size="small" title="推导步骤" style={{ marginTop: 16 }}>
                <List
                  size="small"
                  dataSource={physicsAnalysis.steps.slice(0, 3)}
                  renderItem={(step) => <List.Item><Text type="secondary">{step.title}</Text></List.Item>}
                />
              </Card>
            )}
          </Col>
        </Row>
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
