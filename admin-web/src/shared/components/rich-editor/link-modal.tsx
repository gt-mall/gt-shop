import { useEffect } from 'react';
import { Form, Input, Modal } from 'antd';

type Props = {
  open: boolean;
  initialHref?: string;
  onConfirm: (href: string) => void;
  onCancel: () => void;
};

export function LinkModal({ open, initialHref = '', onConfirm, onCancel }: Props) {
  const [form] = Form.useForm<{ href: string }>();

  useEffect(() => {
    if (open) {
      form.setFieldsValue({ href: initialHref });
    }
  }, [open, initialHref, form]);

  const handleOk = async () => {
    const values = await form.validateFields();
    let href = values.href.trim();
    if (!href.startsWith('http://') && !href.startsWith('https://') && !href.startsWith('//')) {
      href = `https://${href}`;
    }
    onConfirm(href);
  };

  return (
    <Modal
      title="插入 / 编辑链接"
      open={open}
      onOk={handleOk}
      onCancel={onCancel}
      okText="确定"
      cancelText="取消"
      destroyOnClose
      width={440}
    >
      <Form form={form} layout="vertical" initialValues={{ href: initialHref }}>
        <Form.Item
          name="href"
          label="链接地址 (URL)"
          rules={[{ required: true, message: '请输入跳转网址' }]}
        >
          <Input placeholder="https://example.com" autoFocus />
        </Form.Item>
      </Form>
    </Modal>
  );
}
