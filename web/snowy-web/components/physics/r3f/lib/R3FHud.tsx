/**
 * Snowy v6 · R3F · HUD 浮层
 *
 * 使用 @react-three/drei 的 <Html> 在 3D 画布右上角渲染 HUD（场景标签 + 物理量），
 * 数值取自 simRef，每帧通过 useFrame 直接读取 DOM 文本节点更新，避免 React re-render。
 */

'use client';

import React, { useRef } from 'react';
import { Html } from '@react-three/drei';
import { useFrame } from '@react-three/fiber';
import { numberValue, sceneLabel, type SceneKind, type SimState } from './types';

interface Props {
  simRef: React.MutableRefObject<SimState | null>;
  kind: SceneKind;
  running: boolean;
  playbackSpeed: number;
}

function lines(kind: SceneKind, sim: SimState): { label: string; value: string }[] {
  const p = sim.props;
  const meta = sim.meta;
  if (kind === 'orbit') {
    return [
      { label: 'r', value: numberValue(p, 'orbit_radius', 3.6).toFixed(2) },
      { label: 'v', value: numberValue(p, 'tangential_speed', 2.25).toFixed(2) },
      { label: 'G*', value: numberValue(p, 'gravitational_strength', 10).toFixed(1) },
      { label: '轨迹', value: String((sim.trails.satellite || []).length) },
    ];
  }
  if (kind === 'spring') {
    return [
      { label: 'k', value: numberValue(p, 'k', 24).toFixed(1) },
      { label: 'm', value: `${numberValue(p, 'm', 1.2).toFixed(1)} kg` },
      { label: 'x', value: `${(meta.spring_x ?? numberValue(p, 'x', 1.4)).toFixed(2)} m` },
      { label: 'E', value: (meta.energy ?? 0).toFixed(2) },
    ];
  }
  if (kind === 'collision') {
    return [
      { label: 'm₁', value: `${numberValue(p, 'm1', 1.5).toFixed(1)} kg` },
      { label: 'm₂', value: `${numberValue(p, 'm2', 1).toFixed(1)} kg` },
      { label: 'e', value: numberValue(p, 'restitution', 0.9).toFixed(2) },
      { label: 'KE', value: (meta.energy ?? 0).toFixed(1) },
    ];
  }
  if (kind === 'projectile') {
    return [
      { label: 'v₀', value: `${numberValue(p, 'v0', 20).toFixed(1)} m/s` },
      { label: 'θ', value: `${numberValue(p, 'angle_deg', 45).toFixed(0)}°` },
      { label: '轨迹', value: String(sim.trail.length) },
    ];
  }
  if (kind === 'force') {
    return [
      { label: 'm', value: `${numberValue(p, 'm', 2).toFixed(1)} kg` },
      { label: 'a', value: `${numberValue(p, 'a', 3).toFixed(1)} m/s²` },
      { label: 'F', value: `${(numberValue(p, 'm', 2) * numberValue(p, 'a', 3)).toFixed(1)} N` },
    ];
  }
  return [
    { label: 'v', value: `${numberValue(p, 'v', 5).toFixed(1)} m/s` },
    { label: 'a', value: `${numberValue(p, 'a', 0).toFixed(1)} m/s²` },
    { label: '轨迹', value: String(sim.trail.length) },
  ];
}

export default function R3FHud({ simRef, kind, running, playbackSpeed }: Props) {
  const rowsRef = useRef<HTMLDivElement | null>(null);

  useFrame(() => {
    const sim = simRef.current;
    const target = rowsRef.current;
    if (!sim || !target) return;
    const data = lines(kind, sim);
    const children = target.children;
    for (let i = 0; i < data.length; i += 1) {
      const node = children[i] as HTMLDivElement | undefined;
      if (!node) continue;
      const valueNode = node.querySelector('[data-value]') as HTMLSpanElement | null;
      if (valueNode && valueNode.textContent !== data[i].value) {
        valueNode.textContent = data[i].value;
      }
    }
  });

  const initial = lines(kind, { props: {}, meta: {}, trail: [], trails: {} } as unknown as SimState);

  return (
    <Html
      fullscreen
      style={{ pointerEvents: 'none' }}
      transform={false}
      zIndexRange={[10, 0]}
    >
      <div className="snowy-r3f-hud" aria-hidden>
        <div className="snowy-r3f-hud__title">{sceneLabel(kind)}</div>
        <div ref={rowsRef} className="snowy-r3f-hud__rows">
          {initial.map((row) => (
            <div key={row.label} className="snowy-r3f-hud__row">
              <span className="snowy-r3f-hud__key">{row.label}</span>
              <span data-value className="snowy-r3f-hud__val">{row.value}</span>
            </div>
          ))}
        </div>
        <div className="snowy-r3f-hud__foot">
          <span>{running ? '运行中' : '已暂停'}</span>
          <span>× {playbackSpeed.toFixed(2)}</span>
        </div>
      </div>
    </Html>
  );
}
