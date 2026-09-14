import { useCallback, useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Button,
  Card,
  DatePicker,
  Form,
  Input,
  Modal,
  Popconfirm,
  Select,
  Space,
  Table,
  Tag,
  Typography,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  EyeOutlined,
  RedoOutlined,
  ReloadOutlined,
  SearchOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';

import {
  api,
  type AdminAfterSaleListRequest,
  type AdminAfterSaleListRow,
} from '@/framework/http';
import { usePermission } from '@/framework/store';
import { formatTime } from '@/shared/utils/format';

const { RangePicker } = DatePicker;
const { Text } = Typography;

const statusOptions = [
  { label: '全部状态', value: undefined },
  { label: '待审核', value: 10 },
  { label: '待买家退货', value: 20 },
  { label: '待商家收货', value: 30 },
  { label: '退款中', value: 40 },
  { label: '已完成', value: 50 },
  { label: '已拒绝', value: 60 },
  { label: '已撤销', value: 70 },
  { label: '退款失败', value: 80 },
];

const typeOptions = [
  { label: '全部类型', value: undefined },
  { label: '仅退款', value: 1 },
  { label: '退货退款', value: 2 },
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
      return <Tag color="cyan">{label}</Tag>;
    case 50:
      return <Tag color="success">{label}</Tag>;
    case 60:
      return <Tag color="error">{label}</Tag>;
    case 70:
      return <Tag color="default">{label}</Tag>;
    case 80:
      return <Tag color="magenta">{label}</Tag>;
    default:
      return <Tag>{label}</Tag>;
  }
}

interface FilterFormValues {
  keyword?: string;
  after_sale_no?: string;
  order_no?: string;
  type?: number;
  status?: number;
  timeRange?: [dayjs.Dayjs, dayjs.Dayjs];
}

