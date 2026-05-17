'use client';

/**
 * v7 §3 / §6 / §10：ChatBubble
 *
 * 一条消息（user 或 assistant）的渲染容器。当消息附带 `package` 时，
 * 渲染：
 *   - assistant 文本（MarkdownText）
 *   - 引用列表（如有）
 *   - 演示状态机：skeleton → InteractiveDemoCard | DemoFallbackCard
 *
 * 不直接发起 chat 请求；通过 `onRegenerateRequest` 将“换一个”意图上报给上层
 * ask 页面，由 ask 页面统一以 ChatReq.parent_package_id + regenerate_reason 触发新一轮 SSE。
 */

import React from 'react';
import { Avatar, Card, Space, Tag, Typography } from 'antd';
import { RobotOutlined, UserOutlined } from '@ant-design/icons';
import MarkdownText from '@/components/common/MarkdownText';
import type { Citation, GenerativeModelPackage } from '@/lib/api';
import InteractiveDemoCard from './InteractiveDemoCard';
import DemoSkeleton from './DemoSkeleton';
import DemoFallbackCard from './DemoFallbackCard';

const { Text } = Typography;

export type DemoStatus = 'idle' | 'partial' | 'complete' | 'failed';

export interface ChatBubbleProps {
  role: 'user' | 'assistant';
  content: string;
  citations?: Citation[];
  /** 当前已知的 generative package（partial 或 complete 时都可能有）。 */
  pkg?: GenerativeModelPackage | null;
  demoStatus?: DemoStatus;
  demoStage?: 'compile' | 'recompute' | 'regenerate';
  demoError?: string | null;
  onPackageUpdate?: (next: GenerativeModelPackage) => void;
  onRegenerateRequest?: (reason: string) => void;
  onRetry?: () => void;
}

export default function ChatBubble({
  role,
  content,
  citations,
  pkg,
  demoStatus = 'idle',
  demoStage,
  demoError,
  onPackageUpdate,
  onRegenerateRequest,
  onRetry,
}: ChatBubbleProps) {
  const isUser = role === 'user';
  return (
    <div
      style={{
        display: 'flex',
        flexDirection: isUser ? 'row-reverse' : 'row',
        gap: 12,
        marginBottom: 12,
        alignItems: 'flex-start',
      }}
    >
      <Avatar
        icon={isUser ? <UserOutlined /> : <RobotOutlined />}
        style={{ background: isUser ? '#1677ff' : '#52c41a', flexShrink: 0 }}
      />
      <div style={{ maxWidth: '85%', flex: 1 }}>
        <Card
          size="small"
          styles={{ body: { padding: 12 } }}
          style={{
            background: isUser ? '#e6f4ff' : '#fff',
            borderColor: isUser ? '#91caff' : '#f0f0f0',
          }}
        >
          {isUser ? <Text>{content}</Text> : <MarkdownText content={content} />}
        </Card>

        {!isUser && citations && citations.length > 0 && (
          <Space size={[4, 4]} wrap style={{ marginTop: 6 }}>
            {citations.slice(0, 5).map((c, i) => (
              <Tag key={`${c.doc_id || i}`} color="default">
                [{i + 1}] {c.doc_id || c.source_type || '引用'}
              </Tag>
            ))}
          </Space>
        )}

        {!isUser && demoStatus === 'partial' && !pkg && (
          <DemoSkeleton stage={demoStage} />
        )}
        {!isUser && demoStatus === 'failed' && (
          <DemoFallbackCard
            error={demoError ?? undefined}
            onRetry={onRetry}
            onRegenerate={onRegenerateRequest}
          />
        )}
        {!isUser && pkg && (demoStatus === 'complete' || demoStatus === 'partial') && (
          <InteractiveDemoCard
            pkg={pkg}
            onPackageUpdate={onPackageUpdate}
            onRegenerateRequest={onRegenerateRequest}
          />
        )}
      </div>
    </div>
  );
}
