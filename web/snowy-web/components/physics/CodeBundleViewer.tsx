'use client';

import React, { useMemo } from 'react';
import { Button, Space, Tabs, Typography, message } from 'antd';
import { CopyOutlined } from '@ant-design/icons';
import type { RenderArtifact } from '@/lib/api';

const { Paragraph } = Typography;

interface CodeBundleViewerProps {
  artifact: RenderArtifact;
}

export default function CodeBundleViewer({ artifact }: CodeBundleViewerProps) {
  const fullBundleText = useMemo(() => {
    return Object.entries(artifact.code_bundle)
      .sort(([a], [b]) => a.localeCompare(b))
      .map(([file, content]) => `/* ${file} */\n${content}`)
      .join('\n\n');
  }, [artifact]);

  const copyText = async (label: string, content: string) => {
    try {
      await navigator.clipboard.writeText(content);
      message.success(`${label} 已复制`);
    } catch {
      message.error('复制失败，请手动选择代码');
    }
  };

  const items = useMemo(() => {
    return Object.entries(artifact.code_bundle)
      .sort(([a], [b]) => a.localeCompare(b))
      .map(([file, content]) => ({
        key: file,
        label: (
          <Space size={6}>
            <span>{file}</span>
            <Typography.Text type="secondary">{content.length}B</Typography.Text>
          </Space>
        ),
        children: (
          <Space direction="vertical" style={{ width: '100%' }}>
            <Space wrap>
              <Button size="small" icon={<CopyOutlined />} onClick={() => void copyText(file, content)}>
                复制
              </Button>
            </Space>
            <pre
              style={{
                margin: 0,
                whiteSpace: 'pre-wrap',
                wordBreak: 'break-word',
                fontSize: 12,
                lineHeight: 1.6,
                background: '#0f172a',
                color: '#e2e8f0',
                padding: 16,
                borderRadius: 8,
                overflowX: 'auto',
                maxHeight: 420,
              }}
            >
              {content}
            </pre>
          </Space>
        ),
      }));
  }, [artifact]);

  return (
    <div>
      <Space direction="vertical" style={{ width: '100%', marginBottom: 12 }}>
        <Paragraph type="secondary" style={{ marginBottom: 0 }}>
          这里展示的是后端返回的可渲染代码包，默认由浏览器沙箱直接消费 `index.html`。
        </Paragraph>
        <Space wrap>
          <Button size="small" icon={<CopyOutlined />} onClick={() => void copyText('完整代码包', fullBundleText)}>
            复制完整代码包
          </Button>
        </Space>
      </Space>
      <Tabs items={items} />
    </div>
  );
}
