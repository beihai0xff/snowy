'use client';

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Alert, Button, Card, Empty, Form, Input, List, message, Segmented, Space, Spin, Tabs, Tag, Typography } from 'antd';
import {
  BookOutlined,
  BranchesOutlined,
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

const actionTypeIcon: Record<string, React.ReactNode> = {
  search: <SearchOutlined />,
  physics: <ExperimentOutlined />,
  biology: <BranchesOutlined />,
  modeling: <ExperimentOutlined />,
  model_package: <ExperimentOutlined />,
};

const actionTypeColor: Record<string, string> = {
  search: 'blue',
  physics: 'green',
  biology: 'purple',
  modeling: 'cyan',
  answer: 'geekblue',
  evidence: 'gold',
  model_package: 'cyan',
  render_code: 'green',
  model_config: 'purple',
};

const starterActions = [
  { label: '去检索', path: '/search', icon: <SearchOutlined /> },
  { label: '统一建模', path: '/modeling', icon: <ExperimentOutlined /> },
];

function packageTitle(pkg: GenerativeModelPackage): string {
  return pkg.learning_model?.learning_goal || pkg.generative_model?.learning_goal || pkg.question || pkg.package_id;
}

function formatDate(value?: string): string {
  if (!value) return '-';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '-';
  return date.toLocaleDateString('zh-CN');
}

export default function LearningPage() {
  const router = useRouter();
  const [history, setHistory] = useState<HistoryItem[]>([]);
  const [favorites, setFavorites] = useState<Favorite[]>([]);
  const [answers, setAnswers] = useState<AnswerRecord[]>([]);
  const [reactions, setReactions] = useState<Reaction[]>([]);
  const [packages, setPackages] = useState<GenerativeModelPackage[]>([]);
  const [profile, setProfile] = useState<User | null>(null);
  const [authMode, setAuthMode] = useState<'login' | 'register'>('login');
  const [loading, setLoading] = useState(true);
  const [authLoading, setAuthLoading] = useState(false);
  const [errorText, setErrorText] = useState<string | null>(null);

  const loadData = useCallback(async () => {
    setLoading(true);
    setErrorText(null);
    try {
      const [profileRes, historyRes, favRes, answerRes, reactionRes, packageRes] = await Promise.all([
        api.getProfile(),
        api.getHistory(),
        api.listFavorites(),
        api.listAnswers().catch(() => ({ data: { total: 0, page: 1, page_size: 20, items: [] as AnswerRecord[] } })),
        api.listReactions().catch(() => ({ data: { total: 0, page: 1, page_size: 20, items: [] as Reaction[] } })),
        api.listModelingPackages().catch(() => ({ data: { total: 0, page: 1, page_size: 20, items: [] as GenerativeModelPackage[] } })),
      ]);
      setProfile(profileRes.data || null);
      setHistory(historyRes.data?.items || []);
      setFavorites(favRes.data?.items || []);
      setAnswers(answerRes.data?.items || []);
      setReactions(reactionRes.data?.items || []);
      setPackages(packageRes.data?.items || []);
    } catch (error) {
      const messageText = error instanceof Error ? error.message : '加载数据失败';
      setErrorText(messageText);
      message.error(messageText);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadData();
  }, [loadData]);

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
    message.success('已退出，当前为访客模式');
    void loadData();
  };

  const profileLabel = useMemo(() => {
    if (profile?.email) return `当前账户：${profile.email}`;
    if (profile?.nickname) return `当前访客：${profile.nickname}`;
    return '当前为访客模式';
  }, [profile]);

  if (loading) {
    return <div style={{ textAlign: 'center', padding: 80 }}><Spin size="large" tip="正在加载学习记录..." /></div>;
  }

  const handleHistoryClick = (item: HistoryItem) => {
    const routeMap: Record<string, string> = {
      search: '/search',
      physics: '/modeling?type=physics',
      biology: '/modeling?type=biology',
      modeling: '/modeling',
    };
    const route = routeMap[item.action_type];
    if (route) {
      router.push(route.includes('?') ? `${route}&q=${encodeURIComponent(item.query)}` : `${route}?q=${encodeURIComponent(item.query)}`);
    }
  };

  const handleFavoriteClick = (item: Favorite) => {
    if (item.target_type === 'model_package' && item.target_id) {
      router.push(`/modeling?package_id=${encodeURIComponent(item.target_id)}`);
      return;
    }

    const routeMap: Record<string, string> = {
      search: '/search',
      answer: '/search',
      evidence: '/search',
      physics: '/modeling?type=physics',
      biology: '/modeling?type=biology',
      render_code: '/modeling',
      model_config: '/monitoring',
    };
    const route = routeMap[item.target_type];
    if (route) {
      router.push(route.includes('?') ? `${route}&q=${encodeURIComponent(item.title)}` : `${route}?q=${encodeURIComponent(item.title)}`);
    }
  };

  const emptyActions = (description: string) => (
    <Empty
      description={(
        <Space direction="vertical" align="center" size="middle">
          <Text type="secondary">{description}</Text>
          <Space wrap>
            {starterActions.map((item) => (
              <Button key={item.path} icon={item.icon} onClick={() => router.push(item.path)}>
                {item.label}
              </Button>
            ))}
          </Space>
        </Space>
      )}
    />
  );

  const tabItems = [
    {
      key: 'history',
      label: <><HistoryOutlined /> 历史问题</>,
      children: history.length === 0 ? emptyActions('暂无历史记录。完成一次搜索、物理建模或生物建模后会自动记录。') : (
        <List
          dataSource={history}
          renderItem={(item) => (
            <List.Item style={{ cursor: 'pointer' }} onClick={() => handleHistoryClick(item)} actions={[<Text key="time" type="secondary" style={{ fontSize: 12 }}>{formatDate(item.created_at)}</Text>]}>
              <List.Item.Meta avatar={actionTypeIcon[item.action_type] || <HistoryOutlined />} title={item.query} description={<Tag color={actionTypeColor[item.action_type] || 'default'}>{item.action_type}</Tag>} />
            </List.Item>
          )}
        />
      ),
    },
    {
      key: 'answers',
      label: <><SearchOutlined /> 答案归档</>,
      children: answers.length === 0 ? emptyActions('暂无答案归档。完成一次知识检索后，答案摘要、标签和引用证据会持久化到这里。') : (
        <List
          dataSource={answers}
          renderItem={(item) => (
            <List.Item
              style={{ cursor: 'pointer' }}
              onClick={() => router.push(`/search?q=${encodeURIComponent(item.query)}`)}
              actions={[<Text key="time" type="secondary" style={{ fontSize: 12 }}>{formatDate(item.created_at)}</Text>]}
            >
              <List.Item.Meta
                avatar={<SearchOutlined style={{ color: '#69c0ff' }} />}
                title={<Space wrap><Text>{item.query}</Text><Tag color="geekblue">{item.source}</Tag><Tag color="cyan">{Math.round((item.confidence || 0) * 100)}%</Tag></Space>}
                description={(
                  <Space direction="vertical" size={4}>
                    <Text type="secondary" ellipsis>{item.answer_summary}</Text>
                    <Space wrap>{(item.knowledge_tags || []).slice(0, 6).map((tag) => <Tag key={tag}>{tag}</Tag>)}</Space>
                  </Space>
                )}
              />
            </List.Item>
          )}
        />
      ),
    },
    {
      key: 'favorites',
      label: <><StarOutlined /> 收藏内容</>,
      children: favorites.length === 0 ? emptyActions('暂无收藏。搜索、答案、证据和模型包都可以进入收藏。') : (
        <List
          dataSource={favorites}
          renderItem={(item) => (
            <List.Item style={{ cursor: 'pointer' }} onClick={() => handleFavoriteClick(item)} actions={[<Text key="time" type="secondary" style={{ fontSize: 12 }}>{formatDate(item.created_at)}</Text>]}>
              <List.Item.Meta
                avatar={<StarOutlined style={{ color: '#faad14' }} />}
                title={item.title}
                description={(
                  <Space direction="vertical" size={4}>
                    <Space wrap>
                      <Tag color={actionTypeColor[item.target_type] || 'default'}>{item.target_type}</Tag>
                      {item.metadata_json && <Tag color="cyan">已保存快照</Tag>}
                    </Space>
                    {typeof item.metadata_json?.answer_summary === 'string' && <Text type="secondary" ellipsis>{item.metadata_json.answer_summary}</Text>}
                    {typeof item.metadata_json?.learning_goal === 'string' && <Text type="secondary" ellipsis>{item.metadata_json.learning_goal}</Text>}
                  </Space>
                )}
              />
            </List.Item>
          )}
        />
      ),
    },
    {
      key: 'reactions',
      label: <><LikeOutlined /> 质量反馈</>,
      children: reactions.length === 0 ? emptyActions('暂无点赞/点踩。v5 会把你的质量反馈用于社区判断和模型调优。') : (
        <List
          dataSource={reactions}
          renderItem={(item) => (
            <List.Item actions={[<Text key="time" type="secondary" style={{ fontSize: 12 }}>{formatDate(item.updated_at)}</Text>]}>
              <List.Item.Meta
                avatar={item.reaction_type === 'like' ? <LikeOutlined style={{ color: '#52c41a' }} /> : <DislikeOutlined style={{ color: '#ff4d4f' }} />}
                title={<Space><Text>{item.target_id}</Text><Tag color={item.reaction_type === 'like' ? 'green' : 'red'}>{item.reaction_type}</Tag></Space>}
                description={<Space><Tag color={actionTypeColor[item.target_type] || 'default'}>{item.target_type}</Tag><Tag>{item.visibility}</Tag></Space>}
              />
            </List.Item>
          )}
        />
      ),
    },
    {
      key: 'packages',
      label: <><ExperimentOutlined /> 生成式模型包</>,
      children: packages.length === 0 ? emptyActions('暂无模型包。完成一次科学建模后，这里会展示可回看的模型包、可信度和校验状态。') : (
        <List
          dataSource={packages}
          renderItem={(pkg) => (
            <List.Item
              style={{ cursor: 'pointer' }}
              onClick={() => router.push(`/modeling?package_id=${encodeURIComponent(pkg.package_id)}`)}
              actions={[<Text key="time" type="secondary" style={{ fontSize: 12 }}>{formatDate(pkg.created_at)}</Text>]}
            >
              <List.Item.Meta
                avatar={pkg.domain === 'biology' ? <BranchesOutlined /> : <ExperimentOutlined />}
                title={<Space wrap><Text>{packageTitle(pkg)}</Text><Tag color={pkg.domain === 'biology' ? 'gold' : 'green'}>{pkg.domain}</Tag></Space>}
                description={(
                  <Space wrap>
                    <Tag color={pkg.status === 'success' ? 'green' : pkg.status === 'fallback' ? 'orange' : 'red'}>{pkg.status || 'unknown'}</Tag>
                    <Tag color="cyan">可信度 {Math.round((pkg.validation_report?.confidence || pkg.confidence || 0) * 100)}%</Tag>
                    <Text type="secondary">{pkg.question}</Text>
                  </Space>
                )}
              />
            </List.Item>
          )}
        />
      ),
    },
  ];

  return (
    <div>
      <Title level={3}><BookOutlined /> 学习中心</Title>

      {errorText && (
        <Alert type="error" showIcon message="学习记录加载失败" description={errorText} action={<Button size="small" icon={<ReloadOutlined />} onClick={loadData}>重试</Button>} style={{ marginBottom: 16 }} />
      )}

      <Card style={{ marginBottom: 16 }}>
        <Space direction="vertical" size="middle" style={{ width: '100%' }}>
          <Space wrap style={{ justifyContent: 'space-between', width: '100%' }}>
            <div>
              <Text strong>{profileLabel}</Text>
              <Paragraph type="secondary" style={{ marginBottom: 0 }}>v5 支持邮箱登录，历史、答案归档、收藏、反馈和模型包将沉淀为个人学习档案。</Paragraph>
            </div>
            <Space wrap>
              {starterActions.map((item) => <Button key={item.path} icon={item.icon} onClick={() => router.push(item.path)}>{item.label}</Button>)}
              <Button icon={<ReloadOutlined />} onClick={loadData}>刷新档案</Button>
              <Button icon={<LogoutOutlined />} onClick={logout}>退出</Button>
            </Space>
          </Space>

          {!profile?.email && (
            <Card size="small" title={<Space><LoginOutlined /> 登录 / 注册</Space>}>
              <Segmented value={authMode} onChange={(v) => setAuthMode(v as 'login' | 'register')} options={[{ label: '登录', value: 'login' }, { label: '注册', value: 'register' }]} style={{ marginBottom: 16 }} />
              <Form layout="inline" onFinish={handleAuth} style={{ rowGap: 12 }}>
                <Form.Item name="email" rules={[{ required: true, type: 'email', message: '请输入合法邮箱' }]}><Input placeholder="邮箱" /></Form.Item>
                <Form.Item name="password" rules={[{ required: true, min: authMode === 'register' ? 8 : 1, message: '请输入密码' }]}><Input.Password placeholder="密码" /></Form.Item>
                {authMode === 'register' && <Form.Item name="nickname"><Input placeholder="昵称（可选）" /></Form.Item>}
                <Form.Item><Button type="primary" htmlType="submit" loading={authLoading}>{authMode === 'login' ? '登录' : '注册'}</Button></Form.Item>
              </Form>
            </Card>
          )}
        </Space>
      </Card>

      <Card>
        <Tabs items={tabItems} />
      </Card>
    </div>
  );
}
