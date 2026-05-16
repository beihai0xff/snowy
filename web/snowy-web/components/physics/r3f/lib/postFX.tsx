/**
 * Snowy v6 · R3F · 后处理 (Bloom + 可选 DoF)
 * 通过 props 控制画质档位与 prefers-reduced-motion 关闭。
 */

'use client';

import React from 'react';
import { EffectComposer, Bloom, DepthOfField } from '@react-three/postprocessing';

export type QualityLevel = 'eco' | 'standard' | 'high';

interface Props {
  quality: QualityLevel;
}

export default function PostFX({ quality }: Props) {
  if (quality === 'eco') return null;
  return (
    <EffectComposer multisampling={quality === 'high' ? 4 : 0}>
      <Bloom
        intensity={quality === 'high' ? 0.9 : 0.55}
        luminanceThreshold={0.4}
        luminanceSmoothing={0.4}
        mipmapBlur
      />
      {quality === 'high' ? (
        <DepthOfField focusDistance={0.02} focalLength={0.045} bokehScale={2.0} />
      ) : (
        <></>
      )}
    </EffectComposer>
  );
}
