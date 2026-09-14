import React, { useMemo, useState } from 'react';
import { Card, Empty, Radio, Segmented, Space, Spin, Statistic, Tag, Typography, theme } from 'antd';
import { Area, Column, DualAxes } from '@ant-design/charts';
import type { DashboardTrendResponse } from '@/framework/http/generated_api';

const { Text } = Typography;

interface SalesChartProps {
  data?: DashboardTrendResponse;
  loading?: boolean;
  days: number;
  onDaysChange: (days: number) => void;
}

export const SalesChart: React.FC<SalesChartProps> = ({
  data,
  loading,
  days,
  onDaysChange,
}) => {
  const { token } = theme.useToken();
  const [metricType, setMetricType] = useState<'both' | 'sales' | 'orders'>('both');

  const dates = data?.dates ?? [];
  const orderCounts = data?.order_counts ?? [];
  const salesAmounts = data?.sales_amounts ?? [];

  // Summary statistics for the chosen period
  const totalSales = useMemo(() => {
    return salesAmounts.reduce((acc, curr) => acc + (parseFloat(curr) || 0), 0);
  }, [salesAmounts]);

  const totalOrders = useMemo(() => {
    return orderCounts.reduce((acc, curr) => acc + curr, 0);
  }, [orderCounts]);

  const avgSales = useMemo(() => {
    return dates.length > 0 ? (totalSales / dates.length).toFixed(2) : '0.00';
  }, [totalSales, dates.length]);

  const avgOrders = useMemo(() => {
    return dates.length > 0 ? (totalOrders / dates.length).toFixed(1) : '0';
  }, [totalOrders, dates.length]);

  // Transform data for DualAxes or individual charts
  const dualData = useMemo(() => {
    return dates.map((date, index) => ({
      date,
      sales: parseFloat(salesAmounts[index] || '0') || 0,
      orders: orderCounts[index] || 0,
    }));
  }, [dates, salesAmounts, orderCounts]);

  const salesData = useMemo(() => {
    return dates.map((date, index) => ({
      date,
      value: parseFloat(salesAmounts[index] || '0') || 0,
      category: '销售额 (元)',
    }));
  }, [dates, salesAmounts]);

  const ordersData = useMemo(() => {
    return dates.map((date, index) => ({
      date,
      value: orderCounts[index] || 0,
      category: '订单量 (单)',
    }));
  }, [dates, orderCounts]);

  const dualAxesConfig = {
    data: dualData,
    xField: 'date',
    children: [
      {
        type: 'interval',
        yField: 'sales',
        style: {
          fill: token.colorPrimary,
          maxWidth: 24,
          radius: [4, 4, 0, 0],
        },
        axis: {
          y: {
            title: '销售额 (元)',
            titleFontSize: 12,
          },
        },
      },
      {
        type: 'line',
        yField: 'orders',
        style: {
          stroke: token.colorSuccess,
          lineWidth: 2.5,
        },
        axis: {
          y: {
            position: 'right',
            title: '订单量 (单)',
            titleFontSize: 12,
          },
        },
      },
    ],
    interaction: {
      tooltip: {
        shared: true,
      },
    },
    theme: token.colorBgContainer === '#0B0F19' ? 'dark' : 'light',
  };

  const salesAreaConfig = {
    data: salesData,
    xField: 'date',
    yField: 'value',
    style: {
      fill: `linear-gradient(-90deg, ${token.colorPrimaryBg} 0%, ${token.colorPrimary} 100%)`,
      stroke: token.colorPrimary,
      lineWidth: 2,
    },
    axis: {
      y: {
        title: '销售额 (元)',
      },
    },
    interaction: {
      tooltip: {
        render: (_: any, items: any[]) => {
          if (!items || items.length === 0) return null;
          const item = items[0];
          return (
            <div style={{ padding: 8 }}>
              <div>{item.title}</div>
              <div style={{ color: token.colorPrimary, fontWeight: 600 }}>
                销售额：¥{(Number(item.value) || 0).toFixed(2)}
              </div>
            </div>
          );
        },
      },
    },
  };

  const ordersColumnConfig = {
    data: ordersData,
    xField: 'date',
    yField: 'value',
    style: {
      fill: token.colorSuccess,
      maxWidth: 32,
      radius: [4, 4, 0, 0],
    },
    axis: {
      y: {
        title: '订单量 (单)',
      },
    },
  };

  return (
    <Card
      variant="outlined"
      style={{ borderRadius: 8 }}
      title={
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: 12 }}>
          <Space align="center">
            <span style={{ fontWeight: 600, fontSize: 16 }}>销售趋势</span>
            <Tag color="blue">{days === 7 ? '近 7 天' : '近 30 天'}</Tag>
          </Space>
          <Space wrap>
            <Segmented
              size="small"
              value={metricType}
              onChange={(val) => setMetricType(val as 'both' | 'sales' | 'orders')}
              options={[
                { label: '综合趋势', value: 'both' },
                { label: '销售额', value: 'sales' },
                { label: '订单量', value: 'orders' },
              ]}
            />
            <Radio.Group
              size="small"
              value={days}
              onChange={(e) => onDaysChange(e.target.value)}
              buttonStyle="solid"
            >
              <Radio.Button value={7}>近 7 天</Radio.Button>
              <Radio.Button value={30}>近 30 天</Radio.Button>
            </Radio.Group>
          </Space>
        </div>
      }
    >
      {/* 周期统计指标摘要 */}
      <div
        style={{
          display: 'flex',
          gap: 32,
          padding: '10px 16px',
          marginBottom: 16,
          backgroundColor: token.colorFillAlter,
          borderRadius: 6,
          flexWrap: 'wrap',
        }}
      >
        <Statistic
          title={<Text type="secondary" style={{ fontSize: 12 }}>区间总销售额</Text>}
          value={totalSales.toFixed(2)}
          prefix="¥"
          valueStyle={{ fontSize: 18, fontWeight: 600, color: token.colorPrimary }}
        />
        <Statistic
          title={<Text type="secondary" style={{ fontSize: 12 }}>区间总订单量</Text>}
          value={totalOrders}
          suffix="单"
          valueStyle={{ fontSize: 18, fontWeight: 600, color: token.colorSuccess }}
        />
        <Statistic
          title={<Text type="secondary" style={{ fontSize: 12 }}>日均销售额</Text>}
          value={avgSales}
          prefix="¥"
          valueStyle={{ fontSize: 18, fontWeight: 600 }}
        />
        <Statistic
          title={<Text type="secondary" style={{ fontSize: 12 }}>日均订单量</Text>}
          value={avgOrders}
          suffix="单"
          valueStyle={{ fontSize: 18, fontWeight: 600 }}
        />
      </div>

      {/* 图表展示区 */}
      <Spin spinning={loading}>
        {dates.length === 0 ? (
          <Empty description="暂无趋势数据" style={{ padding: '60px 0' }} />
        ) : (
          <div style={{ height: 340 }}>
            {metricType === 'both' && <DualAxes {...dualAxesConfig} />}
            {metricType === 'sales' && <Area {...salesAreaConfig} />}
            {metricType === 'orders' && <Column {...ordersColumnConfig} />}
          </div>
        )}
      </Spin>
    </Card>
  );
};
