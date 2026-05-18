/**
 * Snowy v9 · 物理 2D/3D 视图开发演示页（无后端依赖）
 *
 * 用一组 mock artifact 直接挂载 R3FPhysicsPreview，方便：
 *  - 手工验证教学视图、3D 沉浸视图、数据曲线视图
 *  - 切换场景类型（projectile / spring / collision / motion / force / orbit）
 *  - 调整参数 / 应用预设 / 保存对比
 *
 * 路径：/physics-demo
 */

'use client';

import React, { useMemo, useState } from 'react';
import dynamic from 'next/dynamic';
import { Card, Radio, Slider, Space, Typography } from 'antd';
import type { RenderArtifact } from '@/lib/api';

const R3FPhysicsPreview = dynamic(() => import('@/components/physics/r3f/R3FPhysicsPreview'), { ssr: false });

type SceneOpt = {
  key: string;
  label: string;
  scene_type: string;
  defaults: Record<string, number>;
  sliders: Array<{ key: string; label: string; min: number; max: number; step: number }>;
};

const SCENES: SceneOpt[] = [
  {
    key: 'projectile',
    label: '平抛运动 / 斜抛',
    scene_type: 'physics_projectile_3d',
    defaults: { v0: 20, angle_deg: 45, g: 9.8 },
    sliders: [
      { key: 'v0', label: '初速度 v₀ (m/s)', min: 5, max: 40, step: 1 },
      { key: 'angle_deg', label: '抛射角 θ (°)', min: 0, max: 90, step: 1 },
      { key: 'g', label: '重力 g (m/s²)', min: 1.6, max: 24, step: 0.1 },
    ],
  },
  {
    key: 'motion',
    label: '直线运动（匀加速）',
    scene_type: 'physics_motion_3d',
    defaults: { v0: 4, a: 1.5 },
    sliders: [
      { key: 'v0', label: '初速度 v₀ (m/s)', min: -10, max: 20, step: 0.5 },
      { key: 'a', label: '加速度 a (m/s²)', min: -5, max: 5, step: 0.1 },
    ],
  },
  {
    key: 'force',
    label: '受力运动（牛顿第二）',
    scene_type: 'physics_force_3d',
    defaults: { m: 2, a: 3 },
    sliders: [
      { key: 'm', label: '质量 m (kg)', min: 0.5, max: 10, step: 0.1 },
      { key: 'a', label: '加速度 a (m/s²)', min: 0.5, max: 8, step: 0.1 },
    ],
  },
  {
    key: 'spring',
    label: '弹簧振子（阻尼简谐）',
    scene_type: 'physics_spring_3d',
    defaults: { k: 24, m: 1.2, x: 1.4, damping: 0.3 },
    sliders: [
      { key: 'k', label: '劲度系数 k', min: 4, max: 80, step: 1 },
      { key: 'm', label: '质量 m (kg)', min: 0.2, max: 5, step: 0.1 },
      { key: 'x', label: '初始位移 A (m)', min: 0.2, max: 2.5, step: 0.1 },
      { key: 'damping', label: '阻尼系数', min: 0, max: 4, step: 0.05 },
    ],
  },
  {
    key: 'collision',
    label: '1D 碰撞',
    scene_type: 'physics_collision_3d',
    defaults: { m1: 1.5, m2: 1, v1: 4.5, v2: -2.5, restitution: 0.9 },
    sliders: [
      { key: 'm1', label: 'm₁ (kg)', min: 0.2, max: 5, step: 0.1 },
      { key: 'm2', label: 'm₂ (kg)', min: 0.2, max: 5, step: 0.1 },
      { key: 'v1', label: 'v₁ (m/s)', min: -8, max: 8, step: 0.1 },
      { key: 'v2', label: 'v₂ (m/s)', min: -8, max: 8, step: 0.1 },
      { key: 'restitution', label: '恢复系数 e', min: 0, max: 1, step: 0.05 },
    ],
  },
  {
    key: 'orbit',
    label: '天体轨道',
    scene_type: 'physics_orbit_3d',
    defaults: { orbit_radius: 3.6, tangential_speed: 2.25, eccentricity: 0.18 },
    sliders: [
      { key: 'orbit_radius', label: '轨道半径 r', min: 1.5, max: 6, step: 0.1 },
      { key: 'tangential_speed', label: '切向速度', min: 0.5, max: 4, step: 0.05 },
      { key: 'eccentricity', label: '离心率 e', min: 0, max: 0.5, step: 0.01 },
    ],
  },
];

export default function PhysicsDemoPage() {
  const [sceneKey, setSceneKey] = useState('projectile');
  const scene = SCENES.find((s) => s.key === sceneKey)!;
  const [propsData, setPropsData] = useState<Record<string, number>>(scene.defaults);

  React.useEffect(() => {
    setPropsData(scene.defaults);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sceneKey]);

  const artifact: RenderArtifact = useMemo(() => ({
    scene_type: scene.scene_type,
    render_mode: 'react_iframe',
    render_manifest: {
      entry: 'index.tsx',
      framework: 'react',
      sandbox: 'iframe',
      render_mode: 'react_iframe',
      mount_selector: '#root',
      initial_props: scene.defaults,
    },
    code_bundle: {},
    result_summary: `${scene.label} demo`,
  }), [scene]);

  return (
    <div style={{ padding: '24px 32px', maxWidth: 1400, margin: '0 auto' }}>
      <Typography.Title level={2} style={{ marginTop: 0 }}>
        物理预览开发演示 · 2D 教学视图 / 3D / 数据曲线
      </Typography.Title>
      <Typography.Paragraph type="secondary">
        无需后端，直接挂载 mock artifact。切换场景、调参、点击「保存对比」可看到历史轨迹叠加效果。
      </Typography.Paragraph>

      <Card size="small" style={{ marginBottom: 16 }}>
        <Space direction="vertical" size={12} style={{ width: '100%' }}>
          <div>
            <span style={{ marginRight: 12, color: '#64748b' }}>场景：</span>
            <Radio.Group value={sceneKey} onChange={(e) => setSceneKey(e.target.value)} buttonStyle="solid">
              {SCENES.map((s) => (
                <Radio.Button key={s.key} value={s.key}>{s.label}</Radio.Button>
              ))}
            </Radio.Group>
          </div>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(260px, 1fr))', gap: 12 }}>
            {scene.sliders.map((sl) => (
              <div key={sl.key}>
                <div style={{ fontSize: 12, color: '#475569', marginBottom: 2 }}>
                  {sl.label}：<b style={{ color: '#0369a1' }}>{(propsData[sl.key] ?? sl.min).toFixed(2)}</b>
                </div>
                <Slider
                  min={sl.min}
                  max={sl.max}
                  step={sl.step}
                  value={propsData[sl.key] ?? sl.min}
                  onChange={(v) => setPropsData((p) => ({ ...p, [sl.key]: Array.isArray(v) ? v[0] : v }))}
                />
              </div>
            ))}
          </div>
        </Space>
      </Card>

      <R3FPhysicsPreview artifact={artifact} propsData={propsData} />
    </div>
  );
}