export default function AfterSaleListPage() {
  const navigate = useNavigate();
  const hasPermission = usePermission();

  const canApprove = hasPermission('order:after_sale:approve');
  const canReject = hasPermission('order:after_sale:reject');
  const canRetryRefund = hasPermission('order:after_sale:retry_refund');

  const [form] = Form.useForm<FilterFormValues>();
  const [rows, setRows] = useState<AdminAfterSaleListRow[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);

  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  const [activeQuery, setActiveQuery] = useState<AdminAfterSaleListRequest>({});

  // Reject modal state
  const [rejectModalVisible, setRejectModalVisible] = useState(false);
  const [rejectTargetId, setRejectTargetId] = useState<number>();
  const [rejectForm] = Form.useForm<{ reject_reason: string }>();
  const [rejectSubmitting, setRejectSubmitting] = useState(false);

  // Approve modal state (for return type with address)
  const [approveModalVisible, setApproveModalVisible] = useState(false);
  const [approveTarget, setApproveTarget] = useState<AdminAfterSaleListRow>();
  const [approveForm] = Form.useForm<{
    return_receiver?: string;
    return_phone?: string;
    return_address?: string;
  }>();
  const [approveSubmitting, setApproveSubmitting] = useState(false);

  const reqSeqRef = useRef(0);

  const loadList = useCallback(async () => {
    const currentSeq = ++reqSeqRef.current;
    setLoading(true);

    const res = await api.api.listAfterSales({
      page,
      page_size: pageSize,
      ...activeQuery,
    });

    if (currentSeq !== reqSeqRef.current) return;
    setLoading(false);

    if (res.code === 1) {
      setRows(res.data ?? []);
      setTotal(res.total ?? 0);
    } else {
      message.error(res.message || '获取售后列表失败');
    }
  }, [page, pageSize, activeQuery]);

  useEffect(() => {
    void loadList();
  }, [loadList]);

  const handleSearch = () => {
    const values = form.getFieldsValue();
    const [start, end] = values.timeRange || [];

    setActiveQuery({
      keyword: values.keyword?.trim() || undefined,
      after_sale_no: values.after_sale_no?.trim() || undefined,
      order_no: values.order_no?.trim() || undefined,
      type: values.type,
      status: values.status,
      start_time: start ? start.startOf('day').toISOString() : undefined,
      end_time: end ? end.endOf('day').toISOString() : undefined,
    });
    setPage(1);
  };

  const handleReset = () => {
    form.resetFields();
    setActiveQuery({});
    setPage(1);
  };

  const openApprove = (record: AdminAfterSaleListRow) => {
    setApproveTarget(record);
    if (record.type === 2) {
      approveForm.resetFields();
      setApproveModalVisible(true);
    } else {
      Modal.confirm({
        title: '审核同意售后',
        content: `确认同意售后单【${record.after_sale_no}】？将直接发起退款。`,
        okText: '确认同意',
        cancelText: '取消',
        onOk: async () => {
          const res = await api.api.approveAfterSale({ id: record.id! });
          if (res.code === 1) {
            message.success('审核同意成功');
            void loadList();
          } else {
            message.error(res.message || '操作失败');
          }
        },
      });
    }
  };

  const handleApproveSubmit = async () => {
    if (!approveTarget?.id) return;
    const values = await approveForm.validateFields();
    setApproveSubmitting(true);
    const res = await api.api.approveAfterSale({
      id: approveTarget.id,
      return_receiver: values.return_receiver?.trim(),
      return_phone: values.return_phone?.trim(),
      return_address: values.return_address?.trim(),
    });
    setApproveSubmitting(false);

    if (res.code === 1) {
      message.success('审核通过并已发送退货地址');
      setApproveModalVisible(false);
      void loadList();
    } else {
      message.error(res.message || '操作失败');
    }
  };

  const openReject = (record: AdminAfterSaleListRow) => {
    setRejectTargetId(record.id);
    rejectForm.resetFields();
    setRejectModalVisible(true);
  };

  const handleRejectSubmit = async () => {
    if (!rejectTargetId) return;
    const values = await rejectForm.validateFields();
    setRejectSubmitting(true);
    const res = await api.api.rejectAfterSale({
      id: rejectTargetId,
      reject_reason: values.reject_reason.trim(),
    });
    setRejectSubmitting(false);

    if (res.code === 1) {
      message.success('已拒绝该售后申请');
      setRejectModalVisible(false);
      void loadList();
    } else {
      message.error(res.message || '操作失败');
    }
  };

  const handleRetryRefund = async (id?: number) => {
    if (!id) return;
    const res = await api.api.retryAfterSaleRefund({ id });
    if (res.code === 1) {
      message.success('退款重试已提交');
      void loadList();
    } else {
      message.error(res.message || '退款重试失败');
    }
  };

  const columns: ColumnsType<AdminAfterSaleListRow> = [
    {
      title: '售后单号',
      dataIndex: 'after_sale_no',
      key: 'after_sale_no',
      width: 190,
      render: (text: string, record: AdminAfterSaleListRow) => (
        <Space direction="vertical" size={2}>
          <Text
            copyable
            strong
            className="cursor-pointer text-blue-600"
            onClick={() => navigate(`/order/aftersale/${record.id}`)}
          >
            {text}
          </Text>
          <Text type="secondary" style={{ fontSize: 12 }}>
            关联订单: {record.order_no}
          </Text>
        </Space>
      ),
    },
    {
      title: '售后类型',
      dataIndex: 'type_text',
      key: 'type_text',
      width: 100,
      render: (text: string, record: AdminAfterSaleListRow) => (
        <Tag color={record.type === 2 ? 'orange' : 'cyan'}>{text || (record.type === 2 ? '退货退款' : '仅退款')}</Tag>
      ),
    },
    {
      title: '售后状态',
      dataIndex: 'status',
      key: 'status',
      width: 110,
      render: (status: number, record: AdminAfterSaleListRow) =>
        renderStatusTag(status, record.status_text),
    },
    {
      title: '退款金额',
      dataIndex: 'refund_amount',
      key: 'refund_amount',
      width: 110,
      render: (amount: string) => (
        <span className="font-semibold text-rose-600">¥{amount || '0.00'}</span>
      ),
    },
    {
      title: '申请原因',
      dataIndex: 'reason',
      key: 'reason',
      ellipsis: true,
      render: (reason: string) => reason || '-',
    },
    {
      title: '申请用户',
      key: 'member',
      width: 140,
      render: (_: unknown, record: AdminAfterSaleListRow) => (
        <Space direction="vertical" size={0}>
          <Text>{record.member_nickname || '-'}</Text>
          <Text type="secondary" style={{ fontSize: 12 }}>
            {record.member_mobile || `ID: ${record.member_id}`}
          </Text>
        </Space>
      ),
    },
    {
      title: '申请时间',
      dataIndex: 'applied_at',
      key: 'applied_at',
      width: 165,
      render: (val: string, record: AdminAfterSaleListRow) =>
        formatTime(val || record.created_at),
    },
    {
      title: '操作',
      key: 'action',
      width: 180,
      fixed: 'right',
      render: (_: unknown, record: AdminAfterSaleListRow) => (
        <Space size="small" wrap>
          <Button
            type="link"
            size="small"
            icon={<EyeOutlined />}
            onClick={() => navigate(`/order/aftersale/${record.id}`)}
          >
            详情
          </Button>

          {record.status === 10 && canApprove && (
            <Button
              type="link"
              size="small"
              icon={<CheckCircleOutlined />}
              onClick={() => openApprove(record)}
            >
              同意
            </Button>
          )}

          {record.status === 10 && canReject && (
            <Button
              type="link"
              size="small"
              danger
              icon={<CloseCircleOutlined />}
              onClick={() => openReject(record)}
            >
              拒绝
            </Button>
          )}

          {record.status === 80 && canRetryRefund && (
            <Popconfirm
              title="确定重新发起退款？"
              description="将调用支付机构重新提交退款申请"
              onConfirm={() => handleRetryRefund(record.id)}
              okText="确定"
              cancelText="取消"
            >
              <Button type="link" size="small" icon={<RedoOutlined />}>
                重试退款
              </Button>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ];

  return (
    <div className="aftersale-list-page space-y-4">
      <Card bordered={false}>
        <Form form={form} layout="inline" className="gap-y-3">
          <Form.Item name="keyword">
            <Input
              placeholder="售后单号 / 订单号"
              allowClear
              style={{ width: 190 }}
              onPressEnter={handleSearch}
            />
          </Form.Item>

          <Form.Item name="type">
            <Select
              options={typeOptions}
              placeholder="售后类型"
              allowClear
              style={{ width: 120 }}
            />
          </Form.Item>

          <Form.Item name="status">
            <Select
              options={statusOptions}
              placeholder="售后状态"
              allowClear
              style={{ width: 130 }}
            />
          </Form.Item>

          <Form.Item name="timeRange">
            <RangePicker style={{ width: 250 }} />
          </Form.Item>

          <Form.Item>
            <Space>
              <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
                查询
              </Button>
              <Button icon={<ReloadOutlined />} onClick={handleReset}>
                重置
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Card>

      <Card bordered={false}>
        <Table<AdminAfterSaleListRow>
          rowKey="id"
          columns={columns}
          dataSource={rows}
          loading={loading}
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (t) => `共 ${t} 条售后单`,
            onChange: (p, ps) => {
              setPage(p);
              setPageSize(ps);
            },
          }}
          scroll={{ x: 1100 }}
        />
      </Card>

      {/* Approve Modal with Return Address */}
      <Modal
        title={`同意退货退款申请：${approveTarget?.after_sale_no}`}
        open={approveModalVisible}
        onCancel={() => setApproveModalVisible(false)}
        onOk={handleApproveSubmit}
        confirmLoading={approveSubmitting}
        okText="同意并通知买家退货"
        cancelText="取消"
        destroyOnClose
      >
        <Form form={approveForm} layout="vertical">
          <Form.Item
            name="return_receiver"
            label="退货收件人"
            rules={[{ required: true, whitespace: true, max: 64, message: '请输入收件人姓名' }]}
          >
            <Input placeholder="例：张三" />
          </Form.Item>
          <Form.Item
            name="return_phone"
            label="退货联系电话"
            rules={[{ required: true, whitespace: true, max: 32, message: '请输入联系电话' }]}
          >
            <Input placeholder="例：13800000000" />
          </Form.Item>
          <Form.Item
            name="return_address"
            label="详细退货地址"
            rules={[{ required: true, whitespace: true, max: 255, message: '请输入详细收货地址' }]}
          >
            <Input.TextArea rows={3} placeholder="例：浙江省杭州市西湖区文三路888号1号楼501" />
          </Form.Item>
        </Form>
      </Modal>

      {/* Reject Modal */}
      <Modal
        title="拒绝售后申请"
        open={rejectModalVisible}
        onCancel={() => setRejectModalVisible(false)}
        onOk={handleRejectSubmit}
        confirmLoading={rejectSubmitting}
        okText="确认拒绝"
        cancelText="取消"
        okButtonProps={{ danger: true }}
        destroyOnClose
      >
        <Form form={rejectForm} layout="vertical">
          <Form.Item
            name="reject_reason"
            label="拒绝原因"
            rules={[{ required: true, whitespace: true, max: 255, message: '请输入拒绝原因' }]}
          >
            <Input.TextArea rows={4} maxLength={255} showCount placeholder="请详细说明拒绝该售后申请的原因..." />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
