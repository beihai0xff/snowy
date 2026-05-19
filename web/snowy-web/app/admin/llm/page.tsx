'use client';

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Badge,
  Button,
  Collapse,
  Descriptions,
  Empty,
  Form,
  Input,
  Progress,
  Space,
  Table,
  Tabs,
  Tag,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  ApiOutlined,
  ClockCircleOutlined,
  ExperimentOutlined,
  ReloadOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons';
import { api, type LLMCallRecord, type LLMDashboard, type LLMGroupMetric, type LLMPromptProfile } from '@/lib/api';

const { Title, Paragraph, Text } = Typography;

function numberFmt(value?: number, digits = 0): string {
  if (typeof value !== 'number' || Number.isNaN(value)) return '-';
  return value.toLocaleString(undefined, { maximumFractionDigits: digits, minimumFractionDigits: digits });
}

function percentFmt(value?: number): string {
  if (typeof value !== 'number' || Number.isNaN(value)) return '-';
  return `${Math.round(value * 100)}%`;
}

function timeFmt(value?: string): string {
  if (!value) return '-';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '-';
  return date.toLocaleString();
}

function statusTag(status: string) {
  const ok = status === 'success';
  return <Tag color={ok ? 'green' : 'red'} bordered={false}>{ok ? '成功' : status || '失败'}</Tag>;
}

function operationLabel(operation: string): string {
  const map: Record<string, string> = {
    knowledge_answer_pe:  '知识点直答 PE',
    render_generation_pe: '可视化渲染 PE',
    scene_spec_render:    '场景渲染',
    llm_generate:         '通用 LLM 调用',
  };
  return map[operation] || operation || '-';
}

function GroupMetricsList({ data, title }: { data: LLMGroupMetric[]; title: string }) {
  const maxLatency = Math.max(...data.map((item) => item.avg_latency_ms), 0);
  if (!data.length) {
    return (
      <div style={{ padding: 16, background: 'var(--color-bg-card)', border: '1px solid var(--color-border)', borderRadius: 12 }}>
        <div style={{ fontSize: 14, fontWeight: 600, marginBottom: 8 }}>{title}</div>
        <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无调用数据" />
      </div>
    );
  }
  return (
    <div style={{ padding: 16, background: 'var(--color-bg-card)', border: '1px solid var(--color-border)', borderRadius: 12 }}>
      <div style={{ fontSize: 14, fontWeight: 600, marginBottom: 12 }}>{title}</div>
      <Space direction="vertical" style={{ width: '100%' }} size={12}>
        {data.slice(0, 6).map((item) => {
          const pct = maxLatency > 0 ? Math.max(4, Math.round((item.avg_latency_ms / maxLatency) * 100)) : 0;
          return (
            <div key={item.key}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', marginBottom: 4 }}>
                <Text strong style={{ fontSize: 13 }}>{operationLabel(item.key)}</Text>
                <Text type="secondary" style={{ fontSize: 12 }}>{item.total_calls} 次 · P95 {item.p95_latency_ms}ms</Text>
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <div style={{ flex: 1, height: 8, background: 'var(--color-bg-subtle)', borderRadius: 999, overflow: 'hidden' }}>
                  <div style={{ width: `${pct}%`, height: '100%', background: 'var(--color-primary)', borderRadius: 999 }} />
                </div>
                <Text style={{ width: 80, textAlign: 'right', fontSize: 12 }}>{numberFmt(item.avg_latency_ms, 0)}ms</Text>
                <Tag color={item.success_rate >= 0.9 ? 'green' : item.success_rate >= 0.6 ? 'orange' : 'red'} bordered={false}>
                  {percentFmt(item.success_rate)}
                </Tag>
              </div>
            </div>
          );
        })}
      </Space>
    </div>
  );
}

