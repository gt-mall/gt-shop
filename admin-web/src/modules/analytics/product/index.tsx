import React, { useCallback, useEffect, useState } from 'react';
import {
  Button,
  Card,
  Col,
  Empty,
  Form,
  Input,
  InputNumber,
  Radio,
  Result,
  Row,
  Segmented,
  Select,
  Space,
  Table,
  Tabs,
  Tag,
  Typography,
  message,
  theme,
} from 'antd';
import type { ColumnsType, TablePaginationConfig } from 'antd/es/table';
import {
  Flame,
  PackageSearch,
  RefreshCw,
  RotateCcw,
  Search,
  TrendingUp,
} from 'lucide-react';

import {
  api,
  type AnalyticsProductAnalysisRow,
  type AnalyticsProductRankingItem,
  type AnalyticsProductRankingsResponse,
} from '@/framework/http';
import { usePermission } from '@/framework/store';
import { ContentCard, FilterCard, PageHeader, TableCard } from '@/shared/admin-ui';

import { TimeRangePicker } from '../components/time-range-picker';

const { Text } = Typography;

export default function AnalyticsProductPage() {
  const { token } = theme.useToken();
  const hasPermission = usePermission();
  const canView = hasPermission('analytics:product:view');

  // Filter states
  const [timeRange, setTimeRange] = useState('last_7_days');
  const [startTime, setStartTime] = useState<string>();
  const [endTime, setEndTime] = useState<string>();
  const [dimension, setDimension] = useState<'product' | 'sku'>('product');
  const [keyword, setKeyword] = useState<string>('');
  const [productId, setProductId] = useState<number>();
  const [skuCode, setSkuCode] = useState<string>('');

  // Table pagination & sorting
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [sortBy, setSortBy] = useState('sales_amount');
  const [sortOrder, setSortOrder] = useState<'desc' | 'asc'>('desc');

  // Data states
  const [tableLoading, setTableLoading] = useState(false);
  const [tableData, setTableData] = useState<AnalyticsProductAnalysisRow[]>([]);
  const [total, setTotal] = useState(0);

  const [rankingsLoading, setRankingsLoading] = useState(false);
  const [rankings, setRankings] = useState<AnalyticsProductRankingsResponse>();
  const [rankingTab, setRankingTab] = useState<'amount' | 'quantity' | 'refund'>('amount');

  // Fetch rankings
  const loadRankings = useCallback(async () => {
    if (!canView) return;
    setRankingsLoading(true);
    const res = await api.api.getAnalyticsProductRankings({
      time_range: timeRange,
      start_time: startTime,
      end_time: endTime,
    });
    setRankingsLoading(false);
    if (res.code === 1 && res.data) {
      setRankings(res.data);
    }
  }, [canView, timeRange, startTime, endTime]);

  // Fetch list
  const loadList = useCallback(async () => {
    if (!canView) return;
    setTableLoading(true);

    const res = await api.api.listAnalyticsProducts({
      dimension,
      time_range: timeRange,
      start_time: startTime,
      end_time: endTime,
      keyword: keyword || undefined,
      product_id: productId || undefined,
      sku_code: skuCode || undefined,
      page,
      page_size: pageSize,
      sort_by: sortBy,
      sort_order: sortOrder,
    });

    setTableLoading(false);
    if (res.code === 1) {
      setTableData(res.data || []);
      setTotal(res.total || 0);
    } else {
      message.error(res.message || '加载商品分析列表失败');
    }
  }, [canView, dimension, timeRange, startTime, endTime, keyword, productId, skuCode, page, pageSize, sortBy, sortOrder]);

  useEffect(() => {
    void loadRankings();
  }, [loadRankings]);

  useEffect(() => {
    void loadList();
  }, [loadList]);

  const handleTimeRangeChange = (r: string, s?: string, e?: string) => {
    setTimeRange(r);
    setStartTime(s);
    setEndTime(e);
    setPage(1);
  };

  const handleSearch = () => {
    setPage(1);
    void loadList();
  };

  const handleReset = () => {
    setKeyword('');
    setProductId(undefined);
    setSkuCode('');
    setPage(1);
  };

  if (!canView) {
    return (
      <div style={{ padding: '60px 0' }}>
        <Result
          status="403"
          title="暂无商品分析权限"
          subTitle="您当前所在的角色未被授予商品分析的查看权限，请联系管理员授权。"
        />
      </div>
    );
  }

  // Ranking columns
  const rankingColumns: ColumnsType<AnalyticsProductRankingItem> = [
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
              style={{ width: 32, height: 32, borderRadius: 4, objectFit: 'cover' }}
            />
          ) : (
            <div
              style={{
                width: 32,
                height: 32,
                borderRadius: 4,
                backgroundColor: token.colorFillAlter,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                color: token.colorTextQuaternary,
              }}
            >
              <PackageSearch size={16} strokeWidth={1.8} />
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
      dataIndex: 'sales_quantity',
      key: 'sales_quantity',
      width: 90,
      align: 'right',
      render: (q: number) => <span style={{ fontWeight: 600 }}>{q}</span>,
    },
    {
      title: '销售额',
      dataIndex: 'sales_amount',
      key: 'sales_amount',
      width: 110,
      align: 'right',
      render: (amt: string) => <span style={{ fontWeight: 600, color: token.colorPrimary }}>¥{amt}</span>,
    },
    {
      title: '退款件数',
      dataIndex: 'refund_quantity',
      key: 'refund_quantity',
      width: 90,
      align: 'right',
    },
    {
      title: '退款金额',
      dataIndex: 'refund_amount',
      key: 'refund_amount',
      width: 100,
      align: 'right',
      render: (amt: string) => <span style={{ color: token.colorError }}>¥{amt}</span>,
    },
    {
      title: '退款率',
      dataIndex: 'refund_rate',
      key: 'refund_rate',
      width: 90,
      align: 'right',
      render: (r: string) => <Tag bordered={false}>{r}</Tag>,
    },
  ];

  // Table columns
  const tableColumns: ColumnsType<AnalyticsProductAnalysisRow> = [
    {
      title: '商品信息',
      key: 'product_info',
      width: 280,
      render: (_, record) => (
        <Space size={10} align="start">
          {record.image_url ? (
            <img
              src={record.image_url}
              alt=""
              style={{ width: 44, height: 44, borderRadius: 4, objectFit: 'cover', flexShrink: 0 }}
            />
          ) : (
            <div
              style={{
                width: 44,
                height: 44,
                borderRadius: 4,
                backgroundColor: token.colorFillAlter,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                color: token.colorTextQuaternary,
                flexShrink: 0,
              }}
            >
              <PackageSearch size={16} strokeWidth={1.8} />
            </div>
          )}
          <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
            <Text strong style={{ fontSize: 13 }} ellipsis={{ tooltip: record.product_name }}>
              {record.product_name}
            </Text>
            <Space size={6} wrap>
              <Text type="secondary" style={{ fontSize: 12 }}>
                ID: {record.product_id}
              </Text>
              {record.sku_code && (
                <Tag bordered={false} style={{ margin: 0, fontSize: 11 }}>
                  编码: {record.sku_code}
                </Tag>
              )}
              {record.sku_text && (
                <Tag bordered={false} color="purple" style={{ margin: 0, fontSize: 11 }}>
                  {record.sku_text}
                </Tag>
              )}
            </Space>
          </div>
        </Space>
      ),
    },
    {
      title: '销量',
      dataIndex: 'sales_quantity',
      key: 'sales_quantity',
      width: 100,
      align: 'right',
      sorter: true,
      render: (q: number) => <span style={{ fontWeight: 600 }}>{q}</span>,
    },
    {
      title: '销售额',
      dataIndex: 'sales_amount',
      key: 'sales_amount',
      width: 120,
      align: 'right',
      sorter: true,
      render: (amt: string) => (
        <span style={{ fontWeight: 700, color: token.colorPrimary }}>¥{amt}</span>
      ),
    },
    {
      title: '支付订单数',
      dataIndex: 'paid_order_count',
      key: 'paid_order_count',
      width: 110,
      align: 'right',
      sorter: true,
    },
    {
      title: '购买人数',
      dataIndex: 'buyer_count',
      key: 'buyer_count',
      width: 100,
      align: 'right',
    },
    {
      title: '退款数量',
      dataIndex: 'refund_quantity',
      key: 'refund_quantity',
      width: 100,
      align: 'right',
      sorter: true,
    },
    {
      title: '退款金额',
      dataIndex: 'refund_amount',
      key: 'refund_amount',
      width: 110,
      align: 'right',
      sorter: true,
      render: (amt: string) => (
        <span style={{ fontWeight: 600, color: token.colorError }}>¥{amt}</span>
      ),
    },
    {
      title: '退款率',
      dataIndex: 'refund_rate',
      key: 'refund_rate',
      width: 100,
      align: 'right',
      render: (rate: string) => <Tag bordered={false}>{rate}</Tag>,
    },
  ];

  const handleTableChange = (
    pagination: TablePaginationConfig,
    _: any,
    sorter: any,
  ) => {
    if (pagination.current) setPage(pagination.current);
    if (pagination.pageSize) setPageSize(pagination.pageSize);

    if (sorter && sorter.field) {
      setSortBy(String(sorter.field));
      setSortOrder(sorter.order === 'ascend' ? 'asc' : 'desc');
    }
  };

  const currentRankData =
    rankingTab === 'amount'
      ? rankings?.sales_amount_rankings || []
      : rankingTab === 'quantity'
      ? rankings?.sales_quantity_rankings || []
      : rankings?.refund_rankings || [];

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      {/* 顶部标题 */}
      <PageHeader
        title="商品分析"
        description="多维度查看商品销量、销售额、退款率及 SKU 表现"
        extra={
          <Space wrap size={12} align="center">
            <TimeRangePicker
              value={timeRange}
              startTime={startTime}
              endTime={endTime}
              onChange={handleTimeRangeChange}
            />
            <Button icon={<RefreshCw size={16} strokeWidth={1.8} />} onClick={() => { void loadRankings(); void loadList(); }}>
              刷新
            </Button>
          </Space>
        }
      />

      {/* TOP 排行榜区块 */}
      <Card
        size="small"
        style={{ borderRadius: 8 }}
        title={
          <Space size={8} align="center">
            <Flame size={16} strokeWidth={1.8} style={{ color: token.colorError }} />
            <span>商品排行榜 TOP 10</span>
          </Space>
        }
        extra={
          <Segmented
            value={rankingTab}
            onChange={(v) => setRankingTab(v as any)}
            options={[
              { label: '销售额排行', value: 'amount', icon: <TrendingUp size={14} strokeWidth={1.8} /> },
              { label: '销量排行', value: 'quantity', icon: <PackageSearch size={14} strokeWidth={1.8} /> },
              { label: '退款排行', value: 'refund', icon: <RotateCcw size={14} strokeWidth={1.8} /> },
            ]}
          />
        }
        styles={{ body: { padding: 0 } }}
      >
        <Table<AnalyticsProductRankingItem>
          rowKey="product_id"
          loading={rankingsLoading}
          dataSource={currentRankData}
          columns={rankingColumns}
          pagination={false}
          size="small"
          locale={{ emptyText: <Empty description="暂无排行数据" /> }}
        />
      </Card>

      {/* 筛选与明细表格 */}
      <ContentCard>
        {/* 筛选区 */}
        <div style={{ marginBottom: 16 }}>
          <Space wrap size={16} align="center">
            <Segmented
              value={dimension}
              onChange={(val) => {
                setDimension(val as any);
                setPage(1);
              }}
              options={[
                { label: '商品维度', value: 'product' },
                { label: 'SKU维度', value: 'sku' },
              ]}
            />
            <Input
              placeholder="商品名称 / SKU编码"
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
              onPressEnter={handleSearch}
              style={{ width: 180 }}
              allowClear
            />
            <InputNumber
              placeholder="商品 ID"
              value={productId}
              onChange={(v) => setProductId(v ?? undefined)}
              style={{ width: 110 }}
            />
            {dimension === 'sku' && (
              <Input
                placeholder="SKU 编码"
                value={skuCode}
                onChange={(e) => setSkuCode(e.target.value)}
                onPressEnter={handleSearch}
                style={{ width: 140 }}
                allowClear
              />
            )}
            <Button type="primary" icon={<Search size={16} strokeWidth={1.8} />} onClick={handleSearch}>
              查询
            </Button>
            <Button icon={<RotateCcw size={16} strokeWidth={1.8} />} onClick={handleReset}>重置</Button>
          </Space>
        </div>

        {/* 表格 */}
        <Table<AnalyticsProductAnalysisRow>
          rowKey={(r) => (dimension === 'sku' ? `${r.product_id}_${r.sku_id}` : `${r.product_id}`)}
          loading={tableLoading}
          dataSource={tableData}
          columns={tableColumns}
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: true,
            showTotal: (t) => `共 ${t} 条记录`,
          }}
          onChange={handleTableChange}
          size="middle"
        />
      </ContentCard>
    </div>
  );
}
