import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  Button,
  DatePicker,
  Descriptions,
  Drawer,
  Form,
  Input,
  InputNumber,
  Pagination,
  Select,
  Space,
  Spin,
  Table,
  Tag,
  Tooltip,
  Typography,
  message,
  theme,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  AppstoreOutlined,
  DownOutlined,
  HistoryOutlined,
  InfoCircleOutlined,
  ReloadOutlined,
  SearchOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import { useSearchParams } from 'react-router-dom';

import { api } from '@/framework/http';
import type {
  ProductInventoryLogListRequest,
  ProductInventoryLogRow,
  ProductInventoryRow,
  ProductInventorySpecOption,
  UserRow,
} from '@/framework/http/generated_api';
import { usePermission } from '@/framework/store';
import { formatTime } from '@/shared/utils/format';
import { ContentCard, PageHeader, adminListVars } from '@/shared/admin-ui';

const { RangePicker } = DatePicker;

interface SearchFormValues {
  sku_id?: number;
  change_type?: string;
  stock_change?: 'increase' | 'decrease' | '';
  source?: 'manual' | 'order' | 'system' | '';
  operator_id?: number;
  date_range?: [Dayjs, Dayjs];
  // Advanced filters
  advanced_sku_id?: number;
  advanced_product_id?: number;
  request_id?: string;
  advanced_operator_id?: number;
}

interface SkuSelectOption {
  value: number;
  label: React.ReactNode;
  item: ProductInventoryRow;
}

const CHANGE_TYPE_CONFIG: Record<
  string,
  { label: string; tagColor: string; defaultDir: string; isIncrease?: boolean; isDecrease?: boolean }
> = {
  manual_in: { label: '采购入库', tagColor: 'blue', defaultDir: '入库', isIncrease: true },
  manual_out: { label: '人工出库', tagColor: 'orange', defaultDir: '出库', isDecrease: true },
  stocktake: { label: '盘点调整', tagColor: 'purple', defaultDir: '盘点' },
  warning_set: { label: '设置预警', tagColor: 'cyan', defaultDir: '预警' },
  order_lock: { label: '订单锁定', tagColor: 'gold', defaultDir: '锁定', isDecrease: false },
  order_out: { label: '订单扣减', tagColor: 'volcano', defaultDir: '出库', isDecrease: true },
  order_cancel: { label: '订单释放', tagColor: 'green', defaultDir: '释放', isIncrease: true },
  order_unlock: { label: '订单解锁', tagColor: 'green', defaultDir: '解锁', isIncrease: true },
  after_sale_return: { label: '售后退货', tagColor: 'cyan', defaultDir: '入库', isIncrease: true },
};

