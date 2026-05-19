'use client';

import React, { useCallback, useEffect, useMemo, useState, useSyncExternalStore } from 'react';
import { usePathname, useRouter } from 'next/navigation';
import { Button, ConfigProvider, Dropdown, Form, Input, Layout, Menu, Modal, Segmented, Space, Typography, message, theme as antdTheme } from 'antd';
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
  UserOutlined,
} from '@ant-design/icons';
import { api, clearAuthTokens, setAuthTokens, type User } from '@/lib/api';

const { Header, Content } = Layout;
const { Text } = Typography;

type AuthMode = 'login' | 'register';

const authStorageEvent = 'snowy-auth-change';

const menuItems = [
  { key: '/',          icon: <HomeOutlined />,       label: '首页' },
  { key: '/ask',       icon: <SearchOutlined />,     label: '提问' },
  { key: '/modeling',  icon: <ExperimentOutlined />, label: '推演' },
  { key: '/learning',  icon: <BookOutlined />,       label: '我的学习' },
  { key: '/admin/llm', icon: <DashboardOutlined />,  label: '监控' },
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
  if (pathname.startsWith('/search') || pathname.startsWith('/ask')) return '/ask';
  if (pathname.startsWith('/modeling')) return '/modeling';
  if (pathname.startsWith('/learning')) return '/learning';
  if (pathname.startsWith('/admin/llm') || pathname.startsWith('/monitoring')) return '/admin/llm';
  return '/';
}

function userLabel(profile: User | null, accessToken: string): string {
  if (!accessToken) return '登录 / 注册';
  if (profile?.nickname) return profile.nickname;
  if (profile?.email) return profile.email;
  return '已登录';
}

// 管理后台路径不渲染主导航壳，只渲染纯白页面
function isAdminRoute(pathname: string): boolean {
  return pathname.startsWith('/admin');
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

  const handleMenuClick = (e: { key: string }) => router.push(e.key);

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
        { key: 'login',    icon: <LoginOutlined />, label: '登录' },
        { key: 'register', icon: <UserOutlined />,  label: '注册' },
      ];
    }
    return [
      { key: 'learning', icon: <BookOutlined />,   label: '打开我的学习' },
      { type: 'divider' },
      { key: 'logout',   icon: <LogoutOutlined />, label: '退出登录', danger: true },
    ];
  }, [accessToken]);

  const handleAccountClick: MenuProps['onClick'] = ({ key }) => {
    if (key === 'login' || key === 'register') { openAuth(key); return; }
    if (key === 'learning') { router.push('/learning'); return; }
    if (key === 'logout')   { handleLogout(); }
  };

  const themeConfig = {
    algorithm: antdTheme.defaultAlgorithm,
    token: {
      colorPrimary: '#2563EB',
      colorInfo:    '#2563EB',
      colorSuccess: '#16A34A',
      colorWarning: '#D97706',
      colorError:   '#DC2626',
      colorBgBase:      '#FAFAF9',
      colorBgContainer: '#FFFFFF',
      colorBgLayout:    '#FAFAF9',
      colorBorder:    '#E7E5E4',
      colorBorderSecondary: '#EDEBE9',
      colorTextBase: '#1C1917',
      colorText:     '#1C1917',
      colorTextSecondary: '#57534E',
      colorTextTertiary:  '#A8A29E',
      colorTextQuaternary:'#A8A29E',
      borderRadius: 8,
      borderRadiusLG: 12,
      borderRadiusSM: 6,
      borderRadiusXS: 4,
      fontFamily: '-apple-system, "PingFang SC", "Noto Sans SC", "Microsoft YaHei", Inter, "Segoe UI", Roboto, sans-serif',
      fontSize: 14,
      controlHeight: 36,
      wireframe: false,
      boxShadow:       '0 1px 3px rgba(28, 25, 23, 0.06), 0 1px 2px rgba(28, 25, 23, 0.04)',
      boxShadowSecondary: '0 4px 12px rgba(28, 25, 23, 0.08), 0 2px 4px rgba(28, 25, 23, 0.04)',
    },
    components: {
      Layout: {
        headerBg: 'transparent',
        bodyBg: 'transparent',
        footerBg: 'transparent',
      },
      Menu: {
        itemBg: 'transparent',
        itemSelectedBg: '#DBEAFE',
        itemSelectedColor: '#2563EB',
        itemHoverBg: '#F4F4F2',
        itemBorderRadius: 8,
        horizontalItemSelectedColor: '#2563EB',
        horizontalItemSelectedBg: '#DBEAFE',
        horizontalLineHeight: '36px',
        horizontalItemBorderRadius: 8,
      },
      Card: {
        headerBg: 'transparent',
        paddingLG: 24,
      },
      Button: {
        controlHeight: 36,
      },
      Tag: {
        borderRadiusSM: 4,
      },
    },
  } as const;

  const contentClassName = isAdminRoute(pathname) ? 'snowy-content snowy-content--wide' : 'snowy-content';

  return (
    <ConfigProvider theme={themeConfig}>
      <Layout className="snowy-shell">
        <Header className="snowy-header">
          <div
            className="snowy-brand"
            onClick={() => router.push('/')}
            role="button"
            tabIndex={0}
            onKeyDown={(e) => { if (e.key === 'Enter') router.push('/'); }}
          >
            <span className="snowy-brand-mark">❄</span>
            <span className="snowy-brand-name">Snowy</span>
          </div>
          <Menu
            mode="horizontal"
            selectedKeys={[selectedKey(pathname)]}
            items={menuItems}
            onClick={handleMenuClick}
            className="snowy-nav"
          />
          <Space size={8}>
            <Dropdown menu={{ items: accountItems, onClick: handleAccountClick }} trigger={['click']} placement="bottomRight">
              <Button size="middle" icon={<UserOutlined />} loading={profileLoading} onClick={(event) => event.preventDefault()}>
                <Space size={4}>
                  {userLabel(profile, accessToken)}
                  <DownOutlined style={{ fontSize: 10 }} />
                </Space>
              </Button>
            </Dropdown>
          </Space>
        </Header>
        <Content className={contentClassName}>
          {children}
        </Content>
        <footer className="snowy-footer">
          <div className="snowy-footer-inner">
            <span>Snowy · 面向高中生的 AI 学习工具</span>
            <Space size={16}>
              <Text type="secondary" style={{ fontSize: 14 }}>使用说明</Text>
              <Text type="secondary" style={{ fontSize: 14 }}>反馈</Text>
              <Text type="secondary" style={{ fontSize: 14 }}>2026</Text>
            </Space>
          </div>
        </footer>
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
