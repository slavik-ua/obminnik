'use client';
import { ShieldCheck, TrendingDown, TrendingUp, Zap } from 'lucide-react';
import React, { useState } from 'react';

import { api } from '../../api/client';
import { useAccountStore } from '../../store/useAccountStore';
import { OrderSide } from '../../types';
import { useToast } from '../Toast';

export const OrderForm: React.FC = () => {
  const [side, setSide] = useState<OrderSide>('buy');
  const [price, setPrice] = useState('');
  const [quantity, setQuantity] = useState('');
  const [loading, setLoading] = useState(false);
  const [lastError, setLastError] = useState<string | null>(null);

  const { getBalance, fetchBalances } = useAccountStore();
  const { addToast } = useToast();
  const assetToCheck = side === 'buy' ? 'USD' : 'BTC';
  const balance = getBalance(assetToCheck);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!price || !quantity) return;

    setLoading(true);
    setLastError(null);
    try {
      await api.post('/order', {
        price: Math.round(parseFloat(price) * 1e8),
        quantity: Math.round(parseFloat(quantity) * 1e8),
        side: side,
      });
      setPrice('');
      setQuantity('');
      fetchBalances(); // Refresh balances after order
      addToast(`Limit ${side.toUpperCase()} order placed at ${price}`, 'success');
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'An error occurred';
      setLastError(message);
      addToast(message, 'error');
    } finally {
      setLoading(false);
    }
  };

  return (
    <section className="glass-card rounded-2xl p-6 shadow-2xl relative overflow-hidden group">
      <div className="absolute top-0 left-0 w-full h-1 bg-gradient-to-r from-transparent via-primary/20 to-transparent opacity-0 group-hover:opacity-100 transition-opacity" />

      <div className="flex items-center justify-between mb-8">
        <h3 className="text-foreground font-black text-xs uppercase tracking-widest flex items-center gap-2">
          <Zap className="w-3.5 h-3.5 text-indigo-400 fill-indigo-400 animate-pulse" />
          Quick Execution
        </h3>
        <div className="flex items-center gap-1.5 px-2 py-1 rounded-full bg-buy/10 border border-buy/20">
          <ShieldCheck className="w-3 h-3 text-buy" />
          <span className="text-[9px] text-buy font-black uppercase">Secured</span>
        </div>
      </div>

      {/* Modern Side Selector */}
      <div className="flex p-1 bg-background/50 backdrop-blur-md rounded-xl mb-8 border border-border shadow-inner">
        <button
          onClick={() => setSide('buy')}
          className={`flex-1 flex items-center justify-center gap-2 py-3 rounded-lg font-black text-[11px] uppercase tracking-wider transition-all duration-300 ${
            side === 'buy'
              ? 'bg-buy text-white shadow-lg shadow-buy/20 scale-[1.02]'
              : 'text-muted-foreground hover:text-foreground'
          }`}
        >
          <TrendingUp className="w-3.5 h-3.5" /> BUY
        </button>
        <button
          onClick={() => setSide('sell')}
          className={`flex-1 flex items-center justify-center gap-2 py-3 rounded-lg font-black text-[11px] uppercase tracking-wider transition-all duration-300 ${
            side === 'sell'
              ? 'bg-sell text-white shadow-lg shadow-sell/20 scale-[1.02]'
              : 'text-muted-foreground hover:text-foreground'
          }`}
        >
          <TrendingDown className="w-3.5 h-3.5" /> SELL
        </button>
      </div>

      <form onSubmit={handleSubmit} className="space-y-6">
        <div className="space-y-2">
          <div className="flex justify-between items-end">
            <label className="text-[9px] text-muted-foreground uppercase font-black tracking-widest">
              Limit Price
            </label>
            <span className="text-[9px] text-muted-foreground/60 font-mono font-bold">USD</span>
          </div>
          <div className="relative group">
            <input
              type="number"
              step="1"
              required
              placeholder="0"
              className="w-full bg-background/40 border border-border rounded-xl px-4 py-4 outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary text-foreground font-mono font-bold transition-all placeholder:text-muted-foreground/30 text-lg tabular-nums"
              value={price}
              onChange={(e) => setPrice(e.target.value)}
            />
          </div>
        </div>

        <div className="space-y-2">
          <div className="flex justify-between items-end">
            <label className="text-[9px] text-muted-foreground uppercase font-black tracking-widest">
              Quantity
            </label>
            <span className="text-[9px] text-muted-foreground/60 font-mono font-bold">UNITS</span>
          </div>
          <input
            type="number"
            step="1"
            required
            placeholder="0"
            className="w-full bg-background/40 border border-border rounded-xl px-4 py-4 outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary text-foreground font-mono font-bold transition-all placeholder:text-muted-foreground/30 text-lg tabular-nums"
            value={quantity}
            onChange={(e) => setQuantity(e.target.value)}
          />
        </div>

        {/* Balance Display */}
        <div className="bg-background/20 rounded-xl p-3 border border-border/50 space-y-2">
          <div className="flex justify-between items-center text-[10px] font-black uppercase tracking-widest text-muted-foreground/60">
            <span>{assetToCheck} Balance</span>
            <span className="text-foreground">
              {((balance?.available || 0) / 1e8).toLocaleString()} {assetToCheck}
            </span>
          </div>
          <div className="h-[1px] bg-border/20 w-full" />
          <div className="flex justify-between items-center text-[9px] font-bold text-muted-foreground/40">
            <span>Locked in Orders</span>
            <span>
              {((balance?.locked || 0) / 1e8).toLocaleString()} {assetToCheck}
            </span>
          </div>
        </div>

        {lastError && (
          <div className="text-[10px] text-sell font-bold bg-sell/10 p-3 rounded-xl border border-sell/20 animate-in fade-in slide-in-from-top-1">
            ⚠️ {lastError}
          </div>
        )}

        <div className="pt-2">
          <button
            type="submit"
            disabled={loading}
            className={`w-full py-5 rounded-xl font-black text-xs uppercase tracking-[0.2em] shadow-2xl transition-all active:scale-[0.97] disabled:opacity-50 disabled:cursor-not-allowed group relative overflow-hidden ${
              side === 'buy'
                ? 'bg-buy hover:bg-buy/90 shadow-buy/20'
                : 'bg-sell hover:bg-sell/90 shadow-sell/20'
            } text-white`}
          >
            <div className="absolute inset-0 bg-white/20 translate-x-[-100%] group-hover:translate-x-[100%] transition-transform duration-700" />
            <span className="relative z-10 flex items-center justify-center gap-2">
              {loading ? (
                <>
                  <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                  Processing...
                </>
              ) : (
                `${side} order`
              )}
            </span>
          </button>
        </div>
      </form>
    </section>
  );
};
