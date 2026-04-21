'use client';
import React from 'react';
import { LogOut, Activity, LayoutDashboard, BarChart3, History, Settings } from 'lucide-react';
import { useAuth } from '../../context/AuthContext';

interface HeaderProps {
  isConnected: boolean;
}

export const Header: React.FC<HeaderProps> = ({ isConnected }) => {
  const { logout } = useAuth();

  return (
    <nav className="glass border-b border-border/50 px-8 py-3 flex justify-between items-center sticky top-0 z-50 shadow-2xl">
      <div className="flex items-center gap-10">
        <div className="flex items-center group cursor-pointer">
          <img 
            src="/LOGO.png" 
            alt="OBMINNIK Logo" 
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

      <div className="flex items-center gap-6">
        {/* Status Indicator */}
        <div className="flex items-center gap-3 px-4 py-2 bg-background/50 rounded-full border border-border shadow-inner group">
          <div className="relative">
             <div className={`w-2 h-2 rounded-full ${isConnected ? 'bg-buy animate-pulse' : 'bg-sell'} transition-colors shadow-[0_0_10px_currentcolor]`} />
             {isConnected && (
               <div className="absolute inset-0 w-2 h-2 rounded-full bg-buy animate-ping opacity-40" />
             )}
          </div>
          <span className="text-[10px] font-black uppercase tracking-widest text-foreground/80">
            {isConnected ? 'Network Live' : 'Disconnected'}
          </span>
        </div>

        <button className="p-2 text-muted-foreground hover:text-foreground transition-colors">
          <Settings className="w-4 h-4" />
        </button>

        <div className="h-8 w-[1px] bg-border/50 mx-2" />

        <button 
          onClick={logout} 
          className="flex items-center gap-3 px-4 py-2 rounded-xl bg-destructive/10 text-destructive border border-destructive/20 hover:bg-destructive hover:text-white transition-all duration-300 group shadow-lg shadow-destructive/10"
        >
          <span className="text-[11px] font-black uppercase tracking-widest">Sign Out</span>
          <LogOut className="w-4 h-4 group-hover:translate-x-1 transition-transform" />
        </button>
      </div>
    </nav>
  );
};