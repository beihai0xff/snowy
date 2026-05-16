import React from 'react';

export default function SkeletonPreview({
  height = 320,
  label = '生成中…',
}: {
  height?: number;
  label?: string;
}) {
  return (
    <div
      style={{
        position: 'relative',
        height,
        borderRadius: 12,
        overflow: 'hidden',
        border: '1px solid var(--color-border)',
      }}
      className="snowy-skeleton"
      role="status"
      aria-live="polite"
    >
      <div
        style={{
          position: 'absolute',
          inset: 0,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          color: 'var(--color-text-muted)',
          fontSize: 13,
          letterSpacing: '0.04em',
        }}
      >
        {label}
      </div>
    </div>
  );
}
