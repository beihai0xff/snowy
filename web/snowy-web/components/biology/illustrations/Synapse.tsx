/**
 * Snowy v6 · 生物插画 · 突触传递
 *
 * 突触前膜（含囊泡）→ 突触间隙（含神经递质）→ 突触后膜（含受体）。
 */

'use client';

import React from 'react';
import type { IllustrationProps } from './shared';
import './illustrations.css';

export default function Synapse({ topic = '突触传递' }: IllustrationProps) {
  return (
    <div className="snowy-bio-illust">
      <div className="snowy-bio-illust__title">{topic}</div>
      <div className="snowy-bio-illust__sub">电信号 → 神经递质释放 → 化学信号 → 电信号（单向传递）</div>
      <svg className="snowy-bio-illust__svg" viewBox="0 0 520 260" role="img">
        <defs>
          <linearGradient id="synPre" x1="0" x2="0" y1="0" y2="1">
            <stop offset="0%" stopColor="#fbcfe8" />
            <stop offset="100%" stopColor="#db2777" />
          </linearGradient>
          <linearGradient id="synPost" x1="0" x2="0" y1="0" y2="1">
            <stop offset="0%" stopColor="#bfdbfe" />
            <stop offset="100%" stopColor="#1d4ed8" />
          </linearGradient>
        </defs>

        {/* 突触前膜（轴突末梢） */}
        <path
          d="M 30 40 L 60 30 L 230 80 Q 250 100 230 130 L 60 180 L 30 170 Z"
          fill="url(#synPre)"
          stroke="#9d174d"
          strokeWidth={1.6}
        />
        <text x="40" y="20" fontSize="11" fontWeight={700} fill="#9d174d">突触前膜（轴突末梢）</text>

        {/* 囊泡 */}
        {[
          { x: 110, y: 85, r: 12 },
          { x: 140, y: 105, r: 14 },
          { x: 175, y: 90, r: 12 },
          { x: 170, y: 135, r: 13 },
          { x: 200, y: 115, r: 11 },
        ].map((v, i) => (
          <g key={i} transform={`translate(${v.x} ${v.y})`}>
            <circle r={v.r} fill="#fef3c7" stroke="#d97706" strokeWidth={1.2} />
            {/* 递质小点 */}
            <circle r={2.5} fill="#92400e" cx={-3} cy={-2} />
            <circle r={2.5} fill="#92400e" cx={3} cy={3} />
            <circle r={2.5} fill="#92400e" cx={0} cy={5} />
          </g>
        ))}

        {/* 间隙中的神经递质 */}
        {Array.from({ length: 12 }).map((_, i) => (
          <circle
            key={i}
            cx={250 + (i % 4) * 14}
            cy={75 + Math.floor(i / 4) * 22}
            r={3.5}
            fill="#f59e0b"
          >
            <animate attributeName="cy" values={`${70 + Math.floor(i / 4) * 22};${78 + Math.floor(i / 4) * 22};${70 + Math.floor(i / 4) * 22}`} dur="2s" repeatCount="indefinite" />
          </circle>
        ))}
        <text x="290" y="50" fontSize="11" fontWeight={700} fill="#92400e">突触间隙</text>

        {/* 突触后膜（受体） */}
        <path
          d="M 320 60 Q 360 100 320 150 L 470 180 L 500 170 L 500 40 L 470 30 Z"
          fill="url(#synPost)"
          stroke="#1e3a8a"
          strokeWidth={1.6}
        />
        <text x="400" y="20" fontSize="11" fontWeight={700} fill="#1e3a8a">突触后膜（树突）</text>
        {/* 受体 */}
        {[80, 105, 130].map((y, i) => (
          <g key={i} transform={`translate(330 ${y})`}>
            <rect x="-10" y="-12" width="20" height="24" rx="4" fill="#1d4ed8" />
            <rect x="-7" y="-9" width="14" height="6" rx="2" fill="#bfdbfe" />
          </g>
        ))}

        {/* 信号箭头 */}
        <path d="M 230 110 Q 270 110 320 110" stroke="#dc2626" strokeWidth={2.5} fill="none" markerEnd="url(#synArr)" className="snowy-bio-anim-flow" />
        <defs>
          <marker id="synArr" viewBox="0 0 10 10" refX="8" refY="5" markerWidth="8" markerHeight="8" orient="auto">
            <path d="M 0 0 L 10 5 L 0 10 Z" fill="#dc2626" />
          </marker>
        </defs>

        {/* 电信号（动作电位）输入 */}
        <path d="M 10 100 q 8 -25 16 0 q 8 25 16 0" stroke="#facc15" strokeWidth={2.2} fill="none" />
        <text x="10" y="220" fontSize="10" fill="#475569">↑ 动作电位输入</text>
        <text x="380" y="220" fontSize="10" fill="#475569">↓ 后膜电位变化</text>
      </svg>
      <div className="snowy-bio-illust__legend">
        <span><i style={{ background: '#db2777' }} />前膜</span>
        <span><i style={{ background: '#fef3c7' }} />囊泡</span>
        <span><i style={{ background: '#f59e0b' }} />神经递质</span>
        <span><i style={{ background: '#1d4ed8' }} />受体</span>
      </div>
    </div>
  );
}
