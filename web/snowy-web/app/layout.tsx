import type { Metadata } from 'next';
import { AntdRegistry } from '@ant-design/nextjs-registry';
import AppLayout from '@/components/layout/AppLayout';
import './globals.css';

export const metadata: Metadata = {
  title: 'Snowy - AIGC 学习平台',
  description: '面向高中生的知识检索、物理建模、生物建模学习平台',
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="zh-CN">
      <body>
        <AntdRegistry>
          <AppLayout>{children}</AppLayout>
        </AntdRegistry>
      </body>
    </html>
  );
}
