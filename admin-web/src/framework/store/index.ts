import { useCallback } from 'react';
import { create } from 'zustand';
import { useShallow } from 'zustand/react/shallow';

import { api, configureAuthSession } from '@/framework/http';
import type { CurrentRole, CurrentUserResponse, LoginRequest } from '@/framework/http/generated_api';

const authStorageKey = 'mall_admin_auth';
const legacyTokenKey = 'mall_admin_token';
const emptyRoles: readonly CurrentRole[] = [];
const emptyPermissions: readonly string[] = [];

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
    const result = await api.api.login(request);
    const { token, expires_at: expiresAt } = result.data ?? {};
    if (result.code !== 1 || !token || !expiresAt) {
      set({ loading: false });
      return { success: false, message: result.message || '登录失败，请检查后端服务' };
    }

    const remember = Boolean(request.remember);
    saveStoredAuth({ token, expiresAt }, remember);
    set({ token, expiresAt, remember, current: undefined, initialized: false });
    const success = await get().refreshCurrent();
    return {
      success,
      message: success ? undefined : '获取用户信息失败',
    };
  },
  refreshCurrent: async () => {
    const token = get().token;
    if (!token) return false;
    if (refreshRequest) return refreshRequest;

    set({ loading: true });
    refreshRequest = (async () => {
      const result = await api.api.getCurrentUser();
      // 退出或切换登录后，忽略旧请求，避免覆盖新会话。
      if (get().token !== token) return false;

      refreshRequest = undefined;
      if (result.code !== 1 || !result.data) {
        get().logout();
        return false;
      }

      set({ current: result.data, loading: false, initialized: true });
      return true;
    })();
    return refreshRequest;
  },
  logout: () => {
    refreshRequest = undefined;
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
}));

/** 当前用户基本信息；未登录或尚未恢复时返回 undefined。 */
export function useUser() {
  return useAppStore((state) => state.current?.user);
}

/** 当前用户的角色列表，无角色时返回空数组。 */
export function useRoles() {
  return useAppStore((state) => state.current?.roles ?? emptyRoles);
}

/** 当前用户的权限 code 列表，无权限时返回空数组。 */
export function usePermissions() {
  return useAppStore((state) => state.current?.permissions ?? emptyPermissions);
}

/** 登录状态及登录、退出、刷新当前用户的操作。 */
export function useAuth() {
  return useAppStore(useShallow((state) => ({
    isLoggedIn: Boolean(state.current),
    initialized: state.initialized,
    loading: state.loading,
    login: state.login,
    logout: state.logout,
    refreshCurrent: state.refreshCurrent,
  })));
}

/** 返回权限判断函数；未指定 code 时表示无需权限。 */
export function usePermission() {
  const permissions = usePermissions();
  return useCallback(
    (permission?: string) => !permission || permissions.includes(permission),
    [permissions],
  );
}

configureAuthSession(
  () => useAppStore.getState().token,
  () => useAppStore.getState().logout(),
);
