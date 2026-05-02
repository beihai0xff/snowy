'use client';

import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Alert, Button, Card, Space, Tag, Typography } from 'antd';
import { PauseCircleOutlined, PlayCircleOutlined, ReloadOutlined } from '@ant-design/icons';
import type RAPIER from '@dimforge/rapier3d-compat';
import type { RenderArtifact } from '@/lib/api';
import type { PreviewStatus } from '@/components/common/RenderPreviewSandbox';

const { Text } = Typography;

type RapierModule = typeof import('@dimforge/rapier3d-compat');
type Vec3 = { x: number; y: number; z: number };
type Point2 = { x: number; y: number; scale: number };

type NativePhysicsPreviewProps = {
  artifact: RenderArtifact;
  propsData: Record<string, number>;
  onStatusChange?: (status: PreviewStatus, detail?: string) => void;
};

type CameraState = { yaw: number; pitch: number; distance: number };
type TrailMap = Record<string, Vec3[]>;
type BodyMap = Record<string, RAPIER.RigidBody>;

type SimState = {
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
};

const clamp = (value: number, min: number, max: number) => Math.min(max, Math.max(min, value));
const numberValue = (props: Record<string, number>, key: string, fallback: number) => {
  const value = props[key];
  return Number.isFinite(value) ? value : fallback;
};

function isNativePhysicsArtifact(artifact: RenderArtifact): boolean {
  return artifact.render_manifest.framework === 'snowy-native-physics-engine'
    || artifact.render_manifest.dependencies?.includes('rapier3d') === true;
}

function sceneKind(sceneType: string): 'orbit' | 'spring' | 'collision' | 'projectile' | 'force' | 'motion' {
  if (sceneType.includes('orbit')) return 'orbit';
  if (sceneType.includes('spring')) return 'spring';
  if (sceneType.includes('collision')) return 'collision';
  if (sceneType.includes('projectile')) return 'projectile';
  if (sceneType.includes('force')) return 'force';
  return 'motion';
}

function sceneLabel(sceneType: string): string {
  switch (sceneKind(sceneType)) {
    case 'orbit': return '天体轨道演示';
    case 'spring': return '弹簧振子演示';
    case 'collision': return '碰撞运动演示';
    case 'projectile': return '抛体轨迹演示';
    case 'force': return '受力刚体演示';
    default: return '通用运动演示';
  }
}

function project(point: Vec3, width: number, height: number, camera: CameraState): Point2 {
  const cy = Math.cos(camera.yaw);
  const sy = Math.sin(camera.yaw);
  const cp = Math.cos(camera.pitch);
  const sp = Math.sin(camera.pitch);
  const x1 = point.x * cy - point.z * sy;
  const z1 = point.x * sy + point.z * cy;
  const y1 = point.y * cp - z1 * sp;
  const z2 = point.y * sp + z1 * cp + camera.distance;
  const scale = Math.min(width, height) * 0.82 / Math.max(2, z2);
  return { x: width / 2 + x1 * scale, y: height * 0.58 - y1 * scale, scale };
}

function roundRect(ctx: CanvasRenderingContext2D, x: number, y: number, w: number, h: number, r: number) {
  ctx.beginPath();
  ctx.roundRect(x, y, w, h, r);
}

function drawArrow(ctx: CanvasRenderingContext2D, from: Point2 | { x: number; y: number }, to: Point2 | { x: number; y: number }, color: string, label: string, width = 4) {
  const dx = to.x - from.x;
  const dy = to.y - from.y;
  const length = Math.hypot(dx, dy) || 1;
  const ux = dx / length;
  const uy = dy / length;
  ctx.save();
  ctx.strokeStyle = color;
  ctx.fillStyle = color;
  ctx.shadowColor = color;
  ctx.shadowBlur = 18;
  ctx.lineWidth = width;
  ctx.lineCap = 'round';
  ctx.beginPath();
  ctx.moveTo(from.x, from.y);
  ctx.lineTo(to.x, to.y);
  ctx.stroke();
  ctx.beginPath();
  ctx.moveTo(to.x, to.y);
  ctx.lineTo(to.x - ux * 16 - uy * 7, to.y - uy * 16 + ux * 7);
  ctx.lineTo(to.x - ux * 16 + uy * 7, to.y - uy * 16 - ux * 7);
  ctx.closePath();
  ctx.fill();
  ctx.shadowBlur = 0;
  ctx.font = '800 13px Inter, sans-serif';
  ctx.fillText(label, to.x + 10, to.y - 8);
  ctx.restore();
}

function drawStarField(ctx: CanvasRenderingContext2D, width: number, height: number, time: number) {
  ctx.save();
  for (let i = 0; i < 95; i += 1) {
    const x = (Math.sin(i * 12.9898) * 43758.5453 % 1 + 1) % 1 * width;
    const y = (Math.sin(i * 78.233) * 24634.6345 % 1 + 1) % 1 * height;
    const pulse = 0.25 + 0.55 * (0.5 + 0.5 * Math.sin(time * 0.0015 + i));
    ctx.fillStyle = `rgba(186,230,253,${pulse})`;
    ctx.beginPath();
    ctx.arc(x, y, i % 7 === 0 ? 1.8 : 1.0, 0, Math.PI * 2);
    ctx.fill();
  }
  ctx.restore();
}

function drawBackground(ctx: CanvasRenderingContext2D, width: number, height: number, time: number, kind: string) {
  const gradient = ctx.createLinearGradient(0, 0, width, height);
  gradient.addColorStop(0, kind === 'orbit' ? '#030712' : '#07111f');
  gradient.addColorStop(0.48, '#0f172a');
  gradient.addColorStop(1, kind === 'spring' ? '#172033' : '#111827');
  ctx.fillStyle = gradient;
  ctx.fillRect(0, 0, width, height);
  drawStarField(ctx, width, height, time);

  ctx.save();
  const glow = ctx.createRadialGradient(width * 0.28, height * 0.2, 0, width * 0.28, height * 0.2, width * 0.55);
  glow.addColorStop(0, kind === 'collision' ? 'rgba(251,113,133,0.22)' : 'rgba(34,211,238,0.18)');
  glow.addColorStop(1, 'rgba(34,211,238,0)');
  ctx.fillStyle = glow;
  ctx.fillRect(0, 0, width, height);
  ctx.restore();
}

