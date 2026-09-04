import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react';
import { ApiOutlined, DashboardOutlined, LogoutOutlined, MoonOutlined, SettingOutlined, SunOutlined } from '@ant-design/icons';
import { Button, ConfigProvider, Switch, Tooltip, theme as antdTheme } from 'antd';
import { DefaultFooter, PageContainer, ProLayout } from '@ant-design/pro-components';
import type { MenuDataItem } from '@ant-design/pro-components';
import dayjs from 'dayjs';
import { Link, useLocation, useNavigate } from 'react-router-dom';

import { AppRoutes } from '@/framework/router';
import { useAppStore } from '@/framework/store';

type AuthorizedMenuItem = MenuDataItem & {
  permission?: string;
  children?: AuthorizedMenuItem[];
};

const menuData: AuthorizedMenuItem[] = [
  {
    key: 'dashboard',
    name: '首页',
    path: '/dashboard',
    icon: <DashboardOutlined />,
    children: [{ key: 'dashboard-home', name: '首页', path: '/dashboard/home' }],
  },
  {
    key: 'system',
    name: '系统',
    path: '/system',
    icon: <SettingOutlined />,
    children: [
      { key: 'system-user', name: '用户管理', path: '/system/user', permission: 'system:user:list' },
      { key: 'system-role', name: '权限管理', path: '/system/role', permission: 'system:role:list' },
    ],
  },
  {
    key: 'plugin',
    name: '插件',
    path: '/plugin',
    icon: <ApiOutlined />,
    children: [{ key: 'plugin-hello', name: 'Hello Plugin', path: '/plugin/hello' }],
  },
];

type ThemeMode = 'light' | 'dark';
type ThemePreference = ThemeMode | 'system';

const layoutTokens: Record<ThemeMode, object> = {
  light: {
    bgLayout: '#f5f5f5',
    header: {
      colorBgHeader: '#ffffff',
      colorBgScrollHeader: '#ffffff',
      colorHeaderTitle: '#172033',
      colorBgMenuItemHover: '#f5f5f5',
      colorBgMenuItemSelected: '#e6f4ff',
      colorTextMenu: '#64748b',
      colorTextMenuActive: '#1677ff',
      colorTextMenuSelected: '#1677ff',
    },
    sider: {
      colorMenuBackground: '#ffffff',
      colorBgMenuItemHover: '#f5f5f5',
      colorBgMenuItemSelected: '#e6f4ff',
      colorTextMenu: '#334155',
      colorTextMenuSelected: '#1677ff',
    },
    pageContainer: {
      colorBgPageContainer: '#f5f5f5',
    },
  },
  dark: {
    bgLayout: '#141414',
    header: {
      colorBgHeader: '#001529',
      colorBgScrollHeader: '#001529',
      colorHeaderTitle: '#ffffff',
      colorBgMenuItemHover: '#112a45',
      colorBgMenuItemSelected: '#1677ff',
      colorTextMenu: 'rgba(255, 255, 255, 0.72)',
      colorTextMenuActive: '#ffffff',
      colorTextMenuSelected: '#ffffff',
    },
    sider: {
      colorMenuBackground: '#141414',
      colorBgMenuItemHover: '#1f1f1f',
      colorBgMenuItemSelected: '#112a45',
      colorTextMenu: 'rgba(255, 255, 255, 0.72)',
      colorTextMenuSelected: '#69b1ff',
    },
    pageContainer: {
      colorBgPageContainer: '#141414',
    },
  },
};

