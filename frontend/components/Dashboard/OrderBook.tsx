'use client';
import React, { useMemo, useEffect, useState } from 'react';
import { OrderBookSnapshot, OrderBookEntry } from '../../types';
import { useMarketStore } from '../../store/useMarketStore';

interface OrderBookProps {
  data: OrderBookSnapshot;
}

const OrderRow: React.FC<{ entry: OrderBookEntry; side: 'buy' | 'sell'; maxVol: number }> = ({ entry, side, maxVol }) => {
  const [flash, setFlash] = useState(false);

  useEffect(() => {
    // Use timeout to avoid synchronous setState in effect
    const flashTimer = setTimeout(() => setFlash(true), 0);
    const timer = setTimeout(() => setFlash(false), 500);
    return () => {
      clearTimeout(flashTimer);
      clearTimeout(timer);
    };
  }, [entry.price, entry.total_vol]);

  const barColor = side === 'buy' ? 'hsl(var(--buy) / 0.15)' : 'hsl(var(--sell) / 0.15)';
  const textColor = side === 'buy' ? 'text-buy' : 'text-sell';

  return (
    <div className={`grid grid-cols-3 gap-2 text-[11px] py-1 px-4 relative group transition-colors hover:bg-white/5 ${flash ? (side === 'buy' ? 'animate-flash-buy' : 'animate-flash-sell') : ''}`}>
      <div 
        className="absolute inset-y-0 right-0 pointer-events-none transition-all duration-500 ease-out" 
        style={{ width: `${(entry.total_vol / maxVol) * 100}%`, backgroundColor: barColor }} 
      />
      <span className={`${textColor} font-mono font-bold z-10 tabular-nums`}>{entry.price.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}</span>
      <span className="text-right text-foreground font-mono z-10 tabular-nums">{entry.total_vol.toLocaleString()}</span>
      <span className="text-right text-muted-foreground font-mono z-10 tabular-nums">{(entry.price * entry.total_vol).toLocaleString(undefined, { maximumFractionDigits: 0 })}</span>
    </div>
  );
};

export const OrderBook: React.FC<OrderBookProps> = ({ data }) => {
  const { lastPrice, priceChange } = useMarketStore();
  
  const maxVol = useMemo(() => {
    const allVols = [...data.bids, ...data.asks].map(e => e.total_vol);
    return allVols.length > 0 ? Math.max(...allVols) : 1;
  }, [data]);

  return (
    <section className="glass-card rounded-2xl overflow-hidden h-[500px] lg:h-full flex flex-col shadow-2xl">
      <div className="p-4 border-b border-border flex justify-between items-center bg-card/30">
        <h3 className="text-foreground font-black text-xs uppercase tracking-widest flex items-center gap-2">
          <span className="w-1.5 h-1.5 rounded-full bg-buy shadow-[0_0_8px_hsl(var(--buy)/0.5)]" />
          Order Book
        </h3>
        <div className="flex gap-4 text-[10px] font-black text-muted-foreground uppercase mr-4">
          <span>Price</span>
          <span>Size</span>
        </div>
      </div>
      
      <div className="flex-1 flex flex-col min-h-0 bg-background/20 font-sans">
        {/* Table Header */}
        <div className="grid grid-cols-3 text-[9px] uppercase font-black text-muted-foreground py-2 px-4 border-b border-border/50 bg-card/10">
          <span>Price (USD)</span>
          <span className="text-right">Amount</span>
          <span className="text-right">Total</span>
        </div>
        
        <div className="flex-1 overflow-hidden flex flex-col">
          {/* ASKS (Sells) */}
          <div className="flex flex-col flex-1 min-h-0 custom-scrollbar overflow-y-auto">
            <div className="flex-1" />
            {data.asks.length === 0 && <div className="text-center py-8 text-muted-foreground/30 text-[10px] uppercase font-bold tracking-widest">Awaiting Asks...</div>}
            {data.asks.slice().reverse().map((ask, i) => (
              <OrderRow key={`ask-${ask.price}-${i}`} entry={ask} side="sell" maxVol={maxVol} />
            ))}
          </div>

          {/* Market Spread Bar */}
          <div className="py-3 px-4 my-1 border-y border-border/50 flex items-center justify-between bg-card/40 backdrop-blur-sm relative overflow-hidden group">
            <div className={`absolute inset-0 opacity-10 transition-colors ${priceChange === 'up' ? 'bg-buy' : priceChange === 'down' ? 'bg-sell' : 'bg-transparent'}`} />
            <div className="flex items-center gap-3 z-10">
              <span className={`text-xl font-mono font-black tabular-nums transition-colors ${priceChange === 'up' ? 'text-buy' : priceChange === 'down' ? 'text-sell' : 'text-foreground'}`}>
                {lastPrice > 0 ? lastPrice.toLocaleString(undefined, { minimumFractionDigits: 2 }) : '---'}
              </span>
              {priceChange !== 'neutral' && (
                <span className={`text-[10px] font-black ${priceChange === 'up' ? 'text-buy' : 'text-sell'}`}>
                  {priceChange === 'up' ? '▲' : '▼'}
                </span>
              )}
            </div>
            <div className="text-right z-10">
              <span className="text-[10px] text-muted-foreground uppercase font-black tracking-tighter block leading-none">Spread</span>
              <span className="text-xs text-foreground font-mono font-bold">
                {data.asks.length && data.bids.length 
                  ? (Math.abs(data.asks[0].price - data.bids[0].price)).toLocaleString(undefined, { minimumFractionDigits: 2 }) 
                  : '0.00'}
              </span>
            </div>
          </div>

          {/* BIDS (Buys) */}
          <div className="flex flex-col flex-1 min-h-0 overflow-y-auto custom-scrollbar">
            {data.bids.length === 0 && <div className="text-center py-8 text-muted-foreground/30 text-[10px] uppercase font-bold tracking-widest">Awaiting Bids...</div>}
            {data.bids.map((bid, i) => (
              <OrderRow key={`bid-${bid.price}-${i}`} entry={bid} side="buy" maxVol={maxVol} />
            ))}
          </div>
        </div>
      </div>
    </section>
  );
};