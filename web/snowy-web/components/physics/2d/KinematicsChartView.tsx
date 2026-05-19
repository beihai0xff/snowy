/**
 * Snowy v9 · 物理 · 运动学曲线视图（x-t / v-t / a-t）
 *
 * 用解析公式生成完整 [0, T] 的曲线，红点跟随当前 t；
 * 支持多组参数叠加（灰色虚线），便于"调一调参数看变化"。
 *
 * 支持场景：projectile / motion / force / spring
 * （collision / orbit 暂用简化展示）
 */

'use client';

import React, { useEffect, useMemo, useRef, useState } from 'react';
import type { SceneKind } from '../r3f/lib/types'; // eslint-disable-line @typescript-eslint/no-unused-vars
import { numberValue, sceneKindOf } from '../r3f/lib/types';

interface KinematicsSample {
  t: number; x: number; y?: number; v: number; a: number;
}

/** 历史叠加曲线：与当前曲线同结构 */
export interface OverlaySeries {
  id: string;
  label?: string;
  samples: KinematicsSample[];
}

interface Props {
  sceneType: string;
  props: Record<string, number>;
  running: boolean;
  playbackSpeed: number;
  resetSignal?: number;
  overlays?: OverlaySeries[];
}

/** 为某场景生成 [0, T] 的解析采样 */
export function sampleKinematics(sceneType: string, props: Record<string, number>, N = 120): { samples: KinematicsSample[]; T: number } {
  const kind = sceneKindOf(sceneType);

  if (kind === 'projectile') {
    const v0 = numberValue(props, 'v0', 20);
    const angleDeg = numberValue(props, 'angle_deg', 45);
    const g = numberValue(props, 'g', 9.8);
    const theta = (angleDeg * Math.PI) / 180;
    const vx0 = v0 * Math.cos(theta), vy0 = v0 * Math.sin(theta);
    const T = (2 * vy0) / g;
    const samples: KinematicsSample[] = [];
    for (let i = 0; i <= N; i += 1) {
      const t = (i / N) * T;
      const vy = vy0 - g * t;
      const v = Math.hypot(vx0, vy);
      samples.push({
        t,
        x: vx0 * t,
        y: vy0 * t - 0.5 * g * t * t,
        v,
        a: g, // 加速度大小（向下）
      });
    }
    return { samples, T };
  }

  if (kind === 'motion') {
    const v0 = numberValue(props, 'v0', numberValue(props, 'v', 5));
    const a = numberValue(props, 'a', 0);
    const T = 8;
    const samples: KinematicsSample[] = [];
    for (let i = 0; i <= N; i += 1) {
      const t = (i / N) * T;
      samples.push({ t, x: v0 * t + 0.5 * a * t * t, v: v0 + a * t, a });
    }
    return { samples, T };
  }

  if (kind === 'force') {
    const a = numberValue(props, 'a', 3);
    const T = 8;
    const samples: KinematicsSample[] = [];
    for (let i = 0; i <= N; i += 1) {
      const t = (i / N) * T;
      samples.push({ t, x: 0.5 * a * t * t, v: a * t, a });
    }
    return { samples, T };
  }

  if (kind === 'spring') {
    const k = numberValue(props, 'k', 24);
    const m = Math.max(0.1, numberValue(props, 'm', 1.2));
    const A = numberValue(props, 'x', 1.4);
    const damping = numberValue(props, 'damping', 0.18);
    const omega0 = Math.sqrt(k / m);
    const gamma = damping / (2 * m);
    const omegaD = Math.sqrt(Math.max(0, omega0 * omega0 - gamma * gamma));
    const T = Math.max(6, (4 * Math.PI) / Math.max(omega0, 0.5));
    const samples: KinematicsSample[] = [];
    const dense = N * 2;
    for (let i = 0; i <= dense; i += 1) {
      const t = (i / dense) * T;
      const env = Math.exp(-gamma * t);
      const cos = Math.cos(omegaD * t), sin = Math.sin(omegaD * t);
      const x = A * env * cos;
      const v = A * env * (-gamma * cos - omegaD * sin);
      const a = A * env * ((gamma * gamma - omegaD * omegaD) * cos + 2 * gamma * omegaD * sin);
      samples.push({ t, x, v, a });
    }
    return { samples, T };
  }

  if (kind === 'orbit') {
    const r = numberValue(props, 'orbit_radius', 3.6);
    const speed = numberValue(props, 'tangential_speed', 2.25);
    const ecc = Math.max(0, Math.min(0.6, numberValue(props, 'eccentricity', 0.18)));
    const omega = speed / r;
    const T = (2 * Math.PI) / Math.max(omega, 0.1);
    const a = r, b = r * Math.sqrt(1 - ecc * ecc);
    const samples: KinematicsSample[] = [];
    for (let i = 0; i <= N; i += 1) {
      const t = (i / N) * T;
      const phi = omega * t;
      const x = a * Math.cos(phi);
      const y = b * Math.sin(phi);
      const vx = -a * omega * Math.sin(phi);
      const vy = b * omega * Math.cos(phi);
      const v = Math.hypot(vx, vy);
      samples.push({ t, x, y, v, a: v * v / r });
    }
    return { samples, T };
  }

  // collision: 简化只显示总动能曲线
  if (kind === 'collision') {
    const m1 = Math.max(0.2, numberValue(props, 'm1', 1.5));
    const m2 = Math.max(0.2, numberValue(props, 'm2', 1));
    const e = numberValue(props, 'restitution', 0.9);
    const v1i = numberValue(props, 'v1', 4.5);
    const v2i = numberValue(props, 'v2', -2.5);
    const v1f = ((m1 - e * m2) * v1i + (1 + e) * m2 * v2i) / (m1 + m2);
    const v2f = ((m2 - e * m1) * v2i + (1 + e) * m1 * v1i) / (m1 + m2);
    const tc = 1.0;
    const T = 2.5;
    const samples: KinematicsSample[] = [];
    for (let i = 0; i <= N; i += 1) {
      const t = (i / N) * T;
      const v1 = t < tc ? v1i : v1f;
      const v2 = t < tc ? v2i : v2f;
      samples.push({ t, x: 0, v: v1 - v2, a: 0 });
      samples[samples.length - 1].y = 0.5 * m1 * v1 * v1 + 0.5 * m2 * v2 * v2;
    }
    return { samples, T };
  }

  return { samples: [], T: 1 };
}

