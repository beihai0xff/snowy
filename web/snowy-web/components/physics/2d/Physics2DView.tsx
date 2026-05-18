/**
 * Snowy v9 · 物理 2D 教学视图
 *
 * 设计理念：
 *  - 与 3D 视图并存，作为「课本风格」的教学呈现；
 *  - 不依赖 Rapier 仿真，使用解析公式驱动，物理量绝对准确；
 *  - SVG 渲染：坐标轴 / 刻度 / 矢量箭头 / 关键点标注 / 数据面板；
 *  - 内部 RAF 推进时间 t，与外部 running/speed 联动；
 *  - props/sceneType 变化时自动重置 t=0。
 *
 * 支持场景：projectile / motion / force / spring / collision / orbit
 */

'use client';

import React, { useEffect, useMemo, useRef, useState } from 'react';
import type { SceneKind } from '../r3f/lib/types';
import { numberValue, sceneKindOf } from '../r3f/lib/types';
import './physics2d.css';

interface Physics2DViewProps {
  sceneType: string;
  props: Record<string, number>;
  running: boolean;
  /** 倍速：1.0 = 实时 */
  playbackSpeed: number;
  /** 父级触发重置时 +1 */
  resetSignal?: number;
  /** 步进命令：id 变化时，t += deltaSeconds（受 [0, +∞) 约束） */
  stepCommand?: { id: number; deltaSeconds: number };
  /** 历史轨迹（灰色对比线）—— 由父级提供的解析采样 */
  ghostTrails?: Array<{ id: string; points: { x: number; y: number }[]; label?: string }>;
  /** 是否显示公式条带（默认 true） */
  showFormulaStrip?: boolean;
}

/** 视口配置：以"物理坐标 (米)"为单位，自动适配 SVG 尺寸 */
interface ViewBox {
  xMin: number; xMax: number; yMin: number; yMax: number;
}

const SVG_W = 640;
const SVG_H = 420;
const MARGIN = { top: 28, right: 220, bottom: 48, left: 56 };

function makeScales(vb: ViewBox) {
  const innerW = SVG_W - MARGIN.left - MARGIN.right;
  const innerH = SVG_H - MARGIN.top - MARGIN.bottom;
  const sx = (x: number) => MARGIN.left + ((x - vb.xMin) / (vb.xMax - vb.xMin)) * innerW;
  const sy = (y: number) => MARGIN.top + (1 - (y - vb.yMin) / (vb.yMax - vb.yMin)) * innerH;
  return { sx, sy, innerW, innerH };
}

function niceTicks(min: number, max: number, target = 8): number[] {
  const range = max - min;
  if (range <= 0) return [min];
  const rough = range / target;
  const mag = Math.pow(10, Math.floor(Math.log10(rough)));
  const norm = rough / mag;
  const step = (norm < 1.5 ? 1 : norm < 3 ? 2 : norm < 7 ? 5 : 10) * mag;
  const ticks: number[] = [];
  const start = Math.ceil(min / step) * step;
  for (let v = start; v <= max + 1e-9; v += step) ticks.push(Number(v.toFixed(10)));
  return ticks;
}

/* ───────────────────────── Axes ───────────────────────── */
const Axes: React.FC<{ vb: ViewBox; xLabel: string; yLabel: string }> = ({ vb, xLabel, yLabel }) => {
  const { sx, sy } = makeScales(vb);
  const xTicks = niceTicks(vb.xMin, vb.xMax, 8);
  const yTicks = niceTicks(vb.yMin, vb.yMax, 6);
  // 让坐标原点在合理位置（y=0 通常是地面）
  const xAxisY = sy(Math.max(vb.yMin, Math.min(vb.yMax, 0)));
  const yAxisX = sx(Math.max(vb.xMin, Math.min(vb.xMax, 0)));

  return (
    <g className="snowy-p2d-axis">
      {/* 网格 */}
      {xTicks.map((t) => (
        <g key={`gx-${t}`} className="tick tick-major">
          <line x1={sx(t)} x2={sx(t)} y1={sy(vb.yMin)} y2={sy(vb.yMax)} />
        </g>
      ))}
      {yTicks.map((t) => (
        <g key={`gy-${t}`} className="tick tick-major">
          <line x1={sx(vb.xMin)} x2={sx(vb.xMax)} y1={sy(t)} y2={sy(t)} />
        </g>
      ))}
      {/* x 轴 */}
      <line x1={sx(vb.xMin)} x2={sx(vb.xMax)} y1={xAxisY} y2={xAxisY} />
      {xTicks.map((t) => (
        <g key={`tx-${t}`} className="tick" transform={`translate(${sx(t)},${xAxisY})`}>
          <line y1={0} y2={4} />
          <text y={16} textAnchor="middle">{t}</text>
        </g>
      ))}
      <text x={sx(vb.xMax) + 6} y={xAxisY + 4} className="axis-label">{xLabel}</text>
      {/* y 轴 */}
      <line x1={yAxisX} x2={yAxisX} y1={sy(vb.yMin)} y2={sy(vb.yMax)} />
      {yTicks.map((t) => (
        <g key={`ty-${t}`} className="tick" transform={`translate(${yAxisX},${sy(t)})`}>
          <line x1={-4} x2={0} />
          <text x={-6} y={3} textAnchor="end">{t}</text>
        </g>
      ))}
      <text x={yAxisX - 8} y={sy(vb.yMax) - 8} className="axis-label" textAnchor="end">{yLabel}</text>
    </g>
  );
};

