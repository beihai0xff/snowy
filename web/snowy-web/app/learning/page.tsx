'use client';

import React, { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Card, Typography, Tabs, List, Tag, Empty, Spin, Avatar, Button, Space, message } from 'antd';
import {
  BookOutlined,
  HistoryOutlined,
  StarOutlined,
  UserOutlined,
  SearchOutlined,
  ExperimentOutlined,
  BranchesOutlined,
} from '@ant-design/icons';
import { api, type User, type HistoryItem, type Favorite } from '@/lib/api';
import { useAuthStore } from '@/stores/auth';

const { Title, Text, Paragraph } = Typography;

const actionTypeIcon: Record<string, React.ReactNode> = {
  search: <SearchOutlined />,
  physics: <ExperimentOutlined />,
  biology: <BranchesOutlined />,
  register: <UserOutlined />,
};

const actionTypeColor: Record<string, string> = {
  search: 'blue',
  physics: 'green',
  biology: 'purple',
};

export default function LearningPage() {
  const router = useRouter();
  const { isLoggedIn, user } = useAuthStore();
  const [profile, setProfile] = useState<User | null>(user);
  const [history, setHistory] = useState<HistoryItem[]>([]);
  const [favorites, setFavorites] = useState<Favorite[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!isLoggedIn) {
      setLoading(false);
      return;
    }

    const loadData = async () => {
      try {
        const [profileRes, historyRes, favRes] = await Promise.all([
          api.getProfile(),
          api.getHistory(),
          api.listFavorites(),
        ]);
        if (profileRes.data) setProfile(profileRes.data);
        if (historyRes.data) setHistory(historyRes.data.items || []);
        if (favRes.data) setFavorites(favRes.data.items || []);
      } catch {
        message.error('加载数据失败');
      } finally {
        setLoading(false);
      }
    };
    loadData();
  }, [isLoggedIn]);

  if (!isLoggedIn) {
    return (
      <div style={{ textAlign: 'center', paddingTop: 80 }}>
        <BookOutlined style={{ fontSize: 64, color: '#ccc' }} />
        <Title level={3} style={{ marginTop: 16, color: '#999' }}>请先登录</Title>
        <Paragraph type="secondary">登录后可查看学习历史、收藏内容</Paragraph>
      </div>
    );
  }

  if (loading) {
    return <div style={{ textAlign: 'center', padding: 80 }}><Spin size="large" /></div>;
  }

  const handleHistoryClick = (item: HistoryItem) => {
    const routeMap: Record<string, string> = {
      search: '/search',
      physics: '/physics',
      biology: '/biology',
    };
    const route = routeMap[item.action_type];
    if (route) {
      router.push(`${route}?q=${encodeURIComponent(item.query)}`);
    }
  };

  const handleFavoriteClick = (item: Favorite) => {
    const routeMap: Record<string, string> = {
      search: '/search',
      physics: '/physics',
      biology: '/biology',
    };
    const route = routeMap[item.target_type];
    if (route) {
      router.push(`${route}?q=${encodeURIComponent(item.title)}`);
    }
  };

  const tabItems = [
    {
      key: 'history',
      label: <><HistoryOutlined /> 历史记录</>,
      children: history.length === 0 ? (
        <Empty description="暂无历史记录" />
      ) : (
        <List
          dataSource={history}
          renderItem={(item) => (
            <List.Item
              style={{ cursor: 'pointer' }}
              onClick={() => handleHistoryClick(item)}
              actions={[
                <Text key="time" type="secondary" style={{ fontSize: 12 }}>
                  {new Date(item.created_at).toLocaleDateString('zh-CN')}
                </Text>,
              ]}
            >
              <List.Item.Meta
                avatar={actionTypeIcon[item.action_type]}
                title={item.query}
                description={
                  <Tag color={actionTypeColor[item.action_type]}>{item.action_type}</Tag>
                }
              />
            </List.Item>
          )}
        />
      ),
    },
    {
      key: 'favorites',
      label: <><StarOutlined /> 收藏内容</>,
      children: favorites.length === 0 ? (
        <Empty description="暂无收藏" />
      ) : (
        <List
          dataSource={favorites}
          renderItem={(item) => (
            <List.Item
              style={{ cursor: 'pointer' }}
              onClick={() => handleFavoriteClick(item)}
              actions={[
                <Text key="time" type="secondary" style={{ fontSize: 12 }}>
                  {new Date(item.created_at).toLocaleDateString('zh-CN')}
                </Text>,
              ]}
            >
              <List.Item.Meta
                avatar={<StarOutlined style={{ color: '#faad14' }} />}
                title={item.title}
                description={
                  <Tag color={actionTypeColor[item.target_type]}>{item.target_type}</Tag>
                }
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

      {/* Profile Card */}
      {profile && (
        <Card style={{ marginBottom: 16 }}>
          <Space size="large">
            <Avatar size={64} style={{ backgroundColor: '#1677ff' }} icon={<UserOutlined />} />
            <div>
              <Title level={4} style={{ margin: 0 }}>{profile.nickname}</Title>
              <Text type="secondary">{profile.email || profile.nickname}</Text>
              <div style={{ marginTop: 4 }}>
                <Tag color="blue">{profile.role}</Tag>
                <Text type="secondary" style={{ fontSize: 12 }}>
                  注册于 {new Date(profile.created_at).toLocaleDateString('zh-CN')}
                </Text>
              </div>
            </div>
            <Space>
              <Button onClick={() => router.push('/search')}>去检索</Button>
              <Button onClick={() => router.push('/physics')}>物理建模</Button>
              <Button onClick={() => router.push('/biology')}>生物建模</Button>
            </Space>
          </Space>
        </Card>
      )}

      {/* History & Favorites */}
      <Card>
        <Tabs items={tabItems} />
      </Card>
    </div>
  );
}
