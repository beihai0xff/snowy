'use client';

import React, { useCallback, useEffect, useMemo, useState, useSyncExternalStore } from 'react';
import { usePathname, useRouter } from 'next/navigation';
import { Button, ConfigProvider, Dropdown, Form, Input, Layout, Menu, Modal, Segmented, Space, Tag, Typography, message, theme as antdTheme } from 'antd';
import type { MenuProps } from 'antd';
import {
  BookOutlined,
  DashboardOutlined,
  DownOutlined,
  ExperimentOutlined,
  HomeOutlined,
  LoginOutlined,
  LogoutOutlined,
  SearchOutlined,
  ThunderboltOutlined,
  UserOutlined,
} from '@ant-design/icons';
import { api, clearAuthTokens, setAuthTokens, type User } from '@/lib/api';

const { Header, Content } = Layout;
const { Text } = Typography;

type AuthMode = 'login' | 'register';

const authStorageEvent = 'snowy-auth-change';

const menuItems = [
  { key: '/', icon: <HomeOutlined />, label: '指挥舱' },
  { key: '/search', icon: <SearchOutlined />, label: '知识星图' },
  { key: '/modeling', icon: <ExperimentOutlined />, label: '科学建模舱' },
  { key: '/learning', icon: <BookOutlined />, label: '任务档案' },
  { key: '/monitoring', icon: <DashboardOutlined />, label: 'AI 监控' },
];

function subscribeAuthStorage(callback: () => void) {
  if (typeof window === 'undefined') return () => {};
  window.addEventListener('storage', callback);
  window.addEventListener('focus', callback);
  window.addEventListener(authStorageEvent, callback);
  return () => {
    window.removeEventListener('storage', callback);
    window.removeEventListener('focus', callback);
    window.removeEventListener(authStorageEvent, callback);
  };
}

function emitAuthChange() {
  if (typeof window === 'undefined') return;
  window.dispatchEvent(new Event(authStorageEvent));
}

function getAuthTokenSnapshot(): string {
  if (typeof window === 'undefined') return '';
  return window.localStorage.getItem('snowy_access_token') || '';
}

function selectedKey(pathname: string): string {
  if (pathname === '/physics' || pathname === '/biology') return '/modeling';
  if (pathname.startsWith('/search')) return '/search';
  if (pathname.startsWith('/modeling')) return '/modeling';
  if (pathname.startsWith('/learning')) return '/learning';
  if (pathname.startsWith('/monitoring')) return '/monitoring';
  return '/';
}

function userLabel(profile: User | null, accessToken: string): string {
  if (!accessToken) return '登录 / 注册';
  if (profile?.nickname) return profile.nickname;
  if (profile?.email) return profile.email;
  return '已登录';
}

