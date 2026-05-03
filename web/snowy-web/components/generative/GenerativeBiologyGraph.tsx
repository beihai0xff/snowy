'use client';

import React, { useMemo } from 'react';
import { Alert, Card, Col, Empty, Row, Space, Steps, Tag, Typography } from 'antd';
import { Background, Controls, ReactFlow, type Edge, type Node } from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import type { GenerativeVisualizationSpec } from '@/lib/api';

const { Paragraph, Text } = Typography;

const colorMap: Record<string, string> = {
  factor: '#1677ff',
  result: '#52c41a',
  process: '#fa8c16',
  structure: '#722ed1',
  substance: '#13c2c2',
};

export default function GenerativeBiologyGraph({ spec }: { spec?: GenerativeVisualizationSpec }) {
  const flow = useMemo(() => {
    const nodes: Node[] = (spec?.nodes || []).map((node, index) => ({
      id: node.id,
      data: { label: node.label },
      position: { x: 190 * (index % 3), y: 120 * Math.floor(index / 3) },
      style: { background: colorMap[node.type] || '#1677ff', color: '#fff', borderRadius: 10, border: 0, padding: '8px 14px' },
    }));
    const edges: Edge[] = (spec?.edges || []).map((edge, index) => ({
      id: `e-${index}`,
      source: edge.source,
      target: edge.target,
      label: edge.relation,
      animated: true,
    }));
    return { nodes, edges };
  }, [spec]);

  if (!spec) return <Empty description="暂无生物可视化结构" />;

  return (
    <Space direction="vertical" style={{ width: '100%' }} size="middle">
      <Space wrap>
        <Tag color="purple">{spec.visualization_type}</Tag>
        <Tag color="green">{spec.topic}</Tag>
        {(spec.limiting_factors || []).map((item) => <Tag key={item}>{item}</Tag>)}
      </Space>
      <div style={{ height: 420, border: '1px solid #e5e7eb', borderRadius: 12, overflow: 'hidden' }}>
        {flow.nodes.length > 0 ? (
          <ReactFlow nodes={flow.nodes} edges={flow.edges} fitView proOptions={{ hideAttribution: true }}>
            <Background />
            <Controls />
          </ReactFlow>
        ) : <Empty description="暂无概念节点" style={{ paddingTop: 120 }} />}
      </div>
      {spec.curve_explanation && <Alert type="info" showIcon message="曲线/限制因素解释" description={spec.curve_explanation} />}
      <Row gutter={[12, 12]}>
        {spec.experiment_variables && (
          <Col xs={24} md={10}>
            <Card size="small" title="实验变量">
              <Paragraph><Text strong>自变量：</Text>{spec.experiment_variables.independent?.join('、') || '-'}</Paragraph>
              <Paragraph><Text strong>因变量：</Text>{spec.experiment_variables.dependent?.join('、') || '-'}</Paragraph>
              <Paragraph style={{ marginBottom: 0 }}><Text strong>控制变量：</Text>{spec.experiment_variables.controlled?.join('、') || '-'}</Paragraph>
            </Card>
          </Col>
        )}
        {(spec.process_steps || []).length > 0 && (
          <Col xs={24} md={14}>
            <Card size="small" title="过程阶段">
              <Steps
                size="small"
                direction="vertical"
                current={(spec.process_steps || []).length}
                items={(spec.process_steps || []).slice(0, 5).map((step) => ({ title: step.title, description: step.detail || [...(step.input || []), ...(step.output || [])].join(' → ') }))}
              />
            </Card>
          </Col>
        )}
      </Row>
    </Space>
  );
}