function PromptProfileCard({ profile }: { profile: LLMPromptProfile }) {
  return (
    <div style={{ padding: 16, background: 'var(--color-bg-card)', border: '1px solid var(--color-border)', borderRadius: 12, height: '100%' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 12 }}>
        <div>
          <div style={{ fontSize: 14, fontWeight: 600 }}><ExperimentOutlined /> {profile.title}</div>
          <Space size={4} style={{ marginTop: 4 }}>
            <Tag bordered={false}>{profile.scene}</Tag>
            <Tag bordered={false} color="purple">{profile.mode}</Tag>
          </Space>
        </div>
        <Tag bordered={false} color="blue">{profile.version}</Tag>
      </div>
      <Paragraph type="secondary" ellipsis={{ rows: 4, expandable: true, symbol: '展开 PE' }} style={{ fontSize: 13, marginBottom: 12 }}>
        {profile.system_pe}
      </Paragraph>
      <Alert type="info" showIcon style={{ marginBottom: 12, borderRadius: 8 }} message="User Prompt Contract" description={<Text style={{ fontSize: 12 }}>{profile.user_prompt_contract}</Text>} />
      <div style={{ fontSize: 13, fontWeight: 500, marginBottom: 4 }}>成功标准</div>
      <ul style={{ margin: 0, paddingLeft: 18, fontSize: 12, color: 'var(--color-text-muted)' }}>
        {profile.success_checklist.map((item) => <li key={item}>{item}</li>)}
      </ul>
      {profile.generation_params && (
        <Space wrap size={4} style={{ marginTop: 12 }}>
          {Object.entries(profile.generation_params).map(([key, value]) => (
            <Tag key={key} bordered={false} color="geekblue">{key}: {String(value)}</Tag>
          ))}
        </Space>
      )}
      <Text type="secondary" style={{ fontSize: 12, display: 'block', marginTop: 8 }}>更新：{timeFmt(profile.updated_at)}</Text>
    </div>
  );
}

