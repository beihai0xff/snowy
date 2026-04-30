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
  Progress,
  Row,
  Segmented,
  Slider,
  Space,
  Spin,
  Steps,
  Table,
  Tag,
  Timeline,
  Typography,
  message,
} from 'antd';
import { ExperimentOutlined, PlayCircleOutlined, StarOutlined, StopOutlined } from '@ant-design/icons';
import {
  agentChatStream,
  api,
  type AgentPhysicsPayload,
  type ChatReq,
  type PhysicsModel,
  type RenderArtifact,
} from '@/lib/api';
import RenderPreviewSandbox from '@/components/common/RenderPreviewSandbox';
import CodeBundleViewer from '@/components/physics/CodeBundleViewer';
import MarkdownText from '@/components/common/MarkdownText';

const { Title, Paragraph, Text } = Typography;
const { TextArea } = Input;

type StreamStage = 'idle' | 'connecting' | 'thinking' | 'analyzing' | 'generating' | 'validating' | 'previewing' | 'done' | 'error' | 'aborted';

type StreamLogItem = {
  id: number;
  event: string;
  label: string;
  time: string;
  status?: 'success' | 'error' | 'processing';
};

const stageText: Record<StreamStage, string> = {
  idle: '等待输入',
  connecting: '连接 SSE',
  thinking: 'Agent 思考中',
  analyzing: '解析题目',
  generating: '生成前端代码',
  validating: '校验代码',
  previewing: '浏览器预览挂载中',
  done: '完成',
  error: '失败',
  aborted: '已取消',
};

const stageColor: Record<StreamStage, string> = {
  idle: 'default',
  connecting: 'blue',
  thinking: 'blue',
  analyzing: 'geekblue',
  generating: 'cyan',
  validating: 'gold',
  previewing: 'gold',
  done: 'green',
  error: 'red',
  aborted: 'default',
};

const stagePercent: Record<StreamStage, number> = {
  idle: 0,
  connecting: 10,
  thinking: 20,
  analyzing: 35,
  generating: 60,
  validating: 75,
  previewing: 88,
  done: 100,
  error: 100,
  aborted: 100,
};

const exampleQuestions = [
  '画出质量2kg的物体在6N水平力作用下的 3D 受力模型，显示方块、地面网格、坐标轴、力矢量和加速度矢量',
  '平抛运动，初速度20m/s，抛射角45度，求2秒后的运动轨迹并用浏览器渲染',
  '生成一个3D空间中的抛体轨迹示意，初速度30m/s，角度35度，显示关键点',
];

function buildInitialProps(model: PhysicsModel): Record<string, number> {
  const merged: Record<string, number> = { ...(model.scene_spec?.default_props || {}) };
  model.parameters?.forEach((item) => {
    if (merged[item.name] === undefined) merged[item.name] = item.default;
  });
  return merged;
}

