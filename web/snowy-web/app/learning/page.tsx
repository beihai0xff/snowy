'use client';

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import {
  Alert,
  Button,
  Empty,
  Form,
  Input,
  Segmented,
  Space,
  Table,
  Tabs,
  Tag,
  Typography,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  BookOutlined,
  BranchesOutlined,
  CheckCircleOutlined,
  DislikeOutlined,
  ExperimentOutlined,
  HistoryOutlined,
  LikeOutlined,
  LoginOutlined,
  LogoutOutlined,
  ReloadOutlined,
  SearchOutlined,
  StarOutlined,
} from '@ant-design/icons';
import {
  api,
  clearAuthTokens,
  setAuthTokens,
  type AnswerRecord,
  type Favorite,
  type GenerativeModelPackage,
  type HistoryItem,
  type Reaction,
  type User,
} from '@/lib/api';

const { Title, Text, Paragraph } = Typography;

function formatDateShort(value?: string): string {
  if (!value) return '-';
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return '-';
  return d.toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', hour12: false });
}

function actionLabel(type: string): string {
  if (type.includes('physics'))   return '物理';
  if (type.includes('biology'))   return '生物';
  if (type.includes('search') || type === 'search') return '提问';
  if (type.includes('modeling'))  return '推演';
  return type || '记录';
}

function actionColor(type: string): string {
  if (type.includes('physics')) return 'cyan';
  if (type.includes('biology')) return 'green';
  if (type.includes('search'))  return 'blue';
  return 'default';
}

function targetLabel(type: string): string {
  const map: Record<string, string> = {
    search: '提问', answer: '答案', evidence: '证据',
    physics: '物理推演', biology: '生物图谱',
    model_package: '模型包', render_code: '渲染代码', model_config: '模型配置',
  };
  return map[type] || type;
}

function targetColor(type: string): string {
  if (type === 'physics' || type === 'model_package') return 'cyan';
  if (type === 'biology') return 'green';
  if (type === 'answer' || type === 'search') return 'blue';
  if (type === 'evidence') return 'gold';
  return 'default';
}

function packageTitle(pkg: GenerativeModelPackage): string {
  return pkg.learning_model?.learning_goal || pkg.generative_model?.learning_goal || pkg.question || pkg.package_id;
}

function avatarOf(profile: User | null): string {
  if (profile?.nickname) return profile.nickname.charAt(0).toUpperCase();
  if (profile?.email) return profile.email.charAt(0).toUpperCase();
  return '?';
}

function displayName(profile: User | null): string {
  if (profile?.nickname) return profile.nickname;
  if (profile?.email)    return profile.email;
  return '访客';
}

