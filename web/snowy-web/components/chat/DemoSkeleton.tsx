'use client';

/** v7 §3：生成中骨架，等 SSE `preview status=partial` 切到 InteractiveDemoCard。 */
import React from 'react';
import { Card, Skeleton, Space, Spin, Typography } from 'antd';

const { Text } = Typography;

export interface DemoSkeletonProps {
  stage?: 'compile' | 'recompute' | 'regenerate';
}

const stageLabel: Record<string, string> = {
  compile: '正在生成演示……',
  recompute: '正在按新参数重算……',
  regenerate: '正在重新生成演示……',
};

export default function DemoSkeleton({ stage = 'compile' }: DemoSkeletonProps) {
  return (
    <Card size="small" style={{ marginTop: 8 }}>
      <Space direction="vertical" size="small" style={{ width: '100%' }}>
        <Space size="small">
          <Spin size="small" />
          <Text type="secondary">{stageLabel[stage] ?? stageLabel.compile}</Text>
        </Space>
        <Skeleton active paragraph={{ rows: 3 }} />
      </Space>
    </Card>
  );
}
