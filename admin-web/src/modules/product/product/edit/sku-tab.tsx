import { Button, Divider, Radio, Space, Typography } from 'antd';

import type { ProductSKURequest, ProductSpec } from '@/framework/http';
import { SKUTable } from '../sku-table';
import { SpecEditor } from '../spec-editor';

export interface SKUTabProps {
  specMode: 'single' | 'multi';
  hasSavedSKUs: boolean;
  onSpecModeChange: (mode: 'single' | 'multi') => void;
  specs: ProductSpec[];
  onSpecsChange: (specs: ProductSpec[]) => void;
  skus: ProductSKURequest[];
  onSKUsChange: (skus: ProductSKURequest[]) => void;
  disabled: boolean;
  onPrev: () => void;
  onNext: () => void;
}

export function SKUTab({
  specMode,
  hasSavedSKUs,
  onSpecModeChange,
  specs,
  onSpecsChange,
  skus,
  onSKUsChange,
  disabled,
  onPrev,
  onNext,
}: SKUTabProps) {
  return (
    <div  className=' px-4 py-4 '>
      <div
        className="product-edit-spec-mode"
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          marginBottom: 20,
          padding: '12px 16px',
          background: '#F9FAFB',
          borderRadius: 8,
          border: '1px solid #F3F4F6',
        }}
      >
        <Space align="center">
          <Typography.Text strong style={{ fontSize: 14 }}>
            规格模式：
          </Typography.Text>
          <Radio.Group
            value={specMode}
            disabled={disabled || hasSavedSKUs}
            onChange={(e) => onSpecModeChange(e.target.value)}
            buttonStyle="solid"
          >
            <Radio.Button value="single">单规格（普通商品）</Radio.Button>
            <Radio.Button value="multi">多规格（如颜色、尺码等）</Radio.Button>
          </Radio.Group>
        </Space>
      </div>

      {specMode === 'multi' ? (
        <div>
          <SpecEditor
            specs={specs}
            skus={skus}
            disabled={disabled}
            onSpecsChange={onSpecsChange}
            onGenerateSKUs={onSKUsChange}
          />
          <Divider style={{ margin: '16px 0' }}>SKU 价格与编码明细</Divider>
          <SKUTable
            specs={specs}
            skus={skus}
            disabled={disabled}
            onSKUsChange={onSKUsChange}
          />
        </div>
      ) : (
        <div>
          <SKUTable
            specs={[]}
            skus={skus}
            disabled={disabled}
            onSKUsChange={onSKUsChange}
          />
        </div>
      )}

      <Divider style={{ margin: '24px 0 16px' }} />
      <div style={{ display: 'flex', justifyContent: 'space-between' }}>
        <Button onClick={onPrev}>上一步：基本信息</Button>
        <Button type="primary" onClick={onNext}>
          下一步：配置商品参数
        </Button>
      </div>
    </div>
  );
}
