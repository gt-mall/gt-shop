import {
  Button,
  Card,
  Divider,
  Form,
  Input,
  InputNumber,
  Radio,
  Select,
  TreeSelect,
  type FormInstance,
} from 'antd';

import type { FreightTemplateOption, ProductDetail } from '@/framework/http';
import { ImagePicker } from '@/shared/components';
import { formatRegionLabel } from '../../freight/region-cascader';
import type { ProductFormData } from './types';

export interface BasicTabProps {
  form: FormInstance<ProductFormData>;
  categoryTreeData: any[];
  freightTemplateOptions: FreightTemplateOption[];
  galleryFileIDs: number[];
  onGalleryChange: (ids: number[]) => void;
  watchedFreightType: number;
  watchedFreightTemplateID?: number;
  isView: boolean;
  isEdit: boolean;
  initialDetail: ProductDetail | null;
  onNext: () => void;
}

export function BasicTab({
  form,
  categoryTreeData,
  freightTemplateOptions,
  galleryFileIDs,
  onGalleryChange,
  watchedFreightType,
  watchedFreightTemplateID,
  isView,
  isEdit,
  initialDetail,
  onNext,
}: BasicTabProps) {
  return (
    <div className="px-4 py-4">
      <Form.Item
        name="name"
        label="商品名称"
        rules={[
          { required: true, message: '请输入商品名称' },
          { max: 255, message: '商品名称最多255个字符' },
        ]}
      >
        <Input placeholder="请输入商品名称，如：2026年夏季纯棉短袖T恤" maxLength={255} showCount />
      </Form.Item>

      <Form.Item name="subtitle" label="商品副标题 / 卖点">
        <Input placeholder="请输入商品卖点，展示在商品名称下方" maxLength={255} showCount />
      </Form.Item>

      <Form.Item name="category_id" label="商品分类">
        <TreeSelect
          placeholder="请选择分类"
          allowClear
          treeData={categoryTreeData}
          treeDefaultExpandAll
          showSearch
          treeNodeFilterProp="title"
          style={{ width: '100%' }}
        />
      </Form.Item>

      <Form.Item name="main_image_id" label="商品主图">
        <ImagePicker multiple={false} disabled={isView} />
      </Form.Item>

      <Form.Item
        label="商品轮播图"
        wrapperCol={{ style: { maxWidth: 680 } }}
        extra="支持鼠标拖拽图片卡片调整顺序，第一张将作为商品默认主展示图"
      >
        <ImagePicker
          multiple={true}
          maxCount={10}
          disabled={isView}
          value={galleryFileIDs}
          onChange={onGalleryChange}
        />
      </Form.Item>

      <Card
        className="product-edit-delivery-card"
        size="small"
        title="配送设置"
        style={{ marginBottom: 24, background: '#FAFAFA', borderRadius: 8 }}
      >
        <Form.Item
          name="freight_type"
          label="配送方式"
          rules={[{ required: true, message: '请选择配送方式' }]}
        >
          <Radio.Group
            onChange={(e) => {
              const val = e.target.value;
              if (val === 1) {
                form.setFieldsValue({
                  freight_fee: '0.00',
                  freight_template_id: undefined,
                });
              } else if (val === 2) {
                form.setFieldsValue({
                  freight_template_id: undefined,
                });
              } else if (val === 3) {
                form.setFieldsValue({
                  freight_fee: '0.00',
                });
              }
            }}
          >
            <Radio value={1}>包邮</Radio>
            <Radio value={2}>固定运费</Radio>
            <Radio value={3}>运费模板</Radio>
          </Radio.Group>
        </Form.Item>

        {watchedFreightType === 2 && (
          <Form.Item
            name="freight_fee"
            label="固定运费"
            rules={[{ required: true, message: '请输入固定运费金额' }]}
          >
            <InputNumber
              prefix="¥"
              min={0}
              precision={2}
              style={{ width: 180 }}
              placeholder="0.00"
            />
          </Form.Item>
        )}

        {watchedFreightType === 3 && (
          <>
            <Form.Item
              name="freight_template_id"
              label="运费模板"
              rules={[{ required: true, message: '请选择运费模板' }]}
            >
              <Select
                placeholder="请选择运费模板"
                style={{ width: 320 }}
                options={freightTemplateOptions.map((tmpl) => ({
                  value: tmpl.id,
                  label: tmpl.name,
                }))}
              />
            </Form.Item>

            {(() => {
              const selectedTmpl = freightTemplateOptions.find(
                (t) => t.id === watchedFreightTemplateID,
              );
              if (!selectedTmpl) return null;
              const isFree =
                !selectedTmpl.base_fee ||
                selectedTmpl.base_fee === '0' ||
                selectedTmpl.base_fee === '0.00';
              const extraFees = selectedTmpl.extra_fees || [];
              const noDeliveries = selectedTmpl.no_delivery || [];

              const extraText =
                extraFees.length === 0
                  ? '无'
                  : extraFees
                      .slice(0, 3)
                      .map(
                        (ef) =>
                          `${formatRegionLabel(ef.province_code || '', ef.city_code || '')} +¥${ef.fee}`,
                      )
                      .join(' / ') +
                    (extraFees.length > 3 ? ` 等${extraFees.length}个地区` : '');

              const noDelText =
                noDeliveries.length === 0
                  ? '无'
                  : noDeliveries
                      .slice(0, 3)
                      .map((nd) => formatRegionLabel(nd.province_code || '', nd.city_code || ''))
                      .join(' / ') +
                    (noDeliveries.length > 3 ? ` 等${noDeliveries.length}个地区` : '');

              return (
                <div
                  className="product-edit-freight-summary"
                  style={{
                    marginBottom: 16,
                    padding: 12,
                    background: '#FFF',
                    border: '1px solid #E5E7EB',
                    borderRadius: 6,
                    fontSize: 13,
                    maxWidth: 500,
                  }}
                >
                  <div style={{ color: '#4B5563', marginBottom: 4 }}>
                    <strong>默认：</strong>
                    {isFree ? '全国包邮' : `¥${selectedTmpl.base_fee}`}
                  </div>
                  <div style={{ color: '#4B5563', marginBottom: 4 }}>
                    <strong>地区附加：</strong>
                    {extraText}
                  </div>
                  <div style={{ color: '#4B5563', marginBottom: 6 }}>
                    <strong>不配送：</strong>
                    {noDelText}
                  </div>
                  <div>
                    <Button
                      type="link"
                      size="small"
                      style={{ padding: 0 }}
                      onClick={() =>
                        window.open(`/product/freight/${selectedTmpl.id}/edit`, '_blank')
                      }
                    >
                      查看模板详情 ↗
                    </Button>
                  </div>
                </div>
              );
            })()}
          </>
        )}
      </Card>

      <Form.Item name="status" label="上下架状态" rules={[{ required: true }]}>
        <Radio.Group>
          <Radio value={0}>暂存下架</Radio>
          <Radio value={1}>立即上架</Radio>
        </Radio.Group>
      </Form.Item>

      <Form.Item name="sort" label="列表排序值">
        <InputNumber min={0} max={99999} style={{ width: 200 }} placeholder="0" />
      </Form.Item>

      <Form.Item name="virtual_sales" label="虚拟销量">
        <InputNumber min={0} max={9999999} style={{ width: 200 }} placeholder="0" />
      </Form.Item>

      {isEdit && initialDetail && (
        <Form.Item label="实际真实销量">
          <Input value={initialDetail.sales_count} disabled style={{ width: 200 }} />
        </Form.Item>
      )}

      <Divider style={{ margin: '24px 0 16px' }} />
      <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
        <Button type="primary" onClick={onNext}>
          下一步：配置规格与 SKU
        </Button>
      </div>
    </div>
  );
}