/* ─── 单个子图 ─── */
interface MiniChartProps {
  title: string;
  yLabel: string;
  series: Array<{ t: number; v: number }>;
  overlays?: Array<{ id: string; samples: Array<{ t: number; v: number }> }>;
  tCurrent: number;
  color: string;
  width: number;
  height: number;
}
const MiniChart: React.FC<MiniChartProps> = ({ title, yLabel, series, overlays, tCurrent, color, width, height }) => {
  if (series.length < 2) return null;
  const m = { top: 18, right: 12, bottom: 22, left: 44 };
  const W = width - m.left - m.right;
  const H = height - m.top - m.bottom;
  const tMin = series[0].t, tMax = series[series.length - 1].t;
  const allV = [...series.map((s) => s.v), ...(overlays?.flatMap((o) => o.samples.map((s) => s.v)) ?? [])];
  let vMin = Math.min(...allV), vMax = Math.max(...allV);
  if (vMin === vMax) { vMin -= 1; vMax += 1; }
  const pad = (vMax - vMin) * 0.1;
  vMin -= pad; vMax += pad;
  const sx = (t: number) => m.left + ((t - tMin) / (tMax - tMin || 1)) * W;
  const sy = (v: number) => m.top + (1 - (v - vMin) / (vMax - vMin)) * H;
  const pathD = (pts: Array<{ t: number; v: number }>) => {
    let d = `M ${sx(pts[0].t)} ${sy(pts[0].v)}`;
    for (let i = 1; i < pts.length; i += 1) d += ` L ${sx(pts[i].t)} ${sy(pts[i].v)}`;
    return d;
  };
  // 当前值（插值）
  const cur = (() => {
    let s = series[0];
    for (let i = 1; i < series.length; i += 1) {
      if (series[i].t >= tCurrent) {
        const a = series[i - 1], b = series[i];
        const k = (tCurrent - a.t) / (b.t - a.t || 1);
        s = { t: tCurrent, v: a.v + (b.v - a.v) * k };
        break;
      }
      s = series[i];
    }
    return s;
  })();
  // y 轴刻度
  const yTicks = [vMin, (vMin + vMax) / 2, vMax];
  const xTicks = [tMin, (tMin + tMax) / 2, tMax];
  // y=0 网格线
  const zeroY = vMin < 0 && vMax > 0 ? sy(0) : null;

  return (
    <svg width={width} height={height} className="snowy-kchart">
      <text x={m.left} y={12} fill="#0369a1" fontSize={11} fontWeight={600}>{title}</text>
      <text x={width - m.right} y={12} fill="#0f172a" fontSize={11} textAnchor="end" style={{ fontVariantNumeric: 'tabular-nums' }}>
        {yLabel} = <tspan fill={color} fontWeight={700}>{cur.v.toFixed(2)}</tspan>
      </text>
      {/* 边框 */}
      <rect x={m.left} y={m.top} width={W} height={H} fill="rgba(241, 245, 249, 0.5)" stroke="#cbd5e1" strokeWidth={0.8} />
      {/* y=0 参考线 */}
      {zeroY != null && (
        <line x1={m.left} y1={zeroY} x2={m.left + W} y2={zeroY} stroke="#94a3b8" strokeDasharray="2 3" strokeWidth={0.8} />
      )}
      {/* y 刻度 */}
      {yTicks.map((v, i) => (
        <g key={i}>
          <line x1={m.left - 3} y1={sy(v)} x2={m.left} y2={sy(v)} stroke="#94a3b8" />
          <text x={m.left - 5} y={sy(v) + 3} fontSize={9} fill="#64748b" textAnchor="end" fontFamily="Menlo, monospace">{v.toFixed(1)}</text>
        </g>
      ))}
      {/* x 刻度 */}
      {xTicks.map((t, i) => (
        <g key={i}>
          <line x1={sx(t)} y1={m.top + H} x2={sx(t)} y2={m.top + H + 3} stroke="#94a3b8" />
          <text x={sx(t)} y={m.top + H + 14} fontSize={9} fill="#64748b" textAnchor="middle" fontFamily="Menlo, monospace">{t.toFixed(1)}</text>
        </g>
      ))}
      <text x={m.left + W / 2} y={height - 4} fontSize={10} fill="#475569" textAnchor="middle">t (s)</text>
      {/* 历史叠加 */}
      {overlays?.map((o) => (
        <path key={o.id} d={pathD(o.samples)} fill="none" stroke="#94a3b8" strokeWidth={1.2} strokeDasharray="4 3" opacity={0.6} />
      ))}
      {/* 主曲线 */}
      <path d={pathD(series)} fill="none" stroke={color} strokeWidth={1.8} />
      {/* 当前点 */}
      <line x1={sx(cur.t)} y1={m.top} x2={sx(cur.t)} y2={m.top + H} stroke="#ef4444" strokeDasharray="2 2" strokeWidth={0.8} />
      <circle cx={sx(cur.t)} cy={sy(cur.v)} r={4} fill="#ef4444" stroke="#fff" strokeWidth={1.5} />
    </svg>
  );
};

