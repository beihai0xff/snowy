/**
 * Snowy v6 · R3F · 天体轨道场景
 *
 * 中心恒星 (emissive + bloom)，绕转卫星，半透明轨道环。
 */

'use client';

import React, { useRef } from 'react';
import * as THREE from 'three';
import { useFrame } from '@react-three/fiber';
import R3FStage from '../lib/R3FStage';
import R3FTrail from '../lib/R3FTrail';
import { numberValue, type SimState } from '../lib/types';

interface Props {
  simRef: React.MutableRefObject<SimState | null>;
}

export default function OrbitScene({ simRef }: Props) {
  const satRef = useRef<THREE.Mesh | null>(null);
  const ringRef = useRef<THREE.Mesh | null>(null);
  const starRef = useRef<THREE.Mesh | null>(null);

  useFrame(({ clock }) => {
    const sim = simRef.current;
    if (!sim) return;
    const body = sim.bodies.satellite;
    if (body && satRef.current) {
      const p = body.translation();
      satRef.current.position.set(p.x, p.y, p.z);
    }
    if (ringRef.current) {
      const r = numberValue(sim.props, 'orbit_radius', 3.6);
      ringRef.current.scale.set(r, r, r);
    }
    if (starRef.current) {
      const s = 1 + Math.sin(clock.elapsedTime * 2) * 0.04;
      starRef.current.scale.set(s, s, s);
    }
  });

  return (
    <>
      <R3FStage kind="orbit" showGround={false} lightIntensity={0.7} />
      <pointLight position={[0, 0, 0]} intensity={2.4} color="#fde68a" distance={20} />
      <mesh ref={starRef}>
        <sphereGeometry args={[0.55, 48, 48]} />
        <meshStandardMaterial
          color="#fef3c7"
          emissive="#fbbf24"
          emissiveIntensity={2.2}
          toneMapped={false}
        />
      </mesh>
      <mesh ref={ringRef} rotation={[-Math.PI / 2, 0, 0]}>
        <ringGeometry args={[0.985, 1.015, 96]} />
        <meshBasicMaterial color="#38bdf8" transparent opacity={0.35} side={THREE.DoubleSide} />
      </mesh>
      <R3FTrail simRef={simRef} source="satellite" color="#38bdf8" maxPoints={420} />
      <mesh ref={satRef} castShadow>
        <sphereGeometry args={[0.22, 32, 32]} />
        <meshStandardMaterial color="#bae6fd" emissive="#0ea5e9" emissiveIntensity={0.9} roughness={0.3} />
      </mesh>
    </>
  );
}
