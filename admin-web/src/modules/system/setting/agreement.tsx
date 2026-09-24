import { useEffect, useState } from 'react';
import {
  Button,
  Form,
  Space,
  Spin,
  Tabs,
  Typography,
  message,
} from 'antd';
import { api, type AgreementBasicConfig } from '@/framework/http';
import { usePermission } from '@/framework/store';
import { RichEditor } from '@/shared/components/rich-editor';

export default function AgreementSettingPage() {
  const [form] = Form.useForm<AgreementBasicConfig>();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [activeTab, setActiveTab] = useState('user');

  const canUpdate = usePermission()('system:setting:update');

  useEffect(() => {
    let active = true;
    const load = async () => {
      setLoading(true);
      const res = await api.api.getSystemAgreementSetting();
      if (!active) return;
      setLoading(false);
      if (res.code !== 1 || !res.data) {
        message.error(res.message || '获取协议设置失败');
        return;
      }
      form.setFieldsValue({
        user_agreement: res.data.user_agreement || '',
        privacy_policy: res.data.privacy_policy || '',
        after_sale_policy: res.data.after_sale_policy || '',
      });
    };
    void load();
    return () => {
      active = false;
    };
  }, [form]);

  const save = async (values: AgreementBasicConfig) => {
    setSaving(true);
    const result = await api.api.saveSystemAgreementSetting(values);
    setSaving(false);
    if (result.code !== 1) {
      message.error(result.message || '保存协议设置失败');
      return;
    }
    message.success('协议与政策设置已保存');
    const refreshed = await api.api.getSystemAgreementSetting();
    if (refreshed.code === 1 && refreshed.data) {
      form.setFieldsValue(refreshed.data);
    }
  };

  const tabItems = [
    {
      key: 'user',
      label: '用户协议',
      children: (
        <div>
          <Typography.Paragraph type="secondary" style={{ marginBottom: 12 }}>
            用户在注册、登录及使用商城各项服务时所遵守的基础法律条款。
          </Typography.Paragraph>
          <Form.Item name="user_agreement" noStyle>
            <RichEditor minHeight={420} placeholder="请输入用户协议详细内容..." />
          </Form.Item>
        </div>
      ),
    },
    {
      key: 'privacy',
      label: '隐私政策',
      children: (
        <div>
          <Typography.Paragraph type="secondary" style={{ marginBottom: 12 }}>
            规范商城如何收集、存储、使用及保护买家个人信息的隐私保护政策。
          </Typography.Paragraph>
          <Form.Item name="privacy_policy" noStyle>
            <RichEditor minHeight={420} placeholder="请输入隐私政策详细内容..." />
          </Form.Item>
        </div>
      ),
    },
    {
      key: 'after_sale',
      label: '售后政策',
      children: (
        <div>
          <Typography.Paragraph type="secondary" style={{ marginBottom: 12 }}>
            买家退货、退款、换货及质量问题处理流程与保障规范说明。
          </Typography.Paragraph>
          <Form.Item name="after_sale_policy" noStyle>
            <RichEditor minHeight={420} placeholder="请输入售后政策详细内容..." />
          </Form.Item>
        </div>
      ),
    },
  ];

  return (
    <Spin spinning={loading}>
      <Form
        form={form}
        layout="vertical"
        onFinish={save}
        style={{ maxWidth: 960 }}
      >
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          items={tabItems}
          type="card"
          style={{ marginBottom: 20 }}
        />

        <Form.Item>
          <Space>
            <Button
              type="primary"
              htmlType="submit"
              loading={saving}
              disabled={!canUpdate}
              size="large"
            >
              保存协议设置
            </Button>
          </Space>
        </Form.Item>
      </Form>
    </Spin>
  );
}
