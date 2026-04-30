'use client';

import React from 'react';
import { Typography } from 'antd';
import katex from 'katex';

const { Text, Paragraph, Title } = Typography;

type MarkdownTextProps = {
  content: string;
};

type InlineToken =
  | { type: 'text'; value: string }
  | { type: 'strong'; value: string }
  | { type: 'code'; value: string }
  | { type: 'math'; value: string };

const BLOCK_MATH_PATTERN = /(\\\[[^]*?\\\]|\$\$[^]*?\$\$)/g;

function normalizeMarkdown(content: string): string {
  return content
    .replace(/\r\n/g, '\n')
    .replace(/\\\\\[/g, '\\[')
    .replace(/\\\\\]/g, '\\]')
    .replace(/\\\\\(/g, '\\(')
    .replace(/\\\\\)/g, '\\)')
    .replace(/\\\s+/g, ' ')
    .replace(/\n{3,}/g, '\n\n')
    .trim();
}

function stripHeadingPrefix(line: string): { level: 4 | 5; text: string } | null {
  const match = /^(#{2,4})\s+(.+)$/.exec(line.trim());
  if (!match) return null;
  return { level: match[1].length <= 2 ? 4 : 5, text: match[2].trim() };
}

function normalizeTexSource(tex: string): string {
  return tex.replace(/\\\\(?=[a-zA-Z])/g, '\\').trim();
}

function stripMathDelimiter(token: string): string {
  const trimmed = token.trim();
  if (trimmed.startsWith('\\(') && trimmed.endsWith('\\)')) return normalizeTexSource(trimmed.slice(2, -2));
  if (trimmed.startsWith('\\[') && trimmed.endsWith('\\]')) return normalizeTexSource(trimmed.slice(2, -2));
  if (trimmed.startsWith('$$') && trimmed.endsWith('$$')) return normalizeTexSource(trimmed.slice(2, -2));
  if (trimmed.startsWith('$') && trimmed.endsWith('$')) return normalizeTexSource(trimmed.slice(1, -1));
  return normalizeTexSource(trimmed);
}

function renderKatex(tex: string, displayMode: boolean): string | null {
  const source = tex.trim();
  if (!source) return null;

  try {
    return katex.renderToString(source, {
      displayMode,
      throwOnError: false,
      strict: false,
      trust: false,
      output: 'html',
    });
  } catch {
    return null;
  }
}

function MathNode({ tex, displayMode }: { tex: string; displayMode?: boolean }) {
  const html = renderKatex(tex, Boolean(displayMode));
  if (!html) return <Text code>{tex}</Text>;

  if (displayMode) {
    return (
      <div
        className="snowy-math snowy-math-block"
        dangerouslySetInnerHTML={{ __html: html }}
      />
    );
  }

  return (
    <span
      className="snowy-math snowy-math-inline"
      dangerouslySetInnerHTML={{ __html: html }}
    />
  );
}

function tokenizeInline(text: string): InlineToken[] {
  const tokens: InlineToken[] = [];
  const pattern = /(\\\([^]*?\\\)|\$[^$\n]+\$|\*\*[^*]+\*\*|`[^`]+`)/g;
  let last = 0;
  let match: RegExpExecArray | null;

  while ((match = pattern.exec(text)) !== null) {
    if (match.index > last) tokens.push({ type: 'text', value: text.slice(last, match.index) });
    const value = match[0];
    if (value.startsWith('**')) {
      tokens.push({ type: 'strong', value: value.slice(2, -2) });
    } else if (value.startsWith('`')) {
      tokens.push({ type: 'code', value: value.slice(1, -1) });
    } else {
      tokens.push({ type: 'math', value: stripMathDelimiter(value) });
    }
    last = match.index + value.length;
  }

  if (last < text.length) tokens.push({ type: 'text', value: text.slice(last) });
  return tokens;
}