function drawAxisLabel(ctx: CanvasRenderingContext2D, point: Point2, label: string, color: string, dx = 8, dy = -8) {
  ctx.save();
  ctx.font = '800 14px Inter, sans-serif';
  ctx.fillStyle = color;
  ctx.shadowColor = color;
  ctx.shadowBlur = 8;
  ctx.fillText(label, point.x + dx, point.y + dy);
  ctx.restore();
}

function drawAxisTicks(ctx: CanvasRenderingContext2D, width: number, height: number, camera: CameraState, axis: 'x' | 'z', color: string) {
  ctx.save();
  ctx.font = '11px Inter, sans-serif';
  ctx.fillStyle = color;
  ctx.strokeStyle = color;
  ctx.lineWidth = 1.2;
  for (let i = -8; i <= 8; i += 2) {
    if (i === 0) continue;
    const point = axis === 'x' ? project({ x: i, y: 0, z: 0 }, width, height, camera) : project({ x: 0, y: 0, z: i }, width, height, camera);
    const tickA = axis === 'x' ? project({ x: i, y: 0, z: -0.12 }, width, height, camera) : project({ x: -0.12, y: 0, z: i }, width, height, camera);
    const tickB = axis === 'x' ? project({ x: i, y: 0, z: 0.12 }, width, height, camera) : project({ x: 0.12, y: 0, z: i }, width, height, camera);
    ctx.beginPath();
    ctx.moveTo(tickA.x, tickA.y);
    ctx.lineTo(tickB.x, tickB.y);
    ctx.stroke();
    ctx.fillText(String(i), point.x + 4, point.y + 14);
  }
  ctx.restore();
}

function drawCoordinateLegend(ctx: CanvasRenderingContext2D) {
  ctx.save();
  const x = 18;
  const y = 18;
  ctx.fillStyle = 'rgba(15,23,42,0.72)';
  ctx.strokeStyle = 'rgba(148,163,184,0.24)';
  ctx.lineWidth = 1;
  roundRect(ctx, x, y, 192, 98, 14);
  ctx.fill();
  ctx.stroke();
  ctx.font = '800 13px Inter, sans-serif';
  ctx.fillStyle = '#e2e8f0';
  ctx.fillText('坐标 / 视角', x + 14, y + 24);
  const rows = [['X', '水平位移 / 主运动方向', '#f87171'], ['Y', '高度 / 竖直方向', '#34d399'], ['Z', '空间深度方向', '#60a5fa']] as const;
  ctx.font = '12px Inter, sans-serif';
  rows.forEach(([axis, text, color], index) => {
    const yy = y + 46 + index * 18;
    ctx.fillStyle = color;
    ctx.fillText(axis, x + 14, yy);
    ctx.fillStyle = '#cbd5e1';
    ctx.fillText(text, x + 36, yy);
  });
  ctx.restore();
}

function drawGrid(ctx: CanvasRenderingContext2D, width: number, height: number, camera: CameraState, options: { compact?: boolean } = {}) {
  ctx.save();
  ctx.lineWidth = 1;
  const extent = options.compact ? 6 : 8;
  for (let i = -extent; i <= extent; i += 1) {
    const a = project({ x: -extent, y: 0, z: i }, width, height, camera);
    const b = project({ x: extent, y: 0, z: i }, width, height, camera);
    const c = project({ x: i, y: 0, z: -extent }, width, height, camera);
    const d = project({ x: i, y: 0, z: extent }, width, height, camera);
    ctx.strokeStyle = i === 0 ? 'rgba(125,211,252,0.55)' : 'rgba(148,163,184,0.13)';
    ctx.beginPath();
    ctx.moveTo(a.x, a.y);
    ctx.lineTo(b.x, b.y);
    ctx.moveTo(c.x, c.y);
    ctx.lineTo(d.x, d.y);
    ctx.stroke();
  }

  const origin = project({ x: 0, y: 0, z: 0 }, width, height, camera);
  const xEnd = project({ x: extent + 0.8, y: 0, z: 0 }, width, height, camera);
  const xNeg = project({ x: -extent - 0.8, y: 0, z: 0 }, width, height, camera);
  const yEnd = project({ x: 0, y: 4.8, z: 0 }, width, height, camera);
  const zEnd = project({ x: 0, y: 0, z: extent + 0.8 }, width, height, camera);
  const zNeg = project({ x: 0, y: 0, z: -extent - 0.8 }, width, height, camera);

  ctx.save();
  ctx.lineWidth = 2.2;
  ctx.setLineDash([7, 7]);
  ctx.strokeStyle = 'rgba(248,113,113,0.35)';
  ctx.beginPath();
  ctx.moveTo(xNeg.x, xNeg.y);
  ctx.lineTo(origin.x, origin.y);
  ctx.stroke();
  ctx.strokeStyle = 'rgba(96,165,250,0.35)';
  ctx.beginPath();
  ctx.moveTo(zNeg.x, zNeg.y);
  ctx.lineTo(origin.x, origin.y);
  ctx.stroke();
  ctx.restore();

  drawArrow(ctx, origin, xEnd, '#f87171', 'X', 3.2);
  drawArrow(ctx, origin, yEnd, '#34d399', 'Y', 3.2);
  drawArrow(ctx, origin, zEnd, '#60a5fa', 'Z', 3.2);
  drawAxisTicks(ctx, width, height, camera, 'x', 'rgba(248,113,113,0.82)');
  drawAxisTicks(ctx, width, height, camera, 'z', 'rgba(96,165,250,0.82)');
  drawAxisLabel(ctx, origin, 'O', '#e2e8f0', 8, 16);
  drawCoordinateLegend(ctx);
  ctx.restore();
}

