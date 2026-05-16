/**
 * Snowy v6 · R3F · 抛体场景
 *
 * 视觉：发光球沿抛物线飞过棋盘地面，留下青色尾迹，黄色速度矢量实时跟随。
 */

'use client';

import React, { useRef } from 'react';
import * as THREE from 'three';
import { useFrame } from '@react-three/fiber';
import R3FStage from '../lib/R3FStage';
import R3FTrail from '../lib/R3FTrail';
import FrameVector from '../lib/FrameVector';
import type { SimState } from '../lib/types';

interface Props {
  simRef: React.MutableRefObject<SimState | null>;
}

export default function ProjectileScene({ simRef }: Props) {
  const meshRef = useRef<THREE.Mesh | null>(null);

  useFrame(() => {
    const sim = simRef.current;
    if (!sim || !meshRef.current) return;
    const body = sim.bodies.main;
    if (!body) return;
    const p = body.translation();
    meshRef.current.position.set(p.x, p.y, p.z);
  });

  return (
    <>
      <R3FStage kind="projectile" />
      <R3FTrail simRef={simRef} source="main" color="#22d3ee" />
      <mesh ref={meshRef} castShadow>
        <sphereGeometry args={[0.22, 32, 32]} />
        <meshStandardMaterial
          color="#67e8f9"
          emissive="#22d3ee"
          emissiveIntensity={0.85}
          roughness={0.25}
          metalness={0.55}
        />
      </mesh>
      <FrameVector simRef={simRef} bodyKey="main" color="#facc15" scale={0.18} />
    </>
  );
}