/* ─────────────────────── Vector ──────────────────────── */
interface VectorProps {
  x1: number; y1: number; x2: number; y2: number;
  kind: 'v' | 'vx' | 'vy' | 'a' | 'f';
  label?: string;
  sx: (x: number) => number;
  sy: (y: number) => number;
}
const Vector: React.FC<VectorProps> = ({ x1, y1, x2, y2, kind, label, sx, sy }) => {
  const px1 = sx(x1), py1 = sy(y1), px2 = sx(x2), py2 = sy(y2);
  const len = Math.hypot(px2 - px1, py2 - py1);
  if (len < 4) return null;
  const ux = (px2 - px1) / len, uy = (py2 - py1) / len;
  const ah = 8; // arrowhead length
  const aw = 4; // arrowhead half-width
  const bx = px2 - ux * ah;
  const by = py2 - uy * ah;
  const nx = -uy, ny = ux;
  const tipPath = `M ${px2} ${py2} L ${bx + nx * aw} ${by + ny * aw} L ${bx - nx * aw} ${by - ny * aw} Z`;
  return (
    <g>
      <line className={`snowy-p2d-vec snowy-p2d-vec--${kind}`} x1={px1} y1={py1} x2={bx} y2={by} strokeWidth={2.2} />
      <path className={`snowy-p2d-vec snowy-p2d-vec--${kind}`} d={tipPath} fill="currentColor" stroke="currentColor" />
      {label && (
        <text className={`snowy-p2d-vec-label snowy-p2d-vec-label--${kind}`} x={px2 + ux * 6 + nx * 6} y={py2 + uy * 6 + ny * 6 + 3}>{label}</text>
      )}
    </g>
  );
};

/* ───────────────────── KeyPointLabel ──────────────────── */
const KeyPoint: React.FC<{
  x: number; y: number; label: string; kind?: 'launch' | 'apex' | 'land' | 'default';
  sx: (x: number) => number; sy: (y: number) => number;
  labelOffset?: [number, number];
}> = ({ x, y, label, kind = 'default', sx, sy, labelOffset = [8, -8] }) => (
  <g className={`snowy-p2d-keypoint snowy-p2d-keypoint--${kind}`} transform={`translate(${sx(x)},${sy(y)})`}>
    <circle r={4} />
    <text x={labelOffset[0]} y={labelOffset[1]}>{label}</text>
  </g>
);

/* ────────────────────── Ground hatch ─────────────────── */
const Ground: React.FC<{ y: number; vb: ViewBox; sx: (x: number) => number; sy: (y: number) => number }> = ({ y, vb, sx, sy }) => {
  const px1 = sx(vb.xMin), px2 = sx(vb.xMax), py = sy(y);
  const tick = (px2 - px1) / 30;
  const lines: React.ReactElement[] = [];
  for (let i = 0; i < 30; i += 1) {
    const x = px1 + i * tick;
    lines.push(<line key={i} className="snowy-p2d-ground-hatch" x1={x} y1={py} x2={x - 6} y2={py + 8} />);
  }
  return (
    <g>
      <line className="snowy-p2d-ground" x1={px1} y1={py} x2={px2} y2={py} />
      {lines}
    </g>
  );
};

/* ──────────────── Helper: build polyline ──────────────── */
function polylineD(points: { x: number; y: number }[], sx: (x: number) => number, sy: (y: number) => number) {
  if (points.length < 2) return '';
  let d = `M ${sx(points[0].x)} ${sy(points[0].y)}`;
  for (let i = 1; i < points.length; i += 1) d += ` L ${sx(points[i].x)} ${sy(points[i].y)}`;
  return d;
}

/* ════════════════════════════════════════════════════════
 *  Scene renderers
 * ════════════════════════════════════════════════════════ */

interface SceneCommonProps {
  props: Record<string, number>;
  t: number; // 当前时刻（秒）
  ghostTrails?: Array<{ id: string; points: { x: number; y: number }[] }>;
}

