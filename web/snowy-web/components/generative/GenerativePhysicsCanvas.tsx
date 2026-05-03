'use client';

import React, { useMemo } from 'react';
import { Card, Col, Empty, Row, Space, Tag, Typography } from 'antd';
import type { DynamicSimulationSpec } from '@/lib/api';

const { Text } = Typography;

interface Props {
  spec?: DynamicSimulationSpec;
  values: Record<string, number>;
}

function valueOf(values: Record<string, number>, key: string, fallback: number): number {
  const value = values[key];
  return Number.isFinite(value) ? value : fallback;
}

export default function GenerativePhysicsCanvas({ spec, values }: Props) {
  const points = useMemo(() => {
    if (!spec) return [];
    const v0 = valueOf(values, 'v0', 20);
    const angleDeg = valueOf(values, 'angle_deg', 0);
    const h = valueOf(values, 'h', valueOf(values, 'height', 20));
    const g = valueOf(values, 'g', 9.8);
    const tMax = valueOf(values, 't', Math.sqrt(Math.max(0.1, 2 * h / Math.max(g, 0.1))));
    const angle = angleDeg * Math.PI / 180;
    return Array.from({ length: 32 }, (_, index) => {
      const t = (tMax * index) / 31;
      const x = v0 * Math.cos(angle) * t;
      const y = Math.max(0, h + v0 * Math.sin(angle) * t - 0.5 * g * t * t);
      return { x, y };
    });
  }, [spec, values]);

  if (!spec) return <Empty description="暂无物理仿真逻辑" />;

  const maxX = Math.max(...points.map((p) => p.x), 1);
  const maxY = Math.max(...points.map((p) => p.y), 1);
  const polyline = points.map((p) => `${40 + (p.x / maxX) * 520},${300 - (p.y / maxY) * 240}`).join(' ');
  const last = points[points.length - 1];
  const lastX = last ? 40 + (last.x / maxX) * 520 : 40;
  const lastY = last ? 300 - (last.y / maxY) * 240 : 300;

  return (
    <Space direction="vertical" style={{ width: '100%' }} size="middle">
      <Space wrap>
        <Tag color="blue">{spec.simulation_type}</Tag>
        <Tag color="purple">{spec.runtime}</Tag>
        {spec.local_recompute_allowed && <Tag color="green">本地重算</Tag>}
        {(spec.render_instructions?.layers || []).slice(0, 4).map((layer) => <Tag key={layer}>{layer}</Tag>)}
      </Space>
      <div style={{ border: '1px solid #e5e7eb', borderRadius: 12, background: 'linear-gradient(#eff6ff,#f8fafc)', padding: 12 }}>
        <svg viewBox="0 0 620 340" style={{ width: '100%', minHeight: 360 }} role="img" aria-label="生成式物理仿真画布">
          <line x1="40" y1="300" x2="580" y2="300" stroke="#64748b" strokeWidth="2" />
          <line x1="40" y1="40" x2="40" y2="300" stroke="#64748b" strokeWidth="2" />
          <polyline points={polyline} fill="none" stroke="#1677ff" strokeWidth="4" strokeLinecap="round" />
          <circle cx="40" cy="60" r="8" fill="#52c41a" />
          <circle cx={lastX} cy={lastY} r="9" fill="#fa541c" />
          <line x1={lastX} y1={lastY} x2={Math.min(580, lastX + 52)} y2={lastY} stroke="#fa8c16" strokeWidth="3" markerEnd="url(#arrow)" />
          <defs>
            <marker id="arrow" markerWidth="10" markerHeight="10" refX="8" refY="3" orient="auto" markerUnits="strokeWidth">
              <path d="M0,0 L0,6 L9,3 z" fill="#fa8c16" />
            </marker>
          </defs>
          <text x="45" y="322" fill="#475569">x</text>
          <text x="18" y="48" fill="#475569">y</text>
          <text x={Math.max(60, lastX - 70)} y={Math.max(30, lastY - 18)} fill="#334155">生成式轨迹</text>
        </svg>
      </div>
      <Row gutter={[12, 12]}>
        {(spec.formulas || []).map((formula) => (
          <Col xs={24} md={12} key={formula.id}>
            <Card size="small" title={formula.meaning}>
              <Text code>{formula.expr}</Text>
            </Card>
          </Col>
        ))}
      </Row>
      {(spec.render_instructions?.annotations || []).length > 0 && (
        <Space wrap>
          {spec.render_instructions?.annotations?.map((item) => <Tag color="cyan" key={item}>{item}</Tag>)}
        </Space>
      )}
    </Space>
  );
}
