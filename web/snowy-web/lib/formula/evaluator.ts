/**
 * v7 §3 / §6：纯前端公式求值（不调后端）。
 * 用于 InteractiveDemoCard 拖动滑块时的即时更新（第 1 层交互）。
 *
 * 通过 `expr-eval` 解析受控表达式：仅支持算术、内置数学函数，
 * 不暴露 JS 全局对象，避免任意代码执行。
 */
import { Parser, type Expression } from 'expr-eval';

const parser = new Parser({
  operators: {
    add: true,
    concatenate: false,
    conditional: true,
    divide: true,
    factorial: false,
    multiply: true,
    power: true,
    remainder: true,
    subtract: true,
    logical: true,
    comparison: true,
    'in': false,
    assignment: false,
  },
});

const exprCache = new Map<string, Expression>();

function compile(expr: string): Expression {
  let parsed = exprCache.get(expr);
  if (!parsed) {
    parsed = parser.parse(expr);
    exprCache.set(expr, parsed);
  }
  return parsed;
}

/**
 * 求值单个表达式。失败时返回 `null`，调用方决定如何降级（占位 / 触发 recompute）。
 */
export function evalExpr(expr: string, vars: Record<string, number>): number | null {
  if (!expr) return null;
  try {
    const value = compile(expr).evaluate(vars);
    if (typeof value !== 'number' || !Number.isFinite(value)) return null;
    return value;
  } catch {
    return null;
  }
}

/**
 * 批量求值（curve/vector 多条表达式时使用）。
 */
export function evalAll(
  exprs: Record<string, string>,
  vars: Record<string, number>,
): Record<string, number | null> {
  const out: Record<string, number | null> = {};
  for (const [k, e] of Object.entries(exprs)) {
    out[k] = evalExpr(e, vars);
  }
  return out;
}

/**
 * 在 [x0, x1] 区间内对 `y = f(x)` 采样 n 个点，返回 [x,y] 序列。
 * 用于 curve 预览：拖动滑块时本地重绘曲线。
 */
export function sampleCurve(
  yExpr: string,
  xVar: string,
  vars: Record<string, number>,
  x0: number,
  x1: number,
  n = 64,
): Array<[number, number]> {
  if (n < 2 || !Number.isFinite(x0) || !Number.isFinite(x1)) return [];
  const compiled = (() => {
    try {
      return compile(yExpr);
    } catch {
      return null;
    }
  })();
  if (!compiled) return [];
  const points: Array<[number, number]> = [];
  const step = (x1 - x0) / (n - 1);
  const scope = { ...vars };
  for (let i = 0; i < n; i++) {
    const x = x0 + i * step;
    scope[xVar] = x;
    try {
      const y = compiled.evaluate(scope);
      if (typeof y === 'number' && Number.isFinite(y)) points.push([x, y]);
    } catch {
      // 单点失败时跳过，不中断整条曲线
    }
  }
  return points;
}
