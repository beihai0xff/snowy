/**
 * Snowy v6 · Rapier3D 仿真 Hook
 *
 * 把 NativePhysicsPreview 中的 createSimulation / updateSimulation / appendTrails 抽取出来，
 * 供 R3F 场景与旧 Canvas 渲染共享。Hook 返回最新 SimState（ref + tick）。
 *
 * 设计要点：
 *  - 物理 step 与渲染解耦：调用方负责在 useFrame 里 advance()；
 *  - 仿真会在 props/scene 变化时重建 world；
 *  - WebGL 不可用时本 Hook 仍然可用（不依赖 three）。
 */

'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import type RAPIER from '@dimforge/rapier3d-compat';
import {
  type BodyMap,
  type RapierModule,
  type SimState,
  type TrailMap,
  bodyPosition,
  clamp,
  numberValue,
  sceneKindOf,
} from '../lib/types';

let rapierModulePromise: Promise<RapierModule> | null = null;

async function loadRapier(): Promise<RapierModule> {
  if (!rapierModulePromise) {
    rapierModulePromise = import('@dimforge/rapier3d-compat').then(async (mod) => {
      await mod.init();
      return mod;
    });
  }
  return rapierModulePromise;
}

async function createSimulation(
  rapier: RapierModule,
  scene: string,
  props: Record<string, number>,
): Promise<SimState> {
  const kind = sceneKindOf(scene);
  const gravityY = kind === 'projectile' || kind === 'force' ? -Math.abs(numberValue(props, 'g', 9.8)) : 0;
  const world = new rapier.World({ x: 0, y: gravityY, z: 0 });
  const colliders: RAPIER.Collider[] = [];
  const bodies: BodyMap = {};
  const trails: TrailMap = {};
  const meta: Record<string, number> = {};

  if (kind !== 'orbit' && kind !== 'spring') {
    colliders.push(
      world.createCollider(rapier.ColliderDesc.cuboid(20, 0.08, 20).setTranslation(0, -0.08, 0)),
    );
  }

  if (kind === 'projectile') {
    const v0 = numberValue(props, 'v0', 20);
    const angle = (numberValue(props, 'angle_deg', 45) * Math.PI) / 180;
    const viewDimension = numberValue(props, 'view_dimension', 3);
    bodies.main = world.createRigidBody(
      rapier.RigidBodyDesc.dynamic().setTranslation(-4.5, 0.55, viewDimension >= 3 ? -2.2 : 0),
    );
    bodies.main.setLinvel(
      {
        x: Math.cos(angle) * v0 * 0.18,
        y: Math.sin(angle) * v0 * 0.18,
        z: viewDimension >= 3 ? Math.cos(angle) * v0 * 0.055 : 0,
      },
      true,
    );
    colliders.push(world.createCollider(rapier.ColliderDesc.ball(0.16).setRestitution(0.45), bodies.main));
  } else if (kind === 'force') {
    const mass = Math.max(0.1, numberValue(props, 'm', 2));
    const accel = numberValue(props, 'a', 3);
    bodies.main = world.createRigidBody(
      rapier.RigidBodyDesc.dynamic().setTranslation(-3.4, 0.65, 0).setLinearDamping(0.08),
    );
    colliders.push(
      world.createCollider(
        rapier.ColliderDesc.cuboid(0.62, 0.62, 0.62).setDensity(mass / 1.9).setFriction(0.22),
        bodies.main,
      ),
    );
    bodies.main.addForce({ x: mass * accel, y: 0, z: 0 }, true);
  } else if (kind === 'orbit') {
    const radius = numberValue(props, 'orbit_radius', 3.6);
    const speed = numberValue(props, 'tangential_speed', 2.25);
    const eccentricity = numberValue(props, 'eccentricity', 0.18);
    bodies.satellite = world.createRigidBody(
      rapier.RigidBodyDesc.dynamic()
        .setTranslation(radius * (1 + eccentricity), 0.25, 0)
        .setLinearDamping(0.002),
    );
    bodies.satellite.setLinvel({ x: 0, y: 0, z: speed }, true);
    colliders.push(world.createCollider(rapier.ColliderDesc.ball(0.16), bodies.satellite));
    trails.satellite = [];
  } else if (kind === 'spring') {
    const x = numberValue(props, 'x', 1.4);
    const damping = numberValue(props, 'damping', 0.18);
    bodies.mass = world.createRigidBody(
      rapier.RigidBodyDesc.dynamic().setTranslation(-2.4 + x, 1.2, 0).setLinearDamping(damping),
    );
    colliders.push(
      world.createCollider(
        rapier.ColliderDesc.ball(0.22).setDensity(Math.max(0.2, numberValue(props, 'm', 1.2))),
        bodies.mass,
      ),
    );
    trails.mass = [];
  } else if (kind === 'collision') {
    const m1 = Math.max(0.2, numberValue(props, 'm1', 1.5));
    const m2 = Math.max(0.2, numberValue(props, 'm2', 1));
    const e = clamp(numberValue(props, 'restitution', 0.9), 0, 1);
    bodies.ball1 = world.createRigidBody(
      rapier.RigidBodyDesc.dynamic().setTranslation(-3.2, 0.45, -0.6).setLinearDamping(0.01),
    );
    bodies.ball2 = world.createRigidBody(
      rapier.RigidBodyDesc.dynamic().setTranslation(3.2, 0.45, 0.6).setLinearDamping(0.01),
    );
    bodies.ball1.setLinvel({ x: numberValue(props, 'v1', 4.5), y: 0, z: 0.3 }, true);
    bodies.ball2.setLinvel({ x: numberValue(props, 'v2', -2.5), y: 0, z: -0.3 }, true);
    colliders.push(
      world.createCollider(rapier.ColliderDesc.ball(0.42).setDensity(m1).setRestitution(e), bodies.ball1),
    );
    colliders.push(
      world.createCollider(rapier.ColliderDesc.ball(0.42).setDensity(m2).setRestitution(e), bodies.ball2),
    );
    trails.ball1 = [];
    trails.ball2 = [];
    meta.initial_energy =
      0.5 * m1 * numberValue(props, 'v1', 4.5) ** 2 + 0.5 * m2 * numberValue(props, 'v2', -2.5) ** 2;
  } else {
    bodies.main = world.createRigidBody(
      rapier.RigidBodyDesc.dynamic().setTranslation(-4, 0.55, 0).setLinearDamping(0.04),
    );
    bodies.main.setLinvel(
      { x: numberValue(props, 'v', numberValue(props, 'v0', 5)) * 0.25, y: 0, z: 0 },
      true,
    );
    colliders.push(world.createCollider(rapier.ColliderDesc.ball(0.24).setRestitution(0.35), bodies.main));
  }

  return {
    world,
    bodies,
    colliders,
    startedAt: performance.now(),
    trail: [],
    trails,
    scene,
    props: { ...props },
    meta,
  };
}

