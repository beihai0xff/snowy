'use client';

import { useState } from 'react';
import { Button, message } from 'antd';
import { ShareAltOutlined } from '@ant-design/icons';

import { api } from '@/lib/api';

interface ShareButtonProps {
  packageId: string;
  expiresInHours?: number;
}

export function ShareButton({ packageId, expiresInHours }: ShareButtonProps) {
  const [loading, setLoading] = useState(false);

  const handleClick = async () => {
    if (!packageId) return;
    setLoading(true);
    try {
      const resp = await api.createPackageShare({
        package_id: packageId,
        expires_in_hours: expiresInHours,
      });
      const token = resp.data?.token;
      if (!token) throw new Error('返回的分享信息不完整');
      const url = `${window.location.origin}/share/${token}`;
      try {
        await navigator.clipboard.writeText(url);
        message.success('分享链接已复制到剪贴板');
      } catch {
        message.info(`分享链接：${url}`);
      }
    } catch (err) {
      message.error(err instanceof Error ? err.message : '生成分享链接失败');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Button
      size="small"
      icon={<ShareAltOutlined />}
      loading={loading}
      onClick={handleClick}
      disabled={!packageId}
    >
      分享
    </Button>
  );
}
