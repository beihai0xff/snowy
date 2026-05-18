/**
 * Snowy v8 · 设备分层 hook
 *
 * 综合 hardwareConcurrency / deviceMemory / NetworkInformation / prefers-reduced-* 输出三档：
 *   - 'low'  : 低端机或弱网或显式偏好降级 → 关后处理、关粒子、关 HDRI
 *   - 'mid'  : 大多数中端设备 → 标准 PBR + Bloom
 *   - 'high' : 高端 → 全开 SSAO/DoF/SoftShadows
 *
 * SSR 环境下默认 'mid'，挂载后再检测，避免 hydration mismatch。
 */

'use client';

import { useState } from 'react';

export type DeviceTier = 'low' | 'mid' | 'high';

interface NavWithMemory extends Navigator {
  deviceMemory?: number;
  connection?: { effectiveType?: string; saveData?: boolean };
}

function detect(): DeviceTier {
  if (typeof window === 'undefined' || typeof navigator === 'undefined') return 'mid';
  const nav = navigator as NavWithMemory;
  const cores = nav.hardwareConcurrency || 4;
  const mem = nav.deviceMemory || 4;
  const net = nav.connection?.effectiveType || '4g';
  const saveData = nav.connection?.saveData === true;
  const reducedMotion = window.matchMedia?.('(prefers-reduced-motion: reduce)').matches;
  const reducedData = window.matchMedia?.('(prefers-reduced-data: reduce)').matches || saveData;

  // low
  if (cores <= 2) return 'low';
  if (mem <= 2) return 'low';
  if (net === '2g' || net === 'slow-2g' || net === '3g') return 'low';
  if (reducedMotion || reducedData) return 'low';

  // high
  if (cores >= 8 && mem >= 8 && net === '4g') return 'high';

  return 'mid';
}

export function useDeviceTier(): DeviceTier {
  const [tier] = useState<DeviceTier>(() => detect());
  return tier;
}

/** 同步版本（仅在 client 阶段才准确）—— 用于 R3F 初始化期，避免 useEffect 延迟 */
export function detectDeviceTierSync(): DeviceTier {
  return detect();
}

