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

function isProjectile(spec: DynamicSimulationSpec): boolean {
  return /projectile|平抛|抛体/.test(`${spec.simulation_type} ${(spec.render_instructions?.layers || []).join(' ')}`.toLowerCase());
}

function formatNumber(value: number, digits = 2) {
  if (!Number.isFinite(value)) return '-';
  return Math.abs(value) >= 100 ? value.toFixed(1) : value.toFixed(digits);
}

export default function GenerativePhysicsCanvas({ spec, values }: Props) {
  const trajectory = useMemo(() => {
    if (!spec) return { points: [] as { x: number; y: number }[], landingX: 0, landingTime: 0, targetX: 40, error: 0 };
    const v0 = valueOf(values, 'v0', 20);
    const angleDeg = valueOf(values, 'angle_deg', 0);
    const h = valueOf(values, 'h', valueOf(values, 'height', 20));
    const g = Math.max(0.1, valueOf(values, 'g', 9.8));
    const targetX = valueOf(values, 'target_x', 40);
    const angle = angleDeg * Math.PI / 180;
    const vx = v0 * Math.cos(angle);
    const vy0 = v0 * Math.sin(angle);
    const landingTime = (vy0 + Math.sqrt(Math.max(0, vy0 * vy0 + 2 * g * h))) / g;
    const tMax = Math.max(0.1, landingTime || valueOf(values, 't', 2));
    const points = Array.from({ length: 42 }, (_, index) => {
      const t = (tMax * index) / 41;
      const x = Math.max(0, vx * t);
      const y = Math.max(0, h + vy0 * t - 0.5 * g * t * t);
      return { x, y };
    });
    const landingX = Math.max(0, vx * tMax);
    return { points, landingX, landingTime: tMax, targetX, error: landingX - targetX };
  }, [spec, values]);

  if (!spec) return <Empty description="暂无物理仿真逻辑" />;

  const maxX = Math.max(...trajectory.points.map((p) => p.x), trajectory.targetX, 1);
  const maxY = Math.max(...trajectory.points.map((p) => p.y), 1);
  const sx = (x: number) => 40 + (x / maxX) * 520;
  const sy = (y: number) => 300 - (y / maxY) * 240;
  const polyline = trajectory.points.map((p) => `${sx(p.x)},${sy(p.y)}`).join(' ');
  const last = trajectory.points[trajectory.points.length - 1];
  const lastX = last ? sx(last.x) : 40;
  const lastY = last ? sy(last.y) : 300;
  const targetScreenX = sx(trajectory.targetX);
  const hit = Math.abs(trajectory.error) <= 0.5;

  return (
    <Space direction="vertical" style={{ width: '100%' }} size="middle">
      <Space wrap>
        <Tag color="blue">{spec.simulation_type}</Tag>
        <Tag color="purple">{spec.runtime}</Tag>
        {spec.local_recompute_allowed && <Tag color="green">本地重算</Tag>}
        {(spec.render_instructions?.layers || []).slice(0, 6).map((layer) => <Tag key={layer}>{layer}</Tag>)}
      </Space>

      <div style={{ border: '1px solid #e5e7eb', borderRadius: 12, background: 'linear-gradient(#eff6ff,#f8fafc)', padding: 12 }}>
        <svg viewBox="0 0 620 340" style={{ width: '100%', minHeight: 360 }} role="img" aria-label="生成式物理仿真画布">
          <defs>
            <marker id="arrow" markerWidth="10" markerHeight="10" refX="8" refY="3" orient="auto" markerUnits="strokeWidth">
              <path d="M0,0 L0,6 L9,3 z" fill="#fa8c16" />
            </marker>
            <marker id="blue-arrow" markerWidth="10" markerHeight="10" refX="8" refY="3" orient="auto" markerUnits="strokeWidth">
              <path d="M0,0 L0,6 L9,3 z" fill="#1677ff" />
            </marker>
          </defs>
          <line x1="40" y1="300" x2="580" y2="300" stroke="#64748b" strokeWidth="2" />
          <line x1="40" y1="40" x2="40" y2="300" stroke="#64748b" strokeWidth="2" />
          <polyline points={polyline} fill="none" stroke="#1677ff" strokeWidth="4" strokeLinecap="round" />
          {isProjectile(spec) && (
            <>
              <rect x={targetScreenX - 16} y="284" width="32" height="16" rx="4" fill={hit ? '#52c41a' : '#f59e0b'} opacity="0.85" />
              <line x1={targetScreenX} y1="300" x2={targetScreenX} y2="245" stroke="#f59e0b" strokeDasharray="4 4" />
              <text x={targetScreenX - 28} y="238" fill="#92400e">目标区</text>
            </>
          )}
          <circle cx={40} cy={sy(valueOf(values, 'h', 20))} r="8" fill="#52c41a" />
          <circle cx={lastX} cy={lastY} r="9" fill="#fa541c" />
          <line x1={lastX} y1={lastY} x2={Math.min(580, lastX + 52)} y2={lastY} stroke="#fa8c16" strokeWidth="3" markerEnd="url(#arrow)" />
          <line x1={Math.max(40, lastX - 40)} y1={lastY} x2={Math.max(40, lastX - 40)} y2={Math.min(300, lastY + 48)} stroke="#1677ff" strokeWidth="3" markerEnd="url(#blue-arrow)" />
          <text x="45" y="322" fill="#475569">x</text>
          <text x="18" y="48" fill="#475569">y</text>
          <text x={Math.max(60, lastX - 70)} y={Math.max(30, lastY - 18)} fill="#334155">生成式轨迹</text>
        </svg>
      </div>

      {isProjectile(spec) && (
        <Row gutter={[12, 12]}>
          <Col xs={24} md={8}><Card size="small" title="落地时间"><Text strong>{formatNumber(trajectory.landingTime)} s</Text><br /><Text type="secondary">由竖直方向决定</Text></Card></Col>
          <Col xs={24} md={8}><Card size="small" title="水平落点"><Text strong>{formatNumber(trajectory.landingX)} m</Text><br /><Text type="secondary">x = v0·t</Text></Card></Col>
          <Col xs={24} md={8}><Card size="small" title="命中误差"><Text strong type={hit ? 'success' : 'warning'}>{formatNumber(trajectory.error)} m</Text><br /><Text type="secondary">目标 {formatNumber(trajectory.targetX)} m</Text></Card></Col>
        </Row>
      )}

      {(spec.vectors || []).length > 0 && (
        <Card size="small" title="矢量语义">
          <Space wrap>
            {spec.vectors?.map((vector) => <Tag color="geekblue" key={vector.id}>{vector.label}：{vector.meaning || vector.x_expr || vector.y_expr}</Tag>)}
          </Space>
        </Card>
      )}

      <Row gutter={[12, 12]}>
        {(spec.formulas || []).map((formula) => (
          <Col xs={24} md={12} key={formula.id}>
            <Card size="small" title={formula.meaning}>
              <Text code>{formula.expr}</Text>
            </Card>
          </Col>
        ))}
      </Row>

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
