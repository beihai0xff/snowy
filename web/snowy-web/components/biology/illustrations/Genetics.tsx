/**
 * Snowy v6 · 生物插画 · 遗传学（孟德尔 Punnett 方格）
 *
 * 父代 × 母代基因型 → 子代基因型 + 表现型比例柱状图。
 */

'use client';

import React from 'react';
import type { IllustrationProps } from './shared';
import './illustrations.css';

export default function Genetics({ topic = '孟德尔遗传定律' }: IllustrationProps) {
  // 默认 Aa × Aa
  const alleles1 = ['A', 'a'];
  const alleles2 = ['A', 'a'];
  const grid = alleles1.flatMap((a) => alleles2.map((b) => `${a}${b}`));
  const counts = grid.reduce<Record<string, number>>((acc, g) => {
    const key = g.split('').sort((x, y) => (x > y ? -1 : 1)).join('');
    acc[key] = (acc[key] || 0) + 1;
    return acc;
  }, {});

  return (
    <div className="snowy-bio-illust">
      <div className="snowy-bio-illust__title">{topic}</div>
      <div className="snowy-bio-illust__sub">Aa × Aa → 1 AA : 2 Aa : 1 aa（基因型 1:2:1，表现型 3:1）</div>
      <svg className="snowy-bio-illust__svg" viewBox="0 0 520 260" role="img">
        {/* 父代 */}
        <g transform="translate(40 50)">
          <circle r="32" fill="#fce7f3" stroke="#db2777" strokeWidth={1.6} />
          <text fontSize="14" fontWeight={700} fill="#9d174d" textAnchor="middle" y="5">♀ Aa</text>
          <text fontSize="10" fill="#831843" textAnchor="middle" y="48">母本（杂合）</text>
        </g>
        <g transform="translate(40 180)">
          <circle r="32" fill="#dbeafe" stroke="#1d4ed8" strokeWidth={1.6} />
          <text fontSize="14" fontWeight={700} fill="#1e3a8a" textAnchor="middle" y="5">♂ Aa</text>
          <text fontSize="10" fill="#1e3a8a" textAnchor="middle" y="48">父本（杂合）</text>
        </g>

        {/* Punnett 方格 */}
        <g transform="translate(150 30)">
          {/* 表头 */}
          {['A', 'a'].map((a, i) => (
            <g key={a}>
              <rect x={50 + i * 70} y="0" width="70" height="36" fill="#fce7f3" stroke="#db2777" />
              <text x={85 + i * 70} y="24" fontSize="16" fontWeight={700} fill="#9d174d" textAnchor="middle">{a}</text>
            </g>
          ))}
          {['A', 'a'].map((a, i) => (
            <g key={a}>
              <rect x="0" y={36 + i * 70} width="50" height="70" fill="#dbeafe" stroke="#1d4ed8" />
              <text x="25" y={78 + i * 70} fontSize="16" fontWeight={700} fill="#1e3a8a" textAnchor="middle">{a}</text>
            </g>
          ))}
          {/* 子代格子 */}
          {[0, 1].map((r) => [0, 1].map((c) => {
            const g = ['A', 'a'][r] + ['A', 'a'][c];
            const sorted = g.split('').sort((x, y) => (x > y ? -1 : 1)).join('');
            const fill = sorted === 'AA' ? '#dcfce7' : sorted === 'Aa' ? '#fef3c7' : '#fee2e2';
            const stroke = sorted === 'AA' ? '#16a34a' : sorted === 'Aa' ? '#d97706' : '#dc2626';
            return (
              <g key={`${r}${c}`}>
                <rect x={50 + c * 70} y={36 + r * 70} width="70" height="70" fill={fill} stroke={stroke} strokeWidth={1.6} />
                <text x={85 + c * 70} y={80 + r * 70} fontSize="18" fontWeight={700} fill={stroke} textAnchor="middle">{sorted}</text>
              </g>
            );
          }))}
        </g>

        {/* 比例柱状图 */}
        <g transform="translate(340 60)">
          <text fontSize="11" fontWeight={700} fill="#374151" y="-6">子代表现型比例</text>
          {[
            { key: 'AA', color: '#16a34a', label: '显性纯合 AA', count: counts['AA'] || 0 },
            { key: 'Aa', color: '#d97706', label: '杂合 Aa', count: counts['Aa'] || 0 },
            { key: 'aa', color: '#dc2626', label: '隐性纯合 aa', count: counts['aa'] || 0 },
          ].map((row, i) => (
            <g key={row.key} transform={`translate(0 ${i * 44})`}>
              <text fontSize="11" fill="#374151">{row.label}</text>
              <rect y="6" width="140" height="22" fill="#e5e7eb" rx="4" />
              <rect y="6" width={(140 * row.count) / 4} height="22" fill={row.color} rx="4">
                <animate attributeName="width" to={(140 * row.count) / 4} dur="0.5s" fill="freeze" />
              </rect>
              <text x="148" y="22" fontSize="11" fontWeight={600} fill={row.color}>{row.count} / 4</text>
            </g>
          ))}
        </g>
      </svg>
      <div className="snowy-bio-illust__legend">
        <span><i style={{ background: '#16a34a' }} />显性纯合</span>
        <span><i style={{ background: '#d97706' }} />杂合</span>
        <span><i style={{ background: '#dc2626' }} />隐性纯合</span>
      </div>
    </div>
  );
}
