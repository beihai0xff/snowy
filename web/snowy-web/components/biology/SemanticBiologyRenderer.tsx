/**
 * Snowy v6 · 语义化生物渲染器
 *
 * 输入：visualization_type / topic 字符串；
 * 输出：匹配则路由到对应 SVG 插画，未匹配返回 null 让外层 fallback 到 ReactFlow。
 */

'use client';

import React from 'react';
import dynamic from 'next/dynamic';

const Photosynthesis = dynamic(() => import('./illustrations/Photosynthesis'), { ssr: false });
const Respiration = dynamic(() => import('./illustrations/Respiration'), { ssr: false });
const Enzyme = dynamic(() => import('./illustrations/Enzyme'), { ssr: false });
const Synapse = dynamic(() => import('./illustrations/Synapse'), { ssr: false });
const Genetics = dynamic(() => import('./illustrations/Genetics'), { ssr: false });
const Ecosystem = dynamic(() => import('./illustrations/Ecosystem'), { ssr: false });
const Membrane = dynamic(() => import('./illustrations/Membrane'), { ssr: false });

interface Props {
  topic?: string;
  visualizationType?: string;
  values?: Record<string, number>;
}

const RULES: { match: RegExp; render: (p: Required<Pick<Props, 'topic' | 'values'>>) => React.ReactElement }[] = [
  { match: /(光合|photosynth)/i, render: ({ topic, values }) => <Photosynthesis topic={topic} values={values} /> },
  { match: /(呼吸|respiration|有氧|线粒体)/i, render: ({ topic, values }) => <Respiration topic={topic} values={values} /> },
  { match: /(酶|enzyme|催化)/i, render: ({ topic, values }) => <Enzyme topic={topic} values={values} /> },
  { match: /(突触|神经递质|synap)/i, render: ({ topic, values }) => <Synapse topic={topic} values={values} /> },
  { match: /(遗传|孟德尔|基因型|punnett|genetic)/i, render: ({ topic, values }) => <Genetics topic={topic} values={values} /> },
  { match: /(生态|食物链|能量流|ecosystem|食物网|金字塔)/i, render: ({ topic, values }) => <Ecosystem topic={topic} values={values} /> },
  { match: /(细胞膜|磷脂|物质运输|membrane|扩散|主动运输)/i, render: ({ topic, values }) => <Membrane topic={topic} values={values} /> },
];

export default function SemanticBiologyRenderer({ topic, visualizationType, values }: Props): React.ReactElement | null {
  const text = `${topic || ''} ${visualizationType || ''}`;
  for (const rule of RULES) {
    if (rule.match.test(text)) {
      return rule.render({ topic: topic || '', values: values || {} });
    }
  }
  return null;
}

export function hasSemanticBiologyIllustration(topic?: string, visualizationType?: string): boolean {
  const text = `${topic || ''} ${visualizationType || ''}`;
  return RULES.some((r) => r.match.test(text));
}