function stepSimulation(state: SimState, effectiveDelta: number) {
  const kind = sceneKindOf(state.scene);
  if (kind === 'force') {
    const mass = Math.max(0.1, numberValue(state.props, 'm', 2));
    const accel = numberValue(state.props, 'a', 3);
    state.bodies.main?.addForce({ x: mass * accel, y: 0, z: 0 }, true);
  } else if (kind === 'orbit') {
    const body = state.bodies.satellite;
    if (body) {
      const pos = body.translation();
      const dx = -pos.x;
      const dz = -pos.z;
      const distSq = Math.max(0.55, dx * dx + dz * dz);
      const dist = Math.sqrt(distSq);
      const strength =
        numberValue(state.props, 'gravitational_strength', 10) *
        numberValue(state.props, 'central_mass', 8) *
        0.045;
      body.addForce({ x: (dx / dist) * strength / distSq, y: 0, z: (dz / dist) * strength / distSq }, true);
    }
  } else if (kind === 'spring') {
    const body = state.bodies.mass;
    if (body) {
      const pos = body.translation();
      const vel = body.linvel();
      const restX = -2.4;
      const displacement = pos.x - restX;
      const k = numberValue(state.props, 'k', 24) * 0.16;
      const damping = numberValue(state.props, 'damping', 0.18);
      const forceX = -k * displacement - damping * vel.x;
      body.addForce({ x: forceX, y: 0, z: 0 }, true);
      state.meta.spring_x = displacement;
      state.meta.kinetic = 0.5 * Math.max(0.1, numberValue(state.props, 'm', 1.2)) * vel.x * vel.x;
      state.meta.potential = 0.5 * k * displacement * displacement;
      state.meta.energy = state.meta.kinetic + state.meta.potential;
    }
  } else if (kind === 'collision') {
    const b1 = state.bodies.ball1;
    const b2 = state.bodies.ball2;
    if (b1 && b2) {
      const p1 = b1.translation();
      const p2 = b2.translation();
      const dx = p1.x - p2.x;
      const dz = p1.z - p2.z;
      if (Math.hypot(dx, dz) < 0.9) state.lastImpactAt = performance.now();
      const v1 = b1.linvel();
      const v2 = b2.linvel();
      const m1 = Math.max(0.2, numberValue(state.props, 'm1', 1.5));
      const m2 = Math.max(0.2, numberValue(state.props, 'm2', 1));
      state.meta.energy = 0.5 * m1 * (v1.x * v1.x + v1.z * v1.z) + 0.5 * m2 * (v2.x * v2.x + v2.z * v2.z);
    }
  } else if (kind === 'motion') {
    const a = numberValue(state.props, 'a', 0);
    state.bodies.main?.addForce({ x: a, y: 0, z: 0 }, true);
  }

  state.world.timestep = effectiveDelta;
  state.world.step();
}

