/**
 * Snowy v8 · R3F · HUD（升级版）
 *
 * 升级点：
 *  - 数值键值表保留，但每帧通过 DOM 节点直接更新，无 React re-render
 *  - 新增"实时公式卡"：使用 katex 渲染 LaTeX，参数变化时数值高亮过渡
 *  - 学科色 token glass 卡片
 *
 * 与旧版兼容：保留 sceneLabel / running / playbackSpeed
 */

'use client';

import React, { useEffect, useMemo, useRef } from 'react';
import { Html } from '@react-three/drei';
import { useFrame } from '@react-three/fiber';
import katex from 'katex';
import 'katex/dist/katex.min.css';
import { numberValue, sceneLabel, type SceneKind, type SimState } from './types';

interface Props {
  simRef: React.MutableRefObject<SimState | null>;
  kind: SceneKind;
  running: boolean;
  playbackSpeed: number;
}

interface HudRow { label: string; value: string }
interface HudFormula { id: string; latex: (sim: SimState) => string }

function rows(kind: SceneKind, sim: SimState): HudRow[] {
  const p = sim.props;
  const meta = sim.meta;
  if (kind === 'orbit') {
    return [
      { label: 'r', value: numberValue(p, 'orbit_radius', 3.6).toFixed(2) },
      { label: 'v', value: numberValue(p, 'tangential_speed', 2.25).toFixed(2) },
      { label: 'G*', value: numberValue(p, 'gravitational_strength', 10).toFixed(1) },
      { label: '点', value: String((sim.trails.satellite || []).length) },
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
      { label: 'g', value: `${numberValue(p, 'g', 9.8).toFixed(1)} m/s²` },
      { label: '点', value: String(sim.trail.length) },
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
    { label: '点', value: String(sim.trail.length) },
  ];
}

function formulas(kind: SceneKind): HudFormula[] {
  if (kind === 'projectile') {
    return [
      { id: 'x',  latex: (sim) => `x = v_0 \\cos\\theta \\cdot t = ${numberValue(sim.props,'v0',20).toFixed(1)}\\cos${numberValue(sim.props,'angle_deg',45).toFixed(0)}^\\circ \\cdot t` },
      { id: 'y',  latex: () => `y = v_0 \\sin\\theta \\cdot t - \\tfrac{1}{2}gt^2` },
    ];
  }
  if (kind === 'force') {
    return [
      { id: 'newton', latex: (sim) => {
        const m = numberValue(sim.props,'m',2);
        const a = numberValue(sim.props,'a',3);
        return `F = ma = ${m.toFixed(1)} \\times ${a.toFixed(1)} = \\textbf{${(m*a).toFixed(1)}\\,N}`;
      } },
    ];
  }
  if (kind === 'spring') {
    return [
      { id: 'hooke', latex: (sim) => {
        const k = numberValue(sim.props,'k',24);
        const x = sim.meta.spring_x ?? numberValue(sim.props,'x',1.4);
        return `F = -kx = -${k.toFixed(1)} \\times ${x.toFixed(2)} = \\textbf{${(-k*x).toFixed(2)}\\,N}`;
      } },
      { id: 'T', latex: (sim) => {
        const k = numberValue(sim.props,'k',24);
        const m = numberValue(sim.props,'m',1.2);
        return `T = 2\\pi\\sqrt{m/k} \\approx ${(2*Math.PI*Math.sqrt(m/k)).toFixed(2)}\\,s`;
      } },
    ];
  }
  if (kind === 'orbit') {
    return [
      { id: 'kepler3', latex: (sim) => {
        const r = numberValue(sim.props,'orbit_radius',3.6);
        const v = numberValue(sim.props,'tangential_speed',2.25);
        return `v = \\sqrt{GM/r} \\Rightarrow r v^2 = ${(r*v*v).toFixed(2)}`;
      } },
    ];
  }
  if (kind === 'collision') {
    return [
      { id: 'momentum', latex: () => `m_1 v_1 + m_2 v_2 = m_1 v_1' + m_2 v_2'` },
    ];
  }
  return [
    { id: 'v', latex: (sim) => `v = v_0 + at = ${numberValue(sim.props,'v',5).toFixed(1)} + ${numberValue(sim.props,'a',0).toFixed(2)} t` },
  ];
}

export default function R3FHud({ simRef, kind, running, playbackSpeed }: Props) {
  const rowsRef = useRef<HTMLDivElement | null>(null);
  const formulaRefs = useRef<Record<string, HTMLDivElement | null>>({});

  const initialRows = rows(kind, { props: {}, meta: {}, trail: [], trails: {} } as unknown as SimState);
  const formulaList = useMemo(() => formulas(kind), [kind]);

  // 初次挂载渲染公式（占位）
  useEffect(() => {
    formulaList.forEach((f) => {
      const node = formulaRefs.current[f.id];
      if (!node) return;
      try {
        const tex = f.latex({ props: {}, meta: {}, trail: [], trails: {} } as unknown as SimState);
        katex.render(tex, node, { throwOnError: false, displayMode: false });
      } catch { /* noop */ }
    });
  }, [formulaList]);

  // 每帧更新数值与公式
  const lastFormulaTex = useRef<Record<string, string>>({});
  useFrame(() => {
    const sim = simRef.current;
    if (!sim) return;
    // 数值
    const target = rowsRef.current;
    if (target) {
      const data = rows(kind, sim);
      const children = target.children;
      for (let i = 0; i < data.length; i += 1) {
        const node = children[i] as HTMLDivElement | undefined;
        if (!node) continue;
        const valueNode = node.querySelector('[data-value]') as HTMLSpanElement | null;
        if (valueNode && valueNode.textContent !== data[i].value) {
          valueNode.textContent = data[i].value;
          valueNode.classList.add('is-flash');
          window.setTimeout(() => valueNode.classList.remove('is-flash'), 220);
        }
      }
    }
    // 公式（节流：变化时才重渲）
    formulaList.forEach((f) => {
      const node = formulaRefs.current[f.id];
      if (!node) return;
      let tex: string;
      try { tex = f.latex(sim); } catch { return; }
      if (lastFormulaTex.current[f.id] === tex) return;
      lastFormulaTex.current[f.id] = tex;
      try {
        katex.render(tex, node, { throwOnError: false, displayMode: false });
      } catch { /* noop */ }
    });
  });

  return (
    <Html
      fullscreen
      style={{ pointerEvents: 'none' }}
      transform={false}
      zIndexRange={[10, 0]}
    >
      <div className="snowy-r3f-hud snowy-r3f-hud--v8" aria-hidden>
        <div className="snowy-r3f-hud__title">{sceneLabel(kind)}</div>
        <div ref={rowsRef} className="snowy-r3f-hud__rows">
          {initialRows.map((row) => (
            <div key={row.label} className="snowy-r3f-hud__row">
              <span className="snowy-r3f-hud__key">{row.label}</span>
              <span data-value className="snowy-r3f-hud__val">{row.value}</span>
            </div>
          ))}
        </div>
        {formulaList.length > 0 && (
          <div className="snowy-r3f-hud__formulas">
            {formulaList.map((f) => (
              <div
                key={f.id}
                className="snowy-r3f-hud__formula"
                ref={(el) => { formulaRefs.current[f.id] = el; }}
              />
            ))}
          </div>
        )}
        <div className="snowy-r3f-hud__foot">
          <span>{running ? '运行中' : '已暂停'}</span>
          <span>× {playbackSpeed.toFixed(2)}</span>
        </div>
      </div>
    </Html>
  );
}

