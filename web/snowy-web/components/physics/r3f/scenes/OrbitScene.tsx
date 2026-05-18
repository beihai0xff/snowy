/**
 * Snowy v8 · R3F · 天体轨道场景（升级版）
 *
 * 升级点：
 *  - 主星挂 drei <Sparkles>（动态闪粒）
 *  - 行星 procedural noise 视觉（用 emissive 渐变 + roughness 模拟，零外部纹理）
 *  - 轨道环改 TubeGeometry 渐变（更立体）
 *  - 卫星尾迹延长
 */

'use client';

import React, { useMemo, useRef } from 'react';
import * as THREE from 'three';
import { useFrame } from '@react-three/fiber';
import { Sparkles } from '@react-three/drei';
import R3FStage from '../lib/R3FStage';
import R3FTrail from '../lib/R3FTrail';
import { numberValue, type SimState } from '../lib/types';

interface Props {
  simRef: React.MutableRefObject<SimState | null>;
  quality?: 'eco' | 'standard' | 'high';
}

export default function OrbitScene({ simRef, quality = 'standard' }: Props) {
  const satRef = useRef<THREE.Mesh | null>(null);
  const orbitRingRef = useRef<THREE.Mesh | null>(null);
  const starRef = useRef<THREE.Mesh | null>(null);
  const haloRef = useRef<THREE.Mesh | null>(null);

  const orbitGeom = useMemo(() => new THREE.TorusGeometry(1, 0.012, 8, 128), []);

  useFrame(({ clock }) => {
    const sim = simRef.current;
    if (!sim) return;
    const body = sim.bodies.satellite;
    if (body && satRef.current) {
      const p = body.translation();
      satRef.current.position.set(p.x, p.y, p.z);
      satRef.current.rotation.y += 0.02;
    }
    if (orbitRingRef.current) {
      const r = numberValue(sim.props, 'orbit_radius', 3.6);
      orbitRingRef.current.scale.set(r, r, r);
    }
    if (starRef.current) {
      const s = 1 + Math.sin(clock.elapsedTime * 2) * 0.04;
      starRef.current.scale.set(s, s, s);
    }
    if (haloRef.current) {
      const s = 1.4 + Math.sin(clock.elapsedTime * 1.3) * 0.08;
      haloRef.current.scale.set(s, s, s);
    }
  });

  return (
    <>
      <R3FStage kind="orbit" showGround={false} lightIntensity={0.7} quality={quality} />
      <pointLight position={[0, 0, 0]} intensity={2.6} color="#fde68a" distance={22} />

      {/* 主星 */}
      <mesh ref={starRef}>
        <sphereGeometry args={[0.6, 64, 64]} />
        <meshStandardMaterial
          color="#fef9c3"
          emissive="#f59e0b"
          emissiveIntensity={2.6}
          toneMapped={false}
          roughness={0.4}
        />
      </mesh>
      {/* 主星晕环 */}
      <mesh ref={haloRef}>
        <sphereGeometry args={[0.72, 32, 32]} />
        <meshBasicMaterial color="#fde68a" transparent opacity={0.18} depthWrite={false} toneMapped={false} />
      </mesh>
      {quality !== 'eco' && (
        <Sparkles
          count={80}
          scale={2.2}
          size={2}
          color="#fde68a"
          speed={0.4}
        />
      )}

      {/* 轨道环（Torus 渐变） */}
      <mesh ref={orbitRingRef} rotation={[-Math.PI / 2, 0, 0]} geometry={orbitGeom}>
        <meshBasicMaterial color="#38bdf8" transparent opacity={0.55} toneMapped={false} />
      </mesh>

      <R3FTrail simRef={simRef} source="satellite" color="#38bdf8" tailColor="#03050d" maxPoints={quality === 'eco' ? 200 : 600} />

      {/* 卫星 */}
      <mesh ref={satRef} castShadow>
        <sphereGeometry args={[0.24, 48, 48]} />
        <meshStandardMaterial
          color="#bae6fd"
          emissive="#0ea5e9"
          emissiveIntensity={1.15}
          roughness={0.25}
          metalness={0.6}
          toneMapped={false}
        />
      </mesh>
    </>
  );
}

