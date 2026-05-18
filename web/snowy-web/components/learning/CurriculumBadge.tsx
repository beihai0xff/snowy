/**
 * Snowy v8 · 课纲徽章
 *
 * 在演示卡顶端展示"📚 人教版·必修2·§5.4 平抛运动"；
 * 点击展开查看学习目标与核心公式。
 */

'use client';

import React, { useMemo, useState } from 'react';
import { Popover, Tag, Space, Typography } from 'antd';
import { BookOutlined, RightOutlined } from '@ant-design/icons';
import { topCurriculum, formatCurriculumLabel, type SubjectKey, type CurriculumRef } from '@/lib/curriculum';

const { Text } = Typography;

interface Props {
  subject: SubjectKey;
  tags: string[];
  compact?: boolean;
}

function DifficultyTag({ d }: { d: CurriculumRef['difficulty'] }) {
  const map = { easy: { color: 'green', text: '基础' }, medium: { color: 'gold', text: '中等' }, hard: { color: 'red', text: '拔高' } } as const;
  const m = map[d];
  return <Tag color={m.color} bordered={false} style={{ fontSize: 11 }}>{m.text}</Tag>;
}

export default function CurriculumBadge({ subject, tags, compact = false }: Props) {
  const refs = useMemo(() => topCurriculum(subject, tags, 3), [subject, tags]);
  const [open, setOpen] = useState(false);
  if (refs.length === 0) return null;

  const primary = refs[0];

  const content = (
    <div style={{ maxWidth: 360 }}>
      {refs.map((ref) => (
        <div key={ref.section_id} style={{ padding: '6px 0', borderBottom: '1px dashed var(--color-divider, #e5e7eb)' }}>
          <Space size={6} wrap>
            <Text strong style={{ fontSize: 13 }}>{formatCurriculumLabel(ref)}</Text>
            <DifficultyTag d={ref.difficulty} />
          </Space>
          {ref.learning_goals.length > 0 && (
            <ul style={{ paddingLeft: 18, margin: '6px 0 0', fontSize: 12, color: 'var(--color-text-muted, #64748b)' }}>
              {ref.learning_goals.slice(0, 3).map((g, i) => <li key={i}>{g}</li>)}
            </ul>
          )}
          {ref.core_formulas.length > 0 && (
            <div style={{ marginTop: 4 }}>
              {ref.core_formulas.slice(0, 2).map((f, i) => (
                <Tag key={i} bordered={false} style={{ background: '#f1f5f9', color: '#0f172a', fontFamily: 'KaTeX_Main, Cambria Math, serif' }}>{f}</Tag>
              ))}
            </div>
          )}
        </div>
      ))}
    </div>
  );

  return (
    <Popover
      content={content}
      title={<Space size={6}><BookOutlined />相关课本章节</Space>}
      trigger="click"
      open={open}
      onOpenChange={setOpen}
      placement="bottomLeft"
    >
      <button
        type="button"
        className={`snowy-curriculum-badge ${compact ? 'is-compact' : ''}`}
        aria-label="查看相关课本章节"
      >
        <BookOutlined style={{ fontSize: 13 }} />
        <span className="snowy-curriculum-badge__main">
          🎓 {formatCurriculumLabel(primary)}
        </span>
        {refs.length > 1 && <span className="snowy-curriculum-badge__more">+{refs.length - 1}</span>}
        <RightOutlined style={{ fontSize: 10, opacity: 0.6 }} />
      </button>
    </Popover>
  );
}

