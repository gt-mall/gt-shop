import { useEffect, useState } from 'react';
import {
  Alert,
  Button,
  Card,
  Divider,
  Form,
  Input,
  Space,
  Switch,
  Tag,
  Typography,
  message,
} from 'antd';
import { api, type Config, type PublicConfig } from '@/framework/http';
import { usePermission } from '@/framework/store';

export default function WechatPaySetting() {
  const [form] = Form.useForm<Config>();
  const [status, setStatus] = useState<PublicConfig>();
  const [wechatEnabled, setWechatEnabled] = useState(true);
  const [savingSwitch, setSavingSwitch] = useState(false);
  const [saving, setSaving] = useState(false);
  const canUpdate = usePermission()('system:setting:update');

  useEffect(() => {
    let active = true;
    const load = async () => {
      const [payRes, paymentBasicRes] = await Promise.all([
        api.api.getSystemWechatPaySetting(),
        api.api.getSystemPaymentSetting(),
      ]);

      if (!active) return;

      if (paymentBasicRes.code === 1 && paymentBasicRes.data) {
        setWechatEnabled(paymentBasicRes.data.wechat_enabled ?? true);
      }

      if (payRes.code !== 1 || !payRes.data) {
        message.error(payRes.message || '获取支付设置失败');
        return;
      }
      setStatus(payRes.data);
      form.setFieldsValue({
        mch_id: payRes.data.mch_id,
        merchant_serial_no: payRes.data.merchant_serial_no,
        public_key_id: payRes.data.public_key_id,
        notify_base_url: payRes.data.notify_base_url,
      });
    };
    void load();
    return () => {
      active = false;
    };
  }, [form]);

  const handleToggleWechatEnabled = async (checked: boolean) => {
    setSavingSwitch(true);
    const res = await api.api.saveSystemPaymentSetting({ wechat_enabled: checked });
    setSavingSwitch(false);
    if (res.code !== 1) {
      message.error(res.message || '更新支付状态失败');
      return;
    }
    setWechatEnabled(checked);
    message.success(checked ? '微信支付已开启' : '微信支付已关闭');
  };

  const save = async (values: Config) => {
    setSaving(true);
    const result = await api.api.saveSystemWechatPaySetting(values);
    setSaving(false);
    if (result.code !== 1) {
      message.error(result.message || '保存支付配置失败');
      return;
    }
    message.success('支付配置已保存');
    form.setFieldsValue({ merchant_private_key: '', public_key: '', api_v3_key: '' });
    const refreshed = await api.api.getSystemWechatPaySetting();
    if (refreshed.code === 1 && refreshed.data) setStatus(refreshed.data);
  };

  return (
    <section style={{ maxWidth: 840, marginTop: 16 }}>
      <Typography.Title level={5}>支付方式状态</Typography.Title>
      <Card size="small" style={{ marginBottom: 24 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <div>
            <Space>
              <Typography.Text strong>微信小程序支付</Typography.Text>
              {wechatEnabled ? (
                <Tag color="success">已启用</Tag>
              ) : (
                <Tag color="default">已停用</Tag>
              )}
            </Space>
            <div style={{ marginTop: 4 }}>
              <Typography.Text type="secondary" style={{ fontSize: 13 }}>
                控制买家在小程序结账时是否提供微信支付入口。关闭后前台将隐藏微信支付。
              </Typography.Text>
            </div>
          </div>
          <Switch
            checked={wechatEnabled}
            loading={savingSwitch}
            disabled={!canUpdate}
            onChange={(checked) => void handleToggleWechatEnabled(checked)}
          />
        </div>
      </Card>

      <Divider style={{ margin: '24px 0' }} />

      <Typography.Title level={5}>微信小程序支付 · 普通商户参数</Typography.Title>
      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 20 }}
        message="密钥只会提交给后端加密保存，不会从接口回显。留空表示保留原值。请勿将密钥发送到聊天或提交到代码仓库。"
      />
      <Form form={form} onFinish={save} labelCol={{ flex: '160px' }} wrapperCol={{ style: { maxWidth: 580 } }}>
        <Form.Item name="mch_id" label="商户号" rules={[{ required: true, message: '请输入普通商户号' }]}>
          <Input autoComplete="off" />
        </Form.Item>
        <Form.Item name="merchant_serial_no" label="商户证书序列号" rules={[{ required: true, message: '请输入商户证书序列号' }]}>
          <Input autoComplete="off" />
        </Form.Item>
        <Form.Item
          name="merchant_private_key"
          label="商户 API 私钥"
          extra={status?.merchant_private_key_configured ? '已配置；留空保持原值' : '粘贴 apiclient_key.pem 内容'}
        >
          <Input.TextArea rows={4} autoComplete="off" />
        </Form.Item>
        <Form.Item name="public_key_id" label="微信支付公钥 ID" rules={[{ required: true, message: '请输入公钥 ID' }]}>
          <Input autoComplete="off" placeholder="PUB_KEY_ID_..." />
        </Form.Item>
        <Form.Item
          name="public_key"
          label="微信支付公钥"
          extra={status?.public_key_configured ? '已配置；留空保持原值' : '粘贴微信支付公钥 PEM 内容'}
        >
          <Input.TextArea rows={4} autoComplete="off" />
        </Form.Item>
        <Form.Item
          name="api_v3_key"
          label="APIv3 密钥"
          extra={status?.api_v3_key_configured ? '已配置；留空保持原值' : '32 字节，用于解密支付和退款通知'}
        >
          <Input.Password autoComplete="new-password" />
        </Form.Item>
        <Form.Item
          name="notify_base_url"
          label="支付回调域名"
          rules={[{ required: true, type: 'url', message: '请输入公网 HTTPS 域名' }]}
          extra="支付回调与退款回调会自动拼接接口路径"
        >
          <Input />
        </Form.Item>
        <Form.Item label=" " colon={false}>
          <Space>
            <Button type="primary" htmlType="submit" loading={saving} disabled={!canUpdate}>
              保存商户密钥配置
            </Button>
          </Space>
        </Form.Item>
      </Form>
    </section>
  );
}
