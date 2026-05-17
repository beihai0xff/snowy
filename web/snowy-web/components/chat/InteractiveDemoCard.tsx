'use client';

/**
 * v7 §3 / §6：InteractiveDemoCard
 * 在 ChatBubble 中渲染 generative model package 的交互演示。
 *
 * 3 层参数交互（V7 §3）：
 *   1. 本地表达式即时重算（默认）—— `evalExpr`
 *   2. recompute REST（数值偏离窗口或公式不可本地求值时）
 *   3. regenerate REST（结构性变更，由父级 ChatBubble/对话主流程触发）
 *
 * 此组件聚焦层 1+2；层 3 通过 `onRegenerateRequest` 抛给上层 chat。
 */

import React, { useEffect, useMemo, useRef, useState } from 'react';
import dynamic from 'next/dynamic';
import { Alert, Button, Card, Input, Popover, Slider, Space, Spin, Tag, Tooltip, Typography } from 'antd';
import { ReloadOutlined, ThunderboltOutlined, UndoOutlined } from '@ant-design/icons';
import {
  api,
  type GenerativeModelPackage,
  type VariableSpec,
  type FormulaSpec,
} from '@/lib/api';
import { evalExpr } from '@/lib/formula/evaluator';

const GenerativePhysics3DCanvas = dynamic(
  () => import('@/components/generative/GenerativePhysics3DCanvas'),
  { ssr: false, loading: () => <div style={{ padding: 12, fontSize: 12, color: '#999' }}>加载物理画布…</div> },
);
const ChemistryReactionViewer = dynamic(
  () => import('@/components/chemistry/ChemistryReactionViewer'),
  { ssr: false, loading: () => <div style={{ padding: 12, fontSize: 12, color: '#999' }}>加载化学反应视图…</div> },
);
const GenerativeBiologyGraph = dynamic(
  () => import('@/components/generative/GenerativeBiologyGraph'),
  { ssr: false, loading: () => <div style={{ padding: 12, fontSize: 12, color: '#999' }}>加载生物图…</div> },
);

const { Text } = Typography;

export interface InteractiveDemoCardProps {
  pkg: GenerativeModelPackage;
  /** package 变更时（recompute/regenerate 返回新版本）通知上层。 */
  onPackageUpdate?: (next: GenerativeModelPackage) => void;
  /** 用户希望换一个不同的方案——由上层 chat 串入对话流。 */
  onRegenerateRequest?: (reason: string) => void;
  /** 偏离原值多大比例时改走 recompute（默认 0.5，即 ±50%）。 */
  recomputeThreshold?: number;
}

function collectVariables(pkg: GenerativeModelPackage): VariableSpec[] {
  const seen = new Map<string, VariableSpec>();
  for (const v of pkg.simulation_logic?.variables ?? []) seen.set(v.name, v);
  for (const v of pkg.generative_model.variables ?? []) {
    if (!seen.has(v.name)) seen.set(v.name, v);
  }
  return [...seen.values()];
}

function collectFormulas(pkg: GenerativeModelPackage): FormulaSpec[] {
  return pkg.simulation_logic?.formulas ?? [];
}

