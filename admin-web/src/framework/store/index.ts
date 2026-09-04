import { create } from 'zustand';

import { api, configureAuthSession, type ApiResponse } from '@/framework/http';
import type { CurrentUserResponse, LoginRequest, LoginResponse } from '@/framework/http/generated_api';

const authStorageKey = 'mall_admin_auth';
const legacyTokenKey = 'mall_admin_token';

type StoredAuth = {
  token: string;
  expiresAt: string;
};

type LoginResult = {
  success: boolean;
  message?: string;
};

export type AppStore = {
  token: string;
  expiresAt: string;
  remember: boolean;
  current?: CurrentUserResponse;
  loading: boolean;
  initialized: boolean;
  login: (request: LoginRequest) => Promise<LoginResult>;
  refreshCurrent: () => Promise<boolean>;
  logout: () => void;
  hasPermission: (permission: string) => boolean;
};

function clearStoredAuth() {
  if (typeof window === 'undefined') return;
  window.localStorage.removeItem(authStorageKey);
  window.sessionStorage.removeItem(authStorageKey);
  window.localStorage.removeItem(legacyTokenKey);
}

function parseStoredAuth(value: string | null): StoredAuth | undefined {
  if (!value) return;
  try {
    const auth = JSON.parse(value) as Partial<StoredAuth>;
    if (!auth.token || !auth.expiresAt || Date.parse(auth.expiresAt) <= Date.now()) return;
    return { token: auth.token, expiresAt: auth.expiresAt };
  } catch {
    return;
  }
}

function loadStoredAuth(): StoredAuth & { remember: boolean } {
  if (typeof window === 'undefined') return { token: '', expiresAt: '', remember: false };

  const remembered = parseStoredAuth(window.localStorage.getItem(authStorageKey));
  if (remembered) return { ...remembered, remember: true };

  const session = parseStoredAuth(window.sessionStorage.getItem(authStorageKey));
  if (session) return { ...session, remember: false };

  clearStoredAuth();
  return { token: '', expiresAt: '', remember: false };
}

function saveStoredAuth(auth: StoredAuth, remember: boolean) {
  clearStoredAuth();
  const storage = remember ? window.localStorage : window.sessionStorage;
  storage.setItem(authStorageKey, JSON.stringify(auth));
}

const initialAuth = loadStoredAuth();
let refreshRequest: Promise<boolean> | undefined;

export const useAppStore = create<AppStore>((set, get) => ({
  ...initialAuth,
  current: undefined,
  loading: Boolean(initialAuth.token),
  initialized: !initialAuth.token,
  login: async (request) => {
    set({ loading: true });
    const result = await api.api
      .login(request)
      .catch(() => undefined) as ApiResponse<LoginResponse> | undefined;
    const token = result?.data?.token;
    const expiresAt = result?.data?.expires_at;
    if (result?.code !== 1 || !token || !expiresAt) {
      set({ loading: false });
      return { success: false, message: result?.message || '登录失败，请检查后端服务' };
    }

    const remember = Boolean(request.remember);
    saveStoredAuth({ token, expiresAt }, remember);
    set({ token, expiresAt, remember, current: undefined, initialized: false });
    const success = await get().refreshCurrent();
    set({ loading: false });
    return {
      success,
      message: success ? undefined : '获取用户信息失败',
    };
  },
  refreshCurrent: () => {
    if (refreshRequest) return refreshRequest;
    const token = get().token;
    if (!token) {
      set({ current: undefined, loading: false, initialized: true });
      return Promise.resolve(false);
    }

    set({ loading: true });
    refreshRequest = (async () => {
      const result = await api.api
        .getCurrentUser()
        .catch(() => undefined) as ApiResponse<CurrentUserResponse> | undefined;
      let success = false;
      if (get().token === token && result?.code === 1 && result.data) {
        set({ current: result.data });
        success = true;
      } else if (get().token === token) {
        get().logout();
      }
      refreshRequest = undefined;
      set({ loading: false, initialized: true });
      return success;
    })();
    return refreshRequest;
  },
  logout: () => {
    clearStoredAuth();
    set({
      token: '',
      expiresAt: '',
      remember: false,
      current: undefined,
      loading: false,
      initialized: true,
    });
  },
  hasPermission: (permission) => get().current?.permissions?.includes(permission) ?? false,
}));

configureAuthSession(
  () => useAppStore.getState().token,
  () => useAppStore.getState().logout(),
);
