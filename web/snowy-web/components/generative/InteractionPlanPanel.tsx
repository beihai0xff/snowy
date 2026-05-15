'use client';

import React from 'react';
import { Alert, Card, List, Slider, Space, Tag, Typography } from 'antd';
import type { GenerativeModelPackage } from '@/lib/api';

const { Text } = Typography;

interface Props {
  pkg: GenerativeModelPackage;
  values: Record<string, number>;
  onChange: (name: string, value: number) => void;
}

export default function InteractionPlanPanel({ pkg, values, onChange }: Props) {
  const variables = pkg.simulation_logic?.variables || pkg.generative_model.variables || [];
  const local = new Set(pkg.interaction_plan?.regeneration_policy?.local_recompute || []);
  return (
    <Space direction="vertical" style={{ width: '100%' }} size="middle">
      {pkg.interaction_plan?.challenge && (
        <Alert type="success" showIcon message="生成式挑战" description={pkg.interaction_plan.challenge.goal} />
      )}
      {variables.length > 0 && (
        <Card size="small" title="参数控制" extra={<Tag color="blue">{local.size} 个可本地重算</Tag>}>
          <Space direction="vertical" style={{ width: '100%' }}>
            {variables.map((variable) => (
              <div key={variable.name}>
                <Space style={{ width: '100%', justifyContent: 'space-between' }}>
                  <Text>{variable.label || variable.name}</Text>
                  <Text type="secondary">{values[variable.name] ?? variable.default} {variable.unit}</Text>
                </Space>
                <Slider
                  min={variable.min}
                  max={variable.max}
                  step={variable.step || 1}
                  value={values[variable.name] ?? variable.default}
                  onChange={(value) => onChange(variable.name, value)}
                />
                {!local.has(variable.name) && <Text type="warning">该变量变化可能需要大模型再推理</Text>}
              </div>
            ))}
          </Space>
        </Card>
      )}
      {(pkg.assessment_tasks || []).length > 0 && (
        <Card size="small" title="微练习 / 反思">
          <List
            size="small"
            dataSource={pkg.assessment_tasks}
            renderItem={(item) => (
              <List.Item>
                <Space direction="vertical" size={2}>
                  <Text>{item.question}</Text>
                  {item.expected_key_points && item.expected_key_points.length > 0 && (
                    <Space wrap>{item.expected_key_points.map((point) => <Tag key={point}>{point}</Tag>)}</Space>
                  )}
                  {item.misconception_type && <Text type="warning">易错类型：{item.misconception_type}</Text>}
                  {item.next_action && <Text type="secondary">下一步：{item.next_action}</Text>}
                </Space>
              </List.Item>
            )}
          />
        </Card>
      )}
    </Space>
  );
}