function drawTrail(ctx: CanvasRenderingContext2D, trail: Vec3[], width: number, height: number, camera: CameraState, color: string, viewDimension: number, radius = 2.8) {
  trail.forEach((p, index) => {
    const alpha = 0.08 + (index / Math.max(1, trail.length)) * 0.72;
    const point = project({ x: p.x, y: p.y, z: viewDimension >= 3 ? p.z : 0 }, width, height, camera);
    ctx.fillStyle = color.replace('ALPHA', alpha.toFixed(2));
    ctx.beginPath();
    ctx.arc(point.x, point.y, radius + index / Math.max(20, trail.length), 0, Math.PI * 2);
    ctx.fill();
  });
}

function drawGlowSphere(ctx: CanvasRenderingContext2D, point: Point2, radius: number, color: string, label?: string) {
  ctx.save();
  const gradient = ctx.createRadialGradient(point.x - radius * 0.3, point.y - radius * 0.35, 0, point.x, point.y, radius * 2.2);
  gradient.addColorStop(0, '#ffffff');
  gradient.addColorStop(0.2, color);
  gradient.addColorStop(1, 'rgba(0,0,0,0)');
  ctx.fillStyle = gradient;
  ctx.shadowColor = color;
  ctx.shadowBlur = radius * 1.4;
  ctx.beginPath();
  ctx.arc(point.x, point.y, radius, 0, Math.PI * 2);
  ctx.fill();
  ctx.restore();
  if (label) {
    ctx.save();
    ctx.font = '800 13px Inter, sans-serif';
    ctx.fillStyle = '#e2e8f0';
    ctx.fillText(label, point.x + radius + 8, point.y - radius * 0.25);
    ctx.restore();
  }
}

function drawCube(ctx: CanvasRenderingContext2D, center: Vec3, width: number, height: number, camera: CameraState, size: number) {
  const s = size / 2;
  const corners = [
    { x: center.x - s, y: center.y - s, z: center.z - s }, { x: center.x + s, y: center.y - s, z: center.z - s },
    { x: center.x + s, y: center.y + s, z: center.z - s }, { x: center.x - s, y: center.y + s, z: center.z - s },
    { x: center.x - s, y: center.y - s, z: center.z + s }, { x: center.x + s, y: center.y - s, z: center.z + s },
    { x: center.x + s, y: center.y + s, z: center.z + s }, { x: center.x - s, y: center.y + s, z: center.z + s },
  ].map((pt) => project(pt, width, height, camera));
  const faces = [[0, 1, 2, 3], [4, 5, 6, 7], [0, 1, 5, 4], [2, 3, 7, 6], [1, 2, 6, 5], [0, 3, 7, 4]];
  ctx.save();
  faces.forEach((face, index) => {
    ctx.beginPath();
    face.forEach((idx, i) => {
      const p = corners[idx];
      if (i === 0) ctx.moveTo(p.x, p.y);
      else ctx.lineTo(p.x, p.y);
    });
    ctx.closePath();
    ctx.fillStyle = index % 2 ? 'rgba(56,189,248,0.34)' : 'rgba(16,185,129,0.32)';
    ctx.strokeStyle = 'rgba(226,232,240,0.58)';
    ctx.lineWidth = 1.3;
    ctx.shadowColor = '#22d3ee';
    ctx.shadowBlur = 12;
    ctx.fill();
    ctx.stroke();
  });
  ctx.restore();
}

function drawHUD(ctx: CanvasRenderingContext2D, width: number, scene: string, props: Record<string, number>, running: boolean, playbackSpeed: number, meta: Record<string, number>, trailCount: number) {
  const kind = sceneKind(scene);
  const linesByKind: Record<string, string[]> = {
    orbit: [
      `r=${numberValue(props, 'orbit_radius', 3.6).toFixed(2)}`,
      `v=${numberValue(props, 'tangential_speed', 2.25).toFixed(2)}`,
      `G*=${numberValue(props, 'gravitational_strength', 10).toFixed(1)}`,
      `轨迹点=${trailCount}`,
    ],
    spring: [
      `k=${numberValue(props, 'k', 24).toFixed(1)}`,
      `m=${numberValue(props, 'm', 1.2).toFixed(1)} kg`,
      `x=${(meta.spring_x ?? numberValue(props, 'x', 1.4)).toFixed(2)} m`,
      `E≈${(meta.energy ?? 0).toFixed(2)}`,
    ],
    collision: [
      `m₁=${numberValue(props, 'm1', 1.5).toFixed(1)} kg`,
      `m₂=${numberValue(props, 'm2', 1).toFixed(1)} kg`,
      `e=${numberValue(props, 'restitution', 0.9).toFixed(2)}`,
      `动能≈${(meta.energy ?? 0).toFixed(1)}`,
    ],
    projectile: [
      `v₀=${numberValue(props, 'v0', 20).toFixed(1)} m/s`,
      `θ=${numberValue(props, 'angle_deg', 45).toFixed(0)}°`,
      `轨迹点=${trailCount}`,
    ],
    force: [
      `m=${numberValue(props, 'm', 2).toFixed(1)} kg`,
      `a=${numberValue(props, 'a', 3).toFixed(1)} m/s²`,
      `F≈${numberValue(props, 'force', numberValue(props, 'm', 2) * numberValue(props, 'a', 3)).toFixed(1)} N`,
    ],
    motion: [
      `v=${numberValue(props, 'v', 5).toFixed(1)} m/s`,
      `a=${numberValue(props, 'a', 0).toFixed(1)} m/s²`,
      `轨迹点=${trailCount}`,
    ],
  };
  const lines = [...(linesByKind[kind] || linesByKind.motion), `速度=${playbackSpeed.toFixed(2)}x`, running ? '仿真运行中' : '已暂停'];
  ctx.save();
  const panelWidth = 230;
  const panelHeight = 34 + lines.length * 23;
  const x = width - panelWidth - 18;
  ctx.fillStyle = 'rgba(15,23,42,0.78)';
  ctx.strokeStyle = 'rgba(125,211,252,0.3)';
  ctx.lineWidth = 1;
  roundRect(ctx, x, 18, panelWidth, panelHeight, 16);
  ctx.fill();
  ctx.stroke();
  ctx.fillStyle = '#e0f2fe';
  ctx.font = '800 14px Inter, sans-serif';
  ctx.fillText(sceneLabel(scene), x + 16, 43);
  ctx.font = '13px Inter, sans-serif';
  lines.forEach((line, index) => {
    ctx.fillStyle = index === lines.length - 1 ? '#86efac' : '#cbd5e1';
    ctx.fillText(line, x + 16, 70 + index * 23);
  });
  ctx.restore();
}

