/**
 * Snowy v7 · 协同 WS 客户端 hook
 *
 * 用法：
 *   const { status, presence, send, lastEvent } = useSessionWS(sessionId, { onEvent });
 *
 * 协议：与后端 internal/handler/ws.Event 对齐
 *   { type, session_id, client_id, user_id, payload, timestamp }
 *
 * 注意：单实例 hook 维护一条连接 + 自动重连（exp backoff，最多 5 次）。
 */

'use client';

import { useCallback, useEffect, useRef, useState } from 'react';

export interface WSEvent<P = unknown> {
  type: string;
  session_id: string;
  client_id: string;
  user_id?: string;
  payload?: P;
  timestamp: number;
}

export type WSStatus = 'idle' | 'connecting' | 'open' | 'closed' | 'error';

export interface UseSessionWSOptions<P = unknown> {
  enabled?: boolean;
  onEvent?: (evt: WSEvent<P>) => void;
  onOpen?: () => void;
  onClose?: () => void;
}

function buildURL(sessionId: string): string {
  if (typeof window === 'undefined') return '';
  const base = (process.env.NEXT_PUBLIC_API_BASE || '').replace(/^https?:/, '').replace(/\/$/, '');
  let host: string;
  let proto: string;
  if (base) {
    proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    host = base;
  } else {
    proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    host = `//${window.location.host}`;
  }
  return `${proto}${host}/api/v1/ws/session/${encodeURIComponent(sessionId)}`;
}

export function useSessionWS<P = unknown>(
  sessionId: string | null | undefined,
  opts: UseSessionWSOptions<P> = {},
) {
  const { enabled = true, onEvent, onOpen, onClose } = opts;
  const [status, setStatus] = useState<WSStatus>('idle');
  const [lastEvent, setLastEvent] = useState<WSEvent<P> | null>(null);
  const [presence, setPresence] = useState<string[]>([]);
  const sockRef = useRef<WebSocket | null>(null);
  const attemptRef = useRef(0);
  const onEventRef = useRef(onEvent);
  const onOpenRef = useRef(onOpen);
  const onCloseRef = useRef(onClose);

  useEffect(() => {
    onEventRef.current = onEvent;
    onOpenRef.current = onOpen;
    onCloseRef.current = onClose;
  });

  useEffect(() => {
    if (!enabled || !sessionId) return;
    let cancelled = false;
    let retryTimer: ReturnType<typeof setTimeout> | null = null;

    const connect = () => {
      if (cancelled) return;
      setStatus('connecting');
      const ws = new WebSocket(buildURL(sessionId));
      sockRef.current = ws;
      ws.onopen = () => {
        attemptRef.current = 0;
        setStatus('open');
        onOpenRef.current?.();
      };
      ws.onmessage = (e) => {
        try {
          const evt = JSON.parse(e.data) as WSEvent<P>;
          setLastEvent(evt);
          if (evt.type === 'presence.join') {
            setPresence((prev) => (prev.includes(evt.client_id) ? prev : [...prev, evt.client_id]));
          } else if (evt.type === 'presence.leave') {
            setPresence((prev) => prev.filter((c) => c !== evt.client_id));
          }
          onEventRef.current?.(evt);
        } catch {
          // ignore non-JSON frames
        }
      };
      ws.onerror = () => {
        setStatus('error');
      };
      ws.onclose = () => {
        sockRef.current = null;
        setStatus('closed');
        onCloseRef.current?.();
        if (cancelled) return;
        if (attemptRef.current >= 5) return;
        const delay = Math.min(1000 * 2 ** attemptRef.current, 15000);
        attemptRef.current += 1;
        retryTimer = setTimeout(connect, delay);
      };
    };

    connect();

    return () => {
      cancelled = true;
      if (retryTimer) clearTimeout(retryTimer);
      sockRef.current?.close();
      sockRef.current = null;
    };
  }, [enabled, sessionId]);

  const send = useCallback((type: string, payload?: P) => {
    const ws = sockRef.current;
    if (!ws || ws.readyState !== WebSocket.OPEN) return false;
    ws.send(JSON.stringify({ type, payload, timestamp: Date.now() }));
    return true;
  }, []);

  return { status, lastEvent, presence, send } as const;
}
