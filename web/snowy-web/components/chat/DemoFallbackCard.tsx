'use client';

/** v7 §3 / §10 R6：preview `status=failed` 时展示降级卡片，提供重试入口。 */
import React from 'react';
import { Alert, Button, Card, Space } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';

export interface DemoFallbackCardProps {
  error?: string;
  onRetry?: () => void;
  onRegenerate?: (reason: string) => void;
}

export default function DemoFallbackCard({ error, onRetry, onRegenerate }: DemoFallbackCardProps) {
  return (
    <Card size="small" style={{ marginTop: 8 }}>
      <Alert
        type="warning"
        showIcon
        message="演示生成失败"
        description={error || '本次未能生成可交互的演示，可重试或描述具体调整方向。'}
      />
      <Space style={{ marginTop: 12 }}>
        {onRetry && (
          <Button size="small" icon={<ReloadOutlined />} onClick={onRetry}>
            重试
          </Button>
        )}
        {onRegenerate && (
          <Button
            size="small"
            type="primary"
            onClick={() => onRegenerate('上次演示生成失败，请换一个角度重试')}
          >
            换一个方案
          </Button>
        )}
      </Space>
    </Card>
  );
}
