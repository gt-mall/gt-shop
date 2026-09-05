import { useEffect, useState } from 'react';
import { Button, Form, Input, message, Space, Spin } from 'antd';
import { api, type ShopBasicConfig } from '@/framework/http';
import { usePermission } from '@/framework/store';
import { ImagePicker } from '@/modules/system/file';

export default function ShopSettingPage() {
  const [form] = Form.useForm<ShopBasicConfig>();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const hasPermission = usePermission();
  const canUpdate = hasPermission('system:setting:update');

  useEffect(() => {
    let active = true;
    const load = async () => {
      setLoading(true);
      const result = await api.api.getSystemShopSetting();
      if (!active) return;
      setLoading(false);
      if (result.code !== 1 || !result.data) {
        message.error(result.message || '获取商城基础设置失败');
        return;
      }
      form.setFieldsValue(result.data);
    };
    void load();
    return () => {
      active = false;
    };
  }, [form]);

  const save = async (values: ShopBasicConfig) => {
    setSaving(true);
    const result = await api.api.saveSystemShopSetting(values);
    setSaving(false);
    if (result.code !== 1) {
      message.error(result.message || '保存商城基础设置失败');
      return;
    }
    message.success('商城基础设置已保存');
    const refreshed = await api.api.getSystemShopSetting();
    if (refreshed.code === 1 && refreshed.data) {
      form.setFieldsValue(refreshed.data);
    }
  };

  return (
    <Spin spinning={loading}>
      <Form
        form={form}
        labelCol={{ flex: '150px' }}
        wrapperCol={{ style: { maxWidth: 540 } }}
        onFinish={save}
        style={{ maxWidth: 800 }}
      >
        <Form.Item
          name="name"
          label="商城名称"
          rules={[{ required: true, whitespace: true, message: '请输入商城名称' }]}
        >
          <Input placeholder="请输入商城名称" maxLength={64} showCount />
        </Form.Item>

        <Form.Item
          name="logo_id"
          label="商城 Logo"
          extra="建议尺寸 200x60，支持 png、jpg、svg 格式，透明底色效果更佳"
        >
          <ImagePicker />
        </Form.Item>

        <Form.Item
          name="favicon_id"
          label="浏览器图标"
          extra="浏览器标签页显示的小图标，建议尺寸 32x32 或 64x64，.ico 或 .png 格式"
        >
          <ImagePicker />
        </Form.Item>

        <Form.Item name="service_name" label="客服名称">
          <Input placeholder="如：在线客服 / 售后管家" maxLength={32} />
        </Form.Item>

        <Form.Item name="service_phone" label="客服电话">
          <Input placeholder="请输入客服联系电话或 400 电话" maxLength={32} />
        </Form.Item>

        <Form.Item name="service_wechat" label="客服微信">
          <Input placeholder="请输入客服微信号" maxLength={64} />
        </Form.Item>

        <Form.Item
          name="service_wechat_qrcode_id"
          label="客服微信二维码"
          extra="微信客服或企业微信二维码，方便顾客扫码联系"
        >
          <ImagePicker />
        </Form.Item>

        <Form.Item name="description" label="商城简介">
          <Input.TextArea
            rows={4}
            placeholder="请输入商城简介与品牌介绍"
            maxLength={500}
            showCount
          />
        </Form.Item>

        <Form.Item label=" " colon={false} style={{ marginTop: 24 }}>
          <Space>
            <Button
              type="primary"
              htmlType="submit"
              loading={saving}
              disabled={!canUpdate}
            >
              保存商城设置
            </Button>
          </Space>
        </Form.Item>
      </Form>
    </Spin>
  );
}
