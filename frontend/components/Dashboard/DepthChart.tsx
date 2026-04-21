'use client';
import React, { useMemo } from 'react';
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';
import { OrderBookSnapshot } from '../../types';

interface DepthChartProps {
  data: OrderBookSnapshot;
}

export const DepthChart: React.FC<DepthChartProps> = ({ data }) => {
  const chartData = useMemo(() => {
    const bids: Array<{ price: number; volume: number; side: string }> = [];
    let currentBidTotal = 0;
    const sortedBids = [...data.bids].sort((a, b) => b.price - a.price);
    
    sortedBids.forEach(b => {
      currentBidTotal += b.total_vol;
      bids.push({ price: b.price, volume: currentBidTotal, side: 'bid' });
    });
    bids.reverse();

    const asks: Array<{ price: number; volume: number; side: string }> = [];
    let currentAskTotal = 0;
    const sortedAsks = [...data.asks].sort((a, b) => a.price - b.price);
    
    sortedAsks.forEach(a => {
      currentAskTotal += a.total_vol;
      asks.push({ price: a.price, volume: currentAskTotal, side: 'ask' });
    });

    // Add mid-price points to close the gap visually
    if (bids.length > 0 && asks.length > 0) {
      const mid = (bids[bids.length - 1].price + asks[0].price) / 2;
      return [
        ...bids,
        { price: mid, volume: 0, side: 'bid' },
        { price: mid, volume: 0, side: 'ask' },
        ...asks
      ];
    }

    return [...bids, ...asks];
  }, [data]);

  return (
    <div className="glass-card rounded-2xl p-6 h-full shadow-2xl flex flex-col relative overflow-hidden group">
      <div className="absolute top-0 right-0 w-32 h-32 bg-primary/5 blur-3xl rounded-full -mr-16 -mt-16 pointer-events-none" />
      
      <div className="flex items-center justify-between mb-8 z-10">
        <h3 className="text-foreground font-black text-xs uppercase tracking-widest flex items-center gap-2">
          <span className="w-1.5 h-1.5 rounded-full bg-indigo-500 shadow-[0_0_8px_rgba(99,102,241,0.5)]" />
          Market Depth
        </h3>
        <div className="flex gap-6 text-[10px] font-black uppercase tracking-widest text-muted-foreground">
          <span className="flex items-center gap-2 transition-colors hover:text-buy">
            <span className="w-3 h-1 rounded-full bg-buy/50" /> BIDS
          </span>
          <span className="flex items-center gap-2 transition-colors hover:text-sell">
            <span className="w-3 h-1 rounded-full bg-sell/50" /> ASKS
          </span>
        </div>
      </div>
      
      <div className="flex-1 w-full min-h-0 bg-background/5 rounded-xl border border-border/30 overflow-hidden">
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={chartData} margin={{ top: 20, right: 0, left: -20, bottom: 0 }}>
            <defs>
              <linearGradient id="colorBid" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="#10b981" stopOpacity={0.3}/>
                <stop offset="95%" stopColor="#10b981" stopOpacity={0}/>
              </linearGradient>
              <linearGradient id="colorAsk" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="#f43f5e" stopOpacity={0.3}/>
                <stop offset="95%" stopColor="#f43f5e" stopOpacity={0}/>
              </linearGradient>
            </defs>
            <CartesianGrid strokeDasharray="3 3" stroke="#222" vertical={false} />
            <XAxis 
              dataKey="price" 
              stroke="hsl(var(--muted-foreground) / 0.5)" 
              fontSize={10} 
              tickLine={false} 
              axisLine={false}
              minTickGap={60}
              tickFormatter={(val) => val.toLocaleString()}
            />
            <YAxis 
              stroke="hsl(var(--muted-foreground) / 0.5)" 
              fontSize={10} 
              tickLine={false} 
              axisLine={false} 
              tickFormatter={(val) => val >= 1000 ? `${(val/1000).toFixed(1)}k` : val}
            />
            <Tooltip 
              contentStyle={{ 
                backgroundColor: 'hsl(var(--card) / 0.9)', 
                backdropFilter: 'blur(8px)',
                border: '1px solid hsl(var(--border) / 0.5)', 
                borderRadius: '12px', 
                fontSize: '11px',
                fontWeight: '900',
                color: 'hsl(var(--foreground))',
                boxShadow: '0 10px 15px -3px rgba(0, 0, 0, 0.5)'
              }}
              cursor={{ stroke: 'hsl(var(--primary) / 0.2)', strokeWidth: 1 }}
            />
            <Area 
              type="stepAfter" 
              dataKey="volume" 
              stroke="#10b981" 
              strokeWidth={2}
              fillOpacity={1} 
              fill="url(#colorBid)" 
              data={chartData.filter(d => d.side === 'bid')}
              isAnimationActive={true}
              animationDuration={500}
              activeDot={{ r: 4, fill: '#10b981', stroke: '#fff', strokeWidth: 2 }}
            />
            <Area 
              type="stepAfter" 
              dataKey="volume" 
              stroke="#f43f5e" 
              strokeWidth={2}
              fillOpacity={1} 
              fill="url(#colorAsk)" 
              data={chartData.filter(d => d.side === 'ask')}
              isAnimationActive={true}
              animationDuration={500}
              activeDot={{ r: 4, fill: '#f43f5e', stroke: '#fff', strokeWidth: 2 }}
            />
          </AreaChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
};