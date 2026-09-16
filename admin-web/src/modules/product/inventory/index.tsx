import { useCallback, useEffect, useRef, useState } from "react";
import {
  Alert,
  Button,
  Form,
  Image,
  Input,
  Pagination,
  Select,
  Space,
  Table,
  Tag,
  Tooltip,
  Typography,
  message,
  theme,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import {
  HistoryOutlined,
  InboxOutlined,
  ReloadOutlined,
  SearchOutlined,
} from "@ant-design/icons";
import { useNavigate, useSearchParams } from "react-router-dom";

import { api } from "@/framework/http";
import type {
  ProductInventoryListRequest,
  ProductInventoryRow,
} from "@/framework/http/generated_api";
import { usePermission } from "@/framework/store";
import { formatTimeTwoLines } from "@/shared/utils/format";
import { ContentCard, PageHeader, adminListVars } from "@/shared/admin-ui";
import { AdjustModal, type AdjustMode } from "./adjust-modal";

interface SearchFormValues {
  keyword?: string;
  product_status?: number;
  sku_status?: number;
  stock_filter?: string;
}

export default function ProductInventoryListPage() {
  const { token } = theme.useToken();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const hasPermission = usePermission();

  const canList = hasPermission("product:inventory:list");
  const canAdjust = hasPermission("product:inventory:adjust");
  const canWarning = hasPermission("product:inventory:warning");
  const canLog = hasPermission("product:inventory:log");

  const [form] = Form.useForm<SearchFormValues>();

  const [rows, setRows] = useState<ProductInventoryRow[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);

  // Pagination state
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  // Deep linked ID filters from URL
  const paramProductID = searchParams.get("product_id");
  const paramSKUID = searchParams.get("sku_id");

  const productIDFilter =
    paramProductID && /^\d+$/.test(paramProductID)
      ? Number(paramProductID)
      : undefined;
  const skuIDFilter =
    paramSKUID && /^\d+$/.test(paramSKUID) ? Number(paramSKUID) : undefined;

  // Active query parameters
  const [activeQuery, setActiveQuery] = useState<SearchFormValues>({
    keyword: "",
    product_status: undefined,
    sku_status: undefined,
    stock_filter: "all",
  });

  // Adjust Modal state
  const [modalOpen, setModalOpen] = useState(false);
  const [modalMode, setModalMode] = useState<AdjustMode>("manual_in");
  const [targetSKUId, setTargetSKUId] = useState<number | null>(null);

  const reqSeqRef = useRef(0);

  // Load inventory list
  const loadInventories = useCallback(async () => {
    if (!canList) return;
    const currentSeq = ++reqSeqRef.current;
    setLoading(true);

    const req: ProductInventoryListRequest = {
      page,
      page_size: pageSize,
      keyword: activeQuery.keyword?.trim() || undefined,
      product_id: productIDFilter,
      sku_id: skuIDFilter,
      product_status: activeQuery.product_status,
      sku_status: activeQuery.sku_status,
      stock_filter:
        activeQuery.stock_filter === "all"
          ? undefined
          : activeQuery.stock_filter,
    };

    const res = await api.api.listProductInventories(req);

    if (currentSeq !== reqSeqRef.current) return;
    setLoading(false);

    if (res.code === 1) {
      setRows((res.data as ProductInventoryRow[]) ?? []);
      setTotal(res.total ?? 0);
    } else {
      message.error(res.message || "加载库存列表失败");
    }
  }, [canList, page, pageSize, activeQuery, productIDFilter, skuIDFilter]);

  useEffect(() => {
    void loadInventories();
  }, [loadInventories]);

  // Handle Search submit
  const handleSearch = (values: SearchFormValues) => {
    setPage(1);
    setActiveQuery(values);
  };

  // Handle Reset filters
  const handleReset = () => {
    form.resetFields();
    setSearchParams({}, { replace: true });
    setPage(1);
    setActiveQuery({
      keyword: "",
      product_status: undefined,
      sku_status: undefined,
      stock_filter: "all",
    });
  };

  // Clear specific URL query param
  const handleClearParam = (key: "product_id" | "sku_id") => {
    const newParams = new URLSearchParams(searchParams);
    newParams.delete(key);
    setSearchParams(newParams, { replace: true });
    setPage(1);
  };

  // Open Adjust Modal
  const openAdjustModal = (mode: AdjustMode, skuId: number) => {
    setModalMode(mode);
    setTargetSKUId(skuId);
    setModalOpen(true);
  };

  // Columns definition
  const columns: ColumnsType<ProductInventoryRow> = [
    {
      title: "商品信息",
      key: "product",
      width: 260,
      render: (_, record) => (
        <Space align="start" size="middle">
          <Image
            src={record.image_url || undefined}
            width={52}
            height={52}
            style={{
              objectFit: "cover",
              borderRadius: 4,
              background: "#f5f5f5",
            }}
            fallback="data:image/svg+xml;utf8,<svg xmlns='http://www.w3.org/2000/svg' width='52' height='52' viewBox='0 0 52 52'><rect width='52' height='52' fill='%23f0f0f0'/><text x='50%25' y='50%25' font-size='10' text-anchor='middle' alignment-baseline='middle' fill='%23999'>无图片</text></svg>"
          />
          <Space direction="vertical" size={2}>
            <Typography.Text
              strong
              style={{ fontSize: 13, lineHeight: "18px" }}
              ellipsis={{ tooltip: record.product_name }}
            >
              {record.product_name}
            </Typography.Text>
            <Space size={4}>
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                商品ID: {record.product_id}
              </Typography.Text>
              {record.product_status === 1 ? (
                <Tag
                  color="success"
                  style={{ margin: 0, fontSize: 11, lineHeight: "18px" }}
                >
                  上架中
                </Tag>
              ) : (
                <Tag
                  color="default"
                  style={{ margin: 0, fontSize: 11, lineHeight: "18px" }}
                >
                  已下架
                </Tag>
              )}
            </Space>
          </Space>
        </Space>
      ),
    },
    {
      title: "SKU规格 / 编码",
      key: "sku",
      width: 220,
      render: (_, record) => {
        const specs = record.specs ?? [];
        return (
          <Space orientation="vertical" size={2} style={{ width: "100%" }}>
            <Space size={4} wrap>
              {specs.length === 0 ? (
                <Tag color="default" style={{ margin: 0 }}>
                  默认规格
                </Tag>
              ) : (
                specs.map((s) => (
                  <Tag
                    key={`${s.spec_id}-${s.value_id}`}
                    color="blue"
                    style={{ margin: 0 }}
                  >
                    {s.spec_name}: {s.value_name}
                  </Tag>
                ))
              )}
              {record.sku_status === 0 && (
                <Tag color="warning" style={{ margin: 0 }}>
                  已禁用
                </Tag>
              )}
            </Space>
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              编码: {record.sku_code || "—"}
            </Typography.Text>
            {record.barcode && (
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                条码: {record.barcode}
              </Typography.Text>
            )}
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              SKU ID: {record.sku_id} | 单位: {record.unit || "件"}
            </Typography.Text>
          </Space>
        );
      },
    },
    {
      title: "总库存",
      dataIndex: "stock",
      key: "stock",
      width: 100,
      align: "right",
      render: (stock: number | null, record) => {
        if (!record.initialized || stock === null) {
          return <Tag color="warning">未初始化</Tag>;
        }
        return (
          <Typography.Text strong style={{ fontSize: 14 }}>
            {stock}
          </Typography.Text>
        );
      },
    },
    {
      title: "锁定库存",
      dataIndex: "locked_stock",
      key: "locked_stock",
      width: 100,
      align: "right",
      render: (locked: number | null, record) => {
        if (!record.initialized || locked === null) {
          return <Typography.Text type="secondary">—</Typography.Text>;
        }
        return (
          <Typography.Text type={locked > 0 ? "warning" : "secondary"}>
            {locked}
          </Typography.Text>
        );
      },
    },
    {
      title: "可用库存",
      key: "available_stock",
      width: 130,
      align: "right",
      render: (_, record) => {
        if (
          !record.initialized ||
          record.available_stock === null ||
          record.available_stock === undefined
        ) {
          return <Typography.Text type="secondary">—</Typography.Text>;
        }
        const avail = record.available_stock;
        return (
          <Space direction="vertical" size={2} align="end">
            <Typography.Text
              strong
              style={{
                fontSize: 14,
                color:
                  avail === 0
                    ? "#ff4d4f"
                    : record.is_low_stock
                      ? "#faad14"
                      : "#52c41a",
              }}
            >
              {avail}
            </Typography.Text>
            <Space size={2}>
              {record.is_out_of_stock && (
                <Tag color="error" style={{ margin: 0, fontSize: 10 }}>
                  缺货
                </Tag>
              )}
              {record.is_low_stock && (
                <Tag color="warning" style={{ margin: 0, fontSize: 10 }}>
                  低库存
                </Tag>
              )}
            </Space>
          </Space>
        );
      },
    },
    {
      title: "预警阈值",
      dataIndex: "warning_stock",
      key: "warning_stock",
      width: 100,
      align: "center",
      render: (warning: number | null, record) => {
        if (!record.initialized || warning === null) {
          return <Typography.Text type="secondary">—</Typography.Text>;
        }
        if (warning === 0) {
          return <Tag color="default">未开启</Tag>;
        }
        return <Tag color="cyan">{warning}</Tag>;
      },
    },
    {
      title: "更新时间",
      dataIndex: "updated_at",
      key: "updated_at",
      width: 120,
      render: (t?: string | null) => formatTimeTwoLines(t),
    },
    {
      title: "操作",
      key: "action",
      width: 230,
      fixed: "right",
      render: (_, record) => {
        if (!record.sku_id || !record.initialized) {
          return (
            <Tooltip title="该SKU未生成库存余额，请联系管理员修复数据">
              <Typography.Text type="secondary">未初始化</Typography.Text>
            </Tooltip>
          );
        }

        const skuId = record.sku_id;

        return (
          <Space size="small">
            {canAdjust && (
              <>
                <Button
                  type="link"
                  size="small"
                  onClick={() => openAdjustModal("manual_in", skuId)}
                >
                  入库
                </Button>
                <Button
                  type="link"
                  size="small"
                  onClick={() => openAdjustModal("manual_out", skuId)}
                >
                  出库
                </Button>
                <Button
                  type="link"
                  size="small"
                  onClick={() => openAdjustModal("stocktake", skuId)}
                >
                  盘点
                </Button>
              </>
            )}
            {canWarning && (
              <Button
                type="link"
                size="small"
                onClick={() => openAdjustModal("warning", skuId)}
              >
                设置预警
              </Button>
            )}
            {canLog && (
              <Button
                type="link"
                size="small"
                onClick={() =>
                  navigate(`/product/inventory_log?sku_id=${skuId}`)
                }
              >
                流水
              </Button>
            )}
          </Space>
        );
      },
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
          title="库存管理"
          description="管理商品SKU库存、安全预警与出入库盘点调整，支持快速追溯变动流水"
        />
      </div>

      <ContentCard>
        {/* Active deep link parameter notice */}
        {(productIDFilter || skuIDFilter) && (
          <div className="px-6 pt-4">
            <Alert
              type="info"
              showIcon
              message={
                <Space>
                  <span>当前处于特定条件过滤模式：</span>
                  {productIDFilter && (
                    <Tag
                      closable
                      onClose={() => handleClearParam("product_id")}
                      color="purple"
                    >
                      商品ID: {productIDFilter}
                    </Tag>
                  )}
                  {skuIDFilter && (
                    <Tag
                      closable
                      onClose={() => handleClearParam("sku_id")}
                      color="cyan"
                    >
                      SKU ID: {skuIDFilter}
                    </Tag>
                  )}
                </Space>
              }
            />
          </div>
        )}

        {/* 筛选区 */}
        <Form<SearchFormValues>
          form={form}
          className="admin-list-filters"
          initialValues={activeQuery}
          onFinish={handleSearch}
        >
          <Form.Item name="keyword" noStyle>
            <Input
              placeholder="商品名称 / SKU编码 / 条码"
              allowClear
              prefix={
                <SearchOutlined style={{ color: token.colorTextSecondary }} />
              }
              style={{ width: 220 }}
            />
          </Form.Item>
          <Form.Item name="stock_filter" noStyle>
            <Select
              style={{ width: 140 }}
              options={[
                { label: "全部库存", value: "all" },
                { label: "正常库存", value: "normal" },
                { label: "低库存预警", value: "low" },
                { label: "已缺货 (0件)", value: "empty" },
                { label: "未初始化", value: "uninitialized" },
              ]}
            />
          </Form.Item>
          <Form.Item name="product_status" noStyle>
            <Select
              allowClear
              placeholder="商品状态"
              style={{ width: 120 }}
              options={[
                { label: "上架", value: 1 },
                { label: "下架", value: 0 },
              ]}
            />
          </Form.Item>
          <Form.Item name="sku_status" noStyle>
            <Select
              allowClear
              placeholder="SKU状态"
              style={{ width: 120 }}
              options={[
                { label: "启用", value: 1 },
                { label: "禁用", value: 0 },
              ]}
            />
          </Form.Item>
          <Button type="primary" htmlType="submit" icon={<SearchOutlined />}>
            查询
          </Button>
          <Button onClick={handleReset}>重置</Button>
        </Form>

        {/* 表格工具栏 */}
        <div className="admin-list-toolbar">
          <div className="flex flex-wrap items-center gap-3">
            <h2
              className="m-0 flex items-center gap-2 text-base font-semibold"
              style={{ color: token.colorTextHeading }}
            >
              <InboxOutlined style={{ color: token.colorPrimary }} />
              商品SKU库存列表
            </h2>
            <span
              className="text-sm"
              style={{ color: token.colorTextTertiary }}
            >
              共 {total} 条记录
            </span>
          </div>
          <Tooltip title="刷新列表">
            <Button
              type="text"
              icon={<ReloadOutlined spin={loading} />}
              onClick={() => void loadInventories()}
              style={{ color: token.colorTextSecondary }}
            >
              刷新
            </Button>
          </Tooltip>
        </div>

        {/* 表格数据 */}
        <div className="px-4">
          <Table<ProductInventoryRow>
            rowKey="sku_id"
            columns={columns}
            dataSource={rows}
            loading={loading}
            scroll={{ x: 1200 }}
            className="admin-list-table"
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
            locale={{ jump_to: "前往", page: "页" }}
            onChange={(p, ps) => {
              setPage(p);
              if (ps && ps !== pageSize) {
                setPageSize(ps);
              }
            }}
          />
        </div>
      </ContentCard>

      {/* Adjust Modal */}
      <AdjustModal
        open={modalOpen}
        mode={modalMode}
        skuId={targetSKUId}
        onClose={() => setModalOpen(false)}
        onSuccess={() => void loadInventories()}
      />
    </div>
  );
}
