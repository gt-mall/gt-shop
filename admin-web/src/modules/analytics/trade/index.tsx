import React, { useCallback, useEffect, useState } from 'react';
import { Button, Card, Col, Empty, Result, Row, Space, Table, Tag, Typography, message, theme } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  BadgeDollarSign,
  CheckCircle2,
  Gift,
  ReceiptText,
  RefreshCw,
  ShoppingCart,
  Truck,
  XCircle,
} from 'lucide-react';
import { Column, Line, Pie } from '@ant-design/charts';

import {
  api,
  type AnalyticsChannelDistributionItem,
  type AnalyticsOrderStatusDistributionItem,
  type AnalyticsTradeResponse,
} from '@/framework/http';
import { usePermission } from '@/framework/store';
import { PageHeader } from '@/shared/admin-ui';

import { TimeRangePicker } from '../components/time-range-picker';
import { StatCard } from '../components/stat-card';

const { Text } = Typography;

export default function AnalyticsTradePage() {
  const { token } = theme.useToken();
  const hasPermission = usePermission();
  const canView = hasPermission('analytics:trade:view');

  const [timeRange, setTimeRange] = useState('last_7_days');
  const [startTime, setStartTime] = useState<string>();
  const [endTime, setEndTime] = useState<string>();

  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<AnalyticsTradeResponse>();

  const loadData = useCallback(async () => {
    if (!canView) return;
    setLoading(true);

    const res = await api.api.getAnalyticsTrade({
      time_range: timeRange,
      start_time: startTime,
      end_time: endTime,
    });

    setLoading(false);
    if (res.code === 1 && res.data) {
      setData(res.data);
    } else {
      message.error(res.message || '获取交易分析数据失败');
    }
  }, [canView, timeRange, startTime, endTime]);

  useEffect(() => {
    void loadData();
  }, [loadData]);

  const handleTimeRangeChange = (r: string, s?: string, e?: string) => {
    setTimeRange(r);
    setStartTime(s);
    setEndTime(e);
  };

  if (!canView) {
    return (
      <div style={{ padding: '60px 0' }}>
        <Result
          status="403"
          title="暂无交易分析权限"
          subTitle="您当前所在的角色未被授予交易分析的查看权限，请联系管理员授权。"
        />
      </div>
    );
  }

  const metrics = data?.metrics;

  // Chart data: Order trends
  const orderTrendData = (data?.order_trends || []).flatMap((item) => [
    { date: item.date, count: item.order_count, type: '下单订单' },
    { date: item.date, count: item.paid_order_count, type: '支付订单' },
    { date: item.date, count: item.cancelled_order_count, type: '取消订单' },
  ]);

  const orderTrendConfig = {
    data: orderTrendData,
    xField: 'date',
    yField: 'count',
    colorField: 'type',
    shapeField: 'smooth',
    style: {
      lineWidth: 2.5,
    },
    axis: {
      y: { title: '订单量 (笔)' },
    },
  };

  // Chart data: Amount trends
  const amountTrendData = (data?.amount_trends || []).flatMap((item) => [
    { date: item.date, amount: parseFloat(item.pay_amount || '0'), type: '支付金额' },
    { date: item.date, amount: parseFloat(item.refund_amount || '0'), type: '退款金额' },
    { date: item.date, amount: parseFloat(item.net_sales_amount || '0'), type: '净销售额' },
  ]);

  const amountTrendConfig = {
    data: amountTrendData,
    xField: 'date',
    yField: 'amount',
    colorField: 'type',
    shapeField: 'smooth',
    style: {
      lineWidth: 2.5,
    },
    axis: {
      y: { title: '金额 (元)' },
    },
  };

  // Order status table & chart
  const orderStatusColumns: ColumnsType<AnalyticsOrderStatusDistributionItem> = [
    {
      title: '订单状态',
      dataIndex: 'status_text',
      key: 'status_text',
    },
    {
      title: '订单量 (笔)',
      dataIndex: 'count',
      key: 'count',
      align: 'right',
      render: (c: number) => <span style={{ fontWeight: 600 }}>{c}</span>,
    },
    {
      title: '占比',
      dataIndex: 'percentage',
      key: 'percentage',
      align: 'right',
      render: (p: string) => <Tag bordered={false}>{p}</Tag>,
    },
  ];

  // Channel distribution table
  const channelColumns: ColumnsType<AnalyticsChannelDistributionItem> = [
    {
      title: '支付渠道',
      dataIndex: 'channel_name',
      key: 'channel_name',
    },
    {
      title: '支付笔数',
      dataIndex: 'pay_count',
      key: 'pay_count',
      align: 'right',
      render: (c: number) => <span style={{ fontWeight: 600 }}>{c}</span>,
    },
    {
      title: '支付金额 (元)',
      dataIndex: 'pay_amount',
      key: 'pay_amount',
      align: 'right',
      render: (amt: string) => <span style={{ fontWeight: 600, color: token.colorPrimary }}>¥{amt}</span>,
    },
    {
      title: '金额占比',
      dataIndex: 'percentage',
      key: 'percentage',
      align: 'right',
      render: (p: string) => <Tag bordered={false} color="processing">{p}</Tag>,
    },
  ];

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      {/* 顶部标题与时间筛选 */}
      <PageHeader
        title="交易分析"
        description="深度剖析商城下单、支付、退款转化与支付渠道表现"
        extra={
          <Space wrap size={12} align="center">
            <TimeRangePicker
              value={timeRange}
              startTime={startTime}
              endTime={endTime}
              onChange={handleTimeRangeChange}
            />
            <Button icon={<RefreshCw size={16} strokeWidth={1.8} />} onClick={loadData} loading={loading}>
              刷新
            </Button>
          </Space>
        }
      />

      {/* 顶部指标 */}
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} md={8} lg={3}>
          <StatCard
            title="下单订单数"
            value={metrics?.order_count ?? 0}
            suffix="单"
            tooltip="选定周期内买家创建的所有订单数"
            icon={<ReceiptText size={18} strokeWidth={1.8} />}
            loading={loading}
          />
        </Col>
        <Col xs={24} sm={12} md={8} lg={3}>
          <StatCard
            title="支付订单数"
            value={metrics?.paid_order_count ?? 0}
            suffix="单"
            tooltip="选定周期内实际完成支付的订单数"
            icon={<CheckCircle2 size={18} strokeWidth={1.8} />}
            loading={loading}
          />
        </Col>
        <Col xs={24} sm={12} md={8} lg={3}>
          <StatCard
            title="取消订单数"
            value={metrics?.cancelled_order_count ?? 0}
            suffix="单"
            tooltip="选定周期内关闭或取消的订单数"
            icon={<XCircle size={18} strokeWidth={1.8} />}
            loading={loading}
          />
        </Col>
        <Col xs={24} sm={12} md={8} lg={4}>
          <StatCard
            title="支付金额"
            value={metrics?.pay_amount || '0.00'}
            prefix="¥"
            tooltip="选定周期内买家实际支付成功金额"
            icon={<BadgeDollarSign size={18} strokeWidth={1.8} />}
            loading={loading}
          />
        </Col>
        <Col xs={24} sm={12} md={8} lg={4}>
          <StatCard
            title="优惠金额"
            value={metrics?.discount_amount || '0.00'}
            prefix="¥"
            tooltip="支付成功订单累计减免的优惠总额"
            icon={<Gift size={18} strokeWidth={1.8} />}
            loading={loading}
          />
        </Col>
        <Col xs={24} sm={12} md={8} lg={3}>
          <StatCard
            title="运费金额"
            value={metrics?.freight_amount || '0.00'}
            prefix="¥"
            tooltip="支付成功订单累计收取的运费总额"
            icon={<Truck size={18} strokeWidth={1.8} />}
            loading={loading}
          />
        </Col>
        <Col xs={24} sm={12} md={8} lg={4}>
          <StatCard
            title="客单价"
            value={metrics?.average_order_value || '0.00'}
            prefix="¥"
            tooltip="支付金额 / 支付订单数"
            icon={<ShoppingCart size={18} strokeWidth={1.8} />}
            loading={loading}
          />
        </Col>
      </Row>

      {/* 趋势图表 */}
      <Row gutter={[16, 16]}>
        <Col xs={24} lg={12}>
          <Card
            title="订单趋势 (下单 / 支付 / 取消)"
            size="small"
            style={{ borderRadius: 8, height: '100%' }}
            styles={{ body: { padding: 16 } }}
          >
            {orderTrendData.length > 0 ? (
              <Line {...orderTrendConfig} height={280} />
            ) : (
              <div style={{ height: 280, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                <Empty description="暂无订单趋势" />
              </div>
            )}
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card
            title="交易金额趋势 (支付 / 退款 / 净销售额)"
            size="small"
            style={{ borderRadius: 8, height: '100%' }}
            styles={{ body: { padding: 16 } }}
          >
            {amountTrendData.length > 0 ? (
              <Line {...amountTrendConfig} height={280} />
            ) : (
              <div style={{ height: 280, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                <Empty description="暂无金额趋势" />
              </div>
            )}
          </Card>
        </Col>
      </Row>

      {/* 订单状态分布与支付渠道分布 */}
      <Row gutter={[16, 16]}>
        <Col xs={24} lg={12}>
          <Card
            title="订单状态分布"
            size="small"
            style={{ borderRadius: 8 }}
            styles={{ body: { padding: 0 } }}
          >
            <Table<AnalyticsOrderStatusDistributionItem>
              rowKey="status"
              dataSource={data?.order_status_distributions || []}
              columns={orderStatusColumns}
              pagination={false}
              size="small"
              locale={{ emptyText: <Empty description="暂无状态分布数据" /> }}
            />
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card
            title="支付渠道分布"
            size="small"
            style={{ borderRadius: 8 }}
            styles={{ body: { padding: 0 } }}
          >
            <Table<AnalyticsChannelDistributionItem>
              rowKey="channel"
              dataSource={data?.channel_distributions || []}
              columns={channelColumns}
              pagination={false}
              size="small"
              locale={{ emptyText: <Empty description="暂无渠道支付数据" /> }}
            />
          </Card>
        </Col>
      </Row>
    </div>
  );
}
