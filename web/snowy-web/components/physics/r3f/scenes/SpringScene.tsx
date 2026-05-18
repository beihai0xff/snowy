/**
 * Snowy v6 · R3F · 弹簧振子场景
 *
 * 用 TubeGeometry 沿 helix 渲染弹簧；端点是质点。
 */

'use client';

import React, { useMemo, useRef } from 'react';
import * as THREE from 'three';
import { useFrame } from '@react-three/fiber';
import R3FStage from '../lib/R3FStage';
import { numberValue, type SimState } from '../lib/types';

interface Props {
  simRef: React.MutableRefObject<SimState | null>;
  quality?: 'eco' | 'standard' | 'high';
}

class HelixCurve extends THREE.Curve<THREE.Vector3> {
  constructor(private readonly length: number, private readonly coils: number, private readonly radius: number) {
    super();
  }
  override getPoint(t: number, target = new THREE.Vector3()): THREE.Vector3 {
    const angle = t * Math.PI * 2 * this.coils;
    const x = -2.4 + this.length * t;
    const y = 1.2 + Math.cos(angle) * this.radius;
    const z = Math.sin(angle) * this.radius;
    return target.set(x, y, z);
  }
}

export default function SpringScene({ simRef, quality = 'standard' }: Props) {
  const massRef = useRef<THREE.Mesh | null>(null);
  const springRef = useRef<THREE.Mesh | null>(null);
  const wallRef = useRef<THREE.Mesh | null>(null);

  const geometryCache = useRef<{ length: number; geo: THREE.TubeGeometry } | null>(null);

  const wallGeo = useMemo(() => new THREE.BoxGeometry(0.2, 1.6, 1.4), []);

  useFrame(() => {
    const sim = simRef.current;
    if (!sim || !massRef.current) return;
    const body = sim.bodies.mass;
    if (!body) return;
    const p = body.translation();
    massRef.current.position.set(p.x, p.y, p.z);
    const length = Math.max(0.4, p.x - (-2.4) + 0.2);
    if (springRef.current) {
      const cached = geometryCache.current;
      if (!cached || Math.abs(cached.length - length) > 0.02) {
        cached?.geo.dispose();
        const curve = new HelixCurve(length, 8, 0.18);
        const geo = new THREE.TubeGeometry(curve, 96, 0.045, 8, false);
        geometryCache.current = { length, geo };
        springRef.current.geometry = geo;
      }
    }
    if (wallRef.current) {
      wallRef.current.position.set(-2.6, 1.2, 0);
    }
    sim.meta.spring_x = p.x - (-2.4);
    const k = numberValue(sim.props, 'k', 24) * 0.16;
    const m = Math.max(0.1, numberValue(sim.props, 'm', 1.2));
    const v = body.linvel();
    sim.meta.kinetic = 0.5 * m * v.x * v.x;
    sim.meta.potential = 0.5 * k * (p.x - (-2.4)) ** 2;
    sim.meta.energy = sim.meta.kinetic + sim.meta.potential;
  });

  return (
    <>
      <R3FStage kind="spring" quality={quality} />
      {/* 墙面（红砖金属感） */}
      <mesh ref={wallRef} geometry={wallGeo} castShadow receiveShadow>
        <meshStandardMaterial color="#1f2937" metalness={0.78} roughness={0.35} emissive="#0ea5e9" emissiveIntensity={0.08} />
      </mesh>
      {/* 弹簧 */}
      <mesh ref={springRef} castShadow>
        <tubeGeometry args={[new HelixCurve(2, 8, 0.18), 96, 0.05, 10, false]} />
        <meshStandardMaterial color="#67e8f9" emissive="#0e7490" emissiveIntensity={0.55} metalness={0.75} roughness={0.22} toneMapped={false} />
      </mesh>
      {/* 质点 */}
      <mesh ref={massRef} castShadow>
        <sphereGeometry args={[0.34, 48, 48]} />
        <meshStandardMaterial
          color="#cffafe"
          emissive="#22d3ee"
          emissiveIntensity={0.85}
          roughness={0.18}
          metalness={0.6}
          toneMapped={false}
        />
      </mesh>
    </>
  );
}
