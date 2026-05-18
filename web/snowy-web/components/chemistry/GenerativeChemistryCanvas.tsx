/**
 * Snowy v8 · 生成式化学可视化画布
 *
 * 首版目标：把 chemistry domain 接入统一建模页，提供球棍模型 + 配平动画 + 电子流提示。
 * 后续可直接替换为 v7 §5 后端返回的完整 ChemistryReactionPackage。
 */

'use client';

import React, { useMemo, useRef } from 'react';
import * as THREE from 'three';
import { Canvas, useFrame } from '@react-three/fiber';
import { OrbitControls, Sparkles, Text } from '@react-three/drei';
import { Card, Space, Tag, Typography } from 'antd';
import type { GenerativeModelPackage } from '@/lib/api';

const { Paragraph } = Typography;

type AtomKey = 'H' | 'O' | 'C' | 'N' | 'Na' | 'Cl' | 'S' | 'Fe' | 'Cu';

interface AtomSpec {
  id: string;
  element: AtomKey;
  position: [number, number, number];
}

interface BondSpec {
  a: string;
  b: string;
  order?: 1 | 2 | 3;
}

interface MoleculeSpec {
  label: string;
  atoms: AtomSpec[];
  bonds: BondSpec[];
  offset: [number, number, number];
}

const ATOM_STYLE: Record<AtomKey, { color: string; radius: number; label: string }> = {
  H:  { color: '#f8fafc', radius: 0.18, label: 'H' },
  O:  { color: '#ef4444', radius: 0.28, label: 'O' },
  C:  { color: '#1f2937', radius: 0.30, label: 'C' },
  N:  { color: '#2563eb', radius: 0.30, label: 'N' },
  Na: { color: '#a78bfa', radius: 0.36, label: 'Na' },
  Cl: { color: '#22c55e', radius: 0.34, label: 'Cl' },
  S:  { color: '#facc15', radius: 0.38, label: 'S' },
  Fe: { color: '#fb923c', radius: 0.38, label: 'Fe' },
  Cu: { color: '#06b6d4', radius: 0.38, label: 'Cu' },
};

function water(label = 'H₂O', offset: [number, number, number] = [0, 0, 0]): MoleculeSpec {
  return {
    label,
    offset,
    atoms: [
      { id: 'O', element: 'O', position: [0, 0, 0] },
      { id: 'H1', element: 'H', position: [-0.55, 0.38, 0] },
      { id: 'H2', element: 'H', position: [0.55, 0.38, 0] },
    ],
    bonds: [{ a: 'O', b: 'H1' }, { a: 'O', b: 'H2' }],
  };
}

function diatomic(label: string, element: AtomKey, offset: [number, number, number]): MoleculeSpec {
  return {
    label,
    offset,
    atoms: [
      { id: 'A', element, position: [-0.32, 0, 0] },
      { id: 'B', element, position: [0.32, 0, 0] },
    ],
    bonds: [{ a: 'A', b: 'B', order: element === 'O' ? 2 : 1 }],
  };
}

function sodiumHydroxide(offset: [number, number, number]): MoleculeSpec {
  return {
    label: 'NaOH',
    offset,
    atoms: [
      { id: 'Na', element: 'Na', position: [-0.62, 0, 0] },
      { id: 'O', element: 'O', position: [0, 0, 0] },
      { id: 'H', element: 'H', position: [0.48, 0.28, 0] },
    ],
    bonds: [{ a: 'Na', b: 'O' }, { a: 'O', b: 'H' }],
  };
}

function sulfuricAcid(offset: [number, number, number]): MoleculeSpec {
  return {
    label: 'H₂SO₄',
    offset,
    atoms: [
      { id: 'S', element: 'S', position: [0, 0, 0] },
      { id: 'O1', element: 'O', position: [-0.6, 0.45, 0] },
      { id: 'O2', element: 'O', position: [0.6, 0.45, 0] },
      { id: 'O3', element: 'O', position: [-0.6, -0.45, 0] },
      { id: 'O4', element: 'O', position: [0.6, -0.45, 0] },
      { id: 'H1', element: 'H', position: [-1.0, -0.68, 0] },
      { id: 'H2', element: 'H', position: [1.0, -0.68, 0] },
    ],
    bonds: [
      { a: 'S', b: 'O1', order: 2 }, { a: 'S', b: 'O2', order: 2 },
      { a: 'S', b: 'O3' }, { a: 'S', b: 'O4' },
      { a: 'O3', b: 'H1' }, { a: 'O4', b: 'H2' },
    ],
  };
}

