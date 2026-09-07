import { useEffect, useState } from 'react';
import { Form, Input, Modal, Radio, Space, Typography } from 'antd';
import { VideoCameraOutlined } from '@ant-design/icons';

type VideoModalProps = {
  open: boolean;
  initialUrl?: string;
  initialWidth?: string;
  initialAlignment?: string;
  onConfirm: (values: { src: string; width: string; alignment: string }) => void;
  onCancel: () => void;
};

export function VideoModal({
  open,
  initialUrl = '',
  initialWidth = '100%',
  initialAlignment = 'center',
  onConfirm,
  onCancel,
}: VideoModalProps) {
  const [form] = Form.useForm();
  const [width, setWidth] = useState(initialWidth);
  const [alignment, setAlignment] = useState(initialAlignment);

  useEffect(() => {
    if (open) {
      form.setFieldsValue({
        url: initialUrl,
      });
      setWidth(initialWidth || '100%');
      setAlignment(initialAlignment || 'center');
    }
  }, [open, initialUrl, initialWidth, initialAlignment, form]);

  const handleOk = () => {
    form
      .validateFields()
      .then((values) => {
        let rawUrl = (values.url || '').trim();
        // 如果用户粘贴了整个 <iframe ... src="..."> 标签，提取出 src
        if (rawUrl.startsWith('<iframe') && rawUrl.includes('src=')) {
          const match = rawUrl.match(/src=["']([^"']+)["']/);
          if (match) {
            rawUrl = match[1];
          }
        }
        onConfirm({ src: rawUrl, width, alignment });
        form.resetFields();
      })
      .catch(() => {});
  };

  const isEditing = Boolean(initialUrl);

  return (
    <Modal
      title={
        <Space>
          <VideoCameraOutlined style={{ color: 'var(--ant-color-primary, #4338CA)' }} />
          <span>{isEditing ? '修改视频设置' : '插入视频 (支持 MP4 直链 / B站 / YouTube)'}</span>
        </Space>
      }
      open={open}
      onOk={handleOk}
      onCancel={() => {
        form.resetFields();
        onCancel();
      }}
      okText={isEditing ? '保存修改' : '插入'}
      cancelText="取消"
      destroyOnClose
    >
      <Form form={form} layout="vertical">
        <Form.Item
          name="url"
          label="视频地址 / 嵌入链接"
          rules={[
            { required: true, message: '请输入视频 URL' },
            {
              validator: (_, value) => {
                if (!value) return Promise.resolve();
                const v = value.trim();
                if (
                  v.startsWith('http://') ||
                  v.startsWith('https://') ||
                  v.startsWith('//') ||
                  v.startsWith('<iframe')
                ) {
                  return Promise.resolve();
                }
                return Promise.reject(new Error('请输入合法的 http/https 链接或 iframe 代码'));
              },
            },
          ]}
          extra={
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              示例：MP4 直链 (https://...mp4)、B站视频 (https://www.bilibili.com/video/BV...) 或
              YouTube 链接
            </Typography.Text>
          }
        >
          <Input.TextArea
            rows={3}
            placeholder="支持粘贴 MP4 视频直链、B站视频链接、YouTube 链接或 <iframe> 嵌入代码"
          />
        </Form.Item>

        <Form.Item label="显示宽度">
          <Radio.Group value={width} onChange={(e) => setWidth(e.target.value)}>
            <Radio.Button value="100%">100% (全宽)</Radio.Button>
            <Radio.Button value="80%">80%</Radio.Button>
            <Radio.Button value="60%">60%</Radio.Button>
          </Radio.Group>
        </Form.Item>

        <Form.Item label="对齐方式">
          <Radio.Group value={alignment} onChange={(e) => setAlignment(e.target.value)}>
            <Radio.Button value="left">居左</Radio.Button>
            <Radio.Button value="center">居中</Radio.Button>
            <Radio.Button value="right">居右</Radio.Button>
          </Radio.Group>
        </Form.Item>
      </Form>
    </Modal>
  );
}
