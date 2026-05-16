/**
 * Snowy v6 · R3F · 矢量箭头组件
 * 用于在 3D 场景中表示力、速度、加速度等矢量。
 */

'use client';

import React, { useMemo } from 'react';
import * as THREE from 'three';
import { Line, Text } from '@react-three/drei';

interface Props {
  from: [number, number, number];
  to: [number, number, number];
  color: string;
  label?: string;
  lineWidth?: number;
}

export default function VectorArrow({ from, to, color, label, lineWidth = 3 }: Props) {
  const data = useMemo(() => {
    const start = new THREE.Vector3(...from);
    const end = new THREE.Vector3(...to);
    const dir = end.clone().sub(start);
    const length = dir.length();
    if (length < 0.001) return null;
    dir.normalize();
    const headBase = end.clone().sub(dir.clone().multiplyScalar(Math.min(0.32, length * 0.25)));
    const quat = new THREE.Quaternion();
    quat.setFromUnitVectors(new THREE.Vector3(0, 1, 0), dir);
    return { start, end, headBase, quat, headSize: Math.min(0.22, length * 0.22) };
  }, [from, to]);

  if (!data) return null;
  return (
    <group>
      <Line points={[data.start.toArray(), data.headBase.toArray()]} color={color} lineWidth={lineWidth} />
      <mesh position={data.headBase.toArray()} quaternion={data.quat}>
        <coneGeometry args={[data.headSize * 0.6, data.headSize * 1.4, 12]} />
        <meshBasicMaterial color={color} />
      </mesh>
      {label && (
        <Text
          position={[data.end.x + 0.1, data.end.y + 0.18, data.end.z]}
          fontSize={0.28}
          color={color}
          outlineWidth={0.012}
          outlineColor="#0f172a"
        >
          {label}
        </Text>
      )}
    </group>
  );
}