function drawEnergyBars(ctx: CanvasRenderingContext2D, x: number, y: number, kinetic: number, potential: number) {
  const total = Math.max(0.01, kinetic + potential);
  ctx.save();
  ctx.fillStyle = 'rgba(15,23,42,0.72)';
  ctx.strokeStyle = 'rgba(148,163,184,0.24)';
  roundRect(ctx, x, y, 190, 70, 14);
  ctx.fill();
  ctx.stroke();
  ctx.font = '800 12px Inter, sans-serif';
  ctx.fillStyle = '#e2e8f0';
  ctx.fillText('能量分布', x + 14, y + 22);
  [['动能', kinetic, '#facc15'], ['势能', potential, '#38bdf8']].forEach(([label, value, color], i) => {
    const yy = y + 36 + i * 18;
    ctx.fillStyle = '#94a3b8';
    ctx.fillText(label as string, x + 14, yy + 8);
    ctx.fillStyle = 'rgba(51,65,85,0.8)';
    roundRect(ctx, x + 54, yy, 112, 9, 5);
    ctx.fill();
    ctx.fillStyle = color as string;
    roundRect(ctx, x + 54, yy, 112 * (Number(value) / total), 9, 5);
    ctx.fill();
  });
  ctx.restore();
}

function bodyPosition(body: RAPIER.RigidBody, viewDimension: number): Vec3 {
  const p = body.translation();
  return { x: p.x, y: p.y, z: viewDimension >= 3 ? p.z : 0 };
}

function drawProjectile(ctx: CanvasRenderingContext2D, state: SimState, width: number, height: number, camera: CameraState, viewDimension: number, running: boolean, playbackSpeed: number) {
  drawGrid(ctx, width, height, camera);
  const body = state.bodies.main;
  const pos = bodyPosition(body, viewDimension);
  drawTrail(ctx, state.trail, width, height, camera, 'rgba(45,212,191,ALPHA)', viewDimension, 2.8);
  const p2 = project(pos, width, height, camera);
  drawGlowSphere(ctx, p2, 10, '#67e8f9', 'projectile');
  const velocity = body.linvel();
  drawArrow(ctx, p2, project({ x: pos.x + velocity.x * 0.08, y: pos.y + velocity.y * 0.08, z: pos.z + velocity.z * 0.08 }, width, height, camera), '#fbbf24', 'v');
  drawHUD(ctx, width, state.scene, state.props, running, playbackSpeed, state.meta, state.trail.length);
}

function drawForce(ctx: CanvasRenderingContext2D, state: SimState, width: number, height: number, camera: CameraState, viewDimension: number, running: boolean, playbackSpeed: number) {
  drawGrid(ctx, width, height, camera);
  const body = state.bodies.main;
  const pos = bodyPosition(body, viewDimension);
  pos.y = Math.max(0.65, pos.y);
  drawTrail(ctx, state.trail, width, height, camera, 'rgba(56,189,248,ALPHA)', viewDimension, 1.9);
  drawCube(ctx, pos, width, height, camera, 1.25);
  const center = project(pos, width, height, camera);
  const m = numberValue(state.props, 'm', 2);
  const a = numberValue(state.props, 'a', 3);
  const force = numberValue(state.props, 'force', m * a);
  drawArrow(ctx, center, project({ x: pos.x + clamp(force / 8, 1, 5), y: pos.y, z: pos.z }, width, height, camera), '#fb7185', 'F');
  drawArrow(ctx, center, project({ x: pos.x + clamp(a / 8, 0.8, 4), y: pos.y + 0.85, z: pos.z }, width, height, camera), '#a78bfa', 'a');
  const vel = body.linvel();
  drawArrow(ctx, center, project({ x: pos.x + clamp(vel.x / 3, -4, 4), y: pos.y - 0.9, z: pos.z }, width, height, camera), '#facc15', 'v');
  drawHUD(ctx, width, state.scene, state.props, running, playbackSpeed, state.meta, state.trail.length);
}

