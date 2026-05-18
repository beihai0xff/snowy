/**
 * Snowy v6 · R3F · 物理预览主入口
 *
 * - 与 NativePhysicsPreview 同 prop 签名，可在调用方平滑替换；
 * - 浏览器 WebGL 不可用或 R3F 加载失败时，动态导入 NativePhysicsPreview 作为回退；
 * - 集成 Canvas + OrbitControls + 后处理 + 场景注册表 + HUD。
 */

'use client';

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import dynamic from 'next/dynamic';
import * as THREE from 'three';
import { Canvas, useFrame } from '@react-three/fiber';
import { OrbitControls, AdaptiveDpr, AdaptiveEvents } from '@react-three/drei';
import { Button, Slider, Space, Tag, Tooltip } from 'antd';
import { useDeviceTier } from '@/lib/useDeviceTier';
import { PauseCircleOutlined, PlayCircleOutlined, ReloadOutlined } from '@ant-design/icons';

import type { RenderArtifact } from '@/lib/api';
import type { PreviewStatus } from '@/components/common/RenderPreviewSandbox';
import SceneByKind from './lib/sceneRegistry';
import PostFX, { type QualityLevel } from './lib/postFX';
import R3FHud from './lib/R3FHud';
import { useRapierSimulation } from './hooks/useRapierSimulation';
import { detectWebGL, prefersReducedMotion } from './lib/capability';
import { clamp, numberValue, sceneKindOf, type SceneKind } from './lib/types';
import Physics2DView, { sampleAnalyticalTrail } from '../2d/Physics2DView';
import KinematicsChartView, { sampleKinematics, type OverlaySeries } from '../2d/KinematicsChartView';
import { StepBackwardOutlined, StepForwardOutlined, SaveOutlined, ClearOutlined } from '@ant-design/icons';

import './r3f.css';
import '../2d/physics2d.css';

type PhysicsViewMode = '2d' | '3d' | 'data';

/** 教学预设：每个场景几个典型参数组合，一键应用 */
const PRESETS: Record<SceneKind, Array<{ id: string; label: string; props: Record<string, number> }>> = {
  projectile: [
    { id: 'p30', label: '30°', props: { angle_deg: 30, v0: 20, g: 9.8 } },
    { id: 'p45', label: '45°（最远）', props: { angle_deg: 45, v0: 20, g: 9.8 } },
    { id: 'p60', label: '60°', props: { angle_deg: 60, v0: 20, g: 9.8 } },
    { id: 'p_fast', label: 'v₀=30', props: { angle_deg: 45, v0: 30, g: 9.8 } },
    { id: 'p_moon', label: '月球 g=1.6', props: { angle_deg: 45, v0: 20, g: 1.6 } },
  ],
  motion: [
    { id: 'm_const', label: '匀速', props: { v0: 6, a: 0 } },
    { id: 'm_acc', label: '匀加速', props: { v0: 0, a: 2 } },
    { id: 'm_dec', label: '减速', props: { v0: 8, a: -1.5 } },
  ],
  force: [
    { id: 'f_light', label: 'm=1 a=2', props: { m: 1, a: 2 } },
    { id: 'f_heavy', label: 'm=4 a=2', props: { m: 4, a: 2 } },
    { id: 'f_strong', label: 'F 大', props: { m: 2, a: 5 } },
  ],
  spring: [
    { id: 's_undamped', label: '无阻尼', props: { k: 24, m: 1.2, x: 1.4, damping: 0 } },
    { id: 's_under', label: '欠阻尼', props: { k: 24, m: 1.2, x: 1.4, damping: 0.6 } },
    { id: 's_critical', label: '近临界', props: { k: 24, m: 1.2, x: 1.4, damping: 2 * Math.sqrt(24 * 1.2) } },
    { id: 's_stiff', label: '硬弹簧', props: { k: 60, m: 1.2, x: 1.4, damping: 0.4 } },
  ],
  collision: [
    { id: 'c_elastic', label: '完全弹性 e=1', props: { m1: 1, m2: 1, v1: 4, v2: -2, restitution: 1 } },
    { id: 'c_inelastic', label: '完全非弹性 e=0', props: { m1: 1, m2: 1, v1: 4, v2: -2, restitution: 0 } },
    { id: 'c_heavyhit', label: '重撞轻', props: { m1: 4, m2: 1, v1: 3, v2: 0, restitution: 0.9 } },
  ],
  orbit: [
    { id: 'o_circle', label: '圆轨道', props: { orbit_radius: 3.6, tangential_speed: 2.25, eccentricity: 0 } },
    { id: 'o_ellipse', label: '椭圆 e=0.3', props: { orbit_radius: 3.6, tangential_speed: 2.25, eccentricity: 0.3 } },
    { id: 'o_high', label: '椭圆 e=0.5', props: { orbit_radius: 3.6, tangential_speed: 2.25, eccentricity: 0.5 } },
  ],
};

