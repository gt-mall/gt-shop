import { useState } from 'react';
import { Alert, Button, Card, Form, Space, Typography } from 'antd';
import { ImagePicker } from '@/modules/system/file';

type DemoForm = { cover_file_id?: number; gallery_file_ids?: number[] };
const example = `import { Form } from 'antd';
import { ImagePicker } from '@/modules/system/file';

// Form.Item 自动绑定 value 和 onChange。
<Form.Item name="cover_file_id" label="商品封面">
  <ImagePicker />
</Form.Item>
<Form.Item name="gallery_file_ids" label="商品相册">
  <ImagePicker multiple maxCount={6} />
</Form.Item>

// 也可以在组件中使用受控状态：
const [fileId, setFileId] = useState<number>();
<ImagePicker value={fileId} onChange={setFileId} />`;

export default function ImageDemoPage() {
  const [form] = Form.useForm<DemoForm>();
  const [submitted, setSubmitted] = useState<DemoForm>();
  return (
    <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
      <Card title="图片选择组件 Demo">
        <Alert type="info" showIcon title="选择器支持上传、分组、搜索、预览；上传会写入真实素材库，下面的表单只演示 file_id 提交。" style={{ marginBottom: 24 }} />
        <Form form={form} layout="vertical" onFinish={setSubmitted} initialValues={{ gallery_file_ids: [] }}>
          <Form.Item name="cover_file_id" label="商品封面（单选）"><ImagePicker /></Form.Item>
          <Form.Item name="gallery_file_ids" label="商品相册（最多 6 张）"><ImagePicker multiple maxCount={6} /></Form.Item>
          <Space>
            <Button type="primary" htmlType="submit">查看提交数据</Button>
            <Button onClick={() => { form.resetFields(); setSubmitted(undefined); }}>重置</Button>
          </Space>
        </Form>
        {submitted && <Typography.Paragraph style={{ marginTop: 20 }}><pre>{JSON.stringify(submitted, null, 2)}</pre></Typography.Paragraph>}
      </Card>
      <Card title="调用方法">
        <Typography.Paragraph>业务数据只保存 ID。组件会根据 ID 获取当前图片信息和访问地址，无需拼接 URL。</Typography.Paragraph>
        <Typography.Paragraph copyable={{ text: example }}><pre style={{ overflowX: 'auto' }}>{example}</pre></Typography.Paragraph>
      </Card>
    </Space>
  );
}
