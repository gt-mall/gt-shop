import type { Key } from "react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  Button,
  Dropdown,
  Image,
  Input,
  Modal,
  Pagination,
  Select,
  Space,
  Switch,
  Table,
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
  PlusOutlined,
  ShoppingOutlined,
  ReloadOutlined,
  SearchOutlined,
} from "@ant-design/icons";
import { Link, useNavigate } from "react-router-dom";

import { api } from "@/framework/http";
import type { ProductCategoryOption, ProductRow } from "@/framework/http";
import { usePermission } from "@/framework/store";
import { ContentCard, PageHeader, adminListVars } from "@/shared/admin-ui";
import { formatTime } from "@/shared/utils/format";
import { buildCategoryOptionTree } from "../types";
import "./list.less";

interface SearchState {
  keyword: string;
  category_id?: number;
  status?: number;
}

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

  // Search input state
  const [searchParams, setSearchParams] = useState<SearchState>({
    keyword: "",
    category_id: undefined,
    status: undefined,
  });

  // Active query state used for API requests
  const [activeQuery, setActiveQuery] = useState<SearchState>({
    keyword: "",
    category_id: undefined,
    status: undefined,
  });

  // Category options
  const [categoryOptions, setCategoryOptions] = useState<
    ProductCategoryOption[]
  >([]);
  const reqSeqRef = useRef(0);

  // Load category options for filter
  const loadCategoryOptions = useCallback(async () => {
    const res = await api.api.getProductCategoryOptions();
    if (res.code === 1) {
      setCategoryOptions(res.data ?? []);
    }
  }, []);

  useEffect(() => {
    void loadCategoryOptions();
  }, [loadCategoryOptions]);

  // Load product list
  const loadProducts = useCallback(async () => {
    const currentSeq = ++reqSeqRef.current;
    setLoading(true);

    const res = await api.api.listProducts({
      page,
      page_size: pageSize,
      keyword: activeQuery.keyword.trim() || undefined,
      category_id: activeQuery.category_id,
      status: activeQuery.status,
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
  }, [page, pageSize, activeQuery]);

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
    const empty: SearchState = {
      keyword: "",
      category_id: undefined,
      status: undefined,
    };
    setSearchParams(empty);
    setActiveQuery(empty);
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
      } else {
        message.error(res.message || "修改状态失败");
      }
    },
    [loadProducts],
  );

  // Delete product
  const handleDelete = useCallback(
    async (record: ProductRow) => {
      if (!record.id) return;
      const res = await api.api.deleteProduct({ id: record.id });
      if (res.code === 1) {
        message.success("商品删除成功");
        if (rows.length === 1 && page > 1) {
          setPage(page - 1);
        } else {
          void loadProducts();
        }
      } else {
        message.error(res.message || "删除商品失败");
      }
    },
    [rows.length, page, loadProducts],
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
        width: 160,
        render: (t: string) => (
          <span
            className="tabular-nums text-sm"
            style={{ color: token.colorTextTertiary }}
          >
            {formatTime(t)}
          </span>
        ),
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
          description="管理商品、SKU、价格、销量及上下架状态"
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
          className="admin-list-filters"
          aria-label="商品筛选"
          onSubmit={(event) => {
            event.preventDefault();
            handleSearch();
          }}
        >
          <Input
            prefix={
              <SearchOutlined style={{ color: token.colorTextSecondary }} />
            }
            placeholder="搜索商品名称 / 副标题 / SKU / 条形码..."
            aria-label="搜索商品"
            allowClear
            className="product-list-keyword"
            value={searchParams.keyword}
            onChange={(e) =>
              setSearchParams((prev) => ({ ...prev, keyword: e.target.value }))
            }
          />
          <TreeSelect
            placeholder="全部分类"
            aria-label="商品分类"
            allowClear
            className="w-40"
            treeData={categoryTreeData}
            value={searchParams.category_id}
            onChange={(val) =>
              setSearchParams((prev) => ({ ...prev, category_id: val }))
            }
            treeDefaultExpandAll
            showSearch
            treeNodeFilterProp="title"
          />
          <Select
            placeholder="全部状态"
            aria-label="商品状态"
            allowClear
            className="w-44"
            value={searchParams.status}
            onChange={(val) =>
              setSearchParams((prev) => ({ ...prev, status: val }))
            }
            options={[
              { label: "全部状态", value: undefined },
              { label: "已上架", value: 1 },
              { label: "已下架", value: 0 },
            ]}
          />
          <div className="flex items-center gap-3">
            <Button type="primary" htmlType="submit" icon={<SearchOutlined />}>
              查询
            </Button>
            <Button icon={<ReloadOutlined />} onClick={handleReset}>
              重置
            </Button>
          </div>
        </form>

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
              onClick={() => void loadProducts()}
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
