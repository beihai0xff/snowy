/**
 * Snowy v6 · R3F · 共享舞台与背景
 *
 * 提供：星空背景、棋盘地面、坐标轴、灯光、相机。所有场景共用。
 */

'use client';

import React, { useMemo } from 'react';
import * as THREE from 'three';
import { Stars, Grid, Float } from '@react-three/drei';
import type { SceneKind } from './types';

interface Props {
  kind: SceneKind;
  /** 是否显示地面 */
  showGround?: boolean;
  /** 主光强度 */
  lightIntensity?: number;
}

export default function R3FStage({ kind, showGround = true, lightIntensity = 1.0 }: Props) {
  const groundColor = useMemo(() => {
    if (kind === 'orbit') return '#0b1220';
    if (kind === 'collision') return '#1a1421';
    return '#0f172a';
  }, [kind]);

  return (
    <>
      <color attach="background" args={[kind === 'orbit' ? '#03050d' : '#070b18']} />
      <fog attach="fog" args={[kind === 'orbit' ? '#03050d' : '#070b18', 12, 38]} />

      <ambientLight intensity={0.35 * lightIntensity} />
      <directionalLight
        position={[6, 9, 4]}
        intensity={1.1 * lightIntensity}
        color="#a5b4fc"
        castShadow
      />
      <pointLight position={[-5, 5, -5]} intensity={0.4 * lightIntensity} color="#22d3ee" />
      <pointLight position={[6, 2, 6]} intensity={0.3 * lightIntensity} color="#fb7185" />

      {kind === 'orbit' && (
        <Stars radius={50} depth={50} count={2400} factor={4} saturation={0.4} fade speed={0.6} />
      )}
      {kind !== 'orbit' && (
        <Float speed={0.4} rotationIntensity={0} floatIntensity={0.15}>
          <Stars radius={36} depth={32} count={1100} factor={2.6} saturation={0.2} fade speed={0.4} />
        </Float>
      )}

      {showGround && (
        <>
          <Grid
            args={[28, 28]}
            cellSize={1}
            cellThickness={0.6}
            cellColor="#1e293b"
            sectionSize={4}
            sectionThickness={1.2}
            sectionColor="#22d3ee"
            fadeDistance={26}
            fadeStrength={1.2}
            infiniteGrid
            position={[0, 0, 0]}
          />
          <mesh receiveShadow rotation={[-Math.PI / 2, 0, 0]} position={[0, -0.005, 0]}>
            <planeGeometry args={[50, 50]} />
            <meshStandardMaterial color={groundColor} roughness={0.95} metalness={0.05} transparent opacity={0.85} />
          </mesh>
        </>
      )}

      {/* 坐标轴 */}
      <group>
        <AxisArrow direction={new THREE.Vector3(1, 0, 0)} color="#f87171" label="X" />
        <AxisArrow direction={new THREE.Vector3(0, 1, 0)} color="#34d399" label="Y" />
        <AxisArrow direction={new THREE.Vector3(0, 0, 1)} color="#60a5fa" label="Z" />
      </group>
    </>
  );
}

function AxisArrow({ direction, color, label }: { direction: THREE.Vector3; color: string; label: string }) {
  void label;
  const points = useMemo(() => {
    const end = direction.clone().multiplyScalar(5.4);
    return [new THREE.Vector3(0, 0.005, 0), end];
  }, [direction]);
  return (
    <>
      <line>
        <bufferGeometry attach="geometry">
          <bufferAttribute
            attach="attributes-position"
            args={[new Float32Array(points.flatMap((p) => [p.x, p.y, p.z])), 3]}
            count={points.length}
            itemSize={3}
          />
        </bufferGeometry>
        <lineBasicMaterial attach="material" color={color} transparent opacity={0.55} />
      </line>
      <mesh position={points[1]}>
        <sphereGeometry args={[0.05, 16, 16]} />
        <meshBasicMaterial color={color} />
      </mesh>
    </>
  );
}
