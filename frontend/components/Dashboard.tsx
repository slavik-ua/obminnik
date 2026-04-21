'use client';
import React, { useEffect, useState } from 'react';
import { useAuth } from '../context/AuthContext';
import { useWebSocket } from '../hooks/useWebSocket';
import { useMarketStore } from '../store/useMarketStore';
import { api } from '../api/client';
import { OrderBookSnapshot } from '../types';

import { Header } from './Dashboard/Header';
import { OrderForm } from './Dashboard/OrderForm';
import { OrderBook } from './Dashboard/OrderBook';
import { DepthChart } from './Dashboard/DepthChart';
import { Trades } from './Dashboard/Trades';
import { PriceChart } from './Dashboard/PriceChart';

export const Dashboard: React.FC = () => {
  const { token } = useAuth();
  const { isConnected } = useWebSocket(token);
  const [activeTab, setActiveTab] = useState<'price' | 'depth'>('price');
  
  const { bids, asks, trades, setMarketData, lastPrice, setMetrics, latency, totalVolume, updatePriceHistory } = useMarketStore();
  const apiURL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8000';

  // Initial Data Fetch
  useEffect(() => {
    const fetchInitialData = async () => {
      try {
        const snapshot = await api.get<OrderBookSnapshot>('/orderbook');
        setMarketData(snapshot.bids, snapshot.asks);
      } catch (err) {
        console.error("Failed to fetch initial orderbook", err);
      }
    };

    if (token) {
      fetchInitialData();
    }
  }, [token, setMarketData]);

  // Metrics & Price History Polling
  useEffect(() => {
    const fetchMetrics = async () => {
      if (!token) return;
      try {
        const response = await fetch(`${apiURL}/metrics`);
        const text = await response.text();
        
        const latencyMatch = text.match(/exchange_matching_engine_latency_seconds_sum ([\d.e-]+)/);
        const countMatch = text.match(/exchange_matching_engine_latency_seconds_count (\d+)/);
        const volumeMatch = text.match(/exchange_trades_total (\d+)/);
        
        if (latencyMatch && countMatch) {
          const sum = parseFloat(latencyMatch[1]);
          const count = parseInt(countMatch[1]);
          const avgLatency = count > 0 ? (sum / count) * 1000 : 0; 
          setMetrics(avgLatency, volumeMatch ? parseInt(volumeMatch[1]) : 0);
        }
      } catch (err) {
        console.error("Metrics ping failed", err);
      }
    };

    const interval = setInterval(fetchMetrics, 5000);
    const historyInterval = setInterval(() => {
      updatePriceHistory();
    }, 2000);
    
    fetchMetrics();
    updatePriceHistory();

    return () => {
      clearInterval(interval);
      clearInterval(historyInterval);
    };
  }, [token, apiURL, setMetrics, updatePriceHistory]);

  return (
    <div className="h-screen bg-background flex flex-col font-sans selection:bg-primary/20 overflow-hidden">
      <Header isConnected={isConnected} />
      
      <main className="flex-1 p-4 lg:p-6 grid grid-cols-1 lg:grid-cols-12 gap-6 max-w-[1920px] mx-auto w-full overflow-hidden min-h-0">
        
        {/* Left Column: Execution & Portfolio (2/12) */}
        <div className="lg:col-span-2 space-y-6 flex flex-col min-h-0">
          <OrderForm />
          <div className="flex-1 min-h-0">
            <Trades trades={trades} />
          </div>
        </div>

        {/* Center Column: Charts & Analysis (8/12) */}
        <div className="lg:col-span-8 space-y-6 flex flex-col min-h-0">
          <div className="flex-1 min-h-0 flex flex-col relative">
             <div className="absolute top-4 left-1/2 -translate-x-1/2 z-20 flex bg-background/50 backdrop-blur-md p-1 rounded-full border border-border/50 shadow-xl">
               <button 
                 onClick={() => setActiveTab('price')}
                 className={`px-6 py-1.5 rounded-full text-[10px] font-black uppercase tracking-widest transition-all duration-300 ${activeTab === 'price' ? 'bg-primary text-white shadow-lg shadow-primary/20' : 'text-muted-foreground hover:text-white'}`}
               >
                 Price
               </button>
               <button 
                 onClick={() => setActiveTab('depth')}
                 className={`px-6 py-1.5 rounded-full text-[10px] font-black uppercase tracking-widest transition-all duration-300 ${activeTab === 'depth' ? 'bg-primary text-white shadow-lg shadow-primary/20' : 'text-muted-foreground hover:text-white'}`}
               >
                 Depth
               </button>
             </div>

             <div className="flex-1 min-h-0">
                {activeTab === 'price' ? <PriceChart /> : <DepthChart data={{ bids, asks }} />}
             </div>
          </div>
          
          {/* Market Intelligence Bar */}
          <div className="grid grid-cols-3 gap-4">
            <div className="glass-card p-4 rounded-2xl relative overflow-hidden group">
              <div className="absolute inset-0 bg-glow-buy opacity-0 group-hover:opacity-100 transition-opacity" />
              <p className="text-[10px] text-muted-foreground uppercase font-black mb-1 tracking-widest">Trade Count</p>
              <p className="text-2xl font-mono text-foreground font-bold tabular-nums">
                {totalVolume.toLocaleString()} <span className="text-xs text-muted-foreground uppercase">Fills</span>
              </p>
            </div>
            
            <div className="glass-card p-4 rounded-2xl relative overflow-hidden group">
              <div className="absolute inset-0 bg-glow-buy opacity-0 group-hover:opacity-100 transition-opacity" />
              <p className="text-[10px] text-muted-foreground uppercase font-black mb-1 tracking-widest">Market Price</p>
              <p className="text-2xl font-mono text-buy font-bold tabular-nums">
                {lastPrice > 0 ? lastPrice.toLocaleString(undefined, { minimumFractionDigits: 2 }) : '---'}
              </p>
            </div>
            
            <div className="glass-card p-4 rounded-2xl relative overflow-hidden group">
              <div className="absolute inset-0 bg-indigo-500/5 opacity-0 group-hover:opacity-100 transition-opacity" />
              <p className="text-[10px] text-muted-foreground uppercase font-black mb-1 tracking-widest">Matching Latency</p>
              <p className="text-2xl font-mono text-indigo-400 font-bold tabular-nums">
                {latency > 0 ? `${latency.toFixed(3)}ms` : '< 0.001ms'}
              </p>
            </div>
          </div>
        </div>

        {/* Right Column: Order Book (2/12) */}
        <div className="lg:col-span-2 flex flex-col min-h-0">
          <OrderBook data={{ bids, asks }} />
        </div>

      </main>
      
      {/* Cinematic Status Bar */}
      <footer className="bg-card/50 backdrop-blur-md border-t border-border px-6 py-2.5 flex justify-between items-center text-[10px] text-muted-foreground font-bold tracking-tight">
        <div className="flex gap-6 items-center">
          <span className="flex items-center gap-2">
            <span className={`w-2 h-2 rounded-full ${isConnected ? 'bg-buy shadow-[0_0_8px_rgba(16,185,129,0.5)]' : 'bg-destructive'} animate-pulse`} />
            SERVER: {isConnected ? 'OPERATIONAL' : 'DISCONNECTED'}
          </span>
          <span className="opacity-40">|</span>
          <span className="flex items-center gap-2">
            <span className="w-2 h-2 rounded-full bg-buy" /> 
            MATCHING ENGINE: ACTIVE
          </span>
        </div>
        <div className="uppercase tracking-[0.2em] font-black opacity-80">
          OBMINNIK LOB v2.0 // INSTITUTIONAL GRADE
        </div>
      </footer>
    </div>
  );
};