import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  Button,
  Divider,
  Image,
  Input,
  message,
  Popconfirm,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  theme,
  TreeSelect,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { PlusOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';

import { api } from '@/framework/http';
import type { ProductCategoryOption, ProductRow } from '@/framework/http';
import { usePermission } from '@/framework/store';
import { FilterCard, PageHeader, TableCard } from '@/shared/admin-ui';
import { formatTime } from '@/shared/utils/format';
import { buildCategoryOptionTree } from '../types';

interface SearchState {
  keyword: string;
  category_id?: number;
  status?: number;
}

export default function ProductListPage() {
  const navigate = useNavigate();
  const { token } = theme.useToken();
  const hasPermission = usePermission();
  const canCreate = hasPermission('product:product:create');
  const canUpdate = hasPermission('product:product:update');
  const canDelete = hasPermission('product:product:delete');
  const canViewInventory = hasPermission('product:inventory:list');

  const [rows, setRows] = useState<ProductRow[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);

  // Pagination state
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  // Search input state
  const [searchParams, setSearchParams] = useState<SearchState>({
    keyword: '',
    category_id: undefined,
    status: undefined,
  });

  // Active query state used for API requests
  const [activeQuery, setActiveQuery] = useState<SearchState>({
    keyword: '',
    category_id: undefined,
    status: undefined,
  });

  // Category options
  const [categoryOptions, setCategoryOptions] = useState<ProductCategoryOption[]>([]);
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
      setRows(res.data ?? []);
      setTotal(res.total ?? 0);
    } else {
      message.error(res.message || '加载商品列表失败');
    }
  }, [page, pageSize, activeQuery]);

  useEffect(() => {
    void loadProducts();
  }, [loadProducts]);

  // Category TreeSelect Data
  const categoryTreeData = useMemo(() => {
    const tree = buildCategoryOptionTree(categoryOptions);
    return [
      { key: 'all', value: undefined, title: '全部分类' },
      { key: 'uncategorized', value: 0, title: '未分类' },
      ...tree,
    ];
  }, [categoryOptions]);

  // Search and reset handlers
  const handleSearch = () => {
    setPage(1);
    setActiveQuery({ ...searchParams });
  };

  const handleReset = () => {
    const empty: SearchState = { keyword: '', category_id: undefined, status: undefined };
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
        message.success(nextStatus === 1 ? '商品已上架' : '商品已下架');
        void loadProducts();
      } else {
        message.error(res.message || '修改状态失败');
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
        message.success('商品删除成功');
        if (rows.length === 1 && page > 1) {
          setPage(page - 1);
        } else {
          void loadProducts();
        }
      } else {
        message.error(res.message || '删除商品失败');
      }
    },
    [rows.length, page, loadProducts],
  );

  const columns: ColumnsType<ProductRow> = useMemo(
    () => [
      {
        title: '商品信息',
        key: 'product_info',
        minWidth: 280,
        render: (_, record) => (
          <div className="flex items-center gap-3 py-0.5">
            <div
              className="w-12 h-12 rounded shrink-0 flex items-center justify-center overflow-hidden"
              style={{
                backgroundColor: token.colorFillAlter,
                border: `1px solid ${token.colorBorderSecondary}`,
              }}
            >
              {record.main_image_url ? (
                <Image
                  src={record.main_image_url}
                  alt={record.name}
                  width={48}
                  height={48}
                  style={{ objectFit: 'cover' }}
                />
              ) : (
                <span style={{ color: token.colorTextQuaternary, fontSize: 12 }}>无主图</span>
              )}
            </div>
            <div className="min-w-0 flex-1">
              <Typography.Link
                onClick={() => navigate(`/product/product/${record.id}`)}
                style={{
                  color: token.colorTextHeading,
                  fontWeight: 500,
                  fontSize: 14,
                  lineHeight: 1.4,
                  display: '-webkit-box',
                  WebkitLineClamp: 2,
                  WebkitBoxOrient: 'vertical',
                  overflow: 'hidden',
                }}
                title={record.name}
              >
                {record.name}
              </Typography.Link>
              {record.subtitle && (
                <div
                  className="truncate text-xs mt-1"
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
        title: '商品分类',
        dataIndex: 'category_name',
        key: 'category_name',
        width: 120,
        render: (catName: string) =>
          catName || <span style={{ color: token.colorTextQuaternary }}>未分类</span>,
      },
      {
        title: '售价区间',
        key: 'price',
        width: 150,
        render: (_, record) => {
          if (!record.min_price && !record.max_price) {
            return <span style={{ color: token.colorTextQuaternary }}>—</span>;
          }
          if (record.min_price === record.max_price) {
            return (
              <span style={{ color: token.colorError, fontWeight: 600 }}>
                ¥{record.min_price}
              </span>
            );
          }
          return (
            <span style={{ color: token.colorError, fontWeight: 600 }}>
              ¥{record.min_price} ~ ¥{record.max_price}
            </span>
          );
        },
      },
      {
        title: 'SKU数',
        dataIndex: 'sku_count',
        key: 'sku_count',
        width: 80,
        align: 'center',
      },
      {
        title: '销量',
        key: 'sales',
        width: 110,
        render: (_, record) => (
          <div className="space-y-0.5 text-xs">
            <div>
              <span style={{ color: token.colorTextSecondary }}>实际 </span>
              <span className="font-medium" style={{ color: token.colorText }}>
                {record.sales_count ?? 0}
              </span>
            </div>
            <div>
              <span style={{ color: token.colorTextTertiary }}>虚拟 </span>
              <span style={{ color: token.colorTextSecondary }}>
                {record.virtual_sales ?? 0}
              </span>
            </div>
          </div>
        ),
      },
      {
        title: '状态',
        dataIndex: 'status',
        key: 'status',
        width: 100,
        align: 'center',
        render: (status: number, record) =>
          canUpdate ? (
            <Switch
              checked={status === 1}
              checkedChildren="上架"
              unCheckedChildren="下架"
              onChange={() => handleToggleStatus(record)}
            />
          ) : status === 1 ? (
            <Tag color="success">已上架</Tag>
          ) : (
            <Tag color="default">已下架</Tag>
          ),
      },
      {
        title: '排序',
        dataIndex: 'sort',
        key: 'sort',
        width: 70,
        align: 'center',
      },
      {
        title: '更新时间',
        dataIndex: 'updated_at',
        key: 'updated_at',
        width: 160,
        render: (t: string) => (
          <span style={{ color: token.colorTextSecondary, fontSize: 13 }}>
            {formatTime(t)}
          </span>
        ),
      },
      {
        title: '操作',
        key: 'action',
        width: 190,
        fixed: 'right',
        render: (_, record) => (
          <Space split={<Divider type="vertical" style={{ margin: '0 2px' }} />} size={2}>
            <Button
              type="link"
              size="small"
              style={{ padding: 0 }}
              onClick={() => navigate(`/product/product/${record.id}`)}
            >
              查看
            </Button>
            {canUpdate && (
              <Button
                type="link"
                size="small"
                style={{ padding: 0 }}
                onClick={() => navigate(`/product/product/${record.id}/edit`)}
              >
                编辑
              </Button>
            )}
            {canViewInventory && (
              <Button
                type="link"
                size="small"
                style={{ padding: 0 }}
                onClick={() => navigate(`/product/inventory?product_id=${record.id}`)}
              >
                库存
              </Button>
            )}
            {canDelete && (
              <Popconfirm
                title="确定删除该商品？"
                description="删除后该商品及所有SKU将移入回收站。"
                okText="确定"
                cancelText="取消"
                onConfirm={() => handleDelete(record)}
              >
                <Button type="link" size="small" danger style={{ padding: 0 }}>
                  删除
                </Button>
              </Popconfirm>
            )}
          </Space>
        ),
      },
    ],
    [
      token,
      canUpdate,
      canViewInventory,
      canDelete,
      navigate,
      handleDelete,
      handleToggleStatus,
    ],
  );

  return (
    <div className="product-list-page">
      <PageHeader
        title="商品管理"
        description="管理商品、SKU、价格、销量及上下架状态"
        extra={
          canCreate && (
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => navigate('/product/product/create')}
            >
              新建商品
            </Button>
          )
        }
      />

      <FilterCard>
        <div className="flex flex-wrap items-center gap-3">
          <Input
            placeholder="搜索商品名称/副标题/SKU编码/条码"
            allowClear
            className="w-64"
            value={searchParams.keyword}
            onChange={(e) =>
              setSearchParams((prev) => ({ ...prev, keyword: e.target.value }))
            }
            onPressEnter={handleSearch}
          />

          <TreeSelect
            placeholder="全部分类"
            allowClear
            className="w-44"
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
            allowClear
            className="w-32"
            value={searchParams.status}
            onChange={(val) =>
              setSearchParams((prev) => ({ ...prev, status: val }))
            }
            options={[
              { label: '全部状态', value: undefined },
              { label: '已上架', value: 1 },
              { label: '已下架', value: 0 },
            ]}
          />

          <div className="flex items-center gap-2">
            <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
              查询
            </Button>
            <Button icon={<ReloadOutlined />} onClick={handleReset}>
              重置
            </Button>
          </div>
        </div>
      </FilterCard>

      <TableCard>
        <Table<ProductRow>
          rowKey="id"
          loading={loading}
          columns={columns}
          dataSource={rows}
          scroll={{ x: 1240 }}
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: true,
            pageSizeOptions: ['10', '20', '50', '100'],
            showTotal: (t) => `共 ${t} 件商品`,
            onChange: (p, ps) => {
              setPage(ps !== pageSize ? 1 : p);
              setPageSize(ps);
            },
          }}
          locale={{ emptyText: '暂无商品数据' }}
        />
      </TableCard>
    </div>
  );
}
