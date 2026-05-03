import { useCallback, useEffect, useRef } from 'react';

import { useMarketStore } from '../store/useMarketStore';
import { WSMessage } from '../types';

const WS_BASE = process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8000/ws';

export const useWebSocket = (token: string | null) => {
  const setMarketData = useMarketStore((state) => state.setMarketData);
  const addTrades = useMarketStore((state) => state.addTrades);
  const setConnected = useMarketStore((state) => state.setConnected);
  const isConnected = useMarketStore((state) => state.isConnected);
  const socketRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const connectRef = useRef<(() => void) | null>(null);

  const connect = useCallback(() => {
    if (!token) return;

    if (socketRef.current?.readyState === WebSocket.OPEN) return;

    const socket = new WebSocket(`${WS_BASE}?token=${token}`);

    socket.onopen = () => {
      console.log('WebSocket Connected');
      setConnected(true);
    };

    socket.onmessage = (event) => {
      try {
        console.log('WS Received Raw:', event.data);
        const message: WSMessage = JSON.parse(event.data);
        console.log('WS Decoded:', message);
        if (message.type === 'ORDERBOOK_UPDATE') {
          console.log('Updating Orderbook:', message.payload);
          setMarketData(message.payload.bids, message.payload.asks);
        } else if (message.type === 'TRADES_EXECUTED') {
          console.log('Adding Trades:', message.payload);
          addTrades(message.payload);
        }
      } catch (err) {
        console.error('Failed to parse WebSocket message:', err);
      }
    };

    socket.onerror = (error) => {
      console.error('WebSocket Error:', error);
    };

    socket.onclose = () => {
      console.log('WebSocket Disconnected');
      setConnected(false);

      reconnectTimeoutRef.current = setTimeout(() => {
        console.log('Attempting to reconnect...');
        connectRef.current?.();
      }, 3000);
    };

    socketRef.current = socket;
  }, [token, setMarketData, addTrades, setConnected]);

  useEffect(() => {
    connectRef.current = connect;
  }, [connect]);

  useEffect(() => {
    connect();

    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
      if (socketRef.current) {
        socketRef.current.onclose = null;
        socketRef.current.close();
      }
    };
  }, [connect]);

  return { isConnected };
};
