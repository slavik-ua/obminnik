import { create } from 'zustand';

import { api } from '../api/client';
import { Balance } from '../types';

interface AccountState {
  balances: Balance[];
  isLoading: boolean;
  error: string | null;

  fetchBalances: () => Promise<void>;
  getBalance: (asset: string) => Balance | undefined;
}

export const useAccountStore = create<AccountState>((set, get) => ({
  balances: [],
  isLoading: false,
  error: null,

  fetchBalances: async () => {
    set({ isLoading: true, error: null });
    try {
      const balances = await api.get<Balance[]>('/balances');
      set({ balances, isLoading: false });
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'An error occurred';
      set({ error: message, isLoading: false });
    }
  },

  getBalance: (asset: string) => {
    return get().balances.find((b) => b.asset_symbol === asset);
  },
}));
