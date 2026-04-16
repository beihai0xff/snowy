'use client';

import React, { useState, useCallback, useEffect, useRef } from 'react';
import { Modal, Button, message, Typography, Divider, Spin } from 'antd';
import { GoogleOutlined } from '@ant-design/icons';
import { api } from '@/lib/api';
import { useAuthStore } from '@/stores/auth';

const { Text } = Typography;

const GOOGLE_CLIENT_ID = process.env.NEXT_PUBLIC_GOOGLE_CLIENT_ID || '';

interface AuthModalProps {
  open: boolean;
  onClose: () => void;
}

declare global {
  interface Window {
    google?: {
      accounts: {
        id: {
          initialize: (config: {
            client_id: string;
            callback: (response: { credential: string }) => void;
            auto_select?: boolean;
          }) => void;
          renderButton: (element: HTMLElement, config: {
            theme?: string;
            size?: string;
            width?: number;
            text?: string;
            shape?: string;
            locale?: string;
          }) => void;
          prompt: () => void;
        };
      };
    };
  }
}

export default function AuthModal({ open, onClose }: AuthModalProps) {
  const [loading, setLoading] = useState(false);
  const { setAuth } = useAuthStore();
  const googleBtnRef = useRef<HTMLDivElement>(null);
  const [gsiReady, setGsiReady] = useState(false);

  const handleGoogleCredential = useCallback(async (credential: string) => {
    setLoading(true);
    try {
      const res = await api.googleLogin({ id_token: credential });
      if (res.data) {
        const profileRes = await api.getProfile();
        if (profileRes.data) {
          setAuth(profileRes.data, res.data.access_token, res.data.refresh_token);
          message.success('登录成功');
          onClose();
        }
      }
    } catch {
      message.error('Google 登录失败，请重试');
    } finally {
      setLoading(false);
    }
  }, [setAuth, onClose]);

  // Load Google Identity Services SDK
  useEffect(() => {
    if (!GOOGLE_CLIENT_ID) return;
    if (typeof window !== 'undefined' && window.google?.accounts?.id) {
      setGsiReady(true);
      return;
    }

    const script = document.createElement('script');
    script.src = 'https://accounts.google.com/gsi/client';
    script.async = true;
    script.defer = true;
    script.onload = () => setGsiReady(true);
    document.head.appendChild(script);

    return () => {
      // Cleanup only if we added it
      if (script.parentNode) {
        script.parentNode.removeChild(script);
      }
    };
  }, []);

  // Render Google button when GSI is ready and modal is open
  useEffect(() => {
    if (!open || !gsiReady || !GOOGLE_CLIENT_ID || !googleBtnRef.current) return;
    if (!window.google?.accounts?.id) return;

    window.google.accounts.id.initialize({
      client_id: GOOGLE_CLIENT_ID,
      callback: (response: { credential: string }) => {
        handleGoogleCredential(response.credential);
      },
    });

    // Clear the container before rendering
    googleBtnRef.current.innerHTML = '';
    window.google.accounts.id.renderButton(googleBtnRef.current, {
      theme: 'outline',
      size: 'large',
      width: 320,
      text: 'signin_with',
      shape: 'rectangular',
      locale: 'zh_CN',
    });
  }, [open, gsiReady, handleGoogleCredential]);

  return (
    <Modal
      title="欢迎使用 Snowy"
      open={open}
      onCancel={onClose}
      footer={null}
      width={400}
      centered
    >
      <div style={{ textAlign: 'center', padding: '24px 0' }}>
        <Text type="secondary" style={{ display: 'block', marginBottom: 24 }}>
          使用 Google 账号快速登录，无需注册
        </Text>

        {loading ? (
          <Spin size="large" />
        ) : GOOGLE_CLIENT_ID ? (
          <div style={{ display: 'flex', justifyContent: 'center' }}>
            <div ref={googleBtnRef} />
          </div>
        ) : (
          /* Fallback when Google Client ID is not configured (dev mode) */
          <>
            <Divider>开发模式</Divider>
            <Button
              type="primary"
              icon={<GoogleOutlined />}
              size="large"
              block
              onClick={() => handleGoogleCredential('dev-mock-token')}
            >
              使用 Google 账号登录
            </Button>
            <Text type="secondary" style={{ display: 'block', marginTop: 12, fontSize: 12 }}>
              开发环境未配置 NEXT_PUBLIC_GOOGLE_CLIENT_ID
            </Text>
          </>
        )}
      </div>
    </Modal>
  );
}
