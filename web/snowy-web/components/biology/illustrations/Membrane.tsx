/**
 * Snowy v6 · 生物插画 · 细胞膜与物质运输
 *
 * 磷脂双分子层 + 通道蛋白 + 浓度梯度，箭头表示扩散方向。
 */

'use client';

import React from 'react';
import type { IllustrationProps } from './shared';
import './illustrations.css';

export default function Membrane({ topic = '细胞膜与物质运输' }: IllustrationProps) {
  return (
    <div className="snowy-bio-illust">
      <div className="snowy-bio-illust__title">{topic}</div>
      <div className="snowy-bio-illust__sub">磷脂双分子层 + 通道蛋白 / 载体蛋白 → 选择性透过</div>
      <svg className="snowy-bio-illust__svg" viewBox="0 0 520 260" role="img">
        <defs>
          <radialGradient id="memHead" cx="50%" cy="50%" r="50%">
            <stop offset="0%" stopColor="#fef3c7" />
            <stop offset="100%" stopColor="#d97706" />
          </radialGradient>
        </defs>

        {/* 上下区域文字 */}
        <text x="20" y="20" fontSize="11" fontWeight={700} fill="#1d4ed8">细胞外（高浓度区域）</text>
        <text x="20" y="248" fontSize="11" fontWeight={700} fill="#15803d">细胞内（低浓度区域）</text>

        {/* 上层小分子（高浓度） */}
        {Array.from({ length: 30 }).map((_, i) => (
          <circle
            key={`out-${i}`}
            cx={20 + (i * 17) % 480}
            cy={32 + Math.floor((i * 17) / 480) * 16}
            r={3 + (i % 3)}
            fill={i % 3 === 0 ? '#3b82f6' : i % 3 === 1 ? '#0ea5e9' : '#22d3ee'}
            opacity={0.85}
          />
        ))}

        {/* 磷脂双分子层 */}
        {Array.from({ length: 22 }).map((_, i) => {
          const x = 20 + i * 22;
          return (
            <g key={`p-${i}`}>
              {/* 上层 */}
              <circle cx={x} cy={120} r={9} fill="url(#memHead)" stroke="#92400e" strokeWidth={1} />
              <line x1={x - 3} y1={128} x2={x - 3} y2={148} stroke="#92400e" strokeWidth={2} />
              <line x1={x + 3} y1={128} x2={x + 3} y2={148} stroke="#92400e" strokeWidth={2} />
              {/* 下层 */}
              <circle cx={x} cy={172} r={9} fill="url(#memHead)" stroke="#92400e" strokeWidth={1} />
              <line x1={x - 3} y1={164} x2={x - 3} y2={144} stroke="#92400e" strokeWidth={2} />
              <line x1={x + 3} y1={164} x2={x + 3} y2={144} stroke="#92400e" strokeWidth={2} />
            </g>
          );
        })}

        {/* 通道蛋白 */}
        <g transform="translate(160 110)">
          <rect x="-18" y="0" width="36" height="72" rx="10" fill="#a78bfa" stroke="#5b21b6" strokeWidth={1.5} />
          <rect x="-8" y="8" width="16" height="56" rx="4" fill="#ede9fe" />
          <text y="86" fontSize="10" fontWeight={600} fill="#4c1d95" textAnchor="middle">通道蛋白</text>
        </g>
        {/* 载体蛋白 */}
        <g transform="translate(360 110)">
          <path d="M -22 0 Q -10 -8 0 0 Q 10 -8 22 0 L 22 72 Q 10 80 0 72 Q -10 80 -22 72 Z" fill="#fb7185" stroke="#9f1239" strokeWidth={1.5} />
          <circle r="6" cy={36} fill="#fee2e2" />
          <text y="92" fontSize="10" fontWeight={600} fill="#9f1239" textAnchor="middle">载体蛋白 + ATP</text>
        </g>

        {/* 自由扩散箭头 */}
        <path d="M 80 60 Q 80 120 80 220" stroke="#1d4ed8" strokeWidth={2} markerEnd="url(#memArr)" strokeDasharray="6 5" className="snowy-bio-anim-flow" />
        <text x="86" y="200" fontSize="10" fill="#1d4ed8">自由扩散</text>

        {/* 协助扩散 */}
        <path d="M 160 60 Q 160 110 160 220" stroke="#7c3aed" strokeWidth={2} markerEnd="url(#memArr2)" strokeDasharray="6 5" className="snowy-bio-anim-flow" />
        <text x="166" y="200" fontSize="10" fill="#7c3aed">协助扩散</text>

        {/* 主动运输（逆浓度） */}
        <path d="M 360 220 Q 360 110 360 60" stroke="#dc2626" strokeWidth={2} markerEnd="url(#memArr3)" strokeDasharray="6 5" className="snowy-bio-anim-flow" />
        <text x="366" y="80" fontSize="10" fill="#dc2626">主动运输（耗能）</text>

        <defs>
          <marker id="memArr" viewBox="0 0 10 10" refX="5" refY="5" markerWidth="8" markerHeight="8" orient="auto">
            <path d="M 0 0 L 10 5 L 0 10 Z" fill="#1d4ed8" />
          </marker>
          <marker id="memArr2" viewBox="0 0 10 10" refX="5" refY="5" markerWidth="8" markerHeight="8" orient="auto">
            <path d="M 0 0 L 10 5 L 0 10 Z" fill="#7c3aed" />
          </marker>
          <marker id="memArr3" viewBox="0 0 10 10" refX="5" refY="5" markerWidth="8" markerHeight="8" orient="auto">
            <path d="M 0 0 L 10 5 L 0 10 Z" fill="#dc2626" />
          </marker>
        </defs>

        {/* 下层小分子（低浓度） */}
        {Array.from({ length: 8 }).map((_, i) => (
          <circle
            key={`in-${i}`}
            cx={30 + i * 60}
            cy={228}
            r={4}
            fill="#3b82f6"
            opacity={0.7}
          />
        ))}
      </svg>
      <div className="snowy-bio-illust__legend">
        <span><i style={{ background: '#d97706' }} />磷脂头部</span>
        <span><i style={{ background: '#a78bfa' }} />通道蛋白</span>
        <span><i style={{ background: '#fb7185' }} />载体蛋白</span>
        <span><i style={{ background: '#1d4ed8' }} />自由扩散</span>
        <span><i style={{ background: '#dc2626' }} />主动运输</span>
      </div>
    </div>
  );
}
