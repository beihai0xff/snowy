/**
 * Snowy v6 · R3F · 共享 useFrame 驱动的速度/力矢量箭头
 * 通过 ref 跟踪刚体位置和速度，无需父级 setState。
 */

'use client';

import React, { useRef } from 'react';
import * as THREE from 'three';
import { useFrame } from '@react-three/fiber';
import type { SimState } from './types';

interface Props {
  simRef: React.MutableRefObject<SimState | null>;
  bodyKey: string;
  color: string;
  /** 速度缩放倍率 */
  scale?: number;
  /** velocity = body.linvel(), force = 自定义 (sim) => vec */
  vectorOf?: (sim: SimState) => { x: number; y: number; z: number } | null;
}

export default function FrameVector({ simRef, bodyKey, color, scale = 0.2, vectorOf }: Props) {
  const groupRef = useRef<THREE.Group | null>(null);
  const shaftRef = useRef<THREE.Mesh | null>(null);
  const headRef = useRef<THREE.Mesh | null>(null);

  useFrame(() => {
    const sim = simRef.current;
    const group = groupRef.current;
    if (!sim || !group) return;
    const body = sim.bodies[bodyKey];
    if (!body) return;
    const p = body.translation();
    const v = vectorOf ? vectorOf(sim) : body.linvel();
    if (!v) return;
    const mag = Math.hypot(v.x, v.y, v.z) * scale;
    const len = Math.max(0.001, mag);
    group.position.set(p.x, p.y, p.z);
    if (mag < 0.05) {
      group.visible = false;
      return;
    }
    group.visible = true;
    const dir = new THREE.Vector3(v.x, v.y, v.z).normalize();
    const q = new THREE.Quaternion().setFromUnitVectors(new THREE.Vector3(0, 1, 0), dir);
    group.quaternion.copy(q);
    if (shaftRef.current) {
      shaftRef.current.scale.set(1, len, 1);
      shaftRef.current.position.y = len / 2;
    }
    if (headRef.current) {
      headRef.current.position.y = len + 0.06;
    }
  });

  return (
    <group ref={groupRef}>
      <mesh ref={shaftRef}>
        <cylinderGeometry args={[0.025, 0.025, 1, 8]} />
        <meshBasicMaterial color={color} />
      </mesh>
      <mesh ref={headRef}>
        <coneGeometry args={[0.08, 0.18, 12]} />
        <meshBasicMaterial color={color} />
      </mesh>
    </group>
  );
}
