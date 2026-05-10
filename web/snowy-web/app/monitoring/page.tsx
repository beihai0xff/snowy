'use client';

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Badge,
  Button,
  Card,
  Col,
  Collapse,
  Descriptions,
  Empty,
  Form,
  Input,
  Progress,
  Row,
  Space,
  Spin,
  Statistic,
  Table,
  Tag,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  ApiOutlined,
  ClockCircleOutlined,
  DashboardOutlined,
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
  return <Tag color={ok ? 'green' : 'red'}>{ok ? '成功' : status || '失败'}</Tag>;
}

function operationLabel(operation: string): string {
  const map: Record<string, string> = {
    knowledge_answer_pe: '知识点直答 PE',
    render_generation_pe: '可视化渲染 PE',
    scene_spec_render: '场景渲染',
    llm_generate: '通用 LLM 调用',
  };
  return map[operation] || operation || '-';
}

function miniBar(value: number, max: number, color: string) {
  const pct = max > 0 ? Math.max(4, Math.round((value / max) * 100)) : 0;
  return (
    <div style={{ width: '100%', minWidth: 120 }}>
      <div style={{ height: 8, background: '#f0f0f0', borderRadius: 999, overflow: 'hidden' }}>
        <div style={{ width: `${pct}%`, height: '100%', background: color, borderRadius: 999 }} />
      </div>
    </div>
  );
}

function GroupMetrics({ data, title }: { data: LLMGroupMetric[]; title: string }) {
  const maxLatency = Math.max(...data.map((item) => item.avg_latency_ms), 0);
  if (!data.length) {
    return (
      <Card title={title}>
        <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无调用数据" />
      </Card>
    );
  }

  return (
    <Card title={title}>
      <Space direction="vertical" style={{ width: '100%' }} size="middle">
        {data.slice(0, 6).map((item) => (
          <div key={item.key}>
            <Space style={{ width: '100%', justifyContent: 'space-between' }}>
              <Text strong>{operationLabel(item.key)}</Text>
              <Text type="secondary">{item.total_calls} 次 · P95 {item.p95_latency_ms}ms</Text>
            </Space>
            <Space style={{ width: '100%', marginTop: 8 }} align="center">
              <div style={{ flex: 1 }}>{miniBar(item.avg_latency_ms, maxLatency, '#1677ff')}</div>
              <Text style={{ width: 88, textAlign: 'right' }}>{numberFmt(item.avg_latency_ms, 0)}ms</Text>
              <Tag color={item.success_rate >= 0.9 ? 'green' : item.success_rate >= 0.6 ? 'orange' : 'red'}>
                {percentFmt(item.success_rate)}
              </Tag>
            </Space>
          </div>
        ))}
      </Space>
    </Card>
  );
}

function PromptProfileCard({ profile }: { profile: LLMPromptProfile }) {
  return (
    <Card
      size="small"
      title={<Space><ExperimentOutlined /> {profile.title}</Space>}
      extra={<Tag color="blue">{profile.version}</Tag>}
      style={{ height: '100%' }}
    >
      <Space direction="vertical" style={{ width: '100%' }} size="small">
        <Space wrap>
          <Tag>{profile.scene}</Tag>
          <Tag color="purple">{profile.mode}</Tag>
          <Text type="secondary">更新：{timeFmt(profile.updated_at)}</Text>
        </Space>
        <Paragraph ellipsis={{ rows: 5, expandable: true, symbol: '展开 PE' }} style={{ marginBottom: 0 }}>
          {profile.system_pe}
        </Paragraph>
        <Alert type="info" showIcon message="User Prompt Contract" description={profile.user_prompt_contract} />
        <div>
          <Text strong>成功标准</Text>
          <ul style={{ margin: '8px 0 0 18px', padding: 0 }}>
            {profile.success_checklist.map((item) => <li key={item}>{item}</li>)}
          </ul>
        </div>
        {profile.generation_params && (
          <Space wrap>
            {Object.entries(profile.generation_params).map(([key, value]) => (
              <Tag key={key} color="geekblue">{key}: {String(value)}</Tag>
            ))}
          </Space>
        )}
      </Space>
    </Card>
  );
}