/* ─── Projectile ─── */
const ProjectileScene: React.FC<SceneCommonProps> = ({ props, t, ghostTrails }) => {
  const v0 = numberValue(props, 'v0', 20);
  const angleDeg = numberValue(props, 'angle_deg', 45);
  const g = numberValue(props, 'g', 9.8);
  const theta = (angleDeg * Math.PI) / 180;
  const cosT = Math.cos(theta), sinT = Math.sin(theta);
  const vx0 = v0 * cosT;
  const vy0 = v0 * sinT;

  // 物理预测量
  const tof = (2 * vy0) / g;
  const range = (v0 * v0 * Math.sin(2 * theta)) / g;
  const hMax = (vy0 * vy0) / (2 * g);

  // 限制 t 在 [0, tof]，飞行结束后停在落地点
  const tClamp = Math.max(0, Math.min(t, tof));
  const x = vx0 * tClamp;
  const y = vy0 * tClamp - 0.5 * g * tClamp * tClamp;
  const vx = vx0;
  const vy = vy0 - g * tClamp;
  const vMag = Math.hypot(vx, vy);

  // 视口：留 15% 余量
  const vb: ViewBox = useMemo(() => ({
    xMin: -range * 0.05,
    xMax: range * 1.15,
    yMin: -hMax * 0.15,
    yMax: hMax * 1.25,
  }), [range, hMax]);
  const { sx, sy } = makeScales(vb);

  // 完整解析轨迹（用于绘制）
  const trailPoints = useMemo(() => {
    const pts: { x: number; y: number }[] = [];
    const N = 80;
    for (let i = 0; i <= N; i += 1) {
      const ti = (i / N) * tof;
      pts.push({ x: vx0 * ti, y: vy0 * ti - 0.5 * g * ti * ti });
    }
    return pts;
  }, [vx0, vy0, g, tof]);

  // 已走过的轨迹（亮）+ 未走的轨迹（虚线）
  const progress = tof > 0 ? tClamp / tof : 0;
  const splitIdx = Math.floor(progress * trailPoints.length);
  const passed = trailPoints.slice(0, splitIdx + 1);
  passed[passed.length - 1] = { x, y };

  // 矢量长度缩放：让箭头视觉合理（约占视口宽度 8%）
  const visScale = (vb.xMax - vb.xMin) * 0.08 / Math.max(v0, 1);

  return (
    <>
      <Axes vb={vb} xLabel="x (m)" yLabel="y (m)" />
      <Ground y={0} vb={vb} sx={sx} sy={sy} />
      {/* 历史轨迹 */}
      {ghostTrails?.map((g) => (
        <path key={g.id} className="snowy-p2d-trail snowy-p2d-trail--ghost" d={polylineD(g.points, sx, sy)} />
      ))}
      {/* 未来轨迹（虚线参考） */}
      <path
        d={polylineD(trailPoints, sx, sy)}
        fill="none"
        stroke="#cbd5e1"
        strokeWidth={1.2}
        strokeDasharray="3 3"
      />
      {/* 已飞过的轨迹（实线） */}
      <path className="snowy-p2d-trail" d={polylineD(passed, sx, sy)} />
      {/* 关键点 */}
      <KeyPoint x={0} y={0} label="发射点" kind="launch" sx={sx} sy={sy} labelOffset={[6, 14]} />
      <KeyPoint x={range / 2} y={hMax} label={`最高点 (${(range / 2).toFixed(1)}, ${hMax.toFixed(1)})`} kind="apex" sx={sx} sy={sy} labelOffset={[8, -6]} />
      <KeyPoint x={range} y={0} label={`落点 ${range.toFixed(1)} m`} kind="land" sx={sx} sy={sy} labelOffset={[6, 14]} />
      {/* 速度矢量 */}
      <Vector
        sx={sx} sy={sy}
        x1={x} y1={y}
        x2={x + vx * visScale}
        y2={y + vy * visScale}
        kind="v"
        label={`v=${vMag.toFixed(1)}`}
      />
      <Vector
        sx={sx} sy={sy}
        x1={x} y1={y}
        x2={x + vx * visScale} y2={y}
        kind="vx"
        label={`v_x=${vx.toFixed(1)}`}
      />
      <Vector
        sx={sx} sy={sy}
        x1={x} y1={y}
        x2={x} y2={y + vy * visScale}
        kind="vy"
        label={`v_y=${vy.toFixed(1)}`}
      />
      {/* 当前小球 */}
      <circle className="snowy-p2d-ball" cx={sx(x)} cy={sy(y)} r={6} />
    </>
  );
};

/* ─── Motion (1D 匀加速) ─── */
const MotionScene: React.FC<SceneCommonProps> = ({ props, t }) => {
  const v0 = numberValue(props, 'v0', numberValue(props, 'v', 5));
  const a = numberValue(props, 'a', 0);
  const xMax = Math.max(20, v0 * 4 + Math.abs(a) * 8);
  const tClamp = Math.max(0, Math.min(t, 8));
  const x = v0 * tClamp + 0.5 * a * tClamp * tClamp;
  const v = v0 + a * tClamp;

  const vb: ViewBox = { xMin: -xMax * 0.05, xMax, yMin: -2, yMax: 4 };
  const { sx, sy } = makeScales(vb);
  const visScale = xMax * 0.05 / Math.max(Math.abs(v), 1);

  return (
    <>
      <Axes vb={vb} xLabel="x (m)" yLabel="" />
      <Ground y={0} vb={vb} sx={sx} sy={sy} />
      <Vector sx={sx} sy={sy} x1={x} y1={0.4} x2={x + v * visScale} y2={0.4} kind="v" label={`v=${v.toFixed(1)}`} />
      {Math.abs(a) > 0.01 && (
        <Vector sx={sx} sy={sy} x1={x} y1={1.2} x2={x + Math.sign(a) * Math.abs(a) * visScale * 2} y2={1.2} kind="a" label={`a=${a.toFixed(1)}`} />
      )}
      <circle className="snowy-p2d-ball" cx={sx(x)} cy={sy(0.4)} r={8} />
      <KeyPoint x={0} y={0} label="起点" kind="launch" sx={sx} sy={sy} labelOffset={[6, 14]} />
    </>
  );
};

