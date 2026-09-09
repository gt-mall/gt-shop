import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Button,
  Card,
  Col,
  Divider,
  Form,
  Input,
  InputNumber,
  message,
  Radio,
  Row,
  Select,
  Space,
  Spin,
  Tabs,
  Tag,
  Tooltip,
  TreeSelect,
  Typography,
} from 'antd';
import {
  ArrowDownOutlined,
  ArrowLeftOutlined,
  ArrowUpOutlined,
  CheckOutlined,
  DeleteOutlined,
  EyeOutlined,
  FileImageOutlined,
  ProfileOutlined,
  SaveOutlined,
  ShopOutlined,
  UnorderedListOutlined,
} from '@ant-design/icons';
import { useLocation, useNavigate, useParams, useSearchParams } from 'react-router-dom';

import { api } from '@/framework/http';
import type {
  ProductCategoryOption,
  ProductDetail,
  ProductParam,
  ProductSKURequest,
  ProductSpec,
} from '@/framework/http';
import { usePermission } from '@/framework/store';
import { ImagePicker } from '@/modules/system/file/picker';
import { RichEditor } from '@/shared/components';
import { buildCategoryOptionTree, normalizeMoney } from '../types';
import { ParamEditor } from './param-editor';
import { SKUTable } from './sku-table';
import { SpecEditor } from './spec-editor';

type ProductFormData = {
  name: string;
  subtitle?: string;
  category_id?: number;
  main_image_id?: number | null;
  sort?: number;
  status: number;
  virtual_sales?: number;
  detail?: string;
};

