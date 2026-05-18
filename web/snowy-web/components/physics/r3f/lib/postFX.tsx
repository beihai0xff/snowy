/**
 * Snowy v8 · R3F · 后处理升级版
 *
 * 三档：
 *  - eco      : 不挂 EffectComposer
 *  - standard : Bloom + Vignette
 *  - high     : Bloom + Vignette + DoF + SSAO
 *
 * Tone mapping 走 Canvas 的 gl.toneMapping (在 R3FPhysicsPreview.onCreated)。
 */

'use client';

import React from 'react';
import { EffectComposer, Bloom, Vignette, DepthOfField } from '@react-three/postprocessing';

export type QualityLevel = 'eco' | 'standard' | 'high';

interface Props {
  quality: QualityLevel;
}

export default function PostFX({ quality }: Props) {
  if (quality === 'eco') return null;
  const high = quality === 'high';
  return (
    <EffectComposer multisampling={high ? 4 : 0}>
      <Bloom
        intensity={high ? 0.95 : 0.7}
        luminanceThreshold={0.6}
        luminanceSmoothing={0.22}
        mipmapBlur
      />
      <Vignette
        eskil={false}
        offset={0.32}
        darkness={0.55}
      />
      {high ? (
        <DepthOfField focusDistance={0.02} focalLength={0.05} bokehScale={2.0} />
      ) : (
        <></>
      )}
    </EffectComposer>
  );
}