function inlineParts(text: string): React.ReactNode[] {
  return tokenizeInline(text).map((token, index) => {
    const key = `${token.type}-${index}`;
    switch (token.type) {
      case 'strong':
        return <Text strong key={key}>{inlineParts(token.value)}</Text>;
      case 'code':
        return <Text code key={key}>{token.value}</Text>;
      case 'math':
        return <MathNode key={key} tex={token.value} />;
      default:
        return token.value;
    }
  });
}

function parseListItem(line: string): string | null {
  const unordered = /^[-*]\s+(.+)$/.exec(line.trim());
  if (unordered) return unordered[1].trim();
  const ordered = /^\d+[.)、]\s+(.+)$/.exec(line.trim());
  if (ordered) return ordered[1].trim();
  return null;
}

function isHorizontalRule(lines: string[]): boolean {
  return lines.length === 1 && /^-{3,}$/.test(lines[0].trim());
}

function renderParagraph(lines: string[], key: string) {
  const text = lines.join('\n').trim();
  if (!text) return null;
  if (isHorizontalRule(lines)) return <div key={key} className="snowy-markdown-divider" />;

  const heading = stripHeadingPrefix(text);
  if (heading) {
    return <Title key={key} level={heading.level} style={{ marginTop: 12, marginBottom: 8 }}>{inlineParts(heading.text)}</Title>;
  }

  const listItems = lines.map(parseListItem);
  if (listItems.every(Boolean)) {
    return (
      <ul key={key} style={{ margin: '8px 0 8px 20px', padding: 0, lineHeight: 1.9 }}>
        {listItems.map((item, index) => <li key={index}>{inlineParts(item || '')}</li>)}
      </ul>
    );
  }

  return (
    <Paragraph key={key} style={{ fontSize: 15, lineHeight: 1.9, marginBottom: 10, whiteSpace: 'pre-wrap' }}>
      {inlineParts(text)}
    </Paragraph>
  );
}

function renderTextBlocks(text: string, keyPrefix: string): React.ReactNode[] {
  const blocks: React.ReactNode[] = [];
  let current: string[] = [];

  text.split('\n').forEach((line, index) => {
    if (!line.trim()) {
      if (current.length) {
        blocks.push(renderParagraph(current, `${keyPrefix}-block-${index}`));
        current = [];
      }
      return;
    }

    const heading = stripHeadingPrefix(line);
    if (heading) {
      if (current.length) {
        blocks.push(renderParagraph(current, `${keyPrefix}-block-before-${index}`));
        current = [];
      }
      blocks.push(<Title key={`${keyPrefix}-heading-${index}`} level={heading.level} style={{ marginTop: 12, marginBottom: 8 }}>{inlineParts(heading.text)}</Title>);
      return;
    }

    current.push(line);
  });

  if (current.length) blocks.push(renderParagraph(current, `${keyPrefix}-block-final`));
  return blocks;
}

function renderWithBlockMath(content: string): React.ReactNode[] {
  const blocks: React.ReactNode[] = [];
  let last = 0;
  let match: RegExpExecArray | null;

  while ((match = BLOCK_MATH_PATTERN.exec(content)) !== null) {
    if (match.index > last) {
      blocks.push(...renderTextBlocks(content.slice(last, match.index), `text-${last}`));
    }
    blocks.push(
      <MathNode
        key={`math-block-${match.index}`}
        tex={stripMathDelimiter(match[0])}
        displayMode
      />,
    );
    last = match.index + match[0].length;
  }

  if (last < content.length) blocks.push(...renderTextBlocks(content.slice(last), `text-${last}`));
  return blocks;
}

export default function MarkdownText({ content }: MarkdownTextProps) {
  const normalized = normalizeMarkdown(content || '');
  if (!normalized) return null;

  return <div className="snowy-markdown" style={{ overflowWrap: 'anywhere' }}>{renderWithBlockMath(normalized)}</div>;
}
