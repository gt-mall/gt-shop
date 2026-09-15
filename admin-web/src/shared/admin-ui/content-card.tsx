import type { CSSProperties, ReactNode } from "react";
import { theme } from "antd";

export interface ContentCardProps {
  children: ReactNode;
  className?: string;
  style?: CSSProperties;
}

export function ContentCard({
  children,
  className = "",
  style,
}: ContentCardProps) {
  const { token } = theme.useToken();

  // 后台主内容容器：与定稿的商品列表一致，用背景明度区分层级，默认不加边框。
  return (
    <div
      className={`rounded-lg overflow-hidden ${className}`}
      style={{
        backgroundColor: token.colorBgContainer,
        ...style,
      }}
    >
      {children}
    </div>
  );
}
