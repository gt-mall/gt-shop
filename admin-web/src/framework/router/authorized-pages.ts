import { usePermission } from '@/framework/store';

export interface AuthorizedPageConfig {
  path: string;
  permission: string;
}

/**
 * 默认首页解析优先级列表（严格按照商城后台核心业务优先级排序）
 */
export const AUTHORIZED_PAGES: readonly AuthorizedPageConfig[] = [
  { path: '/dashboard/home', permission: 'dashboard:view' },
  { path: '/analytics/overview', permission: 'analytics:overview:view' },
  { path: '/product/product', permission: 'product:product:list' },
  { path: '/order/order', permission: 'order:order:list' },
  { path: '/order/aftersale', permission: 'order:after_sale:list' },
  { path: '/member/member', permission: 'member:member:list' },
  { path: '/product/inventory', permission: 'product:inventory:list' },
  { path: '/product/category', permission: 'product:category:list' },
  { path: '/product/freight', permission: 'product:freight:list' },
  { path: '/system/file', permission: 'system:file:list' },
  { path: '/system/user', permission: 'system:user:list' },
  { path: '/system/role', permission: 'system:role:list' },
  { path: '/system/setting/shop', permission: 'system:setting:view' },
];

export const NO_PERMISSION_PATH = '/no-permission';

/**
 * 统一默认首页解析规则
 * 根据权限判断函数按严格优先级返回可访问路径，无任何权限时返回 /no-permission
 */
export function getDefaultAuthorizedPath(
  hasPermission: (perm?: string) => boolean,
): string {
  for (const page of AUTHORIZED_PAGES) {
    if (hasPermission(page.permission)) {
      return page.path;
    }
  }
  return NO_PERMISSION_PATH;
}

/**
 * 根据权限 code 列表计算默认可访问路径
 */
export function getDefaultAuthorizedPathFromPermissions(
  permissions?: readonly string[],
): string {
  const permSet = new Set(permissions || []);
  return getDefaultAuthorizedPath((perm) => !perm || permSet.has(perm));
}

/**
 * 获取当前用户的默认可访问路径 Hook
 */
export function useDefaultAuthorizedPath(): string {
  const hasPermission = usePermission();
  return getDefaultAuthorizedPath(hasPermission);
}
