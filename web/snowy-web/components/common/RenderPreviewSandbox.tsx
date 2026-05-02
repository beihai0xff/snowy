'use client';

import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Alert, Button, Space, Tag, Typography } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import type { RenderArtifact } from '@/lib/api';
import NativePhysicsPreview, { isNativePhysicsPreviewArtifact } from '@/components/physics/NativePhysicsPreview';

const { Text } = Typography;

export type PreviewStatus = 'loading' | 'ready' | 'updated' | 'error' | 'timeout';

export interface RenderPreviewSandboxProps {
  artifact: RenderArtifact;
  propsData: Record<string, number>;
  onStatusChange?: (status: PreviewStatus, detail?: string) => void;
}

function buildSrcDoc(artifact: RenderArtifact): string {
  const html = artifact.code_bundle['index.html'] || '';
  const css = artifact.code_bundle['styles.css'];
  const js = artifact.code_bundle['app.js'];

  if (!html) return '';

  let doc = html;
  if (css && !doc.includes(css)) {
    const styleTag = `<style data-snowy-bundle="styles.css">${css}</style>`;
    doc = doc.includes('</head>') ? doc.replace('</head>', `${styleTag}</head>`) : `${styleTag}${doc}`;
  }

  if (js && !doc.includes(js)) {
    const scriptTag = `<script data-snowy-bundle="app.js">${js}</script>`;
    doc = doc.includes('</body>') ? doc.replace('</body>', `${scriptTag}</body>`) : `${doc}${scriptTag}`;
  }

  return doc;
}

function IframeRenderPreviewSandbox({ artifact, propsData, onStatusChange }: RenderPreviewSandboxProps) {
  const iframeRef = useRef<HTMLIFrameElement | null>(null);
  const readyRef = useRef(false);
  const [status, setStatus] = useState<PreviewStatus>('loading');
  const [runtimeError, setRuntimeError] = useState<string | null>(null);
  const [lastSyncText, setLastSyncText] = useState('等待参数同步');

  const srcDoc = useMemo(() => buildSrcDoc(artifact), [artifact]);
  const artifactKey = `${artifact.scene_type}:${artifact.render_manifest.entry}:${Object.keys(artifact.code_bundle).sort().join('|')}`;
  const [iframeKey, setIframeKey] = useState(0);
  const supportsWebGL = useMemo(() => {
    const apis = artifact.render_manifest.allowed_apis || [];
    return apis.some((apiName) => apiName.toLowerCase().includes('webgl')) || artifact.scene_type.includes('3d');
  }, [artifact]);

  const updateStatus = useCallback((nextStatus: PreviewStatus, detail?: string) => {
    setStatus(nextStatus);
    onStatusChange?.(nextStatus, detail);
  }, [onStatusChange]);

  const postToPreview = useCallback((payload: unknown) => {
    iframeRef.current?.contentWindow?.postMessage(payload, '*');
  }, []);

  const pingPreview = useCallback(() => {
    const delays = [0, 250, 750, 1500, 3000];
    const timers = delays.map((delay) => window.setTimeout(() => {
      postToPreview({ type: 'snowy:ping' });
    }, delay));
    return () => timers.forEach((timer) => window.clearTimeout(timer));
  }, [postToPreview]);

  useEffect(() => {
    readyRef.current = false;
    const timer = window.setTimeout(() => {
      updateStatus('loading');
      setRuntimeError(null);
      setLastSyncText('等待参数同步');
    }, 0);

    return () => window.clearTimeout(timer);
  }, [artifactKey, iframeKey, updateStatus]);

  useEffect(() => {
    const handleMessage = (event: MessageEvent) => {
      if (event.source !== iframeRef.current?.contentWindow) return;
      const data = event.data as { source?: string; type?: string; status?: string; message?: string } | undefined;
      if (!data || data.source !== 'snowy-preview') return;

      if (data.type === 'preview') {
        readyRef.current = true;
        if (data.status === 'updated') updateStatus('updated');
        else updateStatus('ready');
        setRuntimeError(null);
      }

      if (data.type === 'error') {
        readyRef.current = false;
        updateStatus('error', data.message || '预览运行失败');
        setRuntimeError(data.message || '预览运行失败');
      }
    };

    window.addEventListener('message', handleMessage);
    return () => window.removeEventListener('message', handleMessage);
  }, [updateStatus]);

  useEffect(() => {
    const target = iframeRef.current?.contentWindow;
    if (!target) return undefined;

    const timer = window.setTimeout(() => {
      setLastSyncText('参数同步中…');
      target.postMessage({ type: 'snowy:update-props', props: propsData }, '*');
      target.postMessage({ type: 'snowy:ping' }, '*');
      setLastSyncText(`参数已发送 ${new Date().toLocaleTimeString('zh-CN', { hour12: false })}`);
    }, 0);

    return () => window.clearTimeout(timer);
  }, [propsData]);

  useEffect(() => {
    if (status !== 'loading') return undefined;
    const stopPings = pingPreview();
    const timer = window.setTimeout(() => {
      if (readyRef.current) return;
      const message = '未收到 ready，但画面可能已渲染；可继续操作，或重新加载预览。';
      updateStatus('timeout', message);
      setRuntimeError(message);
    }, 8000);

    return () => {
      stopPings();
      window.clearTimeout(timer);
    };
  }, [status, updateStatus, pingPreview]);

  const handleLoad = () => {
    if (!readyRef.current) {
      updateStatus('loading');
      setRuntimeError(null);
    }
    postToPreview({ type: 'snowy:update-props', props: propsData });
    pingPreview();
  };

  return (
    <Space direction="vertical" style={{ width: '100%' }}>
      <Space wrap>
        <Tag color={status === 'error' ? 'red' : status === 'timeout' ? 'gold' : status === 'loading' ? 'blue' : 'green'}>
          预览状态：{status}
        </Tag>
        {supportsWebGL && <Tag color="geekblue">3D WebGL</Tag>}
        <Tag color="lime">2D fallback</Tag>
        <Tag color="purple">iframe</Tag>
        <Tag color="purple">{artifact.render_mode}</Tag>
        <Tag color="cyan">{artifact.scene_type}</Tag>
        <Tag color={status === 'updated' || status === 'ready' ? 'green' : 'default'}>{lastSyncText}</Tag>
        <Button size="small" icon={<ReloadOutlined />} onClick={() => setIframeKey((value) => value + 1)}>
          重新加载预览
        </Button>
      </Space>

      {runtimeError && (
        <Alert
          type={status === 'timeout' ? 'warning' : 'error'}
          showIcon
          message={status === 'timeout' ? '浏览器预览未确认 ready' : '浏览器预览运行失败'}
          description={runtimeError}
        />
      )}

      <iframe
        key={`${artifactKey}:${iframeKey}`}
        ref={iframeRef}
        title={artifact.scene_type}
        srcDoc={srcDoc}
        sandbox="allow-scripts"
        referrerPolicy="no-referrer"
        onLoad={handleLoad}
        style={{
          width: '100%',
          minHeight: 520,
          border: '1px solid #e5e7eb',
          borderRadius: 12,
          background: '#0f172a',
        }}
      />

      <Text type="secondary">
        预览在 iframe 沙箱中运行，参数变化优先通过 postMessage 同步到前端代码，而不是重复请求后端。
      </Text>
    </Space>
  );
}

export default function RenderPreviewSandbox(props: RenderPreviewSandboxProps) {
  if (isNativePhysicsPreviewArtifact(props.artifact)) {
    return <NativePhysicsPreview {...props} />;
  }
  return <IframeRenderPreviewSandbox {...props} />;
}
