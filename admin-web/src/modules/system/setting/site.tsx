import { useEffect, useState } from 'react';
import { Button, Form, Input, message, Space, Spin } from 'antd';
import { api, type SiteBasicConfig } from '@/framework/http';
import { usePermission } from '@/framework/store';

export default function SiteSettingPage() {
  const [form] = Form.useForm<SiteBasicConfig>();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const hasPermission = usePermission();
  const canUpdate = hasPermission('system:setting:update');

  useEffect(() => {
    let active = true;
    const load = async () => {
      setLoading(true);
      const result = await api.api.getSystemSiteSetting();
      if (!active) return;
      setLoading(false);
      if (result.code !== 1 || !result.data) {
        message.error(result.message || '获取站点信息设置失败');
        return;
      }
      form.setFieldsValue(result.data);
    };
    void load();
    return () => {
      active = false;
    };
  }, [form]);

  const save = async (values: SiteBasicConfig) => {
    setSaving(true);
    const result = await api.api.saveSystemSiteSetting(values);
    setSaving(false);
    if (result.code !== 1) {
      message.error(result.message || '保存站点信息设置失败');
      return;
    }
    message.success('站点信息设置已保存');
    const refreshed = await api.api.getSystemSiteSetting();
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
          name="title"
          label="网站标题"
          rules={[{ required: true, whitespace: true, message: '请输入网站标题' }]}
        >
          <Input placeholder="请输入网站标题（Title）" maxLength={100} showCount />
        </Form.Item>

        <Form.Item
          name="keywords"
          label="网站关键词"
          extra="用于搜索引擎优化 SEO，多个关键词请用英文逗号或空格隔开"
        >
          <Input placeholder="如：数码商城,手机数码,优质正品" maxLength={200} showCount />
        </Form.Item>

        <Form.Item
          name="description"
          label="网站描述"
          extra="网站的简要介绍（Meta Description），在搜索引擎展示摘要"
        >
          <Input.TextArea
            rows={3}
            placeholder="请输入网站描述"
            maxLength={300}
            showCount
          />
        </Form.Item>

        <Form.Item name="icp" label="ICP备案号">
          <Input placeholder="如：浙ICP备xxxxxxxx号-1" maxLength={64} />
        </Form.Item>

        <Form.Item name="copyright" label="版权信息">
          <Input placeholder="如：© 2026 Mall. All rights reserved." maxLength={128} />
        </Form.Item>

        <Form.Item label=" " colon={false} style={{ marginTop: 24 }}>
          <Space>
            <Button
              type="primary"
              htmlType="submit"
              loading={saving}
              disabled={!canUpdate}
            >
              保存站点设置
            </Button>
          </Space>
        </Form.Item>
      </Form>
    </Spin>
  );
}
