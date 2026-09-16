import { Button, Divider, Form } from 'antd';
import { SaveOutlined } from '@ant-design/icons';

import { ProductDetailEditor } from '../detail-editor';

export interface DetailTabProps {
  detailHTML: string;
  onChange: (html: string) => void;
  disabled: boolean;
  saving: boolean;
  productName: string;
  productPrice: string;
  coverUrl?: string;
  onPrev: () => void;
  onSave: () => void;
}

export function DetailTab({
  detailHTML,
  onChange,
  disabled,
  saving,
  productName,
  productPrice,
  coverUrl,
  onPrev,
  onSave,
}: DetailTabProps) {
  return (
    <div  className=' px-4 py-4'>
      <Form.Item
        label="商品图文详情"
        layout="vertical"
        labelCol={{ span: 24 }}
        wrapperCol={{ span: 24, style: { maxWidth: '100%' } }}
      >
        <ProductDetailEditor
          value={detailHTML}
          onChange={onChange}
          disabled={disabled}
          productName={productName}
          productPrice={productPrice}
          coverUrl={coverUrl}
        />
      </Form.Item>

      <Divider style={{ margin: '24px 0 16px' }} />
      <div style={{ display: 'flex', justifyContent: 'space-between' }}>
        <Button onClick={onPrev}>上一步：配置商品参数</Button>
        {!disabled && (
          <Button
            type="primary"
            icon={<SaveOutlined />}
            loading={saving}
            onClick={onSave}
          >
            保存商品
          </Button>
        )}
      </div>
    </div>
  );
}