export default function ProductEditPage() {
  const { id } = useParams<{ id?: string }>();
  const navigate = useNavigate();
  const location = useLocation();
  const hasPermission = usePermission();
  const canViewInventory = hasPermission('product:inventory:list');

  const isCreate = !id;
  const isView = location.pathname.endsWith(`/${id}`); // /product/product/:id without /edit
  const isEdit = !isCreate && !isView;

  const [searchParams, setSearchParams] = useSearchParams();
  const rawTab = searchParams.get('tab');
  const [activeTab, setActiveTab] = useState<'basic' | 'sku' | 'params' | 'media'>(() => {
    if (rawTab === 'sku' || rawTab === 'params' || rawTab === 'media') return rawTab;
    return 'basic';
  });

  const handleTabChange = useCallback(
    (key: string) => {
      const nextTab = (key === 'sku' || key === 'params' || key === 'media' ? key : 'basic') as
        | 'basic'
        | 'sku'
        | 'params'
        | 'media';
      setActiveTab(nextTab);
      setSearchParams(
        (prev) => {
          const next = new URLSearchParams(prev);
          if (nextTab === 'basic') {
            next.delete('tab');
          } else {
            next.set('tab', nextTab);
          }
          return next;
        },
        { replace: true },
      );
    },
    [setSearchParams],
  );

  useEffect(() => {
    const currentTab = searchParams.get('tab');
    if (currentTab === 'sku' || currentTab === 'params' || currentTab === 'media') {
      setActiveTab(currentTab);
    } else {
      setActiveTab('basic');
    }
  }, [searchParams]);

  const [form] = Form.useForm<ProductFormData>();
  const [loading, setLoading] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  // Category options
  const [categoryOptions, setCategoryOptions] = useState<ProductCategoryOption[]>([]);

  // Product parameters
  const [params, setParams] = useState<ProductParam[]>([]);

  // Product multi-media & rich text
  const [galleryFileIDs, setGalleryFileIDs] = useState<number[]>([]);
  const [detailHTML, setDetailHTML] = useState<string>('');

  // Spec mode: 'single' (specs=[]) vs 'multi' (specs=[...])
  const [specMode, setSpecMode] = useState<'single' | 'multi'>('single');
  const [specs, setSpecs] = useState<ProductSpec[]>([]);
  const [skus, setSKUs] = useState<ProductSKURequest[]>([
    {
      id: 0,
      spec_values: [],
      price: '0.00',
      market_price: '0.00',
      cost_price: '0.00',
      sort: 0,
      status: 1,
    },
  ]);

  // Record initial detail to check if switching modes is safe
  const [initialDetail, setInitialDetail] = useState<ProductDetail | null>(null);
  const hasSavedSKUs = (initialDetail?.skus ?? []).some((sku) => (sku.id ?? 0) > 0);

  // Load category options
  const loadCategories = useCallback(async () => {
    const res = await api.api.getProductCategoryOptions();
    if (res.code === 1) {
      setCategoryOptions(res.data ?? []);
    }
  }, []);

  // Load product detail for edit/view
  const loadDetail = useCallback(async (productID: number) => {
    setLoading(true);
    setLoadError(null);

    const res = await api.api.getProductDetail({ id: productID });
    setLoading(false);

    if (res.code === 1 && res.data) {
      const data = res.data;
      setInitialDetail(data);

      form.setFieldsValue({
        name: data.name ?? '',
        subtitle: data.subtitle ?? '',
        category_id: data.category_id && data.category_id > 0 ? data.category_id : undefined,
        main_image_id: data.main_image_id ?? undefined,
        sort: data.sort ?? 0,
        status: data.status ?? 1,
        virtual_sales: data.virtual_sales ?? 0,
        detail: data.detail || '',
      });

      setGalleryFileIDs(data.gallery_file_ids || []);
      setDetailHTML(data.detail || '');
      setParams(data.params || []);

      const loadedSpecs = data.specs || [];
      setSpecs(loadedSpecs);

      const loadedSKUs: ProductSKURequest[] = (data.skus || []).map((s) => ({
        id: s.id,
        spec_values: s.spec_values || [],
        sku_code: s.sku_code ?? undefined,
        barcode: s.barcode ?? undefined,
        image_id: s.image_id ?? undefined,
        price: normalizeMoney(s.price),
        market_price: normalizeMoney(s.market_price),
        cost_price: normalizeMoney(s.cost_price),
        unit: s.unit ?? undefined,
        weight_gram: s.weight_gram ?? undefined,
        sort: s.sort,
        status: (s.status ?? 1) as 0 | 1,
      }));

      setSKUs(loadedSKUs);
      setSpecMode(loadedSpecs.length > 0 ? 'multi' : 'single');
    } else {
      setLoadError(res.message || '加载商品详情失败');
    }
  }, [form]);

  useEffect(() => {
    void loadCategories();
    if (id) {
      void loadDetail(Number(id));
    }
  }, [id, loadCategories, loadDetail]);

  // Category TreeSelect Data
  const categoryTreeData = useMemo(() => {
    return buildCategoryOptionTree(categoryOptions);
  }, [categoryOptions]);

  // Gallery order handlers
  const moveGalleryItem = (index: number, direction: 'up' | 'down') => {
    const next = [...galleryFileIDs];
    const targetIdx = direction === 'up' ? index - 1 : index + 1;
    if (targetIdx < 0 || targetIdx >= next.length) return;
    const temp = next[index];
    next[index] = next[targetIdx];
    next[targetIdx] = temp;
    setGalleryFileIDs(next);
  };

  const removeGalleryItem = (index: number) => {
    setGalleryFileIDs(galleryFileIDs.filter((_, i) => i !== index));
  };

  // Spec mode switch check
  const handleSpecModeChange = (mode: 'single' | 'multi') => {
    if (mode === specMode) return;
    if (hasSavedSKUs) {
      message.error('商品已有已保存的 SKU，不能切换规格模式。');
      return;
    }
    setSpecMode(mode);
    if (mode === 'single') {
      setSpecs([]);
      // Reset to 1 default SKU
      const defaultSKU: ProductSKURequest = {
        ...skus[0],
        id: 0,
        spec_values: [],
        price: skus[0]?.price || '0.00',
        market_price: skus[0]?.market_price || '0.00',
        cost_price: skus[0]?.cost_price || '0.00',
        unit: skus[0]?.unit,
        sort: skus[0]?.sort ?? 0,
        status: skus[0]?.status ?? 1,
      };
      setSKUs([defaultSKU]);
    } else {
      if (skus.length === 1 && (!skus[0].spec_values || skus[0].spec_values.length === 0)) {
        setSKUs([]);
      }
    }
  };

  // Handle Form Finish Failed
  const handleFinishFailed = (errorInfo: any) => {
    if (errorInfo?.errorFields?.length > 0) {
      const firstField = String(errorInfo.errorFields[0].name[0]);
      if (['name', 'category_id', 'subtitle', 'status', 'sort', 'virtual_sales'].includes(firstField)) {
        handleTabChange('basic');
      } else if (['main_image_id'].includes(firstField)) {
        handleTabChange('media');
      }
      message.error('请检查表单中未通过校验的必填项');
    }
  };

  // Handle Form Finish
  const handleFinish = async (values: ProductFormData) => {
    // 1. Validate SKUs presence
    if (skus.length === 0) {
      message.error('商品必须包含至少一条有效SKU');
      handleTabChange('sku');
      return;
    }

    if (specMode === 'single' && skus.length !== 1) {
      message.error('无规格商品必须且只能有一条默认SKU');
      handleTabChange('sku');
      return;
    }

    // 2. Online validation if status === 1
    if (values.status === 1) {
      if (!values.category_id || values.category_id <= 0) {
        message.error('商品上架时必须选择有效的商品分类');
        handleTabChange('basic');
        return;
      }
      const selectedCat = categoryOptions.find((c) => c.id === values.category_id);
      if (selectedCat && selectedCat.status === 0) {
        message.error('所选分类已禁用，商品无法上架');
        handleTabChange('basic');
        return;
      }
      if (!values.main_image_id || values.main_image_id <= 0) {
        message.error('商品上架时必须设置有效的主图');
        handleTabChange('media');
        return;
      }
      const hasEnabledSKU = skus.some((s) => s.status === 1);
      if (!hasEnabledSKU) {
        message.error('商品上架时必须包含至少一个启用的SKU');
        handleTabChange('sku');
        return;
      }
      const hasZeroPrice = skus.some(
        (s) => s.status === 1 && (normalizeMoney(s.price) === '0.00' || !s.price),
      );
      if (hasZeroPrice) {
        message.error('商品上架时所有启用的SKU销售价格必须大于 0 元');
        handleTabChange('sku');
        return;
      }
    }

    // 3. Format and clean SKUs
    const formattedSKUs: ProductSKURequest[] = skus.map((s) => ({
      id: s.id && s.id > 0 ? s.id : 0,
      spec_values: s.spec_values || [],
      sku_code: s.sku_code ? s.sku_code.trim() : undefined,
      barcode: s.barcode ? s.barcode.trim() : undefined,
      image_id: s.image_id && s.image_id > 0 ? s.image_id : undefined,
      price: normalizeMoney(s.price),
      market_price: normalizeMoney(s.market_price),
      cost_price: normalizeMoney(s.cost_price),
      unit: s.unit ? s.unit.trim() : undefined,
      weight_gram: typeof s.weight_gram === 'number' ? s.weight_gram : undefined,
      sort: s.sort ?? 0,
      status: s.status as 0 | 1,
    }));

    // 4. Format and clean params
    const validParams: ProductParam[] = [];
    for (const p of params) {
      const name = (p.name ?? '').trim();
      const value = (p.value ?? '').trim();
      if (!name && !value) continue;
      if (!name && value) {
        message.error('商品参数名称不能为空');
        handleTabChange('params');
        return;
      }
      validParams.push({ name, value });
    }

    setSaving(true);
    const mainImgVal = values.main_image_id && values.main_image_id > 0 ? values.main_image_id : undefined;

    if (isEdit && id) {
      const res = await api.api.updateProduct({
        id: Number(id),
        name: values.name.trim(),
        subtitle: (values.subtitle || '').trim(),
        category_id: values.category_id || 0,
        main_image_id: mainImgVal,
        gallery_file_ids: galleryFileIDs,
        specs: specMode === 'multi' ? specs : [],
        params: validParams,
        detail: detailHTML,
        sort: values.sort ?? 0,
        status: values.status as 0 | 1,
        virtual_sales: values.virtual_sales ?? 0,
        skus: formattedSKUs,
      });

      setSaving(false);
      if (res.code === 1) {
        message.success('商品更新成功');
        navigate('/product/product');
      } else {
        message.error(res.message || '更新商品失败');
      }
    } else {
      const res = await api.api.createProduct({
        name: values.name.trim(),
        subtitle: (values.subtitle || '').trim(),
        category_id: values.category_id || 0,
        main_image_id: mainImgVal,
        gallery_file_ids: galleryFileIDs,
        specs: specMode === 'multi' ? specs : [],
        params: validParams,
        detail: detailHTML,
        sort: values.sort ?? 0,
        status: values.status as 0 | 1,
        virtual_sales: values.virtual_sales ?? 0,
        skus: formattedSKUs,
      });

      setSaving(false);
      if (res.code === 1) {
        message.success('商品创建成功');
        navigate('/product/product');
      } else {
        message.error(res.message || '创建商品失败');
      }
    }
  };

  if (loading) {
    return (
      <Card style={{ textAlign: 'center', padding: '80px 0' }}>
        <Spin size="large" tip="正在加载商品信息..." />
      </Card>
    );
  }

  if (loadError) {
    return (
      <Card>
        <Alert
          type="error"
          showIcon
          title="数据加载失败"
          description={loadError}
          action={
            <Button size="small" type="primary" onClick={() => loadDetail(Number(id))}>
              重试
            </Button>
          }
        />
      </Card>
    );
  }

  return (
    <Space orientation="vertical" size="large" style={{ width: '100%', paddingBottom: 60 }}>
      {/* Top action header */}
      <Card size="small">
        <Space style={{ width: '100%', justifyContent: 'space-between' }}>
          <Space align="center">
            <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/product/product')}>
              返回列表
            </Button>
            <Typography.Title level={4} style={{ margin: 0 }}>
              {isCreate ? '新建商品' : isView ? '查看商品详情' : '编辑商品'}
            </Typography.Title>
          </Space>

          <Space>
            {!isCreate && id && canViewInventory && (
              <Button onClick={() => navigate(`/product/inventory?product_id=${id}`)}>
                查看库存
              </Button>
            )}
            {!isView && (
              <>
                <Button onClick={() => navigate('/product/product')}>取消</Button>
                <Button
                  type="primary"
                  icon={<SaveOutlined />}
                  loading={saving}
                  onClick={() => form.submit()}
                >
                  保存商品
                </Button>
              </>
            )}
          </Space>
        </Space>
      </Card>

      <Form<ProductFormData>
        form={form}
        layout="vertical"
        disabled={isView}
        onFinish={handleFinish}
        onFinishFailed={handleFinishFailed}
        initialValues={{
          sort: 0,
          status: 0,
          virtual_sales: 0,
        }}
      >
        <Card>
          <Tabs
            activeKey={activeTab}
            onChange={handleTabChange}
            size="large"
            items={[
              {
                key: 'basic',
                label: (
                  <Space>
                    <ShopOutlined style={{ color: '#4338CA' }} />
                    <span>基本信息</span>
                  </Space>
                ),
                forceRender: true,
                children: (
                  <div style={{ paddingTop: 8 }}>
                    <Row gutter={16}>
                      <Col xs={24} md={16}>
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
                      </Col>

                      <Col xs={24} md={8}>
                        <Form.Item
                          name="category_id"
                          label="商品分类"
                          extra="下架商品可不选，上架必须选择已启用的分类"
                        >
                          <TreeSelect
                            placeholder="请选择分类"
                            allowClear
                            treeData={categoryTreeData}
                            treeDefaultExpandAll
                            showSearch
                            treeNodeFilterProp="title"
                          />
                        </Form.Item>
                      </Col>

                      <Col xs={24} md={16}>
                        <Form.Item name="subtitle" label="商品副标题 / 卖点">
                          <Input placeholder="请输入商品卖点，展示在商品名称下方" maxLength={255} showCount />
                        </Form.Item>
                      </Col>

                      <Col xs={24} md={8}>
                        <Form.Item name="status" label="上下架状态" rules={[{ required: true }]}>
                          <Radio.Group>
                            <Radio value={0}>暂存下架</Radio>
                            <Radio value={1}>立即上架</Radio>
                          </Radio.Group>
                        </Form.Item>
                      </Col>

                      <Col xs={12} md={8}>
                        <Form.Item name="sort" label="列表排序值" extra="数字越小越靠前，默认 0">
                          <InputNumber min={0} max={99999} style={{ width: '100%' }} />
                        </Form.Item>
                      </Col>

                      <Col xs={12} md={8}>
                        <Form.Item name="virtual_sales" label="虚拟销量" extra="仅用于前端展示，默认 0">
                          <InputNumber min={0} max={9999999} style={{ width: '100%' }} />
                        </Form.Item>
                      </Col>

                      {isEdit && initialDetail && (
                        <Col xs={12} md={8}>
                          <Form.Item label="实际真实销量">
                            <Input value={initialDetail.sales_count} disabled />
                          </Form.Item>
                        </Col>
                      )}
                    </Row>

                    <Divider style={{ margin: '24px 0 16px' }} />
                    <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
                      <Button type="primary" onClick={() => handleTabChange('sku')}>
                        下一步：配置规格与 SKU
                      </Button>
                    </div>
                  </div>
                ),
              },
              {
                key: 'sku',
                label: (
                  <Space>
                    <UnorderedListOutlined style={{ color: '#10B981' }} />
                    <span>规格与 SKU</span>
                    {skus.length > 0 && (
                      <Tag color="blue" style={{ marginLeft: 4, marginRight: 0 }}>
                        {skus.length} 个 SKU
                      </Tag>
                    )}
                  </Space>
                ),
                forceRender: true,
                children: (
                  <div style={{ paddingTop: 8 }}>
                    <div
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
                          disabled={isView || hasSavedSKUs}
                          onChange={(e) => handleSpecModeChange(e.target.value)}
                          buttonStyle="solid"
                        >
                          <Radio.Button value="single">单规格（普通商品）</Radio.Button>
                          <Radio.Button value="multi">多规格（如颜色、尺码等）</Radio.Button>
                        </Radio.Group>
                      </Space>
                      {hasSavedSKUs && (
                        <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                          商品已有已保存的 SKU，不能切换规格模式
                        </Typography.Text>
                      )}
                    </div>

                    {specMode === 'multi' ? (
                      <div>
                        <SpecEditor
                          specs={specs}
                          skus={skus}
                          disabled={isView}
                          onSpecsChange={setSpecs}
                          onGenerateSKUs={setSKUs}
                        />
                        <Divider style={{ margin: '16px 0' }}>SKU 价格与编码明细</Divider>
                        <SKUTable
                          specs={specs}
                          skus={skus}
                          disabled={isView}
                          onSKUsChange={setSKUs}
                        />
                      </div>
                    ) : (
                      <div>
                        <Alert
                          type="info"
                          showIcon
                          title="当前为单规格商品"
                          description="无需设置规格属性，下方默认提供一条唯一的商品 SKU 信息，用于配置销售价格、编码和库存单位。"
                          style={{ marginBottom: 16 }}
                        />
                        <SKUTable
                          specs={[]}
                          skus={skus}
                          disabled={isView}
                          onSKUsChange={setSKUs}
                        />
                      </div>
                    )}

                    <Divider style={{ margin: '24px 0 16px' }} />
                    <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                      <Button onClick={() => handleTabChange('basic')}>
                        上一步：基本信息
                      </Button>
                      <Button type="primary" onClick={() => handleTabChange('params')}>
                        下一步：配置商品参数
                      </Button>
                    </div>
                  </div>
                ),
              },
              {
                key: 'params',
                label: (
                  <Space>
                    <ProfileOutlined style={{ color: '#F59E0B' }} />
                    <span>商品参数</span>
                    {params.length > 0 && (
                      <Tag color="orange" style={{ marginLeft: 4, marginRight: 0 }}>
                        {params.length} 项
                      </Tag>
                    )}
                  </Space>
                ),
                forceRender: true,
                children: (
                  <div style={{ paddingTop: 8 }}>
                    <ParamEditor
                      params={params}
                      onChange={setParams}
                      disabled={isView}
                    />
                    <Divider style={{ margin: '24px 0 16px' }} />
                    <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                      <Button onClick={() => handleTabChange('sku')}>
                        上一步：规格与 SKU
                      </Button>
                      <Button type="primary" onClick={() => handleTabChange('media')}>
                        下一步：设置图片与详情
                      </Button>
                    </div>
                  </div>
                ),
              },
              {
                key: 'media',
                label: (
                  <Space>
                    <FileImageOutlined style={{ color: '#6366F1' }} />
                    <span>图片与详情</span>
                  </Space>
                ),
                forceRender: true,
                children: (
                  <div style={{ paddingTop: 8 }}>
                    <Form.Item
                      name="main_image_id"
                      label="商品主图"
                      extra="用于列表封面及商品首图展示，上架必须设置有效主图"
                    >
                      <ImagePicker multiple={false} disabled={isView} />
                    </Form.Item>

                    <Form.Item
                      label="商品轮播图"
                      extra="最多支持 10 张，支持点击下方上移/下移调整轮播顺序"
                    >
                      <ImagePicker
                        multiple={true}
                        maxCount={10}
                        disabled={isView}
                        value={galleryFileIDs}
                        onChange={(ids) => setGalleryFileIDs(ids)}
                      />

                      {galleryFileIDs.length > 0 && !isView && (
                        <div style={{ marginTop: 8 }}>
                          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                            已选 {galleryFileIDs.length} 张图片顺序调整：
                          </Typography.Text>
                          <Space wrap size={[6, 6]} style={{ marginTop: 4 }}>
                            {galleryFileIDs.map((fileId, idx) => (
                              <Tag
                                key={fileId}
                                style={{ padding: '2px 8px', display: 'inline-flex', alignItems: 'center' }}
                              >
                                <span style={{ marginRight: 6 }}>第 {idx + 1} 张</span>
                                <Space size={2}>
                                  <Button
                                    size="small"
                                    type="text"
                                    icon={<ArrowUpOutlined style={{ fontSize: 10 }} />}
                                    disabled={idx === 0}
                                    onClick={() => moveGalleryItem(idx, 'up')}
                                  />
                                  <Button
                                    size="small"
                                    type="text"
                                    icon={<ArrowDownOutlined style={{ fontSize: 10 }} />}
                                    disabled={idx === galleryFileIDs.length - 1}
                                    onClick={() => moveGalleryItem(idx, 'down')}
                                  />
                                  <Button
                                    size="small"
                                    type="text"
                                    danger
                                    icon={<DeleteOutlined style={{ fontSize: 10 }} />}
                                    onClick={() => removeGalleryItem(idx)}
                                  />
                                </Space>
                              </Tag>
                            ))}
                          </Space>
                        </div>
                      )}
                    </Form.Item>

                    <Form.Item
                      label="商品图文详情"
                      extra="支持富文本、表格、高亮标注与多媒体混排，自动进行服务端白名单安全清洗"
                    >
                      {isView ? (
                        <div
                          style={{
                            padding: '16px 20px',
                            background: '#F9FAFB',
                            border: '1px solid #E5E7EB',
                            borderRadius: 8,
                            minHeight: 200,
                            lineHeight: 1.8,
                          }}
                          dangerouslySetInnerHTML={{ __html: detailHTML || '<p style="color:#9CA3AF">暂无详情描述</p>' }}
                        />
                      ) : (
                        <RichEditor
                          placeholder="请输入商品详细介绍、规格参数表与图文说明..."
                          minHeight={360}
                          allowImageLibrary={true}
                          value={detailHTML}
                          onChange={setDetailHTML}
                        />
                      )}
                    </Form.Item>

                    <Divider style={{ margin: '24px 0 16px' }} />
                    <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                      <Button onClick={() => handleTabChange('params')}>
                        上一步：配置商品参数
                      </Button>
                      {!isView && (
                        <Button
                          type="primary"
                          icon={<SaveOutlined />}
                          loading={saving}
                          onClick={() => form.submit()}
                        >
                          保存商品
                        </Button>
                      )}
                    </div>
                  </div>
                ),
              },
            ]}
          />
        </Card>
      </Form>
    </Space>
  );
}
