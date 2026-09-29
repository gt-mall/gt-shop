import type { Key } from "react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  Button,
  DatePicker,
  Dropdown,
  Image,
  Input,
  Modal,
  Pagination,
  Select,
  Space,
  Switch,
  Table,
  Tabs,
  Tag,
  theme,
  Tooltip,
  TreeSelect,
  message,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import type { MenuProps } from "antd";
import {
  DeleteOutlined,
  DownOutlined,
  FilterOutlined,
  PlusOutlined,
  ShoppingOutlined,
  ReloadOutlined,
  SearchOutlined,
  UpOutlined,
} from "@ant-design/icons";
import dayjs from "dayjs";
import type { Dayjs } from "dayjs";
import { Link, useNavigate } from "react-router-dom";

import { api } from "@/framework/http";
import type { ProductCategoryOption, ProductRow } from "@/framework/http";
import { usePermission } from "@/framework/store";
import { ContentCard, PageHeader, adminListVars } from "@/shared/admin-ui";
import { formatTimeTwoLines } from "@/shared/utils/format";
import { buildCategoryOptionTree } from "../types";
import "./list.less";

interface SearchState {
  keyword: string;
  category_id?: number;
  status?: number;
  product_code: string;
  sku_code: string;
  freight_type?: number;
  min_price: string;
  max_price: string;
  min_stock?: number;
  max_stock?: number;
  min_sales?: number;
  max_sales?: number;
  created_at?: [Dayjs, Dayjs] | null;
  updated_at?: [Dayjs, Dayjs] | null;
}

const initialSearchState: SearchState = {
  keyword: "",
  category_id: undefined,
  status: undefined,
  product_code: "",
  sku_code: "",
  freight_type: undefined,
  min_price: "",
  max_price: "",
  min_stock: undefined,
  max_stock: undefined,
  min_sales: undefined,
  max_sales: undefined,
  created_at: null,
  updated_at: null,
};