/* ─── Force (牛顿第二定律) ─── */
const ForceScene: React.FC<SceneCommonProps> = ({ props, t }) => {
  const m = Math.max(0.1, numberValue(props, 'm', 2));
  const a = numberValue(props, 'a', 3);
  const F = m * a;
  const tClamp = Math.max(0, Math.min(t, 8));
  const x = 0.5 * a * tClamp * tClamp;
  const v = a * tClamp;
  const xMax = Math.max(15, 0.5 * Math.abs(a) * 64);

  const vb: ViewBox = { xMin: -1, xMax, yMin: -1, yMax: 4 };
  const { sx, sy } = makeScales(vb);
  const visScale = xMax * 0.05 / Math.max(Math.abs(v), Math.abs(F), 1);

  return (
    <>
      <Axes vb={vb} xLabel="x (m)" yLabel="" />
      <Ground y={0} vb={vb} sx={sx} sy={sy} />
      {/* 方块 */}
      <rect className="snowy-p2d-block" x={sx(x) - 12} y={sy(0.6) - 12} width={24} height={24} rx={2} />
      <Vector sx={sx} sy={sy} x1={x} y1={0.6} x2={x + F * visScale} y2={0.6} kind="f" label={`F=${F.toFixed(1)} N`} />
      <Vector sx={sx} sy={sy} x1={x} y1={1.6} x2={x + v * visScale} y2={1.6} kind="v" label={`v=${v.toFixed(1)}`} />
      <Vector sx={sx} sy={sy} x1={x} y1={2.4} x2={x + a * visScale} y2={2.4} kind="a" label={`a=${a.toFixed(1)}`} />
      <KeyPoint x={0} y={0} label="起点" kind="launch" sx={sx} sy={sy} labelOffset={[6, 14]} />
    </>
  );
};

/* ─── Spring (阻尼简谐振动) ─── */
const SpringScene: React.FC<SceneCommonProps> = ({ props, t }) => {
  const k = numberValue(props, 'k', 24);
  const m = Math.max(0.1, numberValue(props, 'm', 1.2));
  const A = numberValue(props, 'x', 1.4); // 初始位移
  const damping = numberValue(props, 'damping', 0.18);
  const omega0 = Math.sqrt(k / m);
  const gamma = damping / (2 * m);
  const omegaD = Math.sqrt(Math.max(0, omega0 * omega0 - gamma * gamma));
  // x(t) = A e^{-γt} cos(ω_d t)
  const x = A * Math.exp(-gamma * t) * Math.cos(omegaD * t);
  const v = A * Math.exp(-gamma * t) * (-gamma * Math.cos(omegaD * t) - omegaD * Math.sin(omegaD * t));

  const vb: ViewBox = { xMin: -A * 1.5 - 1, xMax: A * 1.5 + 1, yMin: -1.5, yMax: 1.5 };
  const { sx, sy } = makeScales(vb);
  const visScale = (vb.xMax - vb.xMin) * 0.05 / Math.max(Math.abs(v), 1);

  // 弹簧线条：从墙(-A*1.5-1, 0)到小球(x, 0)，画 8 段锯齿
  const wallX = vb.xMin + 0.1;
  const ballX = x - 0.3;
  const segs = 10;
  let springD = `M ${sx(wallX)} ${sy(0)}`;
  for (let i = 1; i <= segs; i += 1) {
    const px = wallX + (ballX - wallX) * (i / segs);
    const py = i === segs ? 0 : (i % 2 === 0 ? 0.18 : -0.18);
    springD += ` L ${sx(px)} ${sy(py)}`;
  }

  return (
    <>
      <Axes vb={vb} xLabel="x (m, 相对平衡位置)" yLabel="" />
      <rect className="snowy-p2d-wall" x={sx(vb.xMin)} y={sy(0.6)} width={sx(wallX) - sx(vb.xMin)} height={sy(-0.6) - sy(0.6)} />
      <path className="snowy-p2d-spring" d={springD} />
      <circle className="snowy-p2d-ball" cx={sx(x)} cy={sy(0)} r={10} />
      <Vector sx={sx} sy={sy} x1={x} y1={0.7} x2={x + v * visScale} y2={0.7} kind="v" label={`v=${v.toFixed(2)}`} />
      {/* 平衡位置标记 */}
      <line x1={sx(0)} y1={sy(-0.4)} x2={sx(0)} y2={sy(0.4)} stroke="#94a3b8" strokeDasharray="3 2" strokeWidth={1} />
      <text x={sx(0)} y={sy(-0.5)} textAnchor="middle" fill="#64748b" fontSize={10}>平衡位置</text>
    </>
  );
};

