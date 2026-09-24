import { useEffect, useState } from 'react';
import {
  Button,
  Divider,
  Form,
  Space,
  Spin,
  Switch,
  Typography,
  message,
} from 'antd';
import { api, type ReviewBasicConfig } from '@/framework/http';
import { usePermission } from '@/framework/store';

export default function ReviewSettingPage() {
  const [form] = Form.useForm<ReviewBasicConfig>();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  const canUpdate = usePermission()('system:setting:update');

  useEffect(() => {
    let active = true;
    const load = async () => {
      setLoading(true);
      const res = await api.api.getSystemReviewSetting();
      if (!active) return;
      setLoading(false);
      if (res.code !== 1 || !res.data) {
        message.error(res.message || '获取评价设置失败');
        return;
      }
      form.setFieldsValue(res.data);
    };
    void load();
    return () => {
      active = false;
    };
  }, [form]);

  const save = async (values: ReviewBasicConfig) => {
    setSaving(true);
    const result = await api.api.saveSystemReviewSetting({
      ...values,
      auto_review_enabled: false,
      auto_review_days: values.auto_review_days || 7,
    });
    setSaving(false);
    if (result.code !== 1) {
      message.error(result.message || '保存评价设置失败');
      return;
    }
    message.success('评价设置已保存');
    const refreshed = await api.api.getSystemReviewSetting();
    if (refreshed.code === 1 && refreshed.data) {
      form.setFieldsValue(refreshed.data);
    }
  };

  return (
    <Spin spinning={loading}>
      <Form
        form={form}
        labelCol={{ flex: '180px' }}
        wrapperCol={{ style: { maxWidth: 540 } }}
        onFinish={save}
        style={{ maxWidth: 840 }}
      >
        <Typography.Title level={5}>商品评价基础规则</Typography.Title>

        <Form.Item
          name="enabled"
          label="开启商品评价"
          valuePropName="checked"
          extra="开启后买家可以在已完成的订单中对购买的商品发表图文评价"
        >
          <Switch />
        </Form.Item>

        <Form.Item
          name="need_audit"
          label="评价人工审核"
          valuePropName="checked"
          extra="开启后买家提交的评价初始为「待审核」状态，需管理员审核通过后才在前台公开展示；关闭则提交后直接公开"
        >
          <Switch />
        </Form.Item>

        <Form.Item
          name="default_anonymous"
          label="默认匿名评价"
          valuePropName="checked"
          extra="开启后买家在移动端评价提交页默认勾选「匿名评价」"
        >
          <Switch />
        </Form.Item>

        <Divider style={{ margin: '24px 0' }} />
        <Typography.Text type="secondary">
          自动好评尚未上线，相关配置暂不开放。
        </Typography.Text>

        <Form.Item label=" " colon={false} style={{ marginTop: 32 }}>
          <Space>
            <Button
              type="primary"
              htmlType="submit"
              loading={saving}
              disabled={!canUpdate}
            >
              保存评价设置
            </Button>
          </Space>
        </Form.Item>
      </Form>
    </Spin>
  );
}
