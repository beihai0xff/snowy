'use client';

import React, { useState, useEffect, useCallback, Suspense } from 'react';
import { useSearchParams } from 'next/navigation';
import { Input, Card, Typography, Space, Spin, Tag, Steps, Row, Col, Empty, Button, List, Alert, message } from 'antd';
import { BranchesOutlined, StarOutlined } from '@ant-design/icons';
import { api, type BiologyModel } from '@/lib/api';
import BiologyDiagram from '@/components/biology/BiologyDiagram';

const { Title, Paragraph, Text } = Typography;
const { TextArea } = Input;

function BiologyPageInner() {
  const searchParams = useSearchParams();
  const [question, setQuestion] = useState(searchParams.get('q') || '');
  const [context, setContext] = useState('');
  const [result, setResult] = useState<BiologyModel | null>(null);
  const [loading, setLoading] = useState(false);

  const handleAnalyze = useCallback(async (q?: string) => {
    const text = q || question;
    if (!text.trim()) return;
    setLoading(true);
    try {
      const res = await api.biologyAnalyze({ question: text, context: context || undefined });
      if (res.data) setResult(res.data);
    } catch {
      message.error('生物解析失败');
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
    } catch {
      message.error('收藏失败');
    }
  };

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

      {loading && <div style={{ textAlign: 'center', padding: 40 }}><Spin size="large" tip="正在解析..." /></div>}

      {!loading && result && (
        <Row gutter={16}>
          <Col xs={24} lg={14}>
            {/* Topic & Concepts */}
            <Card
              title={<>主题: <Tag color="purple">{result.topic}</Tag></>}
              extra={<Button icon={<StarOutlined />} size="small" onClick={handleFavorite}>收藏</Button>}
              style={{ marginBottom: 16 }}
            >
              <Title level={5}>核心概念</Title>
              <Space wrap>
                {result.concepts.map((c, i) => (
                  <Tag key={i} color={c.type === 'factor' ? 'blue' : c.type === 'result' ? 'green' : 'default'}>
                    {c.name} ({c.type})
                  </Tag>
                ))}
              </Space>

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
                    {result.experiment_variables.independent.map((v, i) => <Tag key={i} color="blue">{v}</Tag>)}
                  </div>
                  <div>
                    <Text strong>因变量: </Text>
                    {result.experiment_variables.dependent.map((v, i) => <Tag key={i} color="green">{v}</Tag>)}
                  </div>
                  <div>
                    <Text strong>控制变量: </Text>
                    {result.experiment_variables.controlled.map((v, i) => <Tag key={i}>{v}</Tag>)}
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
            {result.diagram && (
              <Card title={result.diagram.title} style={{ marginBottom: 16 }}>
                <Alert
                  message={`图表类型: ${result.diagram.diagram_type}`}
                  type="info"
                  showIcon
                  style={{ marginBottom: 12 }}
                />
                <BiologyDiagram spec={result.diagram} />
              </Card>
            )}
          </Col>
        </Row>
      )}

      {!loading && !result && <Empty description="输入生物问题开始建模" style={{ paddingTop: 60 }} />}
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
