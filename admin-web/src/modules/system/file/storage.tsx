import { useEffect, useState } from 'react';
import { Alert, Button, Card, Checkbox, Form, Input, message, Select, Spin, Tabs } from 'antd';
import { api, type FileStorageConfig } from '@/framework/http';

const providers = [
  { key: 'oss', label: '阿里云 OSS', endpoint: 'https://oss-cn-hangzhou.aliyuncs.com' },
  { key: 'cos', label: '腾讯云 COS', endpoint: '可留空，根据 Bucket 和 Region 自动生成' },
  { key: 's3', label: 'Amazon S3 / 兼容存储', endpoint: 'AWS 可留空；兼容存储填写 Endpoint' },
] as const;

export default function FileStoragePage({ card = true }: { card?: boolean } = {}) {
  const [form] = Form.useForm<FileStorageConfig>();
  const [config, setConfig] = useState<FileStorageConfig>();
  const [saving, setSaving] = useState(false);
  const [loading, setLoading] = useState(true);
  const [version, setVersion] = useState(0);
  const defaultStorage = Form.useWatch('default_storage', form);

  useEffect(() => {
    let active = true;
    const load = async () => {
      setLoading(true);
      const result = await api.api.getFileStorageConfig();
      if (!active) return;
      setLoading(false);
      if (result.code !== 1 || !result.data) { message.error(result.message); return; }
      form.setFieldsValue(result.data);
      setConfig(result.data);
    };
    void load();
    return () => { active = false; };
  }, [form, version]);

  const save = async (values: FileStorageConfig) => {
    setSaving(true);
    const result = await api.api.saveFileStorageConfig(values);
    setSaving(false);
    if (result.code !== 1) { message.error(result.message); return; }
    message.success('存储配置已保存');
    const refreshed = await api.api.getFileStorageConfig();
    if (refreshed.code === 1 && refreshed.data) {
      form.resetFields();
      form.setFieldsValue(refreshed.data);
      setConfig(refreshed.data);
    }
  };

  const content = (
    <div>
      <Alert type="info" showIcon title="切换默认存储只影响新上传的图片，历史图片继续使用各自的存储配置。" style={{ marginBottom: 24 }} />
      {!loading && !config && <Button onClick={() => setVersion((current) => current + 1)}>重新加载配置</Button>}
      <Spin spinning={loading}>
        <Form
          form={form}
          labelCol={{ flex: '150px' }}
          wrapperCol={{ style: { maxWidth: 540 } }}
          onFinish={save}
          style={{ maxWidth: 800 }}
        >
          <Form.Item name="default_storage" label="默认存储方式" rules={[{ required: true }]}>
            <Select options={[{ value: 'local', label: '本地硬盘' }, ...providers.map((provider) => ({ value: provider.key, label: provider.label }))]} />
          </Form.Item>
          <Tabs items={[
            { key: 'local', label: '本地硬盘', forceRender: true, children: <>
              <Form.Item name={['local', 'path']} label="本地存储路径" extra="相对路径基于后端工作目录，默认 uploads；迁移服务器时需一并迁移图片文件。" rules={[{ required: true, whitespace: true }]}><Input placeholder="uploads 或绝对目录" /></Form.Item>
              <Form.Item name={['local', 'domain']} label="访问域名" extra="留空使用同站 /media；自定义域名需要包含图片访问前缀，例如 https://img.example.com/media。"><Input placeholder="https://img.example.com/media" /></Form.Item>
            </> },
            ...providers.map((provider) => ({ key: provider.key, label: provider.label, forceRender: true, children: <>
              <Form.Item name={[provider.key, 'endpoint']} label="Endpoint" rules={[{ required: defaultStorage === provider.key && provider.key === 'oss' }]}><Input placeholder={provider.endpoint} /></Form.Item>
              <Form.Item name={[provider.key, 'region']} label="Region" rules={[{ required: defaultStorage === provider.key && provider.key !== 'oss' }]}><Input placeholder={provider.key === 's3' ? 'us-east-1' : provider.key === 'cos' ? 'ap-guangzhou' : 'cn-hangzhou（可选）'} /></Form.Item>
              <Form.Item name={[provider.key, 'bucket']} label="Bucket" rules={[{ required: defaultStorage === provider.key }]}><Input placeholder={provider.key === 'cos' ? 'bucket-appid' : '存储桶名称'} /></Form.Item>
              <Form.Item name={[provider.key, 'access_key_id']} label={provider.key === 'cos' ? 'Secret ID' : 'Access Key ID'} rules={[{ required: defaultStorage === provider.key }]}><Input autoComplete="off" /></Form.Item>
              <Form.Item name={[provider.key, 'access_key_secret']} label={provider.key === 'cos' ? 'Secret Key' : 'Access Key Secret'} extra="密钥不回显，留空保留已保存的密钥。" rules={[{ required: defaultStorage === provider.key && !config?.[provider.key].secret_set }]}><Input.Password autoComplete="new-password" placeholder={config?.[provider.key].secret_set ? '已配置，留空不修改' : '请输入密钥'} /></Form.Item>
              <Form.Item name={[provider.key, 'domain']} label="访问域名 / CDN" extra="配置时直接拼接相对路径；留空则生成有效期 1 小时的签名地址，支持私有桶。"><Input placeholder="https://img.example.com" /></Form.Item>
              {provider.key === 's3' && <Form.Item name={['s3', 'use_path_style']} label="兼容模式" valuePropName="checked"><Checkbox>使用 Path Style（MinIO 等兼容存储）</Checkbox></Form.Item>}
            </> })),
          ]} />
          <Form.Item label=" " colon={false} style={{ marginTop: 24 }}>
            <Button type="primary" htmlType="submit" loading={saving} disabled={!config}>保存配置</Button>
          </Form.Item>
        </Form>
      </Spin>
    </div>
  );

  return card ? <Card title="图片存储配置">{content}</Card> : content;
}
