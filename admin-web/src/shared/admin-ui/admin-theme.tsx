import type { ReactNode } from "react";
import { useContext } from "react";
import type { ThemeConfig } from "antd";
import { ConfigProvider, theme } from "antd";

/**
 * 后台内容区视觉基线：主色、文字层级、控件高度、圆角与常用组件密度。
 * 明暗两套取值都收敛在这里，业务模块不再各自维护 Theme。
 */
export function adminThemeConfig(isDark: boolean): ThemeConfig {
  const primary = isDark ? "#8B82FF" : "#5143FF";
  const primaryHover = isDark ? "#A69FFF" : "#6B60FF";
  const primaryActive = isDark ? "#7065EB" : "#4334E6";
  const border = isDark ? "#30394D" : "#E8ECF4";

  return {
    token: {
      colorPrimary: primary,
      colorPrimaryHover: primaryHover,
      colorPrimaryActive: primaryActive,
      colorLink: primary,
      colorLinkHover: primaryHover,
      colorTextHeading: isDark ? "#F1F3FB" : "#192139",
      colorText: isDark ? "#DADEEB" : "#303A53",
      colorTextSecondary: isDark ? "#A8B1C9" : "#808DA8",
      colorTextTertiary: isDark ? "#919CB6" : "#8995AF",
      colorTextQuaternary: isDark ? "#65738F" : "#A7B0C2",
      colorTextPlaceholder: isDark ? "#8793AC" : "#8B96AD",
      colorBorder: isDark ? "#3D4760" : "#DFE5F0",
      colorBorderSecondary: border,
      colorFillAlter: isDark ? "#20283B" : "#F6F7F9",
      controlHeight: 40,
      controlHeightSM: 32,
      borderRadius: 6,
    },
    components: {
      Button: {
        colorPrimary: primary,
        colorPrimaryHover: primaryHover,
        colorPrimaryActive: primaryActive,
        controlHeight: 40,
        controlHeightLG: 42,
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
      Switch: { trackHeight: 24, handleSize: 20, trackMinWidth: 66 },
      Pagination: { itemSize: 32 },
    },
  };
}

/** 明暗模式继承外层 Layout 的 ConfigProvider，不单独维护主题开关。 */
function useIsDarkTheme() {
  const { theme: parentTheme } = useContext(ConfigProvider.ConfigContext);
  const algorithms = [parentTheme?.algorithm].flat();
  return algorithms.includes(theme.darkAlgorithm);
}

/**
 * 后台页面视觉基线容器。列表页、详情页等直接包一层即可与已定稿页面保持一致，
 * 不需要为自己的模块新建 Theme。
 */
export function AdminThemeProvider({ children }: { children: ReactNode }) {
  const isDark = useIsDarkTheme();

  return (
    <ConfigProvider theme={adminThemeConfig(isDark)}>{children}</ConfigProvider>
  );
}