export default function LearningPage() {
  const router = useRouter();
  const [authed, setAuthed] = useState<boolean | null>(null);
  const [profile, setProfile] = useState<User | null>(null);
  const [history,   setHistory]   = useState<HistoryItem[]>([]);
  const [favorites, setFavorites] = useState<Favorite[]>([]);
  const [answers,   setAnswers]   = useState<AnswerRecord[]>([]);
  const [reactions, setReactions] = useState<Reaction[]>([]);
  const [packages,  setPackages]  = useState<GenerativeModelPackage[]>([]);
  const [loading, setLoading] = useState(true);
  const [errorText, setErrorText] = useState<string | null>(null);
  const [authMode, setAuthMode] = useState<'login' | 'register'>('login');
  const [authLoading, setAuthLoading] = useState(false);

  const loadData = useCallback(async () => {
    setLoading(true);
    setErrorText(null);
    try {
      const profileRes = await api.getProfile().catch(() => null);
      if (!profileRes?.data) {
        setAuthed(false);
        setProfile(null);
        setLoading(false);
        return;
      }
      setProfile(profileRes.data);
      setAuthed(true);
      const [historyRes, favRes, answerRes, reactionRes, packageRes] = await Promise.all([
        api.getHistory().catch(() => ({ data: { total: 0, page: 1, page_size: 20, items: [] as HistoryItem[] } })),
        api.listFavorites().catch(() => ({ data: { total: 0, page: 1, page_size: 20, items: [] as Favorite[] } })),
        api.listAnswers().catch(() => ({ data: { total: 0, page: 1, page_size: 20, items: [] as AnswerRecord[] } })),
        api.listReactions().catch(() => ({ data: { total: 0, page: 1, page_size: 20, items: [] as Reaction[] } })),
        api.listModelingPackages().catch(() => ({ data: { total: 0, page: 1, page_size: 20, items: [] as GenerativeModelPackage[] } })),
      ]);
      setHistory(historyRes.data?.items || []);
      setFavorites(favRes.data?.items || []);
      setAnswers(answerRes.data?.items || []);
      setReactions(reactionRes.data?.items || []);
      setPackages(packageRes.data?.items || []);
    } catch (error) {
      setErrorText(error instanceof Error ? error.message : '加载学习记录失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { void loadData(); }, [loadData]);

  const stats = useMemo(() => {
    const days = new Set<string>();
    [...history, ...answers, ...packages].forEach((entry) => {
      const t = (entry as { created_at?: string }).created_at;
      if (t) days.add(new Date(t).toDateString());
    });
    return {
      questions: history.length + answers.length,
      models:    packages.length,
      favorites: favorites.length,
      days:      days.size,
    };
  }, [history, answers, packages, favorites]);

  const handleAuth = async (values: { email: string; password: string; nickname?: string }) => {
    setAuthLoading(true);
    try {
      const resp = authMode === 'login'
        ? await api.login({ email: values.email, password: values.password })
        : await api.register(values);
      if (resp.data) {
        setAuthTokens(resp.data.access_token, resp.data.refresh_token);
        message.success(authMode === 'login' ? '登录成功' : '注册成功');
        await loadData();
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : '认证失败');
    } finally {
      setAuthLoading(false);
    }
  };

  const logout = () => {
    clearAuthTokens();
    setProfile(null);
    setAuthed(false);
    message.success('已退出，当前为访客模式');
  };

  const handleHistoryClick = (item: HistoryItem) => {
    const text = item.query || '';
    const subject =
      item.action_type?.includes('physics') ? 'physics' :
      item.action_type?.includes('biology') ? 'biology' : null;
    if (subject) {
      router.push(`/modeling?type=${subject}&q=${encodeURIComponent(text)}`);
    } else {
      router.push(`/ask?q=${encodeURIComponent(text)}`);
    }
  };

  const handleFavoriteClick = (item: Favorite) => {
    if (item.target_type === 'model_package' && item.target_id) {
      router.push(`/modeling?package_id=${encodeURIComponent(item.target_id)}`);
      return;
    }
    const routeMap: Record<string, string> = {
      search:        '/ask',
      answer:        '/ask',
      evidence:      '/ask',
      physics:       '/modeling?type=physics',
      biology:       '/modeling?type=biology',
      render_code:   '/modeling',
      model_config:  '/admin/llm',
    };
    const route = routeMap[item.target_type];
    if (!route) return;
    router.push(route.includes('?') ? `${route}&q=${encodeURIComponent(item.title)}` : `${route}?q=${encodeURIComponent(item.title)}`);
  };

  // ── 未登录态 ─────────────────────────────────────
  if (authed === false) {
    return (
      <div className="snowy-page">
        <div className="snowy-page-heading">
          <h1>我的学习</h1>
          <p>登录后会自动记录提问、收藏、点赞、模型包，方便回顾。</p>
        </div>

        <div className="snowy-auth-cta">
          <div>
            <Title level={3} style={{ marginTop: 0, marginBottom: 8 }}>登录解锁个人学习档案</Title>
            <Paragraph style={{ marginBottom: 0, color: 'var(--color-text-muted)' }}>
              问过的、收藏的、做过的题，都会在这里整理好。无需密码也能继续访问首页、提问与推演。
            </Paragraph>
          </div>
          <Space>
            <Button type="primary" size="large" icon={<LoginOutlined />} onClick={() => document.getElementById('snowy-auth-form')?.scrollIntoView({ behavior: 'smooth' })}>
              立即登录 / 注册
            </Button>
          </Space>
        </div>

        <div id="snowy-auth-form" style={{ background: 'var(--color-bg-card)', border: '1px solid var(--color-border)', borderRadius: 12, padding: 24, maxWidth: 480 }}>
          <Segmented
            block
            value={authMode}
            onChange={(value) => setAuthMode(value as 'login' | 'register')}
            options={[{ label: '登录', value: 'login' }, { label: '注册', value: 'register' }]}
            style={{ marginBottom: 16 }}
          />
          <Form layout="vertical" onFinish={handleAuth} requiredMark={false}>
            <Form.Item name="email" label="邮箱" rules={[{ required: true, type: 'email', message: '请输入合法邮箱' }]}>
              <Input placeholder="you@example.com" autoComplete="email" />
            </Form.Item>
            <Form.Item name="password" label="密码" rules={[{ required: true, min: authMode === 'register' ? 8 : 1, message: authMode === 'register' ? '密码至少 8 位' : '请输入密码' }]}>
              <Input.Password placeholder="请输入密码" autoComplete={authMode === 'login' ? 'current-password' : 'new-password'} />
            </Form.Item>
            {authMode === 'register' && (
              <Form.Item name="nickname" label="昵称">
                <Input placeholder="可选" autoComplete="nickname" />
              </Form.Item>
            )}
            <Button type="primary" htmlType="submit" block loading={authLoading} icon={<LoginOutlined />}>
              {authMode === 'login' ? '登录' : '注册并登录'}
            </Button>
          </Form>
        </div>
      </div>
    );
  }

  if (loading || authed === null) {
    return (
      <div className="snowy-loading-card" style={{ margin: '60px auto', maxWidth: 360 }}>
        <span className="snowy-spinner" />
        <span>正在加载学习记录…</span>
      </div>
    );
  }

  const historyColumns: ColumnsType<HistoryItem> = [
    {
      title: '类型', dataIndex: 'action_type', key: 'action_type', width: 90,
      render: (type: string) => <Tag color={actionColor(type)} bordered={false}>{actionLabel(type)}</Tag>,
    },
    {
      title: '问题', dataIndex: 'query', key: 'query',
      render: (q: string) => <Text style={{ display: '-webkit-box', WebkitLineClamp: 1, WebkitBoxOrient: 'vertical', overflow: 'hidden' }}>{q || '—'}</Text>,
    },
    { title: '时间', dataIndex: 'created_at', key: 'created_at', width: 130, render: formatDateShort },
  ];

  const answerColumns: ColumnsType<AnswerRecord> = [
    { title: '问题', dataIndex: 'query', key: 'query' },
    {
      title: '可信', dataIndex: 'confidence', key: 'confidence', width: 90,
      render: (v: number) => <Tag color={v >= 0.8 ? 'green' : v >= 0.5 ? 'orange' : 'red'} bordered={false}>{Math.round((v || 0) * 100)}%</Tag>,
    },
    { title: '来源', dataIndex: 'source', key: 'source', width: 100, render: (s: string) => <Tag bordered={false}>{s || '-'}</Tag> },
    { title: '时间', dataIndex: 'created_at', key: 'created_at', width: 130, render: formatDateShort },
  ];

  const favoriteColumns: ColumnsType<Favorite> = [
    { title: '收藏内容', dataIndex: 'title', key: 'title' },
    {
      title: '类型', dataIndex: 'target_type', key: 'target_type', width: 110,
      render: (t: string) => <Tag color={targetColor(t)} bordered={false}>{targetLabel(t)}</Tag>,
    },
    { title: '时间', dataIndex: 'created_at', key: 'created_at', width: 130, render: formatDateShort },
  ];

  const reactionColumns: ColumnsType<Reaction> = [
    {
      title: '反馈', dataIndex: 'reaction_type', key: 'reaction_type', width: 80,
      render: (type: 'like' | 'dislike') => type === 'like'
        ? <Tag icon={<LikeOutlined />} color="green" bordered={false}>赞</Tag>
        : <Tag icon={<DislikeOutlined />} color="red" bordered={false}>踩</Tag>,
    },
    { title: '对象', dataIndex: 'target_id', key: 'target_id' },
    {
      title: '类型', dataIndex: 'target_type', key: 'target_type', width: 100,
      render: (t: string) => <Tag bordered={false}>{targetLabel(t)}</Tag>,
    },
    { title: '时间', dataIndex: 'updated_at', key: 'updated_at', width: 130, render: formatDateShort },
  ];

  const packageColumns: ColumnsType<GenerativeModelPackage> = [
    { title: '模型', key: 'title', render: (_, pkg) => packageTitle(pkg) },
    {
      title: '学科', dataIndex: 'domain', key: 'domain', width: 80,
      render: (d: string) => <Tag color={d === 'biology' ? 'green' : 'cyan'} bordered={false}>{d === 'biology' ? '生物' : '物理'}</Tag>,
    },
    {
      title: '可信', key: 'confidence', width: 80,
      render: (_, pkg) => {
        const v = pkg.validation_report?.confidence ?? pkg.confidence ?? 0;
        return <Tag color={v >= 0.8 ? 'green' : v >= 0.5 ? 'orange' : 'red'} bordered={false}>{Math.round(v * 100)}%</Tag>;
      },
    },
    {
      title: '状态', dataIndex: 'status', key: 'status', width: 90,
      render: (s: string) => <Tag color={s === 'success' ? 'green' : 'orange'} bordered={false}>{s || '-'}</Tag>,
    },
    { title: '时间', dataIndex: 'created_at', key: 'created_at', width: 130, render: formatDateShort },
  ];

  const emptyHint = (text: string) => (
    <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={<Text type="secondary">{text}</Text>} />
  );

  const tabItems = [
    {
      key: 'history', label: <Space size={4}><HistoryOutlined />提问历史</Space>,
      children: history.length === 0 ? emptyHint('还没有提问记录') : (
        <Table<HistoryItem>
          dataSource={history}
          columns={historyColumns}
          rowKey={(r) => r.id || `${r.action_type}-${r.created_at}-${r.query}`}
          size="middle"
          pagination={{ pageSize: 10, hideOnSinglePage: true }}
          onRow={(row) => ({ onClick: () => handleHistoryClick(row), style: { cursor: 'pointer' } })}
        />
      ),
    },
    {
      key: 'answers', label: <Space size={4}><CheckCircleOutlined />答案归档</Space>,
      children: answers.length === 0 ? emptyHint('暂无归档答案') : (
        <Table<AnswerRecord>
          dataSource={answers}
          columns={answerColumns}
          rowKey={(r) => r.id}
          size="middle"
          pagination={{ pageSize: 10, hideOnSinglePage: true }}
          onRow={(row) => ({ onClick: () => router.push(`/ask?q=${encodeURIComponent(row.query)}`), style: { cursor: 'pointer' } })}
        />
      ),
    },
    {
      key: 'favorites', label: <Space size={4}><StarOutlined />收藏</Space>,
      children: favorites.length === 0 ? emptyHint('暂无收藏内容') : (
        <Table<Favorite>
          dataSource={favorites}
          columns={favoriteColumns}
          rowKey={(r) => r.id}
          size="middle"
          pagination={{ pageSize: 10, hideOnSinglePage: true }}
          onRow={(row) => ({ onClick: () => handleFavoriteClick(row), style: { cursor: 'pointer' } })}
        />
      ),
    },
    {
      key: 'reactions', label: <Space size={4}><LikeOutlined />我的反馈</Space>,
      children: reactions.length === 0 ? emptyHint('暂无点赞 / 点踩') : (
        <Table<Reaction>
          dataSource={reactions}
          columns={reactionColumns}
          rowKey={(r) => r.id}
          size="middle"
          pagination={{ pageSize: 10, hideOnSinglePage: true }}
        />
      ),
    },
    {
      key: 'packages', label: <Space size={4}><ExperimentOutlined />模型包</Space>,
      children: packages.length === 0 ? emptyHint('暂无生成的模型包') : (
        <Table<GenerativeModelPackage>
          dataSource={packages}
          columns={packageColumns}
          rowKey={(r) => r.package_id}
          size="middle"
          pagination={{ pageSize: 10, hideOnSinglePage: true }}
          onRow={(row) => ({ onClick: () => router.push(`/modeling?package_id=${encodeURIComponent(row.package_id)}`), style: { cursor: 'pointer' } })}
        />
      ),
    },
  ];

  return (
    <div className="snowy-page">
      <div className="snowy-page-heading">
        <h1>我的学习</h1>
        <p>问过的、收藏的、做过的题，一站式回看。</p>
      </div>

      {errorText && (
        <Alert
          type="warning" showIcon
          message="部分数据加载失败"
          description={errorText}
          action={<Button size="small" icon={<ReloadOutlined />} onClick={loadData}>重试</Button>}
          style={{ marginBottom: 16, borderRadius: 12 }}
        />
      )}

      {/* 用户卡 */}
      <div className="snowy-user-card">
        <div className="snowy-avatar">{avatarOf(profile)}</div>
        <div>
          <div style={{ fontSize: 18, fontWeight: 600 }}>{displayName(profile)}</div>
          {profile?.email && <Text type="secondary" style={{ fontSize: 13 }}>{profile.email}</Text>}
        </div>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={loadData}>刷新</Button>
          <Button icon={<LogoutOutlined />} onClick={logout}>退出</Button>
        </Space>
      </div>

      {/* 统计 */}
      <div className="snowy-stats-row">
        <div className="snowy-stat-card">
          <div className="snowy-stat-card__label"><SearchOutlined /> 问过的问题</div>
          <div className="snowy-stat-card__value">{stats.questions}</div>
        </div>
        <div className="snowy-stat-card">
          <div className="snowy-stat-card__label"><ExperimentOutlined /> 生成的模型</div>
          <div className="snowy-stat-card__value">{stats.models}</div>
        </div>
        <div className="snowy-stat-card">
          <div className="snowy-stat-card__label"><StarOutlined /> 收藏数</div>
          <div className="snowy-stat-card__value">{stats.favorites}</div>
        </div>
        <div className="snowy-stat-card">
          <div className="snowy-stat-card__label"><BookOutlined /> 学习天数</div>
          <div className="snowy-stat-card__value">{stats.days}</div>
        </div>
      </div>

      {/* Tabs */}
      <div style={{ background: 'var(--color-bg-card)', border: '1px solid var(--color-border)', borderRadius: 12, padding: '4px 16px 16px' }}>
        <Tabs items={tabItems} defaultActiveKey="history" />
      </div>

      <div style={{ marginTop: 24, padding: 16, background: 'var(--color-bg-subtle)', borderRadius: 12, color: 'var(--color-text-muted)', fontSize: 13 }}>
        <BranchesOutlined /> 提示：在<Button type="link" size="small" onClick={() => router.push('/ask')}>提问页</Button>或
        <Button type="link" size="small" onClick={() => router.push('/modeling')}>推演页</Button>右上角点击「收藏」即可加入这里。
      </div>
    </div>
  );
}
