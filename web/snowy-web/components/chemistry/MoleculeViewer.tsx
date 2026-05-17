/**
 * Snowy v7 · 化学 · 球棍模型查看器（M11）
 *
 * 输入：ChemistrySpeciesModel（已在后端预生成 3D 坐标）
 *   - 若 atoms 为空或 fallback_2d=true，渲染纯文本兜底（formula tag）。
 *   - 否则用 R3F 渲染球（原子）+ 圆柱（键）。
 */

'use client';

import React, { Suspense, useMemo } from 'react';
import { Canvas } from '@react-three/fiber';
import { OrbitControls } from '@react-three/drei';
import { Tag } from 'antd';
import type { ChemistrySpeciesModel } from '@/lib/api';

const ELEMENT_COLOR: Record<string, string> = {
  H: '#ffffff', C: '#404040', N: '#3050f8', O: '#ff0d0d', F: '#90e050',
  Na: '#ab5cf2', Mg: '#8aff00', Al: '#bfa6a6', P: '#ff8000', S: '#ffff30',
  Cl: '#1ff01f', K: '#8f40d4', Ca: '#3dff00', Fe: '#e06633', Cu: '#c88033',
  Zn: '#7d80b0', Br: '#a62929', I: '#940094',
};

const ELEMENT_RADIUS: Record<string, number> = {
  H: 0.22, C: 0.4, N: 0.38, O: 0.36, F: 0.34, Na: 0.5, Mg: 0.48,
  Al: 0.5, P: 0.46, S: 0.46, Cl: 0.44, K: 0.55, Ca: 0.52,
};

const colorOf = (el: string) => ELEMENT_COLOR[el] || '#888888';
const radiusOf = (el: string) => ELEMENT_RADIUS[el] || 0.4;

function AtomMesh({ x, y, z, el }: { x: number; y: number; z: number; el: string }) {
  return (
    <mesh position={[x, y, z]} castShadow receiveShadow>
      <sphereGeometry args={[radiusOf(el), 24, 24]} />
      <meshStandardMaterial color={colorOf(el)} roughness={0.45} metalness={0.05} />
    </mesh>
  );
}

function BondCylinder({
  from,
  to,
  order,
}: {
  from: [number, number, number];
  to: [number, number, number];
  order: number;
}) {
  const meta = useMemo(() => {
    const dx = to[0] - from[0];
    const dy = to[1] - from[1];
    const dz = to[2] - from[2];
    const length = Math.sqrt(dx * dx + dy * dy + dz * dz) || 0.0001;
    const mid: [number, number, number] = [
      (from[0] + to[0]) / 2,
      (from[1] + to[1]) / 2,
      (from[2] + to[2]) / 2,
    ];
    const angleX = Math.atan2(Math.sqrt(dx * dx + dz * dz), dy);
    const angleY = Math.atan2(dx, dz);
    return { length, mid, angleX, angleY };
  }, [from, to]);
  const radius = order >= 3 ? 0.12 : order === 2 ? 0.1 : 0.08;
  return (
    <group position={meta.mid} rotation={[0, meta.angleY, 0]}>
      <group rotation={[meta.angleX, 0, 0]}>
        <mesh castShadow>
          <cylinderGeometry args={[radius, radius, meta.length, 16]} />
          <meshStandardMaterial color="#bdbdbd" roughness={0.6} metalness={0.1} />
        </mesh>
        {order === 2 && (
          <mesh position={[radius * 1.8, 0, 0]} castShadow>
            <cylinderGeometry args={[radius * 0.6, radius * 0.6, meta.length * 0.95, 12]} />
            <meshStandardMaterial color="#bdbdbd" />
          </mesh>
        )}
        {order === 3 && (
          <>
            <mesh position={[radius * 1.8, 0, 0]} castShadow>
              <cylinderGeometry args={[radius * 0.55, radius * 0.55, meta.length * 0.95, 12]} />
              <meshStandardMaterial color="#bdbdbd" />
            </mesh>
            <mesh position={[-radius * 1.8, 0, 0]} castShadow>
              <cylinderGeometry args={[radius * 0.55, radius * 0.55, meta.length * 0.95, 12]} />
              <meshStandardMaterial color="#bdbdbd" />
            </mesh>
          </>
        )}
      </group>
    </group>
  );
}

export interface MoleculeViewerProps {
  species: ChemistrySpeciesModel;
  height?: number;
}

export default function MoleculeViewer({ species, height = 200 }: MoleculeViewerProps) {
  const atoms = species.atoms || [];
  const bonds = species.bonds || [];

  if (atoms.length === 0 || species.fallback_2d) {
    return (
      <div
        style={{
          height,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          background: '#fafafa',
          border: '1px dashed #d9d9d9',
          borderRadius: 8,
          flexDirection: 'column',
          gap: 4,
        }}
      >
        <Tag color="default" style={{ fontSize: 16, padding: '4px 12px' }}>{species.formula}</Tag>
        <span style={{ fontSize: 12, color: '#999' }}>暂无 3D 几何，已用化学式兜底</span>
      </div>
    );
  }

  return (
    <div style={{ height, background: '#f5f7fb', borderRadius: 8, overflow: 'hidden' }}>
      <Suspense fallback={<div style={{ padding: 12, fontSize: 12, color: '#999' }}>加载 3D 球棍模型…</div>}>
        <Canvas
          shadows
          camera={{ position: [3, 2.5, 4], fov: 45 }}
          dpr={[1, 1.5]}
          gl={{ antialias: true }}
        >
          <ambientLight intensity={0.55} />
          <directionalLight position={[5, 6, 4]} intensity={0.7} castShadow />
          <directionalLight position={[-3, 2, -4]} intensity={0.25} />
          {atoms.map((a, i) => (
            <AtomMesh key={`a${i}`} x={a.x} y={a.y} z={a.z} el={a.element} />
          ))}
          {bonds.map((b, i) => {
            const a = atoms[b.a];
            const c = atoms[b.b];
            if (!a || !c) return null;
            return (
              <BondCylinder
                key={`b${i}`}
                from={[a.x, a.y, a.z]}
                to={[c.x, c.y, c.z]}
                order={b.order}
              />
            );
          })}
          <OrbitControls enablePan={false} enableDamping minDistance={2} maxDistance={12} />
        </Canvas>
      </Suspense>
    </div>
  );
}
