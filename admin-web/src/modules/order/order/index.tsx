import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Button,
  DatePicker,
  Form,
  Image,
  Input,
  InputNumber,
  Modal,
  Pagination,
  Select,
  Space,
  Table,
  Tag,
  Tooltip,
  Typography,
  message,
  theme,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { OrderedListOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';

import { api } from '@/framework/http';
import type { OrderListRow } from '@/framework/http';
import { usePermission } from '@/framework/store';
import { formatTimeTwoLines } from '@/shared/utils/format';
import { ContentCard, PageHeader, adminListVars } from '@/shared/admin-ui';

const { RangePicker } = DatePicker;
const { Text } = Typography;

const statusOptions = [
  { label: '全部状态', value: undefined },
  { label: '待付款', value: 10 },
  { label: '待发货', value: 20 },
  { label: '待收货', value: 30 },
  { label: '已完成', value: 40 },
  { label: '已取消', value: 50 },
];

function renderStatusTag(status?: number, text?: string) {
  const label = text || '未知状态';
  switch (status) {
    case 10:
      return <Tag color="warning">{label}</Tag>;
    case 20:
      return <Tag color="processing">{label}</Tag>;
    case 30:
      return <Tag color="blue">{label}</Tag>;
    case 40:
      return <Tag color="success">{label}</Tag>;
    case 50:
      return <Tag color="default">{label}</Tag>;
    default:
      return <Tag>{label}</Tag>;
  }
}

interface FilterFormValues {
  keyword?: string;
  status?: number;
  member_id?: number;
  timeRange?: [dayjs.Dayjs, dayjs.Dayjs];
}

export default function OrderListPage() {
  const { token } = theme.useToken();
  const navigate = useNavigate();
  const hasPermission = usePermission();
  const canShip = hasPermission('order:order:ship');

  const [form] = Form.useForm<FilterFormValues>();
  const [rows, setRows] = useState<OrderListRow[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);

  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  // Active query parameters used for API call
  const [activeQuery, setActiveQuery] = useState<{
    keyword?: string;
    status?: number;
    member_id?: number;
    start_time?: string;
    end_time?: string;
  }>({});

  // Ship modal state
  const [shipModalVisible, setShipModalVisible] = useState(false);
  const [shippingOrder, setShippingOrder] = useState<OrderListRow | null>(null);
  const [shipForm] = Form.useForm<{ express_company: string; express_no: string }>();
  const [shipSubmitting, setShipSubmitting] = useState(false);

  const reqSeqRef = useRef(0);

  const loadOrders = useCallback(async () => {
    const currentSeq = ++reqSeqRef.current;
    setLoading(true);

    const res = await api.api.listOrders({
      page,
      page_size: pageSize,
      keyword: activeQuery.keyword?.trim() || undefined,
      status: activeQuery.status,
      member_id: activeQuery.member_id,
      start_time: activeQuery.start_time || undefined,
      end_time: activeQuery.end_time || undefined,
    });

    if (currentSeq !== reqSeqRef.current) return;
    setLoading(false);

    if (res.code === 1) {
      setRows(res.data ?? []);
      setTotal(res.total ?? 0);
    } else {
      message.error(res.message || '加载订单列表失败');
    }
  }, [page, pageSize, activeQuery]);

  useEffect(() => {
    void loadOrders();
  }, [loadOrders]);

  const handleSearch = () => {
    const values = form.getFieldsValue();
    let startTime: string | undefined;
    let endTime: string | undefined;
    if (values.timeRange && values.timeRange.length === 2) {
      startTime = values.timeRange[0].startOf('day').format('YYYY-MM-DD HH:mm:ss');
      endTime = values.timeRange[1].endOf('day').format('YYYY-MM-DD HH:mm:ss');
    }

    setPage(1);
    setActiveQuery({
      keyword: values.keyword,
      status: values.status,
      member_id: values.member_id,
      start_time: startTime,
      end_time: endTime,
    });
  };

  const handleReset = () => {
    form.resetFields();
    setPage(1);
    setActiveQuery({});
  };

  const openShipModal = (order: OrderListRow) => {
    setShippingOrder(order);
    shipForm.resetFields();
    setShipModalVisible(true);
  };

  const handleShipSubmit = async () => {
    const values = await shipForm.validateFields();
    if (!shippingOrder?.id) return;

    setShipSubmitting(true);
    const res = await api.api.shipOrder({
      id: shippingOrder.id,
      express_company: values.express_company.trim(),
      express_no: values.express_no.trim(),
    });
    setShipSubmitting(false);

    if (res.code === 1) {
      message.success('发货成功');
      setShipModalVisible(false);
      setShippingOrder(null);
      void loadOrders();
    } else {
      message.error(res.message || '发货失败');
    }
  };

  const columns: ColumnsType<OrderListRow> = useMemo(
    () => [
      {
        title: '订单号',
        dataIndex: 'order_no',
        key: 'order_no',
        width: 190,
        render: (no: string, record) => (
          <Space orientation="vertical" size={2}>
            <Text copyable style={{ fontWeight: 600 }}>
              {no}
            </Text>
            <Text type="secondary" style={{ fontSize: 12 }}>
              ID: {record.id}
            </Text>
          </Space>
        ),
      },
      {
        title: '商品信息',
        key: 'items',
        width: 280,
        render: (_, record) => {
          const items = record.items || [];
          if (items.length === 0) return <Text type="secondary">-</Text>;

          const showItems = items.slice(0, 2);
          const remainingCount = items.length - showItems.length;

          return (
            <Space orientation="vertical" size={6} style={{ width: '100%' }}>
              {showItems.map((it) => (
                <div key={it.id} style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                  {it.image_url ? (
                    <Image
                      src={it.image_url}
                      alt={it.product_name}
                      width={36}
                      height={36}
                      style={{ objectFit: 'cover', borderRadius: 4, flexShrink: 0 }}
                      preview={{ mask: false }}
                    />
                  ) : (
                    <div
                      style={{
                        width: 36,
                        height: 36,
                        backgroundColor: '#f5f5f5',
                        borderRadius: 4,
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        fontSize: 10,
                        color: '#999',
                        flexShrink: 0,
                      }}
                    >
                      无图
                    </div>
                  )}
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div
                      style={{
                        fontSize: 13,
                        fontWeight: 500,
                        overflow: 'hidden',
                        textOverflow: 'ellipsis',
                        whiteSpace: 'nowrap',
                      }}
                      title={it.product_name}
                    >
                      {it.product_name}
                    </div>
                    <div style={{ fontSize: 12, color: '#888' }}>
                      ¥{it.unit_price} × {it.quantity}
                    </div>
                  </div>
                </div>
              ))}
              {remainingCount > 0 && (
                <Text type="secondary" style={{ fontSize: 12 }}>
                  共 {record.total_quantity || items.length} 件商品
                </Text>
              )}
            </Space>
          );
        },
      },
      {
        title: '会员',
        key: 'member',
        width: 140,
        render: (_, record) => (
          <Space orientation="vertical" size={2}>
            <span>{record.member_nickname || '未命名'}</span>
            {record.member_mobile && (
              <Text type="secondary" style={{ fontSize: 12 }}>
                {record.member_mobile}
              </Text>
            )}
          </Space>
        ),
      },
      {
        title: '收货人',
        key: 'address',
        width: 160,
        render: (_, record) => {
          const addr = record.address;
          if (!addr) return <Text type="secondary">-</Text>;
          return (
            <Space orientation="vertical" size={2}>
              <span>{addr.name}</span>
              <Text type="secondary" style={{ fontSize: 12 }}>
                {addr.mobile}
              </Text>
            </Space>
          );
        },
      },
      {
        title: '订单金额',
        dataIndex: 'pay_amount',
        key: 'pay_amount',
        width: 130,
        render: (payAmount: string, record) => (
          <Space orientation="vertical" size={2}>
            <span style={{ fontWeight: 600, color: '#f5222d', fontSize: 14 }}>¥{payAmount}</span>
            {record.freight_amount && record.freight_amount !== '0.00' && (
              <Text type="secondary" style={{ fontSize: 11 }}>
                含运费 ¥{record.freight_amount}
              </Text>
            )}
          </Space>
        ),
      },
      {
        title: '状态',
        dataIndex: 'status',
        key: 'status',
        width: 100,
        render: (status?: number, record?: OrderListRow) =>
          renderStatusTag(status, record?.status_text),
      },
      {
        title: '物流信息',
        key: 'express',
        width: 160,
        render: (_, record) => {
          if (!record.express_company && !record.express_no) {
            return <Text type="secondary">-</Text>;
          }
          return (
            <Space orientation="vertical" size={2}>
              <span>{record.express_company || '快递'}</span>
              <Text type="secondary" style={{ fontSize: 12 }}>
                {record.express_no}
              </Text>
            </Space>
          );
        },
      },
      {
        title: '下单时间',
        dataIndex: 'created_at',
        key: 'created_at',
        width: 120,
        render: (time?: string) => formatTimeTwoLines(time),
      },
      {
        title: '操作',
        key: 'action',
        fixed: 'right',
        width: 140,
        render: (_, record) => (
          <Space size={8}>
            <Button
              type="link"
              size="small"
              onClick={() => navigate(`/order/order/${record.id}`)}
              style={{ padding: 0 }}
            >
              查看详情
            </Button>
            {record.status === 20 && canShip && (
              <Button
                type="link"
                size="small"
                onClick={() => openShipModal(record)}
                style={{ padding: 0 }}
              >
                发货
              </Button>
            )}
          </Space>
        ),
      },
    ],
    [canShip, navigate],
  );

  return (
    <div
      className="admin-list-page overflow-hidden rounded-lg"
      style={{
        backgroundColor: token.colorBgContainer,
        ...adminListVars(token),
      }}
    >
      <div className="admin-list-header">
        <PageHeader
          title="订单管理"
          description="查看商城所有客户订单，支持按订单号、收货人、订单状态和下单时间多维筛选与发货处理"
        />
      </div>

      <ContentCard>
        {/* 筛选区 */}
        <Form
          form={form}
          className="admin-list-filters"
          onFinish={handleSearch}
        >
          <Form.Item name="keyword" noStyle>
            <Input
              prefix={<SearchOutlined style={{ color: token.colorTextSecondary }} />}
              placeholder="订单号 / 收货人 / 手机号"
              allowClear
              style={{ width: 220 }}
            />
          </Form.Item>
          <Form.Item name="status" noStyle>
            <Select
              placeholder="全部状态"
              allowClear
              options={statusOptions}
              style={{ width: 130 }}
            />
          </Form.Item>
          <Form.Item name="member_id" noStyle>
            <InputNumber
              placeholder="会员ID"
              min={1}
              style={{ width: 120 }}
            />
          </Form.Item>
          <Form.Item name="timeRange" noStyle>
            <RangePicker style={{ width: 260 }} />
          </Form.Item>
          <Button type="primary" htmlType="submit" icon={<SearchOutlined />}>
            查询
          </Button>
          <Button onClick={handleReset}>
            重置
          </Button>
        </Form>

        {/* 表格工具栏 */}
        <div className="admin-list-toolbar">
          <div className="flex flex-wrap items-center gap-3">
            <h2
              className="m-0 flex items-center gap-2 text-base font-semibold"
              style={{ color: token.colorTextHeading }}
            >
              <OrderedListOutlined style={{ color: token.colorPrimary }} />
              订单列表
            </h2>
            <span className="text-sm" style={{ color: token.colorTextTertiary }}>
              共 {total} 笔订单
            </span>
          </div>
          <Tooltip title="刷新列表">
            <Button
              type="text"
              icon={<ReloadOutlined spin={loading} />}
              onClick={() => void loadOrders()}
              style={{ color: token.colorTextSecondary }}
            >
              刷新
            </Button>
          </Tooltip>
        </div>

        {/* 表格数据 */}
        <div className="px-4">
          <Table<OrderListRow>
            rowKey="id"
            columns={columns}
            dataSource={rows}
            loading={loading}
            scroll={{ x: 1260 }}
            className="admin-list-table"
            pagination={false}
          />
        </div>

        {/* 分页 */}
        <div className="flex flex-wrap items-center justify-between gap-4 px-6 py-4">
          <div
            className="flex items-center gap-2 text-sm"
            style={{ color: token.colorTextSecondary }}
          >
            <span>每页显示</span>
            <Select
              aria-label="每页条数"
              className="w-18"
              size="small"
              value={pageSize}
              options={[10, 20, 50, 100].map((v) => ({
                value: v,
                label: v,
              }))}
              onChange={(size) => {
                setPage(1);
                setPageSize(size);
              }}
            />
            <span>条</span>
          </div>
          <Pagination
            className="admin-list-pagination"
            current={page}
            pageSize={pageSize}
            total={total}
            showSizeChanger={false}
            showQuickJumper
            locale={{ jump_to: '前往', page: '页' }}
            onChange={(p, ps) => {
              setPage(p);
              if (ps && ps !== pageSize) {
                setPageSize(ps);
              }
            }}
          />
        </div>
      </ContentCard>

      {/* 发货弹窗 */}
      <Modal
        title={`订单发货 - ${shippingOrder?.order_no || ''}`}
        open={shipModalVisible}
        onOk={handleShipSubmit}
        confirmLoading={shipSubmitting}
        onCancel={() => {
          setShipModalVisible(false);
          setShippingOrder(null);
        }}
        destroyOnClose
      >
        <Form form={shipForm} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item
            name="express_company"
            label="物流公司"
            rules={[{ required: true, message: '请输入物流公司' }, { max: 64, message: '最多64个字符' }]}
          >
            <Input placeholder="例如：顺丰速运、圆通速递、中通快递" />
          </Form.Item>
          <Form.Item
            name="express_no"
            label="物流单号"
            rules={[{ required: true, message: '请输入物流单号' }, { max: 64, message: '最多64个字符' }]}
          >
            <Input placeholder="请输入快递运单号" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
