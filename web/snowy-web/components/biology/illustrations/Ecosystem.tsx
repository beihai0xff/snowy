/**
 * Snowy v6 · 生物插画 · 生态系统能量流
 *
 * 食物链 + 营养级金字塔，每层能量约为下层 10%。
 */

'use client';

import React from 'react';
import type { IllustrationProps } from './shared';
import './illustrations.css';

export default function Ecosystem({ topic = '生态系统能量流' }: IllustrationProps) {
  const levels = [
    { name: '生产者', emoji: '🌿', color: '#16a34a', energy: 100000 },
    { name: '初级消费者', emoji: '🐛', color: '#84cc16', energy: 10000 },
    { name: '次级消费者', emoji: '🐦', color: '#f59e0b', energy: 1000 },
    { name: '顶级消费者', emoji: '🦅', color: '#dc2626', energy: 100 },
  ];
  const maxW = 380;

  return (
    <div className="snowy-bio-illust">
      <div className="snowy-bio-illust__title">{topic}</div>
      <div className="snowy-bio-illust__sub">能量沿食物链单向流动，每营养级仅约 10% 传递到下一级</div>
      <svg className="snowy-bio-illust__svg" viewBox="0 0 520 260" role="img">
        {/* 太阳 */}
        <g transform="translate(60 40)">
          <circle r="22" fill="#fde68a" stroke="#f59e0b" />
          <text y="4" fontSize="14" textAnchor="middle">☀</text>
          <text y="44" fontSize="10" textAnchor="middle" fill="#92400e">太阳能</text>
        </g>
        <line x1="80" y1="60" x2="130" y2="80" stroke="#f59e0b" strokeWidth={2} strokeDasharray="5 4" className="snowy-bio-anim-flow" />

        {/* 金字塔 */}
        {levels.map((level, i) => {
          const ratio = Math.max(0.05, level.energy / 100000);
          const w = maxW * ratio;
          const y = 40 + i * 48;
          return (
            <g key={level.name} transform={`translate(${260 - w / 2} ${y})`}>
              <rect width={w} height="40" fill={level.color} stroke={level.color} strokeOpacity={0.4} rx="6">
                <animate attributeName="width" from="0" to={w} dur="0.6s" fill="freeze" />
              </rect>
              <text x="-12" y="26" fontSize="20" textAnchor="end">{level.emoji}</text>
              <text x={w / 2} y="16" fontSize="11" fontWeight={700} fill="white" textAnchor="middle">{level.name}</text>
              <text x={w / 2} y="30" fontSize="10" fill="white" textAnchor="middle">{level.energy.toLocaleString()} J</text>
              {i < levels.length - 1 && (
                <path
                  d={`M ${w / 2} 40 L ${w / 2} 48`}
                  stroke="#92400e"
                  strokeWidth={2}
                  markerEnd="url(#ecoArr)"
                />
              )}
            </g>
          );
        })}
        <defs>
          <marker id="ecoArr" viewBox="0 0 10 10" refX="5" refY="5" markerWidth="8" markerHeight="8" orient="auto">
            <path d="M 0 0 L 10 5 L 0 10 Z" fill="#92400e" />
          </marker>
        </defs>

        {/* 损失能量（热量散失） */}
        <text x="490" y="60" fontSize="10" fill="#9ca3af" textAnchor="end">↗ 热量散失</text>
        <text x="490" y="108" fontSize="10" fill="#9ca3af" textAnchor="end">↗ ~90%</text>
        <text x="490" y="156" fontSize="10" fill="#9ca3af" textAnchor="end">↗ ~90%</text>
        <text x="490" y="204" fontSize="10" fill="#9ca3af" textAnchor="end">↗ ~90%</text>
      </svg>
      <div className="snowy-bio-illust__legend">
        {levels.map((l) => (
          <span key={l.name}><i style={{ background: l.color }} />{l.name}</span>
        ))}
      </div>
    </div>
  );
}
