/**
 * Snowy v6 · R3F · 受力刚体场景
 *
 * 平面上的方块在水平力作用下加速运动；矢量箭头标注施加的力。
 */

'use client';

import React, { useRef } from 'react';
import * as THREE from 'three';
import { useFrame } from '@react-three/fiber';
import R3FStage from '../lib/R3FStage';
import R3FTrail from '../lib/R3FTrail';
import FrameVector from '../lib/FrameVector';
import { numberValue, type SimState } from '../lib/types';

interface Props {
  simRef: React.MutableRefObject<SimState | null>;
}

export default function ForceScene({ simRef }: Props) {
  const boxRef = useRef<THREE.Mesh | null>(null);

  useFrame(() => {
    const sim = simRef.current;
    if (!sim || !boxRef.current) return;
    const body = sim.bodies.main;
    if (!body) return;
    const p = body.translation();
    const r = body.rotation();
    boxRef.current.position.set(p.x, p.y, p.z);
    boxRef.current.quaternion.set(r.x, r.y, r.z, r.w);
  });

  return (
    <>
      <R3FStage kind="force" />
      <R3FTrail simRef={simRef} source="main" color="#22d3ee" />
      <mesh ref={boxRef} castShadow>
        <boxGeometry args={[1.2, 1.2, 1.2]} />
        <meshStandardMaterial color="#67e8f9" emissive="#0e7490" emissiveIntensity={0.5} roughness={0.35} metalness={0.55} />
      </mesh>
      <FrameVector
        simRef={simRef}
        bodyKey="main"
        color="#facc15"
        scale={0.05}
        vectorOf={(sim) => {
          const m = numberValue(sim.props, 'm', 2);
          const a = numberValue(sim.props, 'a', 3);
          return { x: m * a, y: 0, z: 0 };
        }}
      />
    </>
  );
}
