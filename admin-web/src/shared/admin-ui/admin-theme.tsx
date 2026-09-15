import type { ThemeConfig } from "antd";
import { theme } from "antd";

/**
 * 后台全站统一 Theme：由 Layout 的 ConfigProvider 直接使用，页面不再包主题容器。
 * 主色、文字层级、控件高度、圆角与常用组件密度以定稿的商品列表为基线。
 */
export function adminThemeConfig(isDark: boolean): ThemeConfig {
  const primary = isDark ? "#8B82FF" : "#5143FF";
  const primaryHover = isDark ? "#A69FFF" : "#6B60FF";
  const primaryActive = isDark ? "#7065EB" : "#4334E6";
  const border = isDark ? "#30394D" : "#E8ECF4";

  return {
    algorithm: isDark ? theme.darkAlgorithm : theme.defaultAlgorithm,
    token: {
      colorPrimary: primary,
      colorPrimaryHover: primaryHover,
      colorPrimaryActive: primaryActive,
      colorPrimaryBg: isDark ? "rgba(99, 102, 241, 0.15)" : "#EEF2FF",
      colorPrimaryBgHover: isDark ? "rgba(99, 102, 241, 0.25)" : "#E0E7FF",
      colorLink: primary,
      colorLinkHover: primaryHover,
      colorLinkActive: isDark ? "#6366F1" : "#3730A3",
      colorSuccess: "#16A34A",
      colorWarning: "#D97706",
      colorError: "#DC2626",
      colorInfo: isDark ? "#818CF8" : "#4338CA",
      colorTextHeading: isDark ? "#F1F3FB" : "#192139",
      colorText: isDark ? "#DADEEB" : "#303A53",
      colorTextSecondary: isDark ? "#A8B1C9" : "#808DA8",
      colorTextTertiary: isDark ? "#919CB6" : "#8995AF",
      colorTextQuaternary: isDark ? "#65738F" : "#A7B0C2",
      colorTextPlaceholder: isDark ? "#8793AC" : "#8B96AD",
      colorBgLayout: isDark ? "#0B0F19" : "#F5F6F8",
      colorBgContainer: isDark ? "#111827" : "#FFFFFF",
      colorBgElevated: isDark ? "#1F2937" : "#FFFFFF",
      colorBorder: isDark ? "#3D4760" : "#DFE5F0",
      colorBorderSecondary: border,
      colorFillAlter: isDark ? "#20283B" : "#F6F7F9",
      borderRadius: 2,
    },
    components: {
      Button: {
        colorPrimary: primary,
        colorPrimaryHover: primaryHover,
        colorPrimaryActive: primaryActive,
        primaryShadow: "none",
        defaultShadow: "none",
      },
      Table: {
        headerBg: isDark ? "#20283B" : "#F1F4FA",
        headerColor: isDark ? "#F1F3FB" : "#263049",
        headerSplitColor: "transparent",
        borderColor: border,
        rowHoverBg: isDark ? "#252D43" : "#F8F9FF",
        rowSelectedBg: isDark ? "#2C2C4F" : "#F3F1FF",
        cellPaddingBlockMD: 10,
        cellPaddingInlineMD: 12,
      },
      Card: {
        colorBgContainer: isDark ? "#111827" : "#FFFFFF",
        colorBorderSecondary: isDark ? "#374151" : "#E5E7EB",
        borderRadiusLG: 8,
      },
      Menu: {
        itemSelectedBg: isDark ? "rgba(99, 102, 241, 0.18)" : "#EEF2FF",
        itemSelectedColor: isDark ? "#818CF8" : "#4338CA",
        itemHoverBg: isDark ? "#1F2937" : "#F3F4F6",
        itemHoverColor: isDark ? "#F9FAFB" : "#111827",
        itemColor: isDark ? "#9CA3AF" : "#374151",
        borderRadius: 6,
      },
      Tabs: {
        itemSelectedColor: isDark ? "#818CF8" : "#4338CA",
        itemHoverColor: isDark ? "#A5B4FC" : "#4F46E5",
        inkBarColor: isDark ? "#818CF8" : "#4338CA",
      },
      Switch: { trackHeight: 24, handleSize: 20, trackMinWidth: 66 },
      Pagination: { itemSize: 32 },
    },
  };
}
