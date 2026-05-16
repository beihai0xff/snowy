/**
 * Snowy v6 · R3F 物理预览 · 共享类型
 *
 * 注意：所有 R3F 场景共用同一组 props（artifact / propsData / running / playbackSpeed），
 * 由顶层 R3FPhysicsPreview 负责注入；具体场景通过 useRapierSimulation 拿到当前 body/trail。
 */

import type RAPIER from '@dimforge/rapier3d-compat';
import type { RenderArtifact } from '@/lib/api';

export type Vec3 = { x: number; y: number; z: number };

export type SceneKind = 'orbit' | 'spring' | 'collision' | 'projectile' | 'force' | 'motion';

export type RapierModule = typeof import('@dimforge/rapier3d-compat');

export type BodyMap = Record<string, RAPIER.RigidBody>;
export type TrailMap = Record<string, Vec3[]>;

export interface SimState {
  world: RAPIER.World;
  bodies: BodyMap;
  colliders: RAPIER.Collider[];
  startedAt: number;
  trail: Vec3[];
  trails: TrailMap;
  scene: string;
  props: Record<string, number>;
  meta: Record<string, number>;
  lastImpactAt?: number;
}

export interface SceneRenderProps {
  sim: SimState;
  kind: SceneKind;
  running: boolean;
  playbackSpeed: number;
  artifact: RenderArtifact;
}

export const numberValue = (props: Record<string, number>, key: string, fallback: number): number => {
  const value = props[key];
  return Number.isFinite(value) ? value : fallback;
};

export const clamp = (value: number, min: number, max: number) => Math.min(max, Math.max(min, value));

export function sceneKindOf(sceneType: string): SceneKind {
  if (sceneType.includes('orbit')) return 'orbit';
  if (sceneType.includes('spring')) return 'spring';
  if (sceneType.includes('collision')) return 'collision';
  if (sceneType.includes('projectile')) return 'projectile';
  if (sceneType.includes('force')) return 'force';
  return 'motion';
}

export function sceneLabel(kind: SceneKind): string {
  switch (kind) {
    case 'orbit': return '天体轨道演示';
    case 'spring': return '弹簧振子演示';
    case 'collision': return '碰撞运动演示';
    case 'projectile': return '抛体轨迹演示';
    case 'force': return '受力刚体演示';
    default: return '通用运动演示';
  }
}

export function bodyPosition(body: RAPIER.RigidBody, viewDimension: number): Vec3 {
  const p = body.translation();
  return { x: p.x, y: p.y, z: viewDimension >= 3 ? p.z : 0 };
}