function sodiumSulfate(offset: [number, number, number]): MoleculeSpec {
  const acid = sulfuricAcid(offset);
  return {
    ...acid,
    label: 'Na₂SO₄',
    atoms: acid.atoms.filter((a) => !a.id.startsWith('H')).concat([
      { id: 'Na1', element: 'Na', position: [-1.1, -0.65, 0] },
      { id: 'Na2', element: 'Na', position: [1.1, -0.65, 0] },
    ]),
    bonds: acid.bonds.filter((b) => !b.b.includes('H')),
  };
}

function deriveReaction(question: string): { title: string; type: string; equation: string; molecules: MoleculeSpec[] } {
  const q = question.toLowerCase();
  if (/电解|water|h2o/.test(q)) {
    return {
      title: '电解水：分解反应',
      type: 'electrolysis',
      equation: '2H₂O → 2H₂ + O₂',
      molecules: [water('2H₂O', [-3.0, 0, 0]), diatomic('2H₂', 'H', [0.6, 0.7, 0]), diatomic('O₂', 'O', [2.8, -0.5, 0])],
    };
  }
  if (/naoh|h2so4|硫酸|中和/.test(q)) {
    return {
      title: '酸碱中和：配平演示',
      type: 'neutralization',
      equation: '2NaOH + H₂SO₄ → Na₂SO₄ + 2H₂O',
      molecules: [sodiumHydroxide([-4.0, 0.65, 0]), sulfuricAcid([-1.6, -0.15, 0]), sodiumSulfate([1.7, 0.1, 0]), water('2H₂O', [4.0, -0.45, 0])],
    };
  }
  return {
    title: '氧化还原 / 反应路径演示',
    type: 'redox',
    equation: 'Fe + Cu²⁺ → Fe²⁺ + Cu',
    molecules: [
      { label: 'Fe', offset: [-2.8, 0, 0], atoms: [{ id: 'Fe', element: 'Fe', position: [0, 0, 0] }], bonds: [] },
      { label: 'Cu²⁺', offset: [-0.7, 0, 0], atoms: [{ id: 'Cu', element: 'Cu', position: [0, 0, 0] }], bonds: [] },
      { label: 'Fe²⁺', offset: [1.4, 0, 0], atoms: [{ id: 'Fe', element: 'Fe', position: [0, 0, 0] }], bonds: [] },
      { label: 'Cu', offset: [3.1, 0, 0], atoms: [{ id: 'Cu', element: 'Cu', position: [0, 0, 0] }], bonds: [] },
    ],
  };
}

function Bond({ a, b, order = 1 }: { a: THREE.Vector3; b: THREE.Vector3; order?: 1 | 2 | 3 }) {
  const mid = useMemo(() => a.clone().add(b).multiplyScalar(0.5), [a, b]);
  const dir = useMemo(() => b.clone().sub(a), [a, b]);
  const quat = useMemo(() => new THREE.Quaternion().setFromUnitVectors(new THREE.Vector3(0, 1, 0), dir.clone().normalize()), [dir]);
  const length = dir.length();
  return (
    <group position={mid} quaternion={quat}>
      {Array.from({ length: order }).map((_, i) => {
        const shift = (i - (order - 1) / 2) * 0.08;
        return (
          <mesh key={i} position={[shift, 0, 0]} castShadow>
            <cylinderGeometry args={[0.035, 0.035, length, 16]} />
            <meshStandardMaterial color="#cbd5e1" metalness={0.25} roughness={0.35} />
          </mesh>
        );
      })}
    </group>
  );
}

