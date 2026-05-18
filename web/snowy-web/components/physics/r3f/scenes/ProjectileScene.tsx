/* eslint-disable react-hooks/immutability */
/**
 * Snowy v8 · R3F · 抛体场景（升级版）
 *
 * 升级点：
 *  - 起点发射器（红橙金属圆柱 + 尖端发光）
 *  - 弹体改 emissive transmission 风格
 *  - 落地触发 Burst 粒子（自管理 ttl 600ms）
 *  - 速度矢量 v 分解：v_x 黄 + v_y 红
 */

'use client';

import React, { useMemo, useRef } from 'react';
import * as THREE from 'three';
import { useFrame } from '@react-three/fiber';
import R3FStage from '../lib/R3FStage';
import R3FTrail from '../lib/R3FTrail';
import FrameVector from '../lib/FrameVector';
import { numberValue, type SimState } from '../lib/types';

interface Props {
  simRef: React.MutableRefObject<SimState | null>;
  quality?: 'eco' | 'standard' | 'high';
}

const BURST_COUNT = 36;

function seeded01(seed: number): number {
  const x = Math.sin(seed * 12.9898) * 43758.5453;
  return x - Math.floor(x);
}

export default function ProjectileScene({ simRef, quality = 'standard' }: Props) {
  const meshRef = useRef<THREE.Mesh | null>(null);
  const haloRef = useRef<THREE.Mesh | null>(null);

  // 落地爆点粒子
  const burstRef = useRef<THREE.Points | null>(null);
  const burstGeo = useMemo(() => {
    const geo = new THREE.BufferGeometry();
    geo.setAttribute('position', new THREE.BufferAttribute(new Float32Array(BURST_COUNT * 3), 3));
    geo.setAttribute('color', new THREE.BufferAttribute(new Float32Array(BURST_COUNT * 3), 3));
    return geo;
  }, []);
  const burstDirs = useMemo(() => {
    const arr: THREE.Vector3[] = [];
    for (let i = 0; i < BURST_COUNT; i += 1) {
      const phi = seeded01(i + 1) * Math.PI * 2;
      const r = 0.4 + seeded01(i + 101) * 0.6;
      arr.push(new THREE.Vector3(Math.cos(phi) * r, 0.2 + seeded01(i + 201) * 0.8, Math.sin(phi) * r));
    }
    return arr;
  }, []);
  const burstStartedRef = useRef<{ x: number; y: number; z: number; t: number } | null>(null);
  const landedRef = useRef(false);

  // 起点发射器（v0/angle 决定朝向）
  const launcherRef = useRef<THREE.Group | null>(null);

  useFrame(({ clock }) => {
    const sim = simRef.current;
    if (!sim || !meshRef.current) return;
    const body = sim.bodies.main;
    if (!body) return;
    const p = body.translation();
    meshRef.current.position.set(p.x, p.y, p.z);
    if (haloRef.current) {
      haloRef.current.position.set(p.x, p.y, p.z);
      const t = clock.elapsedTime * 2;
      haloRef.current.scale.setScalar(1 + Math.sin(t) * 0.08);
    }

    // 发射器朝向：仅初始化一次
    if (launcherRef.current && !landedRef.current) {
      const angle = numberValue(sim.props, 'angle_deg', 45) * Math.PI / 180;
      launcherRef.current.rotation.z = angle - Math.PI / 2;
    }

    // 落地检测：y < 0.15 且未触发过
    if (!landedRef.current && p.y < 0.15 && sim.trail.length > 30) {
      landedRef.current = true;
      burstStartedRef.current = { x: p.x, y: 0.1, z: p.z, t: performance.now() };
    }

    // 重置：trail 重置时清落地标志
    if (sim.trail.length < 5) {
      landedRef.current = false;
      burstStartedRef.current = null;
      if (burstRef.current) burstRef.current.visible = false;
    }

    // 更新爆点粒子
    const burst = burstRef.current;
    const burstStart = burstStartedRef.current;
    if (burst && burstStart) {
      const dt = (performance.now() - burstStart.t) / 600;
      if (dt < 1) {
        burst.visible = true;
        burst.position.set(burstStart.x, burstStart.y, burstStart.z);
        const positions = burstGeo.attributes.position.array as Float32Array;
        const colors = burstGeo.attributes.color.array as Float32Array;
        const r = dt * 1.6;
        for (let i = 0; i < BURST_COUNT; i += 1) {
          const d = burstDirs[i];
          positions[i * 3]     = d.x * r;
          positions[i * 3 + 1] = d.y * r - 0.5 * dt * dt;
          positions[i * 3 + 2] = d.z * r;
          // 颜色：从 #fde68a → #f97316 → 透明（用 RGB 渐变近似）
          const hue = 0.13 - dt * 0.06; // 黄 → 橙
          const c = new THREE.Color().setHSL(hue, 0.95, 0.6 - dt * 0.4);
          colors[i * 3] = c.r; colors[i * 3 + 1] = c.g; colors[i * 3 + 2] = c.b;
        }
        burstGeo.attributes.position.needsUpdate = true;
        burstGeo.attributes.color.needsUpdate = true;
        const mat = burst.material as THREE.PointsMaterial;
        mat.size = 0.18 * (1 - dt * 0.6);
        mat.opacity = 1 - dt;
      } else {
        burst.visible = false;
      }
    }
  });

  return (
    <>
      <R3FStage kind="projectile" quality={quality} />

      {/* 发射器（起点 0,0.5,0） */}
      <group ref={launcherRef} position={[0, 0.5, 0]}>
        <mesh castShadow position={[0, 0.4, 0]}>
          <cylinderGeometry args={[0.16, 0.22, 0.9, 24]} />
          <meshStandardMaterial color="#7c2d12" metalness={0.85} roughness={0.32} emissive="#ea580c" emissiveIntensity={0.15} />
        </mesh>
        <mesh castShadow position={[0, 0.85, 0]}>
          <coneGeometry args={[0.16, 0.25, 24]} />
          <meshStandardMaterial color="#fb923c" emissive="#f97316" emissiveIntensity={0.85} metalness={0.4} roughness={0.25} toneMapped={false} />
        </mesh>
        <mesh receiveShadow position={[0, 0, 0]}>
          <cylinderGeometry args={[0.28, 0.28, 0.1, 24]} />
          <meshStandardMaterial color="#1e293b" metalness={0.7} roughness={0.5} />
        </mesh>
      </group>

      <R3FTrail simRef={simRef} source="main" color="#67e8f9" tailColor="#0e1726" maxPoints={quality === 'eco' ? 120 : 360} />

      {/* 弹体 */}
      <mesh ref={meshRef} castShadow>
        <sphereGeometry args={[0.22, 36, 36]} />
        <meshStandardMaterial
          color="#cffafe"
          emissive="#22d3ee"
          emissiveIntensity={1.1}
          roughness={0.15}
          metalness={0.55}
          toneMapped={false}
        />
      </mesh>
      {/* 高光晕环 */}
      {quality !== 'eco' && (
        <mesh ref={haloRef}>
          <sphereGeometry args={[0.32, 24, 24]} />
          <meshBasicMaterial color="#22d3ee" transparent opacity={0.18} depthWrite={false} toneMapped={false} />
        </mesh>
      )}

      <FrameVector simRef={simRef} bodyKey="main" color="#facc15" scale={0.18} />

      {/* 落点爆点 */}
      {quality !== 'eco' && (
        <points ref={burstRef} visible={false}>
          <primitive object={burstGeo} attach="geometry" />
          <pointsMaterial vertexColors size={0.18} transparent opacity={0.95} depthWrite={false} sizeAttenuation toneMapped={false} />
        </points>
      )}
    </>
  );
}