export default function AdminLLMPage() {
  const [data, setData] = useState<LLMDashboard | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [filters, setFilters] = useState<{ user_id?: string; provider?: string; model?: string; operation?: string; limit?: number }>({ limit: 200 });
  const [activeTab, setActiveTab] = useState('calls');

  const load = useCallback(async (nextFilters = filters) => {
    setLoading(true);
    setError(null);
    try {
      const resp = await api.getLLMMonitoring(nextFilters);
      setData(resp.data || null);
    } catch (err) {
      setError(err instanceof Error ? err.message : '监控数据加载失败');
    } finally {
      setLoading(false);
    }
  }, [filters]);

  useEffect(() => { void load(); }, [load]);

  const recentColumns: ColumnsType<LLMCallRecord> = useMemo(() => [
    { title: '时间',    dataIndex: 'finished_at', width: 160, render: (v: string) => <Text style={{ fontSize: 12 }}>{timeFmt(v)}</Text> },
    { title: '链路',    dataIndex: 'operation',   width: 150, render: (v: string) => <Tag color="blue" bordered={false}>{operationLabel(v)}</Tag> },
    {
      title: '模型', key: 'model', width: 200,
      render: (_, row) => (
        <Space direction="vertical" size={0}>
          <Text strong style={{ fontSize: 13 }}>{row.provider}/{row.model}</Text>
          <Text type="secondary" style={{ fontSize: 11 }}>{row.role}{row.model_provider ? ` · ${row.model_provider}` : ''}</Text>
        </Space>
      ),
    },
    { title: '耗时',    dataIndex: 'latency_ms',  width: 90,  sorter: (a, b) => a.latency_ms - b.latency_ms, render: (v: number) => <Text strong style={{ fontSize: 13 }}>{v}ms</Text> },
    { title: 'Tokens',  key: 'tokens',            width: 100, render: (_, row) => <Text style={{ fontSize: 13 }}>{row.input_tokens}/{row.output_tokens}</Text> },
    { title: 'PE 字符', dataIndex: 'prompt_chars', width: 90, render: (v: number) => <Text style={{ fontSize: 13 }}>{numberFmt(v)}</Text> },
    { title: '状态',    dataIndex: 'status',      width: 80,  render: (v: string) => statusTag(v) },
    { title: '诊断',    key: 'diagnostic',        ellipsis: true, render: (_, row) => row.error ? <Text type="danger" style={{ fontSize: 12 }}>{row.error}</Text> : <Text type="secondary" style={{ fontSize: 12 }}>{row.finish_reason || 'ok'}</Text> },
  ], []);

  const collapseItems = (data?.recent_calls || []).slice(0, 8).map((call) => ({
    key: call.id,
    label: (
      <Space wrap>
        {statusTag(call.status)}
        <Text strong>{operationLabel(call.operation)}</Text>
        <Text type="secondary" style={{ fontSize: 12 }}>{call.provider}/{call.model} · {call.latency_ms}ms</Text>
      </Space>
    ),
    children: (
      <Space direction="vertical" style={{ width: '100%' }} size={12}>
        <Descriptions size="small" column={{ xs: 1, md: 3 }} bordered>
          <Descriptions.Item label="Base URL">{call.base_url || '-'}</Descriptions.Item>
          <Descriptions.Item label="温度">{call.temperature}</Descriptions.Item>
          <Descriptions.Item label="Max Tokens">{call.max_tokens}</Descriptions.Item>
          <Descriptions.Item label="输入 Tokens">{call.input_tokens}</Descriptions.Item>
          <Descriptions.Item label="输出 Tokens">{call.output_tokens}</Descriptions.Item>
          <Descriptions.Item label="Prompt 字符">{call.prompt_chars}</Descriptions.Item>
        </Descriptions>
        {call.system_pe && (
          <div style={{ padding: 12, background: 'var(--color-bg-subtle)', borderRadius: 8 }}>
            <Text strong style={{ fontSize: 12, display: 'block', marginBottom: 6 }}>System PE 快照</Text>
            <Paragraph style={{ whiteSpace: 'pre-wrap', marginBottom: 0, fontSize: 12, fontFamily: 'var(--font-mono)' }}>{call.system_pe}</Paragraph>
          </div>
        )}
        {call.user_prompt && (
          <div style={{ padding: 12, background: 'var(--color-bg-subtle)', borderRadius: 8 }}>
            <Text strong style={{ fontSize: 12, display: 'block', marginBottom: 6 }}>User Prompt 快照</Text>
            <Paragraph style={{ whiteSpace: 'pre-wrap', marginBottom: 0, fontSize: 12, fontFamily: 'var(--font-mono)' }}>{call.user_prompt}</Paragraph>
          </div>
        )}
      </Space>
    ),
  }));

  const summary = data?.summary;
  const tabItems = [
    {
      key: 'calls',
      label: '调用记录',
      children: (
        <Space direction="vertical" size={16} style={{ width: '100%' }}>
          <div>
            <div style={{ fontSize: 14, fontWeight: 600, marginBottom: 12 }}>最近 LLM 调用</div>
            {data?.recent_calls?.length ? (
              <Table
                rowKey="id"
                columns={recentColumns}
                dataSource={data.recent_calls}
                size="middle"
                scroll={{ x: 1100 }}
                pagination={{ pageSize: 8, hideOnSinglePage: true }}
              />
            ) : (
              <Empty description="暂无 LLM 调用记录" />
            )}
          </div>

          <div>
            <div style={{ fontSize: 14, fontWeight: 600, marginBottom: 12 }}>PE 快照与 Prompt 明细</div>
            {collapseItems.length ? <Collapse items={collapseItems} /> : <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无 Prompt 快照" />}
          </div>

          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, minmax(0, 1fr))', gap: 16 }}>
            <GroupMetricsList title="按模型聚合" data={data?.by_provider || []} />
            <GroupMetricsList title="按 PE/链路聚合" data={data?.by_operation || []} />
          </div>
        </Space>
      ),
    },
    {
      key: 'prompts',
      label: 'Prompt 注册表',
      children: data?.prompt_profiles?.length ? (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(360px, 1fr))', gap: 16 }}>
          {data.prompt_profiles.map((p) => <PromptProfileCard key={p.id} profile={p} />)}
        </div>
      ) : <Empty description="暂无 Prompt Profile" />,
    },
    {
      key: 'providers',
      label: '模型配置',
      children: data?.providers?.length ? (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(380px, 1fr))', gap: 16 }}>
          {data.providers.map((provider) => (
            <div key={provider.role} style={{ padding: 16, background: 'var(--color-bg-card)', border: '1px solid var(--color-border)', borderRadius: 12 }}>
              <Space style={{ justifyContent: 'space-between', width: '100%', marginBottom: 12 }}>
                <Space>
                  <Badge status={provider.configured ? 'success' : 'warning'} />
                  <Text strong>{provider.role.toUpperCase()} · {provider.provider || '未配置'}</Text>
                </Space>
                <Tag color={provider.api_key_configured ? 'green' : 'red'} bordered={false}>
                  {provider.api_key_configured ? 'API Key 已注入' : 'API Key 缺失'}
                </Tag>
              </Space>
              <Descriptions size="small" column={1}>
                <Descriptions.Item label="模型">{provider.model || '-'}</Descriptions.Item>
                <Descriptions.Item label="model_provider">{provider.model_provider || '-'}</Descriptions.Item>
                <Descriptions.Item label="base_url">{provider.base_url || '-'}</Descriptions.Item>
                <Descriptions.Item label="超时 / 重试">{provider.timeout || '-'} / {provider.max_retries}</Descriptions.Item>
              </Descriptions>
            </div>
          ))}
        </div>
      ) : <Empty description="暂无模型配置" />,
    },
  ];
  const activeContent = tabItems.find((item) => item.key === activeTab)?.children;

  return (
    <div className="snowy-page">
      <div className="snowy-page-heading" style={{ display: 'flex', alignItems: 'flex-end', justifyContent: 'space-between', marginBottom: 24 }}>
        <div>
          <Title level={1} style={{ margin: '0 0 8px', fontSize: 28, fontWeight: 700 }}>LLM 调用监控</Title>
          <Paragraph style={{ marginBottom: 0, color: 'var(--color-text-muted)' }}>
            观察大模型 PE、调用耗时、成功率、Token、模型配置；用于运维与 Prompt 调优，<strong>非高中生学习入口</strong>。
          </Paragraph>
        </div>
        <Button icon={<ReloadOutlined />} onClick={() => load()} loading={loading}>刷新</Button>
      </div>

      <div style={{ background: 'var(--color-bg-card)', border: '1px solid var(--color-border)', borderRadius: 12, padding: '4px 16px', marginBottom: 16 }}>
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          items={tabItems.map(({ key, label }) => ({ key, label }))}
          tabBarGutter={24}
        />
      </div>

      {error && (
        <Alert
          type="error" showIcon
          message="监控数据加载失败"
          description={error}
          action={<Button size="small" onClick={() => load()}>重试</Button>}
          style={{ marginBottom: 16, borderRadius: 12 }}
        />
      )}

      {/* KPI 行 */}
      <div className="snowy-kpi-row">
        <div className="snowy-kpi">
          <div className="snowy-kpi__label"><ThunderboltOutlined /> 调用总数</div>
          <div className="snowy-kpi__value">{numberFmt(summary?.total_calls)}</div>
          <div className="snowy-kpi__sub">最近 {data?.recent_calls.length || 0} 条热数据</div>
        </div>
        <div className="snowy-kpi">
          <div className="snowy-kpi__label"><ClockCircleOutlined /> 平均耗时</div>
          <div className="snowy-kpi__value">{numberFmt(summary?.avg_latency_ms)}<span style={{ fontSize: 16, fontWeight: 500, marginLeft: 4 }}>ms</span></div>
          <div className="snowy-kpi__sub">P50 {summary?.p50_latency_ms || 0} · P95 {summary?.p95_latency_ms || 0}</div>
        </div>
        <div className="snowy-kpi">
          <div className="snowy-kpi__label">成功率</div>
          <div className="snowy-kpi__value" style={{ color: (summary?.success_rate || 0) >= 0.9 ? 'var(--color-success)' : 'var(--color-warning)' }}>
            {Math.round((summary?.success_rate || 0) * 100)}<span style={{ fontSize: 16, fontWeight: 500, marginLeft: 4 }}>%</span>
          </div>
          <Progress percent={Math.round((summary?.success_rate || 0) * 100)} size="small" showInfo={false} status={(summary?.success_rate || 0) >= 0.9 ? 'success' : 'active'} style={{ marginTop: 8 }} />
        </div>
        <div className="snowy-kpi">
          <div className="snowy-kpi__label"><ApiOutlined /> Token 总量</div>
          <div className="snowy-kpi__value">{numberFmt((summary?.total_input_tokens || 0) + (summary?.total_output_tokens || 0))}</div>
          <div className="snowy-kpi__sub">输入 {numberFmt(summary?.total_input_tokens)} / 输出 {numberFmt(summary?.total_output_tokens)}</div>
        </div>
      </div>

      {summary?.last_error && <Alert type="warning" showIcon message="最近错误" description={summary.last_error} style={{ marginBottom: 16, borderRadius: 12 }} />}

      {/* 筛选器 */}
      <div style={{ padding: 16, background: 'var(--color-bg-card)', border: '1px solid var(--color-border)', borderRadius: 12, marginBottom: 16 }}>
        <Form layout="inline" initialValues={filters} onFinish={(values) => { setFilters(values); void load(values); }}>
          <Form.Item name="user_id"   label="用户"><Input allowClear placeholder="user_id" style={{ width: 200 }} /></Form.Item>
          <Form.Item name="provider"  label="厂商"><Input allowClear placeholder="provider" style={{ width: 130 }} /></Form.Item>
          <Form.Item name="model"     label="模型"><Input allowClear placeholder="model" style={{ width: 170 }} /></Form.Item>
          <Form.Item name="operation" label="链路"><Input allowClear placeholder="operation" style={{ width: 180 }} /></Form.Item>
          <Form.Item name="limit"     label="条数"><Input type="number" style={{ width: 90 }} /></Form.Item>
          <Form.Item><Button type="primary" htmlType="submit" loading={loading}>应用筛选</Button></Form.Item>
        </Form>
      </div>

      {/* Tabs：调用记录 / Prompt / 模型配置 */}
      <div style={{ background: 'var(--color-bg-card)', border: '1px solid var(--color-border)', borderRadius: 12, padding: 16 }}>
        {activeContent}
      </div>
    </div>
  );
}
