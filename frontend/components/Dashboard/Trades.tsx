'use client';
import React from 'react';
import { Trade } from '../../types';
import { Clock, History } from 'lucide-react';

interface TradesProps {
  trades: Trade[];
}

export const Trades: React.FC<TradesProps> = ({ trades }) => {
  return (
    <section className="glass-card rounded-2xl p-0 shadow-2xl overflow-hidden flex flex-col h-[400px] lg:h-full">
      <div className="p-4 border-b border-border flex justify-between items-center bg-card/30">
        <h3 className="text-foreground font-black text-xs uppercase tracking-widest flex items-center gap-2">
          <History className="w-3.5 h-3.5 text-indigo-400" />
          Recent Activity
        </h3>
        <span className="text-[9px] font-black text-muted-foreground uppercase opacity-60">Real-time Feed</span>
      </div>

      <div className="flex-1 overflow-hidden flex flex-col font-sans">
        <div className="grid grid-cols-[1.5fr_1fr_1fr] gap-2 text-[9px] uppercase font-black text-muted-foreground py-2 px-4 border-b border-border/50 bg-card/10">
          <span>Price</span>
          <span className="text-right">Amount</span>
          <span className="text-right">Time</span>
        </div>

        <div className="flex-1 overflow-y-auto custom-scrollbar">
          {trades.length === 0 && (
            <div className="h-full flex flex-col items-center justify-center text-muted-foreground/30 gap-4 py-20">
              <Clock className="w-10 h-10 opacity-10 animate-pulse" />
              <p className="text-[10px] font-black uppercase tracking-widest">Waiting for trades...</p>
            </div>
          )}
          {trades?.map((t, i) => {
            if (!t) return null;
            const timestamp = t.timestamp ? new Date(t.timestamp).toLocaleTimeString([], { hour12: false, hour: '2-digit', minute: '2-digit', second: '2-digit' }) : '---';
            
            return (
              <div 
                key={`${t.id || i}-${i}`} 
                className="grid grid-cols-[1.5fr_1fr_1fr] gap-2 items-center py-2 px-4 border-b border-border/20 last:border-0 hover:bg-white/5 transition-colors group animate-in slide-in-from-top-1 duration-300"
              >
                <span className={`font-mono font-bold text-[11px] tabular-nums truncate ${t.side === 'buy' ? 'text-buy' : 'text-sell'}`}>
                  {(t.price || 0).toLocaleString(undefined, { minimumFractionDigits: 2 })}
                </span>
                <span className="text-right text-foreground font-mono font-bold text-[11px] tabular-nums truncate">
                  {(t.quantity || 0).toLocaleString(undefined, { minimumFractionDigits: 4 })}
                </span>
                <span className="text-right text-muted-foreground font-mono text-[9px] tabular-nums opacity-60 group-hover:opacity-100 transition-opacity">
                  {timestamp}
                </span>
              </div>
            );
          })}
        </div>
      </div>
    </section>
  );
};