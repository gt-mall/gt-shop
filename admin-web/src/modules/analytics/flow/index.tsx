import React, { useCallback, useEffect, useState } from 'react';
import {
  Button,
  Card,
  Col,
  DatePicker,
  Empty,
  Input,
  Result,
  Row,
  Select,
  Space,
  Table,
  Tabs,
  Tag,
  Typography,
  message,
  theme,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  CircleDollarSign,
  RefreshCw,
  RotateCcw,
  Search,
  TrendingDown,
  TrendingUp,
} from 'lucide-react';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';

import {
  api,
  type AnalyticsFinanceFlowRow,
  type AnalyticsFlowListResponse,
} from '@/framework/http';
import { usePermission } from '@/framework/store';
import { ContentCard, PageHeader } from '@/shared/admin-ui';
import { formatTime } from '@/shared/utils/format';

import { StatCard } from '../components/stat-card';

const { Text } = Typography;

export default function AnalyticsFlowPage() {
  const { token } = theme.useToken();
  const hasPermission = usePermission();
  const canView = hasPermission('analytics:flow:view');

  // Filter states
  const [tabKey, setTabKey] = useState<'all' | 'payment' | 'refund'>('all');
  const [bizNo, setBizNo] = useState('');
  const [orderNo, setOrderNo] = useState('');
  const [channel, setChannel] = useState<string>();
  const [status, setStatus] = useState<number>();
  const [dateRange, setDateRange] = useState<[Dayjs | null, Dayjs | null] | null>(null);

  // Pagination
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);

  // Data states
  const [loading, setLoading] = useState(false);
  const [flowRes, setFlowRes] = useState<AnalyticsFlowListResponse>();

  const loadData = useCallback(async () => {
    if (!canView) return;
    setLoading(true);

    let startTime: string | undefined;
    let endTime: string | undefined;
    if (dateRange && dateRange[0] && dateRange[1]) {
      startTime = dateRange[0].startOf('day').format('YYYY-MM-DD HH:mm:ss');
      endTime = dateRange[1].endOf('day').format('YYYY-MM-DD HH:mm:ss');
    }

    const res = await api.api.listAnalyticsFlows({
      page,
      page_size: pageSize,
      flow_type: tabKey,
      biz_no: bizNo || undefined,
      order_no: orderNo || undefined,
      channel: channel || undefined,
      status: tabKey !== 'all' && status !== undefined ? (status as any) : undefined,
      start_time: startTime,
      end_time: endTime,
    });

    setLoading(false);
    if (res.code === 1 && res.data) {
      setFlowRes(res.data);
    } else {
      message.error(res.message || '获取资金流水失败');
    }
  }, [canView, page, pageSize, tabKey, bizNo, orderNo, channel, status, dateRange]);

  useEffect(() => {
    void loadData();
  }, [loadData]);

  const handleTabChange = (key: string) => {
    setTabKey(key as any);
    setStatus(undefined);
    setPage(1);
  };

  const handleSearch = () => {
    setPage(1);
    void loadData();
  };

  const handleReset = () => {
    setBizNo('');
    setOrderNo('');
    setChannel(undefined);
    setStatus(undefined);
    setDateRange(null);
    setPage(1);
  };

  if (!canView) {
    return (
      <div style={{ padding: '60px 0' }}>
        <Result
          status="403"
          title="暂无资金流水权限"
          subTitle="您当前所在的角色未被授予资金流水的查看权限，请联系管理员授权。"
        />
      </div>
    );
  }

  const columns: ColumnsType<AnalyticsFinanceFlowRow> = [
    {
      title: '类型',
      dataIndex: 'flow_type',
      key: 'flow_type',
      width: 90,
      render: (_, record) => {
        const isIncome = record.direction === 'income';
        return (
          <Tag color={isIncome ? 'success' : 'error'} bordered={false}>
            {isIncome ? '收入 (支付)' : '支出 (退款)'}
          </Tag>
        );
      },
    },
    {
      title: '流水单号',
      dataIndex: 'biz_no',
      key: 'biz_no',
      width: 180,
      render: (no: string) => <Text copyable={{ text: no }}>{no}</Text>,
    },
    {
      title: '关联订单号',
      dataIndex: 'order_no',
      key: 'order_no',
      width: 180,
      render: (no: string) => (no ? <Text copyable={{ text: no }}>{no}</Text> : '-'),
    },
    {
      title: '渠道',
      dataIndex: 'channel',
      key: 'channel',
      width: 100,
      render: (c: string) => {
        if (c === 'wechat') return '微信支付';
        if (c === 'mock') return '模拟支付';
        if (c === 'alipay') return '支付宝';
        return c || '-';
      },
    },
    {
      title: '金额 (元)',
      dataIndex: 'amount',
      key: 'amount',
      width: 120,
      align: 'right',
      render: (amt: string, record) => {
        const isIncome = record.direction === 'income';
        return (
          <span
            style={{
              fontWeight: 700,
              fontSize: 14,
              color: isIncome ? token.colorSuccess : token.colorError,
            }}
          >
            {isIncome ? `+${amt}` : `-${amt}`}
          </span>
        );
      },
    },
    {
      title: '状态',
      dataIndex: 'status_text',
      key: 'status_text',
      width: 110,
      render: (txt: string, record) => {
        const isSuccess =
          (record.flow_type === 'payment' && record.status === 20) ||
          (record.flow_type === 'refund' && record.status === 30);
        return (
          <Tag color={isSuccess ? 'success' : 'default'} bordered={false}>
            {txt}
          </Tag>
        );
      },
    },
    {
      title: '第三方流水号',
      dataIndex: 'transaction_no',
      key: 'transaction_no',
      width: 200,
      ellipsis: true,
      render: (no?: string) =>
        no ? <Text copyable={{ text: no }}>{no}</Text> : <Text type="secondary">-</Text>,
    },
    {
      title: '发生时间',
      dataIndex: 'occurred_at',
      key: 'occurred_at',
      width: 170,
      render: (t?: string) => (t ? formatTime(t) : '-'),
    },
  ];

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      {/* 顶部标题 */}
      <PageHeader
        title="资金流水"
        description="统一查询商城全部支付收款与售后退款交易明细及三方对账单号"
        extra={
          <Button icon={<RefreshCw size={16} strokeWidth={1.8} />} onClick={loadData} loading={loading}>
            刷新
          </Button>
        }
      />

      {/* 资金汇总卡片 */}
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={8}>
          <StatCard
            title="筛选总收入"
            value={flowRes?.total_income_amount || '0.00'}
            prefix="¥"
            tooltip="符合当前筛选条件的成功收款金额总和"
            icon={<TrendingUp size={18} strokeWidth={1.8} style={{ color: token.colorSuccess }} />}
            loading={loading}
          />
        </Col>
        <Col xs={24} sm={8}>
          <StatCard
            title="筛选总支出 (退款)"
            value={flowRes?.total_expense_amount || '0.00'}
            prefix="¥"
            tooltip="符合当前筛选条件的成功退款金额总和"
            icon={<TrendingDown size={18} strokeWidth={1.8} style={{ color: token.colorError }} />}
            loading={loading}
          />
        </Col>
        <Col xs={24} sm={8}>
          <StatCard
            title="净流入金额"
            value={flowRes?.net_flow_amount || '0.00'}
            prefix="¥"
            tooltip="净流入 = 收入总额 - 退款总额"
            icon={<CircleDollarSign size={18} strokeWidth={1.8} />}
            loading={loading}
          />
        </Col>
      </Row>

      {/* 主工作区 */}
      <ContentCard>
        {/* Tabs */}
        <Tabs
          activeKey={tabKey}
          onChange={handleTabChange}
          items={[
            { key: 'all', label: '全部流水' },
            { key: 'payment', label: '收入流水 (支付)' },
            { key: 'refund', label: '退款流水 (支出)' },
          ]}
          style={{ marginBottom: 12 }}
        />

        {/* 筛选区 */}
        <div style={{ marginBottom: 16 }}>
          <Space wrap size={12} align="center">
            <Input
              placeholder="流水单号"
              value={bizNo}
              onChange={(e) => setBizNo(e.target.value)}
              onPressEnter={handleSearch}
              style={{ width: 180 }}
              allowClear
            />
            <Input
              placeholder="订单号"
              value={orderNo}
              onChange={(e) => setOrderNo(e.target.value)}
              onPressEnter={handleSearch}
              style={{ width: 180 }}
              allowClear
            />
            <Select
              placeholder="渠道"
              value={channel}
              onChange={setChannel}
              style={{ width: 120 }}
              allowClear
              options={[
                { label: '微信支付', value: 'wechat' },
                { label: '模拟支付', value: 'mock' },
              ]}
            />
            {tabKey !== 'all' && (
              <Select
                placeholder="状态"
                value={status}
                onChange={setStatus}
                style={{ width: 120 }}
                allowClear
                options={
                  tabKey === 'refund'
                    ? [
                        { label: '待退款', value: 10 },
                        { label: '退款处理中', value: 20 },
                        { label: '退款成功', value: 30 },
                        { label: '退款失败', value: 40 },
                      ]
                    : [
                        { label: '待支付', value: 10 },
                        { label: '支付成功', value: 20 },
                        { label: '支付失败', value: 30 },
                        { label: '已关闭', value: 40 },
                      ]
                }
              />
            )}
            <DatePicker.RangePicker
              value={dateRange}
              onChange={(dates) => setDateRange(dates)}
              style={{ width: 260 }}
            />
            <Button type="primary" icon={<Search size={16} strokeWidth={1.8} />} onClick={handleSearch}>
              查询
            </Button>
            <Button icon={<RotateCcw size={16} strokeWidth={1.8} />} onClick={handleReset}>重置</Button>
          </Space>
        </div>

        {/* 流水明细表格 */}
        <Table<AnalyticsFinanceFlowRow>
          rowKey={(r) => `${r.flow_type}_${r.id}`}
          loading={loading}
          dataSource={flowRes?.items || []}
          columns={columns}
          pagination={{
            current: page,
            pageSize,
            total: flowRes?.total || 0,
            showSizeChanger: true,
            showTotal: (t) => `共 ${t} 条流水`,
          }}
          onChange={(pag) => {
            if (pag.current) setPage(pag.current);
            if (pag.pageSize) setPageSize(pag.pageSize);
          }}
          size="middle"
          scroll={{ x: 1000 }}
        />
      </ContentCard>
    </div>
  );
}