function appendTrails(state: SimState, viewDimension: number) {
  const maxTrail = Math.floor(clamp(numberValue(state.props, 'trail_length', 180), 40, 360));
  const kind = sceneKindOf(state.scene);
  const add = (name: string, body: RAPIER.RigidBody) => {
    const list = state.trails[name] || [];
    list.push(bodyPosition(body, viewDimension));
    if (list.length > maxTrail) list.shift();
    state.trails[name] = list;
  };
  if (kind === 'orbit' && state.bodies.satellite) add('satellite', state.bodies.satellite);
  else if (kind === 'spring' && state.bodies.mass) add('mass', state.bodies.mass);
  else if (kind === 'collision') {
    if (state.bodies.ball1) add('ball1', state.bodies.ball1);
    if (state.bodies.ball2) add('ball2', state.bodies.ball2);
  } else if (state.bodies.main) {
    const pos = bodyPosition(state.bodies.main, viewDimension);
    state.trail.push(pos);
    if (state.trail.length > maxTrail) state.trail.shift();
  }
}

function shouldAutoReset(state: SimState, now: number): boolean {
  const kind = sceneKindOf(state.scene);
  const main = state.bodies.main || state.bodies.satellite || state.bodies.mass || state.bodies.ball1;
  if (!main) return false;
  const pos = main.translation();
  if (kind === 'projectile' && pos.y < -1) return true;
  if (Math.abs(pos.x) > 14 || Math.abs(pos.z) > 14) return true;
  if (now - state.startedAt > 26000) return true;
  return false;
}

export type RapierStatus = 'loading' | 'ready' | 'error';

export interface RapierSimulationHandle {
  /** 当前仿真状态（ref，避免触发 React 渲染） */
  ref: React.MutableRefObject<SimState | null>;
  status: RapierStatus;
  error: string | null;
  /** 由 useFrame 调用，自动 step + trail + 自动重置 */
  advance: (deltaSeconds: number, playbackSpeed: number, viewDimension: number) => void;
  /** 手动重置仿真 */
  reset: () => void;
}

export function useRapierSimulation(
  sceneType: string,
  props: Record<string, number>,
  options?: { running?: boolean },
): RapierSimulationHandle {
  const simRef = useRef<SimState | null>(null);
  const rapierRef = useRef<RapierModule | null>(null);
  const [status, setStatus] = useState<RapierStatus>('loading');
  const [error, setError] = useState<string | null>(null);
  const [resetSeq, setResetSeq] = useState(0);
  const lastAutoResetRef = useRef(0);

  useEffect(() => {
    let disposed = false;
    setStatus('loading');
    setError(null);
    loadRapier()
      .then(async (mod) => {
        if (disposed) return;
        rapierRef.current = mod;
        try {
          simRef.current?.world.free();
        } catch {
          /* ignore */
        }
        const sim = await createSimulation(mod, sceneType, props);
        if (disposed) {
          sim.world.free();
          return;
        }
        simRef.current = sim;
        setStatus('ready');
      })
      .catch((err: unknown) => {
        if (disposed) return;
        setError(err instanceof Error ? err.message : 'Rapier 初始化失败');
        setStatus('error');
      });

    return () => {
      disposed = true;
      try {
        simRef.current?.world.free();
      } catch {
        /* ignore */
      }
      simRef.current = null;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sceneType, resetSeq]);

  useEffect(() => {
    const mod = rapierRef.current;
    if (!mod || status !== 'ready') return;
    try {
      simRef.current?.world.free();
    } catch {
      /* ignore */
    }
    void createSimulation(mod, sceneType, props)
      .then((sim) => {
        simRef.current = sim;
      })
      .catch((err: unknown) => {
        setError(err instanceof Error ? err.message : '参数同步失败');
        setStatus('error');
      });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [JSON.stringify(props)]);

  const advance = useCallback(
    (deltaSeconds: number, playbackSpeed: number, viewDimension: number) => {
      const sim = simRef.current;
      if (!sim || options?.running === false) return;
      const realDelta = Math.min(0.05, deltaSeconds);
      const effectiveDelta = Math.min(0.2, realDelta * playbackSpeed);
      const fixedStep = 1 / 60;
      const steps = Math.max(1, Math.min(18, Math.ceil(effectiveDelta / fixedStep)));
      const stepSize = effectiveDelta / steps;
      for (let i = 0; i < steps; i += 1) stepSimulation(sim, stepSize);
      appendTrails(sim, viewDimension);
      const now = performance.now();
      if (shouldAutoReset(sim, now) && now - lastAutoResetRef.current > 800) {
        lastAutoResetRef.current = now;
        setResetSeq((value) => value + 1);
      }
    },
    [options?.running],
  );

  const reset = useCallback(() => setResetSeq((value) => value + 1), []);

  return { ref: simRef, status, error, advance, reset };
}