function drawOrbit(ctx: CanvasRenderingContext2D, state: SimState, width: number, height: number, camera: CameraState, viewDimension: number, running: boolean, playbackSpeed: number) {
  drawGrid(ctx, width, height, camera, { compact: true });
  const center = project({ x: 0, y: 0.25, z: 0 }, width, height, camera);
  const satellite = bodyPosition(state.bodies.satellite, viewDimension);
  const satPoint = project(satellite, width, height, camera);
  const radius = numberValue(state.props, 'orbit_radius', 3.6);

  ctx.save();
  ctx.strokeStyle = 'rgba(96,165,250,0.3)';
  ctx.lineWidth = 1.5;
  ctx.setLineDash([8, 8]);
  ctx.beginPath();
  for (let i = 0; i <= 120; i += 1) {
    const angle = (Math.PI * 2 * i) / 120;
    const p = project({ x: Math.cos(angle) * radius, y: 0.25, z: viewDimension >= 3 ? Math.sin(angle) * radius : 0 }, width, height, camera);
    if (i === 0) ctx.moveTo(p.x, p.y);
    else ctx.lineTo(p.x, p.y);
  }
  ctx.stroke();
  ctx.restore();

  drawTrail(ctx, state.trails.satellite || [], width, height, camera, 'rgba(125,211,252,ALPHA)', viewDimension, 2.4);
  drawGlowSphere(ctx, center, 24, '#f59e0b', '中心天体');
  drawGlowSphere(ctx, satPoint, 11, '#67e8f9', '卫星');
  const velocity = state.bodies.satellite.linvel();
  drawArrow(ctx, satPoint, project({ x: satellite.x + velocity.x * 0.35, y: satellite.y + velocity.y * 0.35, z: satellite.z + velocity.z * 0.35 }, width, height, camera), '#facc15', 'v', 3.2);
  drawArrow(ctx, satPoint, center, '#fb7185', 'F_g', 2.6);
  drawHUD(ctx, width, state.scene, state.props, running, playbackSpeed, state.meta, (state.trails.satellite || []).length);
}

function drawSpring(ctx: CanvasRenderingContext2D, state: SimState, width: number, height: number, camera: CameraState, viewDimension: number, running: boolean, playbackSpeed: number) {
  drawGrid(ctx, width, height, camera, { compact: true });
  const body = state.bodies.mass;
  const pos = bodyPosition(body, viewDimension);
  const anchor: Vec3 = { x: -4.6, y: 1.2, z: 0 };
  const anchorPoint = project(anchor, width, height, camera);
  const massPoint = project(pos, width, height, camera);

  ctx.save();
  ctx.strokeStyle = '#22d3ee';
  ctx.shadowColor = '#22d3ee';
  ctx.shadowBlur = 16;
  ctx.lineWidth = 3;
  ctx.beginPath();
  const coils = 14;
  for (let i = 0; i <= coils; i += 1) {
    const t = i / coils;
    const x = anchor.x + (pos.x - anchor.x) * t;
    const y = anchor.y + (pos.y - anchor.y) * t + Math.sin(t * Math.PI * coils) * 0.22;
    const p = project({ x, y, z: viewDimension >= 3 ? Math.sin(t * Math.PI * 3) * 0.08 : 0 }, width, height, camera);
    if (i === 0) ctx.moveTo(p.x, p.y);
    else ctx.lineTo(p.x, p.y);
  }
  ctx.stroke();
  ctx.restore();

  drawTrail(ctx, state.trails.mass || [], width, height, camera, 'rgba(34,197,94,ALPHA)', viewDimension, 2.0);
  drawGlowSphere(ctx, anchorPoint, 9, '#94a3b8', '固定端');
  drawGlowSphere(ctx, massPoint, 16, '#34d399', '振子');
  const springX = pos.x - anchor.x - 2.2;
  drawArrow(ctx, massPoint, project({ x: pos.x - clamp(springX, -2.5, 2.5), y: pos.y + 0.45, z: pos.z }, width, height, camera), '#fb7185', 'F=-kx', 3.2);
  drawEnergyBars(ctx, 18, height - 94, state.meta.kinetic ?? 0, state.meta.potential ?? 0);
  drawHUD(ctx, width, state.scene, state.props, running, playbackSpeed, state.meta, (state.trails.mass || []).length);
}

function drawCollision(ctx: CanvasRenderingContext2D, state: SimState, width: number, height: number, camera: CameraState, viewDimension: number, running: boolean, playbackSpeed: number, now: number) {
  drawGrid(ctx, width, height, camera, { compact: true });
  const b1 = state.bodies.ball1;
  const b2 = state.bodies.ball2;
  const p1 = bodyPosition(b1, viewDimension);
  const p2 = bodyPosition(b2, viewDimension);
  drawTrail(ctx, state.trails.ball1 || [], width, height, camera, 'rgba(56,189,248,ALPHA)', viewDimension, 2.3);
  drawTrail(ctx, state.trails.ball2 || [], width, height, camera, 'rgba(251,113,133,ALPHA)', viewDimension, 2.3);
  const pt1 = project(p1, width, height, camera);
  const pt2 = project(p2, width, height, camera);
  drawGlowSphere(ctx, pt1, 15, '#38bdf8', 'm₁');
  drawGlowSphere(ctx, pt2, 15, '#fb7185', 'm₂');
  const v1 = b1.linvel();
  const v2 = b2.linvel();
  drawArrow(ctx, pt1, project({ x: p1.x + v1.x * 0.22, y: p1.y + 0.25, z: p1.z + v1.z * 0.22 }, width, height, camera), '#facc15', 'v₁', 3.0);
  drawArrow(ctx, pt2, project({ x: p2.x + v2.x * 0.22, y: p2.y + 0.25, z: p2.z + v2.z * 0.22 }, width, height, camera), '#a78bfa', 'v₂', 3.0);
  if (state.lastImpactAt && now - state.lastImpactAt < 520) {
    const mid = { x: (pt1.x + pt2.x) / 2, y: (pt1.y + pt2.y) / 2 };
    const alpha = 1 - (now - state.lastImpactAt) / 520;
    ctx.save();
    ctx.strokeStyle = `rgba(250,204,21,${alpha})`;
    ctx.shadowColor = '#facc15';
    ctx.shadowBlur = 26;
    ctx.lineWidth = 4;
    ctx.beginPath();
    ctx.arc(mid.x, mid.y, 18 + 42 * (1 - alpha), 0, Math.PI * 2);
    ctx.stroke();
    ctx.restore();
  }
  drawEnergyBars(ctx, 18, height - 94, state.meta.energy ?? 0, Math.max(0, (state.meta.initial_energy ?? 0) - (state.meta.energy ?? 0)));
  drawHUD(ctx, width, state.scene, state.props, running, playbackSpeed, state.meta, (state.trails.ball1 || []).length);
}

