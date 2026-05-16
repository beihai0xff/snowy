/**
 * Snowy v6 · R3F · 碰撞场景
 *
 * 两个发光球体在地面上对撞；碰撞瞬间触发粒子爆点 (短暂高亮)。
 */

'use client';

import React, { useRef } from 'react';
import * as THREE from 'three';
import { useFrame } from '@react-three/fiber';
import R3FStage from '../lib/R3FStage';
import R3FTrail from '../lib/R3FTrail';
import { type SimState } from '../lib/types';

interface Props {
  simRef: React.MutableRefObject<SimState | null>;
}

export default function CollisionScene({ simRef }: Props) {
  const ball1Ref = useRef<THREE.Mesh | null>(null);
  const ball2Ref = useRef<THREE.Mesh | null>(null);
  const burstRef = useRef<THREE.Points | null>(null);
  const [burstGeo] = React.useState<THREE.BufferGeometry>(() => {
    const positions = new Float32Array(60 * 3);
    const geo = new THREE.BufferGeometry();
    geo.setAttribute('position', new THREE.BufferAttribute(positions, 3));
    return geo;
  });
  const [dirs] = React.useState<THREE.Vector3[]>(() => {
    const arr: THREE.Vector3[] = [];
    for (let i = 0; i < 60; i += 1) arr.push(new THREE.Vector3().randomDirection());
    return arr;
  });

  useFrame(() => {
    const sim = simRef.current;
    if (!sim) return;
    if (ball1Ref.current && sim.bodies.ball1) {
      const p = sim.bodies.ball1.translation();
      ball1Ref.current.position.set(p.x, p.y, p.z);
    }
    if (ball2Ref.current && sim.bodies.ball2) {
      const p = sim.bodies.ball2.translation();
      ball2Ref.current.position.set(p.x, p.y, p.z);
    }
    const burst = burstRef.current;
    if (burst && burstGeo) {
      const now = performance.now();
      const last = sim.lastImpactAt || 0;
      const dt = (now - last) / 800;
      if (last && dt < 1) {
        burst.visible = true;
        const positions = burstGeo.attributes.position.array as Float32Array;
        const b1 = sim.bodies.ball1?.translation();
        const b2 = sim.bodies.ball2?.translation();
        if (b1 && b2) {
          const cx = (b1.x + b2.x) / 2;
          const cy = (b1.y + b2.y) / 2;
          const cz = (b1.z + b2.z) / 2;
          burst.position.set(cx, cy, cz);
        }
        const r = dt * 1.4;
        for (let i = 0; i < dirs.length; i += 1) {
          // eslint-disable-next-line react-hooks/immutability
          positions[i * 3] = dirs[i].x * r;
          positions[i * 3 + 1] = dirs[i].y * r * 0.6;
          positions[i * 3 + 2] = dirs[i].z * r;
        }
        burstGeo.attributes.position.needsUpdate = true;
        (burst.material as THREE.PointsMaterial).opacity = 1 - dt;
      } else if (burst) {
        burst.visible = false;
      }
    }
  });

  return (
    <>
      <R3FStage kind="collision" />
      <R3FTrail simRef={simRef} source="ball1" color="#f472b6" maxPoints={220} />
      <R3FTrail simRef={simRef} source="ball2" color="#a78bfa" maxPoints={220} />
      <mesh ref={ball1Ref} castShadow>
        <sphereGeometry args={[0.42, 36, 36]} />
        <meshStandardMaterial color="#fbcfe8" emissive="#ec4899" emissiveIntensity={0.65} roughness={0.28} metalness={0.5} />
      </mesh>
      <mesh ref={ball2Ref} castShadow>
        <sphereGeometry args={[0.42, 36, 36]} />
        <meshStandardMaterial color="#ddd6fe" emissive="#8b5cf6" emissiveIntensity={0.65} roughness={0.28} metalness={0.5} />
      </mesh>
      <points ref={burstRef} visible={false}>
        <primitive object={burstGeo} attach="geometry" />
        <pointsMaterial color="#fde68a" size={0.12} transparent opacity={0.9} depthWrite={false} sizeAttenuation />
      </points>
    </>
  );
}
