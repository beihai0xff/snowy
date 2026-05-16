/**
 * Snowy v6 · 生物插画共享 props
 */

export interface IllustrationProps {
  topic?: string;
  /** 参数值（如光强、温度、CO₂ 浓度等） */
  values?: Record<string, number>;
  /** 是否启用入场动画 */
  animated?: boolean;
}

export function numVal(values: Record<string, number> | undefined, key: string, fallback: number): number {
  const v = values?.[key];
  return Number.isFinite(v) ? (v as number) : fallback;
}
