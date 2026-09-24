import { useEffect, useState } from 'react';
import {
  Alert,
  Button,
  Form,
  Space,
  Spin,
  Switch,
  Typography,
  message,
} from 'antd';
import { api, type MemberBasicConfig } from '@/framework/http';
import { usePermission } from '@/framework/store';

export default function MemberSettingPage() {
  const [form] = Form.useForm<MemberBasicConfig>();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  const canUpdate = usePermission()('system:setting:update');

  useEffect(() => {
    let active = true;
    const load = async () => {
      setLoading(true);
      const res = await api.api.getSystemMemberSetting();
      if (!active) return;
      setLoading(false);
      if (res.code !== 1 || !res.data) {
        message.error(res.message || '获取用户设置失败');
        return;
      }
      form.setFieldsValue(res.data);
    };
    void load();
    return () => {
      active = false;
    };
  }, [form]);

  const save = async (values: MemberBasicConfig) => {
    setSaving(true);
    const result = await api.api.saveSystemMemberSetting(values);
    setSaving(false);
    if (result.code !== 1) {
      message.error(result.message || '保存用户设置失败');
      return;
    }
    message.success('用户设置已保存');
    const refreshed = await api.api.getSystemMemberSetting();
    if (refreshed.code === 1 && refreshed.data) {
      form.setFieldsValue(refreshed.data);
    }
  };

  return (
    <Spin spinning={loading}>
      <Form
        form={form}
        labelCol={{ flex: '200px' }}
        wrapperCol={{ style: { maxWidth: 540 } }}
        onFinish={save}
        style={{ maxWidth: 840 }}
      >
        <Typography.Title level={5}>注册与登录渠道控制</Typography.Title>
        <Alert
          type="info"
          showIcon
          style={{ marginBottom: 24 }}
          message="注意：至少保留一种登录渠道开启，避免造成已有会员无法登录商城系统。"
        />

        <Form.Item
          name="register_enabled"
          label="允许新用户注册"
          valuePropName="checked"
          extra="开启后允许新用户注册账号；关闭后新用户将无法注册，仅已有用户可登录"
        >
          <Switch />
        </Form.Item>

        <Form.Item
          name="password_login_enabled"
          label="账号密码登录"
          valuePropName="checked"
          extra="开启后允许用户使用手机号/账号与密码登录商城"
        >
          <Switch />
        </Form.Item>

        <Form.Item
          name="wechat_login_enabled"
          label="微信小程序一键登录"
          valuePropName="checked"
          extra="开启后允许小程序端用户通过微信快捷授权静默/授权登录"
        >
          <Switch />
        </Form.Item>

        <Form.Item label=" " colon={false} style={{ marginTop: 32 }}>
          <Space>
            <Button
              type="primary"
              htmlType="submit"
              loading={saving}
              disabled={!canUpdate}
            >
              保存用户设置
            </Button>
          </Space>
        </Form.Item>
      </Form>
    </Spin>
  );
}
