import type { CSSProperties } from 'react';
import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Button,
  Card,
  Divider,
  Form,
  Input,
  InputNumber,
  message,
  Radio,
  Select,
  Space,
  Spin,
  Tabs,
  Tag,
  TreeSelect,
  Typography,
  theme,
} from 'antd';
import {
  ArrowLeftOutlined,
  BulbOutlined,
  CheckOutlined,
  DeleteOutlined,
  EyeOutlined,
  FileTextOutlined,
  ProfileOutlined,
  SaveOutlined,
  ShopOutlined,
  UnorderedListOutlined,
} from '@ant-design/icons';
import { useLocation, useNavigate, useParams, useSearchParams } from 'react-router-dom';

import { api } from '@/framework/http';
import type {
  FreightTemplateOption,
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
import { formatRegionLabel, loadRegions } from '../freight/region-cascader';
import { ParamEditor } from './param-editor';
import { ProductDetailEditor } from './detail-editor';
import { SKUTable } from './sku-table';
import { SpecEditor } from './spec-editor';
import './edit.less';

type ProductFormData = {
  name: string;
  subtitle?: string;
  category_id?: number;
  main_image_id?: number | null;
  sort?: number;
  status: number;
  virtual_sales?: number;
  detail?: string;
  freight_type?: number;
  freight_fee?: string;
  freight_template_id?: number;
};

export default function ProductEditPage() {
  const { id } = useParams<{ id?: string }>();
  const navigate = useNavigate();
  const location = useLocation();
  const { token } = theme.useToken();
  const hasPermission = usePermission();
  const canViewInventory = hasPermission('product:inventory:list');

  const isCreate = !id;
  const isView = location.pathname.endsWith(`/${id}`); // /product/product/:id without /edit
  const isEdit = !isCreate && !isView;

  const [searchParams, setSearchParams] = useSearchParams();
  const rawTab = searchParams.get('tab');
  const [activeTab, setActiveTab] = useState<'basic' | 'sku' | 'params' | 'detail'>(() => {
    if (rawTab === 'sku' || rawTab === 'params' || rawTab === 'detail') return rawTab;
    if (rawTab === 'media') return 'detail';
    return 'basic';
  });

  const handleTabChange = useCallback(
    (key: string) => {
      const normalizedKey = key === 'media' ? 'detail' : key;
      const nextTab = (normalizedKey === 'sku' || normalizedKey === 'params' || normalizedKey === 'detail'
        ? normalizedKey
        : 'basic') as 'basic' | 'sku' | 'params' | 'detail';
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
    if (currentTab === 'sku' || currentTab === 'params' || currentTab === 'detail') {
      setActiveTab(currentTab);
    } else if (currentTab === 'media') {
      setActiveTab('detail');
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
  const [, setRegionsLoaded] = useState(false);

  // Live preview watched values
  const watchedName = Form.useWatch('name', form);
  const watchedSubtitle = Form.useWatch('subtitle', form);
  const previewProductName = watchedName || initialDetail?.name || '商品名称';
  const previewProductSubtitle =
    watchedSubtitle || initialDetail?.subtitle || '商品副标题或卖点将在这里展示';
  const previewProductPrice = skus[0]?.price || initialDetail?.skus?.[0]?.price || '0.00';
  const previewCoverUrl = initialDetail?.main_image_url || initialDetail?.gallery_image_urls?.[0];
  const watchedFreightType = Form.useWatch('freight_type', form) ?? 1;
  const watchedFreightTemplateID = Form.useWatch('freight_template_id', form);

  const activeTabHelp = useMemo(() => {
    if (activeTab === 'sku') {
      return {
        title: '规格与 SKU',
        items: [
          specMode === 'multi'
            ? '最多支持 3 个规格维度，每个维度最多 20 个选项'
            : '当前为单规格商品，只需维护一条默认 SKU',
          specMode === 'multi'
            ? '生成组合后，在下方统一设置价格、编码、库存单位和状态'
            : '默认 SKU 用于设置销售价格、编码、库存单位和状态',
          hasSavedSKUs
            ? '商品已有已保存的 SKU，不能切换规格模式或删除已引用的规格选项'
            : '切换规格模式前，请确认当前 SKU 数据是否需要保留',
          '上架商品至少需要一个启用且销售价大于 0 元的 SKU',
        ],
      };
    }

    if (activeTab === 'params') {
      return {
        title: '商品参数',
        items: [
          '商品参数用于展示品牌、产地、材质、保质期等属性',
          '参数仅用于商品信息展示，不影响库存与销售价格',
          '参数名称不能为空，名称和值均为空的参数不会保存',
          '可使用常用参数快速填入，并通过上下移动调整展示顺序',
        ],
      };
    }

    if (activeTab === 'detail') {
      return {
        title: '商品详情',
        items: [
          '富文本编辑器默认按 375px 手机端宽度排版',
          '编辑器右侧提供 1:1 移动端实时预览',
          '图片和段落应优先检查手机端的阅读效果',
          '保存前确认图文内容与商品基础信息一致',
        ],
      };
    }

    const freightTip =
      watchedFreightType === 2
        ? '固定运费适用于全国统一收费，请填写有效的快递运费金额'
        : watchedFreightType === 3
          ? '运费模板会根据买家地址计算基础运费与地区附加运费'
          : '当前选择全国包邮，无需设置额外运费';

    return {
      title: '基本信息',
      items: [
        '商品副标题展示在名称下方，用于提炼核心卖点',
        '上架商品必须选择已启用的分类并设置有效主图',
        '轮播图最多支持 10 张，可直接拖拽图片卡片调整顺序',
        freightTip,
        '排序值越小越靠前；虚拟销量仅用于前端展示',
      ],
    };
  }, [activeTab, hasSavedSKUs, specMode, watchedFreightType]);

  // Load category options
  const loadCategories = useCallback(async () => {
    const res = await api.api.getProductCategoryOptions();
    if (res.code === 1) {
      setCategoryOptions(res.data ?? []);
    }
  }, []);

  // Load freight template options
  const [freightTemplateOptions, setFreightTemplateOptions] = useState<FreightTemplateOption[]>([]);
  const loadFreightTemplates = useCallback(async () => {
    const res = await api.api.getFreightTemplateOptions();
    if (res.code === 1) {
      setFreightTemplateOptions(res.data ?? []);
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
        freight_type: data.freight_type ?? 1,
        freight_fee: data.freight_fee ? normalizeMoney(data.freight_fee) : '0.00',
        freight_template_id: data.freight_template_id && data.freight_template_id > 0 ? data.freight_template_id : undefined,
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
    void loadFreightTemplates();
    void loadRegions()
      .then(() => setRegionsLoaded(true))
      .catch(() => {});
    if (id) {
      void loadDetail(Number(id));
    }
  }, [id, loadCategories, loadFreightTemplates, loadDetail]);

  // Category TreeSelect Data
  const categoryTreeData = useMemo(() => {
    return buildCategoryOptionTree(categoryOptions);
  }, [categoryOptions]);


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
      if (['name', 'category_id', 'subtitle', 'main_image_id', 'status', 'sort', 'virtual_sales'].includes(firstField)) {
        handleTabChange('basic');
      } else if (['detail'].includes(firstField)) {
        handleTabChange('detail');
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
        handleTabChange('basic');
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

    // 5. Format and validate freight
    const freightType = Number(values.freight_type) || 1;
    let freightFee = '0.00';
    let freightTemplateID: number | undefined = undefined;

    if (freightType === 1) {
      freightFee = '0.00';
      freightTemplateID = undefined;
    } else if (freightType === 2) {
      const feeVal = values.freight_fee;
      if (
        feeVal === undefined ||
        feeVal === null ||
        feeVal === '' ||
        isNaN(Number(feeVal)) ||
        Number(feeVal) < 0
      ) {
        message.error('请输入有效的固定运费金额');
        handleTabChange('basic');
        return;
      }
      freightFee = normalizeMoney(feeVal);
      freightTemplateID = undefined;
    } else if (freightType === 3) {
      if (!values.freight_template_id || values.freight_template_id <= 0) {
        message.error('请选择运费模板');
        handleTabChange('basic');
        return;
      }
      freightFee = '0.00';
      freightTemplateID = values.freight_template_id;
    } else {
      message.error('非法配送方式');
      return;
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
        freight_type: freightType,
        freight_fee: freightFee,
        freight_template_id: freightTemplateID,
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
        freight_type: freightType,
        freight_fee: freightFee,
        freight_template_id: freightTemplateID,
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
    <div
      className="product-edit-page"
      style={{
        '--product-edit-bg': token.colorBgContainer,
        '--product-edit-bg-subtle': token.colorFillAlter,
        '--product-edit-border': token.colorBorderSecondary,
        '--product-edit-border-strong': token.colorBorder,
        '--product-edit-primary': token.colorPrimary,
        '--product-edit-primary-bg': token.colorPrimaryBg,
        '--product-edit-heading': token.colorTextHeading,
        '--product-edit-text': token.colorText,
        '--product-edit-secondary': token.colorTextSecondary,
        '--product-edit-tertiary': token.colorTextTertiary,
        '--product-edit-warning': token.colorWarning,
      } as CSSProperties}
    >
      <div className="product-edit-layout">
        <main className="product-edit-main">
          <Card className="product-edit-heading-card">
            <div className="product-edit-heading">
              <Button
                type="text"
                className="product-edit-back"
                icon={<ArrowLeftOutlined />}
                onClick={() => navigate('/product/product')}
              >
                返回
              </Button>
              <div className="product-edit-heading-copy">
                <Typography.Title level={3}>
                  {isCreate ? '新建商品' : isView ? '查看商品详情' : '编辑商品'}
                </Typography.Title>
                <Typography.Paragraph>
                  {isCreate
                    ? '完善商品信息，让你的商品更容易被用户发现'
                    : isView
                      ? '查看商品基础信息、规格和展示内容'
                      : '更新商品基础信息、规格和展示内容'}
                </Typography.Paragraph>
              </div>
            </div>
          </Card>

          <Form<ProductFormData>
            className="product-edit-form"
            form={form}
            layout="horizontal"
            labelCol={{ flex: '150px' }}
            wrapperCol={{ style: { maxWidth: 540 } }}
            disabled={isView}
            onFinish={handleFinish}
            onFinishFailed={handleFinishFailed}
            initialValues={{
              sort: 0,
              status: 0,
              virtual_sales: 0,
            }}
          >
            <Card className="product-edit-form-card">
          <Tabs
            className="product-edit-tabs"
            activeKey={activeTab}
            onChange={handleTabChange}
            size="large"
            items={[
              {
                key: 'basic',
                label: (
                  <Space>
                    <ShopOutlined style={{ color: token.colorPrimary }} />
                    <span>基本信息</span>
                  </Space>
                ),
                forceRender: true,
                children: (
                  <div style={{ paddingTop: 16 }}>
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

                    <Form.Item
                      name="subtitle"
                      label="商品副标题 / 卖点"
                    >
                      <Input placeholder="请输入商品卖点，展示在商品名称下方" maxLength={255} showCount />
                    </Form.Item>

                    <Form.Item
                      name="category_id"
                      label="商品分类"
                    >
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

                    <Form.Item
                      name="main_image_id"
                      label="商品主图"
                    >
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
                        onChange={(ids) => setGalleryFileIDs(ids)}
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
                    <UnorderedListOutlined style={{ color: token.colorPrimary }} />
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
                          disabled={isView || hasSavedSKUs}
                          onChange={(e) => handleSpecModeChange(e.target.value)}
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
                    <ProfileOutlined style={{ color: token.colorPrimary }} />
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
                      <Button type="primary" onClick={() => handleTabChange('detail')}>
                        下一步：编辑商品详情
                      </Button>
                    </div>
                  </div>
                ),
              },
              {
                key: 'detail',
                label: (
                  <Space>
                    <FileTextOutlined style={{ color: token.colorPrimary }} />
                    <span>商品详情</span>
                  </Space>
                ),
                forceRender: true,
                children: (
                  <div style={{ paddingTop: 8 }}>
                    <Form.Item
                      label="商品图文详情"
                      layout="vertical"
                      labelCol={{ span: 24 }}
                      wrapperCol={{ span: 24, style: { maxWidth: '100%' } }}
                    >
                      <ProductDetailEditor
                        value={detailHTML}
                        onChange={setDetailHTML}
                        disabled={isView}
                        productName={previewProductName}
                        productPrice={previewProductPrice}
                        coverUrl={previewCoverUrl}
                      />
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
        </main>

        <aside className="product-edit-aside">
          <div className="product-edit-sticky">
            {(!isView || (!isCreate && id && canViewInventory)) && (
              <Card className="product-edit-action-card" size="small">
                {!isView && (
                  <div className="product-edit-primary-actions">
                    <Button onClick={() => navigate('/product/product')}>取消</Button>
                    <Button
                      type="primary"
                      icon={<SaveOutlined />}
                      loading={saving}
                      onClick={() => form.submit()}
                    >
                      保存商品
                    </Button>
                  </div>
                )}
                {!isCreate && id && canViewInventory && (
                  <Button
                    className="product-edit-inventory-action"
                    block
                    onClick={() => navigate(`/product/inventory?product_id=${id}`)}
                  >
                    查看库存
                  </Button>
                )}
              </Card>
            )}

            <Card
              className="product-edit-help-card"
              size="small"
              title={
                <span className="product-edit-side-title">
                  <BulbOutlined />
                  操作提示
                </span>
              }
              extra={<span className="product-edit-help-context">{activeTabHelp.title}</span>}
            >
              <ol>
                {activeTabHelp.items.map((item) => (
                  <li key={item}>{item}</li>
                ))}
              </ol>
            </Card>

            <Card
              className="product-edit-preview-card"
              size="small"
              title={
                <span className="product-edit-side-title">
                  <EyeOutlined />
                  商品预览
                </span>
              }
            >
              <div className="product-edit-preview-media">
                {previewCoverUrl ? (
                  <img src={previewCoverUrl} alt={previewProductName} />
                ) : (
                  <div className="product-edit-preview-empty">
                    <EyeOutlined />
                    <span>暂无商品主图</span>
                  </div>
                )}
              </div>
              <div className="product-edit-preview-name">{previewProductName}</div>
              <div className="product-edit-preview-subtitle">{previewProductSubtitle}</div>
              <div className="product-edit-preview-price">¥{previewProductPrice}</div>
            </Card>
          </div>
        </aside>
      </div>
    </div>
  );
}
