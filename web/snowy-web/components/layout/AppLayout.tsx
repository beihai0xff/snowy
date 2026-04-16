'use client';

import React, { useState, useEffect } from 'react';
import { usePathname, useRouter } from 'next/navigation';
import { Layout, Menu, Button, Avatar, Dropdown, Space, message } from 'antd';
import {
  HomeOutlined,
  SearchOutlined,
  ExperimentOutlined,
  BranchesOutlined,
  BookOutlined,
  UserOutlined,
  LogoutOutlined,
  LoginOutlined,
} from '@ant-design/icons';
import { useAuthStore } from '@/stores/auth';
import { api } from '@/lib/api';
import AuthModal from '@/components/auth/AuthModal';

const { Header, Content } = Layout;

const menuItems = [
  { key: '/', icon: <HomeOutlined />, label: '首页' },
  { key: '/search', icon: <SearchOutlined />, label: '知识检索' },
  { key: '/physics', icon: <ExperimentOutlined />, label: '物理建模' },
  { key: '/biology', icon: <BranchesOutlined />, label: '生物建模' },
  { key: '/learning', icon: <BookOutlined />, label: '学习中心' },
];

export default function AppLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const { isLoggedIn, user, logout, loadFromStorage, setUser } = useAuthStore();
  const [authModalOpen, setAuthModalOpen] = useState(false);

  useEffect(() => {
    loadFromStorage();
  }, [loadFromStorage]);

  useEffect(() => {
    if (isLoggedIn && !user) {
      api.getProfile().then((res) => {
        if (res.data) setUser(res.data);
      }).catch(() => { /* ignore */ });
    }
  }, [isLoggedIn, user, setUser]);

  const handleMenuClick = (e: { key: string }) => {
    if (e.key === '/learning' && !isLoggedIn) {
      setAuthModalOpen(true);
      return;
    }
    router.push(e.key);
  };

  const handleLogout = () => {
    logout();
    message.success('已退出登录');
    router.push('/');
  };

  const userMenuItems = [
    { key: 'profile', icon: <UserOutlined />, label: user?.nickname || '个人中心', onClick: () => router.push('/learning') },
    { key: 'logout', icon: <LogoutOutlined />, label: '退出登录', onClick: handleLogout },
  ];

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Header style={{
        display: 'flex',
        alignItems: 'center',
        background: '#fff',
        borderBottom: '1px solid #f0f0f0',
        padding: '0 24px',
        position: 'sticky',
        top: 0,
        zIndex: 100,
      }}>
        <div
          style={{ fontSize: 20, fontWeight: 700, color: '#1677ff', cursor: 'pointer', marginRight: 40 }}
          onClick={() => router.push('/')}
        >
          ❄️ Snowy
        </div>
        <Menu
          mode="horizontal"
          selectedKeys={[pathname]}
          items={menuItems}
          onClick={handleMenuClick}
          style={{ flex: 1, border: 'none' }}
        />
        <Space>
          {isLoggedIn ? (
            <Dropdown menu={{ items: userMenuItems }} placement="bottomRight">
              <Avatar
                style={{ backgroundColor: '#1677ff', cursor: 'pointer' }}
                icon={<UserOutlined />}
              />
            </Dropdown>
          ) : (
            <Button type="primary" icon={<LoginOutlined />} onClick={() => setAuthModalOpen(true)}>
              登录
            </Button>
          )}
        </Space>
      </Header>
      <Content style={{ padding: '24px', maxWidth: 1200, margin: '0 auto', width: '100%' }}>
        {children}
      </Content>
      <AuthModal open={authModalOpen} onClose={() => setAuthModalOpen(false)} />
    </Layout>
  );
}
