import { useCallback, useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import {
  Button,
  Card,
  Descriptions,
  Divider,
  Empty,
  Form,
  Image,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Select,
  Space,
  Spin,
  Table,
  Tag,
  Timeline,
  Typography,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  ArrowLeftOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
  CloseCircleOutlined,
  InboxOutlined,
  PlayCircleOutlined,
  RedoOutlined,
  SyncOutlined,
  VideoCameraOutlined,
} from '@ant-design/icons';

import {
  api,
  type AdminAfterSaleDetailResponse,
  type AfterSaleItemRow,
  type AfterSaleLogRow,
  type AfterSaleProofFile,
  type ReturnAddressItem,
} from '@/framework/http';
import { usePermission } from '@/framework/store';
import { formatTime } from '@/shared/utils/format';

const { Text, Title } = Typography;

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

const formatDuration = (ms?: number) => {
  if (!ms || ms <= 0) return '';
  const totalSeconds = Math.floor(ms / 1000);
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return `${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
};

export default function AfterSaleDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const hasPermission = usePermission();

  const canApprove = hasPermission('order:after_sale:approve');
  const canReject = hasPermission('order:after_sale:reject');
  const canReceive = hasPermission('order:after_sale:receive');
  const canRetryRefund = hasPermission('order:after_sale:retry_refund');

  const [detail, setDetail] = useState<AdminAfterSaleDetailResponse>();
  const [loading, setLoading] = useState(false);

  // Video preview modal
  const [videoModalUrl, setVideoModalUrl] = useState<string>();
  const [videoModalName, setVideoModalName] = useState<string>();

  // Approve modal (type 2)
  const [approveModalVisible, setApproveModalVisible] = useState(false);
  const [returnAddresses, setReturnAddresses] = useState<ReturnAddressItem[]>([]);
  const [returnAddressLoading, setReturnAddressLoading] = useState(false);
  const [approveForm] = Form.useForm<{
    return_address_id?: number;
    return_receiver?: string;
    return_phone?: string;
    return_address?: string;
  }>();
  const [approveSubmitting, setApproveSubmitting] = useState(false);

  // Reject modal
  const [rejectModalVisible, setRejectModalVisible] = useState(false);
  const [rejectForm] = Form.useForm<{ reject_reason: string }>();
  const [rejectSubmitting, setRejectSubmitting] = useState(false);

  // Receive modal (confirm receipt with restock qty)
  const [receiveModalVisible, setReceiveModalVisible] = useState(false);
  const [receiveRestockMap, setReceiveRestockMap] = useState<Record<number, number>>({});
  const [receiveSubmitting, setReceiveSubmitting] = useState(false);

  // Sync / retry refund state
  const [syncSubmitting, setSyncSubmitting] = useState(false);
  const [retrySubmitting, setRetrySubmitting] = useState(false);

  const loadDetail = useCallback(async () => {
    const afterSaleId = Number(id);
    if (!afterSaleId || isNaN(afterSaleId)) {
      message.error('无效的售后单 ID');
      return;
    }

    setLoading(true);
    const res = await api.api.getAfterSaleDetail({ id: afterSaleId });
    setLoading(false);

    if (res.code === 1 && res.data) {
      setDetail(res.data);
      // Pre-fill restock quantities to item quantity
      const initMap: Record<number, number> = {};
      (res.data.items ?? []).forEach((item) => {
        if (item.id) initMap[item.id] = item.quantity ?? 1;
      });
      setReceiveRestockMap(initMap);
    } else {
      message.error(res.message || '获取售后详情失败');
    }
  }, [id]);

  useEffect(() => {
    void loadDetail();
  }, [loadDetail]);

  const handleApprove = async () => {
    if (!detail?.id) return;
    if (detail.type === 2) {
      approveForm.resetFields();
      setApproveModalVisible(true);
      setReturnAddressLoading(true);
      const addrRes = await api.api.listOrderReturnAddresses({ status: 1 });
      setReturnAddressLoading(false);
      if (addrRes.code === 1 && addrRes.data) {
        setReturnAddresses(addrRes.data);
        const def = addrRes.data.find((a) => a.is_default === 1) || addrRes.data[0];
        if (def) {
          approveForm.setFieldsValue({
            return_address_id: def.id,
            return_receiver: def.receiver,
            return_phone: def.phone,
            return_address: def.address,
          });
        }
      }
    } else {
      Modal.confirm({
        title: '审核同意退款',
        content: `确认同意售后单【${detail.after_sale_no}】？退款金额 ¥${detail.refund_amount}，将直接发起退款流程。`,
        okText: '确认同意',
        cancelText: '取消',
        onOk: async () => {
          const res = await api.api.approveAfterSale({ id: detail.id! });
          if (res.code === 1) {
            message.success('审核同意成功');
            void loadDetail();
          } else {
            message.error(res.message || '操作失败');
          }
        },
      });
    }
  };

  const handleApproveSubmit = async () => {
    if (!detail?.id) return;
    const values = await approveForm.validateFields();
    setApproveSubmitting(true);
    const res = await api.api.approveAfterSale({
      id: detail.id,
      return_address_id: values.return_address_id,
      return_receiver: values.return_receiver?.trim(),
      return_phone: values.return_phone?.trim(),
      return_address: values.return_address?.trim(),
    });
    setApproveSubmitting(false);

    if (res.code === 1) {
      message.success('审核通过并已发送退货地址');
      setApproveModalVisible(false);
      void loadDetail();
    } else {
      message.error(res.message || '操作失败');
    }
  };

  const handleRejectSubmit = async () => {
    if (!detail?.id) return;
    const values = await rejectForm.validateFields();
    setRejectSubmitting(true);
    const res = await api.api.rejectAfterSale({
      id: detail.id,
      reject_reason: values.reject_reason.trim(),
    });
    setRejectSubmitting(false);

    if (res.code === 1) {
      message.success('已拒绝该售后申请');
      setRejectModalVisible(false);
      void loadDetail();
    } else {
      message.error(res.message || '操作失败');
    }
  };

  const handleReceiveSubmit = async () => {
    if (!detail?.id) return;
    const itemsPayload = (detail.items ?? []).map((item) => ({
      after_sale_item_id: item.id!,
      restock_quantity: receiveRestockMap[item.id!] ?? item.quantity ?? 0,
    }));

    setReceiveSubmitting(true);
    const res = await api.api.confirmReturnReceived({
      id: detail.id,
      items: itemsPayload,
    });
    setReceiveSubmitting(false);

    if (res.code === 1) {
      message.success('确认收货成功，系统已发起退款');
      setReceiveModalVisible(false);
      void loadDetail();
    } else {
      message.error(res.message || '确认收货失败');
    }
  };

  const handleSyncRefund = async () => {
    if (!detail?.id) return;
    setSyncSubmitting(true);
    const res = await api.api.syncAfterSaleRefund({ id: detail.id });
    setSyncSubmitting(false);
    if (res.code === 1) {
      message.success('退款状态同步完成');
      void loadDetail();
    } else {
      message.error(res.message || '退款状态同步失败');
    }
  };

  const handleRetryRefund = async () => {
    if (!detail?.id) return;
    setRetrySubmitting(true);
    const res = await api.api.retryAfterSaleRefund({ id: detail.id });
    setRetrySubmitting(false);
    if (res.code === 1) {
      message.success('退款重试已提交');
      void loadDetail();
    } else {
      message.error(res.message || '退款重试失败');
    }
  };

  // Proof files processing
  const proofFiles = detail?.proof_files ?? [];
  const imageProofs = proofFiles.filter((f) => f.file_type === 'image' || !f.file_type);
  const videoProofs = proofFiles.filter((f) => f.file_type === 'video');

  const itemColumns: ColumnsType<AfterSaleItemRow> = [
    {
      title: '商品图片',
      dataIndex: 'sku_image_url',
      key: 'sku_image_url',
      width: 80,
      render: (url?: string, record?: AfterSaleItemRow) =>
        url ? (
          <Image
            src={url}
            alt={record?.product_name}
            width={48}
            height={48}
            style={{ objectFit: 'cover', borderRadius: 4 }}
          />
        ) : (
          <div className="w-12 h-12 bg-gray-100 rounded flex items-center justify-center text-gray-400 text-xs">
            无图
          </div>
        ),
    },
    {
      title: '商品名称',
      dataIndex: 'product_name',
      key: 'product_name',
      render: (name?: string, record?: AfterSaleItemRow) => (
        <div>
          <div className="font-medium text-gray-800">{name || '-'}</div>
          {record?.sku_text && <div className="text-xs text-gray-400 mt-0.5">{record.sku_text}</div>}
          {record?.sku_code && <div className="text-xs text-gray-400 font-mono">编码: {record.sku_code}</div>}
        </div>
      ),
    },
    {
      title: '申请售后数量',
      dataIndex: 'quantity',
      key: 'quantity',
      width: 120,
      align: 'center',
      render: (q?: number) => <span className="font-medium">{q ?? 1} 件</span>,
    },
    {
      title: '退款金额',
      dataIndex: 'refund_amount',
      key: 'refund_amount',
      width: 130,
      align: 'right',
      render: (amount?: string) => <span className="font-medium text-rose-600">¥{amount || '0.00'}</span>,
    },
    {
      title: '已入库数量',
      dataIndex: 'restock_quantity',
      key: 'restock_quantity',
      width: 120,
      align: 'center',
      render: (q?: number) => (q !== undefined && q > 0 ? <Tag color="green">{q} 件</Tag> : '-'),
    },
  ];

  return (
    <Spin spinning={loading}>
      <div className="aftersale-detail-page space-y-4 max-w-6xl mx-auto pb-8">
        {/* Header Navigation & Operations */}
        <Card bordered={false}>
          <div className="flex flex-wrap items-center justify-between gap-4">
            <div className="flex items-center gap-3">
              <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/order/aftersale')}>
                返回列表
              </Button>
              <Title level={4} style={{ margin: 0 }}>
                售后单号: {detail?.after_sale_no || '-'}
              </Title>
              {renderStatusTag(detail?.status, detail?.status_text)}
              <Tag color={detail?.type === 2 ? 'orange' : 'cyan'}>
                {detail?.type_text || (detail?.type === 2 ? '退货退款' : '仅退款')}
              </Tag>
            </div>

            <Space wrap>
              {detail?.status === 10 && canApprove && (
                <Button type="primary" icon={<CheckCircleOutlined />} onClick={handleApprove}>
                  审核同意
                </Button>
              )}

              {detail?.status === 10 && canReject && (
                <Button
                  danger
                  icon={<CloseCircleOutlined />}
                  onClick={() => {
                    rejectForm.resetFields();
                    setRejectModalVisible(true);
                  }}
                >
                  审核拒绝
                </Button>
              )}

              {detail?.status === 30 && canReceive && (
                <Button
                  type="primary"
                  icon={<InboxOutlined />}
                  onClick={() => setReceiveModalVisible(true)}
                >
                  确认收货并退款
                </Button>
              )}

              {(detail?.status === 40 || detail?.status === 80) && canRetryRefund && (
                <Button
                  icon={<SyncOutlined />}
                  loading={syncSubmitting}
                  onClick={handleSyncRefund}
                >
                  同步退款状态
                </Button>
              )}

              {detail?.status === 80 && canRetryRefund && (
                <Popconfirm
                  title="确定重新发起退款？"
                  description="将重新调用支付退款接口"
                  onConfirm={handleRetryRefund}
                  okText="确定"
                  cancelText="取消"
                >
                  <Button type="primary" danger icon={<RedoOutlined />} loading={retrySubmitting}>
                    重试退款
                  </Button>
                </Popconfirm>
              )}
            </Space>
          </div>
        </Card>

        {/* Basic Information */}
        <Card title="基本信息" bordered={false}>
          <Descriptions column={{ xs: 1, sm: 2, md: 3 }} size="middle">
            <Descriptions.Item label="售后单号">{detail?.after_sale_no || '-'}</Descriptions.Item>
            <Descriptions.Item label="关联订单号">
              <a
                className="text-blue-600 hover:underline"
                onClick={() => navigate(`/order/order/${detail?.order_id}`)}
              >
                {detail?.order_no || '-'}
              </a>
            </Descriptions.Item>
            <Descriptions.Item label="申请时订单状态">
              {detail?.order_status_at_apply_text || '-'}
            </Descriptions.Item>

            <Descriptions.Item label="买家昵称">{detail?.member_nickname || '-'}</Descriptions.Item>
            <Descriptions.Item label="买家手机号">{detail?.member_mobile || '-'}</Descriptions.Item>
            <Descriptions.Item label="申请时间">{formatTime(detail?.applied_at || detail?.created_at)}</Descriptions.Item>

            <Descriptions.Item label="售后类型">
              <Tag color={detail?.type === 2 ? 'orange' : 'cyan'}>
                {detail?.type_text || (detail?.type === 2 ? '退货退款' : '仅退款')}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label="退款金额">
              <span className="font-semibold text-rose-600 text-base">¥{detail?.refund_amount || '0.00'}</span>
            </Descriptions.Item>
            <Descriptions.Item label="当前状态">
              {renderStatusTag(detail?.status, detail?.status_text)}
            </Descriptions.Item>

            <Descriptions.Item label="申请原因" span={3}>
              <Text strong>{detail?.reason || '-'}</Text>
            </Descriptions.Item>
            <Descriptions.Item label="问题描述" span={3}>
              <Text type="secondary">{detail?.description || '无详细描述'}</Text>
            </Descriptions.Item>

            {detail?.reject_reason && (
              <Descriptions.Item label="拒绝原因" span={3}>
                <Text type="danger">{detail.reject_reason}</Text>
              </Descriptions.Item>
            )}
          </Descriptions>
        </Card>

        {/* Return Logistics Information (if Return Type) */}
        {detail?.type === 2 && (
          <Card title="退货物流" bordered={false}>
            <Descriptions column={{ xs: 1, sm: 2, md: 3 }} size="middle">
              <Descriptions.Item label="退货收件人">{detail.return_receiver || '-'}</Descriptions.Item>
              <Descriptions.Item label="退货联系电话">{detail.return_phone || '-'}</Descriptions.Item>
              <Descriptions.Item label="详细退货地址" span={3}>
                {detail.return_address || '-'}
              </Descriptions.Item>

              <Descriptions.Item label="买家物流公司">
                {detail.buyer_return_company || <Text type="secondary">尚未填写</Text>}
              </Descriptions.Item>
              <Descriptions.Item label="买家物流单号">
                {detail.buyer_return_no ? (
                  <Text copyable strong>
                    {detail.buyer_return_no}
                  </Text>
                ) : (
                  <Text type="secondary">尚未填写</Text>
                )}
              </Descriptions.Item>
              <Descriptions.Item label="买家发货时间">
                {detail.buyer_returned_at ? formatTime(detail.buyer_returned_at) : '-'}
              </Descriptions.Item>

              <Descriptions.Item label="商家收货时间" span={3}>
                {detail.merchant_received_at ? formatTime(detail.merchant_received_at) : '未确认收货'}
              </Descriptions.Item>
            </Descriptions>
          </Card>
        )}

        {/* Refund Status & Info */}
        {detail?.refund && (
          <Card title="退款信息" bordered={false}>
            <Descriptions column={{ xs: 1, sm: 2, md: 3 }} size="middle">
              <Descriptions.Item label="退款单号">{detail.refund.refund_no || '-'}</Descriptions.Item>
              <Descriptions.Item label="退款金额">
                <span className="font-semibold text-rose-600">¥{detail.refund.refund_amount || '0.00'}</span>
              </Descriptions.Item>
              <Descriptions.Item label="退款状态">{detail.refund.status_text || '-'}</Descriptions.Item>
              <Descriptions.Item label="渠道退款单号">{detail.refund.provider_refund_no || '-'}</Descriptions.Item>
              <Descriptions.Item label="退款成功时间">
                {detail.refund.refunded_at ? formatTime(detail.refund.refunded_at) : '-'}
              </Descriptions.Item>
            </Descriptions>
          </Card>
        )}

        {/* After-Sale Goods Table */}
        <Card title="售后商品" bordered={false}>
          <Table<AfterSaleItemRow>
            rowKey="id"
            columns={itemColumns}
            dataSource={detail?.items ?? []}
            pagination={false}
          />
        </Card>

        {/* Proof Attachments Section (Images and Videos) */}
        <Card
          title={
            <div className="flex items-center gap-2">
              <span>凭证附件</span>
              <span className="text-xs text-gray-400 font-normal">
                (已加载短效安全预览链接 · 点击图片放大 · 点击视频播放)
              </span>
            </div>
          }
          bordered={false}
        >
          {proofFiles.length === 0 ? (
            <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="买家未上传凭证附件" />
          ) : (
            <div className="space-y-4">
              {/* Image Proofs */}
              {imageProofs.length > 0 && (
                <div>
                  <div className="text-xs font-semibold text-gray-500 mb-2">
                    图片凭证 ({imageProofs.length} 张)
                  </div>
                  <Image.PreviewGroup>
                    <div className="flex flex-wrap gap-3">
                      {imageProofs.map((img) => (
                        <div
                          key={img.id}
                          className="relative border rounded p-1 bg-gray-50 flex flex-col items-center justify-center hover:border-blue-400 transition-colors"
                        >
                          <Image
                            src={img.url}
                            alt={img.original_name || '凭证图片'}
                            width={100}
                            height={100}
                            style={{ objectFit: 'cover', borderRadius: 4 }}
                          />
                          <div className="text-[11px] text-gray-500 truncate max-w-[100px] mt-1 text-center">
                            {img.original_name || '凭证图片'}
                          </div>
                        </div>
                      ))}
                    </div>
                  </Image.PreviewGroup>
                </div>
              )}

              {/* Video Proofs */}
              {videoProofs.length > 0 && (
                <div>
                  <div className="text-xs font-semibold text-gray-500 mb-2">
                    视频凭证 ({videoProofs.length} 个)
                  </div>
                  <div className="flex flex-wrap gap-3">
                    {videoProofs.map((vid) => (
                      <div
                        key={vid.id}
                        className="relative w-44 h-32 border rounded-lg bg-slate-900 cursor-pointer overflow-hidden group shadow-sm flex flex-col justify-between p-2"
                        onClick={() => {
                          setVideoModalUrl(vid.url);
                          setVideoModalName(vid.original_name || '售后视频凭证');
                        }}
                      >
                        {vid.cover_url ? (
                          <img
                            src={vid.cover_url}
                            alt={vid.original_name}
                            className="absolute inset-0 w-full h-full object-cover opacity-60 group-hover:opacity-80 transition-opacity"
                          />
                        ) : null}

                        <div className="z-10 flex items-center justify-between text-white text-xs">
                          <Tag color="geekblue" style={{ margin: 0, fontSize: 10 }}>
                            <VideoCameraOutlined style={{ marginRight: 3 }} />
                            视频
                          </Tag>
                          {vid.duration_ms ? (
                            <span className="bg-black/60 px-1.5 py-0.5 rounded text-[10px]">
                              {formatDuration(vid.duration_ms)}
                            </span>
                          ) : null}
                        </div>

                        {/* Centered Play Button */}
                        <div className="z-10 absolute inset-0 flex items-center justify-center">
                          <PlayCircleOutlined className="text-white/90 text-3xl group-hover:scale-110 group-hover:text-blue-400 transition-all filter drop-shadow" />
                        </div>

                        <div className="z-10 text-[11px] text-white/90 truncate bg-black/40 px-1 rounded">
                          {vid.original_name || '视频凭证'}
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}
        </Card>

        {/* Operation Logs Timeline */}
        <Card title="操作日志" bordered={false}>
          {detail?.logs && detail.logs.length > 0 ? (
            <Timeline mode="left">
              {detail.logs.map((log: AfterSaleLogRow) => (
                <Timeline.Item
                  key={log.id}
                  dot={<ClockCircleOutlined style={{ fontSize: '14px', color: '#1890ff' }} />}
                >
                  <div className="space-y-1">
                    <div className="flex items-center gap-2">
                      <span className="font-semibold text-gray-800">{log.action || '售后流转'}</span>
                      <Tag color="purple">{log.operator_type_text || '系统'}</Tag>
                      <Text type="secondary" style={{ fontSize: 12 }}>
                        操作人: {log.operator_name || '-'}
                      </Text>
                      <Text type="secondary" style={{ fontSize: 12 }}>
                        {formatTime(log.created_at)}
                      </Text>
                    </div>
                    {log.from_status !== undefined && log.to_status !== undefined && (
                      <div className="text-xs text-gray-500">
                        状态变迁: {log.from_status_text} → {log.to_status_text}
                      </div>
                    )}
                    {log.remark && (
                      <div className="text-xs text-gray-600 bg-gray-50 p-2 rounded max-w-xl">
                        {log.remark}
                      </div>
                    )}
                  </div>
                </Timeline.Item>
              ))}
            </Timeline>
          ) : (
            <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无操作日志" />
          )}
        </Card>

        {/* Video Preview Modal (Standard Video Controls, NO Autoplay) */}
        <Modal
          title={
            <div className="flex items-center gap-2">
              <VideoCameraOutlined className="text-blue-500" />
              <span>{videoModalName || '售后凭证视频播放'}</span>
            </div>
          }
          open={!!videoModalUrl}
          onCancel={() => {
            setVideoModalUrl(undefined);
            setVideoModalName(undefined);
          }}
          footer={null}
          width={720}
          destroyOnClose
          centered
        >
          {videoModalUrl && (
            <div style={{ textAlign: 'center', padding: '10px 0', background: '#000', borderRadius: 6 }}>
              <video
                src={videoModalUrl}
                controls
                style={{ maxWidth: '100%', maxHeight: '65vh', borderRadius: 4 }}
                preload="metadata"
              />
            </div>
          )}
        </Modal>

        {/* Approve Return Modal */}
        <Modal
          title={`同意退货申请：${detail?.after_sale_no}`}
          open={approveModalVisible}
          onCancel={() => setApproveModalVisible(false)}
          onOk={handleApproveSubmit}
          confirmLoading={approveSubmitting}
          okText="同意并通知买家退货"
          cancelText="取消"
          destroyOnClose
        >
          <Form form={approveForm} layout="vertical">
            <Form.Item name="return_address_id" label="快捷选择退货地址库">
              <Select
                loading={returnAddressLoading}
                placeholder="从已保存的退货地址库选择并自动填充"
                allowClear
                options={returnAddresses.map((a) => ({
                  value: a.id,
                  label: `${a.name} (${a.receiver} ${a.phone})${a.is_default === 1 ? ' [默认]' : ''}`,
                }))}
                onChange={(selectedId) => {
                  const found = returnAddresses.find((a) => a.id === selectedId);
                  if (found) {
                    approveForm.setFieldsValue({
                      return_receiver: found.receiver,
                      return_phone: found.phone,
                      return_address: found.address,
                    });
                  }
                }}
              />
            </Form.Item>
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

        {/* Confirm Receive & Restock Modal */}
        <Modal
          title="确认收货并处理退款"
          open={receiveModalVisible}
          onCancel={() => setReceiveModalVisible(false)}
          onOk={handleReceiveSubmit}
          confirmLoading={receiveSubmitting}
          okText="确认收货并退款"
          cancelText="取消"
          width={600}
          destroyOnClose
        >
          <div className="space-y-3 py-2">
            <p className="text-gray-600 text-sm">
              请核对退回的商品数量及实际入库数量。确认收货后，系统将自动向买家退款 ¥{detail?.refund_amount}。
            </p>
            <Table<AfterSaleItemRow>
              rowKey="id"
              dataSource={detail?.items ?? []}
              pagination={false}
              size="small"
              columns={[
                {
                  title: '商品',
                  dataIndex: 'product_name',
                  key: 'product_name',
                  render: (name?: string, r?: AfterSaleItemRow) => (
                    <div>
                      <div className="font-medium text-xs">{name}</div>
                      <div className="text-[11px] text-gray-400">{r?.sku_text}</div>
                    </div>
                  ),
                },
                {
                  title: '申请数量',
                  dataIndex: 'quantity',
                  key: 'quantity',
                  width: 90,
                  align: 'center',
                  render: (q?: number) => `${q ?? 1} 件`,
                },
                {
                  title: '实际入库数量',
                  key: 'restock',
                  width: 140,
                  render: (_: unknown, r: AfterSaleItemRow) => (
                    <InputNumber
                      min={0}
                      max={r.quantity ?? 9999}
                      value={receiveRestockMap[r.id!] ?? r.quantity ?? 1}
                      onChange={(val) => {
                        setReceiveRestockMap((prev) => ({
                          ...prev,
                          [r.id!]: Number(val) || 0,
                        }));
                      }}
                    />
                  ),
                },
              ]}
            />
          </div>
        </Modal>
      </div>
    </Spin>
  );
}