function drawMotion(ctx: CanvasRenderingContext2D, state: SimState, width: number, height: number, camera: CameraState, viewDimension: number, running: boolean, playbackSpeed: number) {
  drawGrid(ctx, width, height, camera);
  const body = state.bodies.main;
  const pos = bodyPosition(body, viewDimension);
  drawTrail(ctx, state.trail, width, height, camera, 'rgba(45,212,191,ALPHA)', viewDimension, 2.3);
  drawGlowSphere(ctx, project(pos, width, height, camera), 13, '#22d3ee', 'object');
  const vel = body.linvel();
  drawArrow(ctx, project(pos, width, height, camera), project({ x: pos.x + vel.x * 0.2, y: pos.y + vel.y * 0.2, z: pos.z }, width, height, camera), '#facc15', 'v');
  drawHUD(ctx, width, state.scene, state.props, running, playbackSpeed, state.meta, state.trail.length);
}

async function createSimulation(rapier: RapierModule, scene: string, props: Record<string, number>): Promise<SimState> {
  const kind = sceneKind(scene);
  const gravityY = kind === 'projectile' ? -Math.abs(numberValue(props, 'g', 9.8)) : 0;
  const world = new rapier.World({ x: 0, y: gravityY, z: 0 });
  const colliders: RAPIER.Collider[] = [];
  const bodies: BodyMap = {};
  const trails: TrailMap = {};
  const meta: Record<string, number> = {};

  if (kind !== 'orbit' && kind !== 'spring') {
    colliders.push(world.createCollider(rapier.ColliderDesc.cuboid(20, 0.08, 20).setTranslation(0, -0.08, 0)));
  }

  if (kind === 'projectile') {
    const v0 = numberValue(props, 'v0', 20);
    const angle = numberValue(props, 'angle_deg', 45) * Math.PI / 180;
    const viewDimension = numberValue(props, 'view_dimension', 3);
    bodies.main = world.createRigidBody(rapier.RigidBodyDesc.dynamic().setTranslation(-4.5, 0.55, viewDimension >= 3 ? -2.2 : 0));
    bodies.main.setLinvel({ x: Math.cos(angle) * v0 * 0.18, y: Math.sin(angle) * v0 * 0.18, z: viewDimension >= 3 ? Math.cos(angle) * v0 * 0.055 : 0 }, true);
    colliders.push(world.createCollider(rapier.ColliderDesc.ball(0.16).setRestitution(0.45), bodies.main));
  } else if (kind === 'force') {
    const mass = Math.max(0.1, numberValue(props, 'm', 2));
    const accel = numberValue(props, 'a', 3);
    bodies.main = world.createRigidBody(rapier.RigidBodyDesc.dynamic().setTranslation(-3.4, 0.65, 0).setLinearDamping(0.08));
    colliders.push(world.createCollider(rapier.ColliderDesc.cuboid(0.62, 0.62, 0.62).setDensity(mass / 1.9).setFriction(0.22), bodies.main));
    bodies.main.addForce({ x: mass * accel, y: 0, z: 0 }, true);
  } else if (kind === 'orbit') {
    const radius = numberValue(props, 'orbit_radius', 3.6);
    const speed = numberValue(props, 'tangential_speed', 2.25);
    const eccentricity = numberValue(props, 'eccentricity', 0.18);
    bodies.satellite = world.createRigidBody(rapier.RigidBodyDesc.dynamic().setTranslation(radius * (1 + eccentricity), 0.25, 0).setLinearDamping(0.002));
    bodies.satellite.setLinvel({ x: 0, y: 0, z: speed }, true);
    colliders.push(world.createCollider(rapier.ColliderDesc.ball(0.16), bodies.satellite));
    trails.satellite = [];
  } else if (kind === 'spring') {
    const x = numberValue(props, 'x', 1.4);
    const damping = numberValue(props, 'damping', 0.18);
    bodies.mass = world.createRigidBody(rapier.RigidBodyDesc.dynamic().setTranslation(-2.4 + x, 1.2, 0).setLinearDamping(damping));
    colliders.push(world.createCollider(rapier.ColliderDesc.ball(0.22).setDensity(Math.max(0.2, numberValue(props, 'm', 1.2))), bodies.mass));
    trails.mass = [];
  } else if (kind === 'collision') {
    const m1 = Math.max(0.2, numberValue(props, 'm1', 1.5));
    const m2 = Math.max(0.2, numberValue(props, 'm2', 1));
    const e = clamp(numberValue(props, 'restitution', 0.9), 0, 1);
    bodies.ball1 = world.createRigidBody(rapier.RigidBodyDesc.dynamic().setTranslation(-3.2, 0.45, -0.6).setLinearDamping(0.01));
    bodies.ball2 = world.createRigidBody(rapier.RigidBodyDesc.dynamic().setTranslation(3.2, 0.45, 0.6).setLinearDamping(0.01));
    bodies.ball1.setLinvel({ x: numberValue(props, 'v1', 4.5), y: 0, z: 0.3 }, true);
    bodies.ball2.setLinvel({ x: numberValue(props, 'v2', -2.5), y: 0, z: -0.3 }, true);
    colliders.push(world.createCollider(rapier.ColliderDesc.ball(0.42).setDensity(m1).setRestitution(e), bodies.ball1));
    colliders.push(world.createCollider(rapier.ColliderDesc.ball(0.42).setDensity(m2).setRestitution(e), bodies.ball2));
    trails.ball1 = [];
    trails.ball2 = [];
    meta.initial_energy = 0.5 * m1 * numberValue(props, 'v1', 4.5) ** 2 + 0.5 * m2 * numberValue(props, 'v2', -2.5) ** 2;
  } else {
    bodies.main = world.createRigidBody(rapier.RigidBodyDesc.dynamic().setTranslation(-4, 0.55, 0).setLinearDamping(0.04));
    bodies.main.setLinvel({ x: numberValue(props, 'v', numberValue(props, 'v0', 5)) * 0.25, y: 0, z: 0 }, true);
    colliders.push(world.createCollider(rapier.ColliderDesc.ball(0.24).setRestitution(0.35), bodies.main));
  }

  return { world, bodies, colliders, startedAt: performance.now(), trail: [], trails, scene, props: { ...props }, meta };
}