/* ─── Collision (1D 弹性/非弹性) ─── */
const CollisionScene: React.FC<SceneCommonProps> = ({ props, t }) => {
  const m1 = Math.max(0.2, numberValue(props, 'm1', 1.5));
  const m2 = Math.max(0.2, numberValue(props, 'm2', 1));
  const e = Math.max(0, Math.min(1, numberValue(props, 'restitution', 0.9)));
  const v1i = numberValue(props, 'v1', 4.5);
  const v2i = numberValue(props, 'v2', -2.5);
  const x1i = -6, x2i = 6;
  const r1 = 0.32 + Math.cbrt(m1) * 0.12;
  const r2 = 0.32 + Math.cbrt(m2) * 0.12;

  // 计算相遇时刻：x1i + v1i tc + r1 = x2i + v2i tc - r2
  // (v1i - v2i) tc = x2i - x1i - r1 - r2
  const denom = v1i - v2i;
  const tc = denom > 1e-3 ? (x2i - x1i - r1 - r2) / denom : Infinity;

  // 碰后速度
  const v1f = ((m1 - e * m2) * v1i + (1 + e) * m2 * v2i) / (m1 + m2);
  const v2f = ((m2 - e * m1) * v2i + (1 + e) * m1 * v1i) / (m1 + m2);

  let x1: number, x2: number, v1: number, v2: number;
  if (t < tc) {
    x1 = x1i + v1i * t; x2 = x2i + v2i * t; v1 = v1i; v2 = v2i;
  } else {
    const dt = t - tc;
    const xc1 = x1i + v1i * tc; const xc2 = x2i + v2i * tc;
    x1 = xc1 + v1f * dt; x2 = xc2 + v2f * dt; v1 = v1f; v2 = v2f;
  }

  const xRange = Math.max(10, Math.abs(x1) + 2, Math.abs(x2) + 2);
  const vb: ViewBox = { xMin: -xRange, xMax: xRange, yMin: -1.5, yMax: 2.5 };
  const { sx, sy } = makeScales(vb);
  const visScale = xRange * 0.04 / Math.max(Math.abs(v1), Math.abs(v2), 1);

  const E = 0.5 * m1 * v1 * v1 + 0.5 * m2 * v2 * v2;
  const Ei = 0.5 * m1 * v1i * v1i + 0.5 * m2 * v2i * v2i;

  return (
    <>
      <Axes vb={vb} xLabel="x (m)" yLabel="" />
      <Ground y={0} vb={vb} sx={sx} sy={sy} />
      <circle className="snowy-p2d-ball" cx={sx(x1)} cy={sy(r1)} r={r1 * 22} fill="#0ea5e9" />
      <text x={sx(x1)} y={sy(r1) + 4} textAnchor="middle" fill="#fff" fontSize={10} fontWeight={600}>m₁</text>
      <circle className="snowy-p2d-ball" cx={sx(x2)} cy={sy(r2)} r={r2 * 22} fill="#f59e0b" stroke="#b45309" />
      <text x={sx(x2)} y={sy(r2) + 4} textAnchor="middle" fill="#fff" fontSize={10} fontWeight={600}>m₂</text>
      <Vector sx={sx} sy={sy} x1={x1} y1={r1 * 2 + 0.3} x2={x1 + v1 * visScale} y2={r1 * 2 + 0.3} kind="v" label={`v₁=${v1.toFixed(2)}`} />
      <Vector sx={sx} sy={sy} x1={x2} y1={r2 * 2 + 0.3} x2={x2 + v2 * visScale} y2={r2 * 2 + 0.3} kind="v" label={`v₂=${v2.toFixed(2)}`} />
      <text x={sx(0)} y={sy(vb.yMax) + 14} textAnchor="middle" fill="#64748b" fontSize={11}>
        动能：E={E.toFixed(2)} J（初 {Ei.toFixed(2)} J，损失 {((Ei - E) / Ei * 100).toFixed(1)}%）
      </text>
    </>
  );
};

/* ─── Orbit (二维椭圆/圆轨道近似) ─── */
const OrbitScene: React.FC<SceneCommonProps> = ({ props, t }) => {
  const r = numberValue(props, 'orbit_radius', 3.6);
  const speed = numberValue(props, 'tangential_speed', 2.25);
  const ecc = Math.max(0, Math.min(0.6, numberValue(props, 'eccentricity', 0.18)));
  const omega = speed / r;
  // 简化为椭圆参数方程，焦点在原点附近
  const a = r;
  const b = r * Math.sqrt(1 - ecc * ecc);
  const c = a * ecc;
  const phi = omega * t;
  const x = a * Math.cos(phi) - c;
  const y = b * Math.sin(phi);

  const vb: ViewBox = { xMin: -a * 1.4 - c, xMax: a * 1.4 - c, yMin: -b * 1.4, yMax: b * 1.4 };
  const { sx, sy } = makeScales(vb);

  // 轨道椭圆
  const orbitPts: { x: number; y: number }[] = [];
  for (let i = 0; i <= 100; i += 1) {
    const p = (i / 100) * Math.PI * 2;
    orbitPts.push({ x: a * Math.cos(p) - c, y: b * Math.sin(p) });
  }

  return (
    <>
      <Axes vb={vb} xLabel="x (AU)" yLabel="y (AU)" />
      <path d={polylineD(orbitPts, sx, sy)} fill="none" stroke="#cbd5e1" strokeWidth={1} strokeDasharray="3 3" />
      {/* 中心天体 */}
      <circle cx={sx(0)} cy={sy(0)} r={10} fill="#f59e0b" stroke="#b45309" strokeWidth={1.5} />
      <text x={sx(0)} y={sy(0) + 22} textAnchor="middle" fill="#92400e" fontSize={10}>中心天体</text>
      {/* 卫星 */}
      <circle className="snowy-p2d-ball" cx={sx(x)} cy={sy(y)} r={5} />
      <text x={sx(x) + 8} y={sy(y) - 6} fill="#0369a1" fontSize={10} fontWeight={600}>r={Math.hypot(x, y).toFixed(2)}</text>
    </>
  );
};

