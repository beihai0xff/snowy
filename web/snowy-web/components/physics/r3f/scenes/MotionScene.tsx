/**
 * Snowy v6 · R3F · 通用运动场景 (Motion)
 *
 * 默认兜底渲染：球体 + 尾迹 + 速度矢量。
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

export default function MotionScene({ simRef }: Props) {
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
      <R3FStage kind="motion" />
      <R3FTrail simRef={simRef} source="main" color="#22d3ee" />
      <mesh ref={meshRef} castShadow>
        <sphereGeometry args={[0.28, 32, 32]} />
        <meshStandardMaterial color="#67e8f9" emissive="#22d3ee" emissiveIntensity={0.6} roughness={0.3} metalness={0.5} />
      </mesh>
      <FrameVector simRef={simRef} bodyKey="main" color="#facc15" scale={0.2} />
    </>
  );
}
