import type { ReactNode } from 'react';
import { theme } from 'antd';

export interface PageHeaderProps {
  title: ReactNode;
  description?: ReactNode;
  extra?: ReactNode;
  className?: string;
  style?: React.CSSProperties;
}

export function PageHeader({
  title,
  description,
  extra,
  className = '',
  style,
}: PageHeaderProps) {
  const { token } = theme.useToken();

  return (
    <div
      className={`flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-4 ${className}`}
      style={style}
    >
      <div className="min-w-0">
        <h1
          className="text font-semibold tracking-tight leading-tight m-0"
          style={{ color: token.colorTextHeading }}
        >
          {title}
        </h1>
        {description && (
          <p
            className="text-sm mt-1.5 mb-0 leading-normal"
            style={{ color: token.colorTextSecondary }}
          >
            {description}
          </p>
        )}
      </div>
      {extra && <div className="flex items-center gap-3 shrink-0">{extra}</div>}
    </div>
  );
}