export default function MonitoringPage() {
  const [data, setData] = useState<LLMDashboard | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [filters, setFilters] = useState<{ user_id?: string; provider?: string; model?: string; operation?: string; limit?: number }>({ limit: 200 });

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

  useEffect(() => {
    void load();
  }, [load]);

  const recentColumns: ColumnsType<LLMCallRecord> = useMemo(() => [
    {
      title: '时间',
      dataIndex: 'finished_at',
      width: 170,
      render: (value: string) => timeFmt(value),
    },
    {
      title: '链路',
      dataIndex: 'operation',
      width: 160,
      render: (value: string) => <Tag color="blue">{operationLabel(value)}</Tag>,
    },
    {
      title: '模型',
      key: 'model',
      width: 220,
      render: (_, row) => (
        <Space direction="vertical" size={0}>
          <Text strong>{row.provider}/{row.model}</Text>
          <Text type="secondary" style={{ fontSize: 12 }}>{row.role}{row.model_provider ? ` · ${row.model_provider}` : ''}</Text>
        </Space>
      ),
    },
    {
      title: '耗时',
      dataIndex: 'latency_ms',
      width: 100,
      sorter: (a, b) => a.latency_ms - b.latency_ms,
      render: (value: number) => <Text strong>{value}ms</Text>,
    },
    {
      title: 'Tokens',
      key: 'tokens',
      width: 120,
      render: (_, row) => `${row.input_tokens}/${row.output_tokens}`,
    },
    {
      title: 'PE 字符',
      dataIndex: 'prompt_chars',
      width: 100,
      render: (value: number) => numberFmt(value),
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (value: string) => statusTag(value),
    },
    {
      title: '诊断',
      key: 'diagnostic',
      ellipsis: true,
      render: (_, row) => row.error ? <Text type="danger">{row.error}</Text> : <Text type="secondary">{row.finish_reason || 'ok'}</Text>,
    },
  ], []);

  const collapseItems = (data?.recent_calls || []).slice(0, 8).map((call) => ({
    key: call.id,
    label: (
      <Space wrap>
        {statusTag(call.status)}
        <Text strong>{operationLabel(call.operation)}</Text>
        <Text type="secondary">{call.provider}/{call.model} · {call.latency_ms}ms</Text>
      </Space>
    ),
    children: (
      <Space direction="vertical" style={{ width: '100%' }} size="middle">
        <Descriptions size="small" column={{ xs: 1, md: 3 }} bordered>
          <Descriptions.Item label="Base URL">{call.base_url || '-'}</Descriptions.Item>
          <Descriptions.Item label="温度">{call.temperature}</Descriptions.Item>
          <Descriptions.Item label="Max Tokens">{call.max_tokens}</Descriptions.Item>
          <Descriptions.Item label="输入 Tokens">{call.input_tokens}</Descriptions.Item>
          <Descriptions.Item label="输出 Tokens">{call.output_tokens}</Descriptions.Item>
          <Descriptions.Item label="Prompt 字符">{call.prompt_chars}</Descriptions.Item>
        </Descriptions>
        {call.system_pe && (
          <Card size="small" title="System PE 快照">
            <Paragraph style={{ whiteSpace: 'pre-wrap', marginBottom: 0 }}>{call.system_pe}</Paragraph>
          </Card>
        )}
        {call.user_prompt && (
          <Card size="small" title="User Prompt 快照">
            <Paragraph style={{ whiteSpace: 'pre-wrap', marginBottom: 0 }}>{call.user_prompt}</Paragraph>
          </Card>
        )}
      </Space>
    ),
  }));

  const summary = data?.summary;

  return (
    <div>
      <Space style={{ width: '100%', justifyContent: 'space-between', marginBottom: 20 }} align="start">
        <div>
          <Title level={2} style={{ marginBottom: 4 }}><DashboardOutlined /> 监控看板</Title>
          <Paragraph type="secondary" style={{ marginBottom: 0 }}>
            展示大模型 PE、调用耗时、成功率、Token 与模型配置。API Key 仅检测是否配置，不展示明文。
          </Paragraph>
        </div>
        <Button icon={<ReloadOutlined />} onClick={() => load()} loading={loading}>刷新</Button>
      </Space>

      {error && (
        <Alert
          type="error"
          showIcon
          message="监控数据加载失败"
          description={error}
          action={<Button size="small" onClick={() => load()}>重试</Button>}
          style={{ marginBottom: 16 }}
        />
      )}

      <Card style={{ marginBottom: 16 }} title="筛选器">
        <Form layout="inline" initialValues={filters} onFinish={(values) => { setFilters(values); void load(values); }}>
          <Form.Item name="user_id" label="用户"><Input allowClear placeholder="user_id" style={{ width: 220 }} /></Form.Item>
          <Form.Item name="provider" label="厂商"><Input allowClear placeholder="provider" style={{ width: 140 }} /></Form.Item>
          <Form.Item name="model" label="模型"><Input allowClear placeholder="model" style={{ width: 180 }} /></Form.Item>
          <Form.Item name="operation" label="链路"><Input allowClear placeholder="operation" style={{ width: 190 }} /></Form.Item>
          <Form.Item name="limit" label="条数"><Input type="number" style={{ width: 100 }} /></Form.Item>
          <Form.Item><Button type="primary" htmlType="submit" loading={loading}>应用筛选</Button></Form.Item>
        </Form>
      </Card>

      {loading && !data ? (
        <div style={{ textAlign: 'center', padding: 80 }}><Spin tip="正在加载监控指标..." /></div>
      ) : data ? (
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
          <Row gutter={[16, 16]}>
            <Col xs={24} sm={12} lg={6}>
              <Card>
                <Statistic title="LLM 调用总数" value={summary?.total_calls || 0} prefix={<ThunderboltOutlined />} />
                <Text type="secondary">最近 {data.recent_calls.length} 条内存记录</Text>
              </Card>
            </Col>
            <Col xs={24} sm={12} lg={6}>
              <Card>
                <Statistic title="平均耗时" value={summary?.avg_latency_ms || 0} precision={0} suffix="ms" prefix={<ClockCircleOutlined />} />
                <Text type="secondary">P50 {summary?.p50_latency_ms || 0}ms · P95 {summary?.p95_latency_ms || 0}ms</Text>
              </Card>
            </Col>
            <Col xs={24} sm={12} lg={6}>
              <Card>
                <Statistic title="成功率" value={(summary?.success_rate || 0) * 100} precision={0} suffix="%" />
                <Progress percent={Math.round((summary?.success_rate || 0) * 100)} size="small" status={(summary?.success_rate || 0) >= 0.9 ? 'success' : 'active'} />
              </Card>
            </Col>
            <Col xs={24} sm={12} lg={6}>
              <Card>
                <Statistic title="Token 总量" value={(summary?.total_input_tokens || 0) + (summary?.total_output_tokens || 0)} prefix={<ApiOutlined />} />
                <Text type="secondary">输入 {numberFmt(summary?.total_input_tokens)} / 输出 {numberFmt(summary?.total_output_tokens)}</Text>
              </Card>
            </Col>
          </Row>

          {summary?.last_error && <Alert type="warning" showIcon message="最近错误" description={summary.last_error} />}

          <Card title="模型配置（已脱敏）">
            <Row gutter={[16, 16]}>
              {data.providers.map((provider) => (
                <Col xs={24} md={12} key={provider.role}>
                  <Card size="small" bordered={false} style={{ background: '#fafafa' }}>
                    <Space direction="vertical" style={{ width: '100%' }}>
                      <Space style={{ justifyContent: 'space-between', width: '100%' }}>
                        <Space>
                          <Badge status={provider.configured ? 'success' : 'warning'} />
                          <Text strong>{provider.role.toUpperCase()} · {provider.provider || '未配置'}</Text>
                        </Space>
                        <Tag color={provider.api_key_configured ? 'green' : 'red'}>
                          {provider.api_key_configured ? 'API Key 已注入' : 'API Key 缺失'}
                        </Tag>
                      </Space>
                      <Descriptions size="small" column={1}>
                        <Descriptions.Item label="模型">{provider.model || '-'}</Descriptions.Item>
                        <Descriptions.Item label="model_provider">{provider.model_provider || '-'}</Descriptions.Item>
                        <Descriptions.Item label="base_url">{provider.base_url || '-'}</Descriptions.Item>
                        <Descriptions.Item label="超时 / 重试">{provider.timeout || '-'} / {provider.max_retries}</Descriptions.Item>
                      </Descriptions>
                    </Space>
                  </Card>
                </Col>
              ))}
            </Row>
          </Card>

          <Row gutter={[16, 16]}>
            <Col xs={24} lg={12}>
              <GroupMetrics title="按模型聚合" data={data.by_provider} />
            </Col>
            <Col xs={24} lg={12}>
              <GroupMetrics title="按 PE/链路聚合" data={data.by_operation} />
            </Col>
          </Row>

          <Card title="大模型 PE 注册表">
            <Row gutter={[16, 16]}>
              {data.prompt_profiles.map((profile) => (
                <Col xs={24} lg={8} key={profile.id}>
                  <PromptProfileCard profile={profile} />
                </Col>
              ))}
            </Row>
          </Card>

          <Card title="最近 LLM 调用（MySQL 持久化 + 内存热数据）">
            {data.recent_calls.length ? (
              <Table
                rowKey="id"
                columns={recentColumns}
                dataSource={data.recent_calls}
                size="middle"
                scroll={{ x: 1100 }}
                pagination={{ pageSize: 8 }}
              />
            ) : (
              <Empty description="暂无真实 LLM 调用记录。可以先执行一次知识检索或生物演示生成，再刷新看板。" />
            )}
          </Card>

          <Card title="PE 快照与 Prompt 明细">
            {collapseItems.length ? <Collapse items={collapseItems} /> : <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无 Prompt 快照" />}
          </Card>
        </Space>
      ) : (
        <Empty description="暂无监控数据" />
      )}
    </div>
  );
}
