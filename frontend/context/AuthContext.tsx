'use client';

import React, { createContext, useCallback, useContext, useEffect, useState } from 'react';

import { UNAUTHORIZED_EVENT } from '../api/client';
import { safeStorage } from '../utils/storage';

interface AuthContextType {
  token: string | null;
  isAuthenticated: boolean;
  login: (token: string) => void;
  logout: () => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  // Use a function to initialize state safely
  const [token, setToken] = useState<string | null>(() => {
    // Initial load from storage
    return safeStorage.getItem('token');
  });

  useEffect(() => {
    // Sync token if it changes in another tab
    const handleStorageChange = (e: StorageEvent) => {
      if (e.key === 'token') {
        const value = safeStorage.getItem('token');
        setToken(value);
      }
    };
    window.addEventListener('storage', handleStorageChange);
    return () => window.removeEventListener('storage', handleStorageChange);
  }, []);

  const logout = useCallback(() => {
    safeStorage.removeItem('token');
    setToken(null);
  }, []);

  const login = useCallback((newToken: string) => {
    safeStorage.setItem('token', newToken);
    setToken(newToken);
  }, []);

  // Listen for the global unauthorized event from our API client
  useEffect(() => {
    const handleUnauthorized = () => {
      logout();
    };

    window.addEventListener(UNAUTHORIZED_EVENT, handleUnauthorized);
    return () => {
      window.removeEventListener(UNAUTHORIZED_EVENT, handleUnauthorized);
    };
  }, [logout]);

  const value = {
    token,
    isAuthenticated: !!token,
    login,
    logout,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};

/**
 * Custom hook to access auth state throughout the app
 */
export const useAuth = () => {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};
