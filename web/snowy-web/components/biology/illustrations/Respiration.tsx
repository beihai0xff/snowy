/**
 * Snowy v6 · 生物插画 · 细胞呼吸
 *
 * 三段流程：糖酵解（细胞质）→ 柠檬酸循环（基质）→ 电子传递链（内膜）；
 * 标注每段 ATP 产量，配色按区域区分。
 */

'use client';

import React from 'react';
import type { IllustrationProps } from './shared';
import './illustrations.css';

export default function Respiration({ topic = '细胞呼吸' }: IllustrationProps) {
  return (
    <div className="snowy-bio-illust">
      <div className="snowy-bio-illust__title">{topic}</div>
      <div className="snowy-bio-illust__sub">C₆H₁₂O₆ + O₂ → CO₂ + H₂O + 能量(ATP)</div>
      <svg className="snowy-bio-illust__svg" viewBox="0 0 520 260" role="img" aria-label="细胞呼吸示意图">
        <defs>
          <radialGradient id="rsMito" cx="50%" cy="50%" r="50%">
            <stop offset="0%" stopColor="#fecaca" />
            <stop offset="100%" stopColor="#dc2626" />
          </radialGradient>
        </defs>
        {/* 线粒体外形 */}
        <ellipse cx="320" cy="135" rx="170" ry="105" fill="#fef2f2" stroke="#fca5a5" strokeWidth={2} />
        {/* 内膜（嵴） */}
        {[0, 1, 2, 3].map((i) => (
          <path
            key={i}
            d={`M ${230 + i * 50} 60 q -10 25 0 50 q 10 25 0 50`}
            fill="none"
            stroke="#dc2626"
            strokeWidth={2.2}
            opacity={0.6}
          />
        ))}

        {/* 糖酵解（细胞质） */}
        <g transform="translate(70 60)">
          <rect x="0" y="0" width="120" height="140" rx="12" fill="#fef3c7" stroke="#d97706" strokeWidth={1.6} />
          <text x="60" y="22" fontSize="12" fontWeight={700} fill="#92400e" textAnchor="middle">糖酵解</text>
          <text x="60" y="38" fontSize="10" fill="#92400e" textAnchor="middle">细胞质</text>
          <g transform="translate(60 70)" textAnchor="middle">
            <circle r="22" fill="#fde68a" stroke="#d97706" />
            <text y="-2" fontSize="10" fill="#92400e">葡萄糖</text>
            <text y="12" fontSize="10" fill="#92400e">C₆</text>
          </g>
          <line x1="60" y1="100" x2="60" y2="120" stroke="#d97706" strokeWidth={2} markerEnd="url(#rsArr)" />
          <text x="60" y="135" fontSize="10" fill="#b45309" textAnchor="middle">2 丙酮酸</text>
          <text x="60" y="155" fontSize="11" fontWeight={700} fill="#16a34a" textAnchor="middle">+2 ATP</text>
        </g>

        {/* 柠檬酸循环 */}
        <g transform="translate(220 90)">
          <circle r="46" fill="#fee2e2" stroke="#dc2626" strokeWidth={1.8} />
          <text y="-2" fontSize="11" fontWeight={700} fill="#7f1d1d" textAnchor="middle">柠檬酸循环</text>
          <text y="14" fontSize="10" fill="#7f1d1d" textAnchor="middle">基质</text>
          {/* 循环箭头 */}
          {Array.from({ length: 8 }).map((_, i) => {
            const a1 = (i / 8) * Math.PI * 2 - Math.PI / 2;
            const a2 = ((i + 0.7) / 8) * Math.PI * 2 - Math.PI / 2;
            return (
              <path
                key={i}
                d={`M ${Math.cos(a1) * 32} ${Math.sin(a1) * 32} A 32 32 0 0 1 ${Math.cos(a2) * 32} ${Math.sin(a2) * 32}`}
                stroke="#dc2626"
                strokeWidth={1.5}
                fill="none"
                opacity={0.6}
              />
            );
          })}
          <text y="70" fontSize="11" fontWeight={700} fill="#16a34a" textAnchor="middle">+2 ATP</text>
        </g>

        {/* 电子传递链 */}
        <g transform="translate(360 75)">
          <rect width="130" height="120" rx="12" fill="#ede9fe" stroke="#7c3aed" strokeWidth={1.6} />
          <text x="65" y="22" fontSize="12" fontWeight={700} fill="#4c1d95" textAnchor="middle">电子传递链</text>
          <text x="65" y="38" fontSize="10" fill="#4c1d95" textAnchor="middle">内膜（嵴）</text>
          {[0, 1, 2, 3].map((i) => (
            <rect key={i} x={12 + i * 28} y="55" width="22" height="40" rx="6" fill="#a78bfa" stroke="#5b21b6" />
          ))}
          <text x="65" y="115" fontSize="11" fontWeight={700} fill="#16a34a" textAnchor="middle">+~34 ATP</text>
        </g>

        {/* 输入输出 */}
        <line x1="190" y1="130" x2="220" y2="115" stroke="#dc2626" strokeWidth={2} className="snowy-bio-anim-flow" />
        <line x1="270" y1="115" x2="360" y2="115" stroke="#7c3aed" strokeWidth={2} className="snowy-bio-anim-flow" />

        <g transform="translate(60 230)">
          <text fontSize="11" fill="#475569">O₂ 进入 → 终末电子受体</text>
        </g>
        <g transform="translate(330 230)">
          <text fontSize="11" fill="#475569">CO₂ + H₂O 释放</text>
        </g>
        <defs>
          <marker id="rsArr" viewBox="0 0 10 10" refX="5" refY="5" markerWidth="6" markerHeight="6" orient="auto">
            <path d="M 0 0 L 10 5 L 0 10 Z" fill="#d97706" />
          </marker>
        </defs>
      </svg>
      <div className="snowy-bio-illust__legend">
        <span><i style={{ background: '#fde68a' }} />糖酵解</span>
        <span><i style={{ background: '#fee2e2' }} />柠檬酸循环</span>
        <span><i style={{ background: '#ede9fe' }} />电子传递链</span>
        <span><i style={{ background: '#16a34a' }} />ATP 产能</span>
      </div>
      <div className="snowy-bio-illust__metrics">
        <div className="snowy-bio-illust__metric"><span>糖酵解</span><b>2 ATP</b></div>
        <div className="snowy-bio-illust__metric"><span>柠檬酸循环</span><b>2 ATP</b></div>
        <div className="snowy-bio-illust__metric"><span>电子传递链</span><b>~34 ATP</b></div>
        <div className="snowy-bio-illust__metric"><span>合计</span><b>~38 ATP</b></div>
      </div>
    </div>
  );
}
