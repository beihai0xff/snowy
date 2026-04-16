'use client';

import React, { useMemo } from 'react';
import {
  ReactFlow,
  Background,
  Controls,
  type Node,
  type Edge,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import type { DiagramSpec } from '@/lib/api';

interface BiologyDiagramProps {
  spec: DiagramSpec;
}

const nodeColorMap: Record<string, string> = {
  factor: '#1677ff',
  result: '#52c41a',
  process: '#fa8c16',
  structure: '#722ed1',
};

/**
 * BiologyDiagram 将 DiagramSpec 渲染为 React Flow 图。
 * 重要：edges.source/target 引用的是 DiagramNode.id（如 n1, n2），
 * 不是 label 文本值。参考技术方案 §16.4。
 */
export default function BiologyDiagram({ spec }: BiologyDiagramProps) {
  const { nodes, edges } = useMemo(() => {
    const flowNodes: Node[] = spec.nodes.map((n, i) => ({
      id: n.id,
      data: { label: n.label },
      position: { x: 150 * (i % 3), y: 120 * Math.floor(i / 3) },
      style: {
        background: nodeColorMap[n.type] || '#1677ff',
        color: '#fff',
        borderRadius: 8,
        padding: '8px 16px',
        fontSize: 14,
        fontWeight: 500,
        border: 'none',
      },
    }));

    const flowEdges: Edge[] = spec.edges.map((e, i) => ({
      id: `e-${i}`,
      source: e.source,
      target: e.target,
      label: e.label,
      animated: true,
      style: { stroke: '#999' },
    }));

    return { nodes: flowNodes, edges: flowEdges };
  }, [spec]);

  return (
    <div style={{ height: 400 }}>
      <ReactFlow
        nodes={nodes}
        edges={edges}
        fitView
        proOptions={{ hideAttribution: true }}
      >
        <Background />
        <Controls />
      </ReactFlow>
    </div>
  );
}
