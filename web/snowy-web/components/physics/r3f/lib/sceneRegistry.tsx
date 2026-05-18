/**
 * Snowy v6/v8 · R3F · 场景注册表
 * v8: 透传 quality 给各 scene；改同步 import 以保留 TS prop 类型推断。
 *     R3FPhysicsPreview 已在外层 dynamic import，故内层无需再 dynamic。
 */

'use client';

import React from 'react';
import type { QualityLevel } from './postFX';
import type { SceneKind, SimState } from './types';
import OrbitScene from '../scenes/OrbitScene';
import ProjectileScene from '../scenes/ProjectileScene';
import SpringScene from '../scenes/SpringScene';
import CollisionScene from '../scenes/CollisionScene';
import ForceScene from '../scenes/ForceScene';
import MotionScene from '../scenes/MotionScene';

interface Props {
  kind: SceneKind;
  simRef: React.MutableRefObject<SimState | null>;
  quality?: QualityLevel;
}

export default function SceneByKind({ kind, simRef, quality = 'standard' }: Props) {
  switch (kind) {
    case 'orbit':      return <OrbitScene simRef={simRef} quality={quality} />;
    case 'projectile': return <ProjectileScene simRef={simRef} quality={quality} />;
    case 'spring':     return <SpringScene simRef={simRef} quality={quality} />;
    case 'collision':  return <CollisionScene simRef={simRef} quality={quality} />;
    case 'force':      return <ForceScene simRef={simRef} quality={quality} />;
    default:           return <MotionScene simRef={simRef} quality={quality} />;
  }
}