/* ════════════════════════════════════════════════════════
 *  Main view
 * ════════════════════════════════════════════════════════ */

function buildDataPanel(kind: SceneKind, props: Record<string, number>, t: number): Array<[string, string]> {
  if (kind === 'projectile') {
    const v0 = numberValue(props, 'v0', 20);
    const angleDeg = numberValue(props, 'angle_deg', 45);
    const g = numberValue(props, 'g', 9.8);
    const theta = (angleDeg * Math.PI) / 180;
    const vx0 = v0 * Math.cos(theta), vy0 = v0 * Math.sin(theta);
    const tof = (2 * vy0) / g;
    const range = (v0 * v0 * Math.sin(2 * theta)) / g;
    const hMax = (vy0 * vy0) / (2 * g);
    const tClamp = Math.max(0, Math.min(t, tof));
    const x = vx0 * tClamp, y = vy0 * tClamp - 0.5 * g * tClamp * tClamp;
    const vy = vy0 - g * tClamp;
    return [
      ['t', `${tClamp.toFixed(2)} s`],
      ['x', `${x.toFixed(2)} m`],
      ['y', `${y.toFixed(2)} m`],
      ['v_x', `${vx0.toFixed(2)} m/s`],
      ['v_y', `${vy.toFixed(2)} m/s`],
      ['__divider__', ''],
      ['射程 R', `${range.toFixed(2)} m`],
      ['最大高度 H', `${hMax.toFixed(2)} m`],
      ['飞行时间 T', `${tof.toFixed(2)} s`],
    ];
  }
  if (kind === 'motion') {
    const v0 = numberValue(props, 'v0', numberValue(props, 'v', 5));
    const a = numberValue(props, 'a', 0);
    const tc = Math.max(0, Math.min(t, 8));
    return [
      ['t', `${tc.toFixed(2)} s`],
      ['x', `${(v0 * tc + 0.5 * a * tc * tc).toFixed(2)} m`],
      ['v', `${(v0 + a * tc).toFixed(2)} m/s`],
      ['__divider__', ''],
      ['v₀', `${v0.toFixed(2)} m/s`],
      ['a', `${a.toFixed(2)} m/s²`],
    ];
  }
  if (kind === 'force') {
    const m = Math.max(0.1, numberValue(props, 'm', 2));
    const a = numberValue(props, 'a', 3);
    const F = m * a;
    const tc = Math.max(0, Math.min(t, 8));
    return [
      ['t', `${tc.toFixed(2)} s`],
      ['x', `${(0.5 * a * tc * tc).toFixed(2)} m`],
      ['v', `${(a * tc).toFixed(2)} m/s`],
      ['__divider__', ''],
      ['F = ma', `${F.toFixed(2)} N`],
      ['m', `${m.toFixed(2)} kg`],
      ['a', `${a.toFixed(2)} m/s²`],
    ];
  }
  if (kind === 'spring') {
    const k = numberValue(props, 'k', 24);
    const m = Math.max(0.1, numberValue(props, 'm', 1.2));
    const A = numberValue(props, 'x', 1.4);
    const damping = numberValue(props, 'damping', 0.18);
    const omega0 = Math.sqrt(k / m);
    const T = (2 * Math.PI) / omega0;
    const gamma = damping / (2 * m);
    const x = A * Math.exp(-gamma * t) * Math.cos(Math.sqrt(Math.max(0, omega0 * omega0 - gamma * gamma)) * t);
    return [
      ['t', `${t.toFixed(2)} s`],
      ['x', `${x.toFixed(3)} m`],
      ['__divider__', ''],
      ['ω₀ = √(k/m)', `${omega0.toFixed(2)} rad/s`],
      ['周期 T', `${T.toFixed(2)} s`],
      ['振幅 A', `${A.toFixed(2)} m`],
    ];
  }
  if (kind === 'collision') {
    const m1 = Math.max(0.2, numberValue(props, 'm1', 1.5));
    const m2 = Math.max(0.2, numberValue(props, 'm2', 1));
    const e = numberValue(props, 'restitution', 0.9);
    const v1i = numberValue(props, 'v1', 4.5);
    const v2i = numberValue(props, 'v2', -2.5);
    const v1f = ((m1 - e * m2) * v1i + (1 + e) * m2 * v2i) / (m1 + m2);
    const v2f = ((m2 - e * m1) * v2i + (1 + e) * m1 * v1i) / (m1 + m2);
    return [
      ['m₁ / m₂', `${m1.toFixed(1)} / ${m2.toFixed(1)} kg`],
      ['v₁ᵢ / v₂ᵢ', `${v1i.toFixed(2)} / ${v2i.toFixed(2)}`],
      ['__divider__', ''],
      ['v₁f (预测)', `${v1f.toFixed(2)} m/s`],
      ['v₂f (预测)', `${v2f.toFixed(2)} m/s`],
      ['恢复系数 e', `${e.toFixed(2)}`],
    ];
  }
  if (kind === 'orbit') {
    const r = numberValue(props, 'orbit_radius', 3.6);
    const speed = numberValue(props, 'tangential_speed', 2.25);
    const T = (2 * Math.PI * r) / speed;
    return [
      ['t', `${t.toFixed(2)} s`],
      ['r', `${r.toFixed(2)} AU`],
      ['v_t', `${speed.toFixed(2)}`],
      ['周期 T', `${T.toFixed(2)} s`],
    ];
  }
  return [['t', `${t.toFixed(2)} s`]];
}

