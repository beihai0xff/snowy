'use client';

import { Suspense, useEffect } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { Spin } from 'antd';

function BiologyRedirectInner() {
  const router = useRouter();
  const searchParams = useSearchParams();

  useEffect(() => {
    const q = searchParams.get('q');
    router.replace(`/modeling?type=biology${q ? `&q=${encodeURIComponent(q)}` : ''}`);
  }, [router, searchParams]);

  return <div style={{ textAlign: 'center', padding: 80 }}><Spin tip="正在进入统一建模页..." /></div>;
}

export default function BiologyRedirectPage() {
  return (
    <Suspense fallback={<Spin />}>
      <BiologyRedirectInner />
    </Suspense>
  );
}