export default function InteractiveDemoCard({
  pkg,
  onPackageUpdate,
  onRegenerateRequest,
  recomputeThreshold = 0.5,
}: InteractiveDemoCardProps) {
  const variables = useMemo(() => collectVariables(pkg), [pkg]);
  const formulas = useMemo(() => collectFormulas(pkg), [pkg]);
  const localRecomputeAllowed = pkg.simulation_logic?.local_recompute_allowed ?? true;

  const initial = useMemo(() => {
    const map: Record<string, number> = {};
    for (const v of variables) map[v.name] = v.default;
    return map;
  }, [variables]);

  const [values, setValues] = useState<Record<string, number>>(initial);
  const [recomputing, setRecomputing] = useState(false);
  const [recomputeError, setRecomputeError] = useState<string | null>(null);
  const [regenReason, setRegenReason] = useState('');
  const [regenOpen, setRegenOpen] = useState(false);
  const recomputeTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    setValues(initial);
    setRecomputeError(null);
  }, [initial, pkg.package_id]);

  const formulaResults = useMemo(() => {
    return formulas.map((f) => ({ ...f, value: evalExpr(f.expr, values) }));
  }, [formulas, values]);

  const shouldRecompute = (next: Record<string, number>): boolean => {
    if (!localRecomputeAllowed) return true;
    for (const v of variables) {
      const cur = next[v.name];
      const ref = v.default;
      const span = Math.max(Math.abs(v.max - v.min), Math.abs(ref) || 1);
      if (Math.abs(cur - ref) / span > recomputeThreshold) return true;
    }
    // 任一公式无法本地求值也降级到 recompute。
    for (const f of formulas) {
      if (evalExpr(f.expr, next) === null) return true;
    }
    return false;
  };

  const triggerRecompute = (next: Record<string, number>) => {
    if (recomputeTimer.current) clearTimeout(recomputeTimer.current);
    recomputeTimer.current = setTimeout(async () => {
      setRecomputing(true);
      setRecomputeError(null);
      try {
        const res = await api.recomputeModelingPackage(pkg.package_id, {
          overrides: next,
        });
        if (res.data) onPackageUpdate?.(res.data);
      } catch (err) {
        setRecomputeError(err instanceof Error ? err.message : 'recompute 失败');
      } finally {
        setRecomputing(false);
      }
    }, 300);
  };

  const handleChange = (name: string, value: number) => {
    const next = { ...values, [name]: value };
    setValues(next);
    if (shouldRecompute(next)) triggerRecompute(next);
  };

  useEffect(() => () => {
    if (recomputeTimer.current) clearTimeout(recomputeTimer.current);
  }, []);

  return (
    <Card
      size="small"
      title={
        <Space>
          <ThunderboltOutlined />
          <span>{pkg.learning_model.topic || '交互演示'}</span>
          <Tag color="blue">{pkg.domain}</Tag>
          {pkg.status === 'fallback' && (
            <Tooltip title="AI 没找到完整公式或仿真逻辑，已用近似版本展示，仅供参考">
              <Tag color="orange" style={{ cursor: 'help' }}>降级</Tag>
            </Tooltip>
          )}
          {pkg.status === 'regenerate_failed' && <Tag color="red">再生失败</Tag>}
        </Space>
      }
      extra={
        <Space size={4}>
          {variables.length > 0 && (
            <Tooltip title="重置为默认参数">
              <Button size="small" icon={<UndoOutlined />} onClick={() => setValues(initial)} />
            </Tooltip>
          )}
          {onRegenerateRequest && (
            <Popover
              trigger="click"
              open={regenOpen}
              onOpenChange={setRegenOpen}
              placement="bottomRight"
              content={
                <div style={{ width: 260 }}>
                  <Input.TextArea
                    autoSize={{ minRows: 2, maxRows: 4 }}
                    placeholder="想换什么？例如：加入摩擦力 / 用更简单的公式"
                    value={regenReason}
                    onChange={(e) => setRegenReason(e.target.value)}
                  />
                  <Space style={{ marginTop: 8, justifyContent: 'flex-end', width: '100%' }}>
                    <Button size="small" onClick={() => { setRegenOpen(false); setRegenReason(''); }}>取消</Button>
                    <Button size="small" type="primary" onClick={() => {
                      onRegenerateRequest(regenReason.trim() || '用户希望换一个演示方案');
                      setRegenOpen(false);
                      setRegenReason('');
                    }}>再生</Button>
                  </Space>
                </div>
              }
            >
              <Button size="small" icon={<ReloadOutlined />}>换一个</Button>
            </Popover>
          )}
        </Space>
      }
      style={{ marginTop: 8 }}
    >
      <Space direction="vertical" size="small" style={{ width: '100%' }}>
        <Text type="secondary">{pkg.learning_model.learning_goal}</Text>

        {pkg.domain === 'physics' && pkg.simulation_logic && (
          <GenerativePhysics3DCanvas spec={pkg.simulation_logic} values={values} />
        )}
        {pkg.domain === 'chemistry' && pkg.simulation_logic?.chemistry_reaction && (
          <ChemistryReactionViewer pkg={pkg.simulation_logic.chemistry_reaction} />
        )}
        {pkg.domain === 'biology' && pkg.visualization_graph && (
          <GenerativeBiologyGraph spec={pkg.visualization_graph} values={values} />
        )}

        {variables.length > 0 && pkg.domain !== 'chemistry' && (
          <div>
            {variables.map((v) => (
              <div key={v.name} style={{ marginBottom: 8 }}>
                <Space size="small">
                  <Text strong>{v.label}</Text>
                  {v.unit && <Text type="secondary">({v.unit})</Text>}
                  <Text>{values[v.name]?.toFixed(2)}</Text>
                </Space>
                <Slider
                  min={v.min}
                  max={v.max}
                  step={v.step ?? (v.max - v.min) / 100}
                  value={values[v.name] ?? v.default}
                  onChange={(value) => handleChange(v.name, value as number)}
                />
              </div>
            ))}
          </div>
        )}
        {pkg.domain === 'chemistry' && variables.length > 0 && (
          <Text type="secondary" style={{ fontSize: 12 }}>
            化学反应包暂不响应参数调节，如需变体请点「换一个」描述需求
          </Text>
        )}

        {formulaResults.length > 0 && (
          <div>
            <Text type="secondary">实时计算：</Text>
            <ul style={{ marginTop: 4, paddingLeft: 18 }}>
              {formulaResults.map((f) => (
                <li key={f.id}>
                  <Text code>{f.expr}</Text>
                  {' = '}
                  {f.value === null ? <Tag color="orange">需重算</Tag> : f.value.toFixed(3)}
                  {f.meaning && (
                    <Text type="secondary" style={{ marginLeft: 8 }}>
                      {f.meaning}
                    </Text>
                  )}
                </li>
              ))}
            </ul>
          </div>
        )}

        {recomputing && (
          <Space>
            <Spin size="small" />
            <Text type="secondary">正在重新求解……</Text>
          </Space>
        )}
        {recomputeError && <Alert type="warning" message={recomputeError} showIcon />}
      </Space>
    </Card>
  );
}
