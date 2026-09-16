import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  Button,
  Card,
  Drawer,
  Form,
  Space,
  Spin,
  Tabs,
  Tag,
  Tooltip,
  Typography,
  message,
  theme,
} from 'antd';
import {
  ArrowLeftOutlined,
  BulbOutlined,
  CheckOutlined,
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
import { loadRegions } from '../../freight/region-cascader';
import { buildCategoryOptionTree, normalizeMoney } from '../../types';
import { BasicTab } from './basic-tab';
import { DetailTab } from './detail-tab';
import { ParamsTab } from './params-tab';
import { SKUTab } from './sku-tab';
import type { ProductFormData } from './types';
import './edit.less';

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
            ? '已保存过 SKU 的商品禁止切换规格模式，避免线上库存与订单错乱'
            : '未创建真实 SKU 前可自由切换单规格与多规格模式',
          '上架时必须包含至少一个启用的 SKU，且启用 SKU 的售价必须大于 0',
        ],
      };
    }

    if (activeTab === 'params') {
      return {
        title: '商品参数',
        items: [
          '用于描述材质、产地、保质期、适用人群等非规格属性',
          '参数按填写的顺序展示在用户端详情页的“参数”面板中',
          '空行会在保存时自动过滤，无需手动清理未填写的行',
        ],
      };
    }

    if (activeTab === 'detail') {
      return {
        title: '商品详情',
        items: [
          '支持富文本排版，可插入标题、段落、图片与列表',
          '建议图片宽度控制在 750px 左右以获得最佳移动端视觉效果',
          '点击下方“保存商品”将一并提交基本信息、SKU、参数与详情',
        ],
      };
    }

    const freightTip =
      watchedFreightType === 2
        ? '当前选择固定运费，请确保运费金额设置合理'
        : watchedFreightType === 3
          ? '当前使用运费模板计算运费，可点击下方查看模板详情'
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

  // Freight template options
  const [freightTemplateOptions, setFreightTemplateOptions] = useState<FreightTemplateOption[]>([]);
  const loadFreightTemplates = useCallback(async () => {
    const res = await api.api.getFreightTemplateOptions();
    if (res.code === 1) {
      setFreightTemplateOptions(res.data ?? []);
    }
  }, []);

  // Load product detail
  const loadProductDetail = useCallback(async () => {
    if (!id) return;
    setLoading(true);
    setLoadError(null);
    const numID = Number(id);
    const res = await api.api.getProductDetail({ id: numID });
    setLoading(false);

    if (res.code === 1 && res.data) {
      const data = res.data;
      setInitialDetail(data);

      form.setFieldsValue({
        name: data.name ?? '',
        subtitle: data.subtitle ?? '',
        category_id: data.category_id && data.category_id > 0 ? data.category_id : undefined,
        main_image_id: data.main_image_id && data.main_image_id > 0 ? data.main_image_id : undefined,
        sort: data.sort ?? 0,
        status: data.status ?? 1,
        virtual_sales: data.virtual_sales ?? 0,
        freight_type: data.freight_type ?? 1,
        freight_fee: data.freight_fee ? normalizeMoney(data.freight_fee) : '0.00',
        freight_template_id:
          data.freight_template_id && data.freight_template_id > 0
            ? data.freight_template_id
            : undefined,
      });

      setGalleryFileIDs(data.gallery_file_ids || []);
      setParams(data.params || []);
      setDetailHTML(data.detail || '');

      const isMulti = data.specs && data.specs.length > 0;
      setSpecMode(isMulti ? 'multi' : 'single');
      setSpecs(data.specs || []);

      if (data.skus && data.skus.length > 0) {
        setSKUs(
          data.skus.map((s) => ({
            ...s,
            price: normalizeMoney(s.price),
            market_price: normalizeMoney(s.market_price),
            cost_price: normalizeMoney(s.cost_price),
            status: (s.status as 0 | 1) ?? 1,
          })),
        );
      }
    } else {
      const errMsg = res.message || '获取商品详情失败';
      setLoadError(errMsg);
      message.error(errMsg);
    }
  }, [id, form]);

  useEffect(() => {
    void loadCategories();
    void loadFreightTemplates();
    void loadProductDetail();
    void loadRegions().then(() => setRegionsLoaded(true));
  }, [loadCategories, loadFreightTemplates, loadProductDetail]);

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
      const feeVal = values.freight_fee ? String(values.freight_fee).trim() : '0.00';
      if (!feeVal || isNaN(Number(feeVal)) || Number(feeVal) < 0) {
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
    }

    // 6. Build final payload
    const payload = {
      name: values.name.trim(),
      subtitle: values.subtitle ? values.subtitle.trim() : undefined,
      category_id: values.category_id && values.category_id > 0 ? values.category_id : 0,
      main_image_id: values.main_image_id && values.main_image_id > 0 ? values.main_image_id : undefined,
      gallery_file_ids: galleryFileIDs,
      detail: detailHTML,
      sort: values.sort ?? 0,
      status: values.status as 0 | 1,
      virtual_sales: values.virtual_sales ?? 0,
      freight_type: freightType as 1 | 2 | 3,
      freight_fee: freightFee,
      freight_template_id: freightTemplateID,
      specs: specMode === 'multi' ? specs : [],
      skus: formattedSKUs,
      params: validParams,
    };

    setSaving(true);
    if (isEdit && id) {
      const res = await api.api.updateProduct({
        id: Number(id),
        ...payload,
      });
      setSaving(false);
      if (res.code === 1) {
        message.success('商品更新成功');
        navigate('/product/product');
      } else {
        message.error(res.message || '更新商品失败');
      }
    } else {
      const res = await api.api.createProduct(payload);
      setSaving(false);
      if (res.code === 1) {
        message.success('商品创建成功');
        navigate('/product/product');
      } else {
        message.error(res.message || '创建商品失败');
      }
    }
  };

  const [previewDrawerOpen, setPreviewDrawerOpen] = useState(false);
  const detailPreviewRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (previewDrawerOpen && detailPreviewRef.current && detailHTML) {
      detailPreviewRef.current.innerHTML = detailHTML;
    }
  }, [previewDrawerOpen, detailHTML]);

  const h5PreviewDrawer = (
    <Drawer
      title="移动端商详页预览"
      placement="right"
      width={420}
      onClose={() => setPreviewDrawerOpen(false)}
      open={previewDrawerOpen}
      styles={{
        body: { padding: 0, background: '#F3F4F6' },
      }}
    >
      <div style={{ maxWidth: 375, margin: '0 auto', background: '#FFF', minHeight: '100%' }}>
        {previewCoverUrl ? (
          <div style={{ width: '100%', height: 375, overflow: 'hidden' }}>
            <img
              src={previewCoverUrl}
              alt="cover"
              style={{ width: '100%', height: '100%', objectFit: 'contain' }}
            />
          </div>
        ) : (
          <div
            style={{
              width: '100%',
              height: 240,
              background: '#E5E7EB',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: '#9CA3AF',
            }}
          >
            暂无主图
          </div>
        )}

        <div style={{ padding: 16 }}>
          <div style={{ color: token.colorPrimary, fontSize: 24, fontWeight: 700 }}>
            <span style={{ fontSize: 16 }}>¥</span>
            {previewProductPrice}
          </div>
          <div
            style={{
              fontSize: 16,
              fontWeight: 600,
              marginTop: 8,
              color: '#111827',
              lineHeight: 1.4,
            }}
          >
            {previewProductName}
          </div>
          <div style={{ fontSize: 13, color: '#6B7280', marginTop: 4, lineHeight: 1.4 }}>
            {previewProductSubtitle}
          </div>
        </div>

        {params.length > 0 && (
          <div style={{ borderTop: '8px solid #F3F4F6', padding: 16 }}>
            <div style={{ fontSize: 14, fontWeight: 600, marginBottom: 12 }}>商品参数</div>
            <div style={{ display: 'grid', gridTemplateColumns: '80px 1fr', gap: '8px 12px', fontSize: 13 }}>
              {params.map((p, idx) => (
                <div key={idx} style={{ display: 'contents' }}>
                  <div style={{ color: '#9CA3AF' }}>{p.name}</div>
                  <div style={{ color: '#374151' }}>{p.value}</div>
                </div>
              ))}
            </div>
          </div>
        )}

        <div style={{ borderTop: '8px solid #F3F4F6', padding: 16 }}>
          <div style={{ fontSize: 14, fontWeight: 600, marginBottom: 12 }}>图文详情</div>
          {detailHTML ? (
            <div
              ref={detailPreviewRef}
              className="product-detail-rendered-preview"
              style={{ fontSize: 14, lineHeight: 1.6 }}
            />
          ) : (
            <div style={{ color: '#9CA3AF', fontSize: 13, textAlign: 'center', padding: '24px 0' }}>
              暂未编辑图文详情
            </div>
          )}
        </div>
      </div>
    </Drawer>
  );

  const pageTitle = isCreate ? '新建商品' : isView ? '查看商品详情' : '编辑商品';

  return (
    <div
      className="product-edit-page"
      style={
        {
          '--product-edit-primary': token.colorPrimary,
          '--product-edit-primary-bg': token.colorPrimaryBg,
          '--product-edit-heading': token.colorTextHeading,
          '--product-edit-text': token.colorText,
          '--product-edit-secondary': token.colorTextSecondary,
          '--product-edit-tertiary': token.colorTextTertiary,
          '--product-edit-border': token.colorBorderSecondary,
          '--product-edit-border-strong': token.colorBorder,
          '--product-edit-bg': token.colorBgContainer,
          '--product-edit-bg-subtle': token.colorFillAlter,
          '--product-edit-warning': token.colorWarning,
        } as React.CSSProperties
      }
    >
      <Card className="product-edit-heading-card">
        <div className="product-edit-heading">
          <Button
            type="text"
            icon={<ArrowLeftOutlined />}
            onClick={() => navigate('/product/product')}
            className="product-edit-back"
          >
            返回列表
          </Button>

          <div className="product-edit-heading-copy">
            <Typography.Title level={3}>{pageTitle}</Typography.Title>
            <Typography.Text type="secondary">
              {isView
                ? '查看商品信息、规格价格、库存及详细描述'
                : '按步骤维护商品基础资料、多规格 SKU、参数属性与图文详情'}
            </Typography.Text>
          </div>

          <div style={{ marginLeft: 'auto', display: 'flex', alignItems: 'center', gap: 12 }}>
            {isEdit && (
              <Tag color={initialDetail?.status === 1 ? 'success' : 'default'} style={{ margin: 0 }}>
                {initialDetail?.status === 1 ? '已上架' : '已下架'}
              </Tag>
            )}
            <Button icon={<EyeOutlined />} onClick={() => setPreviewDrawerOpen(true)}>
              商详页预览
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
      </Card>

      <Spin spinning={loading} tip="正在加载商品信息...">
        {loadError && (
          <div style={{ textAlign: 'center', padding: '48px 0' }}>
            <p style={{ color: token.colorError, marginBottom: 16 }}>{loadError}</p>
            <Button onClick={() => void loadProductDetail()}>重试</Button>
          </div>
        )}

        {!loadError && (
          <div className="product-edit-layout">
            <main className="product-edit-main">
              <Form<ProductFormData>
                form={form}
                layout="horizontal"
                labelCol={{ span: 4, style: { maxWidth: 140 } }}
                wrapperCol={{ span: 20 }}
                className="product-edit-form"
                onFinish={handleFinish}
                onFinishFailed={handleFinishFailed}
                disabled={isView}
                initialValues={{
                  status: 1,
                  sort: 0,
                  freight_type: 1,
                  freight_fee: '0.00',
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
                          <BasicTab
                            form={form}
                            categoryTreeData={categoryTreeData}
                            freightTemplateOptions={freightTemplateOptions}
                            galleryFileIDs={galleryFileIDs}
                            onGalleryChange={setGalleryFileIDs}
                            watchedFreightType={watchedFreightType}
                            watchedFreightTemplateID={watchedFreightTemplateID}
                            isView={isView}
                            isEdit={isEdit}
                            initialDetail={initialDetail}
                            onNext={() => handleTabChange('sku')}
                          />
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
                          <SKUTab
                            specMode={specMode}
                            hasSavedSKUs={hasSavedSKUs}
                            onSpecModeChange={handleSpecModeChange}
                            specs={specs}
                            onSpecsChange={setSpecs}
                            skus={skus}
                            onSKUsChange={setSKUs}
                            disabled={isView}
                            onPrev={() => handleTabChange('basic')}
                            onNext={() => handleTabChange('params')}
                          />
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
                          <ParamsTab
                            params={params}
                            onChange={setParams}
                            disabled={isView}
                            onPrev={() => handleTabChange('sku')}
                            onNext={() => handleTabChange('detail')}
                          />
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
                          <DetailTab
                            detailHTML={detailHTML}
                            onChange={setDetailHTML}
                            disabled={isView}
                            saving={saving}
                            productName={previewProductName}
                            productPrice={previewProductPrice}
                            coverUrl={previewCoverUrl}
                            onPrev={() => handleTabChange('params')}
                            onSave={() => form.submit()}
                          />
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
                          icon={<CheckOutlined />}
                          loading={saving}
                          onClick={() => form.submit()}
                        >
                          保存修改
                        </Button>
                      </div>
                    )}

                    {!isCreate && id && canViewInventory && (
                      <Tooltip title="快捷跳转查看该商品在库存管理中的实时入库流水与盘点日志">
                        <Button
                          block
                          className="product-edit-inventory-action"
                          onClick={() => navigate(`/product/inventory?keyword=${encodeURIComponent(watchedName || initialDetail?.name || '')}`)}
                        >
                          查看商品库存明细 ↗
                        </Button>
                      </Tooltip>
                    )}
                  </Card>
                )}

                <Card
                  className="product-edit-help-card"
                  size="small"
                  title={
                    <span className="product-edit-side-title">
                      <BulbOutlined />
                      <span>填写帮助提示</span>
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
                      <span>实时效果预览</span>
                    </span>
                  }
                  extra={
                    <Button type="link" size="small" style={{ padding: 0 }} onClick={() => setPreviewDrawerOpen(true)}>
                      商详预览 ↗
                    </Button>
                  }
                >
                  <div className="product-edit-preview-media">
                    {previewCoverUrl ? (
                      <img src={previewCoverUrl} alt="预览图片" />
                    ) : (
                      <div className="product-edit-preview-empty">
                        <ShopOutlined />
                        <span>暂无商品封面主图</span>
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
        )}
      </Spin>

      {h5PreviewDrawer}
    </div>
  );
}
