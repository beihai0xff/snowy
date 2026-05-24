import type { Metadata } from 'next';
import { AntdRegistry } from '@ant-design/nextjs-registry';
import AppLayout from '@/components/layout/AppLayout';
import 'katex/dist/katex.min.css';
import './globals.css';

export const metadata: Metadata = {
  metadataBase: new URL(process.env.NEXT_PUBLIC_SITE_URL ?? 'https://snowy.local'),
  title: 'Snowy · AI 学习工具',
  description: '面向高中生的 AI 学习工具：用大白话问问题、用动画看公式、用图谱理流程。',
  openGraph: {
    title: 'Snowy · AI 学习工具',
    description: '面向高中生的 AI 学习工具：用大白话问问题、用动画看公式、用图谱理流程。',
    type: 'website',
    siteName: 'Snowy',
    locale: 'zh_CN',
    images: [
      {
        url: '/brand/og-image.png',
        width: 1200,
        height: 630,
        alt: 'Snowy — 面向高中生的 AI 学习工具',
      },
    ],
  },
  twitter: {
    card: 'summary_large_image',
    title: 'Snowy · AI 学习工具',
    description: '面向高中生的 AI 学习工具：用大白话问问题、用动画看公式、用图谱理流程。',
    images: ['/brand/og-image.png'],
  },
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
