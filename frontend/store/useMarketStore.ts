import { create } from 'zustand';

import { OrderBookEntry, Trade } from '../types';

interface Candle {
  time: number; // unix timestamp
  open: number;
  high: number;
  low: number;
  close: number;
}

interface MarketState {
  bids: OrderBookEntry[];
  asks: OrderBookEntry[];
  trades: Trade[];
  isConnected: boolean;
  lastPrice: number;
  priceChange: 'up' | 'down' | 'neutral';
  latency: number;
  totalVolume: number;
  priceHistory: Candle[];

  setMarketData: (bids: OrderBookEntry[], asks: OrderBookEntry[]) => void;
  addTrades: (trades: Trade[]) => void;
  setConnected: (status: boolean) => void;
  setMetrics: (latency: number, volume: number) => void;
  updatePriceHistory: () => void;
}

export const useMarketStore = create<MarketState>((set, get) => ({
  bids: [],
  asks: [],
  trades: [],
  isConnected: false,
  lastPrice: 0,
  priceChange: 'neutral',
  latency: 0,
  totalVolume: 0,
  priceHistory: [],

  setMarketData: (bids, asks) => {
    const sortedBids = [...bids].sort((a, b) => b.price - a.price);
    const sortedAsks = [...asks].sort((a, b) => a.price - b.price);
    const newPrice = sortedBids[0]?.price || get().lastPrice;
    const oldPrice = get().lastPrice;

    set({
      bids: sortedBids,
      asks: sortedAsks,
      lastPrice: newPrice,
      priceChange: newPrice > oldPrice ? 'up' : newPrice < oldPrice ? 'down' : 'neutral',
    });
  },

  addTrades: (newTrades) =>
    set((state) => {
      const tradesWithMeta = newTrades.map((t) => ({
        ...t,
        timestamp: t.timestamp || Date.now(),
        side: t.side || 'buy',
      }));
      return {
        trades: [...tradesWithMeta, ...state.trades].slice(0, 50),
      };
    }),

  setConnected: (status) => set({ isConnected: status }),
  setMetrics: (latency, volume) => set({ latency, totalVolume: volume }),

  updatePriceHistory: () => {
    const { lastPrice, priceHistory } = get();
    if (lastPrice <= 0) return;

    // We update every 2 seconds, but we want 1-minute (or 5s for live feel) resolution in the chart
    // For now, let's just make each point a new candle to keep it "bar like" as requested
    const now = Math.floor(Date.now() / 1000);
    const lastCandle = priceHistory[priceHistory.length - 1];

    // If we're within the same 5-second window, update the current candle
    const WINDOW = 5;
    const candleTime = Math.floor(now / WINDOW) * WINDOW;

    if (lastCandle && lastCandle.time === candleTime) {
      const updatedCandle = {
        ...lastCandle,
        high: Math.max(lastCandle.high, lastPrice),
        low: Math.min(lastCandle.low, lastPrice),
        close: lastPrice,
      };
      set({
        priceHistory: [...priceHistory.slice(0, -1), updatedCandle],
      });
    } else {
      // New candle
      const newCandle = {
        time: candleTime,
        open: lastCandle ? lastCandle.close : lastPrice,
        high: lastPrice,
        low: lastPrice,
        close: lastPrice,
      };
      set({
        priceHistory: [...priceHistory, newCandle].slice(-200),
      });
    }
  },
}));
