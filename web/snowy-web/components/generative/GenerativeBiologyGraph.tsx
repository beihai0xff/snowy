'use client';

import React, { useMemo } from 'react';
import { Alert, Card, Empty, List, Space, Steps, Tabs, Tag, Timeline, Typography } from 'antd';
import { Background, Controls, ReactFlow, type Edge, type Node } from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import type { GenerativeVisualizationSpec } from '@/lib/api';
import SemanticBiologyRenderer, { hasSemanticBiologyIllustration } from '@/components/biology/SemanticBiologyRenderer';

const { Paragraph, Text } = Typography;

const colorMap: Record<string, string> = {
  factor: '#1677ff',
  result: '#52c41a',
  process: '#fa8c16',
  structure: '#722ed1',
  substance: '#13c2c2',
};

interface Props {
  spec?: GenerativeVisualizationSpec;
  values?: Record<string, number>;
}

export default function GenerativeBiologyGraph({ spec, values }: Props) {
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

  const semanticAvailable = hasSemanticBiologyIllustration(spec.topic, spec.visualization_type);
  const hasProcessSteps = (spec.process_steps || []).length > 0;
  const hasExperiment = spec.experiment_variables || (spec.variable_effects || []).length > 0;
  const hasMechanism = (spec.mechanism_stages || []).length > 0;
  const hasGraph = flow.nodes.length > 0;

  const mechanismPane = (
    <Space direction="vertical" style={{ width: '100%' }} size="middle">
      {semanticAvailable ? (
        <SemanticBiologyRenderer
          topic={spec.topic}
          visualizationType={spec.visualization_type}
          values={values}
        />
      ) : hasGraph ? (
        <div style={{ height: 420, border: '1px solid var(--color-border, #e5e7eb)', borderRadius: 12, overflow: 'hidden' }}>
          <ReactFlow nodes={flow.nodes} edges={flow.edges} fitView proOptions={{ hideAttribution: true }}>
            <Background />
            <Controls />
          </ReactFlow>
        </div>
      ) : (
        <Empty description="暂无可视化结构" />
      )}
      {hasMechanism && (
        <Card size="small" title="动态机制阶段">
          <Timeline
            items={(spec.mechanism_stages || []).map((stage) => ({
              children: (
                <Space direction="vertical" size={2}>
                  <Text strong>{stage.title}</Text>
                  {stage.description && <Text type="secondary">{stage.description}</Text>}
                  {((stage.inputs || []).length > 0 || (stage.outputs || []).length > 0) && (
                    <Text type="secondary">输入：{stage.inputs?.join('、') || '-'} → 输出：{stage.outputs?.join('、') || '-'}</Text>
                  )}
                </Space>
              ),
            }))}
          />
        </Card>
      )}
      {spec.curve_explanation && <Alert type="info" showIcon message="曲线 / 限制因素解释" description={spec.curve_explanation} />}
    </Space>
  );

  const processPane = hasProcessSteps ? (
    <Card size="small" title="过程阶段">
      <Steps
        size="small"
        direction="vertical"
        current={(spec.process_steps || []).length}
        items={(spec.process_steps || []).map((step) => ({
          title: step.title,
          description: step.detail || [...(step.input || []), ...(step.output || [])].join(' → '),
        }))}
      />
    </Card>
  ) : (
    <Empty description="暂无过程拆解" />
  );

  const experimentPane = (
    <Space direction="vertical" style={{ width: '100%' }} size="middle">
      {spec.experiment_variables ? (
        <Card size="small" title="实验变量">
          <Paragraph><Text strong>自变量：</Text>{spec.experiment_variables.independent?.join('、') || '-'}</Paragraph>
          <Paragraph><Text strong>因变量：</Text>{spec.experiment_variables.dependent?.join('、') || '-'}</Paragraph>
          <Paragraph style={{ marginBottom: 0 }}><Text strong>控制变量：</Text>{spec.experiment_variables.controlled?.join('、') || '-'}</Paragraph>
        </Card>
      ) : (
        <Empty description="暂无实验变量设计" />
      )}
      {(spec.variable_effects || []).length > 0 && (
        <Card size="small" title="变量影响与限制因素">
          <List
            size="small"
            dataSource={spec.variable_effects}
            renderItem={(item) => (
              <List.Item>
                <List.Item.Meta
                  title={<Space><Tag color="blue">{item.variable}</Tag><Text>{item.effect}</Text></Space>}
                  description={item.condition ? `条件：${item.condition}` : item.evidence}
                />
              </List.Item>
            )}
          />
        </Card>
      )}
      {(spec.limiting_factors || []).length > 0 && (
        <Space wrap>
          <Text type="secondary">限制因素：</Text>
          {(spec.limiting_factors || []).map((item) => <Tag key={item}>{item}</Tag>)}
        </Space>
      )}
    </Space>
  );

  const graphPane = hasGraph ? (
    <div style={{ height: 420, border: '1px solid var(--color-border, #e5e7eb)', borderRadius: 12, overflow: 'hidden' }}>
      <ReactFlow nodes={flow.nodes} edges={flow.edges} fitView proOptions={{ hideAttribution: true }}>
        <Background />
        <Controls />
      </ReactFlow>
    </div>
  ) : (
    <Empty description="暂无概念节点" />
  );

  const items = [
    { key: 'mechanism', label: '机制图', children: mechanismPane },
    { key: 'process', label: '流程拆解', children: processPane, disabled: !hasProcessSteps && !hasMechanism },
    { key: 'experiment', label: '实验设计', children: experimentPane, disabled: !hasExperiment },
  ];
  if (semanticAvailable && hasGraph) {
    items.push({ key: 'graph', label: '概念关系图', children: graphPane, disabled: false });
  }

  return (
    <Space direction="vertical" style={{ width: '100%' }} size="middle">
      <Space wrap>
        <Tag color="purple">{spec.visualization_type}</Tag>
        <Tag color="green">{spec.topic}</Tag>
        {semanticAvailable && <Tag color="cyan">语义化插画</Tag>}
      </Space>
      <Tabs defaultActiveKey="mechanism" items={items} />
    </Space>
  );
}
