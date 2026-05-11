'use client';

import React, { Suspense, useCallback, useEffect, useMemo, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { Alert, Button, Card, Col, Empty, Input, List, Progress, Row, Select, Space, Spin, Tag, Typography, message } from 'antd';
import {
  BranchesOutlined,
  BulbOutlined,
  CompassOutlined,
  DeploymentUnitOutlined,
  ExperimentOutlined,
  FileSearchOutlined,
  FunctionOutlined,
  ReloadOutlined,
  SearchOutlined,
  StarOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons';
import { api, type FavoriteReq, type SearchResponse } from '@/lib/api';
import MarkdownText from '@/components/common/MarkdownText';
import ReactionBar from '@/components/common/ReactionBar';

const { Title, Paragraph, Text } = Typography;
const { Search } = Input;

const searchExamples = [
  '牛顿第二定律的适用条件是什么？',
  '光合作用中光照强度如何影响有机物积累？',
  '平抛运动 2 秒后的轨迹怎么分析？',
];

const sourceColor: Record<string, string> = {
  textbook: 'cyan',
  exam: 'gold',
  exercise: 'green',
  lecture: 'blue',
};

function confidenceLabel(value: number): { color: string; text: string } {
  if (value >= 0.8) return { color: 'green', text: '高可信' };
  if (value >= 0.5) return { color: 'orange', text: '需核验' };
  return { color: 'red', text: '低可信' };
}

function inferModelingSubject(text: string, selectedSubject?: string): 'physics' | 'biology' {
  if (selectedSubject === 'biology' || selectedSubject === 'physics') return selectedSubject;
  if (/光合|细胞|酶|遗传|突触|神经|生态|呼吸|膜|DNA|RNA|蛋白质/.test(text)) return 'biology';
  return 'physics';
}

function SearchPageInner() {
  const searchParams = useSearchParams();
  const router = useRouter();
  const [query, setQuery] = useState(searchParams.get('q') || '');
  const [result, setResult] = useState<SearchResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [errorText, setErrorText] = useState<string | null>(null);
  const [subject, setSubject] = useState<string | undefined>();
  const [grade, setGrade] = useState<string | undefined>();

  const handleSearch = useCallback(async (value: string) => {
    const text = value.trim();
    if (!text) return;

    setLoading(true);
    setErrorText(null);
    try {
      const res = await api.searchQuery({
        query: text,
        filters: { subject, grade },
      });
      setResult(res.data ?? null);
    } catch (error) {
      const messageText = error instanceof Error ? error.message : '检索失败，请稍后重试';
      setErrorText(messageText);
      setResult(null);
      message.error(messageText);
    } finally {
      setLoading(false);
    }
  }, [subject, grade]);

  useEffect(() => {
    const q = searchParams.get('q');
    if (q) {
      setQuery(q);
      void handleSearch(q);
    }
  }, [searchParams, handleSearch]);

  const confidence = result ? confidenceLabel(result.confidence) : null;
  const evidenceScore = useMemo(() => {
    if (!result?.citations?.length) return 0;
    const avg = result.citations.reduce((sum, citation) => sum + citation.score, 0) / result.citations.length;
    return Math.round(avg * 100);
  }, [result]);
  const inferredSubject = inferModelingSubject(query, subject);

  const handleFavorite = async () => {
    if (!result) return;
    const req: FavoriteReq = {
      target_type: 'search',
      target_id: query,
      title: query,
    };
    try {
      await api.addFavorite(req);
      message.success('收藏成功');
    } catch (error) {
      message.error(error instanceof Error ? error.message : '收藏失败');
    }
  };

  const renderEmptyActions = () => (
    <Space direction="vertical" align="center" size="middle">
      <Text type="secondary">输入问题开始检索，Snowy 会先找证据，再组织答案和建模入口。</Text>
      <Space wrap>
        {searchExamples.map((item) => (
          <Button
            key={item}
            onClick={() => {
              setQuery(item);
              void handleSearch(item);
            }}
          >
            试试：{item}
          </Button>
        ))}
      </Space>
    </Space>
  );

  return (
    <div className="snowy-page">
      <div className="snowy-page-heading">
        <span className="snowy-kicker"><FileSearchOutlined /> Evidence Star Map</span>
        <Title level={1}>知识星图</Title>
        <Paragraph>
          v5 检索不只返回答案，而是把引用、公式卡、易错点、题型映射和下一步建模任务连成证据链，帮助你判断 AI 结论是否可信。
        </Paragraph>
      </div>

      <Card className="snowy-glass-strong" style={{ marginBottom: 18 }}>
        <Search
          className="snowy-command-search"
          placeholder="输入问题或题目文本，例如：牛顿第二定律在斜面题中怎么用？"
          enterButton={<><SearchOutlined /> 检索证据</>}
          size="large"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          onSearch={handleSearch}
          loading={loading}
        />
        <Row gutter={[12, 12]} style={{ marginTop: 14 }} align="middle">
          <Col>
            <Select
              placeholder="学科"
              allowClear
              style={{ width: 140 }}
              value={subject}
              onChange={setSubject}
              options={[
                { value: 'physics', label: '物理' },
                { value: 'biology', label: '生物' },
                { value: 'chemistry', label: '化学' },
                { value: 'math', label: '数学' },
              ]}
            />
          </Col>
          <Col>
            <Select
              placeholder="年级"
              allowClear
              style={{ width: 140 }}
              value={grade}
              onChange={setGrade}
              options={[
                { value: 'high_school_1', label: '高一' },
                { value: 'high_school_2', label: '高二' },
                { value: 'high_school_3', label: '高三' },
              ]}
            />
          </Col>
          <Col flex="auto">
            <Space wrap>
              <Tag color="cyan">证据先于生成</Tag>
              <Tag color="green">可跳转建模</Tag>
              <Tag color="gold">题型迁移</Tag>
            </Space>
          </Col>
        </Row>
      </Card>

      {errorText && !loading && (
        <Alert
          type="error"
          showIcon
          message="检索请求失败"
          description={errorText}
          action={<Button size="small" icon={<ReloadOutlined />} onClick={() => void handleSearch(query)}>重试</Button>}
          style={{ marginBottom: 16 }}
        />
      )}

      {loading && (
        <Card className="snowy-glass" styles={{ body: { textAlign: 'center', padding: 54 } }}>
          <Spin size="large" tip="正在检索课本、考纲、题库和讲义证据..." />
        </Card>
      )}

      {!loading && result && (
        <div className="snowy-evidence-layout">
          <Space direction="vertical" size="middle" style={{ width: '100%' }}>
            <Card
              title={<Space><DeploymentUnitOutlined /> 答案摘要与证据等级</Space>}
              extra={(
                <Space wrap>
                  {confidence && <Tag color={confidence.color}>{confidence.text} {(result.confidence * 100).toFixed(0)}%</Tag>}
                  <ReactionBar targetType="answer" targetID={query} />
                  <Button icon={<StarOutlined />} size="small" onClick={handleFavorite}>收藏</Button>
                </Space>
              )}
              className="snowy-glass"
            >
              <MarkdownText content={result.answer} />
              <div className="snowy-stat-row" style={{ marginTop: 18 }}>
                <div className="snowy-stat"><strong>{result.citations?.length || 0}</strong><span>引用证据</span></div>
                <div className="snowy-stat"><strong>{evidenceScore}%</strong><span>平均证据分</span></div>
                <div className="snowy-stat"><strong>{result.knowledge_tags?.length || 0}</strong><span>知识标签</span></div>
              </div>
              {result.knowledge_tags && result.knowledge_tags.length > 0 && (
                <Space wrap style={{ marginTop: 14 }}>
                  {result.knowledge_tags.map((tag) => <Tag key={tag} color="cyan">{tag}</Tag>)}
                </Space>
              )}
            </Card>

            {result.citations && result.citations.length > 0 && (
              <Card title={<Space><FileSearchOutlined /> 引用来源</Space>} className="snowy-glass">
                <List
                  dataSource={result.citations}
                  renderItem={(citation, index) => (
                    <List.Item>
                      <List.Item.Meta
                        title={<Space wrap><Text>[{index + 1}] {citation.source_type} · {citation.doc_id}</Text><Tag color={sourceColor[citation.source_type] || 'blue'}>{(citation.score * 100).toFixed(0)}%</Tag></Space>}
                        description={citation.snippet}
                      />
                    </List.Item>
                  )}
                />
              </Card>
            )}

            <div className="snowy-insight-grid">
              {result.formula_cards && result.formula_cards.length > 0 && (
                <Card title={<Space><FunctionOutlined /> 公式 / 规律卡</Space>} className="snowy-glass">
                  <List
                    dataSource={result.formula_cards}
                    renderItem={(card) => (
                      <List.Item>
                        <List.Item.Meta
                          title={<Space wrap><Text strong>{card.name}</Text><Text code>{card.expression}</Text></Space>}
                          description={(
                            <Space direction="vertical" size={4}>
                              {card.variables && card.variables.length > 0 && <Text type="secondary">变量：{card.variables.join('；')}</Text>}
                              {card.applies_to && card.applies_to.length > 0 && <Text type="secondary">适用：{card.applies_to.join('；')}</Text>}
                              {card.limits && card.limits.length > 0 && <Text type="warning">边界：{card.limits.join('；')}</Text>}
                            </Space>
                          )}
                        />
                      </List.Item>
                    )}
                  />
                </Card>
              )}

              {result.misconceptions && result.misconceptions.length > 0 && (
                <Card title={<Space><BulbOutlined /> 易错点纠偏</Space>} className="snowy-glass">
                  <List
                    dataSource={result.misconceptions}
                    renderItem={(item) => (
                      <List.Item>
                        <List.Item.Meta title={<Text>{item.description}</Text>} description={item.correction} />
                        <Tag color="orange">{item.type}</Tag>
                      </List.Item>
                    )}
                  />
                </Card>
              )}
            </div>

            {result.exam_mappings && result.exam_mappings.length > 0 && (
              <Card title="题型映射" className="snowy-glass">
                <List
                  dataSource={result.exam_mappings}
                  renderItem={(item) => (
                    <List.Item>
                      <List.Item.Meta
                        title={<Space wrap><Tag color="gold">{item.question_type}</Tag><Text>{item.focus}</Text></Space>}
                        description={item.practice_hint}
                      />
                    </List.Item>
                  )}
                />
              </Card>
            )}
          </Space>

          <Space direction="vertical" size="middle" style={{ width: '100%' }}>
            <Card title={<Space><CompassOutlined /> 下一步任务</Space>} className="snowy-glass-strong">
              <Space direction="vertical" style={{ width: '100%' }} size="middle">
                <div>
                  <Text type="secondary">证据链完整度</Text>
                  <Progress percent={Math.max(Math.round(result.confidence * 100), evidenceScore)} strokeColor={{ '0%': '#22d3ee', '100%': '#34d399' }} />
                </div>
                <Button
                  block
                  type="primary"
                  icon={<ThunderboltOutlined />}
                  onClick={() => router.push(`/modeling?type=${inferredSubject}&q=${encodeURIComponent(query)}`)}
                >
                  用当前问题生成模型包
                </Button>
                <Button
                  block
                  icon={<ExperimentOutlined />}
                  onClick={() => router.push(`/modeling?type=physics&q=${encodeURIComponent(query)}`)}
                >
                  进入物理仿真挑战
                </Button>
                <Button
                  block
                  icon={<BranchesOutlined />}
                  onClick={() => router.push(`/modeling?type=biology&q=${encodeURIComponent(query)}`)}
                >
                  进入生物机制图谱
                </Button>
              </Space>
            </Card>

            {result.next_actions && result.next_actions.length > 0 && (
              <Card title="学习建议队列" className="snowy-glass">
                <List
                  size="small"
                  dataSource={result.next_actions}
                  renderItem={(action) => (
                    <List.Item>
                      <Space direction="vertical" size={2}>
                        <Text strong>{action.label}</Text>
                        {action.description && <Text type="secondary">{action.description}</Text>}
                        {action.tags && <Space wrap>{action.tags.map((tag) => <Tag key={tag}>{tag}</Tag>)}</Space>}
                      </Space>
                    </List.Item>
                  )}
                />
              </Card>
            )}

            {result.related_questions && result.related_questions.length > 0 && (
              <Card title="相关问题星轨" className="snowy-glass">
                <List
                  dataSource={result.related_questions}
                  renderItem={(related) => (
                    <List.Item
                      style={{ cursor: 'pointer' }}
                      onClick={() => {
                        setQuery(related.title);
                        void handleSearch(related.title);
                      }}
                    >
                      <Text type="secondary">{related.title}</Text>
                    </List.Item>
                  )}
                />
              </Card>
            )}

            {result.confidence < 0.5 && (
              <Alert
                message="结果可信度不足"
                description="建议补充题干条件、限定学科年级，或进入建模舱让 AI 标出假设与再推理条件。"
                type="warning"
                showIcon
              />
            )}
          </Space>
        </div>
      )}

      {!loading && !result && !errorText && (
        <Card className="snowy-glass" styles={{ body: { padding: '70px 16px' } }}>
          <Empty description={renderEmptyActions()} />
        </Card>
      )}
    </div>
  );
}

export default function SearchPage() {
  return (
    <Suspense fallback={<Spin />}>
      <SearchPageInner />
    </Suspense>
  );
}
