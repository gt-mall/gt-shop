import { useCallback } from 'react';
import { create } from 'zustand';
import { useShallow } from 'zustand/react/shallow';

import { api, configureAuthSession } from '@/framework/http';
import type {
  CurrentRole,
  CurrentUserResponse,
  LoginRequest,
  ShopBasicConfig,
  SiteBasicConfig,
} from '@/framework/http/generated_api';

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

export function updateFavicon(url?: string) {
  if (typeof document === 'undefined') return;
  let link: HTMLLinkElement | null = document.querySelector("link[rel*='icon']");
  if (!url) {
    return;
  }
  if (!link) {
    link = document.createElement('link');
    link.rel = 'shortcut icon';
    document.head.appendChild(link);
  }
  link.href = url;
}

export function updateDocumentTitle(title?: string) {
  if (typeof document === 'undefined' || !title) return;
  document.title = title;
}

export type AppStore = {
  token: string;
  expiresAt: string;
  remember: boolean;
  current?: CurrentUserResponse;
  shopSetting?: ShopBasicConfig;
  siteSetting?: SiteBasicConfig;
  loading: boolean;
  initialized: boolean;
  login: (request: LoginRequest) => Promise<LoginResult>;
  refreshCurrent: () => Promise<boolean>;
  refreshSettings: () => Promise<void>;
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
  shopSetting: undefined,
  siteSetting: undefined,
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
    if (success) {
      void get().refreshSettings();
    }
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
      void get().refreshSettings();
      return true;
    })();
    return refreshRequest;
  },
  refreshSettings: async () => {
    const token = get().token;
    if (!token) return;
    try {
      const [shopRes, siteRes] = await Promise.all([
        api.api.getSystemShopSetting(),
        api.api.getSystemSiteSetting(),
      ]);
      const updates: Partial<AppStore> = {};
      if (shopRes.code === 1 && shopRes.data) {
        updates.shopSetting = shopRes.data;
        if (shopRes.data.favicon) {
          updateFavicon(shopRes.data.favicon);
        }
      }
      if (siteRes.code === 1 && siteRes.data) {
        updates.siteSetting = siteRes.data;
      }
      const title = siteRes.data?.title || shopRes.data?.name;
      if (title) {
        updateDocumentTitle(title);
      }
      if (Object.keys(updates).length > 0) {
        set(updates);
      }
    } catch {
      // 忽略设置读取异常，保留默认界面状态
    }
  },
  logout: () => {
    refreshRequest = undefined;
    clearStoredAuth();
    set({
      token: '',
      expiresAt: '',
      remember: false,
      current: undefined,
      shopSetting: undefined,
      siteSetting: undefined,
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

/** 商城基础设置（商城名称、Logo、浏览器图标、客服信息等） */
export function useShopSetting() {
  return useAppStore((state) => state.shopSetting);
}

/** 站点基础设置（网站标题、备案号、版权信息等） */
export function useSiteSetting() {
  return useAppStore((state) => state.siteSetting);
}

/** 系统配置及主动刷新能力 */
export function useSystemSetting() {
  return useAppStore(
    useShallow((state) => ({
      shopSetting: state.shopSetting,
      siteSetting: state.siteSetting,
      refreshSettings: state.refreshSettings,
    })),
  );
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
    refreshSettings: state.refreshSettings,
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
