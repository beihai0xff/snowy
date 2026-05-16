'use client';

import React from 'react';
import { Alert, Button, Card, List, Slider, Space, Tag, Typography } from 'antd';
import { ThunderboltOutlined } from '@ant-design/icons';
import type { GenerativeModelPackage } from '@/lib/api';

const { Text } = Typography;

type Subject = 'physics' | 'biology';

interface Props {
  pkg: GenerativeModelPackage;
  values: Record<string, number>;
  subject: Subject;
  onChange: (name: string, value: number) => void;
  onRegenerate?: () => void;
}

const SUBJECT_TOKENS: Record<Subject, { color: string; soft: string }> = {
  physics: { color: 'var(--color-physics, #0891B2)', soft: 'var(--color-physics-soft, #CFFAFE)' },
  biology: { color: 'var(--color-biology, #16A34A)', soft: 'var(--color-biology-soft, #DCFCE7)' },
};

export default function InteractionPlanPanel({ pkg, values, subject, onChange, onRegenerate }: Props) {
  const variables = pkg.simulation_logic?.variables || pkg.generative_model.variables || [];
  const local = new Set(pkg.interaction_plan?.regeneration_policy?.local_recompute || []);
  const tokens = SUBJECT_TOKENS[subject];

  return (
    <Space direction="vertical" style={{ width: '100%' }} size="middle">
      {pkg.interaction_plan?.challenge && (
        <Alert type="success" showIcon message="生成式挑战" description={pkg.interaction_plan.challenge.goal} />
      )}
      {variables.length > 0 && (
        <Card
          size="small"
          title="参数控制"
          extra={<Tag style={{ background: tokens.soft, color: tokens.color, border: 0 }}>{local.size} 个可本地重算</Tag>}
        >
          <Space direction="vertical" style={{ width: '100%' }} size={12}>
            {variables.map((variable) => {
              const current = values[variable.name] ?? variable.default;
              const needsRecompile = !local.has(variable.name);
              return (
                <div key={variable.name} className="snowy-param-row" style={{ ['--slider-color' as string]: tokens.color, ['--slider-soft' as string]: tokens.soft }}>
                  <div className="snowy-param-row__head">
                    <Text strong style={{ fontSize: 13 }}>{variable.label || variable.name}</Text>
                    <span className="snowy-param-chip" style={{ background: tokens.soft, color: tokens.color }}>
                      {typeof current === 'number' ? current.toFixed(variable.step && variable.step < 1 ? 2 : 0) : current}
                      {variable.unit ? ` ${variable.unit}` : ''}
                    </span>
                  </div>
                  <Slider
                    min={variable.min}
                    max={variable.max}
                    step={variable.step || 1}
                    value={typeof current === 'number' ? current : variable.default}
                    onChange={(value) => onChange(variable.name, Array.isArray(value) ? value[0] : value)}
                    styles={{
                      track: { background: tokens.color },
                      rail: { background: tokens.soft },
                      handle: { borderColor: tokens.color },
                    }}
                    tooltip={{ formatter: (v) => `${v}${variable.unit ? ` ${variable.unit}` : ''}` }}
                  />
                  {needsRecompile && (
                    <div className="snowy-param-row__hint">
                      <Text type="warning" style={{ fontSize: 12 }}>该变量变化超出本地重算范围，需要再推理</Text>
                      {onRegenerate && (
                        <Button size="small" type="link" icon={<ThunderboltOutlined />} onClick={onRegenerate} style={{ padding: 0, height: 'auto' }}>
                          以当前参数再推理
                        </Button>
                      )}
                    </div>
                  )}
                </div>
              );
            })}
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
