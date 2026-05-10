'use client';

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Button, Space, Tooltip, Typography, message } from 'antd';
import { DislikeOutlined, LikeOutlined } from '@ant-design/icons';
import { api, type ReactionReq, type ReactionSummary } from '@/lib/api';

const { Text } = Typography;

interface ReactionBarProps {
  targetType: ReactionReq['target_type'];
  targetID?: string;
  size?: 'small' | 'middle' | 'large';
  showUsers?: boolean;
}

function userTooltip(summary: ReactionSummary | null, type: 'like' | 'dislike'): string {
  if (!summary) return '暂无用户';
  const users = type === 'like' ? summary.like_users : summary.dislike_users;
  if (!users?.length) return '暂无公开用户';
  return users.map((user) => user.nickname || user.id).slice(0, 8).join('、');
}

export default function ReactionBar({ targetType, targetID, size = 'small', showUsers = true }: ReactionBarProps) {
  const [summary, setSummary] = useState<ReactionSummary | null>(null);
  const [loading, setLoading] = useState(false);

  const effectiveTargetID = useMemo(() => (targetID || '').trim(), [targetID]);

  const load = useCallback(async () => {
    if (!effectiveTargetID) {
      setSummary(null);
      return;
    }
    try {
      const resp = await api.getReactionSummary(targetType, effectiveTargetID, showUsers);
      setSummary(resp.data || null);
    } catch {
      // 反馈是辅助质量信号，不阻断主学习链路。
      setSummary(null);
    }
  }, [effectiveTargetID, showUsers, targetType]);

  useEffect(() => {
    void load();
  }, [load]);

  const mutate = async (reactionType: 'like' | 'dislike') => {
    if (!effectiveTargetID) return;
    setLoading(true);
    try {
      const current = summary?.current_reaction;
      if (current === reactionType) {
        await api.deleteReaction(targetType, effectiveTargetID);
        message.success('已撤销反馈');
      } else {
        const resp = await api.setReaction({
          target_type: targetType,
          target_id: effectiveTargetID,
          reaction_type: reactionType,
          visibility: 'public',
        });
        setSummary(resp.data || null);
        message.success(reactionType === 'like' ? '已点赞' : '已点踩');
        return;
      }
      await load();
    } catch (error) {
      message.error(error instanceof Error ? error.message : '反馈提交失败');
    } finally {
      setLoading(false);
    }
  };

  const likeActive = summary?.current_reaction === 'like';
  const dislikeActive = summary?.current_reaction === 'dislike';

  return (
    <Space size={6} wrap>
      <Tooltip title={showUsers ? `公开点赞用户：${userTooltip(summary, 'like')}` : undefined}>
        <Button
          size={size}
          type={likeActive ? 'primary' : 'default'}
          icon={<LikeOutlined />}
          loading={loading && likeActive}
          disabled={!effectiveTargetID}
          onClick={() => void mutate('like')}
        >
          {summary?.like_count ?? 0}
        </Button>
      </Tooltip>
      <Tooltip title={showUsers ? `公开点踩用户：${userTooltip(summary, 'dislike')}` : undefined}>
        <Button
          size={size}
          danger={dislikeActive}
          icon={<DislikeOutlined />}
          loading={loading && dislikeActive}
          disabled={!effectiveTargetID}
          onClick={() => void mutate('dislike')}
        >
          {summary?.dislike_count ?? 0}
        </Button>
      </Tooltip>
      <Text type="secondary" style={{ fontSize: 12 }}>社区质量信号</Text>
    </Space>
  );
}
