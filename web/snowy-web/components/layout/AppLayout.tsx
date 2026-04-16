'use client';

import React from 'react';
import { usePathname, useRouter } from 'next/navigation';
import { Layout, Menu } from 'antd';
import {
  HomeOutlined,
  SearchOutlined,
  ExperimentOutlined,
  BranchesOutlined,
  BookOutlined,
} from '@ant-design/icons';

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

  const handleMenuClick = (e: { key: string }) => {
    router.push(e.key);
  };

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
      </Header>
      <Content style={{ padding: '24px', maxWidth: 1200, margin: '0 auto', width: '100%' }}>
        {children}
      </Content>
    </Layout>
  );
}
