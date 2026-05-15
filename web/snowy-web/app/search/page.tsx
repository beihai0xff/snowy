'use client';

import React, { Suspense, useEffect } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';

function SearchRedirect() {
  const router = useRouter();
  const params = useSearchParams();

  useEffect(() => {
    const q = params.get('q');
    const target = q ? `/ask?q=${encodeURIComponent(q)}` : '/ask';
    router.replace(target);
  }, [params, router]);

  return (
    <div className="snowy-loading-card" style={{ margin: '40px auto', maxWidth: 360 }}>
      <span className="snowy-spinner" />
      <span>正在跳转到新的提问页…</span>
    </div>
  );
}

export default function SearchPage() {
  return (
    <Suspense fallback={null}>
      <SearchRedirect />
    </Suspense>
  );
}