const NativePhysicsPreview = dynamic(() => import('../NativePhysicsPreview'), { ssr: false });

type Props = {
  artifact: RenderArtifact;
  propsData: Record<string, number>;
  onStatusChange?: (status: PreviewStatus, detail?: string) => void;
};

interface FrameAdvanceProps {
  advance: (delta: number, speed: number, dim: number) => void;
  running: boolean;
  speed: number;
  viewDimension: number;
}

function FrameAdvance({ advance, running, speed, viewDimension }: FrameAdvanceProps) {
  useFrame((_, delta) => {
    if (!running) return;
    advance(delta, speed, viewDimension);
  });
  return null;
}

export default function R3FPhysicsPreview({ artifact, propsData, onStatusChange }: Props) {
  const [webglAvailable, setWebglAvailable] = useState<boolean | null>(null);
  const [reducedMotion, setReducedMotion] = useState(false);
  const [running, setRunning] = useState(true);
  const tier = useDeviceTier();
  const [quality, setQuality] = useState<QualityLevel>(tier === 'low' ? 'eco' : tier === 'high' ? 'high' : 'standard');
  const [viewMode, setViewMode] = useState<PhysicsViewMode>('2d');

  useEffect(() => {
    setWebglAvailable(detectWebGL());
    setReducedMotion(prefersReducedMotion());
  }, []);

  const mergedProps = useMemo(
    () => ({ ...(artifact.render_manifest.initial_props || {}), ...propsData }),
    [artifact, propsData],
  );

  const sceneKind = sceneKindOf(artifact.scene_type);
  const viewDimension = numberValue(mergedProps, 'view_dimension', 3);
  const playbackSpeed = clamp(numberValue(mergedProps, 'animation_speed', 1), 0.1, 4);
  const [speed, setSpeed] = useState(playbackSpeed);
  useEffect(() => { setSpeed(playbackSpeed); }, [playbackSpeed]);

  // ───── P1：教学预设 / 历史轨迹 / 步进 ─────
  const [activePresetId, setActivePresetId] = useState<string | null>(null);
  const [presetOverride, setPresetOverride] = useState<Record<string, number> | null>(null);
  const effectiveProps = useMemo(
    () => (presetOverride ? { ...mergedProps, ...presetOverride } : mergedProps),
    [mergedProps, presetOverride],
  );
  // 切换场景类型时清空 preset（避免跨场景串数据）
  useEffect(() => {
    setPresetOverride(null);
    setActivePresetId(null);
    setGhostTrails([]);
    setKinematicsOverlays([]);
  }, [sceneKind]);

  const [ghostTrails, setGhostTrails] = useState<Array<{ id: string; points: { x: number; y: number }[]; label?: string }>>([]);
  const [kinematicsOverlays, setKinematicsOverlays] = useState<OverlaySeries[]>([]);
  const [stepCommand, setStepCommand] = useState<{ id: number; deltaSeconds: number }>({ id: 0, deltaSeconds: 0 });

  const handleApplyPreset = useCallback((preset: { id: string; props: Record<string, number> }) => {
    setActivePresetId(preset.id);
    setPresetOverride(preset.props);
  }, []);
  const handleClearPreset = useCallback(() => {
    setActivePresetId(null);
    setPresetOverride(null);
  }, []);
  const handleSaveTrail = useCallback(() => {
    const points = sampleAnalyticalTrail(artifact.scene_type, effectiveProps);
    const kin = sampleKinematics(artifact.scene_type, effectiveProps);
    const id = `${Date.now()}`;
    const label = activePresetId
      ? PRESETS[sceneKind]?.find((p) => p.id === activePresetId)?.label
      : undefined;
    if (points.length > 0) {
      setGhostTrails((arr) => [...arr.slice(-4), { id, points, label }]);
    }
    if (kin.samples.length > 0) {
      setKinematicsOverlays((arr) => [...arr.slice(-4), { id, label, samples: kin.samples }]);
    }
  }, [artifact.scene_type, effectiveProps, activePresetId, sceneKind]);
  const handleClearTrails = useCallback(() => {
    setGhostTrails([]);
    setKinematicsOverlays([]);
  }, []);
  const handleStep = useCallback((deltaSeconds: number) => {
    setRunning(false);
    setStepCommand((cmd) => ({ id: cmd.id + 1, deltaSeconds }));
  }, []);

  const { ref: simRef, status, error, advance, reset } = useRapierSimulation(
    artifact.scene_type,
    effectiveProps,
    { running: running && viewMode === '3d' },
  );

  const [resetSignal, setResetSignal] = useState(0);

  useEffect(() => {
    if (status === 'loading') onStatusChange?.('loading');
    else if (status === 'ready') onStatusChange?.('ready');
    else if (status === 'error') onStatusChange?.('error', error || undefined);
  }, [status, error, onStatusChange]);

  const handleReset = useCallback(() => {
    reset();
    setResetSignal((v) => v + 1);
  }, [reset]);

  if (webglAvailable === null) {
    return (
      <div className="snowy-r3f-shell snowy-r3f-shell--loading" role="status" aria-live="polite">
        <span>检测 WebGL…</span>
      </div>
    );
  }

  // 3D 视图遇到 WebGL 不可用或 Rapier 失败时，回退到 NativePhysicsPreview
  if (viewMode === '3d' && (webglAvailable === false || status === 'error')) {
    return <NativePhysicsPreview artifact={artifact} propsData={propsData} onStatusChange={onStatusChange} />;
  }

  const effectiveQuality: QualityLevel = reducedMotion ? 'eco' : quality;
  const isDark = viewMode === '3d';
  const shellHeight: React.CSSProperties = { height: 'clamp(420px, 60vh, 640px)', position: 'relative' };

  /** 视图切换栏 */
  const viewBar = (
    <div className={`snowy-physics-viewbar${isDark ? ' snowy-physics-viewbar--dark' : ''}`}>
      <button
        type="button"
        className={viewMode === '2d' ? 'is-active' : ''}
        onClick={() => setViewMode('2d')}
        aria-pressed={viewMode === '2d'}
      >📐 教学视图</button>
      <button
        type="button"
        className={viewMode === '3d' ? 'is-active' : ''}
        onClick={() => setViewMode('3d')}
        aria-pressed={viewMode === '3d'}
        disabled={webglAvailable === false}
      >🎬 3D 沉浸</button>
      <button
        type="button"
        className={viewMode === 'data' ? 'is-active' : ''}
        onClick={() => setViewMode('data')}
        aria-pressed={viewMode === 'data'}
      >📊 数据曲线</button>
    </div>
  );

  /** 教学预设 + 历史轨迹工具栏（2D 与数据视图） */
  const presets = PRESETS[sceneKind] ?? [];
  const supportsCompare = viewMode === '2d'
    ? (sceneKind === 'projectile' || sceneKind === 'orbit')
    : true; // 数据视图对所有场景都支持叠加曲线
  const presetToolbar = (viewMode === '2d' || viewMode === 'data') && (presets.length > 0 || supportsCompare) && (
    <div className="snowy-physics-toolbar" role="toolbar" aria-label="教学预设与对比">
      {presets.length > 0 && (
        <>
          <span className="snowy-physics-toolbar__label">预设：</span>
          {presets.map((p) => (
            <button
              key={p.id}
              type="button"
              className={`snowy-physics-toolbar__chip${activePresetId === p.id ? ' is-active' : ''}`}
              onClick={() => handleApplyPreset(p)}
            >{p.label}</button>
          ))}
          {activePresetId && (
            <button
              type="button"
              className="snowy-physics-toolbar__chip"
              onClick={handleClearPreset}
              title="恢复滑块原值"
            >× 取消预设</button>
          )}
        </>
      )}
      {supportsCompare && (
        <>
          <span className="snowy-physics-toolbar__divider" />
          <Tooltip title="保存当前轨迹/曲线用于对比">
            <button type="button" className="snowy-physics-toolbar__chip" onClick={handleSaveTrail}>
              <SaveOutlined /> 保存对比
            </button>
          </Tooltip>
          {(ghostTrails.length > 0 || kinematicsOverlays.length > 0) && (
            <Tooltip title={`已保存 ${Math.max(ghostTrails.length, kinematicsOverlays.length)} 条`}>
              <button type="button" className="snowy-physics-toolbar__chip" onClick={handleClearTrails}>
                <ClearOutlined /> 清除
              </button>
            </Tooltip>
          )}
        </>
      )}
    </div>
  );
  const controlsBar = (
    <div className={viewMode === '2d' ? 'snowy-p2d-controls' : 'snowy-r3f-controls'}>
      <Space size={8} wrap>
        <Tooltip title={running ? '暂停' : '播放'}>
          <Button
            size="small"
            type="primary"
            icon={running ? <PauseCircleOutlined /> : <PlayCircleOutlined />}
            onClick={() => setRunning((v) => !v)}
          >
            {running ? '暂停' : '播放'}
          </Button>
        </Tooltip>
        <Tooltip title="重置仿真">
          <Button size="small" icon={<ReloadOutlined />} onClick={handleReset}>
            重置
          </Button>
        </Tooltip>
        {viewMode === '2d' && (
          <Space.Compact size="small">
            <Tooltip title="后退 0.1s">
              <Button size="small" icon={<StepBackwardOutlined />} onClick={() => handleStep(-0.1)} />
            </Tooltip>
            <Tooltip title="前进 0.1s">
              <Button size="small" icon={<StepForwardOutlined />} onClick={() => handleStep(0.1)} />
            </Tooltip>
          </Space.Compact>
        )}
        <span className={viewMode === '2d' ? '' : 'snowy-r3f-controls__label'}>速度</span>
        <Slider
          min={0.2}
          max={3}
          step={0.1}
          value={speed}
          onChange={(value) => setSpeed(Array.isArray(value) ? value[0] : value)}
          style={{ width: 120 }}
        />
        <span className={viewMode === '2d' ? '' : 'snowy-r3f-controls__value'} style={{ minWidth: 44, fontVariantNumeric: 'tabular-nums' }}>×{speed.toFixed(2)}</span>
        {viewMode === '3d' && !reducedMotion && (
          <Space size={4}>
            <span className="snowy-r3f-controls__label">画质</span>
            {(['eco', 'standard', 'high'] as QualityLevel[]).map((q) => (
              <Tag.CheckableTag key={q} checked={quality === q} onChange={() => setQuality(q)}>
                {q === 'eco' ? '省电' : q === 'standard' ? '标准' : '高画质'}
              </Tag.CheckableTag>
            ))}
          </Space>
        )}
      </Space>
    </div>
  );

  if (viewMode === '2d') {
    return (
      <div style={shellHeight}>
        {viewBar}
        {presetToolbar}
        <Physics2DView
          sceneType={artifact.scene_type}
          props={effectiveProps}
          running={running}
          playbackSpeed={speed}
          resetSignal={resetSignal}
          stepCommand={stepCommand}
          ghostTrails={ghostTrails}
        />
        {controlsBar}
      </div>
    );
  }

  if (viewMode === 'data') {
    return (
      <div style={shellHeight}>
        {viewBar}
        {presetToolbar}
        <KinematicsChartView
          sceneType={artifact.scene_type}
          props={effectiveProps}
          running={running}
          playbackSpeed={speed}
          resetSignal={resetSignal}
          overlays={kinematicsOverlays}
        />
        {controlsBar}
      </div>
    );
  }

  return (
    <div
      className="snowy-r3f-shell"
      data-scene={sceneKind}
      data-quality={effectiveQuality}
      style={shellHeight}
    >
      {viewBar}
      <Canvas
        className="snowy-r3f-canvas"
        style={{ width: '100%', height: '100%' }}
        camera={{ position: sceneKind === 'orbit' ? [0, 6, 10] : [4, 5, 8], fov: 45, near: 0.1, far: 200 }}
        shadows
        dpr={[1, effectiveQuality === 'high' ? 2 : effectiveQuality === 'eco' ? 1 : 1.5]}
        gl={{ antialias: true, alpha: false, powerPreference: 'high-performance' }}
        onCreated={({ gl }) => {
          gl.toneMapping = THREE.ACESFilmicToneMapping;
          gl.toneMappingExposure = 1.15;
        }}
      >
        <AdaptiveDpr pixelated />
        <AdaptiveEvents />
        <FrameAdvance advance={advance} running={running} speed={speed} viewDimension={viewDimension} />
        <SceneByKind kind={sceneKind} simRef={simRef} quality={effectiveQuality} />
        <OrbitControls
          enablePan={false}
          enableDamping
          dampingFactor={0.08}
          minDistance={3}
          maxDistance={28}
          maxPolarAngle={Math.PI * 0.49}
        />
        <PostFX quality={effectiveQuality} />
        <R3FHud simRef={simRef} kind={sceneKind} running={running} playbackSpeed={speed} />
      </Canvas>
      {controlsBar}
    </div>
  );
}

export function isR3FPhysicsArtifact(artifact: RenderArtifact): boolean {
  return /3d|physics|orbit|spring|collision|projectile|force|motion/.test(artifact.scene_type);
}
