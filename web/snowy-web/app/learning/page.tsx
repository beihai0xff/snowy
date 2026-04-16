'use client';

import React, { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Card, Typography, Tabs, List, Tag, Empty, Spin, Button, Space, message } from 'antd';
import {
  BookOutlined,
  HistoryOutlined,
  StarOutlined,
  SearchOutlined,
  ExperimentOutlined,
  BranchesOutlined,
} from '@ant-design/icons';
import { api, type HistoryItem, type Favorite } from '@/lib/api';

const { Title, Text } = Typography;

const actionTypeIcon: Record<string, React.ReactNode> = {
  search: <SearchOutlined />,
  physics: <ExperimentOutlined />,
  biology: <BranchesOutlined />,
};

const actionTypeColor: Record<string, string> = {
  search: 'blue',
  physics: 'green',
  biology: 'purple',
};

export default function LearningPage() {
  const router = useRouter();
  const [history, setHistory] = useState<HistoryItem[]>([]);
  const [favorites, setFavorites] = useState<Favorite[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const loadData = async () => {
      try {
        const [historyRes, favRes] = await Promise.all([
          api.getHistory(),
          api.listFavorites(),
        ]);
        if (historyRes.data) setHistory(historyRes.data.items || []);
        if (favRes.data) setFavorites(favRes.data.items || []);
      } catch {
        message.error('加载数据失败');
      } finally {
        setLoading(false);
      }
    };
    loadData();
  }, []);

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

      {/* Quick Navigation */}
      <Card style={{ marginBottom: 16 }}>
        <Space>
          <Button onClick={() => router.push('/search')}>去检索</Button>
          <Button onClick={() => router.push('/physics')}>物理建模</Button>
          <Button onClick={() => router.push('/biology')}>生物建模</Button>
        </Space>
      </Card>

      {/* History & Favorites */}
      <Card>
        <Tabs items={tabItems} />
      </Card>
    </div>
  );
}
