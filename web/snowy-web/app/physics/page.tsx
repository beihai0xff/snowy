'use client';

import { Suspense, useEffect } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { Spin } from 'antd';

function PhysicsRedirectInner() {
  const router = useRouter();
  const searchParams = useSearchParams();

  useEffect(() => {
    const q = searchParams.get('q');
    router.replace(`/modeling?type=physics${q ? `&q=${encodeURIComponent(q)}` : ''}`);
  }, [router, searchParams]);

  return <div style={{ textAlign: 'center', padding: 80 }}><Spin tip="正在进入统一建模页..." /></div>;
}

export default function PhysicsRedirectPage() {
  return (
    <Suspense fallback={<Spin />}>
      <PhysicsRedirectInner />
    </Suspense>
  );
}
