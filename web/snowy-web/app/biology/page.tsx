'use client';

import React, { useState, useEffect, useCallback, Suspense } from 'react';
import { useSearchParams } from 'next/navigation';
import { Input, Card, Typography, Space, Spin, Tag, Steps, Row, Col, Empty, Button, List, Alert, message } from 'antd';
import { BranchesOutlined, StarOutlined, ReloadOutlined, ExperimentOutlined } from '@ant-design/icons';
import { api, type BiologyModel, type RenderArtifact } from '@/lib/api';
import BiologyDiagram from '@/components/biology/BiologyDiagram';
import RenderPreviewSandbox from '@/components/common/RenderPreviewSandbox';

const { Title, Paragraph, Text } = Typography;
const { TextArea } = Input;

const biologyExamples = [
  '光照强度对光合作用有机物积累的影响',
  '细胞膜的结构如何决定选择透过性？',
  '神经冲动在突触处如何传递？',
];

function BiologyPageInner() {
  const searchParams = useSearchParams();
  const [question, setQuestion] = useState(searchParams.get('q') || '');
  const [context, setContext] = useState('');
  const [result, setResult] = useState<BiologyModel | null>(null);
  const [artifact, setArtifact] = useState<RenderArtifact | null>(null);
  const [previewLoading, setPreviewLoading] = useState(false);
  const [previewError, setPreviewError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [errorText, setErrorText] = useState<string | null>(null);

  const handleAnalyze = useCallback(async (q?: string) => {
    const text = (q || question).trim();
    if (!text) return;

    setLoading(true);
    setErrorText(null);
    try {
      const res = await api.biologyAnalyze({ question: text, context: context || undefined });
      const model = res.data ?? null;
      setResult(model);
      setArtifact(null);
      setPreviewError(null);
      if (model?.scene_spec) {
        setPreviewLoading(true);
        try {
          const renderRes = await api.renderGenerate({ scene_spec: model.scene_spec, render_mode: model.scene_spec.render_mode || 'html_iframe' });
          setArtifact(renderRes.data ?? null);
        } catch (renderError) {
          setPreviewError(renderError instanceof Error ? renderError.message : '生物演示页生成失败');
        } finally {
          setPreviewLoading(false);
        }
      }
    } catch (error) {
      const messageText = error instanceof Error ? error.message : '生物解析失败';
      setErrorText(messageText);
      setResult(null);
      setArtifact(null);
      setPreviewError(null);
      message.error(messageText);
    } finally {
      setLoading(false);
    }
  }, [question, context]);

  useEffect(() => {
    const q = searchParams.get('q');
    if (q) {
      setQuestion(q);
      handleAnalyze(q);
    }
  }, [searchParams, handleAnalyze]);

  const handleFavorite = async () => {
    try {
      await api.addFavorite({ target_type: 'biology', target_id: question, title: question });
      message.success('收藏成功');
    } catch (error) {
      message.error(error instanceof Error ? error.message : '收藏失败');
    }
  };

  const renderEmptyActions = () => (
    <Space direction="vertical" align="center" size="middle">
      <Text type="secondary">输入生物问题生成概念图、过程拆解和实验变量分析。</Text>
      <Space wrap>
        {biologyExamples.map((item) => (
          <Button
            key={item}
            onClick={() => {
              setQuestion(item);
              handleAnalyze(item);
            }}
          >
            试试：{item}
          </Button>
        ))}
      </Space>
    </Space>
  );

  return (
    <div>
      <Title level={3}><BranchesOutlined /> 生物建模</Title>

      {/* Input Area */}
      <Card style={{ marginBottom: 16 }}>
        <Space direction="vertical" style={{ width: '100%' }}>
          <TextArea
            placeholder="输入生物问题或知识点，如：光合作用中光照强度对有机物积累的影响..."
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
          <Button type="primary" onClick={() => handleAnalyze()} loading={loading} size="large">
            开始分析
          </Button>
        </Space>
      </Card>

      {errorText && !loading && (
        <Alert
          type="error"
          showIcon
          message="生物建模请求失败"
          description={errorText}
          action={(
            <Button size="small" icon={<ReloadOutlined />} onClick={() => handleAnalyze()}>
              重试
            </Button>
          )}
          style={{ marginBottom: 16 }}
        />
      )}

      {loading && <div style={{ textAlign: 'center', padding: 40 }}><Spin size="large" tip="正在解析并生成动态演示..." /></div>}

      {!loading && result && (
        <>
          <Card
            title={<><ExperimentOutlined /> 生物动态可视化演示</>}
            extra={result.scene_spec && <Tag color="purple">{result.scene_spec.scene_type}</Tag>}
            style={{ marginBottom: 16 }}
          >
            {previewLoading && <div style={{ textAlign: 'center', padding: 32 }}><Spin tip="正在生成炫酷演示页..." /></div>}
            {!previewLoading && artifact && (
              <RenderPreviewSandbox artifact={artifact} propsData={result.scene_spec?.default_props || {}} />
            )}
            {!previewLoading && !artifact && previewError && (
              <Alert
                type="warning"
                showIcon
                message="动态演示页生成失败，已回退到概念图谱"
                description={previewError}
                action={result.scene_spec && (
                  <Button
                    size="small"
                    icon={<ReloadOutlined />}
                    onClick={() => void handleAnalyze()}
                  >
                    重试
                  </Button>
                )}
              />
            )}
            {!previewLoading && !artifact && !previewError && (
              <Empty description="当前结果没有可视化 scene_spec，将展示结构化概念图谱。" />
            )}
          </Card>

          <Row gutter={16}>
            <Col xs={24} lg={14}>
            {/* Topic & Concepts */}
            <Card
              title={<>主题: <Tag color="purple">{result.topic}</Tag></>}
              extra={<Button icon={<StarOutlined />} size="small" onClick={handleFavorite}>收藏</Button>}
              style={{ marginBottom: 16 }}
            >
              <Title level={5}>核心概念</Title>
              {result.concepts.length > 0 ? (
                <Space wrap>
                  {result.concepts.map((c, i) => (
                    <Tag key={i} color={c.type === 'factor' ? 'blue' : c.type === 'result' ? 'green' : 'default'}>
                      {c.name} ({c.type})
                    </Tag>
                  ))}
                </Space>
              ) : (
                <Alert type="info" showIcon message="未识别到明确概念，已保留总结和图谱兜底。" />
              )}

              {/* Relations */}
              {result.relations.length > 0 && (
                <>
                  <Title level={5} style={{ marginTop: 16 }}>概念关系</Title>
                  <List
                    size="small"
                    dataSource={result.relations}
                    renderItem={(r, i) => (
                      <List.Item key={i}>
                        <Text>{r.source}</Text>
                        <Tag color="orange" style={{ margin: '0 8px' }}>{r.type}</Tag>
                        <Text>{r.target}</Text>
                      </List.Item>
                    )}
                  />
                </>
              )}
            </Card>

            {/* Process Steps */}
            {result.process_steps.length > 0 && (
              <Card title="过程拆解" style={{ marginBottom: 16 }}>
                <Steps
                  direction="vertical"
                  current={result.process_steps.length}
                  items={result.process_steps.map((step) => ({
                    title: <Text strong>{step.title}</Text>,
                    description: <Paragraph style={{ whiteSpace: 'pre-wrap' }}>{step.content}</Paragraph>,
                  }))}
                />
              </Card>
            )}

            {/* Experiment Variables */}
            {result.experiment_variables && (
              <Card title="实验变量分析" style={{ marginBottom: 16 }}>
                <Space direction="vertical" style={{ width: '100%' }}>
                  <div>
                    <Text strong>自变量: </Text>
                    {result.experiment_variables.independent.length > 0
                      ? result.experiment_variables.independent.map((v, i) => <Tag key={i} color="blue">{v}</Tag>)
                      : <Text type="secondary">未识别</Text>}
                  </div>
                  <div>
                    <Text strong>因变量: </Text>
                    {result.experiment_variables.dependent.length > 0
                      ? result.experiment_variables.dependent.map((v, i) => <Tag key={i} color="green">{v}</Tag>)
                      : <Text type="secondary">未识别</Text>}
                  </div>
                  <div>
                    <Text strong>控制变量: </Text>
                    {result.experiment_variables.controlled.length > 0
                      ? result.experiment_variables.controlled.map((v, i) => <Tag key={i}>{v}</Tag>)
                      : <Text type="secondary">未识别</Text>}
                  </div>
                </Space>
              </Card>
            )}

            {/* Result Summary */}
            <Card title="总结" style={{ marginBottom: 16 }}>
              <Paragraph style={{ fontSize: 15 }}>{result.result_summary}</Paragraph>
            </Card>
          </Col>

            <Col xs={24} lg={10}>
              {/* Diagram */}
            {result.diagram ? (
              <Card title={result.diagram.title} style={{ marginBottom: 16 }}>
                <Alert
                  message={`图表类型: ${result.diagram.diagram_type}`}
                  type="info"
                  showIcon
                  style={{ marginBottom: 12 }}
                />
                <BiologyDiagram spec={result.diagram} />
              </Card>
            ) : (
              <Card title="概念图谱" style={{ marginBottom: 16 }}>
                <Empty description="本次未生成图谱，可补充更具体的概念或过程后重试。" />
              </Card>
            )}
            </Col>
          </Row>
        </>
      )}

      {!loading && !result && !errorText && <Empty description={renderEmptyActions()} style={{ paddingTop: 60 }} />}
    </div>
  );
}

export default function BiologyPage() {
  return (
    <Suspense fallback={<Spin />}>
      <BiologyPageInner />
    </Suspense>
  );
}
