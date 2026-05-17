/**
 * Snowy v7 · 协同 · 在场气泡（D2 §6）
 *
 * 渲染当前会话在线客户端的小头像气泡。client_id 默认作为种子色生成；
 * 当事件附带 user_id 时显示首字母。
 */

'use client';

import React from 'react';
import { Avatar, Space, Tooltip } from 'antd';
import { TeamOutlined } from '@ant-design/icons';

export interface CollabPresenceProps {
  clientIds: string[];
  selfId?: string;
}

function hashColor(id: string): string {
  let h = 0;
  for (let i = 0; i < id.length; i++) h = (h * 31 + id.charCodeAt(i)) & 0xffffffff;
  const hue = Math.abs(h) % 360;
  return `hsl(${hue}, 60%, 55%)`;
}

export default function CollabPresence({ clientIds, selfId }: CollabPresenceProps) {
  if (clientIds.length === 0) {
    return (
      <Space size={4} style={{ fontSize: 12, color: '#999' }}>
        <TeamOutlined />
        <span>暂无其他人</span>
      </Space>
    );
  }
  return (
    <Space size={4}>
      <TeamOutlined style={{ color: '#888' }} />
      <Avatar.Group size="small">
        {clientIds.slice(0, 6).map((id) => (
          <Tooltip key={id} title={id === selfId ? '你' : id.slice(0, 8)}>
            <Avatar style={{ background: hashColor(id), fontSize: 11 }}>
              {(id || '?').slice(0, 1).toUpperCase()}
            </Avatar>
          </Tooltip>
        ))}
      </Avatar.Group>
      {clientIds.length > 6 && (
        <span style={{ fontSize: 12, color: '#999' }}>+{clientIds.length - 6}</span>
      )}
    </Space>
  );
}
