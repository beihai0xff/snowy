'use client';

import React, { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Alert, Space, Tag, Typography } from 'antd';
import {
  BookOutlined,
  BranchesOutlined,
  ExperimentOutlined,
  RightOutlined,
  SearchOutlined,
} from '@ant-design/icons';
import { api, type RecommendationItem, type RecommendationsResp } from '@/lib/api';

const { Title, Paragraph, Text } = Typography;

const exampleQueries = [
  '为什么平抛运动水平方向是匀速？',
  '光合作用的限制因素有哪些？',
  '怎么判断带电粒子在磁场里的运动方向？',
];

type RecentEntry = {
  id: string;
  title: string;
  subject: 'physics' | 'biology' | 'general';
  href: string;
  time: string;
};

const fallbackRecommendations: RecommendationsResp = {
  hot_topics: [
    { id: 'newton-law', title: '牛顿第二定律：F=ma 如何用图理解？', description: '加速度与合外力、质量的关系演示，配合斜面迁移题。', category: 'physics' },
    { id: 'projectile', title: '平抛运动：水平为何是匀速？', description: '把速度拆成水平/竖直两条独立曲线，看图就懂。', category: 'physics' },
    { id: 'photosynthesis', title: '光合作用的"平台期"是怎么形成的？', description: '光照、CO₂、温度三个变量逐一压力测试。', category: 'biology' },
  ],
  physics_models: [
    { id: 'projectile-model', title: '平抛运动轨迹', description: '调节初速度和高度，落点、落地时间实时更新。', category: 'physics' },
    { id: 'force-model', title: '斜面受力分析', description: '可视化重力分力、支持力、摩擦力与加速度。', category: 'physics' },
    { id: 'oscillation-model', title: '简谐运动周期', description: '改变质量和劲度系数，观察周期与频率变化。', category: 'physics' },
  ],
  biology_topics: [
    { id: 'cell-membrane', title: '细胞膜的选择透过性', description: '结构 ↔ 功能 ↔ 证据，三层关系图谱。', category: 'biology' },
    { id: 'enzyme', title: '酶活性实验：变量分清楚', description: '自变量、因变量、控制变量逐一识别。', category: 'biology' },
    { id: 'synapse', title: '突触信号是怎么单向传递的？', description: '从动作电位到神经递质的完整过程图。', category: 'biology' },
  ],
};

const capabilityCards = [
  {
    title: '问问题',
    desc: '用大白话提问，得到带引用、带公式卡的答案。',
    icon: <SearchOutlined />,
    iconClass: 'snowy-capability__icon--ask',
    path: '/ask',
    meta: ['课本引用', '公式卡', '易错点'],
  },
  {
    title: '推演与图谱',
    desc: '物理仿真 + 生物图谱，看动画、调参数、对照证据。',
    icon: <ExperimentOutlined />,
    iconClass: 'snowy-capability__icon--lab',
    path: '/modeling',
    meta: ['可调参数', '动画演示', '证据可追溯'],
  },
  {
    title: '我的学习',
    desc: '问过的、收藏的、做过的题，一站式回看。',
    icon: <BookOutlined />,
    iconClass: 'snowy-capability__icon--my',
    path: '/learning',
    meta: ['答题历史', '收藏夹', '模型包'],
  },
];

function subjectTagColor(category: string): { color: string; label: string } {
  if (category === 'physics')   return { color: 'cyan',  label: '物理' };
  if (category === 'biology')   return { color: 'green', label: '生物' };
  if (category === 'chemistry') return { color: 'orange', label: '化学' };
  return { color: 'default', label: '通用' };
}

function recommendationHref(item: RecommendationItem): string {
  if (item.category === 'biology') return `/modeling?type=biology&q=${encodeURIComponent(item.title)}`;
  if (item.category === 'physics') return `/modeling?type=physics&q=${encodeURIComponent(item.title)}`;
  return `/ask?q=${encodeURIComponent(item.title)}`;
}

function inferAskRoute(text: string): string {
  if (!text.trim()) return '/ask';
  const wantsModel = /画|图|演示|推导|模拟|仿真|怎么动|轨迹|曲线|过程图/.test(text);
  const isBiology = /光合|细胞|酶|遗传|突触|神经|生态|呼吸|膜|DNA|RNA|蛋白质/.test(text);
  if (wantsModel) {
    return `/modeling?type=${isBiology ? 'biology' : 'physics'}&q=${encodeURIComponent(text)}`;
  }
  return `/ask?q=${encodeURIComponent(text)}`;
}

