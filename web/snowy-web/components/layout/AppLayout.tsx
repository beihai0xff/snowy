'use client';

import React, { useSyncExternalStore } from 'react';
import { usePathname, useRouter } from 'next/navigation';
import { Button, ConfigProvider, Layout, Menu, Space, Tag, theme as antdTheme } from 'antd';
import {
  HomeOutlined,
  SearchOutlined,
  ExperimentOutlined,
  BookOutlined,
  DashboardOutlined,
  ThunderboltOutlined,
  UserOutlined,
} from '@ant-design/icons';

const { Header, Content } = Layout;

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
  return () => {
    window.removeEventListener('storage', callback);
    window.removeEventListener('focus', callback);
  };
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

export default function AppLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const accessToken = useSyncExternalStore(subscribeAuthStorage, getAuthTokenSnapshot, () => '');

  const handleMenuClick = (e: { key: string }) => {
    router.push(e.key);
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
            <Button size="small" icon={<UserOutlined />} onClick={() => router.push('/learning')}>
              {accessToken ? '已登录' : '访客模式'}
            </Button>
          </Space>
        </Header>
        <Content className="snowy-content">
          {children}
        </Content>
      </Layout>
    </ConfigProvider>
  );
}