export default function ProductInventoryLogPage() {
  const { token } = theme.useToken();
  const [searchParams, setSearchParams] = useSearchParams();
  const hasPermission = usePermission();
  const canLog = hasPermission('product:inventory:log');

  const [form] = Form.useForm<SearchFormValues>();

  const [rows, setRows] = useState<ProductInventoryLogRow[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [selectedRow, setSelectedRow] = useState<ProductInventoryLogRow | null>(null);

  // Sku search & options
  const [skuOptions, setSkuOptions] = useState<SkuSelectOption[]>([]);
  const [skuSearching, setSkuSearching] = useState(false);
  const skuSearchTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // User options for operator selector
  const [userOptions, setUserOptions] = useState<{ label: string; value: number }[]>([]);

  // Pagination state
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  // Deep linked ID filters from URL
  const paramProductID = searchParams.get('product_id');
  const paramSKUID = searchParams.get('sku_id');

  const initialProductID =
    paramProductID && /^\d+$/.test(paramProductID) ? Number(paramProductID) : undefined;
  const initialSKUID =
    paramSKUID && /^\d+$/.test(paramSKUID) ? Number(paramSKUID) : undefined;

  // Active query parameters
  const [activeQuery, setActiveQuery] = useState<SearchFormValues>({
    sku_id: initialSKUID,
    advanced_product_id: initialProductID,
    advanced_sku_id: initialSKUID,
    change_type: undefined,
    stock_change: undefined,
    source: undefined,
    operator_id: undefined,
    request_id: undefined,
    date_range: undefined,
  });

  const reqSeqRef = useRef(0);

  // Load system users for operator selector
  useEffect(() => {
    const loadUsers = async () => {
      const res = await api.api.listSystemUsers({ page: 1, page_size: 100 });
      if (res.code === 1 && res.data) {
        setUserOptions(
          res.data
            .filter((u): u is UserRow & { id: number } => typeof u.id === 'number')
            .map((u) => ({
              value: u.id,
              label: u.nickname ? `${u.nickname} (${u.username || ''})` : (u.username || `用户 #${u.id}`),
            }))
        );
      }
    };
    void loadUsers();
  }, []);

  // Preload initial SKU if passed via URL parameter
  useEffect(() => {
    if (!initialSKUID) return;
    const loadInitialSKU = async () => {
      const res = await api.api.listProductInventories({
        sku_id: initialSKUID,
        page_size: 1,
      });
      const data = (res.data as ProductInventoryRow[] | undefined) ?? [];
      if (res.code === 1 && data.length > 0) {
        const item: ProductInventoryRow = data[0];
        const specText =
          item.specs?.map((s: ProductInventorySpecOption) => s.value_name).filter(Boolean).join(' / ') ||
          '默认规格';
        setSkuOptions([
          {
            value: item.sku_id!,
            item,
            label: (
              <div className="flex flex-col py-0.5">
                <span className="font-medium text-sm text-gray-800 truncate">
                  {item.product_name}
                </span>
                <span className="text-xs text-gray-500">
                  规格: {specText} {item.sku_code ? `· 编码: ${item.sku_code}` : ''}
                </span>
              </div>
            ),
          },
        ]);
        form.setFieldValue('sku_id', initialSKUID);
        form.setFieldValue('advanced_sku_id', initialSKUID);
      }
    };
    void loadInitialSKU();
  }, [initialSKUID, form]);

  // Handle remote SKU search
  const handleSearchSKU = useCallback((query: string) => {
    if (skuSearchTimerRef.current) clearTimeout(skuSearchTimerRef.current);
    if (!query || !query.trim()) {
      return;
    }
    skuSearchTimerRef.current = setTimeout(async () => {
      setSkuSearching(true);
      const res = await api.api.listProductInventories({
        keyword: query.trim(),
        page_size: 20,
      });
      setSkuSearching(false);
      const data = (res.data as ProductInventoryRow[] | undefined) ?? [];
      if (res.code === 1 && data.length > 0) {
        setSkuOptions(
          data
            .filter((item: ProductInventoryRow): item is ProductInventoryRow & { sku_id: number } => typeof item.sku_id === 'number')
            .map((item: ProductInventoryRow & { sku_id: number }) => {
              const specText =
                item.specs?.map((s: ProductInventorySpecOption) => s.value_name).filter(Boolean).join(' / ') ||
                '默认规格';
              return {
                value: item.sku_id,
                item,
                label: (
                  <div className="flex flex-col py-0.5">
                    <span className="font-medium text-sm text-gray-800 truncate">
                      {item.product_name}
                    </span>
                    <span className="text-xs text-gray-500">
                      规格: {specText} {item.sku_code ? `· 编码: ${item.sku_code}` : ''}
                    </span>
                  </div>
                ),
              };
            })
        );
      }
    }, 300);
  }, []);

  // Load logs
  const loadLogs = useCallback(async () => {
    if (!canLog) return;
    const currentSeq = ++reqSeqRef.current;
    setLoading(true);

    let createdFrom: string | undefined;
    let createdTo: string | undefined;

    if (activeQuery.date_range && activeQuery.date_range[0] && activeQuery.date_range[1]) {
      // Format as RFC3339
      createdFrom = activeQuery.date_range[0].startOf('day').toISOString();
      // Use start of next day to fully include the selected end day
      createdTo = activeQuery.date_range[1].add(1, 'day').startOf('day').toISOString();
    }

    const resolvedSKUID = activeQuery.advanced_sku_id ?? activeQuery.sku_id;
    const resolvedProductID = activeQuery.advanced_product_id;
    let effectiveOperatorID = activeQuery.advanced_operator_id ?? activeQuery.operator_id;
    if (activeQuery.source === 'system' && effectiveOperatorID === undefined) {
      effectiveOperatorID = 0;
    }

    const req: ProductInventoryLogListRequest = {
      page,
      page_size: pageSize,
      product_id: resolvedProductID,
      sku_id: resolvedSKUID,
      change_type: activeQuery.change_type || undefined,
      operator_id: effectiveOperatorID,
      request_id: activeQuery.request_id?.trim() || undefined,
      created_from: createdFrom,
      created_to: createdTo,
    };

    const res = await api.api.listProductInventoryLogs(req);

    if (currentSeq !== reqSeqRef.current) return;
    setLoading(false);

    if (res.code === 1) {
      setRows((res.data as ProductInventoryLogRow[]) ?? []);
      setTotal(res.total ?? 0);
    } else {
      message.error(res.message || '加载库存流水失败');
    }
  }, [canLog, page, pageSize, activeQuery]);

  useEffect(() => {
    void loadLogs();
  }, [loadLogs]);

  // Handle Search submit
  const handleSearch = (values: SearchFormValues) => {
    setPage(1);
    const resolvedSKUID = values.advanced_sku_id ?? values.sku_id;
    const resolvedProductID = values.advanced_product_id ?? initialProductID;
    const resolvedOperatorID = values.advanced_operator_id ?? values.operator_id;

    const newQuery: SearchFormValues = {
      ...values,
      sku_id: resolvedSKUID,
      advanced_product_id: resolvedProductID,
      advanced_sku_id: resolvedSKUID,
      operator_id: resolvedOperatorID,
    };
    setActiveQuery(newQuery);

    // Sync URL params
    const newParams = new URLSearchParams();
    if (resolvedProductID) newParams.set('product_id', String(resolvedProductID));
    if (resolvedSKUID) newParams.set('sku_id', String(resolvedSKUID));
    setSearchParams(newParams, { replace: true });
  };

  // Handle Reset filters
  const handleReset = () => {
    form.resetFields();
    setSearchParams({}, { replace: true });
    setPage(1);
    setActiveQuery({
      sku_id: undefined,
      change_type: undefined,
      stock_change: undefined,
      source: undefined,
      operator_id: undefined,
      request_id: undefined,
      advanced_sku_id: undefined,
      advanced_product_id: undefined,
      advanced_operator_id: undefined,
      date_range: undefined,
    });
  };

  // Client-side filtering for business filters (stock_change, source)
  const displayRows = useMemo(() => {
    let list = rows;
    if (activeQuery.stock_change === 'increase') {
      list = list.filter((r) => (r.stock_delta ?? 0) > 0);
    } else if (activeQuery.stock_change === 'decrease') {
      list = list.filter((r) => (r.stock_delta ?? 0) < 0);
    }

    if (activeQuery.source === 'manual') {
      const manualTypes = new Set(['manual_in', 'manual_out', 'stocktake', 'warning_set']);
      list = list.filter((r) => r.change_type && manualTypes.has(r.change_type));
    } else if (activeQuery.source === 'order') {
      const orderTypes = new Set([
        'order_lock',
        'order_out',
        'order_cancel',
        'order_unlock',
        'after_sale_return',
      ]);
      list = list.filter((r) => r.change_type && orderTypes.has(r.change_type));
    } else if (activeQuery.source === 'system') {
      list = list.filter((r) => r.operator_id === 0 || !r.operator_id);
    }

    return list;
  }, [rows, activeQuery.stock_change, activeQuery.source]);

  // Redesigned ERP-style Columns definition
  const columns: ColumnsType<ProductInventoryLogRow> = [
    {
      title: '商品信息',
      key: 'product_info',
      render: (_, record) => {
        const snap = record.sku_snapshot;
        const specs = snap?.specs ?? [];
        return (
          <div className="flex flex-col gap-1 py-1">
            <Typography.Text
              strong
              className="text-sm"
              ellipsis={{ tooltip: snap?.product_name }}
              style={{ color: token.colorTextHeading }}
            >
              {snap?.product_name || '—'}
            </Typography.Text>
            <div className="flex flex-wrap items-center gap-1">
              {specs.length === 0 ? (
                <Tag color="default" style={{ margin: 0, fontSize: 11, lineHeight: '18px' }}>
                  默认规格
                </Tag>
              ) : (
                specs.map((s) => (
                  <Tag
                    key={`${s.spec_id}-${s.value_id}`}
                    color="blue"
                    style={{ margin: 0, fontSize: 11, lineHeight: '18px' }}
                  >
                    {s.spec_name}: {s.value_name}
                  </Tag>
                ))
              )}
            </div>
            {snap?.sku_code && (
              <span className="text-xs font-mono" style={{ color: token.colorTextTertiary }}>
                编码: {snap.sku_code}
              </span>
            )}
          </div>
        );
      },
    },
    {
      title: '库存变动',
      key: 'stock_change',
      width: 130,
      align: 'right',
      render: (_, record) => {
        if (record.change_type === 'warning_set') {
          return (
            <div className="flex flex-col items-end leading-tight">
              <span className="text-sm" style={{ color: token.colorTextTertiary }}>
                —
              </span>
              <span className="text-xs opacity-70 mt-0.5" style={{ color: token.colorTextTertiary }}>
                预警变更
              </span>
            </div>
          );
        }
        const delta = record.stock_delta ?? 0;
        const isPositive = delta > 0;
        const isNegative = delta < 0;

        let dirLabel = '无变动';
        if (record.change_type === 'manual_in' || record.change_type === 'after_sale_return') {
          dirLabel = '入库';
        } else if (record.change_type === 'manual_out' || record.change_type === 'order_out') {
          dirLabel = '出库';
        } else if (record.change_type === 'stocktake') {
          dirLabel = isPositive ? '盘盈入库' : isNegative ? '盘亏出库' : '盘点调整';
        } else if (record.change_type === 'order_cancel' || record.change_type === 'order_unlock') {
          dirLabel = '释放';
        } else if (record.change_type === 'order_lock') {
          dirLabel = '锁定';
        } else if (isPositive) {
          dirLabel = '增加';
        } else if (isNegative) {
          dirLabel = '减少';
        }

        const textColor = isPositive
          ? '#52c41a'
          : isNegative
            ? '#ff4d4f'
            : token.colorTextSecondary;
        const sign = isPositive ? '+' : '';

        return (
          <div className="flex flex-col items-end leading-tight">
            <span className="font-semibold text-base tabular-nums" style={{ color: textColor }}>
              {sign}
              {delta}
            </span>
            <span className="text-xs opacity-75 mt-0.5" style={{ color: token.colorTextTertiary }}>
              {dirLabel}
            </span>
          </div>
        );
      },
    },
    {
      title: '当前库存',
      key: 'available_stock',
      width: 110,
      align: 'right',
      render: (_, record) => {
        const avail =
          record.available_after ??
          (record.stock_after !== undefined
            ? record.stock_after - (record.locked_after ?? 0)
            : undefined);
        if (avail === undefined || avail === null) {
          return <span style={{ color: token.colorTextTertiary }}>—</span>;
        }
        const isZero = avail === 0;
        return (
          <div className="flex flex-col items-end leading-tight">
            <span
              className="font-semibold text-base tabular-nums"
              style={{ color: isZero ? '#ff4d4f' : token.colorTextHeading }}
            >
              {avail}
            </span>
            <span className="text-xs" style={{ color: token.colorTextTertiary }}>
              可用
            </span>
          </div>
        );
      },
    },
    {
      title: '变更原因',
      key: 'change_type',
      width: 150,
      render: (_, record) => {
        const conf = CHANGE_TYPE_CONFIG[record.change_type || ''] ?? {
          label: record.change_type || '其他变更',
          tagColor: 'default',
        };
        return (
          <div className="flex flex-col gap-1">
            <div>
              <Tag color={conf.tagColor} style={{ margin: 0 }}>
                {conf.label}
              </Tag>
            </div>
            {record.remark && (
              <Typography.Paragraph
                ellipsis={{ rows: 1, tooltip: record.remark }}
                className="text-xs m-0"
                style={{ color: token.colorTextTertiary, maxWidth: 140 }}
              >
                {record.remark}
              </Typography.Paragraph>
            )}
          </div>
        );
      },
    },
    {
      title: '操作信息',
      key: 'operator_info',
      width: 170,
      render: (_, record) => {
        const operatorName =
          record.operator_name || (record.operator_id === 0 ? '系统自动' : `用户 #${record.operator_id}`);
        const timeStr = record.created_at
          ? dayjs(record.created_at).isValid()
            ? dayjs(record.created_at).format('YYYY-MM-DD HH:mm')
            : record.created_at
          : '—';
        return (
          <div className="flex flex-col leading-tight">
            <span className="font-medium text-sm" style={{ color: token.colorText }}>
              {operatorName}
            </span>
            <span className="text-xs opacity-70 mt-1 tabular-nums" style={{ color: token.colorTextTertiary }}>
              {timeStr}
            </span>
          </div>
        );
      },
    },
    {
      title: '操作',
      key: 'action',
      width: 80,
      align: 'center',
      render: (_, record) => (
        <Button
          type="link"
          size="small"
          onClick={(e) => {
            e.stopPropagation();
            setSelectedRow(record);
          }}
        >
          详情
        </Button>
      ),
    },
  ];

  return (
    <div
      className="admin-list-page overflow-hidden rounded-lg"
      style={{
        backgroundColor: token.colorBgContainer,
        ...adminListVars(token),
      }}
    >
      <div className="admin-list-header">
        <PageHeader
          title="库存流水"
          description="追踪商品SKU库存的所有变更明细与业务操作记录，方便仓库核对进出与审计追溯"
        />
      </div>

      <ContentCard>
        {/* 筛选区 */}
        <Form<SearchFormValues>
          form={form}
          className="admin-list-filters !flex-col !items-stretch !gap-3"
          style={{ width: '100%' }}
          initialValues={{
            sku_id: initialSKUID,
            advanced_sku_id: initialSKUID,
            advanced_product_id: initialProductID,
          }}
          onFinish={handleSearch}
        >
          {/* 第一行：主要业务筛选 与 查询按钮 */}
          <div className="flex flex-wrap items-center justify-between gap-3 w-full">
            <div className="flex flex-wrap items-center gap-3">
              <Form.Item name="sku_id" noStyle>
                <Select
                  showSearch
                  allowClear
                  placeholder="搜索商品名称 / SKU规格 / 编码"
                  filterOption={false}
                  onSearch={handleSearchSKU}
                  loading={skuSearching}
                  notFoundContent={skuSearching ? <Spin size="small" /> : '未找到匹配商品/SKU'}
                  options={skuOptions}
                  style={{ width: 280 }}
                />
              </Form.Item>

              <Form.Item name="change_type" noStyle>
                <Select
                  allowClear
                  placeholder="变更原因"
                  style={{ width: 135 }}
                  options={[
                    { label: '全部原因', value: '' },
                    { label: '入库', value: 'manual_in' },
                    { label: '出库', value: 'manual_out' },
                    { label: '盘点调整', value: 'stocktake' },
                    { label: '设置库存预警', value: 'warning_set' },
                    { label: '订单扣减', value: 'order_out' },
                    { label: '订单释放', value: 'order_cancel' },
                  ]}
                />
              </Form.Item>

              <Form.Item name="stock_change" noStyle>
                <Select
                  allowClear
                  placeholder="库存变化"
                  style={{ width: 115 }}
                  options={[
                    { label: '全部变化', value: '' },
                    { label: '增加库存', value: 'increase' },
                    { label: '减少库存', value: 'decrease' },
                  ]}
                />
              </Form.Item>

              <Form.Item name="date_range" noStyle>
                <RangePicker style={{ width: 240 }} />
              </Form.Item>
            </div>

            <div className="flex items-center gap-2">
              <Button type="primary" htmlType="submit" icon={<SearchOutlined />}>
                查询
              </Button>
              <Button onClick={handleReset}>
                重置
              </Button>
              <Button
                type="link"
                onClick={() => setShowAdvanced((v) => !v)}
                className="flex items-center text-xs px-1"
                style={{ color: token.colorPrimary }}
              >
                {showAdvanced ? '收起高级筛选' : '高级筛选'}
                <DownOutlined
                  style={{
                    fontSize: 10,
                    marginLeft: 4,
                    transform: showAdvanced ? 'rotate(180deg)' : 'none',
                    transition: 'transform 0.2s',
                  }}
                />
              </Button>
            </div>
          </div>

          {/* 第二行：操作来源与操作人 */}
          <div className="flex flex-wrap items-center gap-3 w-full pt-1">
            <span className="text-xs font-medium" style={{ color: token.colorTextSecondary }}>
              业务维度：
            </span>
            <Form.Item name="source" noStyle>
              <Select
                allowClear
                placeholder="操作来源"
                style={{ width: 130 }}
                options={[
                  { label: '全部来源', value: '' },
                  { label: '人工调整', value: 'manual' },
                  { label: '订单业务', value: 'order' },
                  { label: '系统任务', value: 'system' },
                ]}
              />
            </Form.Item>

            <Form.Item name="operator_id" noStyle>
              <Select
                allowClear
                showSearch
                placeholder="选择操作人"
                style={{ width: 160 }}
                optionFilterProp="label"
                options={userOptions}
              />
            </Form.Item>
          </div>

          {/* 第三行：高级筛选区（收起/展开） */}
          {showAdvanced && (
            <div
              className="p-3 rounded-md border flex flex-wrap items-center gap-3 w-full text-xs mt-1"
              style={{
                backgroundColor: token.colorFillAlter,
                borderColor: token.colorBorderSecondary,
              }}
            >
              <span className="font-medium" style={{ color: token.colorTextSecondary }}>
                精确 ID 筛选：
              </span>
              <Form.Item name="advanced_sku_id" noStyle>
                <InputNumber min={1} precision={0} placeholder="精确 SKU ID" style={{ width: 120 }} />
              </Form.Item>
              <Form.Item name="advanced_product_id" noStyle>
                <InputNumber min={1} precision={0} placeholder="商品 ID" style={{ width: 120 }} />
              </Form.Item>
              <Form.Item name="request_id" noStyle>
                <Input placeholder="操作编号 (Request UUID)" allowClear style={{ width: 220 }} />
              </Form.Item>
              <Form.Item name="advanced_operator_id" noStyle>
                <InputNumber min={0} precision={0} placeholder="操作者 ID" style={{ width: 120 }} />
              </Form.Item>
            </div>
          )}
        </Form>

        {/* 表格工具栏 */}
        <div className="admin-list-toolbar">
          <div className="flex flex-wrap items-center gap-3">
            <h2
              className="m-0 flex items-center gap-2 text-base font-semibold"
              style={{ color: token.colorTextHeading }}
            >
              <HistoryOutlined style={{ color: token.colorPrimary }} />
              流水明细记录
            </h2>
            <span className="text-sm" style={{ color: token.colorTextTertiary }}>
              共 {total} 条流水
              {displayRows.length !== rows.length
                ? `（当前页筛选显示 ${displayRows.length} 条）`
                : ''}
            </span>
          </div>
          <Tooltip title="刷新列表">
            <Button
              type="text"
              icon={<ReloadOutlined spin={loading} />}
              onClick={() => void loadLogs()}
              style={{ color: token.colorTextSecondary }}
            >
              刷新
            </Button>
          </Tooltip>
        </div>

        {/* 表格数据 */}
        <div className="px-4">
          <Table<ProductInventoryLogRow>
            rowKey="id"
            columns={columns}
            dataSource={displayRows}
            loading={loading}
            className="admin-list-table"
            size="middle"
            onRow={(record) => ({
              onClick: () => setSelectedRow(record),
              className: 'cursor-pointer hover:bg-black/5 dark:hover:bg-white/5 transition-colors',
            })}
            pagination={false}
          />
        </div>

        {/* 分页 */}
        <div className="flex flex-wrap items-center justify-between gap-4 px-6 py-4">
          <div
            className="flex items-center gap-2 text-sm"
            style={{ color: token.colorTextSecondary }}
          >
            <span>每页显示</span>
            <Select
              aria-label="每页条数"
              className="w-18"
              size="small"
              value={pageSize}
              options={[10, 20, 50, 100].map((v) => ({
                value: v,
                label: v,
              }))}
              onChange={(size) => {
                setPage(1);
                setPageSize(size);
              }}
            />
            <span>条</span>
          </div>
          <Pagination
            className="admin-list-pagination"
            current={page}
            pageSize={pageSize}
            total={total}
            showSizeChanger={false}
            showQuickJumper
            locale={{ jump_to: '前往', page: '页' }}
            onChange={(p, ps) => {
              setPage(p);
              if (ps && ps !== pageSize) {
                setPageSize(ps);
              }
            }}
          />
        </div>
      </ContentCard>

      {/* 详情 Drawer */}
      <Drawer
        title={
          <div className="flex items-center justify-between pr-4">
            <div className="flex items-center gap-2">
              <HistoryOutlined style={{ color: token.colorPrimary }} />
              <span>库存流水详情</span>
            </div>
            {selectedRow && (
              <Tag color="blue" style={{ margin: 0 }}>
                流水 ID: {selectedRow.id}
              </Tag>
            )}
          </div>
        }
        width={560}
        open={!!selectedRow}
        onClose={() => setSelectedRow(null)}
        destroyOnClose
      >
        {selectedRow && (() => {
          const snap = selectedRow.sku_snapshot;
          const specs = snap?.specs ?? [];
          const conf = CHANGE_TYPE_CONFIG[selectedRow.change_type || ''] ?? {
            label: selectedRow.change_type || '其他变更',
            tagColor: 'default',
          };
          const delta = selectedRow.stock_delta ?? 0;
          const isPositive = delta > 0;
          const isNegative = delta < 0;

          return (
            <div className="flex flex-col gap-6">
              {/* 1. 商品与 SKU */}
              <div>
                <h4 className="text-sm font-semibold mb-3 flex items-center gap-1.5" style={{ color: token.colorTextHeading }}>
                  <AppstoreOutlined style={{ color: token.colorPrimary }} />
                  商品与规格信息
                </h4>
                <Descriptions size="small" bordered column={1}>
                  <Descriptions.Item label="商品名称">
                    <span className="font-medium">{snap?.product_name || '—'}</span>
                  </Descriptions.Item>
                  <Descriptions.Item label="SKU 规格">
                    <Space size={4} wrap>
                      {specs.length === 0 ? (
                        <Tag color="default">默认规格</Tag>
                      ) : (
                        specs.map((s) => (
                          <Tag key={`${s.spec_id}-${s.value_id}`} color="blue">
                            {s.spec_name}: {s.value_name}
                          </Tag>
                        ))
                      )}
                    </Space>
                  </Descriptions.Item>
                  <Descriptions.Item label="SKU 编码">
                    <span className="font-mono">{snap?.sku_code || '—'}</span>
                  </Descriptions.Item>
                  <Descriptions.Item label="计量单位">
                    {snap?.unit || '件'}
                  </Descriptions.Item>
                  <Descriptions.Item label="商品 ID">
                    <span className="font-mono">{selectedRow.product_id}</span>
                  </Descriptions.Item>
                  <Descriptions.Item label="SKU ID">
                    <span className="font-mono">{selectedRow.sku_id}</span>
                  </Descriptions.Item>
                </Descriptions>
              </div>

              {/* 2. 变更明细 */}
              <div>
                <h4 className="text-sm font-semibold mb-3 flex items-center gap-1.5" style={{ color: token.colorTextHeading }}>
                  <HistoryOutlined style={{ color: '#52c41a' }} />
                  库存变动详情
                </h4>
                <Descriptions size="small" bordered column={2}>
                  <Descriptions.Item label="变更原因" span={2}>
                    <Tag color={conf.tagColor}>{conf.label}</Tag>
                    <span className="text-xs font-mono" style={{ color: token.colorTextTertiary }}>
                      ({selectedRow.change_type})
                    </span>
                  </Descriptions.Item>
                  <Descriptions.Item label="库存变化">
                    <span
                      className="font-bold tabular-nums text-base"
                      style={{
                        color: isPositive ? '#52c41a' : isNegative ? '#ff4d4f' : token.colorText,
                      }}
                    >
                      {isPositive ? `+${delta}` : delta}
                    </span>
                  </Descriptions.Item>
                  <Descriptions.Item label="变动数量">
                    <span className="tabular-nums">{selectedRow.quantity ?? '—'}</span>
                  </Descriptions.Item>
                  <Descriptions.Item label="变动前总库存">
                    <span className="tabular-nums">{selectedRow.stock_before}</span>
                  </Descriptions.Item>
                  <Descriptions.Item label="变动后总库存">
                    <span className="tabular-nums font-semibold">{selectedRow.stock_after}</span>
                  </Descriptions.Item>
                  <Descriptions.Item label="可用库存">
                    <span className="tabular-nums">
                      {selectedRow.available_before ?? '—'} →{' '}
                      <strong style={{ color: token.colorPrimary }}>
                        {selectedRow.available_after ?? '—'}
                      </strong>
                    </span>
                  </Descriptions.Item>
                  <Descriptions.Item label="锁定库存">
                    <span className="tabular-nums">
                      {selectedRow.locked_before ?? 0} →{' '}
                      <strong style={{ color: '#faad14' }}>
                        {selectedRow.locked_after ?? 0}
                      </strong>
                    </span>
                  </Descriptions.Item>
                  <Descriptions.Item label="预警阈值" span={2}>
                    {selectedRow.change_type === 'warning_set' ? (
                      <Tag color="cyan">
                        {selectedRow.warning_before === 0 ? '未开启' : selectedRow.warning_before} →{' '}
                        {selectedRow.warning_after === 0 ? '已关闭' : selectedRow.warning_after}
                      </Tag>
                    ) : (
                      <span>
                        {selectedRow.warning_after === 0 ? '未开启' : selectedRow.warning_after}
                      </span>
                    )}
                  </Descriptions.Item>
                  <Descriptions.Item label="完整备注说明" span={2}>
                    {selectedRow.remark || <span style={{ color: token.colorTextTertiary }}>无备注</span>}
                  </Descriptions.Item>
                </Descriptions>
              </div>

              {/* 3. 审计与追踪信息 */}
              <div>
                <h4 className="text-sm font-semibold mb-3 flex items-center gap-1.5" style={{ color: token.colorTextHeading }}>
                  <InfoCircleOutlined style={{ color: '#722ed1' }} />
                  操作与审计追踪
                </h4>
                <Descriptions size="small" bordered column={1}>
                  <Descriptions.Item label="操作人">
                    <span>
                      {selectedRow.operator_name || (selectedRow.operator_id === 0 ? '系统自动' : '未知')}
                      {selectedRow.operator_id ? ` (ID: ${selectedRow.operator_id})` : ''}
                    </span>
                  </Descriptions.Item>
                  <Descriptions.Item label="操作时间">
                    <span className="tabular-nums">
                      {formatTime(selectedRow.created_at)}
                    </span>
                  </Descriptions.Item>
                  <Descriptions.Item label="版本变更">
                    <span className="font-mono">
                      v{selectedRow.version_before} → v{selectedRow.version_after}
                    </span>
                  </Descriptions.Item>
                  <Descriptions.Item label="库存记录 ID">
                    <span className="font-mono">{selectedRow.inventory_id}</span>
                  </Descriptions.Item>
                  <Descriptions.Item label="操作编号 (Request ID)">
                    <Typography.Text copyable className="font-mono text-xs">
                      {selectedRow.request_id}
                    </Typography.Text>
                  </Descriptions.Item>
                </Descriptions>
              </div>
            </div>
          );
        })()}
      </Drawer>
    </div>
  );
}
