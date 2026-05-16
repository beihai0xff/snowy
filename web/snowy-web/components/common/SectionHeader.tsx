import React from 'react';

export default function SectionHeader({
  title,
  description,
  extra,
}: {
  title: React.ReactNode;
  description?: React.ReactNode;
  extra?: React.ReactNode;
}) {
  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'flex-end',
        justifyContent: 'space-between',
        gap: 12,
        marginBottom: 12,
      }}
    >
      <div>
        <div style={{ fontSize: 16, fontWeight: 600, color: 'var(--color-text)' }}>{title}</div>
        {description && (
          <div style={{ fontSize: 13, color: 'var(--color-text-muted)', marginTop: 2 }}>{description}</div>
        )}
      </div>
      {extra && <div>{extra}</div>}
    </div>
  );
}
