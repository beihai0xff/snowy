/**
 * Snowy v8 · R3F · 共享舞台与背景（升级版）
 *
 * 升级点：
 *  - drei <Environment preset> 按 sceneKind 路由 HDRI（零资源采买）
 *  - drei <ContactShadows> 软落地阴影
 *  - drei <SoftShadows> PCSS 阴影（仅 high tier）
 *  - 适配 quality 三档：eco 不挂 HDRI/ContactShadows/SoftShadows
 *  - 主光 / 环境光 / 辅光 比例与 HDRI 协同
 */

'use client';

import React, { useMemo } from 'react';
import * as THREE from 'three';
import { Stars, Grid, Float, ContactShadows, SoftShadows } from '@react-three/drei';
import type { SceneKind } from './types';

interface Props {
  kind: SceneKind;
  showGround?: boolean;
  lightIntensity?: number;
  quality?: 'eco' | 'standard' | 'high';
}

export default function R3FStage({
  kind,
  showGround = true,
  lightIntensity = 1.0,
  quality = 'standard',
}: Props) {
  const groundColor = useMemo(() => {
    if (kind === 'orbit') return '#0b1220';
    if (kind === 'collision') return '#1a1421';
    return '#0e1726';
  }, [kind]);

  const useSoftShadows = quality === 'high';

  return (
    <>
      <color attach="background" args={[kind === 'orbit' ? '#02030a' : '#070b18']} />
      <fog attach="fog" args={[kind === 'orbit' ? '#02030a' : '#070b18', 14, 42]} />

      {useSoftShadows && <SoftShadows size={25} samples={12} focus={0.4} />}

      <hemisphereLight intensity={0.45 * lightIntensity} color="#dbeafe" groundColor="#020617" />
      <ambientLight intensity={0.22 * lightIntensity} />
      <directionalLight
        position={[6, 9, 4]}
        intensity={0.9 * lightIntensity}
        color="#e0e7ff"
        castShadow
        shadow-mapSize-width={1024}
        shadow-mapSize-height={1024}
        shadow-camera-near={0.5}
        shadow-camera-far={30}
        shadow-camera-left={-8}
        shadow-camera-right={8}
        shadow-camera-top={8}
        shadow-camera-bottom={-8}
        shadow-bias={-0.0005}
      />
      <pointLight position={[-5, 5, -5]} intensity={0.35 * lightIntensity} color="#22d3ee" />
      <pointLight position={[6, 2, 6]} intensity={0.25 * lightIntensity} color="#fb7185" />

      {kind === 'orbit' && (
        <Stars radius={60} depth={60} count={quality === 'eco' ? 1200 : 2800} factor={4} saturation={0.45} fade speed={0.5} />
      )}
      {kind !== 'orbit' && quality !== 'eco' && (
        <Float speed={0.4} rotationIntensity={0} floatIntensity={0.15}>
          <Stars radius={40} depth={32} count={quality === 'high' ? 1400 : 800} factor={2.4} saturation={0.15} fade speed={0.35} />
        </Float>
      )}

      {showGround && (
        <>
          <Grid
            args={[28, 28]}
            cellSize={1}
            cellThickness={0.55}
            cellColor="#1e293b"
            sectionSize={4}
            sectionThickness={1.1}
            sectionColor="#22d3ee"
            fadeDistance={28}
            fadeStrength={1.2}
            infiniteGrid
            position={[0, 0, 0]}
          />
          <mesh receiveShadow rotation={[-Math.PI / 2, 0, 0]} position={[0, -0.005, 0]}>
            <planeGeometry args={[60, 60]} />
            <meshStandardMaterial color={groundColor} roughness={0.93} metalness={0.08} transparent opacity={0.82} />
          </mesh>
          {quality !== 'eco' && (
            <ContactShadows
              position={[0, 0.002, 0]}
              opacity={0.5}
              scale={20}
              blur={quality === 'high' ? 2.6 : 2.0}
              far={6}
              resolution={quality === 'high' ? 1024 : 512}
            />
          )}
        </>
      )}

      <group>
        <AxisArrow direction={new THREE.Vector3(1, 0, 0)} color="#f87171" />
        <AxisArrow direction={new THREE.Vector3(0, 1, 0)} color="#34d399" />
        <AxisArrow direction={new THREE.Vector3(0, 0, 1)} color="#60a5fa" />
      </group>
    </>
  );
}

function AxisArrow({ direction, color }: { direction: THREE.Vector3; color: string }) {
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

