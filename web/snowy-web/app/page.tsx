'use client';

import React, { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Input, Card, Row, Col, Typography, Space, Tag, Spin, Alert } from 'antd';
import {
  SearchOutlined,
  ExperimentOutlined,
  BranchesOutlined,
  RocketOutlined,
} from '@ant-design/icons';
import { api, type RecommendationsResp } from '@/lib/api';

const { Title, Paragraph, Text } = Typography;
const { Search } = Input;

const fallbackRecommendations: RecommendationsResp = {
  hot_topics: [
    { id: 'newton-law', title: '牛顿第二定律', description: '受力分析与加速度', category: 'physics', icon: '⚙️' },
    { id: 'projectile', title: '平抛运动', description: '运动分解与轨迹', category: 'physics', icon: '🏀' },
    { id: 'photosynthesis', title: '光合作用', description: '光反应、暗反应与变量分析', category: 'biology', icon: '🌿' },
  ],
  physics_models: [
    { id: 'projectile-model', title: '平抛运动轨迹', description: 'Rapier 3D 原生仿真', category: 'physics' },
    { id: 'force-model', title: '斜面受力分析', description: '拆解受力与加速度', category: 'physics' },
    { id: 'oscillation-model', title: '简谐运动', description: '位移-时间关系可视化', category: 'physics' },
  ],
  biology_topics: [
    { id: 'cell-membrane', title: '细胞膜选择透过性', description: '结构与功能关系', category: 'biology' },
    { id: 'enzyme', title: '酶活性影响因素', description: '实验变量与结果分析', category: 'biology' },
    { id: 'synapse', title: '突触传递', description: '信号传导过程图谱', category: 'biology' },
  ],
};

export default function HomePage() {
  const router = useRouter();
  const [recommendations, setRecommendations] = useState<RecommendationsResp>(fallbackRecommendations);
  const [loading, setLoading] = useState(true);
  const [recommendationError, setRecommendationError] = useState<string | null>(null);

  useEffect(() => {
    api.getRecommendations()
      .then((res) => {
        if (res.data) {
          setRecommendations(res.data);
          setRecommendationError(null);
        }
      })
      .catch((error) => {
        setRecommendationError(error instanceof Error ? error.message : '推荐加载失败，已使用本地推荐');
        setRecommendations(fallbackRecommendations);
      })
      .finally(() => setLoading(false));
  }, []);

  const handleSearch = (value: string) => {
    if (value.trim()) {
      router.push(`/search?q=${encodeURIComponent(value.trim())}`);
    }
  };

  return (
    <div style={{ maxWidth: 900, margin: '0 auto', paddingTop: 60 }}>
      {/* Hero */}
      <div style={{ textAlign: 'center', marginBottom: 48 }}>
        <Title level={1} style={{ marginBottom: 8 }}>
          ❄️ Snowy 学习平台
        </Title>
        <Paragraph type="secondary" style={{ fontSize: 16 }}>
          面向高中生的 AIGC 知识检索、物理仿真与生物可视化统一建模工具
        </Paragraph>
        <Search
          placeholder="输入你的学习问题，如：牛顿第二定律的推导过程"
          enterButton={<><SearchOutlined /> 搜索</>}
          size="large"
          onSearch={handleSearch}
          style={{ maxWidth: 600, marginTop: 24 }}
        />
      </div>

      {/* Quick Links */}
      <Row gutter={[16, 16]} style={{ marginBottom: 32 }}>
        <Col xs={24} sm={12}>
          <Card
            hoverable
            onClick={() => router.push('/search')}
            style={{ textAlign: 'center', borderColor: '#1677ff' }}
          >
            <SearchOutlined style={{ fontSize: 32, color: '#1677ff' }} />
            <Title level={4} style={{ marginTop: 12 }}>知识检索</Title>
            <Paragraph type="secondary">搜索课本知识、考纲要点、题库</Paragraph>
          </Card>
        </Col>
        <Col xs={24} sm={12}>
          <Card
            hoverable
            onClick={() => router.push('/modeling')}
            style={{ textAlign: 'center', borderColor: '#52c41a' }}
          >
            <ExperimentOutlined style={{ fontSize: 32, color: '#52c41a' }} />
            <Title level={4} style={{ marginTop: 12 }}>统一建模</Title>
            <Paragraph type="secondary">物理 Rapier 3D 仿真 + 生物动态可视化</Paragraph>
          </Card>
        </Col>
      </Row>

      {recommendationError && (
        <Alert
          type="warning"
          showIcon
          message="推荐接口暂不可用，已展示本地兜底内容"
          description={recommendationError}
          style={{ marginBottom: 16 }}
        />
      )}

      {/* Recommendations */}
      {loading ? (
        <div style={{ textAlign: 'center', padding: 40 }}><Spin tip="正在加载推荐..." /></div>
      ) : (
        <>
          {/* Hot Topics */}
          <Card title={<><RocketOutlined /> 热门知识</>} style={{ marginBottom: 16 }}>
            {recommendations.hot_topics.length > 0 ? (
              <Space wrap>
                {recommendations.hot_topics.map((item) => (
                  <Tag
                    key={item.id}
                    color="blue"
                    style={{ cursor: 'pointer', padding: '4px 12px', fontSize: 14 }}
                    onClick={() => router.push(`/search?q=${encodeURIComponent(item.title)}`)}
                  >
                    {item.icon} {item.title}
                  </Tag>
                ))}
              </Space>
            ) : (
              <Text type="secondary">暂无热门知识，可直接使用上方搜索。</Text>
            )}
          </Card>

          {/* Physics Models */}
          <Card title={<><ExperimentOutlined /> 物理模型</>} style={{ marginBottom: 16 }}>
            <Row gutter={[12, 12]}>
              {recommendations.physics_models.map((item) => (
                <Col key={item.id} xs={12} sm={8} md={6}>
                  <Card
                    size="small"
                    hoverable
                    onClick={() => router.push(`/modeling?type=physics&q=${encodeURIComponent(item.title)}`)}
                  >
                    <div style={{ fontWeight: 500 }}>{item.title}</div>
                    <div style={{ fontSize: 12, color: '#999' }}>{item.description}</div>
                  </Card>
                </Col>
              ))}
            </Row>
          </Card>

          {/* Biology Topics */}
          <Card title={<><BranchesOutlined /> 生物主题</>} style={{ marginBottom: 16 }}>
            <Row gutter={[12, 12]}>
              {recommendations.biology_topics.map((item) => (
                <Col key={item.id} xs={12} sm={8} md={6}>
                  <Card
                    size="small"
                    hoverable
                    onClick={() => router.push(`/modeling?type=biology&q=${encodeURIComponent(item.title)}`)}
                  >
                    <div style={{ fontWeight: 500 }}>{item.title}</div>
                    <div style={{ fontSize: 12, color: '#999' }}>{item.description}</div>
                  </Card>
                </Col>
              ))}
            </Row>
          </Card>
        </>
      )}
    </div>
  );
}
