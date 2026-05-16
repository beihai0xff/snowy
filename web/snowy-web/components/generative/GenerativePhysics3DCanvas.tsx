/**
 * Snowy v6 · 生成式物理 3D 画布
 *
 * 把 modeling/compile 返回的 simulation_logic + 当前 values 适配为 RenderArtifact，
 * 直接交给 R3FPhysicsPreview 渲染（轨道 / 弹簧 / 碰撞 / 抛体 / 受力 / 通用运动）。
 *
 * 按用户决策（v6 / 2026-05）：主链路直接走 R3F，不做 SVG 回退。
 */

'use client';

import React, { useMemo } from 'react';
import dynamic from 'next/dynamic';
import { Card, Col, Empty, Row, Space, Tag, Typography } from 'antd';
import type { DynamicSimulationSpec, RenderArtifact } from '@/lib/api';

const { Text } = Typography;

const R3FPhysicsPreview = dynamic(
  () => import('@/components/physics/r3f/R3FPhysicsPreview'),
  { ssr: false, loading: () => <div className="snowy-r3f-shell snowy-r3f-shell--loading" role="status" aria-live="polite"><span>加载 3D 引擎…</span></div> },
);

interface Props {
  spec?: DynamicSimulationSpec;
  values: Record<string, number>;
}

const SCENE_KEYWORDS = ['orbit', 'spring', 'collision', 'projectile', 'force', 'motion'] as const;

function deriveSceneType(spec: DynamicSimulationSpec): string {
  const raw = `${spec.simulation_type || ''} ${(spec.render_instructions?.layers || []).join(' ')}`.toLowerCase();
  // 英文关键词直接匹配
  for (const kw of SCENE_KEYWORDS) {
    if (raw.includes(kw)) return kw;
  }
  // 中文关键词映射
  if (/平抛|抛体|抛物/.test(raw)) return 'projectile';
  if (/轨道|公转|开普勒|引力/.test(raw)) return 'orbit';
  if (/弹簧|振子|简谐|胡克/.test(raw)) return 'spring';
  if (/碰撞|相撞|动量守恒/.test(raw)) return 'collision';
  if (/斜面|受力|摩擦|牛二|牛顿第二/.test(raw)) return 'force';
  return 'motion';
}

function defaultInitialProps(spec: DynamicSimulationSpec): Record<string, number> {
  return (spec.variables || []).reduce<Record<string, number>>((acc, v) => {
    if (typeof v.default === 'number' && Number.isFinite(v.default)) acc[v.name] = v.default;
    return acc;
  }, {});
}

export default function GenerativePhysics3DCanvas({ spec, values }: Props) {
  const sceneType = useMemo(() => (spec ? deriveSceneType(spec) : 'motion'), [spec]);
  const initialProps = useMemo(() => (spec ? defaultInitialProps(spec) : {}), [spec]);
  const artifact: RenderArtifact | null = useMemo(() => {
    if (!spec) return null;
    return {
      scene_type: sceneType,
      render_mode: 'react_iframe',
      render_manifest: {
        entry: 'r3f://snowy/modeling',
        framework: 'react-three-fiber',
        sandbox: 'inline',
        render_mode: 'react_iframe',
        mount_selector: '#snowy-modeling-canvas',
        initial_props: initialProps,
      },
      code_bundle: {},
      result_summary: spec.simulation_type || '生成式物理仿真',
    };
  }, [spec, sceneType, initialProps]);

  if (!spec || !artifact) return <Empty description="暂无物理仿真逻辑" />;

  const sceneLabel = (
    sceneType === 'orbit' ? '天体轨道' :
    sceneType === 'projectile' ? '抛体运动' :
    sceneType === 'spring' ? '弹簧振子' :
    sceneType === 'collision' ? '碰撞演示' :
    sceneType === 'force' ? '受力分析' : '通用运动'
  );

  return (
    <Space direction="vertical" style={{ width: '100%' }} size="middle">
      <Space wrap>
        <Tag color="cyan">{sceneLabel}</Tag>
        <Tag color="blue">{spec.simulation_type}</Tag>
        <Tag color="purple">{spec.runtime}</Tag>
        {spec.local_recompute_allowed && <Tag color="green">本地重算</Tag>}
        {(spec.render_instructions?.layers || []).slice(0, 4).map((layer) => <Tag key={layer}>{layer}</Tag>)}
      </Space>

      <R3FPhysicsPreview artifact={artifact} propsData={values} />

      {(spec.formulas || []).length > 0 && (
        <Row gutter={[12, 12]}>
          {spec.formulas?.map((formula) => (
            <Col xs={24} md={12} key={formula.id}>
              <Card size="small" title={formula.meaning}>
                <Text code>{formula.expr}</Text>
              </Card>
            </Col>
          ))}
        </Row>
      )}

      {(spec.vectors || []).length > 0 && (
        <Card size="small" title="矢量语义">
          <Space wrap>
            {spec.vectors?.map((vector) => (
              <Tag color="geekblue" key={vector.id}>
                {vector.label}：{vector.meaning || vector.x_expr || vector.y_expr}
              </Tag>
            ))}
          </Space>
        </Card>
      )}

      {(spec.curves || []).length > 0 && (
        <Row gutter={[12, 12]}>
          {spec.curves?.map((curve) => (
            <Col xs={24} md={12} key={curve.id}>
              <Card size="small" title={curve.title}>
                {curve.y_expr && <Text code>{curve.y_expr}</Text>}
                {curve.meaning && <div><Text type="secondary">{curve.meaning}</Text></div>}
              </Card>
            </Col>
          ))}
        </Row>
      )}

      {(spec.render_instructions?.annotations || []).length > 0 && (
        <Space wrap>
          {spec.render_instructions?.annotations?.map((item) => <Tag color="cyan" key={item}>{item}</Tag>)}
        </Space>
      )}
    </Space>
  );
}
