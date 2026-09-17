import React, { useCallback, useEffect, useState } from 'react';
import {
  Button,
  Card,
  Col,
  Descriptions,
  Empty,
  Result,
  Row,
  Space,
  Table,
  Tag,
  Typography,
  message,
  theme,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  BadgeDollarSign,
  RefreshCw,
  ShoppingCart,
  User,
  UserCheck,
  UserPlus,
  Users,
  UsersRound,
} from 'lucide-react';
import { Line } from '@ant-design/charts';

import {
  api,
  type AnalyticsMemberRankingItem,
  type AnalyticsMemberResponse,
} from '@/framework/http';
import { usePermission } from '@/framework/store';
import { PageHeader } from '@/shared/admin-ui';
import { formatTime } from '@/shared/utils/format';

import { TimeRangePicker } from '../components/time-range-picker';
import { StatCard } from '../components/stat-card';

const { Text } = Typography;

export default function AnalyticsMemberPage() {
  const { token } = theme.useToken();
  const hasPermission = usePermission();
  const canView = hasPermission('analytics:member:view');

  const [timeRange, setTimeRange] = useState('last_7_days');
  const [startTime, setStartTime] = useState<string>();
  const [endTime, setEndTime] = useState<string>();

  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<AnalyticsMemberResponse>();

  const loadData = useCallback(async () => {
    if (!canView) return;
    setLoading(true);

    const res = await api.api.getAnalyticsMember({
      time_range: timeRange,
      start_time: startTime,
      end_time: endTime,
    });

    setLoading(false);
    if (res.code === 1 && res.data) {
      setData(res.data);
    } else {
      message.error(res.message || '获取会员分析数据失败');
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
          title="暂无会员分析权限"
          subTitle="您当前所在的角色未被授予会员分析的查看权限，请联系管理员授权。"
        />
      </div>
    );
  }

  const metrics = data?.metrics;
  const segment = data?.customer_segment;

  // Chart data: Member growth & active trend
  const trendData = (data?.trends || []).flatMap((item) => [
    { date: item.date, count: item.new_member_count, type: '新增会员' },
    { date: item.date, count: item.paid_member_count, type: '支付会员' },
  ]);

  const trendConfig = {
    data: trendData,
    xField: 'date',
    yField: 'count',
    colorField: 'type',
    shapeField: 'smooth',
    style: {
      lineWidth: 2.5,
    },
    axis: {
      y: { title: '人数 (人)' },
    },
  };

  // Top members columns
  const topMemberColumns: ColumnsType<AnalyticsMemberRankingItem> = [
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
      title: '会员',
      key: 'member_info',
      render: (_, record) => (
        <Space size={8}>
          <div
            style={{
              width: 32,
              height: 32,
              borderRadius: '50%',
              backgroundColor: token.colorFillAlter,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: token.colorPrimary,
            }}
          >
            <User size={16} strokeWidth={1.8} />
          </div>
          <div>
            <div style={{ fontWeight: 600, fontSize: 13 }}>
              {record.nickname || '微信用户'}
            </div>
            {record.mobile && (
              <Text type="secondary" style={{ fontSize: 12 }}>
                {record.mobile}
              </Text>
            )}
          </div>
        </Space>
      ),
    },
    {
      title: '支付订单数',
      dataIndex: 'paid_order_count',
      key: 'paid_order_count',
      width: 120,
      align: 'right',
      render: (v: number) => <span style={{ fontWeight: 600 }}>{v}</span>,
    },
    {
      title: '累计支付金额 (元)',
      dataIndex: 'total_pay_amount',
      key: 'total_pay_amount',
      width: 160,
      align: 'right',
      render: (amt: string) => (
        <span style={{ fontWeight: 700, color: token.colorPrimary }}>¥{amt}</span>
      ),
    },
    {
      title: '最近支付时间',
      dataIndex: 'last_paid_at',
      key: 'last_paid_at',
      width: 180,
      render: (t?: string) => (t ? formatTime(t) : '-'),
    },
  ];

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      {/* 顶部标题与时间筛选 */}
      <PageHeader
        title="会员分析"
        description="洞察会员新增走势、新老客户付费结构与高价值买家"
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
        <Col xs={24} sm={12} md={8} lg={4}>
          <StatCard
            title="新增会员"
            value={metrics?.new_member_count ?? 0}
            suffix="人"
            tooltip="选定统计周期内注册的新会员数"
            icon={<UserPlus size={18} strokeWidth={1.8} />}
            loading={loading}
          />
        </Col>
        <Col xs={24} sm={12} md={8} lg={4}>
          <StatCard
            title="支付会员"
            value={metrics?.paid_member_count ?? 0}
            suffix="人"
            tooltip="选定周期内有成功支付行为的去重会员数"
            icon={<Users size={18} strokeWidth={1.8} />}
            loading={loading}
          />
        </Col>
        <Col xs={24} sm={12} md={8} lg={4}>
          <StatCard
            title="新增支付会员"
            value={metrics?.new_paid_member_count ?? 0}
            suffix="人"
            tooltip="在所选周期内首次产生支付成功的会员数（此前从未购买）"
            icon={<UserCheck size={18} strokeWidth={1.8} />}
            loading={loading}
          />
        </Col>
        <Col xs={24} sm={12} md={8} lg={4}>
          <StatCard
            title="累计会员"
            value={metrics?.total_member_count ?? 0}
            suffix="人"
            tooltip="截至统计周期末系统注册会员总人数"
            icon={<UsersRound size={18} strokeWidth={1.8} />}
            loading={loading}
          />
        </Col>
        <Col xs={24} sm={12} md={8} lg={4}>
          <StatCard
            title="人均消费金额"
            value={metrics?.average_member_spending || '0.00'}
            prefix="¥"
            tooltip="支付金额 / 支付会员数"
            icon={<BadgeDollarSign size={18} strokeWidth={1.8} />}
            loading={loading}
          />
        </Col>
        <Col xs={24} sm={12} md={8} lg={4}>
          <StatCard
            title="人均订单数"
            value={metrics?.average_member_order_count || '0.00'}
            suffix="单"
            tooltip="支付订单数 / 支付会员数"
            icon={<ShoppingCart size={18} strokeWidth={1.8} />}
            loading={loading}
          />
        </Col>
      </Row>

      {/* 趋势与新老客分析 */}
      <Row gutter={[16, 16]}>
        <Col xs={24} lg={14}>
          <Card
            title="新增与支付会员走势"
            size="small"
            style={{ borderRadius: 8, height: '100%' }}
            styles={{ body: { padding: 16 } }}
          >
            {trendData.length > 0 ? (
              <Line {...trendConfig} height={280} />
            ) : (
              <div style={{ height: 280, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                <Empty description="暂无趋势数据" />
              </div>
            )}
          </Card>
        </Col>
        <Col xs={24} lg={10}>
          <Card
            title="新老客户付费结构"
            size="small"
            style={{ borderRadius: 8, height: '100%' }}
            styles={{ body: { padding: 16 } }}
          >
            <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
              <Card
                size="small"
                style={{
                  backgroundColor: token.colorFillAlter,
                  border: `1px solid ${token.colorBorderSecondary}`,
                  borderRadius: 6,
                }}
              >
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
                  <Space size={6}>
                    <Tag color="green" bordered={false}>新客户</Tag>
                    <Text type="secondary" style={{ fontSize: 12 }}>统计周期内首次支付</Text>
                  </Space>
                  <Tag color="success" bordered={false}>{segment?.new_customer_percentage || '0.00%'}</Tag>
                </div>
                <Row gutter={12}>
                  <Col span={8}>
                    <div style={{ fontSize: 12, color: token.colorTextSecondary }}>客户人数</div>
                    <div style={{ fontSize: 18, fontWeight: 700 }}>{segment?.new_customer_count ?? 0} 人</div>
                  </Col>
                  <Col span={8}>
                    <div style={{ fontSize: 12, color: token.colorTextSecondary }}>支付金额</div>
                    <div style={{ fontSize: 18, fontWeight: 700, color: token.colorPrimary }}>¥{segment?.new_customer_pay_amount || '0.00'}</div>
                  </Col>
                  <Col span={8}>
                    <div style={{ fontSize: 12, color: token.colorTextSecondary }}>订单数</div>
                    <div style={{ fontSize: 18, fontWeight: 700 }}>{segment?.new_customer_order_count ?? 0} 单</div>
                  </Col>
                </Row>
              </Card>

              <Card
                size="small"
                style={{
                  backgroundColor: token.colorFillAlter,
                  border: `1px solid ${token.colorBorderSecondary}`,
                  borderRadius: 6,
                }}
              >
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
                  <Space size={6}>
                    <Tag color="blue" bordered={false}>老客户</Tag>
                    <Text type="secondary" style={{ fontSize: 12 }}>周期前已有历史支付记录</Text>
                  </Space>
                  <Tag color="processing" bordered={false}>{segment?.old_customer_percentage || '0.00%'}</Tag>
                </div>
                <Row gutter={12}>
                  <Col span={8}>
                    <div style={{ fontSize: 12, color: token.colorTextSecondary }}>客户人数</div>
                    <div style={{ fontSize: 18, fontWeight: 700 }}>{segment?.old_customer_count ?? 0} 人</div>
                  </Col>
                  <Col span={8}>
                    <div style={{ fontSize: 12, color: token.colorTextSecondary }}>支付金额</div>
                    <div style={{ fontSize: 18, fontWeight: 700, color: token.colorPrimary }}>¥{segment?.old_customer_pay_amount || '0.00'}</div>
                  </Col>
                  <Col span={8}>
                    <div style={{ fontSize: 12, color: token.colorTextSecondary }}>订单数</div>
                    <div style={{ fontSize: 18, fontWeight: 700 }}>{segment?.old_customer_order_count ?? 0} 单</div>
                  </Col>
                </Row>
              </Card>
            </div>
          </Card>
        </Col>
      </Row>

      {/* 消费会员排行 */}
      <Card
        title="消费会员排行 TOP 10"
        size="small"
        style={{ borderRadius: 8 }}
        styles={{ body: { padding: 0 } }}
      >
        <Table<AnalyticsMemberRankingItem>
          rowKey="member_id"
          dataSource={data?.top_members || []}
          columns={topMemberColumns}
          pagination={false}
          size="small"
          locale={{ emptyText: <Empty description="暂无会员消费排行" /> }}
        />
      </Card>
    </div>
  );
}