/**
 * 构建公式条带：每行 HTML，substituted 值用 <b>...</b> 高亮。
 * 教学目标：让公式和数字"对应"起来，看到每一项的含义。
 */
function buildFormulaStrip(kind: SceneKind, props: Record<string, number>, t: number): string[] | null {
  const b = (v: string | number) => `<b>${typeof v === 'number' ? v.toFixed(2) : v}</b>`;
  if (kind === 'projectile') {
    const v0 = numberValue(props, 'v0', 20);
    const angleDeg = numberValue(props, 'angle_deg', 45);
    const g = numberValue(props, 'g', 9.8);
    const theta = (angleDeg * Math.PI) / 180;
    const sinT = Math.sin(theta), cosT = Math.cos(theta);
    const tof = (2 * v0 * sinT) / g;
    const tc = Math.max(0, Math.min(t, tof));
    const x = v0 * cosT * tc;
    const y = v0 * sinT * tc - 0.5 * g * tc * tc;
    return [
      `x = v₀ cosθ · t = ${b(v0)} × ${b(cosT.toFixed(3))} × ${b(tc)} = ${b(x)} m`,
      `y = v₀ sinθ · t − ½ g t² = ${b((v0 * sinT * tc).toFixed(2))} − ${b((0.5 * g * tc * tc).toFixed(2))} = ${b(y)} m`,
    ];
  }
  if (kind === 'motion') {
    const v0 = numberValue(props, 'v0', numberValue(props, 'v', 5));
    const a = numberValue(props, 'a', 0);
    const tc = Math.max(0, Math.min(t, 8));
    const v = v0 + a * tc;
    const x = v0 * tc + 0.5 * a * tc * tc;
    return [
      `v = v₀ + a · t = ${b(v0)} + ${b(a)} × ${b(tc)} = ${b(v)} m/s`,
      `x = v₀ · t + ½ a t² = ${b(v0 * tc)} + ${b(0.5 * a * tc * tc)} = ${b(x)} m`,
    ];
  }
  if (kind === 'force') {
    const m = Math.max(0.1, numberValue(props, 'm', 2));
    const a = numberValue(props, 'a', 3);
    return [
      `F = m · a = ${b(m)} × ${b(a)} = ${b(m * a)} N`,
      `x = ½ a t² = ½ × ${b(a)} × ${b((t * t).toFixed(2))} = ${b(0.5 * a * t * t)} m`,
    ];
  }
  if (kind === 'spring') {
    const k = numberValue(props, 'k', 24);
    const m = Math.max(0.1, numberValue(props, 'm', 1.2));
    const omega0 = Math.sqrt(k / m);
    const T = (2 * Math.PI) / omega0;
    return [
      `ω₀ = √(k/m) = √(${b(k)}/${b(m)}) = ${b(omega0)} rad/s`,
      `T = 2π/ω₀ = ${b(T)} s`,
    ];
  }
  if (kind === 'collision') {
    const m1 = Math.max(0.2, numberValue(props, 'm1', 1.5));
    const m2 = Math.max(0.2, numberValue(props, 'm2', 1));
    const e = numberValue(props, 'restitution', 0.9);
    const v1i = numberValue(props, 'v1', 4.5);
    const v2i = numberValue(props, 'v2', -2.5);
    const v1f = ((m1 - e * m2) * v1i + (1 + e) * m2 * v2i) / (m1 + m2);
    return [
      `动量守恒：m₁v₁ + m₂v₂ = ${b(m1 * v1i + m2 * v2i)}`,
      `v₁' = [(m₁−em₂)v₁ + (1+e)m₂v₂] / (m₁+m₂) = ${b(v1f)} m/s`,
    ];
  }
  if (kind === 'orbit') {
    const r = numberValue(props, 'orbit_radius', 3.6);
    const speed = numberValue(props, 'tangential_speed', 2.25);
    return [
      `T = 2πr / v = ${b((2 * Math.PI * r) / speed)} s`,
    ];
  }
  return null;
}

/**
 * 为「保存对比」功能采样某场景在给定 props 下的解析轨迹。
 * 目前仅 projectile 支持完整 2D 轨迹采样；其它场景返回空数组。
 */
