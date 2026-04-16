'use client';

import React, { useState } from 'react';
import { Modal, Form, Input, Button, Tabs, message } from 'antd';
import { PhoneOutlined, LockOutlined, UserOutlined } from '@ant-design/icons';
import { api } from '@/lib/api';
import { useAuthStore } from '@/stores/auth';

interface AuthModalProps {
  open: boolean;
  onClose: () => void;
}

export default function AuthModal({ open, onClose }: AuthModalProps) {
  const [activeTab, setActiveTab] = useState('login');
  const [loading, setLoading] = useState(false);
  const [codeSent, setCodeSent] = useState(false);
  const { setAuth } = useAuthStore();
  const [loginForm] = Form.useForm();
  const [registerForm] = Form.useForm();

  const handleSendCode = async () => {
    try {
      const phone = loginForm.getFieldValue('phone') || registerForm.getFieldValue('phone');
      if (!phone) {
        message.warning('请输入手机号');
        return;
      }
      await api.sendCode({ phone });
      setCodeSent(true);
      message.success('验证码已发送（开发环境任意验证码即可）');
    } catch {
      message.error('发送验证码失败');
    }
  };

  const handleLogin = async (values: { phone: string; code: string }) => {
    setLoading(true);
    try {
      const res = await api.login(values);
      if (res.data) {
        const profileRes = await api.getProfile();
        if (profileRes.data) {
          setAuth(profileRes.data, res.data.access_token, res.data.refresh_token);
          message.success('登录成功');
          onClose();
        }
      }
    } catch {
      message.error('登录失败，请检查手机号和验证码');
    } finally {
      setLoading(false);
    }
  };

  const handleRegister = async (values: { phone: string; nickname: string }) => {
    setLoading(true);
    try {
      await api.register(values);
      message.success('注册成功，请登录');
      setActiveTab('login');
    } catch {
      message.error('注册失败');
    } finally {
      setLoading(false);
    }
  };

  const tabItems = [
    {
      key: 'login',
      label: '登录',
      children: (
        <Form form={loginForm} onFinish={handleLogin} layout="vertical" size="large">
          <Form.Item name="phone" rules={[{ required: true, message: '请输入手机号' }]}>
            <Input prefix={<PhoneOutlined />} placeholder="手机号" />
          </Form.Item>
          <Form.Item name="code" rules={[{ required: true, message: '请输入验证码' }]}>
            <Input
              prefix={<LockOutlined />}
              placeholder="验证码"
              suffix={
                <Button type="link" size="small" onClick={handleSendCode} disabled={codeSent}>
                  {codeSent ? '已发送' : '获取验证码'}
                </Button>
              }
            />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={loading} block>
              登录
            </Button>
          </Form.Item>
        </Form>
      ),
    },
    {
      key: 'register',
      label: '注册',
      children: (
        <Form form={registerForm} onFinish={handleRegister} layout="vertical" size="large">
          <Form.Item name="phone" rules={[{ required: true, message: '请输入手机号' }]}>
            <Input prefix={<PhoneOutlined />} placeholder="手机号" />
          </Form.Item>
          <Form.Item name="nickname" rules={[{ required: true, message: '请输入昵称' }]}>
            <Input prefix={<UserOutlined />} placeholder="昵称" />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={loading} block>
              注册
            </Button>
          </Form.Item>
        </Form>
      ),
    },
  ];

  return (
    <Modal title="欢迎使用 Snowy" open={open} onCancel={onClose} footer={null} width={400}>
      <Tabs activeKey={activeTab} onChange={setActiveTab} items={tabItems} centered />
    </Modal>
  );
}
