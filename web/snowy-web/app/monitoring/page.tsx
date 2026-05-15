'use client';

import React, { useEffect } from 'react';
import { useRouter } from 'next/navigation';

export default function MonitoringRedirect() {
  const router = useRouter();
  useEffect(() => {
    router.replace('/admin/llm');
  }, [router]);
  return (
    <div className="snowy-loading-card" style={{ margin: '40px auto', maxWidth: 360 }}>
      <span className="snowy-spinner" />
      <span>监控已迁移到管理后台，正在跳转…</span>
    </div>
  );
}