export default function AppLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const accessToken = useSyncExternalStore(subscribeAuthStorage, getAuthTokenSnapshot, () => '');
  const [authOpen, setAuthOpen] = useState(false);
  const [authMode, setAuthMode] = useState<AuthMode>('login');
  const [authLoading, setAuthLoading] = useState(false);
  const [profile, setProfile] = useState<User | null>(null);
  const [profileLoading, setProfileLoading] = useState(false);
  const [form] = Form.useForm<{ email: string; password: string; nickname?: string }>();

  const loadProfile = useCallback(async () => {
    if (!accessToken) {
      setProfile(null);
      return;
    }

    setProfileLoading(true);
    try {
      const resp = await api.getProfile();
      setProfile(resp.data || null);
    } catch (error) {
      setProfile(null);
      message.warning(error instanceof Error ? `登录状态校验失败：${error.message}` : '登录状态校验失败');
    } finally {
      setProfileLoading(false);
    }
  }, [accessToken]);

  useEffect(() => {
    void loadProfile();
  }, [loadProfile]);

  const handleMenuClick = (e: { key: string }) => {
    router.push(e.key);
  };

  const openAuth = (mode: AuthMode = 'login') => {
    setAuthMode(mode);
    form.resetFields();
    setAuthOpen(true);
  };

  const closeAuth = () => {
    if (!authLoading) setAuthOpen(false);
  };

  const handleAuthSubmit = async (values: { email: string; password: string; nickname?: string }) => {
    setAuthLoading(true);
    try {
      const resp = authMode === 'login'
        ? await api.login({ email: values.email, password: values.password })
        : await api.register({ email: values.email, password: values.password, nickname: values.nickname });
      if (!resp.data) throw new Error('登录响应缺少 token');

      setAuthTokens(resp.data.access_token, resp.data.refresh_token);
      setProfile(resp.data.user);
      emitAuthChange();
      setAuthOpen(false);
      message.success(authMode === 'login' ? '登录成功' : '注册成功');
    } catch (error) {
      message.error(error instanceof Error ? error.message : '认证失败');
    } finally {
      setAuthLoading(false);
    }
  };

  const handleLogout = () => {
    clearAuthTokens();
    setProfile(null);
    emitAuthChange();
    message.success('已退出，当前为访客模式');
  };

  const accountItems: MenuProps['items'] = useMemo(() => {
    if (!accessToken) {
      return [
        { key: 'login', icon: <LoginOutlined />, label: '登录' },
        { key: 'register', icon: <UserOutlined />, label: '注册' },
      ];
    }

    return [
      { key: 'learning', icon: <BookOutlined />, label: '打开任务档案' },
      { type: 'divider' },
      { key: 'logout', icon: <LogoutOutlined />, label: '退出登录', danger: true },
    ];
  }, [accessToken]);

  const handleAccountClick: MenuProps['onClick'] = ({ key }) => {
    if (key === 'login' || key === 'register') {
      openAuth(key);
      return;
    }
    if (key === 'learning') {
      router.push('/learning');
      return;
    }
    if (key === 'logout') {
      handleLogout();
    }
  };

  return (
    <ConfigProvider
      theme={{
        algorithm: antdTheme.darkAlgorithm,
        token: {
          colorPrimary: '#38bdf8',
          colorInfo: '#22d3ee',
          colorSuccess: '#34d399',
          colorWarning: '#fbbf24',
          colorError: '#fb7185',
          colorBgBase: '#020617',
          colorBgContainer: 'rgba(15, 23, 42, 0.72)',
          colorBorder: 'rgba(148, 163, 184, 0.22)',
          colorTextBase: '#e2e8f0',
          borderRadius: 18,
          wireframe: false,
        },
        components: {
          Layout: {
            headerBg: 'transparent',
            bodyBg: 'transparent',
          },
          Menu: {
            darkItemBg: 'transparent',
            darkSubMenuItemBg: 'transparent',
            darkItemSelectedBg: 'rgba(56, 189, 248, 0.18)',
            itemBorderRadius: 999,
          },
          Card: {
            headerBg: 'transparent',
          },
        },
      }}
    >
      <Layout className="snowy-shell">
        <Header className="snowy-header">
          <div
            className="snowy-brand"
            onClick={() => router.push('/')}
            role="button"
            tabIndex={0}
            onKeyDown={(event) => {
              if (event.key === 'Enter') router.push('/');
            }}
          >
            <span className="snowy-brand-mark">❄</span>
            <span>
              <span className="snowy-brand-name">Snowy</span>
              <span className="snowy-brand-subtitle">AI Science Engine</span>
            </span>
          </div>
          <Menu
            mode="horizontal"
            theme="dark"
            selectedKeys={[selectedKey(pathname)]}
            items={menuItems}
            onClick={handleMenuClick}
            className="snowy-nav"
          />
          <Space className="snowy-header-status" size={8}>
            <Tag color="cyan" className="snowy-version-tag">V5</Tag>
            <Tag icon={<ThunderboltOutlined />} color="green" className="snowy-live-tag">Lab Online</Tag>
            <Dropdown menu={{ items: accountItems, onClick: handleAccountClick }} trigger={['click']} placement="bottomRight">
              <Button size="small" icon={<UserOutlined />} loading={profileLoading} onClick={(event) => event.preventDefault()}>
                <Space size={4}>
                  {userLabel(profile, accessToken)}
                  <DownOutlined style={{ fontSize: 10 }} />
                </Space>
              </Button>
            </Dropdown>
          </Space>
        </Header>
        <Content className="snowy-content">
          {children}
        </Content>
      </Layout>

      <Modal
        title={<Space><LoginOutlined /> {authMode === 'login' ? '登录 Snowy' : '注册 Snowy'}</Space>}
        open={authOpen}
        onCancel={closeAuth}
        footer={null}
        destroyOnHidden
      >
        <Space direction="vertical" size="middle" style={{ width: '100%' }}>
          <Segmented
            block
            value={authMode}
            onChange={(value) => {
              setAuthMode(value as AuthMode);
              form.resetFields(['nickname']);
            }}
            options={[{ label: '登录', value: 'login' }, { label: '注册', value: 'register' }]}
          />
          <Text type="secondary">
            登录后历史、收藏、反馈和模型包会写入你的学习档案；未登录仍可按访客模式试用。
          </Text>
          <Form form={form} layout="vertical" onFinish={handleAuthSubmit} requiredMark={false}>
            <Form.Item name="email" label="邮箱" rules={[{ required: true, type: 'email', message: '请输入合法邮箱' }]}>
              <Input autoComplete="email" placeholder="you@example.com" />
            </Form.Item>
            <Form.Item
              name="password"
              label="密码"
              rules={[{ required: true, min: authMode === 'register' ? 8 : 1, message: authMode === 'register' ? '密码至少 8 位' : '请输入密码' }]}
            >
              <Input.Password autoComplete={authMode === 'login' ? 'current-password' : 'new-password'} placeholder="请输入密码" />
            </Form.Item>
            {authMode === 'register' && (
              <Form.Item name="nickname" label="昵称" rules={[{ max: 32, message: '昵称最多 32 个字符' }]}>
                <Input autoComplete="nickname" placeholder="可选" />
              </Form.Item>
            )}
            <Button type="primary" htmlType="submit" block loading={authLoading} icon={<LoginOutlined />}>
              {authMode === 'login' ? '登录' : '注册并登录'}
            </Button>
          </Form>
        </Space>
      </Modal>
    </ConfigProvider>
  );
}
