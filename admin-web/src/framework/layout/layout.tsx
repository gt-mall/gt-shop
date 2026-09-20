import {
  useCallback,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import {
  AccountBookOutlined,
  ApiOutlined,
  AppstoreOutlined,
  CarOutlined,
  CommentOutlined,
  CustomerServiceOutlined,
  DashboardOutlined,
  DownOutlined,
  EditOutlined,
  ExperimentOutlined,
  FileImageOutlined,
  FundOutlined,
  HistoryOutlined,
  HomeOutlined,
  InboxOutlined,
  LogoutOutlined,
  MoonOutlined,
  OrderedListOutlined,
  PictureOutlined,
  SafetyCertificateOutlined,
  SettingOutlined,
  ShopOutlined,
  ShoppingOutlined,
  SunOutlined,
  TeamOutlined,
  ToolOutlined,
  TransactionOutlined,
  UserOutlined,
} from "@ant-design/icons";
import { Avatar, ConfigProvider, Dropdown, Tag, Tooltip } from "antd";
import {
  DefaultFooter,
  PageContainer,
  ProLayout,
} from "@ant-design/pro-components";
import type { MenuDataItem } from "@ant-design/pro-components";
import dayjs from "dayjs";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { financeMenuIcons } from "@/modules/analytics/icons";

import { AppRoutes } from "@/framework/router";
import {
  useAuth,
  usePermission,
  useRoles,
  useShopSetting,
  useSiteSetting,
  useSystemSetting,
  useUser,
} from "@/framework/store";
import { adminThemeConfig } from "@/shared/admin-ui";

import "./layout.less";

type MenuItem = {
  key: string;
  name: string;
  path: string;
  permission?: string;
  icon?: ReactNode;
};

type MenuModule = MenuItem & {
  icon: ReactNode;
  children: MenuItem[];
};

const menuData: MenuModule[] = [
  {
    key: "dashboard",
    name: "首页",
    path: "/dashboard",
    icon: <DashboardOutlined />,
    children: [
      {
        key: "dashboard-home",
        name: "数据看板",
        path: "/dashboard/home",
        permission: 'dashboard:view',
        icon: <HomeOutlined />,
      },
    ],
  },
  {
    key: "product",
    name: "商品",
    path: "/product",
    icon: <ShopOutlined />,
    children: [
      {
        key: "product-product",
        name: "商品列表",
        path: "/product/product",
        permission: "product:product:list",
        icon: <ShoppingOutlined />,
      },
      {
        key: "product-category",
        name: "商品分类",
        path: "/product/category",
        permission: "product:category:list",
        icon: <AppstoreOutlined />,
      },
      {
        key: "product-inventory",
        name: "库存管理",
        path: "/product/inventory",
        permission: "product:inventory:list",
        icon: <InboxOutlined />,
      },
      {
        key: "product-inventory-log",
        name: "库存流水",
        path: "/product/inventory_log",
        permission: "product:inventory:log",
        icon: <HistoryOutlined />,
      },
      {
        key: "product-freight",
        name: "运费模板",
        path: "/product/freight",
        permission: "product:freight:list",
        icon: <CarOutlined />,
      },
      {
        key: "product-review",
        name: "商品评价",
        path: "/product/review",
        permission: "product:review:list",
        icon: <CommentOutlined />,
      },
    ],
  },
  {
    key: "order",
    name: "订单",
    path: "/order",
    icon: <OrderedListOutlined />,
    children: [
      {
        key: "order-order",
        name: "订单管理",
        path: "/order/order",
        permission: "order:order:list",
        icon: <OrderedListOutlined />,
      },
      {
        key: "order-aftersale",
        name: "售后管理",
        path: "/order/aftersale",
        permission: "order:after_sale:list",
        icon: <CustomerServiceOutlined />,
      },
    ],
  },
  {
    key: "member",
    name: "会员",
    path: "/member",
    icon: <TeamOutlined />,
    children: [
      {
        key: "member-member",
        name: "会员列表",
        path: "/member/member",
        permission: "member:member:list",
        icon: <UserOutlined />,
      },
    ],
  },
  {
    key: "analytics",
    name: "数据财务",
    path: "/analytics",
    icon: financeMenuIcons.finance,
    children: [
      {
        key: "analytics-overview",
        name: "经营概览",
        path: "/analytics/overview",
        permission: "analytics:overview:view",
        icon: financeMenuIcons.overview,
      },
      {
        key: "analytics-trade",
        name: "交易分析",
        path: "/analytics/trade",
        permission: "analytics:trade:view",
        icon: financeMenuIcons.trade,
      },
      {
        key: "analytics-product",
        name: "商品分析",
        path: "/analytics/product",
        permission: "analytics:product:view",
        icon: financeMenuIcons.product,
      },
      {
        key: "analytics-member",
        name: "会员分析",
        path: "/analytics/member",
        permission: "analytics:member:view",
        icon: financeMenuIcons.member,
      },
      {
        key: "analytics-flow",
        name: "资金流水",
        path: "/analytics/flow",
        permission: "analytics:flow:view",
        icon: financeMenuIcons.flow,
      },
    ],
  },
  {
    key: "system",
    name: "系统",
    path: "/system",
    icon: <SettingOutlined />,
    children: [
      {
        key: "system-user",
        name: "用户管理",
        path: "/system/user",
        permission: "system:user:list",
        icon: <UserOutlined />,
      },
      {
        key: "system-role",
        name: "权限管理",
        path: "/system/role",
        permission: "system:role:list",
        icon: <SafetyCertificateOutlined />,
      },
      {
        key: "system-file",
        name: "图片素材",
        path: "/system/file",
        permission: "system:file:list",
        icon: <PictureOutlined />,
      },
      {
        key: "system-setting",
        name: "系统设置",
        path: "/system/setting/shop",
        permission: "system:setting:view",
        icon: <ToolOutlined />,
      },
      {
        key: "system-product-service",
        name: "服务保障",
        path: "/system/product-service",
        permission: "product:dict_service:list",
        icon: <SafetyCertificateOutlined />,
      },
      {
        key: "system-product-unit",
        name: "商品单位",
        path: "/system/product-unit",
        permission: "product:dict_unit:list",
        icon: <AppstoreOutlined />,
      },
    ],
  },
  {
    key: "plugin",
    name: "扩展",
    path: "/plugin",
    icon: <ApiOutlined />,
    children: [
      {
        key: "plugin-hello",
        name: "Hello Plugin",
        path: "/plugin/hello",
        icon: <ApiOutlined />,
      },
    ],
  },
];

type ThemeMode = "light" | "dark";
type ThemePreference = ThemeMode | "system";

const layoutTokens: Record<ThemeMode, object> = {
  light: {
    bgLayout: "#F5F6F8",
    header: {
      colorBgHeader: "rgba(255, 255, 255, 0.6)",
      colorBgScrollHeader: "rgba(255, 255, 255, 0.75)",
      colorHeaderTitle: "#111827",
      colorBgMenuItemHover: "rgba(0, 0, 0, 0.04)",
      colorBgMenuItemSelected: "rgba(67, 56, 202, 0.09)",
      colorTextMenu: "#374151",
      colorTextMenuActive: "#4338CA",
      colorTextMenuSelected: "#4338CA",
      heightLayoutHeader: 56,
    },
    sider: {
      colorMenuBackground: "#FFFFFF",
      colorBgMenuItemHover: "#F3F4F6",
      colorBgMenuItemSelected: "#EEF2FF",
      colorTextMenu: "#374151",
      colorTextMenuActive: "#4338CA",
      colorTextMenuSelected: "#4338CA",
      colorMenuItemDivider: "#E5E7EB",
    },
    pageContainer: {
      colorBgPageContainer: "#F5F6F8",
    },
  },
  dark: {
    bgLayout: "#0B0F19",
    header: {
      colorBgHeader: "rgba(17, 24, 39, 0.65)",
      colorBgScrollHeader: "rgba(17, 24, 39, 0.8)",
      colorHeaderTitle: "#F9FAFB",
      colorBgMenuItemHover: "rgba(255, 255, 255, 0.06)",
      colorBgMenuItemSelected: "rgba(99, 102, 241, 0.18)",
      colorTextMenu: "#9CA3AF",
      colorTextMenuActive: "#818CF8",
      colorTextMenuSelected: "#818CF8",
      heightLayoutHeader: 56,
    },
    sider: {
      colorMenuBackground: "#111827",
      colorBgMenuItemHover: "#1F2937",
      colorBgMenuItemSelected: "rgba(99, 102, 241, 0.18)",
      colorTextMenu: "#9CA3AF",
      colorTextMenuActive: "#818CF8",
      colorTextMenuSelected: "#818CF8",
      colorMenuItemDivider: "#1F2937",
    },
    pageContainer: {
      colorBgPageContainer: "#0B0F19",
    },
  },
};

function getSystemTheme(): ThemeMode {
  return typeof window !== "undefined" &&
    window.matchMedia("(prefers-color-scheme: dark)").matches
    ? "dark"
    : "light";
}

function useSystemTheme(): ThemeMode {
  const [systemTheme, setSystemTheme] = useState<ThemeMode>(getSystemTheme);

  useEffect(() => {
    const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
    const updateSystemTheme = (event: MediaQueryListEvent) => {
      setSystemTheme(event.matches ? "dark" : "light");
    };

    setSystemTheme(mediaQuery.matches ? "dark" : "light");
    mediaQuery.addEventListener("change", updateSystemTheme);

    return () => mediaQuery.removeEventListener("change", updateSystemTheme);
  }, []);

  return systemTheme;
}

const BrandLogo = ({
  logoUrl,
  mallName,
}: {
  logoUrl?: string;
  mallName?: string;
}) => {
  if (logoUrl) {
    return (
      <div
        style={{
          display: "inline-flex",
          alignItems: "center",
          justifyContent: "center",
          height: 32,
          maxWidth: 140,
          flexShrink: 0,
        }}
      >
        <img
          src={logoUrl}
          alt={mallName || "Logo"}
          style={{
            maxHeight: 32,
            maxWidth: "100%",
            objectFit: "contain",
            display: "block",
          }}
        />
      </div>
    );
  }

  return (
    <div
      style={{
        display: "inline-flex",
        alignItems: "center",
        justifyContent: "center",
        width: 32,
        height: 32,
        borderRadius: 6,
        background: "#4338CA",
        color: "#fff",
        fontSize: 17,
        flexShrink: 0,
      }}
    >
      <ShopOutlined />
    </div>
  );
};

const BrandTitle = ({
  mallName,
  isDark,
}: {
  mallName?: string;
  isDark: boolean;
}) => (
  <div style={{ display: "inline-flex", alignItems: "center", gap: 8 }}>
    <span
      style={{
        fontWeight: 600,
        fontSize: 16,
        letterSpacing: "-0.01em",
        color: isDark ? "#F9FAFB" : "#111827",
        whiteSpace: "nowrap",
        overflow: "hidden",
        textOverflow: "ellipsis",
        maxWidth: 160,
      }}
    >
      {mallName || "Mall Admin"}
    </span>
  </div>
);

export function AdminLayout() {
  const user = useUser();
  const roles = useRoles();
  const hasPermission = usePermission();
  const { logout } = useAuth();
  const { shopSetting, siteSetting, refreshSettings } = useSystemSetting();
  const location = useLocation();
  const navigate = useNavigate();
  const systemTheme = useSystemTheme();
  const [themePreference, setThemePreference] =
    useState<ThemePreference>("system");
  const themeMode =
    themePreference === "system" ? systemTheme : themePreference;
  const isDark = themeMode === "dark";

  useEffect(() => {
    if (!shopSetting) {
      void refreshSettings();
    }
  }, [shopSetting, refreshSettings]);

  let menuPathname = location.pathname;
  if (
    location.pathname.startsWith("/product/inventory/log") ||
    location.pathname.startsWith("/product/inventory_log")
  ) {
    menuPathname = "/product/inventory_log";
  } else if (location.pathname.startsWith("/system/role/")) {
    menuPathname = "/system/role";
  } else if (location.pathname.startsWith("/system/setting")) {
    menuPathname = "/system/setting/shop";
  } else if (location.pathname.startsWith("/product/product")) {
    menuPathname = "/product/product";
  } else if (location.pathname.startsWith("/order/order")) {
    menuPathname = "/order/order";
  } else if (location.pathname.startsWith("/member/member")) {
    menuPathname = "/member/member";
  }

  const visibleMenuData = useMemo(
    () =>
      menuData
        .map((item) => ({
          ...item,
          children: item.children.filter((child) =>
            hasPermission(child.permission),
          ),
        }))
        .filter((item) => item.children.length > 0),
    [hasPermission],
  );

  useEffect(() => {
    document.documentElement.style.colorScheme = themeMode;
    document.body.style.backgroundColor = isDark ? "#0B0F19" : "#F5F6F8";
  }, [themeMode, isDark]);

  const menuItemRender = useCallback(
    (item: MenuDataItem, defaultDom: ReactNode) => {
      const module = visibleMenuData.find((menu) => menu.key === item.key);
      return (
        <Link to={module ? module.children[0].path : (item.path ?? "/")}>
          {defaultDom}
        </Link>
      );
    },
    [visibleMenuData],
  );

  const roleName =
    roles[0]?.name ||
    (roles[0]?.code === "super_admin" ? "超级管理员" : "管理员");

  return (
    <ConfigProvider theme={adminThemeConfig(isDark)}>
      <ProLayout
        className={`mall-layout-root mall-theme-${themeMode}`}
        logo={
          <BrandLogo logoUrl={shopSetting?.logo} mallName={shopSetting?.name} />
        }
        title={false}
        headerTitleRender={(logo) => (
          <div
            style={{
              display: "inline-flex",
              alignItems: "center",
              gap: 10,
              cursor: "pointer",
            }}
            onClick={() => navigate("/")}
          >
            {logo}
            <BrandTitle mallName={shopSetting?.name} isDark={isDark} />
          </div>
        )}
        layout="mix"
        splitMenus
        navTheme={isDark ? "realDark" : "light"}
        locale="zh-CN"
        route={{ children: visibleMenuData }}
        location={{ pathname: menuPathname }}
        style={{ minHeight: "100vh" }}
        menuItemRender={menuItemRender}
        onMenuHeaderClick={() => navigate("/")}
        avatarProps={false}
        footerRender={() => (
          <DefaultFooter
            links={false}
            copyright={
              siteSetting?.copyright ||
              `${dayjs().year()} ${shopSetting?.name || "Mall Admin"} · 电商数字化运营平台`
            }
          />
        )}
        actionsRender={() => [
          <Tooltip
            key="theme"
            title={isDark ? "切换到浅色模式" : "切换到深色模式"}
          >
            <button
              type="button"
              className="mall-theme-btn"
              aria-label={isDark ? "切换到浅色模式" : "切换到深色模式"}
              onClick={() => setThemePreference(isDark ? "light" : "dark")}
            >
              {isDark ? (
                <SunOutlined style={{ color: "#fbbf24" }} />
              ) : (
                <MoonOutlined />
              )}
            </button>
          </Tooltip>,
          <Dropdown
            key="user"
            menu={{
              items: [
                {
                  key: "user-info",
                  label: (
                    <div style={{ padding: "4px 2px" }}>
                      <div style={{ fontWeight: 600, fontSize: 13 }}>
                        {user?.nickname || user?.username || "管理员"}
                      </div>
                      <div
                        style={{
                          fontSize: 12,
                          color: isDark ? "#94A3B8" : "#6B7280",
                        }}
                      >
                        {user?.username ? `@${user.username}` : ""}
                      </div>
                    </div>
                  ),
                  disabled: true,
                },
                { type: "divider" },
                {
                  key: "theme-quick",
                  label: isDark ? "浅色模式" : "深色模式",
                  icon: isDark ? <SunOutlined /> : <MoonOutlined />,
                  onClick: () => setThemePreference(isDark ? "light" : "dark"),
                },
                { type: "divider" },
                {
                  key: "logout",
                  label: "退出登录",
                  icon: <LogoutOutlined />,
                  danger: true,
                  onClick: logout,
                },
              ],
            }}
            placement="bottomRight"
            arrow
          >
            <div className="mall-header-user-btn">
              <Avatar
                size={28}
                style={{
                  background: isDark ? "#4F46E5" : "#4338CA",
                  fontWeight: 600,
                  fontSize: 13,
                }}
              >
                {(user?.nickname || user?.username || "管")
                  .slice(0, 1)
                  .toUpperCase()}
              </Avatar>
              <span className="user-name">
                {user?.nickname || user?.username || "管理员"}
              </span>
              <Tag
                bordered={false}
                style={{
                  margin: 0,
                  fontSize: 11,
                  padding: "0 6px",
                  lineHeight: "18px",
                  background: isDark ? "rgba(99, 102, 241, 0.2)" : "#EEF2FF",
                  color: isDark ? "#818CF8" : "#4338CA",
                  border: `1px solid ${isDark ? "rgba(99, 102, 241, 0.35)" : "#E0E7FF"}`,
                  fontWeight: 500,
                }}
              >
                {roleName}
              </Tag>
              <DownOutlined
                style={{ fontSize: 10, color: isDark ? "#94A3B8" : "#6B7280" }}
              />
            </div>
          </Dropdown>,
        ]}
        siderWidth={210}
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