export function sampleAnalyticalTrail(sceneType: string, props: Record<string, number>): { x: number; y: number }[] {
  const kind = sceneKindOf(sceneType);
  if (kind === 'projectile') {
    const v0 = numberValue(props, 'v0', 20);
    const angleDeg = numberValue(props, 'angle_deg', 45);
    const g = numberValue(props, 'g', 9.8);
    const theta = (angleDeg * Math.PI) / 180;
    const vx0 = v0 * Math.cos(theta);
    const vy0 = v0 * Math.sin(theta);
    const tof = (2 * vy0) / g;
    const pts: { x: number; y: number }[] = [];
    const N = 80;
    for (let i = 0; i <= N; i += 1) {
      const ti = (i / N) * tof;
      pts.push({ x: vx0 * ti, y: vy0 * ti - 0.5 * g * ti * ti });
    }
    return pts;
  }
  if (kind === 'orbit') {
    const r = numberValue(props, 'orbit_radius', 3.6);
    const ecc = Math.max(0, Math.min(0.6, numberValue(props, 'eccentricity', 0.18)));
    const a = r, b2 = r * Math.sqrt(1 - ecc * ecc), c = a * ecc;
    const pts: { x: number; y: number }[] = [];
    for (let i = 0; i <= 100; i += 1) {
      const p = (i / 100) * Math.PI * 2;
      pts.push({ x: a * Math.cos(p) - c, y: b2 * Math.sin(p) });
    }
    return pts;
  }
  return [];
}

const Physics2DView: React.FC<Physics2DViewProps> = ({
  sceneType, props, running, playbackSpeed, resetSignal = 0, stepCommand, ghostTrails, showFormulaStrip = true,
}) => {
  const kind = sceneKindOf(sceneType);
  const [t, setT] = useState(0);
  const lastRef = useRef<number | null>(null);
  const rafRef = useRef<number | null>(null);

  // 重置：props 变化、resetSignal、sceneType 变化
  const propsKey = JSON.stringify(props);
  useEffect(() => {
    setT(0);
    lastRef.current = null;
  }, [propsKey, sceneType, resetSignal]);

  // 步进命令（pause 时可用）
  const lastStepIdRef = useRef<number | undefined>(undefined);
  useEffect(() => {
    if (!stepCommand) return;
    if (lastStepIdRef.current === stepCommand.id) return;
    lastStepIdRef.current = stepCommand.id;
    setT((prev) => Math.max(0, prev + stepCommand.deltaSeconds));
  }, [stepCommand]);

  useEffect(() => {
    const loop = (now: number) => {
      if (!running) {
        lastRef.current = now;
        rafRef.current = requestAnimationFrame(loop);
        return;
      }
      if (lastRef.current == null) lastRef.current = now;
      const dt = Math.min(0.05, (now - lastRef.current) / 1000);
      lastRef.current = now;
      setT((prev) => prev + dt * playbackSpeed);
      rafRef.current = requestAnimationFrame(loop);
    };
    rafRef.current = requestAnimationFrame(loop);
    return () => {
      if (rafRef.current != null) cancelAnimationFrame(rafRef.current);
      lastRef.current = null;
    };
  }, [running, playbackSpeed]);

  const rows = useMemo(() => buildDataPanel(kind, props, t), [kind, props, t]);
  const formulaStrip = useMemo(() => (showFormulaStrip ? buildFormulaStrip(kind, props, t) : null), [showFormulaStrip, kind, props, t]);

  const sceneNode = (() => {
    switch (kind) {
      case 'projectile': return <ProjectileScene props={props} t={t} ghostTrails={ghostTrails} />;
      case 'motion': return <MotionScene props={props} t={t} />;
      case 'force': return <ForceScene props={props} t={t} />;
      case 'spring': return <SpringScene props={props} t={t} />;
      case 'collision': return <CollisionScene props={props} t={t} />;
      case 'orbit': return <OrbitScene props={props} t={t} />;
      default: return <MotionScene props={props} t={t} />;
    }
  })();

  return (
    <div className="snowy-p2d-shell" data-scene={kind}>
      <svg
        className="snowy-p2d-svg"
        viewBox={`0 0 ${SVG_W} ${SVG_H}`}
        preserveAspectRatio="xMidYMid meet"
        role="img"
        aria-label="物理 2D 教学视图"
      >
        {sceneNode}
      </svg>
      <aside className="snowy-p2d-panel" aria-label="物理量">
        <div className="snowy-p2d-panel__title">📐 物理量</div>
        <div className="snowy-p2d-panel__rows">
          {rows.map(([k, v], i) => k === '__divider__' ? (
            <div key={`d-${i}`} className="snowy-p2d-panel__divider" />
          ) : (
            <React.Fragment key={k + i}>
              <span className="snowy-p2d-panel__key">{k}</span>
              <span className="snowy-p2d-panel__val">{v}</span>
            </React.Fragment>
          ))}
        </div>
      </aside>
      {formulaStrip && (
        <div className="snowy-p2d-formula-strip" aria-label="实时公式">
          {formulaStrip.map((line, i) => (
            <div key={i} className="snowy-p2d-formula-line" dangerouslySetInnerHTML={{ __html: line }} />
          ))}
        </div>
      )}
    </div>
  );
};

export default Physics2DView;





