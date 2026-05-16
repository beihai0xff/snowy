/**
 * Snowy v6 · 生物插画 · 酶催化
 *
 * 底物-酶-产物模型 + 温度/pH 活性曲线（钟形）。
 */

'use client';

import React from 'react';
import type { IllustrationProps } from './shared';
import { numVal } from './shared';
import './illustrations.css';

export default function Enzyme({ values, topic = '酶催化反应' }: IllustrationProps) {
  const temp = numVal(values, 'temperature', 37);
  const ph = numVal(values, 'ph', 7);
  const optimalTemp = 37;
  const optimalPh = 7;
  const tempAct = Math.exp(-((temp - optimalTemp) ** 2) / 200);
  const phAct = Math.exp(-((ph - optimalPh) ** 2) / 4);
  const activity = Math.max(0.05, Math.min(1, tempAct * phAct));

  const curvePoints = Array.from({ length: 80 }).map((_, i) => {
    const t = i / 79;
    const x = t * 320;
    const T = 10 + t * 60;
    const a = Math.exp(-((T - optimalTemp) ** 2) / 200);
    const y = 140 - a * 110;
    return `${x.toFixed(1)} ${y.toFixed(1)}`;
  }).join(' L ');

  return (
    <div className="snowy-bio-illust">
      <div className="snowy-bio-illust__title">{topic}</div>
      <div className="snowy-bio-illust__sub">底物 + 酶 → 酶-底物复合物 → 产物 + 酶（不变）</div>
      <svg className="snowy-bio-illust__svg" viewBox="0 0 520 260" role="img">
        <defs>
          <linearGradient id="enzEnzyme" x1="0" x2="1">
            <stop offset="0%" stopColor="#86efac" />
            <stop offset="100%" stopColor="#15803d" />
          </linearGradient>
        </defs>
        {/* 三阶段图 */}
        <g transform="translate(20 30)">
          {[
            { x: 0, label: '识别', sub: 1, key: 1 },
            { x: 70, label: '结合', sub: 0.6, key: 2 },
            { x: 140, label: '释放', sub: 0, key: 3 },
          ].map((stage, idx) => (
            <g key={stage.key} transform={`translate(${stage.x} 0)`}>
              {/* 酶（V 型缺口） */}
              <path
                d="M 0 30 Q -8 10 8 0 L 32 0 Q 50 4 50 22 L 36 30 L 36 38 L 50 50 Q 50 62 36 64 L 8 64 Q -8 56 0 36 Z"
                fill="url(#enzEnzyme)"
                stroke="#15803d"
                strokeWidth={1.5}
              />
              {/* 底物 */}
              {stage.sub > 0 && (
                <circle cx="42" cy="34" r="9" fill="#fb7185" stroke="#be123c" strokeWidth={1.2} opacity={stage.sub} />
              )}
              {/* 产物 */}
              {idx === 2 && (
                <>
                  <circle cx="68" cy="20" r="5" fill="#f97316" />
                  <circle cx="72" cy="40" r="5" fill="#f97316" />
                </>
              )}
              <text x="22" y="84" fontSize="11" fill="#166534" fontWeight={600} textAnchor="middle">{stage.label}</text>
            </g>
          ))}
        </g>

        {/* 活性曲线 */}
        <g transform="translate(170 110)">
          <text x="0" y="-6" fontSize="11" fill="#374151" fontWeight={600}>温度 vs 酶活性</text>
          <rect x="0" y="0" width="320" height="140" fill="white" stroke="#cbd5e1" rx="6" />
          {/* 网格 */}
          {[0.25, 0.5, 0.75].map((t) => (
            <line key={t} x1="0" y1={140 * t} x2="320" y2={140 * t} stroke="#e2e8f0" strokeDasharray="3 3" />
          ))}
          <path d={`M 0 140 L ${curvePoints}`} fill="none" stroke="#16a34a" strokeWidth={2.4} />
          {/* 当前点 */}
          <g transform={`translate(${Math.max(0, Math.min(320, ((temp - 10) / 60) * 320))} ${140 - tempAct * 110})`}>
            <circle r="6" fill="#dc2626" />
            <circle r="10" fill="none" stroke="#dc2626" strokeOpacity={0.4} strokeWidth={1.5}>
              <animate attributeName="r" values="10;18;10" dur="1.6s" repeatCount="indefinite" />
              <animate attributeName="stroke-opacity" values="0.4;0;0.4" dur="1.6s" repeatCount="indefinite" />
            </circle>
          </g>
          {/* X 轴标 */}
          <text x="0" y="156" fontSize="10" fill="#64748b">10℃</text>
          <text x="160" y="156" fontSize="10" fill="#64748b" textAnchor="middle">40℃</text>
          <text x="320" y="156" fontSize="10" fill="#64748b" textAnchor="end">70℃</text>
        </g>
      </svg>
      <div className="snowy-bio-illust__legend">
        <span><i style={{ background: '#15803d' }} />酶</span>
        <span><i style={{ background: '#fb7185' }} />底物</span>
        <span><i style={{ background: '#f97316' }} />产物</span>
        <span><i style={{ background: '#dc2626' }} />当前活性</span>
      </div>
      <div className="snowy-bio-illust__metrics">
        <div className="snowy-bio-illust__metric"><span>温度</span><b>{temp.toFixed(1)} ℃</b></div>
        <div className="snowy-bio-illust__metric"><span>pH</span><b>{ph.toFixed(2)}</b></div>
        <div className="snowy-bio-illust__metric"><span>相对活性</span><b>{(activity * 100).toFixed(0)} %</b></div>
        <div className="snowy-bio-illust__metric"><span>最适温度</span><b>{optimalTemp} ℃</b></div>
      </div>
    </div>
  );
}