export default function ProductListPage() {
  const navigate = useNavigate();
  const { token } = theme.useToken();
  const hasPermission = usePermission();
  const canCreate = hasPermission("product:product:create");
  const canUpdate = hasPermission("product:product:update");
  const canDelete = hasPermission("product:product:delete");
  const canViewInventory = hasPermission("product:inventory:list");

  const [rows, setRows] = useState<ProductRow[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);

  // Pagination state
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [selectedRowKeys, setSelectedRowKeys] = useState<Key[]>([]);

  // Filter expand toggle
  const [isAdvancedOpen, setIsAdvancedOpen] = useState(false);

  // Search input state
  const [searchParams, setSearchParams] = useState<SearchState>({
    ...initialSearchState,
  });

  // Active query state used for API requests
  const [activeQuery, setActiveQuery] = useState<SearchState>({
    ...initialSearchState,
  });

  // Category options
  const [categoryOptions, setCategoryOptions] = useState<
    ProductCategoryOption[]
  >([]);
  const reqSeqRef = useRef(0);

  type ProductTabKey = "all" | "selling" | "warning" | "sold_out" | "warehouse";

  const [activeTab, setActiveTab] = useState<ProductTabKey>("all");
  const [tabCounts, setTabCounts] = useState<{
    all: number;
    selling: number;
    warning: number;
    sold_out: number;
    warehouse: number;
  }>({
    all: 0,
    selling: 0,
    warning: 0,
    sold_out: 0,
    warehouse: 0,
  });

  // Load category options for filter
  const loadCategoryOptions = useCallback(async () => {
    const res = await api.api.getProductCategoryOptions();
    if (res.code === 1) {
      setCategoryOptions(res.data ?? []);
    }
  }, []);

  const loadTabCounts = useCallback(async () => {
    const res = await api.api.getProductTabCounts();
    if (res.code === 1 && res.data) {
      setTabCounts({
        all: res.data.all ?? 0,
        selling: res.data.selling ?? 0,
        warning: res.data.warning ?? 0,
        sold_out: res.data.sold_out ?? 0,
        warehouse: res.data.warehouse ?? 0,
      });
    }
  }, []);

  useEffect(() => {
    void loadCategoryOptions();
    void loadTabCounts();
  }, [loadCategoryOptions, loadTabCounts]);

  const handleTabChange = useCallback((key: string) => {
    setActiveTab(key as ProductTabKey);
    setPage(1);
  }, []);

  // Load product list
  const loadProducts = useCallback(async () => {
    const currentSeq = ++reqSeqRef.current;
    setLoading(true);

    const res = await api.api.listProducts({
      page,
      page_size: pageSize,
      tab: activeTab === "all" ? undefined : activeTab,
      keyword: activeQuery.keyword.trim() || undefined,
      category_id: activeQuery.category_id,
      product_code: activeQuery.product_code.trim() || undefined,
      sku_code: activeQuery.sku_code.trim() || undefined,
      freight_type: activeQuery.freight_type,
      min_price: activeQuery.min_price.trim() || undefined,
      max_price: activeQuery.max_price.trim() || undefined,
      min_stock: activeQuery.min_stock,
      max_stock: activeQuery.max_stock,
      min_sales: activeQuery.min_sales,
      max_sales: activeQuery.max_sales,
      created_at_start: activeQuery.created_at?.[0]
        ? activeQuery.created_at[0].startOf("day").format("YYYY-MM-DD HH:mm:ss")
        : undefined,
      created_at_end: activeQuery.created_at?.[1]
        ? activeQuery.created_at[1].endOf("day").format("YYYY-MM-DD HH:mm:ss")
        : undefined,
      updated_at_start: activeQuery.updated_at?.[0]
        ? activeQuery.updated_at[0].startOf("day").format("YYYY-MM-DD HH:mm:ss")
        : undefined,
      updated_at_end: activeQuery.updated_at?.[1]
        ? activeQuery.updated_at[1].endOf("day").format("YYYY-MM-DD HH:mm:ss")
        : undefined,
    });

    if (currentSeq !== reqSeqRef.current) return;
    setLoading(false);

    if (res.code === 1) {
      setSelectedRowKeys([]);
      setRows(res.data ?? []);
      setTotal(res.total ?? 0);
    } else {
      message.error(res.message || "加载商品列表失败");
    }
  }, [page, pageSize, activeQuery, activeTab]);

  useEffect(() => {
    void loadProducts();
  }, [loadProducts]);

  // Category TreeSelect Data
  const categoryTreeData = useMemo(() => {
    const tree = buildCategoryOptionTree(categoryOptions);
    return [
      { key: "all", value: undefined, title: "全部分类" },
      { key: "uncategorized", value: 0, title: "未分类" },
      ...tree,
    ];
  }, [categoryOptions]);

  // Search and reset handlers
  const handleSearch = () => {
    setPage(1);
    setActiveQuery({ ...searchParams });
  };

  const handleReset = () => {
    setSearchParams({ ...initialSearchState });
    setActiveQuery({ ...initialSearchState });
    setPage(1);
  };

  // Toggle status
  const handleToggleStatus = useCallback(
    async (record: ProductRow) => {
      if (!record.id) return;
      const nextStatus = record.status === 1 ? 0 : 1;
      const res = await api.api.updateProductStatus({
        id: record.id,
        status: nextStatus,
      });
      if (res.code === 1) {
        message.success(nextStatus === 1 ? "商品已上架" : "商品已下架");
        void loadProducts();
        void loadTabCounts();
      } else {
        message.error(res.message || "修改状态失败");
      }
    },
    [loadProducts, loadTabCounts],
  );

  // Delete product
  const handleDelete = useCallback(
    async (record: ProductRow) => {
      if (!record.id) return;
      const res = await api.api.deleteProduct({ id: record.id });
      if (res.code === 1) {
        message.success("商品删除成功");
        void loadTabCounts();
        if (rows.length === 1 && page > 1) {
          setPage(page - 1);
        } else {
          void loadProducts();
        }
      } else {
        message.error(res.message || "删除商品失败");
      }
    },
    [rows.length, page, loadProducts, loadTabCounts],
  );

  // Confirmation dialog for delete
  const confirmDelete = useCallback(
    (record: ProductRow) => {
      Modal.confirm({
        title: "确定删除该商品？",
        content: "删除后该商品及所有SKU将移入回收站。",
        okText: "确定",
        okType: "danger",
        cancelText: "取消",
        onOk: () => handleDelete(record),
      });
    },
    [handleDelete],
  );

  const columns: ColumnsType<ProductRow> = useMemo(
    () => [
      {
        title: "商品信息",
        key: "product_info",
        width: 328,
        render: (_, record) => (
          <div className="flex items-center gap-4">
            <div
              className="w-[50px] h-[50px] rounded-md shrink-0 flex items-center justify-center overflow-hidden"
              style={{
                backgroundColor: token.colorFillAlter,
                border: `1px solid ${token.colorBorderSecondary}`,
              }}
            >
              {record.main_image_url ? (
                <Image
                  src={record.main_image_url}
                  alt={record.name}
                  width={50}
                  height={50}
                  style={{ objectFit: "cover" }}
                />
              ) : (
                <span
                  style={{ color: token.colorTextQuaternary, fontSize: 12 }}
                >
                  无图
                </span>
              )}
            </div>
            <div className="min-w-0 flex-1">
              <Link
                to={`/product/product/${record.id}`}
                className="font-semibold text-sm leading-snug truncate transition-colors block hover:underline"
                style={{ color: token.colorTextHeading }}
                title={record.name}
              >
                {record.name}
              </Link>
              {record.subtitle && (
                <div
                  className="truncate text-sm mt-1 leading-normal"
                  style={{ color: token.colorTextTertiary }}
                  title={record.subtitle}
                >
                  {record.subtitle}
                </div>
              )}
            </div>
          </div>
        ),
      },
      {
        title: "商品分类",
        dataIndex: "category_name",
        key: "category_name",
        width: 116,
        render: (catName: string) => (
          <span className="text-sm" style={{ color: token.colorTextSecondary }}>
            {catName || "—"}
          </span>
        ),
      },
      {
        title: "售价区间",
        key: "price",
        width: 148,
        render: (_, record) => {
          if (!record.min_price && !record.max_price) {
            return <span style={{ color: token.colorTextQuaternary }}>—</span>;
          }
          const priceText =
            record.min_price === record.max_price
              ? `¥${record.min_price}`
              : `¥${record.min_price} ~ ¥${record.max_price}`;
          return (
            <span
              className="font-medium text-sm tabular-nums"
              style={{ color: token.colorTextHeading }}
            >
              {priceText}
            </span>
          );
        },
      },
      {
        title: "SKU数",
        dataIndex: "sku_count",
        key: "sku_count",
        width: 70,
        align: "center",
        render: (count: number) => (
          <span
            className="tabular-nums text-sm"
            style={{ color: token.colorTextSecondary }}
          >
            {count ?? 0}
          </span>
        ),
      },
      {
        title: "销量",
        key: "sales",
        width: 100,
        render: (_, record) => (
          <div className="space-y-0.5 text-sm tabular-nums">
            <div className="flex items-center gap-1">
              <span style={{ color: token.colorTextSecondary }}>实际:</span>
              <span
                className="font-medium"
                style={{ color: token.colorTextHeading }}
              >
                {record.sales_count ?? 0}
              </span>
            </div>
            <div
              className="flex items-center gap-1"
              style={{ color: token.colorTextTertiary }}
            >
              <span>虚拟:</span>
              <span>{record.virtual_sales ?? 0}</span>
            </div>
          </div>
        ),
      },
      {
        title: "状态",
        dataIndex: "status",
        key: "status",
        width: 85,
        align: "center",
        render: (status: number, record) =>
          canUpdate ? (
            <Switch
              checked={status === 1}
              checkedChildren="上架"
              unCheckedChildren="下架"
              onChange={() => handleToggleStatus(record)}
            />
          ) : status === 1 ? (
            <Tag bordered={false} color="success" className="m-0 text-xs">
              已上架
            </Tag>
          ) : (
            <Tag
              bordered={false}
              className="m-0 text-xs"
              style={{ color: token.colorTextTertiary }}
            >
              已下架
            </Tag>
          ),
      },
      {
        title: "排序",
        dataIndex: "sort",
        key: "sort",
        width: 65,
        align: "center",
        render: (s: number) => (
          <span
            className="tabular-nums text-sm"
            style={{ color: token.colorTextTertiary }}
          >
            {s ?? 0}
          </span>
        ),
      },
      {
        title: "更新时间",
        dataIndex: "updated_at",
        key: "updated_at",
        width: 120,
        render: (t: string) =>
          formatTimeTwoLines(t, { style: { color: token.colorTextTertiary } }),
      },
      {
        title: "操作",
        key: "action",
        width: 156,
        fixed: "right",
        render: (_, record) => {
          const menuItems: MenuProps["items"] = [];
          if (canDelete) {
            menuItems.push({
              key: "delete",
              label: "删除",
              danger: true,
              icon: <DeleteOutlined />,
              onClick: () => confirmDelete(record),
            });
          }

          const hasMore = menuItems.length > 0;

          return (
            <Space size={12} className="admin-list-actions">
              {canUpdate && (
                <Button
                  type="link"
                  size="small"
                  className="p-0 text-sm"
                  onClick={() => navigate(`/product/product/${record.id}/edit`)}
                >
                  编辑
                </Button>
              )}
              {canViewInventory && (
                <Button
                  type="link"
                  size="small"
                  className="p-0 text-sm"
                  onClick={() =>
                    navigate(`/product/inventory?product_id=${record.id}`)
                  }
                >
                  库存
                </Button>
              )}
              {hasMore && (
                <Dropdown
                  menu={{ items: menuItems }}
                  trigger={["click"]}
                  placement="bottomRight"
                >
                  <Button
                    type="link"
                    size="small"
                    className="p-0 text-sm flex items-center gap-1"
                  >
                    更多
                    <DownOutlined className="text-[9px]" />
                  </Button>
                </Dropdown>
              )}
            </Space>
          );
        },
      },
    ],
    [
      token,
      canUpdate,
      canViewInventory,
      canDelete,
      navigate,
      confirmDelete,
      handleToggleStatus,
    ],
  );

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
          title="商品管理"
          description="管理和维护您的商品信息，支持多条件筛选、批量操作，提升运营效率。"
          extra={
            canCreate && (
              <Button
                type="primary"
                icon={<PlusOutlined />}
                onClick={() => navigate("/product/product/create")}
              >
                新建商品
              </Button>
            )
          }
        />
      </div>

      <ContentCard>
        <form
          className="admin-list-filters !flex-col !items-stretch !gap-0"
          aria-label="商品筛选"
          onSubmit={(event) => {
            event.preventDefault();
            handleSearch();
          }}
        >
          {/* Top basic filter row */}
          <div className="flex flex-wrap items-end justify-between gap-4 w-full">
            <div className="flex flex-wrap items-end gap-4">
              <div className="flex flex-col gap-1.5">
                <Input
                  prefix={
                    <SearchOutlined style={{ color: token.colorTextSecondary }} />
                  }
                  placeholder="请输入商品名称"
                  aria-label="商品名称"
                  allowClear
                  className="w-60"
                  value={searchParams.keyword}
                  onChange={(e) =>
                    setSearchParams((prev) => ({
                      ...prev,
                      keyword: e.target.value,
                    }))
                  }
                />
              </div>

              <div className="flex flex-col gap-1.5">
                <TreeSelect
                  placeholder="请选择分类"
                  aria-label="分类"
                  allowClear
                  className="w-48"
                  treeData={categoryTreeData}
                  value={searchParams.category_id}
                  onChange={(val) =>
                    setSearchParams((prev) => ({ ...prev, category_id: val }))
                  }
                  treeDefaultExpandAll
                  showSearch
                  treeNodeFilterProp="title"
                />
              </div>

              <div className="flex items-center gap-3">
                <Button type="primary" htmlType="submit">
                  查询
                </Button>
                <Button onClick={handleReset}>
                  重置
                </Button>
              </div>
            </div>

            <Button
              type="text"
              className="flex items-center gap-1.5 font-medium px-4 py-1.5 rounded-lg"
              style={{
                backgroundColor: token.colorPrimaryBg,
                color: token.colorPrimary,
              }}
              onClick={() => setIsAdvancedOpen((prev) => !prev)}
            >
              更多筛选
              {isAdvancedOpen ? (
                <UpOutlined className="text-xs" />
              ) : (
                <DownOutlined className="text-xs" />
              )}
            </Button>
          </div>

          {/* Expanded More Filters Panel */}
          {isAdvancedOpen && (
            <div
              className="mt-4 p-5 rounded-xl border transition-all"
              style={{
                backgroundColor: token.colorFillAlter,
                borderColor: token.colorBorderSecondary,
              }}
            >
              <div
                className="flex items-center gap-2 mb-4 font-semibold text-sm"
                style={{ color: token.colorPrimary }}
              >
                <FilterOutlined />
                <span>更多筛选条件</span>
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-x-4 gap-y-4">
                {/* 1. 商品编码 */}
                <div className="flex flex-col gap-1.5">
                  <label
                    className="text-sm font-medium"
                    style={{ color: token.colorTextHeading }}
                  >
                    商品编码
                  </label>
                  <Input
                    placeholder="请输入商品编码"
                    allowClear
                    value={searchParams.product_code}
                    onChange={(e) =>
                      setSearchParams((prev) => ({
                        ...prev,
                        product_code: e.target.value,
                      }))
                    }
                  />
                </div>

                {/* 2. SKU */}
                <div className="flex flex-col gap-1.5">
                  <label
                    className="text-sm font-medium"
                    style={{ color: token.colorTextHeading }}
                  >
                    SKU
                  </label>
                  <Input
                    placeholder="请输入SKU"
                    allowClear
                    value={searchParams.sku_code}
                    onChange={(e) =>
                      setSearchParams((prev) => ({
                        ...prev,
                        sku_code: e.target.value,
                      }))
                    }
                  />
                </div>

                {/* 3. 运费类型 */}
                <div className="flex flex-col gap-1.5">
                  <label
                    className="text-sm font-medium"
                    style={{ color: token.colorTextHeading }}
                  >
                    运费类型
                  </label>
                  <Select
                    placeholder="请选择运费类型"
                    allowClear
                    value={searchParams.freight_type}
                    onChange={(val) =>
                      setSearchParams((prev) => ({
                        ...prev,
                        freight_type: val,
                      }))
                    }
                    options={[
                      { label: "全部", value: undefined },
                      { label: "包邮", value: 1 },
                      { label: "固定运费", value: 2 },
                      { label: "运费模板", value: 3 },
                    ]}
                  />
                </div>

                {/* 4. 价格区间 */}
                <div className="flex flex-col gap-1.5">
                  <label
                    className="text-sm font-medium"
                    style={{ color: token.colorTextHeading }}
                  >
                    价格区间
                  </label>
                  <div className="flex items-center gap-2">
                    <Input
                      prefix={
                        <span style={{ color: token.colorTextPlaceholder }}>
                          ¥
                        </span>
                      }
                      placeholder="最低价"
                      allowClear
                      value={searchParams.min_price}
                      onChange={(e) =>
                        setSearchParams((prev) => ({
                          ...prev,
                          min_price: e.target.value,
                        }))
                      }
                    />
                    <span style={{ color: token.colorTextTertiary }}>-</span>
                    <Input
                      prefix={
                        <span style={{ color: token.colorTextPlaceholder }}>
                          ¥
                        </span>
                      }
                      placeholder="最高价"
                      allowClear
                      value={searchParams.max_price}
                      onChange={(e) =>
                        setSearchParams((prev) => ({
                          ...prev,
                          max_price: e.target.value,
                        }))
                      }
                    />
                  </div>
                </div>

                {/* 5. 库存范围 */}
                <div className="flex flex-col gap-1.5">
                  <label
                    className="text-sm font-medium"
                    style={{ color: token.colorTextHeading }}
                  >
                    库存范围
                  </label>
                  <div className="flex items-center gap-2">
                    <Input
                      type="number"
                      placeholder="最低库存"
                      allowClear
                      value={searchParams.min_stock ?? ""}
                      onChange={(e) => {
                        const val = e.target.value.trim();
                        setSearchParams((prev) => ({
                          ...prev,
                          min_stock:
                            val !== "" ? parseInt(val, 10) : undefined,
                        }));
                      }}
                    />
                    <span style={{ color: token.colorTextTertiary }}>-</span>
                    <Input
                      type="number"
                      placeholder="最高库存"
                      allowClear
                      value={searchParams.max_stock ?? ""}
                      onChange={(e) => {
                        const val = e.target.value.trim();
                        setSearchParams((prev) => ({
                          ...prev,
                          max_stock:
                            val !== "" ? parseInt(val, 10) : undefined,
                        }));
                      }}
                    />
                  </div>
                </div>

                {/* 6. 销量范围 */}
                <div className="flex flex-col gap-1.5">
                  <label
                    className="text-sm font-medium"
                    style={{ color: token.colorTextHeading }}
                  >
                    销量范围
                  </label>
                  <div className="flex items-center gap-2">
                    <Input
                      type="number"
                      placeholder="最低销量"
                      allowClear
                      value={searchParams.min_sales ?? ""}
                      onChange={(e) => {
                        const val = e.target.value.trim();
                        setSearchParams((prev) => ({
                          ...prev,
                          min_sales:
                            val !== "" ? parseInt(val, 10) : undefined,
                        }));
                      }}
                    />
                    <span style={{ color: token.colorTextTertiary }}>-</span>
                    <Input
                      type="number"
                      placeholder="最高销量"
                      allowClear
                      value={searchParams.max_sales ?? ""}
                      onChange={(e) => {
                        const val = e.target.value.trim();
                        setSearchParams((prev) => ({
                          ...prev,
                          max_sales:
                            val !== "" ? parseInt(val, 10) : undefined,
                        }));
                      }}
                    />
                  </div>
                </div>

                {/* 7. 创建时间 */}
                <div className="flex flex-col gap-1.5">
                  <label
                    className="text-sm font-medium"
                    style={{ color: token.colorTextHeading }}
                  >
                    创建时间
                  </label>
                  <DatePicker.RangePicker
                    className="w-full"
                    placeholder={["开始日期", "结束日期"]}
                    allowClear
                    value={searchParams.created_at}
                    onChange={(dates) =>
                      setSearchParams((prev) => ({
                        ...prev,
                        created_at: dates as [Dayjs, Dayjs] | null,
                      }))
                    }
                  />
                </div>

                {/* 8. 更新时间 */}
                <div className="flex flex-col gap-1.5">
                  <label
                    className="text-sm font-medium"
                    style={{ color: token.colorTextHeading }}
                  >
                    更新时间
                  </label>
                  <DatePicker.RangePicker
                    className="w-full"
                    placeholder={["开始日期", "结束日期"]}
                    allowClear
                    value={searchParams.updated_at}
                    onChange={(dates) =>
                      setSearchParams((prev) => ({
                        ...prev,
                        updated_at: dates as [Dayjs, Dayjs] | null,
                      }))
                    }
                  />
                </div>
              </div>

              {/* Panel Bottom Actions */}
              <div className="flex justify-end items-center gap-3 mt-4 pt-2">
                <Button type="primary" htmlType="submit">
                  查询
                </Button>
                <Button onClick={handleReset}>
                  重置
                </Button>
              </div>
            </div>
          )}
        </form>

        <div className="px-6 pt-3">
          <Tabs
            activeKey={activeTab}
            onChange={handleTabChange}
            items={[
              { key: "all", label: `全部(${tabCounts.all})` },
              { key: "selling", label: `销售中(${tabCounts.selling})` },
              { key: "warning", label: `库存预警(${tabCounts.warning})` },
              { key: "sold_out", label: `已售罄(${tabCounts.sold_out})` },
              { key: "warehouse", label: `仓库中(${tabCounts.warehouse})` },
            ]}
            tabBarStyle={{ marginBottom: 0 }}
          />
        </div>

        <div className="admin-list-toolbar">
          <div className="flex flex-wrap items-center gap-4">
            <h2
              className="m-0 flex items-center gap-3 text-base font-semibold"
              style={{ color: token.colorTextHeading }}
            >
              <ShoppingOutlined
                className="text-xl"
                style={{ color: token.colorPrimary }}
              />
              商品列表
            </h2>
            <span
              className="text-sm"
              style={{ color: token.colorTextTertiary }}
            >
              共 {total} 件
            </span>
            {selectedRowKeys.length > 0 && (
              <Button
                type="link"
                size="small"
                onClick={() => setSelectedRowKeys([])}
              >
                已选 {selectedRowKeys.length} 件 · 取消选择
              </Button>
            )}
          </div>
          <Tooltip title="刷新列表">
            <Button
              type="text"
              icon={<ReloadOutlined spin={loading} />}
              onClick={() => {
                void loadProducts();
                void loadTabCounts();
              }}
              style={{ color: token.colorTextSecondary }}
            >
              刷新
            </Button>
          </Tooltip>
        </div>

        <div className="px-4">
          <Table<ProductRow>
            rowKey="id"
            loading={loading}
            columns={columns}
            dataSource={rows}
            size="middle"
            tableLayout="fixed"
            scroll={{ x: 1268 }}
            className="admin-list-table"
            rowSelection={{
              selectedRowKeys,
              onChange: setSelectedRowKeys,
              columnWidth: 40,
            }}
            pagination={false}
            locale={{ emptyText: "暂无商品数据" }}
          />
        </div>
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
              options={[10, 20, 50, 100].map((value) => ({
                value,
                label: value,
              }))}
              onChange={(value) => {
                setPage(1);
                setPageSize(value);
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
            onChange={setPage}
          />
        </div>
      </ContentCard>
    </div>
  );
}
