'use client';

import React, { useCallback, useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Alert, Card, Typography, Tabs, List, Tag, Empty, Spin, Button, Space, message } from 'antd';
import {
  BookOutlined,
  HistoryOutlined,
  StarOutlined,
  SearchOutlined,
  ExperimentOutlined,
  BranchesOutlined,
  ReloadOutlined,
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

const starterActions = [
  { label: '去检索', path: '/search', icon: <SearchOutlined /> },
  { label: '物理建模', path: '/physics', icon: <ExperimentOutlined /> },
  { label: '生物建模', path: '/biology', icon: <BranchesOutlined /> },
];

export default function LearningPage() {
  const router = useRouter();
  const [history, setHistory] = useState<HistoryItem[]>([]);
  const [favorites, setFavorites] = useState<Favorite[]>([]);
  const [loading, setLoading] = useState(true);
  const [errorText, setErrorText] = useState<string | null>(null);

  const loadData = useCallback(async () => {
    setLoading(true);
    setErrorText(null);
    try {
      const [historyRes, favRes] = await Promise.all([
        api.getHistory(),
        api.listFavorites(),
      ]);
      setHistory(historyRes.data?.items || []);
      setFavorites(favRes.data?.items || []);
    } catch (error) {
      const messageText = error instanceof Error ? error.message : '加载数据失败';
      setErrorText(messageText);
      message.error(messageText);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadData();
  }, [loadData]);

  if (loading) {
    return <div style={{ textAlign: 'center', padding: 80 }}><Spin size="large" tip="正在加载学习记录..." /></div>;
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
      label: <><HistoryOutlined /> 历史记录</>,
      children: history.length === 0 ? (
        emptyActions('暂无历史记录。完成一次搜索、物理建模或生物建模后会自动记录。')
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
                avatar={actionTypeIcon[item.action_type] || <HistoryOutlined />}
                title={item.query}
                description={
                  <Tag color={actionTypeColor[item.action_type] || 'default'}>{item.action_type}</Tag>
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
        emptyActions('暂无收藏。搜索、物理建模、生物建模结果页可点击收藏。')
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
                  <Tag color={actionTypeColor[item.target_type] || 'default'}>{item.target_type}</Tag>
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

      {errorText && (
        <Alert
          type="error"
          showIcon
          message="学习记录加载失败"
          description={errorText}
          action={(
            <Button size="small" icon={<ReloadOutlined />} onClick={loadData}>
              重试
            </Button>
          )}
          style={{ marginBottom: 16 }}
        />
      )}

      {/* Quick Navigation */}
      <Card style={{ marginBottom: 16 }}>
        <Space wrap>
          {starterActions.map((item) => (
            <Button key={item.path} icon={item.icon} onClick={() => router.push(item.path)}>
              {item.label}
            </Button>
          ))}
        </Space>
      </Card>

      {/* History & Favorites */}
      <Card>
        <Tabs items={tabItems} />
      </Card>
    </div>
  );
}