function PhysicsPageInner() {
  const searchParams = useSearchParams();
  const abortRef = useRef<AbortController | null>(null);
  const eventSeqRef = useRef(0);
  const [question, setQuestion] = useState(searchParams.get('q') || '');
  const [context, setContext] = useState('');
  const [analysis, setAnalysis] = useState<PhysicsModel | null>(null);
  const [artifact, setArtifact] = useState<RenderArtifact | null>(null);
  const [previewProps, setPreviewProps] = useState<Record<string, number>>({});
  const [loading, setLoading] = useState(false);
  const [stage, setStage] = useState<StreamStage>('idle');
  const [streamError, setStreamError] = useState<string | null>(null);
  const [previewStatus, setPreviewStatus] = useState<'idle' | 'loading' | 'ready' | 'updated' | 'error' | 'timeout'>('idle');
  const [previewError, setPreviewError] = useState<string | null>(null);
  const [streamingText, setStreamingText] = useState('');
  const [toolCalls, setToolCalls] = useState<{ tool: string; status: string }[]>([]);
  const [streamLog, setStreamLog] = useState<StreamLogItem[]>([]);
  const [startedAt, setStartedAt] = useState<number | null>(null);
  const [finishedAt, setFinishedAt] = useState<number | null>(null);

  useEffect(() => () => abortRef.current?.abort(), []);

  const appendStreamLog = useCallback((event: string, label: string, status?: StreamLogItem['status']) => {
    const now = new Date();
    const nextItem: StreamLogItem = {
      id: eventSeqRef.current += 1,
      event,
      label,
      status,
      time: now.toLocaleTimeString('zh-CN', { hour12: false }),
    };
    setStreamLog((prev) => [...prev.slice(-11), nextItem]);
  }, []);

  const handlePreviewStatusChange = useCallback((status: 'loading' | 'ready' | 'updated' | 'error' | 'timeout', detail?: string) => {
    setPreviewStatus(status);
    setPreviewError(status === 'error' || status === 'timeout' ? detail || '预览运行失败' : null);
    if (status === 'ready' || status === 'updated') {
      setStage((prev) => (prev === 'previewing' ? 'done' : prev));
    }
    appendStreamLog('iframe', `预览 ${status}${detail ? `：${detail}` : ''}`, status === 'error' ? 'error' : 'success');
  }, [appendStreamLog]);

  const cancelStream = () => {
    abortRef.current?.abort();
    abortRef.current = null;
    setLoading(false);
    setStage('aborted');
    appendStreamLog('abort', '用户取消本次 SSE 连接', 'error');
  };

  useEffect(() => {
    const q = searchParams.get('q');
    if (q) {
      setQuestion(q);
      void handleAnalyze(q);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchParams]);

  const handleAnalyze = async (nextQuestion?: string) => {
    const text = (nextQuestion || question).trim();
    if (!text) return;

    abortRef.current?.abort();
    const controller = new AbortController();
    abortRef.current = controller;
    eventSeqRef.current = 0;

    setLoading(true);
    setStartedAt(Date.now());
    setFinishedAt(null);
    setStage('connecting');
    setStreamError(null);
    setPreviewStatus('idle');
    setPreviewError(null);
    setStreamingText('');
    setToolCalls([]);
    setStreamLog([]);
    setAnalysis(null);
    setArtifact(null);
    setPreviewProps({});
    appendStreamLog('connect', '准备连接 Agent SSE', 'processing');

    const payload: ChatReq = {
      message: text,
      mode: 'physics',
      filters: context ? { subject: 'physics' } : undefined,
    };

    try {
      await agentChatStream(payload, (event) => {
        appendStreamLog(event.event, `收到 ${event.event} 事件`, event.event === 'done' ? 'success' : 'processing');

        if (event.event === 'thinking') {
          setStage('thinking');
          return;
        }

        if (event.event === 'heartbeat') {
          const data = event.data as { tool?: string; status?: string } | undefined;
          if (data?.tool === 'RenderCodeTool') setStage('generating');
          else if (data?.tool === 'PhysicsAnalyzeTool') setStage('analyzing');
          else setStage((prev) => (prev === 'idle' || prev === 'connecting' ? 'thinking' : prev));
          return;
        }

        if (event.event === 'content') {
          setStage('thinking');
          const data = event.data as { content?: string } | string;
          const content = typeof data === 'string' ? data : data?.content || '';
          if (content) setStreamingText((prev) => prev + content);
          return;
        }

        if (event.event === 'tool_call') {
          const call = event.data as { tool?: string; status?: string };
          if (!call?.tool) return;
          const toolName = call.tool;
          const status = call.status || 'running';
          if (toolName === 'PhysicsAnalyzeTool') setStage(status === 'success' ? 'validating' : 'analyzing');
          if (toolName === 'RenderCodeTool') setStage(status === 'success' ? 'previewing' : 'generating');
          setToolCalls((prev) => {
            const rest = prev.filter((item) => item.tool !== toolName);
            return [...rest, { tool: toolName, status }];
          });
          return;
        }

        if (event.event === 'render_code') {
          setStage('previewing');
          const renderArtifact = event.data as RenderArtifact;
          setArtifact(renderArtifact);
          setPreviewStatus('loading');
          return;
        }

        if (event.event === 'preview') {
          setStage('previewing');
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
          setStage('done');
          setFinishedAt(Date.now());
          const data = event.data as { structured_payload?: AgentPhysicsPayload; answer?: string };
          const payloadData = data?.structured_payload;
          if (payloadData?.analysis) {
            setAnalysis(payloadData.analysis);
            setPreviewProps(buildInitialProps(payloadData.analysis));
          }
          if (payloadData?.render_artifact) {
            setArtifact(payloadData.render_artifact);
          }
          if (data?.answer) {
            setStreamingText(data.answer);
          }
        }
      }, {
        signal: controller.signal,
        onOpen: () => {
          setStage('thinking');
          appendStreamLog('open', 'SSE 连接已建立', 'success');
        },
        onDone: () => {
          appendStreamLog('close', 'SSE 连接正常结束', 'success');
        },
        onError: (error) => {
          if (error.name === 'AbortError') return;
          setStage('error');
          setStreamError(error.message);
          appendStreamLog('error', error.message, 'error');
        },
      });
    } catch (error) {
      if (error instanceof Error && error.name === 'AbortError') return;
      const messageText = error instanceof Error ? error.message : '分析失败';
      setStage('error');
      setFinishedAt(Date.now());
      setStreamError(messageText);
      message.error(messageText);
    } finally {
      if (abortRef.current === controller) {
        abortRef.current = null;
      }
      setLoading(false);
    }
  };

  const handleFavorite = async () => {
    try {
      await api.addFavorite({ target_type: 'physics', target_id: question, title: question });
      message.success('收藏成功');
    } catch (error) {
      message.error(error instanceof Error ? error.message : '收藏失败');
    }
  };

  const currentWarnings = useMemo(() => ([
    ...(analysis?.warnings || []),
    ...(artifact?.warnings || []),
  ]), [analysis, artifact]);

  const isForce3DArtifact = artifact?.scene_type === 'physics_force_3d';
  const currentViewDimension = (previewProps.view_dimension ?? artifact?.render_manifest.initial_props?.view_dimension ?? 3) >= 2.5 ? '3d' : '2d';

  const elapsedText = useMemo(() => {
    if (!startedAt) return '0.0s';
    const end = finishedAt || Date.now();
    return `${Math.max(0, (end - startedAt) / 1000).toFixed(1)}s`;
  }, [finishedAt, startedAt]);

  return (
    <div>
      <Title level={3}><ExperimentOutlined /> 物理 / 3D 场景代码生成</Title>

      <Card style={{ marginBottom: 16 }}>
        <Space direction="vertical" style={{ width: '100%' }}>
          <TextArea
            placeholder="输入物理题目、3D 场景描述或建模目标，如：平抛运动的空间轨迹如何随初速度变化"
            rows={3}
            value={question}
            onChange={(e) => setQuestion(e.target.value)}
          />
          <TextArea
            placeholder="补充上下文（可选）"
            rows={2}
            value={context}
            onChange={(e) => setContext(e.target.value)}
          />
          <Space wrap>
            <Button type="primary" onClick={() => void handleAnalyze()} loading={loading} size="large">
              通过 Agent 流式分析并生成预览
            </Button>
            <Button onClick={cancelStream} disabled={!loading} icon={<StopOutlined />}>
              取消
            </Button>
            <Button onClick={handleFavorite} icon={<StarOutlined />}>
              收藏当前问题
            </Button>
          </Space>
          <Space wrap size={[8, 8]}>
            <Text type="secondary">示例：</Text>
            {exampleQuestions.map((example) => (
              <Button
                key={example}
                size="small"
                icon={<PlayCircleOutlined />}
                onClick={() => {
                  setQuestion(example);
                  void handleAnalyze(example);
                }}
                disabled={loading}
              >
                {example.length > 24 ? `${example.slice(0, 24)}...` : example}
              </Button>
            ))}
          </Space>
        </Space>
      </Card>

      {(loading || streamingText || toolCalls.length > 0 || streamLog.length > 0 || streamError) && (
        <Card title="Agent 流式输出" style={{ marginBottom: 16 }}>
          <Space direction="vertical" style={{ width: '100%' }}>
            <Space wrap>
              <Tag color={stageColor[stage]}>阶段：{stageText[stage]}</Tag>
              <Tag color={previewStatus === 'error' ? 'red' : previewStatus === 'ready' || previewStatus === 'updated' ? 'green' : 'default'}>
                预览：{previewStatus}
              </Tag>
              <Tag>耗时：{elapsedText}</Tag>
            </Space>

            <Progress
              percent={stagePercent[stage]}
              status={stage === 'error' ? 'exception' : stage === 'done' ? 'success' : 'active'}
              showInfo={false}
            />

            {streamError && (
              <Alert
                type="error"
                showIcon
                message="SSE 流式请求失败"
                description={streamError}
              />
            )}

            {previewError && (
              <Alert
                type={previewStatus === 'timeout' ? 'warning' : 'error'}
                showIcon
                message={previewStatus === 'timeout' ? '浏览器预览未确认 ready' : '浏览器预览失败'}
                description={previewError}
              />
            )}

            <div>
              <Text strong>工具调用：</Text>
              <Space wrap style={{ marginLeft: 8 }}>
                {toolCalls.map((call) => (
                  <Tag key={call.tool} color={call.status === 'success' ? 'green' : call.status === 'failed' ? 'red' : 'blue'}>
                    {call.tool}: {call.status}
                  </Tag>
                ))}
              </Space>
            </div>
            {loading && <Spin size="small" tip="正在接收 SSE 事件..." />}
            {streamingText && <MarkdownText content={streamingText} />}
            {streamLog.length > 0 && (
              <Timeline
                style={{ marginTop: 8 }}
                items={streamLog.map((item) => ({
                  color: item.status === 'error' ? 'red' : item.status === 'success' ? 'green' : 'blue',
                  children: (
                    <Space wrap size={8}>
                      <Text type="secondary">{item.time}</Text>
                      <Tag>{item.event}</Tag>
                      <Text>{item.label}</Text>
                    </Space>
                  ),
                }))}
              />
            )}
          </Space>
        </Card>
      )}

      {(analysis || artifact) && (
        <Row gutter={16} align="top">
          <Col xs={24} xl={11}>
            {analysis && artifact && (
              <Card size="small" title="生成结果" style={{ marginBottom: 16 }}>
                <Space wrap>
                  <Tag color="green">{artifact.scene_type}</Tag>
                  <Tag color="blue">{artifact.render_mode}</Tag>
                  <Tag>{Object.keys(artifact.code_bundle).length} 个文件</Tag>
                  <Tag>耗时 {elapsedText}</Tag>
                </Space>
              </Card>
            )}

            {analysis ? (
              <>
                <Card
                  title={
                    <Space wrap>
                      <span>模型：<Tag color="blue">{analysis.model_type}</Tag></span>
                      {analysis.scene_spec && <Tag color="purple">{analysis.scene_spec.scene_type}</Tag>}
                    </Space>
                  }
                  extra={<Tag color="green">Agent 驱动</Tag>}
                  style={{ marginBottom: 16 }}
                >
                  <Paragraph style={{ marginBottom: 12 }}>
                    {analysis.scene_spec?.summary || analysis.result_summary}
                  </Paragraph>
                  <Title level={5}>已知条件</Title>
                  <Table
                    dataSource={analysis.conditions.map((item, index) => ({ ...item, key: index }))}
                    columns={[
                      { title: '名称', dataIndex: 'name' },
                      { title: '值', dataIndex: 'value' },
                      { title: '单位', dataIndex: 'unit' },
                    ]}
                    pagination={false}
                    size="small"
                    locale={{ emptyText: '未抽取到明确条件，将使用默认参数模板' }}
                  />
                </Card>

                <Card title="推导与生成步骤" style={{ marginBottom: 16 }}>
                  <Steps
                    direction="vertical"
                    current={analysis.steps.length}
                    items={analysis.steps.map((step) => ({
                      title: <Text strong>{step.title}</Text>,
                      description: <Paragraph style={{ whiteSpace: 'pre-wrap', marginBottom: 0 }}>{step.content}</Paragraph>,
                    }))}
                  />
                </Card>
              </>
            ) : (
              <Card style={{ marginBottom: 16 }}>
                <Empty description="已收到渲染代码，等待 Agent 汇总分析结果..." />
              </Card>
            )}

            {analysis?.parameters && analysis.parameters.length > 0 && (
              <Card title="参数调节（Agent 生成的预览本地刷新）" style={{ marginBottom: 16 }}>
                <Space direction="vertical" style={{ width: '100%' }}>
                  {isForce3DArtifact && (
                    <div>
                      <Text>视图模式</Text>
                      <div style={{ marginTop: 8 }}>
                        <Segmented
                          value={currentViewDimension}
                          options={[
                            { label: '3D WebGL', value: '3d' },
                            { label: '2D fallback', value: '2d' },
                          ]}
                          onChange={(value) => {
                            setPreviewProps((prev) => ({ ...prev, view_dimension: value === '3d' ? 3 : 2 }));
                          }}
                        />
                      </div>
                      <Text type="secondary">切换后通过 postMessage 同步到 iframe，不重新请求后端。</Text>
                    </div>
                  )}
                  {analysis.parameters.map((parameter) => (
                    <div key={parameter.name}>
                      <Text>{parameter.label}（{parameter.unit}）</Text>
                      <Slider
                        min={parameter.min}
                        max={parameter.max}
                        step={parameter.step}
                        value={previewProps[parameter.name] ?? parameter.default}
                        onChange={(value) => {
                          setPreviewProps((prev) => ({ ...prev, [parameter.name]: value }));
                        }}
                      />
                      <Text type="secondary">
                        当前值：{previewProps[parameter.name] ?? parameter.default} {parameter.unit}
                      </Text>
                    </div>
                  ))}
                </Space>
              </Card>
            )}

            {currentWarnings.length > 0 && (
              <Alert
                type="warning"
                showIcon
                message="注意事项"
                description={currentWarnings.join('；')}
                style={{ marginBottom: 16 }}
              />
            )}
          </Col>

          <Col xs={24} xl={13}>
            <Card title="浏览器渲染预览" extra={artifact && <Tag color="green">{artifact.render_manifest.sandbox}</Tag>} style={{ marginBottom: 16 }}>
              {artifact ? (
                <RenderPreviewSandbox
                  artifact={artifact}
                  propsData={previewProps}
                  onStatusChange={handlePreviewStatusChange}
                />
              ) : (
                <Empty description="等待 Agent 下发 render_code 事件..." />
              )}
            </Card>

            {artifact && (
              <Card title="Render Manifest" size="small" style={{ marginBottom: 16 }}>
                <Space wrap>
                  <Tag color="blue">entry: {artifact.render_manifest.entry}</Tag>
                  <Tag color="purple">framework: {artifact.render_manifest.framework}</Tag>
                  <Tag color="cyan">render_mode: {artifact.render_manifest.render_mode}</Tag>
                </Space>
                <Paragraph type="secondary" style={{ marginTop: 12, marginBottom: 8 }}>
                  允许 API：{(artifact.render_manifest.allowed_apis || []).join('、')}
                </Paragraph>
                <Paragraph type="secondary" style={{ marginBottom: 0 }}>
                  屏蔽 API：{(artifact.render_manifest.blocked_apis || []).join('、')}
                </Paragraph>
              </Card>
            )}

            {artifact && (
              <Card title="生成的前端代码包">
                <CodeBundleViewer artifact={artifact} />
              </Card>
            )}
          </Col>
        </Row>
      )}

      {!loading && !analysis && !streamingText && (
        <Empty description="输入题目后开始通过 Agent 流式生成浏览器预览代码" style={{ paddingTop: 60 }}>
          <Space wrap>
            {exampleQuestions.slice(0, 2).map((example) => (
              <Button key={example} onClick={() => { setQuestion(example); void handleAnalyze(example); }}>
                试试：{example.slice(0, 12)}...
              </Button>
            ))}
          </Space>
        </Empty>
      )}
    </div>
  );
}

export default function PhysicsPage() {
  return (
    <Suspense fallback={<Spin />}>
      <PhysicsPageInner />
    </Suspense>
  );
}
