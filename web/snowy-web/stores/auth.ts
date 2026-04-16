import { create } from 'zustand';
import type { User } from '@/lib/api';

interface AuthState {
  user: User | null;
  accessToken: string | null;
  isLoggedIn: boolean;
  setAuth: (user: User, accessToken: string, refreshToken: string) => void;
  setUser: (user: User) => void;
  logout: () => void;
  loadFromStorage: () => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  accessToken: null,
  isLoggedIn: false,

  setAuth: (user, accessToken, refreshToken) => {
    localStorage.setItem('snowy_access_token', accessToken);
    localStorage.setItem('snowy_refresh_token', refreshToken);
    set({ user, accessToken, isLoggedIn: true });
  },

  setUser: (user) => set({ user }),

  logout: () => {
    localStorage.removeItem('snowy_access_token');
    localStorage.removeItem('snowy_refresh_token');
    set({ user: null, accessToken: null, isLoggedIn: false });
  },

  loadFromStorage: () => {
    const token = localStorage.getItem('snowy_access_token');
    if (token) {
      set({ accessToken: token, isLoggedIn: true });
    }
  },
}));