/* ─── 主组件 ─── */
const KinematicsChartView: React.FC<Props> = ({ sceneType, props, running, playbackSpeed, resetSignal = 0, overlays = [] }) => {
  const kind = sceneKindOf(sceneType);
  const { samples, T } = useMemo(() => sampleKinematics(sceneType, props), [sceneType, props]);
  const [t, setT] = useState(0);
  const lastRef = useRef<number | null>(null);
  const propsKey = JSON.stringify(props);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setT(0);
    lastRef.current = null;
  }, [propsKey, sceneType, resetSignal]);

  useEffect(() => {
    let raf: number;
    const loop = (now: number) => {
      if (!running) { lastRef.current = now; raf = requestAnimationFrame(loop); return; }
      if (lastRef.current == null) lastRef.current = now;
      const dt = Math.min(0.05, (now - lastRef.current) / 1000);
      lastRef.current = now;
      setT((prev) => {
        const nv = prev + dt * playbackSpeed;
        return nv > T ? 0 : nv; // 自动循环
      });
      raf = requestAnimationFrame(loop);
    };
    raf = requestAnimationFrame(loop);
    return () => cancelAnimationFrame(raf);
  }, [running, playbackSpeed, T]);

  // 抽取每个变量的时间序列
  const xSeries = samples.map((s) => ({ t: s.t, v: s.x }));
  const ySeries = samples[0]?.y !== undefined ? samples.map((s) => ({ t: s.t, v: s.y as number })) : null;
  const vSeries = samples.map((s) => ({ t: s.t, v: s.v }));
  const aSeries = samples.map((s) => ({ t: s.t, v: s.a }));
  const xOverlays = overlays.map((o) => ({ id: o.id, samples: o.samples.map((s) => ({ t: s.t, v: s.x })) }));
  const vOverlays = overlays.map((o) => ({ id: o.id, samples: o.samples.map((s) => ({ t: s.t, v: s.v })) }));
  const aOverlays = overlays.map((o) => ({ id: o.id, samples: o.samples.map((s) => ({ t: s.t, v: s.a })) }));

  return (
    <div className="snowy-kchart-shell">
      <div className="snowy-kchart-grid">
        <MiniChart title="位置–时间 (x–t)" yLabel="x" series={xSeries} overlays={xOverlays} tCurrent={t} color="#0ea5e9" width={420} height={170} />
        {ySeries && (
          <MiniChart title="位置–时间 (y–t)" yLabel="y" series={ySeries} tCurrent={t} color="#10b981" width={420} height={170} />
        )}
        <MiniChart title="速度–时间 (v–t)" yLabel="v" series={vSeries} overlays={vOverlays} tCurrent={t} color="#ef4444" width={420} height={170} />
        <MiniChart title="加速度–时间 (a–t)" yLabel="a" series={aSeries} overlays={aOverlays} tCurrent={t} color="#a855f7" width={420} height={170} />
      </div>
      {overlays.length > 0 && (
        <div className="snowy-kchart-legend">
          叠加历史：{overlays.length} 条（灰色虚线）
        </div>
      )}
      <div className="snowy-kchart-scene-tag">场景：{kind}</div>
    </div>
  );
};

export default KinematicsChartView;



