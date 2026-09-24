import { useEffect, useState } from 'react';
import {
  Button,
  Card,
  Divider,
  Form,
  Input,
  Space,
  Spin,
  Typography,
  message,
} from 'antd';
import { api, type LogisticsBasicConfig } from '@/framework/http';
import { usePermission } from '@/framework/store';
import ReturnAddressModal from './return-address-modal';

export default function LogisticsSettingPage() {
  const [form] = Form.useForm<LogisticsBasicConfig>();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [returnModalOpen, setReturnModalOpen] = useState(false);

  const canUpdate = usePermission()('system:setting:update');

  useEffect(() => {
    let active = true;
    const load = async () => {
      setLoading(true);
      const res = await api.api.getSystemLogisticsSetting();
      if (!active) return;
      setLoading(false);
      if (res.code !== 1 || !res.data) {
        message.error(res.message || '获取配送设置失败');
        return;
      }
      form.setFieldsValue(res.data);
    };
    void load();
    return () => {
      active = false;
    };
  }, [form]);

  const save = async (values: LogisticsBasicConfig) => {
    setSaving(true);
    const result = await api.api.saveSystemLogisticsSetting(values);
    setSaving(false);
    if (result.code !== 1) {
      message.error(result.message || '保存配送设置失败');
      return;
    }
    message.success('配送设置已保存');
    const refreshed = await api.api.getSystemLogisticsSetting();
    if (refreshed.code === 1 && refreshed.data) {
      form.setFieldsValue(refreshed.data);
    }
  };

  return (
    <Spin spinning={loading}>
      <Form
        form={form}
        labelCol={{ flex: '160px' }}
        wrapperCol={{ style: { maxWidth: 540 } }}
        onFinish={save}
        style={{ maxWidth: 840 }}
      >
        <Typography.Title level={5}>默认发货人信息</Typography.Title>
        <Typography.Paragraph type="secondary">
          设置商城订单发货时的默认发件人信息，可用于物流面单打印及快递承运人备案。
        </Typography.Paragraph>

        <Form.Item
          name="sender_name"
          label="默认发货人"
          rules={[{ max: 64, message: '发货人姓名最多 64 个字符' }]}
        >
          <Input placeholder="请输入发件人姓名" maxLength={64} />
        </Form.Item>

        <Form.Item
          name="sender_phone"
          label="联系电话"
          rules={[{ max: 32, message: '联系电话最多 32 个字符' }]}
        >
          <Input placeholder="请输入发件人联系电话" maxLength={32} />
        </Form.Item>

        <Form.Item
          name="sender_address"
          label="发货地址"
          rules={[{ max: 255, message: '发件详细地址最多 255 个字符' }]}
        >
          <Input.TextArea
            rows={3}
            placeholder="请输入仓库详细发货地址"
            maxLength={255}
            showCount
          />
        </Form.Item>

        <Form.Item label=" " colon={false}>
          <Space>
            <Button
              type="primary"
              htmlType="submit"
              loading={saving}
              disabled={!canUpdate}
            >
              保存配送设置
            </Button>
          </Space>
        </Form.Item>

        <Divider style={{ margin: '32px 0 24px' }} />

        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
          <div>
            <Typography.Title level={5} style={{ margin: 0 }}>
              退货地址库
            </Typography.Title>
            <Typography.Text type="secondary" style={{ fontSize: 13 }}>
              管理商城退货仓与收件信息。
            </Typography.Text>
          </div>
          <Button type="default" onClick={() => setReturnModalOpen(true)}>
            管理退货地址
          </Button>
        </div>

        <Card size="small" style={{ background: '#fafafa' }}>
          <Typography.Text type="secondary">
            运费规则请前往「商品管理 → 运费模板」进行配置。
          </Typography.Text>
        </Card>
      </Form>

      <ReturnAddressModal
        open={returnModalOpen}
        onClose={() => setReturnModalOpen(false)}
      />
    </Spin>
  );
}
