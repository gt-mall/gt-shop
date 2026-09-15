import type { CSSProperties, ReactNode } from 'react';
import { Card } from 'antd';

export interface FilterCardProps {
  children: ReactNode;
  className?: string;
  style?: CSSProperties;
  bodyStyle?: CSSProperties;
}

export function FilterCard({ children, className = '', style, bodyStyle }: FilterCardProps) {
  return (
    <Card
      className={`mb-4 ${className}`}
      style={style}
      styles={{
        body: { padding: '16px 20px', ...bodyStyle },
      }}
    >
      {children}
    </Card>
  );
}
