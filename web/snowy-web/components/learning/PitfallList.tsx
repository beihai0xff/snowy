/**
 * Snowy v8 · 易错点提示卡
 *
 * 根据 model package 的 knowledge_tags 匹配人教版易错点库，展示 1~3 条最相关的
 * 「易错理解 → 正确理解 → 纠错提示」。
 */

'use client';

import React, { useMemo } from 'react';
import { Alert, Card, Space, Tag, Typography } from 'antd';
import { WarningOutlined } from '@ant-design/icons';
import { topPitfalls, type SubjectKey, type Pitfall } from '@/lib/curriculum';

const { Text, Paragraph } = Typography;

interface Props {
  subject: SubjectKey;
  tags: string[];
  limit?: number;
}

const severityMeta: Record<Pitfall['severity'], { label: string; color: string }> = {
  high:   { label: '高频易错', color: 'red' },
  medium: { label: '常见易错', color: 'orange' },
  low:    { label: '提醒',     color: 'blue' },
};

export default function PitfallList({ subject, tags, limit = 3 }: Props) {
  const pitfalls = useMemo(() => topPitfalls(subject, tags, limit), [subject, tags, limit]);
  if (pitfalls.length === 0) {
    return (
      <Alert
        type="info"
        showIcon
        message="暂未匹配到精准易错点"
        description="当前模型还没有足够的知识标签；可以先查看课纲徽章或继续追问，让 Snowy 补充证据标签。"
      />
    );
  }

  return (
    <Card
      size="small"
      title={<Space size={6}><WarningOutlined style={{ color: '#f97316' }} />易错点提醒</Space>}
      className="snowy-pitfall-card"
    >
      <Space direction="vertical" size={10} style={{ width: '100%' }}>
        {pitfalls.map((item) => {
          const meta = severityMeta[item.severity] || severityMeta.medium;
          return (
            <div key={item.id} className={`snowy-pitfall-item is-${item.severity}`}>
              <Space size={6} wrap style={{ marginBottom: 4 }}>
                <Tag color={meta.color} bordered={false}>{meta.label}</Tag>
                {item.evidence_section && <Tag bordered={false}>{item.evidence_section.replace(/^pep\./, '')}</Tag>}
              </Space>
              <Paragraph style={{ marginBottom: 4, fontSize: 13 }}>
                <Text strong>容易错：</Text>{item.pitfall}
              </Paragraph>
              <Paragraph style={{ marginBottom: 4, fontSize: 13 }}>
                <Text strong type="success">正确理解：</Text>{item.correct}
              </Paragraph>
              {item.correction_hint && (
                <Text type="secondary" style={{ fontSize: 12 }}>提示：{item.correction_hint}</Text>
              )}
            </div>
          );
        })}
      </Space>
    </Card>
  );
}

