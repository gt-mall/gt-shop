import { useCallback, useEffect, useState, type ReactNode } from 'react';
import { LaptopOutlined, MoonOutlined, NotificationOutlined, SunOutlined, UserOutlined } from '@ant-design/icons';
import { ConfigProvider, Switch, Tooltip, theme as antdTheme } from 'antd';
import { DefaultFooter, PageContainer, ProLayout } from '@ant-design/pro-components';
import type { MenuDataItem } from '@ant-design/pro-components';
import dayjs from 'dayjs';
import { Link, useLocation, useNavigate } from 'react-router-dom';

import { AppRoutes } from '@/framework/router';

type MenuLeaf = {
  key: string;
  label: string;
  path: string;
};

type MenuGroup = {
  key: string;
  label: string;
  icon: ReactNode;
  children: MenuLeaf[];
};

type ModuleNavigation = {
  key: string;
  label: string;
  groups: MenuGroup[];
};

const groupDefinitions = [
  { key: 'subnav1', label: 'subnav 1', icon: <UserOutlined /> },
  { key: 'subnav2', label: 'subnav 2', icon: <LaptopOutlined /> },
  { key: 'subnav3', label: 'subnav 3', icon: <NotificationOutlined /> },
];

const specialPaths: Record<string, string> = {
  'home-subnav1-option1': '/',
  'home-subnav1-option2': '/demo',
  'home-subnav1-option3': '/plugins/hello',
};

function createModule(moduleKey: string, label: string): ModuleNavigation {
  return {
    key: moduleKey,
    label,
    groups: groupDefinitions.map((group) => {
      const groupKey = `${moduleKey}-${group.key}`;

      return {
        ...group,
        key: groupKey,
        children: ['option1', 'option2', 'option3', 'option4'].map((option) => {
          const key = `${moduleKey}-${group.key}-${option}`;

          return {
            key,
            label: option,
            path: specialPaths[key] ?? `/${moduleKey}/${groupKey}/${option}`,
          };
        }),
      };
    }),
  };
}

const modules = [
  createModule('home', '首页'),
  createModule('product', '商品'),
  createModule('order', '订单'),
  createModule('user', '用户'),
  createModule('shop', '店铺'),
  createModule('settings', '设置'),
];

const firstPathByModuleKey = Object.fromEntries(
  modules.map((module) => [module.key, module.groups[0].children[0].path]),
);

const menuData: MenuDataItem[] = modules.map((module) => ({
  key: module.key,
  name: module.label,
  path: `/${module.key}`,
  children: module.groups.map((group) => ({
    key: group.key,
    name: group.label,
    path: `/${module.key}/${group.key}`,
    icon: group.icon,
    children: group.children.map((child) => ({
      key: child.key,
      name: child.label,
      path: child.path,
    })),
  })),
}));

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

function resolveMenuPath(item: MenuDataItem): string {
  const modulePath = item.key ? firstPathByModuleKey[item.key] : undefined;
  return modulePath ?? item.path ?? '/';
}

export function AdminLayout() {
  const location = useLocation();
  const navigate = useNavigate();
  const systemTheme = useSystemTheme();
  const [themePreference, setThemePreference] = useState<ThemePreference>('system');
  const themeMode = themePreference === 'system' ? systemTheme : themePreference;
  const isDark = themeMode === 'dark';

  useEffect(() => {
    document.documentElement.style.colorScheme = themeMode;
  }, [themeMode]);

  const menuItemRender = useCallback(
    (item: MenuDataItem, defaultDom: ReactNode) => (
      <Link to={resolveMenuPath(item)}>{defaultDom}</Link>
    ),
    [],
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
        route={{ children: menuData }}
        location={{ pathname: location.pathname }}
        style={{ minHeight: '100vh' }}
        menuItemRender={menuItemRender}
        onMenuHeaderClick={() => navigate('/')}
        avatarProps={{
          title: '管理员',
          children: '管理员',
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