function getSystemTheme(): ThemeMode {
  return typeof window !== 'undefined' && window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

function useSystemTheme(): ThemeMode {
  const [systemTheme, setSystemTheme] = useState<ThemeMode>(getSystemTheme);

  useEffect(() => {
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    const updateSystemTheme = (event: MediaQueryListEvent) => {
      setSystemTheme(event.matches ? 'dark' : 'light');
    };

    setSystemTheme(mediaQuery.matches ? 'dark' : 'light');
    mediaQuery.addEventListener('change', updateSystemTheme);

    return () => mediaQuery.removeEventListener('change', updateSystemTheme);
  }, []);

  return systemTheme;
}

function resolveMenuPath(item: MenuDataItem, firstPathByModuleKey: Record<string, string>): string {
  const modulePath = item.key ? firstPathByModuleKey[String(item.key)] : undefined;
  return modulePath ?? item.path ?? '/';
}

export function AdminLayout() {
  const current = useAppStore((state) => state.current);
  const hasPermission = useAppStore((state) => state.hasPermission);
  const logout = useAppStore((state) => state.logout);
  const location = useLocation();
  const navigate = useNavigate();
  const systemTheme = useSystemTheme();
  const [themePreference, setThemePreference] = useState<ThemePreference>('system');
  const themeMode = themePreference === 'system' ? systemTheme : themePreference;
  const isDark = themeMode === 'dark';
  const visibleMenuData = useMemo<MenuDataItem[]>(
    () =>
      menuData
        .map((item) => ({
          ...item,
          children: item.children?.filter((child) => !child.permission || hasPermission(child.permission)),
        }))
        .filter((item) => !item.children || item.children.length > 0),
    [hasPermission],
  );
  const firstPathByModuleKey = useMemo<Record<string, string>>(
    () =>
      Object.fromEntries(
        visibleMenuData.flatMap((item) => {
          if (!item.key) return [];
          return [[String(item.key), item.children?.[0]?.path ?? item.path ?? '/']];
        }),
      ),
    [visibleMenuData],
  );

  useEffect(() => {
    document.documentElement.style.colorScheme = themeMode;
  }, [themeMode]);

  const menuItemRender = useCallback(
    (item: MenuDataItem, defaultDom: ReactNode) => (
      <Link to={resolveMenuPath(item, firstPathByModuleKey)}>{defaultDom}</Link>
    ),
    [firstPathByModuleKey],
  );

  return (
    <ConfigProvider
      theme={{
        algorithm: isDark ? antdTheme.darkAlgorithm : antdTheme.defaultAlgorithm,
        token: {
          colorPrimary: '#1677ff',
          borderRadius: 8,
        },
      }}
    >
      <ProLayout
        title="Mall Admin"
        logo={false}
        layout="mix"
        splitMenus
        navTheme={isDark ? 'realDark' : 'light'}
        locale="zh-CN"
        route={{ children: visibleMenuData }}
        location={{ pathname: location.pathname }}
        style={{ minHeight: '100vh' }}
        menuItemRender={menuItemRender}
        onMenuHeaderClick={() => navigate('/')}
        avatarProps={{
          title: current?.user?.nickname || current?.user?.username || '管理员',
          children: (current?.user?.nickname || current?.user?.username || '管').slice(0, 1),
          style: {
            backgroundColor: '#1677ff',
          },
        }}
        footerRender={() => (
          <DefaultFooter links={false} copyright={`© ${dayjs().year()} Mall Admin · 版权所有`} />
        )}
        actionsRender={() => [
          <Tooltip key="theme" title={isDark ? '切换到亮色主题' : '切换到暗色主题'}>
            <Switch
              aria-label={isDark ? '切换到亮色主题' : '切换到暗色主题'}
              checked={isDark}
              size="small"
              checkedChildren={<MoonOutlined />}
              unCheckedChildren={<SunOutlined />}
              style={{ padding: 0 }}
              onChange={(checked) => setThemePreference(checked ? 'dark' : 'light')}
            />
          </Tooltip>,
          <Tooltip key="logout" title="退出登录">
            <Button type="text" icon={<LogoutOutlined />} onClick={logout} />
          </Tooltip>,
        ]}
        siderWidth={180}
        fixedHeader
        fixSiderbar
        token={layoutTokens[themeMode]}
      >
        <PageContainer
          title={false}
          pageHeaderRender={false}
          breadcrumbRender={false}
          childrenContentStyle={{ padding: 20 }}
        >
          <AppRoutes />
        </PageContainer>
      </ProLayout>
    </ConfigProvider>
  );
}