export default function HomePage() {
  const router = useRouter();
  const [query, setQuery] = useState('');
  const [recommendations, setRecommendations] = useState<RecommendationsResp>(fallbackRecommendations);
  const [recommendError, setRecommendError] = useState<string | null>(null);
  const [recentEntries, setRecentEntries] = useState<RecentEntry[] | null>(null);

  useEffect(() => {
    api.getRecommendations()
      .then((res) => {
        if (res.data) {
          setRecommendations(res.data);
          setRecommendError(null);
        }
      })
      .catch((error) => {
        setRecommendError(error instanceof Error ? error.message : '推荐接口未连通');
        setRecommendations(fallbackRecommendations);
      });
  }, []);

  useEffect(() => {
    api.getHistory()
      .then((res) => {
        const items = res.data?.items || [];
        if (!items.length) { setRecentEntries([]); return; }
        const mapped = items.slice(0, 6).map<RecentEntry>((entry) => {
          const text = entry.query || entry.action_type || '历史记录';
          const subject: RecentEntry['subject'] =
            entry.action_type?.includes('physics') ? 'physics' :
            entry.action_type?.includes('biology') ? 'biology' : 'general';
          const href =
            subject === 'physics' ? `/modeling?type=physics&q=${encodeURIComponent(text)}` :
            subject === 'biology' ? `/modeling?type=biology&q=${encodeURIComponent(text)}` :
                                    `/ask?q=${encodeURIComponent(text)}`;
          const time = entry.created_at ? new Date(entry.created_at).toLocaleString('zh-CN', { hour12: false, month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }) : '';
          return {
            id: entry.id || `${entry.action_type}-${entry.created_at}`,
            title: text,
            subject,
            href,
            time,
          };
        });
        setRecentEntries(mapped);
      })
      .catch(() => setRecentEntries([]));
  }, []);

  const submitSearch = () => {
    const text = query.trim();
    if (!text) return;
    router.push(inferAskRoute(text));
  };

  return (
    <div className="snowy-page">
      {/* Hero */}
      <section className="snowy-hero">
        <Title level={1} className="snowy-hero-title">
          像问同学一样问 <em>AI</em>，<br />得到带证据的答案
        </Title>
        <Paragraph className="snowy-hero-sub">
          用大白话提问 · 用动画看公式 · 用图谱理流程<br />
          专为高中生设计的学习工具，每个结论都能追溯到课本和考纲。
        </Paragraph>

        <form
          className="snowy-search snowy-search--lg snowy-hero-search"
          onSubmit={(event) => { event.preventDefault(); submitSearch(); }}
          role="search"
        >
          <span className="snowy-search__icon"><SearchOutlined /></span>
          <input
            className="snowy-search__input"
            type="search"
            placeholder="例如：为什么平抛运动水平方向是匀速？"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            aria-label="输入你的问题"
          />
          <button type="submit" className="snowy-search__submit">
            <RightOutlined /> 提问
          </button>
        </form>

        <div className="snowy-hero-examples">
          {exampleQueries.map((q) => (
            <button key={q} className="snowy-chip" type="button" onClick={() => { setQuery(q); router.push(inferAskRoute(q)); }}>
              {q}
            </button>
          ))}
        </div>
      </section>

      {/* 三大能力 */}
      <div className="snowy-capability">
        {capabilityCards.map((card) => (
          <a
            key={card.title}
            className="snowy-capability__card"
            href={card.path}
            onClick={(event) => { event.preventDefault(); router.push(card.path); }}
          >
            <span className={`snowy-capability__icon ${card.iconClass}`}>{card.icon}</span>
            <h3 className="snowy-capability__title">{card.title}</h3>
            <p className="snowy-capability__desc">{card.desc}</p>
            <div className="snowy-capability__meta">
              {card.meta.map((tag) => <Tag key={tag} bordered={false} color="default" style={{ background: 'var(--color-bg-subtle)', color: 'var(--color-text-muted)' }}>{tag}</Tag>)}
            </div>
          </a>
        ))}
      </div>

      {/* 最近学习 */}
      <section>
        <div className="snowy-section-title">
          <h2>最近在学</h2>
          <a href="/learning" onClick={(event) => { event.preventDefault(); router.push('/learning'); }}>查看全部 →</a>
        </div>

        {recentEntries === null && (
          <div className="snowy-loading-card"><span className="snowy-spinner" /><span>加载历史中…</span></div>
        )}

        {recentEntries && recentEntries.length === 0 && (
          <div className="snowy-loading-card" style={{ padding: 32 }}>
            <Text type="secondary">还没有学习记录，去<a href="/ask" onClick={(e) => { e.preventDefault(); router.push('/ask'); }}>问第一个问题</a>试试 ↗</Text>
          </div>
        )}

        {recentEntries && recentEntries.length > 0 && (
          <div className="snowy-scroller">
            {recentEntries.map((entry) => {
              const tagInfo = subjectTagColor(entry.subject);
              return (
                <article
                  key={entry.id}
                  className="snowy-recent-card"
                  onClick={() => router.push(entry.href)}
                  role="button"
                  tabIndex={0}
                  onKeyDown={(event) => { if (event.key === 'Enter') router.push(entry.href); }}
                >
                  <div className="snowy-recent-card__head">
                    <Tag color={tagInfo.color} bordered={false}>{tagInfo.label}</Tag>
                    <Text className="snowy-recent-card__time">{entry.time || '近期'}</Text>
                  </div>
                  <div className="snowy-recent-card__title">{entry.title}</div>
                  <Text type="secondary" style={{ fontSize: 12 }}>继续学习 →</Text>
                </article>
              );
            })}
          </div>
        )}
      </section>

      {/* 推荐 */}
      <section>
        <div className="snowy-section-title">
          <h2>推荐给你</h2>
          <Text type="secondary" style={{ fontSize: 14 }}>基于高中物理 / 生物高频考点</Text>
        </div>

        {recommendError && (
          <Alert
            type="warning"
            showIcon
            message="推荐接口暂不可用，已显示离线推荐"
            description={recommendError}
            style={{ marginBottom: 16, borderRadius: 12 }}
            closable
          />
        )}

        <div className="snowy-recommend-grid">
          {recommendations.hot_topics.slice(0, 3).map((item) => {
            const tagInfo = subjectTagColor(item.category);
            return (
              <article
                key={item.id}
                className="snowy-recommend-card"
                onClick={() => router.push(recommendationHref(item))}
                role="button"
                tabIndex={0}
                style={{ cursor: 'pointer' }}
                onKeyDown={(event) => { if (event.key === 'Enter') router.push(recommendationHref(item)); }}
              >
                <Space size={6} style={{ marginBottom: 12 }}>
                  <Tag color={tagInfo.color} bordered={false}>{tagInfo.label}</Tag>
                  <Tag color="default" bordered={false} style={{ background: 'var(--color-bg-subtle)', color: 'var(--color-text-muted)' }}>高频</Tag>
                </Space>
                <div className="snowy-recommend-card__title">{item.title}</div>
                <p className="snowy-recommend-card__desc">{item.description}</p>
                <Text type="secondary" style={{ fontSize: 13 }}>开始学习 →</Text>
              </article>
            );
          })}
        </div>

        <div className="snowy-section-title" style={{ marginTop: 8 }}>
          <h2>建模模板</h2>
          <Text type="secondary" style={{ fontSize: 14 }}>直接打开仿真，调参数看变化</Text>
        </div>
        <div className="snowy-recommend-grid">
          {recommendations.physics_models.slice(0, 3).map((item) => (
            <article
              key={item.id}
              className="snowy-recommend-card"
              onClick={() => router.push(recommendationHref(item))}
              role="button"
              tabIndex={0}
              style={{ cursor: 'pointer' }}
              onKeyDown={(event) => { if (event.key === 'Enter') router.push(recommendationHref(item)); }}
            >
              <Space size={6} style={{ marginBottom: 12 }}>
                <Tag color="cyan" bordered={false} icon={<ExperimentOutlined />}>物理仿真</Tag>
              </Space>
              <div className="snowy-recommend-card__title">{item.title}</div>
              <p className="snowy-recommend-card__desc">{item.description}</p>
              <Text type="secondary" style={{ fontSize: 13 }}>打开模型 →</Text>
            </article>
          ))}
        </div>

        <div className="snowy-section-title" style={{ marginTop: 8 }}>
          <h2>生物图谱</h2>
          <Text type="secondary" style={{ fontSize: 14 }}>用图理清「谁影响谁、按什么顺序发生」</Text>
        </div>
        <div className="snowy-recommend-grid">
          {recommendations.biology_topics.slice(0, 3).map((item) => (
            <article
              key={item.id}
              className="snowy-recommend-card"
              onClick={() => router.push(recommendationHref(item))}
              role="button"
              tabIndex={0}
              style={{ cursor: 'pointer' }}
              onKeyDown={(event) => { if (event.key === 'Enter') router.push(recommendationHref(item)); }}
            >
              <Space size={6} style={{ marginBottom: 12 }}>
                <Tag color="green" bordered={false} icon={<BranchesOutlined />}>生物图谱</Tag>
              </Space>
              <div className="snowy-recommend-card__title">{item.title}</div>
              <p className="snowy-recommend-card__desc">{item.description}</p>
              <Text type="secondary" style={{ fontSize: 13 }}>打开图谱 →</Text>
            </article>
          ))}
        </div>
      </section>
    </div>
  );
}
