import { useCallback, useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import {
  Button,
  Card,
  Descriptions,
  Divider,
  Form,
  Image,
  Input,
  Modal,
  Space,
  Spin,
  Table,
  Tag,
  Timeline,
  Typography,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { ArrowLeftOutlined, CheckCircleOutlined, ClockCircleOutlined, SendOutlined } from '@ant-design/icons';

import { api } from '@/framework/http';
import type { OrderDetailResponse, OrderItemRow, OrderLogRow } from '@/framework/http';
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
      return <Tag color="success">{label}</Tag>;
    case 50:
      return <Tag color="default">{label}</Tag>;
    default:
      return <Tag>{label}</Tag>;
  }
}

export default function OrderDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const hasPermission = usePermission();
  const canShip = hasPermission('order:order:ship');

  const [detail, setDetail] = useState<OrderDetailResponse>();
  const [loading, setLoading] = useState(false);

  // Ship modal
  const [shipModalVisible, setShipModalVisible] = useState(false);
  const [shipForm] = Form.useForm<{ express_company: string; express_no: string }>();
  const [shipSubmitting, setShipSubmitting] = useState(false);

  const loadDetail = useCallback(async () => {
    const orderId = Number(id);
    if (!orderId || isNaN(orderId)) {
      message.error('无效的订单 ID');
      return;
    }

    setLoading(true);
    const res = await api.api.getOrderDetail({ id: orderId });
    setLoading(false);

    if (res.code === 1 && res.data) {
      setDetail(res.data);
    } else {
      message.error(res.message || '获取订单详情失败');
    }
  }, [id]);

  useEffect(() => {
    void loadDetail();
  }, [loadDetail]);

  const handleShipSubmit = async () => {
    const values = await shipForm.validateFields();
    if (!detail?.id) return;

    setShipSubmitting(true);
    const res = await api.api.shipOrder({
      id: detail.id,
      express_company: values.express_company.trim(),
      express_no: values.express_no.trim(),
    });
    setShipSubmitting(false);

    if (res.code === 1) {
      message.success('发货成功');
      setShipModalVisible(false);
      void loadDetail();
    } else {
      message.error(res.message || '发货失败');
    }
  };

  const itemColumns: ColumnsType<OrderItemRow> = [
    {
      title: '商品图片',
      dataIndex: 'image_url',
      key: 'image_url',
      width: 80,
      render: (url?: string, record?: OrderItemRow) =>
        url ? (
          <Image
            src={url}
            alt={record?.product_name}
            width={48}
            height={48}
            style={{ objectFit: 'cover', borderRadius: 4 }}
          />
        ) : (
          <div
            style={{
              width: 48,
              height: 48,
              backgroundColor: '#f5f5f5',
              borderRadius: 4,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: '#999',
              fontSize: 12,
            }}
          >
            无图
          </div>
        ),
    },
    {
      title: '商品名称',
      dataIndex: 'product_name',
      key: 'product_name',
      render: (name: string, record) => (
        <Space orientation="vertical" size={2}>
          <span style={{ fontWeight: 500 }}>{name}</span>
          {record.sku_code && (
            <Text type="secondary" style={{ fontSize: 12 }}>
              编码: {record.sku_code}
            </Text>
          )}
        </Space>
      ),
    },
    {
      title: 'SKU规格',
      dataIndex: 'sku_spec_values',
      key: 'sku_spec_values',
      render: (spec?: string) => {
        if (!spec) return <Text type="secondary">-</Text>;
        let parsed: Array<{ spec_name?: string; value_name?: string }> = [];
        if (typeof spec === 'string') {
          if (spec.startsWith('[') || spec.startsWith('{')) {
            const raw = JSON.parse(spec);
            parsed = Array.isArray(raw) ? raw : [raw];
          } else {
            return <span>{spec}</span>;
          }
        }
        if (parsed.length === 0) return <Text type="secondary">-</Text>;
        return (
          <Space size={4} wrap>
            {parsed.map((p, i) => (
              <Tag key={i}>
                {p.spec_name && p.value_name
                  ? `${p.spec_name}: ${p.value_name}`
                  : p.value_name || JSON.stringify(p)}
              </Tag>
            ))}
          </Space>
        );
      },
    },
    {
      title: '单价',
      dataIndex: 'unit_price',
      key: 'unit_price',
      width: 120,
      render: (p: string) => `¥${p}`,
    },
    {
      title: '数量',
      dataIndex: 'quantity',
      key: 'quantity',
      width: 100,
      render: (qty: number) => `× ${qty}`,
    },
    {
      title: '小计',
      dataIndex: 'total_amount',
      key: 'total_amount',
      width: 120,
      render: (amt: string) => (
        <span style={{ fontWeight: 600, color: '#f5222d' }}>¥{amt}</span>
      ),
    },
  ];

  if (loading && !detail) {
    return (
      <div style={{ padding: 48, textAlign: 'center' }}>
        <Spin size="large" />
      </div>
    );
  }

  if (!detail) {
    return (
      <Card>
        <Space orientation="vertical" align="center" style={{ width: '100%', padding: 32 }}>
          <Text type="secondary">订单不存在或已被删除</Text>
          <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/order/order')}>
            返回列表
          </Button>
        </Space>
      </Card>
    );
  }

  const addr = detail.address;
  const fullAddress = addr
    ? [addr.province, addr.city, addr.district, addr.address].filter(Boolean).join(' ')
    : '-';

  return (
    <Space orientation="vertical" size={16} style={{ width: '100%' }}>
      {/* 顶部操作与基本状态 */}
      <Card>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <Space size={16} align="center">
            <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/order/order')}>
              返回列表
            </Button>
            <div>
              <Space align="center" size={8}>
                <Title level={4} style={{ margin: 0 }}>
                  订单号：{detail.order_no}
                </Title>
                {renderStatusTag(detail.status, detail.status_text)}
              </Space>
              <div style={{ marginTop: 4 }}>
                <Text type="secondary">下单时间：{formatTime(detail.created_at)}</Text>
              </div>
            </div>
          </Space>
          {detail.status === 20 && canShip && (
            <Button
              type="primary"
              icon={<SendOutlined />}
              onClick={() => {
                shipForm.resetFields();
                setShipModalVisible(true);
              }}
            >
              订单发货
            </Button>
          )}
        </div>
      </Card>

      {/* 商品信息 */}
      <Card title="商品清单">
        <Table<OrderItemRow>
          rowKey="id"
          columns={itemColumns}
          dataSource={detail.items ?? []}
          pagination={false}
          size="middle"
        />
      </Card>

      {/* 订单金额与备注 */}
      <Card title="金额与结算">
        <Descriptions bordered column={{ xs: 1, sm: 2, md: 4 }}>
          <Descriptions.Item label="商品总额">¥{detail.total_amount}</Descriptions.Item>
          <Descriptions.Item label="运费金额">¥{detail.freight_amount}</Descriptions.Item>
          <Descriptions.Item label="优惠减免">¥{detail.discount_amount || '0.00'}</Descriptions.Item>
          <Descriptions.Item label="实付金额">
            <span style={{ fontSize: 18, fontWeight: 700, color: '#f5222d' }}>
              ¥{detail.pay_amount}
            </span>
          </Descriptions.Item>
          <Descriptions.Item label="买家留言 / 备注" span={4}>
            {detail.remark || <Text type="secondary">无备注</Text>}
          </Descriptions.Item>
        </Descriptions>
      </Card>

      {/* 会员与收货信息 */}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
        <Card title="买家信息">
          <Descriptions column={1}>
            <Descriptions.Item label="会员 ID">{detail.member_id}</Descriptions.Item>
            <Descriptions.Item label="会员昵称">
              {detail.member_nickname || <Text type="secondary">未设置</Text>}
            </Descriptions.Item>
            <Descriptions.Item label="绑定手机">
              {detail.member_mobile || <Text type="secondary">未绑定</Text>}
            </Descriptions.Item>
          </Descriptions>
        </Card>

        <Card title="收货信息">
          <Descriptions column={1}>
            <Descriptions.Item label="收货人">{addr?.name || '-'}</Descriptions.Item>
            <Descriptions.Item label="联系电话">{addr?.mobile || '-'}</Descriptions.Item>
            <Descriptions.Item label="收货地址">{fullAddress}</Descriptions.Item>
            {addr?.postal_code && (
              <Descriptions.Item label="邮政编码">{addr.postal_code}</Descriptions.Item>
            )}
          </Descriptions>
        </Card>
      </div>

      {/* 物流与关键时间节点 */}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
        <Card title="物流信息">
          <Descriptions column={1}>
            <Descriptions.Item label="物流公司">
              {detail.express_company || <Text type="secondary">-</Text>}
            </Descriptions.Item>
            <Descriptions.Item label="物流运单号">
              {detail.express_no ? (
                <Text copyable>{detail.express_no}</Text>
              ) : (
                <Text type="secondary">-</Text>
              )}
            </Descriptions.Item>
            <Descriptions.Item label="发货时间">
              {formatTime(detail.shipped_at || undefined) || <Text type="secondary">-</Text>}
            </Descriptions.Item>
          </Descriptions>
        </Card>

        <Card title="订单时间节点">
          <Descriptions column={1}>
            <Descriptions.Item label="创建时间">
              {formatTime(detail.created_at) || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="支付时间">
              {formatTime(detail.paid_at || undefined) || <Text type="secondary">-</Text>}
            </Descriptions.Item>
            <Descriptions.Item label="发货时间">
              {formatTime(detail.shipped_at || undefined) || <Text type="secondary">-</Text>}
            </Descriptions.Item>
            <Descriptions.Item label="完成时间">
              {formatTime(detail.completed_at || undefined) || <Text type="secondary">-</Text>}
            </Descriptions.Item>
            <Descriptions.Item label="取消时间">
              {formatTime(detail.cancelled_at || undefined) || <Text type="secondary">-</Text>}
            </Descriptions.Item>
          </Descriptions>
        </Card>
      </div>

      {/* 订单日志 */}
      <Card title="订单流转日志">
        {detail.logs && detail.logs.length > 0 ? (
          <Timeline
            style={{ marginTop: 16 }}
            items={detail.logs.map((log: OrderLogRow) => {
              const statusTransition =
                log.from_status !== undefined && log.to_status !== undefined ? (
                  <Tag color="cyan">
                    {log.from_status_text || `状态 ${log.from_status}`} →{' '}
                    {log.to_status_text || `状态 ${log.to_status}`}
                  </Tag>
                ) : null;

              return {
                color: 'blue',
                children: (
                  <Space orientation="vertical" size={2} style={{ marginBottom: 12 }}>
                    <Space size={8} align="center">
                      <span style={{ fontWeight: 600 }}>{log.action}</span>
                      {statusTransition}
                      <Text type="secondary" style={{ fontSize: 12 }}>
                        {formatTime(log.created_at)}
                      </Text>
                    </Space>
                    <div>
                      <Text type="secondary" style={{ fontSize: 13 }}>
                        操作人：{log.operator_name || log.operator_type || '系统'} ({log.operator_type})
                      </Text>
                    </div>
                    {log.remark && (
                      <div style={{ fontSize: 13, color: '#555' }}>
                        备注：{log.remark}
                      </div>
                    )}
                  </Space>
                ),
              };
            })}
          />
        ) : (
          <Text type="secondary">暂无流转日志</Text>
        )}
      </Card>

      {/* 发货弹窗 */}
      <Modal
        title={`订单发货 - ${detail.order_no}`}
        open={shipModalVisible}
        onOk={handleShipSubmit}
        confirmLoading={shipSubmitting}
        onCancel={() => setShipModalVisible(false)}
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
    </Space>
  );
}