function Molecule({ spec }: { spec: MoleculeSpec }) {
  const groupRef = useRef<THREE.Group | null>(null);
  useFrame(({ clock }) => {
    if (!groupRef.current) return;
    groupRef.current.rotation.y = Math.sin(clock.elapsedTime * 0.45 + spec.offset[0]) * 0.18;
    groupRef.current.position.y = spec.offset[1] + Math.sin(clock.elapsedTime * 1.1 + spec.offset[0]) * 0.04;
  });
  const atoms = useMemo(() => new Map(spec.atoms.map((a) => [a.id, a])), [spec.atoms]);
  return (
    <group ref={groupRef} position={spec.offset}>
      {spec.bonds.map((bond, idx) => {
        const a = atoms.get(bond.a);
        const b = atoms.get(bond.b);
        if (!a || !b) return null;
        return <Bond key={`${bond.a}-${bond.b}-${idx}`} a={new THREE.Vector3(...a.position)} b={new THREE.Vector3(...b.position)} order={bond.order} />;
      })}
      {spec.atoms.map((atom) => {
        const style = ATOM_STYLE[atom.element];
        return (
          <group key={atom.id} position={atom.position}>
            <mesh castShadow>
              <sphereGeometry args={[style.radius, 40, 40]} />
              <meshStandardMaterial color={style.color} roughness={0.18} metalness={0.35} emissive={style.color} emissiveIntensity={0.08} />
            </mesh>
            <Text position={[0, style.radius + 0.14, 0]} fontSize={0.14} color="#e2e8f0" anchorX="center" anchorY="middle">
              {style.label}
            </Text>
          </group>
        );
      })}
      <Text position={[0, -0.9, 0]} fontSize={0.18} color="#c4b5fd" anchorX="center" anchorY="middle">
        {spec.label}
      </Text>
    </group>
  );
}

function ElectronFlow({ type }: { type: string }) {
  const ref = useRef<THREE.Mesh | null>(null);
  useFrame(({ clock }) => {
    if (!ref.current) return;
    const t = (Math.sin(clock.elapsedTime * 2.2) + 1) / 2;
    ref.current.position.set(-0.9 + t * 2.2, 1.45, 0);
    ref.current.scale.setScalar(0.8 + t * 0.35);
  });
  if (type !== 'redox' && type !== 'electrolysis') return null;
  return (
    <group>
      <mesh ref={ref}>
        <sphereGeometry args={[0.08, 20, 20]} />
        <meshBasicMaterial color="#facc15" toneMapped={false} />
      </mesh>
      <Text position={[0.2, 1.75, 0]} fontSize={0.16} color="#facc15" anchorX="center" anchorY="middle">
        e⁻ transfer
      </Text>
    </group>
  );
}

function ChemistryScene({ reaction }: { reaction: ReturnType<typeof deriveReaction> }) {
  return (
    <Canvas
      camera={{ position: [0, 3.0, 7.5], fov: 45 }}
      shadows
      gl={{ antialias: true, alpha: false, powerPreference: 'high-performance' }}
    >
      <color attach="background" args={["#080716"]} />
      <fog attach="fog" args={["#080716", 9, 18]} />
      <ambientLight intensity={0.35} />
      <hemisphereLight intensity={0.5} color="#e0e7ff" groundColor="#111827" />
      <directionalLight position={[4, 6, 4]} intensity={1.2} color="#ddd6fe" castShadow />
      <pointLight position={[-4, 2.2, -2]} intensity={0.7} color="#22d3ee" />
      <pointLight position={[4, 1.5, 3]} intensity={0.55} color="#a78bfa" />
      <Sparkles count={90} scale={7} size={1.3} color="#c4b5fd" speed={0.35} />
      {reaction.molecules.map((molecule) => <Molecule key={`${molecule.label}-${molecule.offset.join(',')}`} spec={molecule} />)}
      <ElectronFlow type={reaction.type} />
      <OrbitControls enablePan={false} enableDamping dampingFactor={0.08} minDistance={4} maxDistance={12} />
    </Canvas>
  );
}

interface Props {
  pkg: GenerativeModelPackage;
  values?: Record<string, number>;
}

export default function GenerativeChemistryCanvas({ pkg }: Props) {
  const reaction = useMemo(() => deriveReaction(`${pkg.question} ${pkg.learning_model?.topic || ''}`), [pkg.question, pkg.learning_model?.topic]);
  return (
    <Space direction="vertical" style={{ width: '100%' }} size="middle">
      <Space wrap>
        <Tag color="purple">化学建模</Tag>
        <Tag color="geekblue">{reaction.type}</Tag>
        <Tag color="cyan">球棍模型</Tag>
        <Tag color="gold">配平动画</Tag>
      </Space>
      <div className="snowy-chemistry-canvas">
        <div className="snowy-chemistry-equation">{reaction.equation}</div>
        <ChemistryScene reaction={reaction} />
      </div>
      <Card size="small" title={reaction.title}>
        <Paragraph style={{ marginBottom: 0 }}>
          当前版本根据题干识别典型高中化学反应类型，展示原子球棍、配平结果与电子/离子变化方向；后续可无缝替换为后端返回的完整 ChemistryReactionPackage。
        </Paragraph>
      </Card>
    </Space>
  );
}