function updateSimulation(state: SimState, effectiveDelta: number) {
  const kind = sceneKind(state.scene);
  if (kind === 'force') {
    const mass = Math.max(0.1, numberValue(state.props, 'm', 2));
    const accel = numberValue(state.props, 'a', 3);
    state.bodies.main.addForce({ x: mass * accel, y: 0, z: 0 }, true);
  } else if (kind === 'orbit') {
    const body = state.bodies.satellite;
    const pos = body.translation();
    const dx = -pos.x;
    const dz = -pos.z;
    const distSq = Math.max(0.55, dx * dx + dz * dz);
    const dist = Math.sqrt(distSq);
    const strength = numberValue(state.props, 'gravitational_strength', 10) * numberValue(state.props, 'central_mass', 8) * 0.045;
    body.addForce({ x: (dx / dist) * strength / distSq, y: 0, z: (dz / dist) * strength / distSq }, true);
  } else if (kind === 'spring') {
    const body = state.bodies.mass;
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
  } else if (kind === 'collision') {
    const b1 = state.bodies.ball1;
    const b2 = state.bodies.ball2;
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
  } else if (kind === 'motion') {
    const a = numberValue(state.props, 'a', 0);
    state.bodies.main.addForce({ x: a, y: 0, z: 0 }, true);
  }

  state.world.timestep = effectiveDelta;
  state.world.step();
}

function appendTrails(state: SimState, viewDimension: number) {
  const maxTrail = Math.floor(clamp(numberValue(state.props, 'trail_length', 180), 40, 360));
  const kind = sceneKind(state.scene);
  const add = (name: string, body: RAPIER.RigidBody) => {
    const list = state.trails[name] || [];
    const pos = bodyPosition(body, viewDimension);
    list.push(pos);
    if (list.length > maxTrail) list.shift();
    state.trails[name] = list;
  };

  if (kind === 'orbit') add('satellite', state.bodies.satellite);
  else if (kind === 'spring') add('mass', state.bodies.mass);
  else if (kind === 'collision') {
    add('ball1', state.bodies.ball1);
    add('ball2', state.bodies.ball2);
  } else if (state.bodies.main) {
    const pos = bodyPosition(state.bodies.main, viewDimension);
    state.trail.push(pos);
    if (state.trail.length > maxTrail) state.trail.shift();
  }
}

export function isNativePhysicsPreviewArtifact(artifact: RenderArtifact): boolean {
  return isNativePhysicsArtifact(artifact);
}

