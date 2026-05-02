'use client';
import React, { useMemo } from 'react';
import {
  Area,
  AreaChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';

import { OrderBookSnapshot } from '../../types';

interface DepthChartProps {
  data: OrderBookSnapshot;
}

export const DepthChart: React.FC<DepthChartProps> = ({ data }) => {
  const chartData = useMemo(() => {
    const points: Array<{ price: number; bidVolume?: number; askVolume?: number }> = [];

    let currentBidTotal = 0;
    const sortedBids = [...data.bids].sort((a, b) => b.price - a.price);
    sortedBids.forEach((b) => {
      currentBidTotal += b.total_vol;
      points.push({ price: b.price, bidVolume: currentBidTotal });
    });

    let currentAskTotal = 0;
    const sortedAsks = [...data.asks].sort((a, b) => a.price - b.price);
    sortedAsks.forEach((a) => {
      currentAskTotal += a.total_vol;
      points.push({ price: a.price, askVolume: currentAskTotal });
    });

    // Add mid-price point to ensure chart starts/ends in the middle
    if (sortedBids.length > 0 && sortedAsks.length > 0) {
      const mid = (sortedBids[0].price + sortedAsks[0].price) / 2;
      points.push({ price: mid, bidVolume: 0, askVolume: 0 });
    }

    return points.sort((a, b) => a.price - b.price);
  }, [data]);

  return (
    <div className="glass-card rounded-2xl p-4 lg:p-6 min-h-[400px] lg:h-full shadow-2xl flex flex-col relative overflow-hidden group">
      <div className="absolute top-0 right-0 w-32 h-32 bg-primary/5 blur-3xl rounded-full -mr-16 -mt-16 pointer-events-none" />

      <div className="flex flex-col sm:flex-row sm:items-center justify-between mb-8 z-10 gap-4">
        <h3 className="text-foreground font-black text-xs uppercase tracking-widest flex items-center gap-2">
          <span className="w-1.5 h-1.5 rounded-full bg-indigo-500 shadow-[0_0_8px_rgba(99,102,241,0.5)]" />
          Market Depth
        </h3>
        <div className="flex gap-4 text-[9px] font-black uppercase tracking-widest text-muted-foreground">
          <span className="flex items-center gap-2">
            <span className="w-3 h-1 rounded-full bg-buy" /> BIDS
          </span>
          <span className="flex items-center gap-2">
            <span className="w-3 h-1 rounded-full bg-sell" /> ASKS
          </span>
        </div>
      </div>

      <div className="h-[350px] lg:flex-1 w-full min-h-0 bg-background/5 rounded-xl border border-border/30 overflow-hidden">
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={chartData} margin={{ top: 20, right: 0, left: -20, bottom: 0 }}>
            <defs>
              <linearGradient id="colorBid" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="#10b981" stopOpacity={0.6} />
                <stop offset="95%" stopColor="#10b981" stopOpacity={0} />
              </linearGradient>
              <linearGradient id="colorAsk" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="#f43f5e" stopOpacity={0.6} />
                <stop offset="95%" stopColor="#f43f5e" stopOpacity={0} />
              </linearGradient>
            </defs>
            <CartesianGrid strokeDasharray="3 3" stroke="#222" vertical={false} />
            <XAxis
              dataKey="price"
              type="number"
              domain={['auto', 'auto']}
              padding={{ left: 20, right: 20 }}
              stroke="hsl(var(--muted-foreground) / 0.5)"
              fontSize={10}
              tickLine={false}
              axisLine={false}
              tickFormatter={(val) => val.toLocaleString()}
            />
            <YAxis
              stroke="hsl(var(--muted-foreground) / 0.5)"
              fontSize={10}
              tickLine={false}
              axisLine={false}
              tickFormatter={(val) =>
                val >= 1000 ? `${(val / 1000).toFixed(1)}k` : val.toLocaleString()
              }
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
                boxShadow: '0 10px 15px -3px rgba(0, 0, 0, 0.5)',
              }}
              cursor={{ stroke: 'hsl(var(--primary) / 0.2)', strokeWidth: 1 }}
              formatter={(value: unknown) => [
                typeof value === 'number' ? value.toLocaleString() : String(value),
                'Volume',
              ]}
              labelFormatter={(label: unknown) =>
                `Price: ${typeof label === 'number' ? label.toLocaleString() : String(label)}`
              }
            />
            <Area
              type="stepAfter"
              dataKey="bidVolume"
              stroke="#10b981"
              strokeWidth={2}
              fillOpacity={1}
              fill="url(#colorBid)"
              isAnimationActive={false}
              connectNulls={true}
            />
            <Area
              type="stepBefore"
              dataKey="askVolume"
              stroke="#f43f5e"
              strokeWidth={2}
              fillOpacity={1}
              fill="url(#colorAsk)"
              isAnimationActive={false}
              connectNulls={true}
            />
          </AreaChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
};
