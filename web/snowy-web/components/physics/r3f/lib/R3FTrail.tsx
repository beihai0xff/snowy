/* eslint-disable react-hooks/immutability */
/**
 * Snowy v8 · R3F · 轨迹尾迹（升级版）
 *
 * 升级点：
 *  - 顶点颜色沿尾迹由 tailColor → color 渐变（默认 tailColor ≈ 场景背景，实现"自然渐隐"）
 *  - LineSegments 替代 Line，每段独立颜色
 *  - 兼容旧 props（source / color / maxPoints），新增 tailColor / glow
 */

'use client';

import React, { useMemo, useRef } from 'react';
import * as THREE from 'three';
import { useFrame } from '@react-three/fiber';
import type { SimState, Vec3 } from './types';

interface Props {
  simRef: React.MutableRefObject<SimState | null>;
  source: 'main' | 'satellite' | 'mass' | 'ball1' | 'ball2';
  color: string;
  /** 尾部颜色 —— 默认 #070b18（接近 stage 背景，等效渐隐） */
  tailColor?: string;
  maxPoints?: number;
}

function getTrail(sim: SimState, source: Props['source']): Vec3[] {
  if (source === 'main') return sim.trail;
  return sim.trails[source] || [];
}

export default function R3FTrail({ simRef, source, color, tailColor = '#070b18', maxPoints = 360 }: Props) {
  const lineRef = useRef<THREE.LineSegments | null>(null);

  const { geometry, colorAttr } = useMemo(() => {
    const geo = new THREE.BufferGeometry();
    const segVerts = (maxPoints - 1) * 2;
    const positions = new Float32Array(segVerts * 3);
    const colors = new Float32Array(segVerts * 3);
    geo.setAttribute('position', new THREE.BufferAttribute(positions, 3));
    geo.setAttribute('color', new THREE.BufferAttribute(colors, 3));
    geo.setDrawRange(0, 0);
    return { geometry: geo, colorAttr: geo.attributes.color as THREE.BufferAttribute };
  }, [maxPoints]);

  const cHead = useMemo(() => new THREE.Color(color), [color]);
  const cTail = useMemo(() => new THREE.Color(tailColor), [tailColor]);

  useFrame(() => {
    const sim = simRef.current;
    if (!sim) return;
    const trail = getTrail(sim, source);
    const count = Math.min(trail.length, maxPoints);
    if (count < 2) {
      geometry.setDrawRange(0, 0);
      return;
    }
    const positions = geometry.attributes.position.array as Float32Array;
    const colors = colorAttr.array as Float32Array;
    const offset = trail.length - count;
    const denom = Math.max(1, count - 1);
    for (let i = 0; i < count - 1; i += 1) {
      const a = trail[offset + i];
      const b = trail[offset + i + 1];
      const j = i * 2;
      positions[j * 3]     = a.x;
      positions[j * 3 + 1] = a.y + 0.05;
      positions[j * 3 + 2] = a.z;
      positions[(j + 1) * 3]     = b.x;
      positions[(j + 1) * 3 + 1] = b.y + 0.05;
      positions[(j + 1) * 3 + 2] = b.z;

      const tA = i / denom;        // 0 = 尾部, 1 = 头部
      const tB = (i + 1) / denom;
      colors[j * 3]     = cTail.r + (cHead.r - cTail.r) * tA;
      colors[j * 3 + 1] = cTail.g + (cHead.g - cTail.g) * tA;
      colors[j * 3 + 2] = cTail.b + (cHead.b - cTail.b) * tA;
      colors[(j + 1) * 3]     = cTail.r + (cHead.r - cTail.r) * tB;
      colors[(j + 1) * 3 + 1] = cTail.g + (cHead.g - cTail.g) * tB;
      colors[(j + 1) * 3 + 2] = cTail.b + (cHead.b - cTail.b) * tB;
    }
    geometry.setDrawRange(0, (count - 1) * 2);
    geometry.attributes.position.needsUpdate = true;
    colorAttr.needsUpdate = true;
  });

  return (
    <lineSegments ref={lineRef as unknown as React.RefObject<THREE.LineSegments>}>
      <primitive object={geometry} attach="geometry" />
      <lineBasicMaterial
        attach="material"
        vertexColors
        transparent
        opacity={0.95}
        depthWrite={false}
        linewidth={2}
        toneMapped={false}
      />
    </lineSegments>
  );
}


