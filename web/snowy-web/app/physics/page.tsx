'use client';

import React, { useState, useEffect, Suspense } from 'react';
import { useSearchParams } from 'next/navigation';
import { Input, Card, Typography, Space, Spin, Tag, Steps, Slider, Row, Col, Table, Empty, Button, Alert, message } from 'antd';
import { ExperimentOutlined, StarOutlined } from '@ant-design/icons';
import { api, type PhysicsModel, type ComputeResult } from '@/lib/api';
import { useAuthStore } from '@/stores/auth';
import PhysicsChart from '@/components/physics/PhysicsChart';

const { Title, Paragraph, Text } = Typography;
const { TextArea } = Input;

function PhysicsPageInner() {
  const searchParams = useSearchParams();
  const { isLoggedIn } = useAuthStore();
  const [question, setQuestion] = useState(searchParams.get('q') || '');
  const [context, setContext] = useState('');
  const [result, setResult] = useState<PhysicsModel | null>(null);
  const [simulateResult, setSimulateResult] = useState<ComputeResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [simulating, setSimulating] = useState(false);
  const [params, setParams] = useState<Record<string, number>>({});

  useEffect(() => {
    const q = searchParams.get('q');
    if (q) {
      setQuestion(q);
      handleAnalyze(q);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchParams]);

  const handleAnalyze = async (q?: string) => {
    const text = q || question;
    if (!text.trim()) return;
    setLoading(true);
    setSimulateResult(null);
    try {
      const res = await api.physicsAnalyze({ question: text, context: context || undefined });
      if (res.data) {
        setResult(res.data);
        // Initialize parameters
        if (res.data.parameters) {
          const initial: Record<string, number> = {};
          res.data.parameters.forEach((p) => { initial[p.name] = p.default; });
          setParams(initial);
        }
      }
    } catch {
      message.error('物理解析失败');
    } finally {
      setLoading(false);
    }
  };

  const handleSimulate = async () => {
    if (!result) return;
    setSimulating(true);
    try {
      const res = await api.physicsSimulate({
        model_type: result.model_type,
        parameters: params,
      });
      if (res.data) setSimulateResult(res.data);
    } catch {
      message.error('调参计算失败');
    } finally {
      setSimulating(false);
    }
  };

  const handleFavorite = async () => {
    if (!isLoggedIn) { message.warning('请先登录'); return; }
    try {
      await api.addFavorite({ target_type: 'physics', target_id: question, title: question });
      message.success('收藏成功');
    } catch {
      message.error('收藏失败');
    }
  };

  const chartSpec = simulateResult?.chart || result?.chart;

  return (
    <div>
      <Title level={3}><ExperimentOutlined /> 物理建模</Title>

      {/* Input Area */}
      <Card style={{ marginBottom: 16 }}>
        <Space direction="vertical" style={{ width: '100%' }}>
          <TextArea
            placeholder="输入物理题目或问题，如：一个质量为2kg的物体从10m高处自由落下..."
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

      {loading && <div style={{ textAlign: 'center', padding: 40 }}><Spin size="large" tip="正在解析..." /></div>}

      {!loading && result && (
        <Row gutter={16}>
          <Col xs={24} lg={14}>
            {/* Conditions */}
            <Card
              title={<>模型: <Tag color="blue">{result.model_type}</Tag></>}
              extra={<Button icon={<StarOutlined />} size="small" onClick={handleFavorite}>收藏</Button>}
              style={{ marginBottom: 16 }}
            >
              <Title level={5}>已知条件</Title>
              <Table
                dataSource={result.conditions.map((c, i) => ({ ...c, key: i }))}
                columns={[
                  { title: '名称', dataIndex: 'name' },
                  { title: '值', dataIndex: 'value' },
                  { title: '单位', dataIndex: 'unit' },
                ]}
                pagination={false}
                size="small"
              />
            </Card>

            {/* Derivation Steps */}
            <Card title="推导过程" style={{ marginBottom: 16 }}>
              <Steps
                direction="vertical"
                current={result.steps.length}
                items={result.steps.map((step) => ({
                  title: <Text strong>{step.title}</Text>,
                  description: <Paragraph style={{ whiteSpace: 'pre-wrap' }}>{step.content}</Paragraph>,
                }))}
              />
            </Card>

            {/* Result Summary */}
            <Card title="结果总结" style={{ marginBottom: 16 }}>
              <Paragraph style={{ fontSize: 15 }}>{result.result_summary}</Paragraph>
              {result.warnings && result.warnings.length > 0 && (
                <Alert
                  type="warning"
                  message="注意事项"
                  description={result.warnings.join('；')}
                  showIcon
                  style={{ marginTop: 8 }}
                />
              )}
            </Card>
          </Col>

          <Col xs={24} lg={10}>
            {/* Chart */}
            {chartSpec && (
              <Card title={chartSpec.title} style={{ marginBottom: 16 }}>
                <PhysicsChart spec={chartSpec} />
              </Card>
            )}

            {/* Parameter Sliders */}
            {result.parameters && result.parameters.length > 0 && (
              <Card title="参数调节" style={{ marginBottom: 16 }}>
                <Space direction="vertical" style={{ width: '100%' }}>
                  {result.parameters.map((p) => (
                    <div key={p.name}>
                      <Text>{p.label} ({p.unit})</Text>
                      <Slider
                        min={p.min}
                        max={p.max}
                        step={p.step}
                        value={params[p.name] ?? p.default}
                        onChange={(v) => setParams((prev) => ({ ...prev, [p.name]: v }))}
                      />
                      <Text type="secondary">{params[p.name] ?? p.default} {p.unit}</Text>
                    </div>
                  ))}
                  <Button type="primary" onClick={handleSimulate} loading={simulating} block>
                    重新计算
                  </Button>
                </Space>
              </Card>
            )}

            {/* Compute Result Values */}
            {simulateResult && simulateResult.values && (
              <Card title="计算结果" size="small">
                {Object.entries(simulateResult.values).map(([key, val]) => (
                  <div key={key}><Text strong>{key}</Text>: {val}</div>
                ))}
                {simulateResult.warnings && simulateResult.warnings.length > 0 && (
                  <Alert type="warning" message={simulateResult.warnings.join('；')} style={{ marginTop: 8 }} />
                )}
              </Card>
            )}
          </Col>
        </Row>
      )}

      {!loading && !result && <Empty description="输入物理问题开始建模" style={{ paddingTop: 60 }} />}
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
