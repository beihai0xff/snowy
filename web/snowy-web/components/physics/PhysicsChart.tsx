'use client';

import React from 'react';
import ReactEChartsCore from 'echarts-for-react/lib/core';
import * as echarts from 'echarts/core';
import { LineChart, ScatterChart, BarChart } from 'echarts/charts';
import {
  GridComponent,
  TooltipComponent,
  TitleComponent,
  LegendComponent,
} from 'echarts/components';
import { CanvasRenderer } from 'echarts/renderers';
import type { ChartSpec } from '@/lib/api';

echarts.use([
  LineChart,
  ScatterChart,
  BarChart,
  GridComponent,
  TooltipComponent,
  TitleComponent,
  LegendComponent,
  CanvasRenderer,
]);

interface PhysicsChartProps {
  spec: ChartSpec;
}

export default function PhysicsChart({ spec }: PhysicsChartProps) {
  const chartTypeMap: Record<string, string> = {
    line: 'line',
    scatter: 'scatter',
    bar: 'bar',
  };

  const option = {
    tooltip: { trigger: 'axis' as const },
    legend: {
      data: spec.series.map((s) => s.name),
    },
    xAxis: {
      type: 'value' as const,
      name: `${spec.x_axis.label} (${spec.x_axis.unit})`,
    },
    yAxis: {
      type: 'value' as const,
      name: `${spec.y_axis.label} (${spec.y_axis.unit})`,
    },
    series: spec.series.map((s) => ({
      name: s.name,
      type: chartTypeMap[spec.chart_type] || 'line',
      data: s.data,
      smooth: spec.chart_type === 'line',
    })),
  };

  return (
    <ReactEChartsCore
      echarts={echarts}
      option={option}
      style={{ height: 350 }}
      notMerge
    />
  );
}
