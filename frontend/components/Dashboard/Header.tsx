'use client';
import { BarChart3, History, LayoutDashboard, LogOut, Plus, Settings, Wallet } from 'lucide-react';
import Image from 'next/image';
import React, { useState } from 'react';

import { api } from '../../api/client';
import { useAuth } from '../../context/AuthContext';
import { useAccountStore } from '../../store/useAccountStore';
import { useToast } from '../Toast';

interface HeaderProps {
  isConnected: boolean;
}

export const Header: React.FC<HeaderProps> = ({ isConnected }) => {
  const { logout } = useAuth();
  const { balances, fetchBalances } = useAccountStore();
  const { addToast } = useToast();
  const [showBalances, setShowBalances] = useState(false);
  const [isDepositing, setIsDepositing] = useState(false);

  const handleQuickDeposit = async () => {
    setIsDepositing(true);
    try {
      // Quick deposit for testing (1M USD, 1M BTC)
      await api.post('/deposit', { asset: 'USD', amount: 1000000 * 1e8 });
      await api.post('/deposit', { asset: 'BTC', amount: 1000000 * 1e8 });
      await fetchBalances();
      addToast('Quick deposit successful: +1,000,000 USD, +1,000,000 BTC', 'success');
    } catch (err) {
      console.error('Deposit failed', err);
      addToast('Deposit failed', 'error');
    } finally {
      setIsDepositing(false);
    }
  };

  return (
    <nav className="glass border-b border-border/50 px-4 lg:px-8 py-3 flex justify-between items-center sticky top-0 z-50 shadow-2xl">
      <div className="flex items-center gap-4 lg:gap-10">
        <div className="flex items-center group cursor-pointer">
          <Image
            src="/LOGO.png"
            alt="OBMINNIK Logo"
            width={120}
            height={32}
            className="h-8 w-auto group-hover:scale-105 transition-transform duration-300"
          />
        </div>

        <div className="hidden lg:flex items-center gap-8 text-[11px] font-black uppercase tracking-[0.2em] text-muted-foreground/60">
          <button className="flex items-center gap-2 text-foreground transition-all">
            <LayoutDashboard className="w-3.5 h-3.5" />
            Exchange
          </button>
          <button className="flex items-center gap-2 hover:text-foreground transition-all">
            <BarChart3 className="w-3.5 h-3.5" />
            Markets
          </button>
          <button className="flex items-center gap-2 hover:text-foreground transition-all">
            <History className="w-3.5 h-3.5" />
            Orders
          </button>
        </div>
      </div>

      <div className="flex items-center gap-2 lg:gap-6">
        {/* Status Indicator */}
        <div className="flex items-center gap-3 px-4 py-2 bg-background/50 rounded-full border border-border shadow-inner group">
          <div className="relative">
            <div
              className={`w-2 h-2 rounded-full ${isConnected ? 'bg-buy animate-pulse' : 'bg-sell'} transition-colors shadow-[0_0_10px_currentcolor]`}
            />
            {isConnected && (
              <div className="absolute inset-0 w-2 h-2 rounded-full bg-buy animate-ping opacity-40" />
            )}
          </div>
          <span className="text-[10px] font-black uppercase tracking-widest text-foreground/80 hidden sm:inline">
            {isConnected ? 'Network Live' : 'Disconnected'}
          </span>
        </div>

        {/* Balance Dropdown */}
        <div className="relative">
          <button
            onClick={() => setShowBalances(!showBalances)}
            className="flex items-center gap-3 px-4 py-2 bg-background/50 rounded-xl border border-border shadow-inner hover:border-primary/50 transition-all group"
          >
            <Wallet className="w-4 h-4 text-primary group-hover:scale-110 transition-transform" />
            <span className="text-[10px] font-black uppercase tracking-widest text-foreground/80 hidden lg:inline">
              Portfolio
            </span>
          </button>

          {showBalances && (
            <div className="absolute top-full right-0 mt-4 w-64 glass-card rounded-2xl p-4 shadow-2xl border border-border/50 animate-in fade-in slide-in-from-top-2 z-[60]">
              <div className="space-y-4">
                <h4 className="text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground/60 mb-2">
                  Asset Summary
                </h4>
                {balances.length === 0 ? (
                  <p className="text-[10px] text-muted-foreground text-center py-2 italic">
                    No balances found
                  </p>
                ) : (
                  balances.map((b) => (
                    <div
                      key={b.asset_symbol}
                      className="space-y-1.5 p-2 rounded-lg bg-background/30 border border-border/20"
                    >
                      <div className="flex justify-between items-center">
                        <span className="text-[11px] font-black text-foreground">
                          {b.asset_symbol}
                        </span>
                        <span className="text-[11px] font-mono font-bold text-buy">
                          {(b.available / 1e8).toLocaleString()}
                        </span>
                      </div>
                      <div className="flex justify-between items-center text-[9px] text-muted-foreground/60 font-bold uppercase tracking-tight">
                        <span>Locked</span>
                        <span>{(b.locked / 1e8).toLocaleString()}</span>
                      </div>
                    </div>
                  ))
                )}

                <button
                  onClick={handleQuickDeposit}
                  disabled={isDepositing}
                  className="w-full mt-2 flex items-center justify-center gap-2 py-3 rounded-xl bg-primary text-white font-black text-[10px] uppercase tracking-widest shadow-lg shadow-primary/20 hover:bg-primary/90 transition-all active:scale-95 disabled:opacity-50"
                >
                  {isDepositing ? (
                    <div className="w-3 h-3 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                  ) : (
                    <Plus className="w-3 h-3" />
                  )}
                  Quick Deposit
                </button>
              </div>
            </div>
          )}
        </div>

        <button className="p-2 text-muted-foreground hover:text-foreground transition-colors">
          <Settings className="w-4 h-4" />
        </button>

        <div className="h-8 w-[1px] bg-border/50 mx-2" />

        <button
          onClick={logout}
          className="flex items-center gap-3 px-4 py-2 rounded-xl bg-destructive/10 text-destructive border border-destructive/20 hover:bg-destructive hover:text-white transition-all duration-300 group shadow-lg shadow-destructive/10"
        >
          <span className="text-[11px] font-black uppercase tracking-widest hidden sm:inline">
            Sign Out
          </span>
          <LogOut className="w-4 h-4 group-hover:translate-x-1 transition-transform" />
        </button>
      </div>
    </nav>
  );
};