export default function NativePhysicsPreview({ artifact, propsData, onStatusChange }: NativePhysicsPreviewProps) {
  const canvasRef = useRef<HTMLCanvasElement | null>(null);
  const simRef = useRef<SimState | null>(null);
  const rapierRef = useRef<RapierModule | null>(null);
  const frameRef = useRef<number | null>(null);
  const draggingRef = useRef<{ x: number; y: number } | null>(null);
  const [camera, setCamera] = useState<CameraState>({
    yaw: numberValue(propsData, 'camera_yaw', 0.65),
    pitch: numberValue(propsData, 'camera_pitch', 0.46),
    distance: sceneKind(artifact.scene_type) === 'orbit' ? 13 : 11,
  });
  const [running, setRunning] = useState(true);
  const [resetSeq, setResetSeq] = useState(0);
  const [status, setStatus] = useState<PreviewStatus>('loading');
  const [error, setError] = useState<string | null>(null);
  const [lastSyncText, setLastSyncText] = useState('等待物理引擎同步');

  const mergedProps = useMemo(() => ({ ...(artifact.render_manifest.initial_props || {}), ...propsData }), [artifact, propsData]);
  const viewDimension = numberValue(mergedProps, 'view_dimension', artifact.scene_type.includes('2d') ? 2 : 3);
  const playbackSpeed = clamp(numberValue(mergedProps, 'animation_speed', 1), 0.1, 4);

  const notifyStatus = useCallback((next: PreviewStatus, detail?: string) => {
    setStatus(next);
    onStatusChange?.(next, detail);
  }, [onStatusChange]);

  useEffect(() => {
    let disposed = false;
    notifyStatus('loading');
    setError(null);
    import('@dimforge/rapier3d-compat')
      .then(async (mod) => {
        await mod.init();
        if (disposed) return;
        rapierRef.current = mod;
        simRef.current = await createSimulation(mod, artifact.scene_type, mergedProps);
        notifyStatus('ready');
        setLastSyncText('Rapier 3D 已初始化');
      })
      .catch((err: unknown) => {
        const message = err instanceof Error ? err.message : 'Rapier 3D 初始化失败';
        setError(message);
        notifyStatus('error', message);
      });
    return () => {
      disposed = true;
      simRef.current?.world.free();
      simRef.current = null;
    };
  }, [artifact.scene_type, resetSeq]); // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    const mod = rapierRef.current;
    if (!mod) return;
    simRef.current?.world.free();
    void createSimulation(mod, artifact.scene_type, mergedProps).then((sim) => {
      simRef.current = sim;
      notifyStatus('updated');
      setLastSyncText(`已同步到物理引擎 ${new Date().toLocaleTimeString('zh-CN', { hour12: false })}`);
    }).catch((err: unknown) => {
      const message = err instanceof Error ? err.message : '参数同步失败';
      setError(message);
      notifyStatus('error', message);
    });
  }, [mergedProps, artifact.scene_type, notifyStatus]);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return undefined;
    const ctx = canvas.getContext('2d');
    if (!ctx) return undefined;
    let last = performance.now();
    const render = (now: number) => {
      const rect = canvas.getBoundingClientRect();
      const dpr = window.devicePixelRatio || 1;
      const width = Math.max(320, Math.floor(rect.width * dpr));
      const height = Math.max(380, Math.floor(rect.height * dpr));
      if (canvas.width !== width || canvas.height !== height) {
        canvas.width = width;
        canvas.height = height;
      }
      ctx.setTransform(1, 0, 0, 1, 0, 0);
      drawBackground(ctx, width, height, now, sceneKind(artifact.scene_type));
      ctx.save();
      ctx.scale(dpr, dpr);
      const cssWidth = width / dpr;
      const cssHeight = height / dpr;
      const sim = simRef.current;
      if (sim) {
        if (running) {
          const realDelta = Math.min(0.05, (now - last) / 1000);
          const effectiveDelta = Math.min(0.2, realDelta * playbackSpeed);
          const fixedStep = 1 / 60;
          const steps = Math.max(1, Math.min(18, Math.ceil(effectiveDelta / fixedStep)));
          const stepSize = effectiveDelta / steps;
          for (let i = 0; i < steps; i += 1) updateSimulation(sim, stepSize);
          appendTrails(sim, viewDimension);
          const kind = sceneKind(sim.scene);
          const mainBody = sim.bodies.main || sim.bodies.satellite || sim.bodies.mass || sim.bodies.ball1;
          const pos = mainBody.translation();
          if ((kind === 'projectile' && pos.y < -1) || Math.abs(pos.x) > 10 || Math.abs(pos.z) > 10 || now - sim.startedAt > 26000) {
            setResetSeq((value) => value + 1);
          }
        }
        const kind = sceneKind(sim.scene);
        if (kind === 'projectile') drawProjectile(ctx, sim, cssWidth, cssHeight, camera, viewDimension, running, playbackSpeed);
        else if (kind === 'force') drawForce(ctx, sim, cssWidth, cssHeight, camera, viewDimension, running, playbackSpeed);
        else if (kind === 'orbit') drawOrbit(ctx, sim, cssWidth, cssHeight, camera, viewDimension, running, playbackSpeed);
        else if (kind === 'spring') drawSpring(ctx, sim, cssWidth, cssHeight, camera, viewDimension, running, playbackSpeed);
        else if (kind === 'collision') drawCollision(ctx, sim, cssWidth, cssHeight, camera, viewDimension, running, playbackSpeed, now);
        else drawMotion(ctx, sim, cssWidth, cssHeight, camera, viewDimension, running, playbackSpeed);
      } else {
        ctx.fillStyle = '#cbd5e1';
        ctx.font = '16px Inter, sans-serif';
        ctx.fillText('正在初始化 Rapier 3D 物理引擎…', 24, 44);
      }
      ctx.restore();
      last = now;
      frameRef.current = window.requestAnimationFrame(render);
    };
    frameRef.current = window.requestAnimationFrame(render);
    return () => {
      if (frameRef.current !== null) window.cancelAnimationFrame(frameRef.current);
    };
  }, [artifact.scene_type, camera, playbackSpeed, running, viewDimension]);

  const handlePointerDown = (event: React.PointerEvent<HTMLCanvasElement>) => {
    draggingRef.current = { x: event.clientX, y: event.clientY };
    event.currentTarget.setPointerCapture(event.pointerId);
  };

  const handlePointerMove = (event: React.PointerEvent<HTMLCanvasElement>) => {
    const last = draggingRef.current;
    if (!last) return;
    const dx = event.clientX - last.x;
    const dy = event.clientY - last.y;
    draggingRef.current = { x: event.clientX, y: event.clientY };
    setCamera((prev) => ({
      ...prev,
      yaw: prev.yaw + dx * 0.008,
      pitch: clamp(prev.pitch + dy * 0.006, -0.75, 0.95),
    }));
  };

  const handlePointerUp = () => {
    draggingRef.current = null;
  };

  return (
    <Space direction="vertical" style={{ width: '100%' }} size="middle">
      <Space wrap>
        <Tag color={status === 'error' ? 'red' : status === 'ready' || status === 'updated' ? 'green' : 'blue'}>{status}</Tag>
        <Tag color="geekblue">Rapier 3D</Tag>
        <Tag color="purple">Canvas 教学演示</Tag>
        <Tag color="cyan">{artifact.scene_type}</Tag>
        <Tag color="blue">{viewDimension >= 3 ? '3D 视角' : '2D 投影'}</Tag>
        <Text type="secondary">{lastSyncText}</Text>
      </Space>
      {error && <Alert type="error" showIcon message="Rapier 3D 预览失败" description={error} />}
      <Card
        styles={{ body: { padding: 0, overflow: 'hidden', background: '#020617' } }}
        style={{ borderColor: 'rgba(56,189,248,0.32)', boxShadow: '0 24px 80px rgba(15, 23, 42, 0.18)' }}
      >
        <canvas
          ref={canvasRef}
          style={{ width: '100%', height: 620, display: 'block', cursor: 'grab', touchAction: 'none' }}
          onPointerDown={handlePointerDown}
          onPointerMove={handlePointerMove}
          onPointerUp={handlePointerUp}
          onPointerCancel={handlePointerUp}
        />
      </Card>
      <Space wrap style={{ justifyContent: 'space-between', width: '100%' }}>
        <Space wrap>
          <Button icon={running ? <PauseCircleOutlined /> : <PlayCircleOutlined />} onClick={() => setRunning((value) => !value)}>
            {running ? '暂停仿真' : '继续仿真'}
          </Button>
          <Button icon={<ReloadOutlined />} onClick={() => setResetSeq((value) => value + 1)}>重置仿真</Button>
        </Space>
        <Text type="secondary">拖拽画布旋转视角；参数滑块会重置并同步到 Rapier world。</Text>
      </Space>
    </Space>
  );
}
