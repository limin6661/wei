import { defineStore } from 'pinia';
import http from '@/services/http';

interface UserPayload {
  id: number;
  username: string;
  require_reset: boolean;
}

interface LoginRequest {
  username: string;
  password: string;
}

interface ApiResponse<T> {
  success: boolean;
  data: T;
  error?: string;
}

export const useAuthStore = defineStore('auth', {
  state: () => ({
    user: null as UserPayload | null,
    initialized: false,
  }),
  getters: {
    isAuthenticated: (state) => !!state.user && state.user.require_reset === false,
    needsReset: (state) => !!state.user?.require_reset,
  },
  actions: {
    async login(payload: LoginRequest) {
      const res = await http.post<ApiResponse<UserPayload>>('/api/login', payload);
      if (res.data.success) {
        this.user = res.data.data;
        return res.data.data;
      }
      throw new Error(res.data.error || '登录失败');
    },
    async fetchMe() {
      try {
        const res = await http.get<ApiResponse<UserPayload>>('/api/me');
        if (res.data.success) {
          this.user = res.data.data;
        }
      } finally {
        this.initialized = true;
      }
    },
    async updatePassword(oldPassword: string, newPassword: string) {
      const res = await http.post<ApiResponse<{ status: string }>>('/api/password', {
        old_password: oldPassword,
        new_password: newPassword,
      });
      if (res.data.success) {
        if (this.user) {
          this.user.require_reset = false;
        }
        return res.data.data;
      }
      throw new Error(res.data.error || '修改失败');
    },
    async logout() {
      await http.post('/api/logout');
      this.user = null;
      this.initialized = true;
    },
  },
});
