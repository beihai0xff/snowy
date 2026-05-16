/**
 * Snowy v6 · R3F · 轨迹尾迹组件
 *
 * 接收 SimState 中的 trail / trails 数组（ref 形式），按帧把点更新到 BufferGeometry。
 */

'use client';

import React, { useRef } from 'react';
import * as THREE from 'three';
import { useFrame } from '@react-three/fiber';
import type { SimState, Vec3 } from './types';

interface Props {
  simRef: React.MutableRefObject<SimState | null>;
  source: 'main' | 'satellite' | 'mass' | 'ball1' | 'ball2';
  color: string;
  maxPoints?: number;
}

function getTrail(sim: SimState, source: Props['source']): Vec3[] {
  if (source === 'main') return sim.trail;
  return sim.trails[source] || [];
}

export default function R3FTrail({ simRef, source, color, maxPoints = 360 }: Props) {
  const lineRef = useRef<THREE.Line | null>(null);
  const [geometry] = React.useState<THREE.BufferGeometry>(() => {
    const geo = new THREE.BufferGeometry();
    const positions = new Float32Array(maxPoints * 3);
    geo.setAttribute('position', new THREE.BufferAttribute(positions, 3));
    geo.setDrawRange(0, 0);
    return geo;
  });

  useFrame(() => {
    const sim = simRef.current;
    if (!sim || !geometry) return;
    const trail = getTrail(sim, source);
    const positions = geometry.attributes.position.array as Float32Array;
    const count = Math.min(trail.length, maxPoints);
    for (let i = 0; i < count; i += 1) {
      const p = trail[trail.length - count + i];
      // eslint-disable-next-line react-hooks/immutability
      positions[i * 3] = p.x;
      positions[i * 3 + 1] = p.y + 0.05;
      positions[i * 3 + 2] = p.z;
    }
    geometry.setDrawRange(0, count);
    geometry.attributes.position.needsUpdate = true;
  });

  return (
    // @ts-expect-error R3F line element
    <line ref={lineRef as unknown as React.RefObject<THREE.Line>}>
      <primitive object={geometry} attach="geometry" />
      <lineBasicMaterial attach="material" color={color} transparent opacity={0.85} linewidth={2} />
    </line>
  );
}
