import type { CSSProperties, ReactNode } from 'react';
import { theme } from 'antd';

export interface ContentCardProps {
  children: ReactNode;
  className?: string;
  style?: CSSProperties;
}

export function ContentCard({ children, className = '', style }: ContentCardProps) {
  const { token } = theme.useToken();

  return (
    <div
      className={`rounded-lg overflow-hidden ${className}`}
      style={{
        backgroundColor: token.colorBgContainer,
        border: `1px solid ${token.colorBorderSecondary}`,
        ...style,
      }}
    >
      {children}
    </div>
  );
}
