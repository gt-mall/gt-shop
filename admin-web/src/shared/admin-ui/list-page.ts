import type { CSSProperties } from "react";
import type { GlobalToken } from "antd";

/**
 * 把主题 Token 注入 list-page.less 需要的 CSS 变量，
 * 保证公共列表样式只维护一份，页面不重复写颜色映射。
 *
 * 用法：<div className="admin-list-page" style={{ backgroundColor: token.colorBgContainer, ...adminListVars(token) }}>
 */
export function adminListVars(token: GlobalToken): CSSProperties {
  return {
    "--admin-list-border": token.colorBorderSecondary,
    "--admin-list-primary": token.colorPrimary,
    "--admin-list-on-primary": token.colorTextLightSolid,
  } as CSSProperties;
}
