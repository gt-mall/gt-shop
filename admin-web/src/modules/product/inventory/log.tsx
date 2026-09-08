import React, { useCallback, useEffect, useRef, useState } from 'react';
import {
  Alert,
  Button,
  Card,
  Col,
  DatePicker,
  Descriptions,
  Form,
  Input,
  InputNumber,
  message,
  Row,
  Select,
  Space,
  Table,
  Tag,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  ArrowLeftOutlined,
  HistoryOutlined,
  ReloadOutlined,
  SearchOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import { useNavigate, useSearchParams } from 'react-router-dom';

import { api } from '@/framework/http';
import type {
  ProductInventoryLogListRequest,
  ProductInventoryLogRow,
} from '@/framework/http/generated_api';
import { usePermission } from '@/framework/store';
import { formatTime } from '@/shared/utils/format';

const { RangePicker } = DatePicker;

interface SearchFormValues {
  sku_id?: number;
  product_id?: number;
  change_type?: string;
  operator_id?: number;
  request_id?: string;
  date_range?: [Dayjs, Dayjs];
}

const changeTypeLabels: Record<string, { label: string; color: string }> = {
  manual_in: { label: '人工入库', color: 'blue' },
  manual_out: { label: '人工出库', color: 'orange' },
  stocktake: { label: '盘点校正', color: 'purple' },
  warning_set: { label: '设置预警', color: 'cyan' },
};

export default function ProductInventoryLogPage() {
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const hasPermission = usePermission();
  const canLog = hasPermission('product:inventory:log');
  const canList = hasPermission('product:inventory:list');

  const [form] = Form.useForm<SearchFormValues>();

  const [rows, setRows] = useState<ProductInventoryLogRow[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);

  // Pagination state
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  // Deep linked ID filters from URL
  const paramProductID = searchParams.get('product_id');
  const paramSKUID = searchParams.get('sku_id');

  const initialProductID =
    paramProductID && /^\d+$/.test(paramProductID) ? Number(paramProductID) : undefined;
  const initialSKUID =
    paramSKUID && /^\d+$/.test(paramSKUID) ? Number(paramSKUID) : undefined;

  // Active query parameters
  const [activeQuery, setActiveQuery] = useState<SearchFormValues>({
    product_id: initialProductID,
    sku_id: initialSKUID,
    change_type: undefined,
    operator_id: undefined,
    request_id: undefined,
    date_range: undefined,
  });

  const reqSeqRef = useRef(0);

  // Load logs
  const loadLogs = useCallback(async () => {
    if (!canLog) return;
    const currentSeq = ++reqSeqRef.current;
    setLoading(true);

    let createdFrom: string | undefined;
    let createdTo: string | undefined;

    if (activeQuery.date_range && activeQuery.date_range[0] && activeQuery.date_range[1]) {
      // Format as RFC3339
      createdFrom = activeQuery.date_range[0].startOf('day').toISOString();
      // Use start of next day to fully include the selected end day
      createdTo = activeQuery.date_range[1].add(1, 'day').startOf('day').toISOString();
    }

    const req: ProductInventoryLogListRequest = {
      page,
      page_size: pageSize,
      product_id: activeQuery.product_id,
      sku_id: activeQuery.sku_id,
      change_type: activeQuery.change_type,
      operator_id: activeQuery.operator_id,
      request_id: activeQuery.request_id?.trim() || undefined,
      created_from: createdFrom,
      created_to: createdTo,
    };

    const res = await api.api.listProductInventoryLogs(req);

    if (currentSeq !== reqSeqRef.current) return;
    setLoading(false);

    if (res.code === 1) {
      setRows((res.data as ProductInventoryLogRow[]) ?? []);
      setTotal(res.total ?? 0);
    } else {
      message.error(res.message || '加载库存流水失败');
    }
  }, [canLog, page, pageSize, activeQuery]);

  useEffect(() => {
    void loadLogs();
  }, [loadLogs]);

  // Handle Search submit
  const handleSearch = (values: SearchFormValues) => {
    setPage(1);
    setActiveQuery(values);

    // Sync URL params
    const newParams = new URLSearchParams();
    if (values.product_id) newParams.set('product_id', String(values.product_id));
    if (values.sku_id) newParams.set('sku_id', String(values.sku_id));
    setSearchParams(newParams, { replace: true });
  };

  // Handle Reset filters
  const handleReset = () => {
    form.resetFields();
    setSearchParams({}, { replace: true });
    setPage(1);
    setActiveQuery({
      product_id: undefined,
      sku_id: undefined,
      change_type: undefined,
      operator_id: undefined,
      request_id: undefined,
      date_range: undefined,
    });
  };

  // Columns definition
  const columns: ColumnsType<ProductInventoryLogRow> = [
    {
      title: '记录时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 170,
      render: (t: string) => formatTime(t),
    },
    {
      title: '商品 / SKU 历史快照',
      key: 'snapshot',
      width: 250,
      render: (_, record) => {
        const snap = record.sku_snapshot;
        const specs = snap?.specs ?? [];
        return (
          <Space orientation="vertical" size={2} style={{ width: '100%' }}>
            <Typography.Text strong ellipsis={{ tooltip: snap?.product_name }}>
              {snap?.product_name || '—'}
            </Typography.Text>
            <Space size={4} wrap>
              {specs.length === 0 ? (
                <Tag color="default" style={{ margin: 0 }}>默认规格</Tag>
              ) : (
                specs.map((s) => (
                  <Tag key={`${s.spec_id}-${s.value_id}`} color="blue" style={{ margin: 0 }}>
                    {s.spec_name}: {s.value_name}
                  </Tag>
                ))
              )}
            </Space>
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              编码: {snap?.sku_code || '—'} | 单位: {snap?.unit || '件'}
            </Typography.Text>
            <Typography.Text type="secondary" style={{ fontSize: 11 }}>
              SKU ID: {record.sku_id} | 商品ID: {record.product_id}
            </Typography.Text>
          </Space>
        );
      },
    },
    {
      title: '操作类型',
      dataIndex: 'change_type',
      key: 'change_type',
      width: 110,
      align: 'center',
      render: (type: string) => {
        const conf = changeTypeLabels[type] ?? { label: type, color: 'default' };
        return <Tag color={conf.color}>{conf.label}</Tag>;
      },
    },
    {
      title: '总库存变化',
      dataIndex: 'stock_delta',
      key: 'stock_delta',
      width: 110,
      align: 'right',
      render: (delta: number, record) => {
        if (record.change_type === 'warning_set') {
          return <Typography.Text type="secondary">—</Typography.Text>;
        }
        if (delta > 0) {
          return (
            <Typography.Text strong style={{ color: '#52c41a' }}>
              +{delta}
            </Typography.Text>
          );
        }
        if (delta < 0) {
          return (
            <Typography.Text strong style={{ color: '#ff4d4f' }}>
              {delta}
            </Typography.Text>
          );
        }
        return <Typography.Text type="secondary">0</Typography.Text>;
      },
    },
    {
      title: '总库存 (前 → 后)',
      key: 'stock_range',
      width: 140,
      align: 'right',
      render: (_, record) => (
        <Space size={4}>
          <Typography.Text type="secondary">{record.stock_before}</Typography.Text>
          <span>→</span>
          <Typography.Text strong>{record.stock_after}</Typography.Text>
        </Space>
      ),
    },
    {
      title: '锁定库存 (前 → 后)',
      key: 'locked_range',
      width: 140,
      align: 'right',
      render: (_, record) => {
        const lockedBefore = record.locked_before ?? 0;
        const lockedAfter = record.locked_after ?? 0;
        return (
          <Space size={4}>
            <Typography.Text type="secondary">{lockedBefore}</Typography.Text>
            <span>→</span>
            <Typography.Text type={lockedAfter > 0 ? 'warning' : 'secondary'}>
              {lockedAfter}
            </Typography.Text>
          </Space>
        );
      },
    },
    {
      title: '可用库存 (前 → 后)',
      key: 'available_range',
      width: 140,
      align: 'right',
      render: (_, record) => {
        const availBefore = record.available_before ?? 0;
        const availAfter = record.available_after ?? 0;
        return (
          <Space size={4}>
            <Typography.Text type="secondary">{availBefore}</Typography.Text>
            <span>→</span>
            <Typography.Text
              strong
              style={{ color: availAfter === 0 ? '#ff4d4f' : '#52c41a' }}
            >
              {availAfter}
            </Typography.Text>
          </Space>
        );
      },
    },
    {
      title: '预警阈值',
      key: 'warning_range',
      width: 130,
      align: 'center',
      render: (_, record) => {
        if (record.change_type === 'warning_set') {
          const beforeStr = record.warning_before === 0 ? '未开启' : `${record.warning_before}`;
          const afterStr = record.warning_after === 0 ? '已关闭' : `${record.warning_after}`;
          return (
            <Tag color="cyan">
              {beforeStr} → {afterStr}
            </Tag>
          );
        }
        if (record.warning_after === 0) {
          return <Tag color="default">未开启</Tag>;
        }
        return <Typography.Text type="secondary">{record.warning_after}</Typography.Text>;
      },
    },
    {
      title: '操作者',
      key: 'operator',
      width: 130,
      render: (_, record) => (
        <Space orientation="vertical" size={2}>
          <Typography.Text strong>{record.operator_name}</Typography.Text>
          <Typography.Text type="secondary" style={{ fontSize: 11 }}>
            ID: {record.operator_id}
          </Typography.Text>
        </Space>
      ),
    },
    {
      title: '操作原因',
      dataIndex: 'remark',
      key: 'remark',
      width: 180,
      ellipsis: true,
      render: (remark: string) => (
        <Typography.Paragraph
          ellipsis={{ rows: 2, tooltip: remark }}
          style={{ margin: 0, fontSize: 12 }}
        >
          {remark}
        </Typography.Paragraph>
      ),
    },
  ];

  return (
    <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
      {/* Top Header Card */}
      <Card size="small">
        <Space style={{ width: '100%', justifyContent: 'space-between' }}>
          <Space align="center">
            {canList && (
              <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/product/inventory')}>
                返回库存列表
              </Button>
            )}
            <Typography.Title level={4} style={{ margin: 0 }}>
              库存变更流水
            </Typography.Title>
          </Space>

          <Button
            icon={<ReloadOutlined />}
            onClick={() => void loadLogs()}
            loading={loading}
          >
            刷新
          </Button>
        </Space>
      </Card>

      {/* Filter Card */}
      <Card>
        <Form<SearchFormValues>
          form={form}
          layout="horizontal"
          initialValues={{
            product_id: initialProductID,
            sku_id: initialSKUID,
          }}
          onFinish={handleSearch}
        >
          <Row gutter={[16, 12]}>
            <Col xs={24} sm={12} md={4}>
              <Form.Item name="sku_id" label="SKU ID" style={{ marginBottom: 0 }}>
                <InputNumber min={1} precision={0} placeholder="SKU ID" style={{ width: '100%' }} />
              </Form.Item>
            </Col>

            <Col xs={24} sm={12} md={4}>
              <Form.Item name="product_id" label="商品 ID" style={{ marginBottom: 0 }}>
                <InputNumber min={1} precision={0} placeholder="商品 ID" style={{ width: '100%' }} />
              </Form.Item>
            </Col>

            <Col xs={24} sm={12} md={4}>
              <Form.Item name="change_type" label="操作类型" style={{ marginBottom: 0 }}>
                <Select
                  allowClear
                  placeholder="全部类型"
                  options={[
                    { label: '人工入库', value: 'manual_in' },
                    { label: '人工出库', value: 'manual_out' },
                    { label: '盘点校正', value: 'stocktake' },
                    { label: '设置预警', value: 'warning_set' },
                  ]}
                />
              </Form.Item>
            </Col>

            <Col xs={24} sm={12} md={7}>
              <Form.Item name="date_range" label="时间范围" style={{ marginBottom: 0 }}>
                <RangePicker style={{ width: '100%' }} />
              </Form.Item>
            </Col>

            <Col xs={24} sm={12} md={5} style={{ textAlign: 'right' }}>
              <Space>
                <Button onClick={handleReset}>重置</Button>
                <Button type="primary" htmlType="submit" icon={<SearchOutlined />}>
                  查询
                </Button>
              </Space>
            </Col>
          </Row>

          <Row gutter={[16, 12]} style={{ marginTop: 12 }}>
            <Col xs={24} sm={12} md={6}>
              <Form.Item name="request_id" label="操作编号" style={{ marginBottom: 0 }}>
                <Input placeholder="输入 Request UUID 检索" allowClear />
              </Form.Item>
            </Col>
            <Col xs={24} sm={12} md={6}>
              <Form.Item name="operator_id" label="操作者ID" style={{ marginBottom: 0 }}>
                <InputNumber min={1} precision={0} placeholder="操作者用户ID" style={{ width: '100%' }} />
              </Form.Item>
            </Col>
          </Row>
        </Form>
      </Card>

      {/* Table Card */}
      <Card
        title={
          <Space>
            <HistoryOutlined style={{ color: '#4338CA' }} />
            <span>流水明细记录</span>
            <Typography.Text type="secondary" style={{ fontSize: 13 }}>
              （共 {total} 条流水）
            </Typography.Text>
          </Space>
        }
      >
        <Table<ProductInventoryLogRow>
          rowKey="id"
          columns={columns}
          dataSource={rows}
          loading={loading}
          scroll={{ x: 1300 }}
          expandable={{
            expandedRowRender: (record) => (
              <Descriptions size="small" bordered column={2}>
                <Descriptions.Item label="流水ID">{record.id}</Descriptions.Item>
                <Descriptions.Item label="操作编号 (Request ID)">
                  <Typography.Text copyable>{record.request_id}</Typography.Text>
                </Descriptions.Item>
                <Descriptions.Item label="版本变更">
                  v{record.version_before} → v{record.version_after}
                </Descriptions.Item>
                <Descriptions.Item label="库存记录ID">{record.inventory_id}</Descriptions.Item>
                <Descriptions.Item label="完整原因说明" span={2}>
                  {record.remark}
                </Descriptions.Item>
              </Descriptions>
            ),
          }}
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: true,
            pageSizeOptions: ['20', '50', '100'],
            showQuickJumper: true,
            showTotal: (t) => `共 ${t} 条`,
            onChange: (p, ps) => {
              setPage(p);
              setPageSize(ps);
            },
          }}
        />
      </Card>
    </Space>
  );
}
