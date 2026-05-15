'use client';

import React from 'react';
import { Alert, Card, List, Space, Tag, Typography } from 'antd';
import type { ModelValidationReport } from '@/lib/api';

const { Text } = Typography;

export default function ValidationReportPanel({ report }: { report?: ModelValidationReport }) {
  if (!report) return null;
  const ok = report.schema_valid && report.domain_valid && report.safety_valid && !report.fallback_required;
  return (
    <Card size="small" title="校验报告" extra={<Tag color={ok ? 'green' : 'orange'}>{ok ? '通过' : '需注意'}</Tag>}>
      <Space wrap style={{ marginBottom: 8 }}>
        <Tag color={report.schema_valid ? 'green' : 'red'}>Schema</Tag>
        <Tag color={report.evidence_valid ? 'green' : 'gold'}>Evidence</Tag>
        <Tag color={report.domain_valid ? 'green' : 'red'}>Domain</Tag>
        <Tag color={report.safety_valid ? 'green' : 'red'}>Safety</Tag>
      </Space>
      {report.fallback_required && <Alert type="warning" showIcon message="校验未通过，当前结果不可作为成功生成结果" description={report.fallback_reason} style={{ marginBottom: 8 }} />}
      <List
        size="small"
        dataSource={report.checks || []}
        renderItem={(item) => <List.Item><Text type={item.status === 'fail' ? 'danger' : item.status === 'warn' ? 'warning' : 'secondary'}>{item.name}：{item.status}{item.message ? ` - ${item.message}` : ''}</Text></List.Item>}
      />
    </Card>
  );
}
