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
import { Canvas, useFrame } from '@react-three/fiber';
import { OrbitControls, AdaptiveDpr, AdaptiveEvents } from '@react-three/drei';
import { Button, Slider, Space, Tag, Tooltip } from 'antd';
import { PauseCircleOutlined, PlayCircleOutlined, ReloadOutlined } from '@ant-design/icons';

import type { RenderArtifact } from '@/lib/api';
import type { PreviewStatus } from '@/components/common/RenderPreviewSandbox';
import SceneByKind from './lib/sceneRegistry';
import PostFX, { type QualityLevel } from './lib/postFX';
import R3FHud from './lib/R3FHud';
import { useRapierSimulation } from './hooks/useRapierSimulation';
import { detectWebGL, prefersReducedMotion } from './lib/capability';
import { clamp, numberValue, sceneKindOf } from './lib/types';

import './r3f.css';

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
  const [quality, setQuality] = useState<QualityLevel>('standard');

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

  const { ref: simRef, status, error, advance, reset } = useRapierSimulation(
    artifact.scene_type,
    mergedProps,
    { running },
  );

  useEffect(() => {
    if (status === 'loading') onStatusChange?.('loading');
    else if (status === 'ready') onStatusChange?.('ready');
    else if (status === 'error') onStatusChange?.('error', error || undefined);
  }, [status, error, onStatusChange]);

  const handleReset = useCallback(() => reset(), [reset]);

  if (webglAvailable === null) {
    return (
      <div className="snowy-r3f-shell snowy-r3f-shell--loading" role="status" aria-live="polite">
        <span>检测 WebGL…</span>
      </div>
    );
  }

  if (webglAvailable === false || status === 'error') {
    return <NativePhysicsPreview artifact={artifact} propsData={propsData} onStatusChange={onStatusChange} />;
  }

  const effectiveQuality: QualityLevel = reducedMotion ? 'eco' : quality;

  return (
    <div className="snowy-r3f-shell" data-scene={sceneKind}>
      <Canvas
        className="snowy-r3f-canvas"
        camera={{ position: sceneKind === 'orbit' ? [0, 6, 10] : [4, 5, 8], fov: 45, near: 0.1, far: 200 }}
        shadows
        dpr={[1, quality === 'high' ? 2 : 1.5]}
        gl={{ antialias: true, alpha: false, powerPreference: 'high-performance' }}
      >
        <AdaptiveDpr pixelated />
        <AdaptiveEvents />
        <FrameAdvance advance={advance} running={running} speed={speed} viewDimension={viewDimension} />
        <SceneByKind kind={sceneKind} simRef={simRef} />
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
      <div className="snowy-r3f-controls">
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
          <span className="snowy-r3f-controls__label">速度</span>
          <Slider
            min={0.2}
            max={3}
            step={0.1}
            value={speed}
            onChange={(value) => setSpeed(Array.isArray(value) ? value[0] : value)}
            style={{ width: 120 }}
          />
          <span className="snowy-r3f-controls__value">×{speed.toFixed(2)}</span>
          {!reducedMotion && (
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
    </div>
  );
}

export function isR3FPhysicsArtifact(artifact: RenderArtifact): boolean {
  return /3d|physics|orbit|spring|collision|projectile|force|motion/.test(artifact.scene_type);
}
