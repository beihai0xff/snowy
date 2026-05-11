'use client';

import React, { useEffect, useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Alert, Button, Card, Col, Input, Row, Space, Spin, Tag, Typography } from 'antd';
import {
  AimOutlined,
  BranchesOutlined,
  CompassOutlined,
  ExperimentOutlined,
  FireOutlined,
  RadarChartOutlined,
  RocketOutlined,
  SearchOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons';
import { api, type RecommendationItem, type RecommendationsResp } from '@/lib/api';

const { Title, Paragraph, Text } = Typography;
const { Search } = Input;

const fallbackRecommendations: RecommendationsResp = {
  hot_topics: [
    { id: 'newton-law', title: '牛顿第二定律挑战', description: '用证据解释 F、m、a 的变量关系，并完成斜面迁移题', category: 'physics', icon: 'F' },
    { id: 'projectile', title: '平抛命中目标区', description: '调节高度与初速度，让轨迹穿过目标窗口', category: 'physics', icon: 'P' },
    { id: 'photosynthesis', title: '光合作用平台期诊断', description: '判断光照、CO2 和温度谁是限制因素', category: 'biology', icon: 'B' },
  ],
  physics_models: [
    { id: 'projectile-model', title: '平抛运动轨迹舱', description: '本地重算水平位移、落地时间和速度方向', category: 'physics' },
    { id: 'force-model', title: '斜面受力校验舱', description: '拆解重力分力、支持力、摩擦力和加速度', category: 'physics' },
    { id: 'oscillation-model', title: '简谐运动参数舱', description: '观察质量、劲度系数与周期变化关系', category: 'physics' },
  ],
  biology_topics: [
    { id: 'cell-membrane', title: '细胞膜证据图谱', description: '结构、功能与选择透过性证据链', category: 'biology' },
    { id: 'enzyme', title: '酶活性变量挑战', description: '区分自变量、因变量和无关变量', category: 'biology' },
    { id: 'synapse', title: '突触传递过程舱', description: '跟踪信号、递质、受体与方向性', category: 'biology' },
  ],
};

const capabilityCards = [
  {
    title: '知识星图',
    description: '先检索证据，再生成答案。每个结论带引用、公式卡、易错点和题型映射。',
    icon: <SearchOutlined />,
    accent: 'rgba(34, 211, 238, 0.36)',
    color: '#22d3ee',
    path: '/search',
  },
  {
    title: '物理仿真',
    description: '把题干变量变成可调参数，轨迹、向量、曲线和结论同步更新。',
    icon: <ExperimentOutlined />,
    accent: 'rgba(52, 211, 153, 0.32)',
    color: '#34d399',
    path: '/modeling?type=physics',
  },
  {
    title: '生物可视化',
    description: '将过程、机制、限制因素和实验变量组织成可解释的动态图谱。',
    icon: <BranchesOutlined />,
    accent: 'rgba(251, 191, 36, 0.3)',
    color: '#fbbf24',
    path: '/modeling?type=biology',
  },
];

const learningChain = [
  ['问题输入', '输入题目、现象或实验目标'],
  ['证据检索', '先找课本、考纲、题库依据'],
  ['领域解析', '抽取变量、机制和约束'],
  ['模型包', '生成可校验 LearningModelSpec'],
  ['交互仿真', '本地重算参数与曲线'],
  ['AI 教练', '解释失败、提示下一步'],
  ['微练习', '完成迁移题并归档'],
];

const radarStats = [
  ['Evidence', '可信证据'],
  ['Model', '结构化模型包'],
  ['Sim', '实时仿真'],
  ['Coach', 'AI 教练反馈'],
];

function missionPath(item: RecommendationItem): string {
  if (item.category === 'biology') return `/modeling?type=biology&q=${encodeURIComponent(item.title)}`;
  if (item.category === 'physics') return `/modeling?type=physics&q=${encodeURIComponent(item.title)}`;
  return `/search?q=${encodeURIComponent(item.title)}`;
}

function inferMissionSubject(text: string): 'physics' | 'biology' {
  if (/光合|细胞|酶|遗传|突触|神经|生态|呼吸|膜|DNA|RNA|蛋白质/.test(text)) return 'biology';
  return 'physics';
}

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
        setRecommendationError(error instanceof Error ? error.message : '推荐加载失败，已使用本地任务');
        setRecommendations(fallbackRecommendations);
      })
      .finally(() => setLoading(false));
  }, []);

  const missionCards = useMemo(() => [
    ...recommendations.hot_topics,
    ...recommendations.physics_models.slice(0, 2),
    ...recommendations.biology_topics.slice(0, 2),
  ].slice(0, 6), [recommendations]);

  const handleSearch = (value: string) => {
    const text = value.trim();
    if (text) router.push(`/modeling?type=${inferMissionSubject(text)}&q=${encodeURIComponent(text)}`);
  };

  return (
    <div className="snowy-page">
      <Row gutter={[24, 24]} align="middle" style={{ marginBottom: 28 }}>
        <Col xs={24} xl={13}>
          <Space direction="vertical" size="large" style={{ width: '100%' }}>
            <span className="snowy-kicker"><RocketOutlined /> Snowy V5 Mission Cockpit</span>
            <div>
              <Title className="snowy-hero-title" style={{ fontSize: 'clamp(46px, 7vw, 86px)' }}>
                AI 科学任务舱，<span className="snowy-gradient-text">把问题变成模型</span>
              </Title>
              <Paragraph className="snowy-hero-copy">
                面向高中生的 AIGC 知识检索、物理仿真与生物可视化统一建模工具。Snowy V5 以可信证据为燃料，以结构化模型包为核心，用任务、挑战和即时反馈完成学习闭环。
              </Paragraph>
            </div>
            <Search
              className="snowy-command-search"
              placeholder="输入题目、现象或建模目标，如：平抛运动怎样命中目标区？"
              enterButton={<><ThunderboltOutlined /> 发起任务</>}
              size="large"
              onSearch={handleSearch}
            />
            <Space wrap>
              <Button type="primary" icon={<ExperimentOutlined />} onClick={() => router.push('/modeling')}>
                进入科学建模舱
              </Button>
              <Button icon={<SearchOutlined />} onClick={() => router.push('/search')}>
                打开知识星图
              </Button>
              <Button icon={<CompassOutlined />} onClick={() => router.push('/learning')}>
                查看任务档案
              </Button>
            </Space>
          </Space>
        </Col>
        <Col xs={24} xl={11}>
          <Card className="snowy-glass-strong snowy-orbit" styles={{ body: { minHeight: 420, position: 'relative' } }}>
            <div className="snowy-core">
              <div>
                <strong>Model Spec</strong>
                <span>Evidence First</span>
              </div>
            </div>
            <div className="snowy-orbit-node"><strong>可信证据</strong><span>RAG 引用、知识标签、置信度先于生成答案。</span></div>
            <div className="snowy-orbit-node"><strong>生成模型包</strong><span>LearningModelSpec + SimulationSpec + InteractionPlan。</span></div>
            <div className="snowy-orbit-node"><strong>即时反馈</strong><span>参数变化同步轨迹、曲线、公式项和一句话结论。</span></div>
            <div className="snowy-orbit-node"><strong>迁移挑战</strong><span>演示关、单变量、多变量、变式题逐步推进。</span></div>
          </Card>
        </Col>
      </Row>

      <div className="snowy-card-grid" style={{ marginBottom: 28 }}>
        {capabilityCards.map((card) => (
          <button
            key={card.title}
            className="snowy-mission-card"
            style={{ '--accent': card.accent, '--chip-color': card.color } as React.CSSProperties}
            onClick={() => router.push(card.path)}
            type="button"
          >
            <span className="snowy-icon-chip">{card.icon}</span>
            <h3>{card.title}</h3>
            <p>{card.description}</p>
          </button>
        ))}
      </div>

      {recommendationError && (
        <Alert
          type="warning"
          showIcon
          message="任务接口暂不可用，已展示本地兜底任务"
          description={recommendationError}
          style={{ marginBottom: 16 }}
        />
      )}

      <Row gutter={[18, 18]} style={{ marginBottom: 28 }}>
        <Col xs={24} xl={17}>
          <Card title={<Space><AimOutlined /> v5 统一学习链路</Space>} className="snowy-glass">
            <div className="snowy-chain">
              {learningChain.map(([title, desc], index) => (
                <div className="snowy-chain-step" key={title}>
                  <b>{index + 1}</b>
                  <strong>{title}</strong>
                  <span>{desc}</span>
                </div>
              ))}
            </div>
          </Card>
        </Col>
        <Col xs={24} xl={7}>
          <Card title={<Space><RadarChartOutlined /> 能力雷达</Space>} className="snowy-glass" styles={{ body: { minHeight: 222 } }}>
            <div className="snowy-stat-row" style={{ gridTemplateColumns: 'repeat(2, minmax(0, 1fr))' }}>
              {radarStats.map(([code, label]) => (
                <div className="snowy-stat" key={code}>
                  <strong>{code}</strong>
                  <span>{label}</span>
                </div>
              ))}
            </div>
          </Card>
        </Col>
      </Row>

      <Card
        title={<Space><FireOutlined /> 今日任务卡</Space>}
        extra={<Tag color="cyan">Level v5.0</Tag>}
        className="snowy-glass"
      >
        {loading ? (
          <div style={{ textAlign: 'center', padding: 40 }}><Spin tip="正在同步任务卡..." /></div>
        ) : (
          <div className="snowy-task-list">
            {missionCards.map((item) => (
              <div className="snowy-task-card" key={`${item.category}-${item.id}`} onClick={() => router.push(missionPath(item))} role="button" tabIndex={0} onKeyDown={(event) => { if (event.key === 'Enter') router.push(missionPath(item)); }}>
                <Space align="center" style={{ width: '100%', justifyContent: 'space-between' }}>
                  <Tag color={item.category === 'biology' ? 'gold' : item.category === 'physics' ? 'green' : 'cyan'}>{item.category}</Tag>
                  <Text type="secondary">Mission</Text>
                </Space>
                <strong style={{ marginTop: 12 }}>{item.title}</strong>
                <p>{item.description}</p>
                <Space wrap>
                  <Tag color="cyan">证据</Tag>
                  <Tag color="green">建模</Tag>
                  <Tag color="orange">挑战</Tag>
                </Space>
              </div>
            ))}
          </div>
        )}
      </Card>
    </div>
  );
}
