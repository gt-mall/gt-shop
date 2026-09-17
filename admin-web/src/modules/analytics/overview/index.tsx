import React, { useCallback, useEffect, useState } from 'react';
import { Button, Card, Col, Empty, Result, Row, Space, Spin, Table, Tag, Typography, message, theme } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  BadgeDollarSign,
  CircleDollarSign,
  PackageSearch,
  ReceiptText,
  RefreshCw,
  RotateCcw,
  ShoppingCart,
  Users,
} from 'lucide-react';
import { Column, DualAxes, Line } from '@ant-design/charts';

import {
  api,
  type AnalyticsOverviewResponse,
  type AnalyticsRecentFlowItem,
  type AnalyticsTopProductItem,
} from '@/framework/http';
import { usePermission } from '@/framework/store';
import { PageHeader } from '@/shared/admin-ui';
import { formatTime } from '@/shared/utils/format';

import { TimeRangePicker } from '../components/time-range-picker';
import { StatCard } from '../components/stat-card';

const { Text } = Typography;

export default function AnalyticsOverviewPage() {
  const { token } = theme.useToken();
  const hasPermission = usePermission();
  const canView = hasPermission('analytics:overview:view');

  const [timeRange, setTimeRange] = useState('today');
  const [startTime, setStartTime] = useState<string>();
  const [endTime, setEndTime] = useState<string>();

  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<AnalyticsOverviewResponse>();

  const loadData = useCallback(async () => {
    if (!canView) return;
    setLoading(true);

    const res = await api.api.getAnalyticsOverview({
      time_range: timeRange,
      start_time: startTime,
      end_time: endTime,
    });

    setLoading(false);
    if (res.code === 1 && res.data) {
      setData(res.data);
    } else {
      message.error(res.message || '获取经营概览数据失败');
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
          title="暂无经营概览权限"
          subTitle="您当前所在的角色未被授予经营概览的查看权限，请联系管理员授权。"
        />
      </div>
    );
  }

  const metrics = data?.metrics;

  // Chart configs
  const salesTrendData = (data?.sales_trends || []).flatMap((item) => [
    { date: item.date, amount: parseFloat(item.pay_amount || '0'), type: '支付金额' },
    { date: item.date, amount: parseFloat(item.net_sales_amount || '0'), type: '净销售额' },
  ]);

  const salesTrendConfig = {
    data: salesTrendData,
    xField: 'date',
    yField: 'amount',
    colorField: 'type',
    shapeField: 'smooth',
    style: {
      lineWidth: 2.5,
    },
    axis: {
      y: {
        title: '金额 (元)',
      },
    },
  };

  const orderTrendData = (data?.order_trends || []).flatMap((item) => [
    { date: item.date, count: item.order_count, type: '下单订单数' },
    { date: item.date, count: item.paid_order_count, type: '支付订单数' },
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
      y: {
        title: '订单量 (笔)',
      },
    },
  };

  // Top products columns
  const productColumns: ColumnsType<AnalyticsTopProductItem> = [
    {
      title: '排名',
      key: 'rank',
      width: 60,
      render: (_, __, index) => (
        <span
          style={{
            display: 'inline-block',
            width: 20,
            height: 20,
            lineHeight: '20px',
            textAlign: 'center',
            borderRadius: '50%',
            fontSize: 12,
            fontWeight: 600,
            backgroundColor: index < 3 ? token.colorPrimary : token.colorFillSecondary,
            color: index < 3 ? '#fff' : token.colorTextSecondary,
          }}
        >
          {index + 1}
        </span>
      ),
    },
    {
      title: '商品名称',
      dataIndex: 'product_name',
      key: 'product_name',
      ellipsis: true,
      render: (name: string, record) => (
        <Space size={8}>
          {record.image_url ? (
            <img
              src={record.image_url}
              alt=""
              style={{ width: 36, height: 36, borderRadius: 4, objectFit: 'cover' }}
            />
          ) : (
            <div
              style={{
                width: 36,
                height: 36,
                borderRadius: 4,
                backgroundColor: token.colorFillAlter,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                color: token.colorTextQuaternary,
              }}
            >
              <PackageSearch size={18} strokeWidth={1.8} />
            </div>
          )}
          <Text strong style={{ fontSize: 13 }} ellipsis={{ tooltip: name }}>
            {name}
          </Text>
        </Space>
      ),
    },
    {
      title: '销量',
      dataIndex: 'quantity',
      key: 'quantity',
      width: 90,
      align: 'right',
      render: (v: number) => <span style={{ fontWeight: 600 }}>{v}</span>,
    },
    {
      title: '销售额 (元)',
      dataIndex: 'sales_amount',
      key: 'sales_amount',
      width: 120,
      align: 'right',
      render: (v: string) => <span style={{ fontWeight: 600, color: token.colorPrimary }}>¥{v}</span>,
    },
    {
      title: '支付订单数',
      dataIndex: 'paid_order_count',
      key: 'paid_order_count',
      width: 100,
      align: 'right',
    },
  ];

  // Recent flows columns
  const flowColumns: ColumnsType<AnalyticsRecentFlowItem> = [
    {
      title: '类型',
      dataIndex: 'flow_type',
      key: 'flow_type',
      width: 80,
      render: (type: string, record) => (
        <Tag color={record.direction === 'income' ? 'success' : 'error'} bordered={false}>
          {record.direction === 'income' ? '收入' : '退款'}
        </Tag>
      ),
    },
    {
      title: '流水单号',
      dataIndex: 'biz_no',
      key: 'biz_no',
      width: 160,
      render: (no: string) => <Text copyable={{ text: no }}>{no}</Text>,
    },
    {
      title: '订单号',
      dataIndex: 'order_no',
      key: 'order_no',
      width: 160,
      render: (no: string) => no || '-',
    },
    {
      title: '金额',
      dataIndex: 'amount',
      key: 'amount',
      width: 110,
      align: 'right',
      render: (amt: string, record) => (
        <span
          style={{
            fontWeight: 700,
            color: record.direction === 'income' ? token.colorSuccess : token.colorError,
          }}
        >
          {record.direction === 'income' ? `+${amt}` : `-${amt}`}
        </span>
      ),
    },
    {
      title: '渠道',
      dataIndex: 'channel',
      key: 'channel',
      width: 90,
      render: (c: string) => (c === 'wechat' ? '微信支付' : c === 'mock' ? '模拟支付' : c || '-'),
    },
    {
      title: '发生时间',
      dataIndex: 'occurred_at',
      key: 'occurred_at',
      width: 160,
      render: (t?: string) => (t ? formatTime(t) : '-'),
    },
  ];

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      {/* 顶部标题与时间筛选 */}
      <PageHeader
        title="经营概览"
        description="实时掌握商城经营全局动态、核心交易指标与近期资金走势"
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

      {/* 核心指标卡片 */}
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} md={8} lg={4}>
          <StatCard
            title="支付金额"
            value={metrics?.pay_amount || '0.00'}
            prefix="¥"
            tooltip="选定统计周期内实际支付成功的交易金额总和（以 paid_at 为准）"
            growthRate={metrics?.pay_amount_growth_rate}
            icon={<BadgeDollarSign size={18} strokeWidth={1.8} />}
            loading={loading}
          />
        </Col>
        <Col xs={24} sm={12} md={8} lg={4}>
          <StatCard
            title="退款金额"
            value={metrics?.refund_amount || '0.00'}
            prefix="¥"
            tooltip="选定统计周期内退款成功的资金总和（以 refunded_at 为准）"
            growthRate={metrics?.refund_amount_growth_rate}
            icon={<RotateCcw size={18} strokeWidth={1.8} />}
            loading={loading}
          />
        </Col>
        <Col xs={24} sm={12} md={8} lg={4}>
          <StatCard
            title="净销售额"
            value={metrics?.net_sales_amount || '0.00'}
            prefix="¥"
            tooltip="净销售额 = 支付成功金额 - 退款成功金额"
            growthRate={metrics?.net_sales_growth_rate}
            icon={<CircleDollarSign size={18} strokeWidth={1.8} />}
            loading={loading}
          />
        </Col>
        <Col xs={24} sm={12} md={8} lg={4}>
          <StatCard
            title="支付订单数"
            value={metrics?.paid_order_count ?? 0}
            suffix="单"
            tooltip="选定周期内完成支付的去重订单数"
            growthRate={metrics?.paid_order_count_growth_rate}
            icon={<ReceiptText size={18} strokeWidth={1.8} />}
            loading={loading}
          />
        </Col>
        <Col xs={24} sm={12} md={8} lg={4}>
          <StatCard
            title="支付买家数"
            value={metrics?.paid_member_count ?? 0}
            suffix="人"
            tooltip="选定周期内有成功支付行为的去重会员数"
            growthRate={metrics?.paid_member_count_growth_rate}
            icon={<Users size={18} strokeWidth={1.8} />}
            loading={loading}
          />
        </Col>
        <Col xs={24} sm={12} md={8} lg={4}>
          <StatCard
            title="客单价"
            value={metrics?.average_order_value || '0.00'}
            prefix="¥"
            tooltip="客单价 = 支付金额 / 支付订单数"
            growthRate={metrics?.aov_growth_rate}
            icon={<ShoppingCart size={18} strokeWidth={1.8} />}
            loading={loading}
          />
        </Col>
      </Row>

      {/* 趋势图表区 */}
      <Row gutter={[16, 16]}>
        <Col xs={24} lg={12}>
          <Card
            title="销售趋势"
            size="small"
            style={{ borderRadius: 8, height: '100%' }}
            styles={{ body: { padding: 16 } }}
          >
            {salesTrendData.length > 0 ? (
              <Line {...salesTrendConfig} height={280} />
            ) : (
              <div style={{ height: 280, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                <Empty description="暂无趋势数据" />
              </div>
            )}
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card
            title="订单趋势"
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
      </Row>

      {/* 排行与最近流水 */}
      <Row gutter={[16, 16]}>
        <Col xs={24} lg={12}>
          <Card
            title="商品销售排行 TOP 10"
            size="small"
            style={{ borderRadius: 8 }}
            styles={{ body: { padding: 0 } }}
          >
            <Table<AnalyticsTopProductItem>
              rowKey="product_id"
              dataSource={data?.top_products || []}
              columns={productColumns}
              pagination={false}
              size="small"
              locale={{ emptyText: <Empty description="暂无商品销售数据" /> }}
            />
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card
            title="最近资金动态"
            size="small"
            style={{ borderRadius: 8 }}
            styles={{ body: { padding: 0 } }}
          >
            <Table<AnalyticsRecentFlowItem>
              rowKey={(r) => `${r.flow_type}_${r.id}`}
              dataSource={data?.recent_flows || []}
              columns={flowColumns}
              pagination={false}
              size="small"
              locale={{ emptyText: <Empty description="暂无资金流水记录" /> }}
            />
          </Card>
        </Col>
      </Row>
    </div>
  );
}
