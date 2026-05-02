'use client';

import React, { useState, useEffect, useCallback, Suspense } from 'react';
import { useSearchParams, useRouter } from 'next/navigation';
import { Input, Card, Tag, Typography, Space, Spin, Alert, Select, Row, Col, List, Empty, Button, message } from 'antd';
import { SearchOutlined, StarOutlined, ExperimentOutlined, BranchesOutlined, ReloadOutlined } from '@ant-design/icons';
import { api, type SearchResponse, type FavoriteReq } from '@/lib/api';
import MarkdownText from '@/components/common/MarkdownText';

const { Title, Text } = Typography;
const { Search } = Input;

const searchExamples = [
  '牛顿第二定律的适用条件是什么？',
  '光合作用中光照强度如何影响有机物积累？',
  '平抛运动 2 秒后的轨迹怎么分析？',
];

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
      handleSearch(q);
    }
  }, [searchParams, handleSearch]);

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
      <Text type="secondary">输入问题开始检索，或先试一个示例。</Text>
      <Space wrap>
        {searchExamples.map((item) => (
          <Button
            key={item}
            onClick={() => {
              setQuery(item);
              handleSearch(item);
            }}
          >
            试试：{item}
          </Button>
        ))}
      </Space>
    </Space>
  );

  return (
    <div>
      <Title level={3}><SearchOutlined /> 知识检索</Title>

      {/* Search Bar */}
      <Card style={{ marginBottom: 16 }}>
        <Search
          placeholder="输入问题或题目文本"
          enterButton="搜索"
          size="large"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onSearch={handleSearch}
          loading={loading}
        />
        <Row gutter={16} style={{ marginTop: 12 }}>
          <Col>
            <Select
              placeholder="学科"
              allowClear
              style={{ width: 120 }}
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
              style={{ width: 120 }}
              value={grade}
              onChange={setGrade}
              options={[
                { value: 'high_school_1', label: '高一' },
                { value: 'high_school_2', label: '高二' },
                { value: 'high_school_3', label: '高三' },
              ]}
            />
          </Col>
        </Row>
      </Card>

      {errorText && !loading && (
        <Alert
          type="error"
          showIcon
          message="检索请求失败"
          description={errorText}
          action={(
            <Button size="small" icon={<ReloadOutlined />} onClick={() => handleSearch(query)}>
              重试
            </Button>
          )}
          style={{ marginBottom: 16 }}
        />
      )}

      {/* Loading */}
      {loading && (
        <div style={{ textAlign: 'center', padding: 40 }}><Spin size="large" tip="正在检索..." /></div>
      )}

      {/* Results */}
      {!loading && result && (
        <Row gutter={16}>
          <Col xs={24} lg={16}>
            {/* Answer */}
            <Card
              title="答案摘要"
              extra={
                <Space>
                  <Tag color={result.confidence >= 0.8 ? 'green' : result.confidence >= 0.5 ? 'orange' : 'red'}>
                    可信度 {(result.confidence * 100).toFixed(0)}%
                  </Tag>
                  <Button icon={<StarOutlined />} size="small" onClick={handleFavorite}>收藏</Button>
                </Space>
              }
              style={{ marginBottom: 16 }}
            >
              <MarkdownText content={result.answer} />
              {result.knowledge_tags && result.knowledge_tags.length > 0 && (
                <Space wrap style={{ marginTop: 8 }}>
                  {result.knowledge_tags.map((tag, i) => (
                    <Tag key={i} color="blue">{tag}</Tag>
                  ))}
                </Space>
              )}
            </Card>

            {/* Citations */}
            {result.citations && result.citations.length > 0 && (
              <Card title="引用来源" style={{ marginBottom: 16 }}>
                <List
                  dataSource={result.citations}
                  renderItem={(citation, index) => (
                    <List.Item>
                      <List.Item.Meta
                        title={<Text>[{index + 1}] {citation.source_type} · {citation.doc_id}</Text>}
                        description={citation.snippet}
                      />
                      <Tag color="geekblue">{(citation.score * 100).toFixed(0)}%</Tag>
                    </List.Item>
                  )}
                />
              </Card>
            )}

            {result.confidence < 0.5 && (
              <Alert
                message="结果可信度不足"
                description="建议更换关键词、补充更多条件，或跳转到物理/生物建模继续分析。"
                type="warning"
                showIcon
                style={{ marginBottom: 16 }}
              />
            )}
          </Col>

          <Col xs={24} lg={8}>
            {/* Related Questions */}
            {result.related_questions && result.related_questions.length > 0 && (
              <Card title="相关问题" style={{ marginBottom: 16 }}>
                <List
                  dataSource={result.related_questions}
                  renderItem={(q) => (
                    <List.Item
                      style={{ cursor: 'pointer' }}
                      onClick={() => {
                        setQuery(q.title);
                        handleSearch(q.title);
                      }}
                    >
                      <Text type="secondary">{q.title}</Text>
                    </List.Item>
                  )}
                />
              </Card>
            )}

            {/* Jump to Modeling */}
            <Card title="跳转建模" size="small">
              <Space direction="vertical" style={{ width: '100%' }}>
                <Button
                  block
                  icon={<ExperimentOutlined />}
                  onClick={() => router.push(`/modeling?type=physics&q=${encodeURIComponent(query)}`)}
                >
                  物理建模
                </Button>
                <Button
                  block
                  icon={<BranchesOutlined />}
                  onClick={() => router.push(`/modeling?type=biology&q=${encodeURIComponent(query)}`)}
                >
                  生物建模
                </Button>
              </Space>
            </Card>
          </Col>
        </Row>
      )}

      {!loading && !result && !errorText && (
        <Empty description={renderEmptyActions()} style={{ paddingTop: 60 }} />
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
