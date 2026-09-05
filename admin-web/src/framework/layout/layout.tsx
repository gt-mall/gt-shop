import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react';
import { ApiOutlined, DashboardOutlined, LogoutOutlined, MoonOutlined, SettingOutlined, SunOutlined } from '@ant-design/icons';
import { Button, ConfigProvider, Switch, Tooltip, theme as antdTheme } from 'antd';
import { DefaultFooter, PageContainer, ProLayout } from '@ant-design/pro-components';
import type { MenuDataItem } from '@ant-design/pro-components';
import dayjs from 'dayjs';
import { Link, useLocation, useNavigate } from 'react-router-dom';

import { AppRoutes } from '@/framework/router';
import { useAuth, usePermission, useUser } from '@/framework/store';

type MenuItem = {
  key: string;
  name: string;
  path: string;
  permission?: string;
};

type MenuModule = MenuItem & {
  icon: ReactNode;
  children: MenuItem[];
};

const menuData: MenuModule[] = [
  {
    key: 'dashboard',
    name: '首页',
    path: '/dashboard',
    icon: <DashboardOutlined />,
    children: [
      { key: 'dashboard-home', name: '首页', path: '/dashboard/home' },
      { key: 'dashboard-demo', name: 'Hooks 示例', path: '/dashboard/demo' },
      { key: 'dashboard-image-demo', name: '图片选择示例', path: '/dashboard/image-demo', permission: 'system:file:list' },
    ],
  },
  {
    key: 'system',
    name: '系统',
    path: '/system',
    icon: <SettingOutlined />,
    children: [
      { key: 'system-user', name: '用户管理', path: '/system/user', permission: 'system:user:list' },
      { key: 'system-role', name: '权限管理', path: '/system/role', permission: 'system:role:list' },
      { key: 'system-file', name: '图片素材', path: '/system/file', permission: 'system:file:list' },
      { key: 'system-file-storage', name: '存储配置', path: '/system/file-storage', permission: 'system:file:config' },
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

export function AdminLayout() {
  const user = useUser();
  const hasPermission = usePermission();
  const { logout } = useAuth();
  const location = useLocation();
  const navigate = useNavigate();
  const systemTheme = useSystemTheme();
  const [themePreference, setThemePreference] = useState<ThemePreference>('system');
  const themeMode = themePreference === 'system' ? systemTheme : themePreference;
  const isDark = themeMode === 'dark';
  const menuPathname = location.pathname.startsWith('/system/role/') ? '/system/role' : location.pathname;
  const visibleMenuData = useMemo(
    () =>
      menuData
        .map((item) => ({
          ...item,
          children: item.children.filter((child) => hasPermission(child.permission)),
        }))
        .filter((item) => item.children.length > 0),
    [hasPermission],
  );

  useEffect(() => {
    document.documentElement.style.colorScheme = themeMode;
  }, [themeMode]);

  const menuItemRender = useCallback(
    (item: MenuDataItem, defaultDom: ReactNode) => {
      const module = visibleMenuData.find((menu) => menu.key === item.key);
      return <Link to={module ? module.children[0].path : item.path ?? '/'}>{defaultDom}</Link>;
    },
    [visibleMenuData],
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
        location={{ pathname: menuPathname }}
        style={{ minHeight: '100vh' }}
        menuItemRender={menuItemRender}
        onMenuHeaderClick={() => navigate('/')}
        avatarProps={{
          title: user?.nickname || user?.username || '管理员',
          children: (user?.nickname || user?.username || '管').slice(0, 1),
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
