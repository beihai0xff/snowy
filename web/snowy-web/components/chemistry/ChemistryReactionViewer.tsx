/**
 * Snowy v7 · 化学 · 反应可视化（M12）
 *
 * 渲染 ChemistryReactionPackage：
 *   - 配平方程式（反应物 → 产物）
 *   - 反应条件（温度、催化剂、能量来源等）
 *   - 各物种 3D 球棍模型（MoleculeViewer）
 *   - 电子转移（如有）
 *   - 能量曲线（SVG 简笔，定性）
 */

'use client';

import React from 'react';
import dynamic from 'next/dynamic';
import { Card, Col, Row, Space, Tag, Typography } from 'antd';
import type { ChemistryReactionPackage } from '@/lib/api';

const MoleculeViewer = dynamic(() => import('./MoleculeViewer'), { ssr: false });

const { Text } = Typography;

const REACTION_TYPE_LABEL: Record<string, string> = {
  neutralization: '中和反应',
  redox: '氧化还原',
  displacement: '置换反应',
  metathesis: '复分解反应',
  electrolysis: '电解反应',
  ionization: '电离',
  hydrolysis: '水解',
  combustion: '燃烧',
  unknown: '反应',
};

function formatSide(side: { species: string; coef: number }[]): string {
  return side
    .map((s) => (s.coef > 1 ? `${s.coef}${s.species}` : s.species))
    .join(' + ');
}

interface EnergyDiagramProps {
  reactant: number;
  product: number;
  activation: number;
  exothermic: boolean;
}

function EnergyDiagram({ reactant, product, activation, exothermic }: EnergyDiagramProps) {
  const w = 320;
  const h = 120;
  const minE = Math.min(reactant, product, reactant + activation, product + activation) - 5;
  const maxE = Math.max(reactant, product, reactant + activation, product + activation) + 5;
  const span = maxE - minE || 1;
  const ny = (e: number) => h - 18 - ((e - minE) / span) * (h - 32);
  const top = Math.max(reactant, product) + activation;
  const path = [
    `M 20 ${ny(reactant)}`,
    `L 80 ${ny(reactant)}`,
    `Q ${w / 2} ${ny(top)} ${w - 80} ${ny(product)}`,
    `L ${w - 20} ${ny(product)}`,
  ].join(' ');

  return (
    <svg width={w} height={h} aria-label="反应能量曲线" style={{ background: '#fafafa', borderRadius: 8 }}>
      <path d={path} fill="none" stroke={exothermic ? '#fa8c16' : '#1677ff'} strokeWidth={2} />
      <text x={20} y={ny(reactant) - 6} fontSize={11} fill="#666">反应物</text>
      <text x={w - 70} y={ny(product) - 6} fontSize={11} fill="#666">产物</text>
      <text x={w / 2 - 18} y={ny(top) - 4} fontSize={11} fill="#999">过渡态</text>
      <text x={4} y={h - 4} fontSize={10} fill="#bbb">{exothermic ? '放热（ΔH<0）' : '吸热（ΔH>0）'}</text>
    </svg>
  );
}

export interface ChemistryReactionViewerProps {
  pkg: ChemistryReactionPackage;
}

export default function ChemistryReactionViewer({ pkg }: ChemistryReactionViewerProps) {
  const lhs = formatSide(pkg.equation.reactants);
  const rhs = formatSide(pkg.equation.products);
  const arrow = pkg.equation.arrow || '→';
  const species = pkg.species || [];
  const typeLabel = REACTION_TYPE_LABEL[pkg.reaction_type] || '化学反应';

  return (
    <Space direction="vertical" size="middle" style={{ width: '100%' }}>
      <Card size="small" styles={{ body: { padding: 12 } }}>
        <Space direction="vertical" size={6} style={{ width: '100%' }}>
          <Space size={6} wrap>
            <Tag color="magenta">{typeLabel}</Tag>
            {pkg.conditions.temperature && <Tag color="orange">{pkg.conditions.temperature}</Tag>}
            {pkg.conditions.catalyst && <Tag color="purple">催化剂：{pkg.conditions.catalyst}</Tag>}
            {pkg.conditions.solvent && <Tag color="cyan">{pkg.conditions.solvent}</Tag>}
            {pkg.conditions.energy && <Tag color="gold">{pkg.conditions.energy}</Tag>}
            {pkg.conditions.pressure && <Tag>{pkg.conditions.pressure}</Tag>}
          </Space>
          <Text style={{ fontFamily: 'monospace', fontSize: 18 }}>
            {lhs}  <span style={{ color: '#1677ff' }}>{arrow}</span>  {rhs}
          </Text>
        </Space>
      </Card>

      {species.length > 0 && (
        <Row gutter={[12, 12]}>
          {species.map((sp, i) => (
            <Col xs={24} sm={12} md={8} key={`${sp.formula}-${i}`}>
              <Card size="small" title={<Text strong>{sp.formula}</Text>} styles={{ body: { padding: 8 } }}>
                <MoleculeViewer species={sp} height={180} />
              </Card>
            </Col>
          ))}
        </Row>
      )}

      {(pkg.electron_transfer || []).length > 0 && (
        <Card size="small" title="电子转移">
          <Space direction="vertical" size={4} style={{ width: '100%' }}>
            {(pkg.electron_transfer || []).map((e, i) => (
              <Text key={i} style={{ fontFamily: 'monospace' }}>
                {e.from_element}({e.from_oxidation >= 0 ? `+${e.from_oxidation}` : e.from_oxidation})
                {' → '}
                {e.to_element}({e.to_oxidation >= 0 ? `+${e.to_oxidation}` : e.to_oxidation})
                ，转移 <Tag color="blue">{e.electrons}e⁻</Tag>
              </Text>
            ))}
          </Space>
        </Card>
      )}

      {pkg.energy_profile && (
        <Card size="small" title="能量曲线（定性）">
          <EnergyDiagram
            reactant={pkg.energy_profile.reactant_energy}
            product={pkg.energy_profile.product_energy}
            activation={pkg.energy_profile.activation_energy}
            exothermic={pkg.energy_profile.exothermic}
          />
        </Card>
      )}

      {(pkg.warnings || []).length > 0 && (
        <Space wrap>
          {(pkg.warnings || []).map((w, i) => (
            <Tag color="orange" key={i}>⚠ {w}</Tag>
          ))}
        </Space>
      )}
    </Space>
  );
}
