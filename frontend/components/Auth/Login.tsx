'use client';
import React, { useState, useEffect } from 'react';
import { Mail, Lock, Activity, AlertCircle, Loader2, ArrowRight, Shield } from 'lucide-react';
import { api } from '../../api/client';
import { useAuth } from '../../context/AuthContext';
import { AuthResponse } from '../../types';

export const Login: React.FC = () => {
  const { login } = useAuth();
  const [isRegister, setIsRegister] = useState(false);
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  useEffect(() => {
    console.log('Login component mounted');
  }, []);

  const handleSubmit = async (e?: React.FormEvent | React.MouseEvent) => {
    e?.preventDefault();
    
    if (!email || !password) {
      setError('Please provide both terminal ID and security key.');
      return;
    }

    setLoading(true);
    setError(null);

    try {
      if (isRegister) {
        setSuccess('Account created! Negotiating session...');
        await api.post('/register', { email, password });
        // Auto-login after registration
        const loginData = await api.post<AuthResponse>('/login', { email, password });
        login(loginData.token);
      } else {
        const data = await api.post<AuthResponse>('/login', { email, password });
        login(data.token);
      }
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'An error occurred';
      console.error('Auth error:', message);
      setError(message);
      setSuccess(null);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-background flex items-center justify-center p-6 relative overflow-hidden">
      {/* Cinematic Background Elements */}
      <div className="absolute top-0 left-0 w-full h-full pointer-events-none">
        <div className="absolute top-[-10%] left-[-10%] w-[40%] h-[40%] bg-primary/10 blur-[120px] rounded-full animate-pulse" />
        <div className="absolute bottom-[-10%] right-[-10%] w-[40%] h-[40%] bg-indigo-500/10 blur-[120px] rounded-full animate-pulse delay-700" />
      </div>

      <div className="w-full max-w-[440px] relative z-20">
        {/* Brand Section */}
        <div className="flex flex-col items-center mb-10 group cursor-default">
          <img 
            src="/LOGO.png" 
            alt="OBMINNIK Logo" 
            className="h-16 w-auto mb-4 group-hover:scale-105 transition-transform duration-500"
          />
          <div className="h-[2px] w-12 bg-primary/30 mt-2 rounded-full" />
        </div>

        <div className="glass-card rounded-[2rem] p-10 shadow-[0_20px_50px_rgba(0,0,0,0.5)] border border-white/5 relative overflow-hidden">
           <div className="absolute top-0 left-0 w-full h-1 bg-gradient-to-r from-transparent via-primary/40 to-transparent" />
           
          <div className="mb-10">
            <h2 className="text-2xl font-black text-foreground mb-2 tracking-tight">
              {isRegister ? 'JOIN THE ELITE' : 'WELCOME BACK'}
            </h2>
            <p className="text-muted-foreground text-sm font-bold opacity-60 uppercase tracking-widest">
              Institutional Grade Trading Infrastrucure
            </p>
          </div>

          {error && (
            <div className="mb-8 flex items-center gap-4 bg-destructive/10 border border-destructive/20 p-4 rounded-2xl text-destructive text-xs font-black uppercase tracking-tight animate-in fade-in slide-in-from-top-2">
              <AlertCircle className="w-5 h-5 shrink-0" />
              <p>{error}</p>
            </div>
          )}

          {success && (
            <div className="mb-8 flex items-center gap-4 bg-buy/10 border border-buy/20 p-4 rounded-2xl text-buy text-xs font-black uppercase tracking-tight animate-in fade-in slide-in-from-top-2">
              <Shield className="w-5 h-5 shrink-0" />
              <p>{success}</p>
            </div>
          )}

          <form onSubmit={handleSubmit} className="space-y-6">
            <div className="space-y-2">
              <label className="text-[10px] font-black text-muted-foreground uppercase tracking-widest ml-1">Terminal ID (Email)</label>
              <div className="relative group">
                <Mail className="absolute left-4 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground group-focus-within:text-primary transition-colors" />
                <input
                  type="email"
                  placeholder="name@obminnik.com"
                  required
                  className="w-full bg-background/40 border border-border rounded-2xl py-4 pl-12 pr-4 text-foreground font-bold placeholder:text-muted-foreground/20 focus:ring-2 focus:ring-primary/20 focus:border-primary outline-none transition-all"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                />
              </div>
            </div>

            <div className="space-y-2">
              <label className="text-[10px] font-black text-muted-foreground uppercase tracking-widest ml-1">Security Key (Password)</label>
              <div className="relative group">
                <Lock className="absolute left-4 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground group-focus-within:text-primary transition-colors" />
                <input
                  type="password"
                  placeholder="••••••••"
                  required
                  className="w-full bg-background/40 border border-border rounded-2xl py-4 pl-12 pr-4 text-foreground font-bold placeholder:text-muted-foreground/20 focus:ring-2 focus:ring-primary/20 focus:border-primary outline-none transition-all"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                />
              </div>
            </div>

            <button
              type="submit"
              disabled={loading}
              className="group relative w-full bg-primary hover:bg-primary/90 disabled:bg-primary/50 text-white font-black uppercase tracking-[0.2em] text-xs py-5 rounded-2xl transition-all shadow-2xl shadow-primary/20 active:scale-[0.98] overflow-hidden"
            >
              <div className="absolute inset-0 bg-white/20 translate-x-[-100%] group-hover:translate-x-[100%] transition-transform duration-700" />
              <div className="relative flex items-center justify-center gap-3">
                {loading ? (
                  <Loader2 className="w-5 h-5 animate-spin" />
                ) : (
                  <>
                    <span>{isRegister ? 'INITIALIZE ACCOUNT' : 'ESTABLISH SESSION'}</span>
                    <ArrowRight className="w-4 h-4 group-hover:translate-x-1 transition-transform" />
                  </>
                )}
              </div>
            </button>
          </form>

          <div className="mt-10 pt-8 border-t border-border/50">
            <div className="flex flex-col items-center gap-6">
              <p className="text-center text-muted-foreground text-[11px] font-bold uppercase tracking-wider">
                {isRegister ? 'ALREADY REGISTERED?' : "REQUIRE TERMINAL ACCESS?"}{' '}
                <button
                  type="button"
                  onClick={(e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    setIsRegister(!isRegister);
                    setError(null);
                    setSuccess(null);
                  }}
                  className="inline-block text-primary hover:text-indigo-300 font-black transition-all ml-2 underline underline-offset-4 decoration-primary/30 hover:decoration-primary cursor-pointer py-1 px-2 rounded-md hover:bg-white/5 active:scale-95"
                >
                  {isRegister ? 'LOGIN' : 'REGISTER'}
                </button>
              </p>
              
              <div className="flex items-center gap-2 px-3 py-1.5 rounded-full bg-background/30 border border-border/30">
                <Shield className="w-3 h-3 text-muted-foreground" />
                <span className="text-[9px] text-muted-foreground font-black uppercase tracking-widest">End-to-End Encrypted</span>
              </div>
            </div>
          </div>
        </div>
        
        <p className="text-center mt-10 text-[9px] text-muted-foreground/40 font-black tracking-[0.3em] uppercase">
          OBMINNIK Global Trading Network // Secure Gateway
        </p>
      </div>
    </div>
  );
};