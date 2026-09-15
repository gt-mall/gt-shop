import type { CSSProperties, ReactNode } from 'react';
import { Card } from 'antd';

export interface TableCardProps {
  children: ReactNode;
  className?: string;
  style?: CSSProperties;
  title?: ReactNode;
  extra?: ReactNode;
  bodyStyle?: CSSProperties;
}

export function TableCard({
  children,
  className = '',
  style,
  title,
  extra,
  bodyStyle,
}: TableCardProps) {
  return (
    <Card
      className={className}
      style={style}
      title={title}
      extra={extra}
      styles={{
        body: { padding: '16px 20px', ...bodyStyle },
      }}
    >
      {children}
    </Card>
  );
}
