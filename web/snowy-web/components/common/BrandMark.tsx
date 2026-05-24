import React from 'react';

export interface BrandMarkProps {
  size?: number;
  title?: string;
  className?: string;
  style?: React.CSSProperties;
  /**
   * Whether the mark stands alone (needs its own a11y label) or is paired
   * with an adjacent text label and should be hidden from assistive tech.
   * Defaults to `true` (decorative) — the most common use case is mark
   * next to a `Snowy` wordmark, where the text already carries the label.
   */
  decorative?: boolean;
}

const NODES_MID: ReadonlyArray<readonly [number, number]> = [
  [12, 6],
  [17.196, 9],
  [17.196, 15],
  [12, 18],
  [6.804, 15],
  [6.804, 9],
];

const NODES_TIP: ReadonlyArray<readonly [number, number]> = [
  [12, 2],
  [20.66, 7],
  [20.66, 17],
  [12, 22],
  [3.34, 17],
  [3.34, 7],
];

export default function BrandMark({
  size = 24,
  title = 'Snowy',
  className,
  style,
  decorative = true,
}: BrandMarkProps) {
  const a11yProps = decorative
    ? ({ 'aria-hidden': true, focusable: false } as const)
    : ({ role: 'img', 'aria-label': title } as const);
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      width={size}
      height={size}
      className={className}
      style={{ color: 'var(--color-primary)', display: 'block', ...style }}
      {...a11yProps}
    >
      {!decorative && <title>{title}</title>}
      <g stroke="currentColor" strokeWidth={1.3} strokeLinecap="round" fill="none">
        {NODES_TIP.map(([x, y], i) => (
          <line key={`line-${i}`} x1={12} y1={12} x2={x} y2={y} />
        ))}
      </g>
      <g fill="currentColor">
        {NODES_MID.map(([x, y], i) => (
          <circle key={`mid-${i}`} cx={x} cy={y} r={1.4} />
        ))}
        {NODES_TIP.map(([x, y], i) => (
          <circle key={`tip-${i}`} cx={x} cy={y} r={0.9} />
        ))}
        <circle cx={12} cy={12} r={2.5} />
      </g>
    </svg>
  );
}
