/**
 * Snowy v6 · 生物插画 · 光合作用
 *
 * 渲染一片叶子 + 叶绿体，箭头表示光、CO₂、H₂O 输入与 O₂、葡萄糖输出；
 * 接收 values.light_intensity / co2 / temperature，反映在光合速率曲线高亮位置。
 */

'use client';

import React from 'react';
import type { IllustrationProps } from './shared';
import { numVal } from './shared';
import './illustrations.css';

export default function Photosynthesis({ values, topic = '光合作用' }: IllustrationProps) {
  const light = numVal(values, 'light_intensity', 60);
  const co2 = numVal(values, 'co2', 0.04);
  const temp = numVal(values, 'temperature', 25);

  const rate = Math.min(100, light * 0.7 + co2 * 800 + (temp > 35 ? 95 - (temp - 35) * 6 : Math.min(95, temp * 2.4)));
  const ratePct = Math.max(5, Math.min(100, rate));

  return (
    <div className="snowy-bio-illust">
      <div className="snowy-bio-illust__title">{topic}</div>
      <div className="snowy-bio-illust__sub">光能 + CO₂ + H₂O → 葡萄糖 + O₂</div>
      <svg className="snowy-bio-illust__svg" viewBox="0 0 480 260" role="img" aria-label="光合作用示意图">
        <defs>
          <radialGradient id="psSun" cx="50%" cy="50%" r="50%">
            <stop offset="0%" stopColor="#fde68a" />
            <stop offset="100%" stopColor="#fbbf24" />
          </radialGradient>
          <linearGradient id="psLeaf" x1="0" x2="1" y1="0" y2="1">
            <stop offset="0%" stopColor="#86efac" />
            <stop offset="100%" stopColor="#16a34a" />
          </linearGradient>
          <radialGradient id="psChloro" cx="50%" cy="50%" r="50%">
            <stop offset="0%" stopColor="#bbf7d0" />
            <stop offset="100%" stopColor="#15803d" />
          </radialGradient>
        </defs>

        {/* 太阳 */}
        <g transform="translate(70 60)">
          <circle r="30" fill="url(#psSun)" />
          {Array.from({ length: 10 }).map((_, i) => {
            const a = (i / 10) * Math.PI * 2;
            return (
              <line
                key={i}
                x1={Math.cos(a) * 36}
                y1={Math.sin(a) * 36}
                x2={Math.cos(a) * 50}
                y2={Math.sin(a) * 50}
                stroke="#f59e0b"
                strokeWidth={3}
                strokeLinecap="round"
                opacity={0.5 + (light / 200)}
              />
            );
          })}
        </g>

        {/* 光线 */}
        {Array.from({ length: 5 }).map((_, i) => (
          <line
            key={i}
            x1={110 + i * 8}
            y1={80 + i * 8}
            x2={210 + i * 8}
            y2={140 + i * 4}
            stroke="#fbbf24"
            strokeWidth={2}
            strokeDasharray="6 6"
            opacity={0.35 + light / 280}
            className="snowy-bio-anim-flow"
          />
        ))}

        {/* 叶子 */}
        <g transform="translate(280 130)">
          <path
            d="M -90 0 C -90 -55 -10 -90 60 -70 C 80 -10 80 30 60 70 C -10 80 -90 55 -90 0 Z"
            fill="url(#psLeaf)"
            stroke="#15803d"
            strokeWidth={2}
          />
          {/* 主叶脉 */}
          <path d="M -90 0 Q 0 -10 60 -70" stroke="#15803d" strokeWidth={1.5} fill="none" opacity={0.6} />
          <path d="M -90 0 Q 0 10 60 70" stroke="#15803d" strokeWidth={1.5} fill="none" opacity={0.6} />
          {/* 叶绿体 */}
          {[
            { x: -50, y: -25, r: 12 },
            { x: -10, y: -10, r: 14 },
            { x: 20, y: 20, r: 12 },
            { x: -30, y: 30, r: 11 },
            { x: 30, y: -30, r: 10 },
          ].map((c, i) => (
            <ellipse key={i} cx={c.x} cy={c.y} rx={c.r * 1.3} ry={c.r * 0.9} fill="url(#psChloro)" opacity={0.85}>
              <title>叶绿体</title>
            </ellipse>
          ))}
        </g>

        {/* CO₂ / H₂O 输入 */}
        <g transform="translate(60 200)">
          <circle r="14" fill="#dbeafe" stroke="#1d4ed8" strokeWidth={1.5} />
          <text x={0} y={4} fontSize={11} textAnchor="middle" fill="#1d4ed8" fontWeight={600}>CO₂</text>
        </g>
        <line x1={75} y1={195} x2={195} y2={155} stroke="#1d4ed8" strokeWidth={2} strokeDasharray="5 4" className="snowy-bio-anim-flow" />
        <g transform="translate(60 240)">
          <circle r="14" fill="#cffafe" stroke="#0e7490" strokeWidth={1.5} />
          <text x={0} y={4} fontSize={11} textAnchor="middle" fill="#0e7490" fontWeight={600}>H₂O</text>
        </g>
        <line x1={75} y1={240} x2={195} y2={170} stroke="#0e7490" strokeWidth={2} strokeDasharray="5 4" className="snowy-bio-anim-flow" />

        {/* O₂ / 葡萄糖 输出 */}
        <line x1={365} y1={140} x2={445} y2={90} stroke="#ea580c" strokeWidth={2} strokeDasharray="5 4" className="snowy-bio-anim-flow" />
        <g transform="translate(445 80)">
          <circle r="14" fill="#fee2e2" stroke="#dc2626" strokeWidth={1.5} />
          <text x={0} y={4} fontSize={11} textAnchor="middle" fill="#dc2626" fontWeight={700}>O₂</text>
        </g>
        <line x1={365} y1={170} x2={445} y2={220} stroke="#92400e" strokeWidth={2} strokeDasharray="5 4" className="snowy-bio-anim-flow" />
        <g transform="translate(445 220)">
          <circle r="16" fill="#fef3c7" stroke="#92400e" strokeWidth={1.5} />
          <text x={0} y={4} fontSize={10} textAnchor="middle" fill="#92400e" fontWeight={600}>C₆H₁₂O₆</text>
        </g>

        {/* 速率指示条 */}
        <g transform="translate(40 14)">
          <rect width="160" height="10" rx="5" fill="#e5e7eb" />
          <rect width={(160 * ratePct) / 100} height="10" rx="5" fill="#16a34a">
            <animate attributeName="width" to={(160 * ratePct) / 100} dur="0.5s" fill="freeze" />
          </rect>
          <text x={170} y={9} fontSize={11} fill="#15803d" fontWeight={600}>{ratePct.toFixed(0)}%</text>
        </g>
      </svg>

      <div className="snowy-bio-illust__legend">
        <span><i style={{ background: '#fbbf24' }} />光照</span>
        <span><i style={{ background: '#1d4ed8' }} />CO₂</span>
        <span><i style={{ background: '#0e7490' }} />H₂O</span>
        <span><i style={{ background: '#16a34a' }} />叶绿体</span>
        <span><i style={{ background: '#dc2626' }} />O₂</span>
        <span><i style={{ background: '#92400e' }} />葡萄糖</span>
      </div>
      <div className="snowy-bio-illust__metrics">
        <div className="snowy-bio-illust__metric"><span>光照强度</span><b>{light.toFixed(0)} %</b></div>
        <div className="snowy-bio-illust__metric"><span>CO₂ 浓度</span><b>{(co2 * 100).toFixed(2)} %</b></div>
        <div className="snowy-bio-illust__metric"><span>温度</span><b>{temp.toFixed(1)} ℃</b></div>
        <div className="snowy-bio-illust__metric"><span>相对光合速率</span><b>{ratePct.toFixed(0)} %</b></div>
      </div>
    </div>
  );
}
