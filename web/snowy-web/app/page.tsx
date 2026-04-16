'use client';

import React, { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Input, Card, Row, Col, Typography, Space, Tag, Spin } from 'antd';
import {
  SearchOutlined,
  ExperimentOutlined,
  BranchesOutlined,
  RocketOutlined,
} from '@ant-design/icons';
import { api, type RecommendationsResp } from '@/lib/api';

const { Title, Paragraph } = Typography;
const { Search } = Input;

export default function HomePage() {
  const router = useRouter();
  const [recommendations, setRecommendations] = useState<RecommendationsResp | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api.getRecommendations()
      .then((res) => { if (res.data) setRecommendations(res.data); })
      .catch(() => { /* ignore */ })
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
          面向高中生的 AIGC 知识检索、物理建模、生物建模工具
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
        <Col xs={24} sm={8}>
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
        <Col xs={24} sm={8}>
          <Card
            hoverable
            onClick={() => router.push('/physics')}
            style={{ textAlign: 'center', borderColor: '#52c41a' }}
          >
            <ExperimentOutlined style={{ fontSize: 32, color: '#52c41a' }} />
            <Title level={4} style={{ marginTop: 12 }}>物理建模</Title>
            <Paragraph type="secondary">推导过程、2D 图表、参数调节</Paragraph>
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card
            hoverable
            onClick={() => router.push('/biology')}
            style={{ textAlign: 'center', borderColor: '#722ed1' }}
          >
            <BranchesOutlined style={{ fontSize: 32, color: '#722ed1' }} />
            <Title level={4} style={{ marginTop: 12 }}>生物建模</Title>
            <Paragraph type="secondary">概念图谱、过程分析、实验设计</Paragraph>
          </Card>
        </Col>
      </Row>

      {/* Recommendations */}
      {loading ? (
        <div style={{ textAlign: 'center', padding: 40 }}><Spin /></div>
      ) : recommendations && (
        <>
          {/* Hot Topics */}
          <Card title={<><RocketOutlined /> 热门知识</>} style={{ marginBottom: 16 }}>
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
          </Card>

          {/* Physics Models */}
          <Card title={<><ExperimentOutlined /> 物理模型</>} style={{ marginBottom: 16 }}>
            <Row gutter={[12, 12]}>
              {recommendations.physics_models.map((item) => (
                <Col key={item.id} xs={12} sm={8} md={6}>
                  <Card
                    size="small"
                    hoverable
                    onClick={() => router.push(`/physics?q=${encodeURIComponent(item.title)}`)}
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
                    onClick={() => router.push(`/biology?q=${encodeURIComponent(item.title)}`)}
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
