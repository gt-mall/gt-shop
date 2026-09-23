import { useEffect, useRef, useState } from 'react';
import { Alert, Button, Divider, Form, Input, message, Modal, Space, Spin } from 'antd';
import { UploadOutlined, ThunderboltOutlined } from '@ant-design/icons';
import { api, type MpConfig } from '@/framework/http';
import { usePermission } from '@/framework/store';
import { ImagePicker } from '@/modules/system/file';
import WechatPaySetting from './wechat-pay';

export default function MpSettingPage() {
  const [form] = Form.useForm<MpConfig>();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [quickDomainModalVisible, setQuickDomainModalVisible] = useState(false);
  const [quickDomainValue, setQuickDomainValue] = useState('');
  const fileInputRef = useRef<HTMLInputElement>(null);

  const hasPermission = usePermission();
  const canUpdate = hasPermission('system:setting:update');

  useEffect(() => {
    let active = true;
    const load = async () => {
      setLoading(true);
      const result = await api.api.getSystemMpSetting();
      if (!active) return;
      setLoading(false);
      if (result.code !== 1 || !result.data) {
        message.error(result.message || '获取小程序设置失败');
        return;
      }
      form.setFieldsValue(result.data);
    };
    void load();
    return () => {
      active = false;
    };
  }, [form]);

  const save = async (values: MpConfig) => {
    setSaving(true);
    const result = await api.api.saveSystemMpSetting(values);
    setSaving(false);
    if (result.code !== 1) {
      message.error(result.message || '保存小程序设置失败');
      return;
    }
    message.success('小程序设置已保存');
    const refreshed = await api.api.getSystemMpSetting();
    if (refreshed.code === 1 && refreshed.data) {
      form.setFieldsValue(refreshed.data);
    }
  };

  const handleKeyFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = (ev) => {
      const content = ev.target?.result as string;
      if (content) {
        form.setFieldValue('upload_private_key', content.trim());
        message.success(`已从文件 ${file.name} 导入密钥内容`);
      }
    };
    reader.readAsText(file);
    e.target.value = '';
  };

  const handleQuickFillDomains = () => {
    let host = quickDomainValue.trim();
    if (!host) {
      message.warning('请输入主域名');
      return;
    }
    host = host.replace(/^https?:\/\//i, '').replace(/\/+$/, '');
    form.setFieldsValue({
      request_domain: `https://${host}`,
      socket_domain: `wss://${host}`,
      upload_domain: `https://${host}`,
      download_domain: `https://${host}`,
    });
    setQuickDomainModalVisible(false);
    setQuickDomainValue('');
    message.success('已自动填充服务器域名');
  };

  return (
    <Spin spinning={loading}>
      <Form
        form={form}
        labelCol={{ flex: '160px' }}
        wrapperCol={{ style: { maxWidth: 580 } }}
        onFinish={save}
        style={{ maxWidth: 840 }}
      >
        <Divider titlePlacement="start" plain>
          基础信息
        </Divider>

        <Form.Item
          name="name"
          label="小程序名称"
          rules={[{ required: true, whitespace: true, message: '请输入小程序名称' }]}
          extra="例如：Mall 官方商城，用于后台识别与展示"
        >
          <Input placeholder="请输入小程序名称" maxLength={64} showCount />
        </Form.Item>

        <Form.Item
          name="original_id"
          label="原始 ID"
          extra="可在微信公众平台【设置 - 基本设置 - 账号信息】中查看，通常以 gh_ 开头"
        >
          <Input placeholder="例如：gh_123456789abc" maxLength={32} />
        </Form.Item>

        <Form.Item
          name="qrcode_id"
          label="小程序码"
          extra="可在微信公众平台下载小程序码并上传，便于后台与客户扫码访问"
        >
          <ImagePicker />
        </Form.Item>

        <Form.Item
          name="app_id"
          label="AppID"
          extra="可在微信公众平台【开发 - 开发管理 - 开发设置 - 开发者ID】中查看"
        >
          <Input placeholder="例如：wx0123456789abcdef" maxLength={64} />
        </Form.Item>

        <Form.Item
          name="app_secret"
          label="AppSecret"
          extra="小程序开发者密码（AppSecret），用于服务端接口调用凭证 access_token 获取"
        >
          <Input.Password placeholder="留空保持已有 AppSecret" maxLength={64} />
        </Form.Item>

        <Divider titlePlacement="start" plain>
          <Space>
            <span>服务器域名</span>
            <Button
              type="link"
              size="small"
              icon={<ThunderboltOutlined />}
              onClick={() => setQuickDomainModalVisible(true)}
              style={{ padding: 0 }}
            >
              一键快速填充域名
            </Button>
          </Space>
        </Divider>

        <Alert
          type="info"
          showIcon
          style={{ marginBottom: 20 }}
          message="微信小程序必须配置合法域名，所有域名必须走 HTTPS 或 WSS 协议。可在微信公众平台【开发 - 开发管理 - 开发设置 - 服务器域名】中同步登记。"
        />

        <Form.Item
          name="request_domain"
          label="request 合法域名"
          extra="网络请求 API 域名，必须使用 https:// 协议，多个域名请使用分号 (;) 或换行分隔"
        >
          <Input placeholder="例如：https://api.yourdomain.com" maxLength={500} />
        </Form.Item>

        <Form.Item
          name="socket_domain"
          label="socket 合法域名"
          extra="WebSocket 通信域名，必须使用 wss:// 协议，多个域名请使用分号 (;) 或换行分隔"
        >
          <Input placeholder="例如：wss://api.yourdomain.com" maxLength={500} />
        </Form.Item>

        <Form.Item
          name="upload_domain"
          label="uploadFile 合法域名"
          extra="文件上传服务域名，必须使用 https:// 协议，多个域名请使用分号 (;) 或换行分隔"
        >
          <Input placeholder="例如：https://api.yourdomain.com" maxLength={500} />
        </Form.Item>

        <Form.Item
          name="download_domain"
          label="downloadFile 合法域名"
          extra="文件下载与图片资源域名，必须使用 https:// 协议，多个域名请使用分号 (;) 或换行分隔"
        >
          <Input placeholder="例如：https://api.yourdomain.com;https://res.yourdomain.com" maxLength={500} />
        </Form.Item>

        <Form.Item
          name="udp_domain"
          label="udp 合法域名"
          extra="UDP 通信域名（选填），多个域名请使用分号 (;) 或换行分隔"
        >
          <Input placeholder="例如：udp://127.0.0.1" maxLength={500} />
        </Form.Item>

        <Form.Item
          name="tcp_domain"
          label="tcp 合法域名"
          extra="TCP 通信域名（选填），多个域名请使用分号 (;) 或换行分隔"
        >
          <Input placeholder="例如：tcp://127.0.0.1" maxLength={500} />
        </Form.Item>

        <Divider titlePlacement="start" plain>
          业务域名
        </Divider>

        <Form.Item
          name="business_domain"
          label="业务域名"
          extra="小程序 web-view 嵌入 H5 页面所需域名，必须使用 https:// 协议。配置后需在微信公众平台配置并上传校验文件"
        >
          <Input.TextArea
            rows={2}
            placeholder="例如：https://m.yourdomain.com"
            maxLength={500}
            showCount
          />
        </Form.Item>

        <Divider titlePlacement="start" plain>
          代码上传设置
        </Divider>

        <Form.Item
          name="upload_private_key"
          label="代码上传密钥"
          extra={
            <div>
              <div>微信公众平台【开发 - 开发管理 - 开发设置 - 小程序代码上传】生成的 private.key 文件内容。用于 miniprogram-ci 自动化上传与免密发布。</div>
              <div style={{ marginTop: 8 }}>
                <input
                  type="file"
                  ref={fileInputRef}
                  accept=".key,.txt,.pem"
                  style={{ display: 'none' }}
                  onChange={handleKeyFileChange}
                />
                <Button
                  size="small"
                  icon={<UploadOutlined />}
                  onClick={() => fileInputRef.current?.click()}
                >
                  从 .key / .pem 文件导入密钥
                </Button>
              </div>
            </div>
          }
        >
          <Input.TextArea
            rows={5}
            placeholder="留空保持已有上传密钥"
            style={{ fontFamily: 'monospace', fontSize: '12px' }}
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
              保存小程序设置
            </Button>
          </Space>
        </Form.Item>
      </Form>

      <WechatPaySetting />

      <Modal
        title="快速填充服务器域名"
        open={quickDomainModalVisible}
        onOk={handleQuickFillDomains}
        onCancel={() => {
          setQuickDomainModalVisible(false);
          setQuickDomainValue('');
        }}
        okText="一键填充"
        cancelText="取消"
      >
        <p style={{ color: '#6b7280', fontSize: 13, marginBottom: 12 }}>
          输入您的后端 API 域名，将自动为 request、socket、uploadFile、downloadFile 填充对应协议的域名：
        </p>
        <Input
          placeholder="例如：api.yourmall.com"
          value={quickDomainValue}
          onChange={(e) => setQuickDomainValue(e.target.value)}
          onPressEnter={handleQuickFillDomains}
        />
      </Modal>
    </Spin>
  );
}
