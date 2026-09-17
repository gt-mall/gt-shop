import React from 'react';
import { Card, Skeleton, Space, Tag, Tooltip, Typography, theme } from 'antd';
import { Info, TrendingDown, TrendingUp } from 'lucide-react';

const { Text } = Typography;

export interface StatCardProps {
  title: string;
  value: string | number;
  prefix?: string;
  suffix?: string;
  tooltip?: string;
  growthRate?: string;
  growthLabel?: string;
  icon?: React.ReactNode;
  loading?: boolean;
}

export const StatCard: React.FC<StatCardProps> = ({
  title,
  value,
  prefix,
  suffix,
  tooltip,
  growthRate,
  growthLabel = '环比',
  icon,
  loading = false,
}) => {
  const { token } = theme.useToken();

  const isPositive = growthRate && growthRate.startsWith('+');
  const isNegative = growthRate && growthRate.startsWith('-');

  return (
    <Card
      size="small"
      style={{
        borderRadius: 8,
        height: '100%',
        backgroundColor: token.colorBgContainer,
        borderColor: token.colorBorderSecondary,
      }}
      styles={{
        body: { padding: '16px 20px' },
      }}
    >
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
        <Space size={4} align="center">
          <Text type="secondary" style={{ fontSize: 13, fontWeight: 500 }}>
            {title}
          </Text>
          {tooltip && (
            <Tooltip title={tooltip}>
              <Info size={13} strokeWidth={1.8} style={{ color: token.colorTextQuaternary }} />
            </Tooltip>
          )}
        </Space>
        {icon && (
          <div
            style={{
              width: 32,
              height: 32,
              borderRadius: 6,
              backgroundColor: token.colorFillAlter,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: token.colorPrimary,
              fontSize: 16,
            }}
          >
            {icon}
          </div>
        )}
      </div>

      {loading ? (
        <Skeleton.Input active size="small" style={{ width: 100, height: 32 }} />
      ) : (
        <div style={{ display: 'flex', alignItems: 'baseline', gap: 4 }}>
          {prefix && (
            <span style={{ fontSize: 16, fontWeight: 600, color: token.colorText }}>
              {prefix}
            </span>
          )}
          <span style={{ fontSize: 24, fontWeight: 700, color: token.colorText, fontFamily: 'sans-serif' }}>
            {value}
          </span>
          {suffix && (
            <span style={{ fontSize: 13, color: token.colorTextSecondary }}>
              {suffix}
            </span>
          )}
        </div>
      )}

      {growthRate && (
        <div style={{ marginTop: 8, display: 'flex', alignItems: 'center', gap: 6 }}>
          <Text type="secondary" style={{ fontSize: 12 }}>
            {growthLabel}
          </Text>
          {isPositive ? (
            <Tag
              bordered={false}
              color="success"
              style={{ margin: 0, padding: '0 4px', fontSize: 12, display: 'inline-flex', alignItems: 'center', gap: 2 }}
            >
              <TrendingUp size={12} strokeWidth={2} />
              {growthRate}
            </Tag>
          ) : isNegative ? (
            <Tag
              bordered={false}
              color="error"
              style={{ margin: 0, padding: '0 4px', fontSize: 12, display: 'inline-flex', alignItems: 'center', gap: 2 }}
            >
              <TrendingDown size={12} strokeWidth={2} />
              {growthRate}
            </Tag>
          ) : (
            <Tag bordered={false} style={{ margin: 0, padding: '0 4px', fontSize: 12 }}>
              {growthRate}
            </Tag>
          )}
        </div>
      )}
    </Card>
  );
};
