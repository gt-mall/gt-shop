import { useEffect, useState } from 'react';
import {
  Alert,
  Button,
  Card,
  Descriptions,
  Divider,
  Form,
  InputNumber,
  Space,
  Spin,
  Switch,
  Typography,
  message,
} from 'antd';
import { api, type ReturnAddressItem, type TradeBasicConfig } from '@/framework/http';
import { usePermission } from '@/framework/store';
import ReturnAddressModal from './return-address-modal';

export default function TradeSettingPage() {
  const [form] = Form.useForm<TradeBasicConfig>();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [defaultReturnAddress, setDefaultReturnAddress] = useState<ReturnAddressItem | null>(null);
  const [returnModalOpen, setReturnModalOpen] = useState(false);

  const canUpdate = usePermission()('system:setting:update');

  const loadSettings = async () => {
    setLoading(true);
    const [tradeRes, defAddrRes] = await Promise.all([
      api.api.getSystemTradeSetting(),
      api.api.getDefaultOrderReturnAddress(),
    ]);
    setLoading(false);

    if (tradeRes.code === 1 && tradeRes.data) {
      form.setFieldsValue(tradeRes.data);
    } else {
      message.error(tradeRes.message || '获取交易设置失败');
    }

    if (defAddrRes.code === 1 && defAddrRes.data) {
      setDefaultReturnAddress(defAddrRes.data);
    } else {
      setDefaultReturnAddress(null);
    }
  };

  useEffect(() => {
    void loadSettings();
  }, []);

  const save = async (values: TradeBasicConfig) => {
    setSaving(true);
    const result = await api.api.saveSystemTradeSetting(values);
    setSaving(false);
    if (result.code !== 1) {
      message.error(result.message || '保存交易设置失败');
      return;
    }
    message.success('交易设置已保存');
    const refreshed = await api.api.getSystemTradeSetting();
    if (refreshed.code === 1 && refreshed.data) {
      form.setFieldsValue(refreshed.data);
    }
  };

  return (
    <Spin spinning={loading}>
      <Form
        form={form}
        labelCol={{ flex: '220px' }}
        wrapperCol={{ style: { maxWidth: 500 } }}
        onFinish={save}
        style={{ maxWidth: 840 }}
      >
        <Typography.Title level={5}>订单设置</Typography.Title>
        <Form.Item
          name="order_close_minutes"
          label="待付款订单自动关闭"
          rules={[{ required: true, message: '请输入关闭时间' }]}
          extra="买家提交订单后超时未付款将自动取消订单并回退锁定库存（建议 15~60 分钟，范围 1~1440）"
        >
          <InputNumber min={1} max={1440} addonAfter="分钟" style={{ width: 180 }} />
        </Form.Item>

        <Form.Item
          name="auto_receive_days"
          label="发货后自动确认收货"
          rules={[{ required: true, message: '请输入自动收货天数' }]}
          extra="商家发货后买家超时未确认收货，系统将自动变更为完成收货状态（范围 1~30 天）"
        >
          <InputNumber min={1} max={30} addonAfter="天" style={{ width: 180 }} />
        </Form.Item>

        <Divider style={{ margin: '24px 0' }} />

        <Typography.Title level={5}>售后与退款设置</Typography.Title>
        <Form.Item
          name="after_sale_days"
          label="订单完成后允许申请售后"
          rules={[{ required: true, message: '请输入售后有效天数' }]}
          extra="确认收货或订单完成之后，买家可在该有效期限内发起退货退款申请（范围 0~90 天，0 表示不允许申请）"
        >
          <InputNumber min={0} max={90} addonAfter="天" style={{ width: 180 }} />
        </Form.Item>

        <Form.Item
          name="allow_cancel_unshipped"
          label="待发货订单允许买家取消"
          valuePropName="checked"
          extra="开启后买家已付款但在待发货状态时，可自主直接申请取消订单"
        >
          <Switch />
        </Form.Item>

        <Form.Item
          name="auto_refund_on_cancel"
          label="待发货取消后自动退款"
          valuePropName="checked"
          extra="开启后买家取消待发货订单时，系统将直接调用微信退款原路退回，无需人工二次审核"
        >
          <Switch />
        </Form.Item>

        <Divider style={{ margin: '24px 0' }} />

        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
          <Typography.Title level={5} style={{ margin: 0 }}>
            退货地址库
          </Typography.Title>
          <Button type="default" onClick={() => setReturnModalOpen(true)}>
            管理退货地址
          </Button>
        </div>

        <Card size="small" style={{ marginBottom: 24, background: '#fafafa' }}>
          {defaultReturnAddress ? (
            <Descriptions column={1} size="small" title="当前默认退货地址快照">
              <Descriptions.Item label="仓库/名称">
                {defaultReturnAddress.name}
              </Descriptions.Item>
              <Descriptions.Item label="收件人">
                {defaultReturnAddress.receiver} ({defaultReturnAddress.phone})
              </Descriptions.Item>
              <Descriptions.Item label="退货地址">
                {defaultReturnAddress.address}
              </Descriptions.Item>
            </Descriptions>
          ) : (
            <Alert
              type="warning"
              showIcon
              message="暂未设置默认退货地址"
              description="买家申请退货退款审核通过时将无法自动回填退货收件信息，建议点击右上角「管理退货地址」添加默认退货地址。"
            />
          )}
        </Card>

        <Form.Item label=" " colon={false}>
          <Space>
            <Button
              type="primary"
              htmlType="submit"
              loading={saving}
              disabled={!canUpdate}
            >
              保存交易设置
            </Button>
          </Space>
        </Form.Item>
      </Form>

      <ReturnAddressModal
        open={returnModalOpen}
        onClose={() => setReturnModalOpen(false)}
        onDefaultChange={(addr) => setDefaultReturnAddress(addr)}
      />
    </Spin>
  );
}
