/**
 * Snowy v6 · R3F · 场景注册表
 *
 * 由 SceneKind 路由到具体场景组件。
 */

'use client';

import React from 'react';
import dynamic from 'next/dynamic';
import type { SceneKind, SimState } from './types';

const OrbitScene = dynamic(() => import('../scenes/OrbitScene'), { ssr: false });
const ProjectileScene = dynamic(() => import('../scenes/ProjectileScene'), { ssr: false });
const SpringScene = dynamic(() => import('../scenes/SpringScene'), { ssr: false });
const CollisionScene = dynamic(() => import('../scenes/CollisionScene'), { ssr: false });
const ForceScene = dynamic(() => import('../scenes/ForceScene'), { ssr: false });
const MotionScene = dynamic(() => import('../scenes/MotionScene'), { ssr: false });

interface Props {
  kind: SceneKind;
  simRef: React.MutableRefObject<SimState | null>;
}

export default function SceneByKind({ kind, simRef }: Props) {
  switch (kind) {
    case 'orbit': return <OrbitScene simRef={simRef} />;
    case 'projectile': return <ProjectileScene simRef={simRef} />;
    case 'spring': return <SpringScene simRef={simRef} />;
    case 'collision': return <CollisionScene simRef={simRef} />;
    case 'force': return <ForceScene simRef={simRef} />;
    default: return <MotionScene simRef={simRef} />;
  }
}
