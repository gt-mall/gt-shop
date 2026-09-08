import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  Button,
  Card,
  Image,
  Input,
  message,
  Popconfirm,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  TreeSelect,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { PlusOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';

import { api } from '@/framework/http';
import type { ProductCategoryOption, ProductRow } from '@/framework/http';
import { usePermission } from '@/framework/store';
import { formatTime } from '@/shared/utils/format';
import { buildCategoryOptionTree } from '../types';

interface SearchState {
  keyword: string;
  category_id?: number;
  status?: number;
}

export default function ProductListPage() {
  const navigate = useNavigate();
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
  const handleToggleStatus = async (record: ProductRow) => {
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
  };

  // Delete product
  const handleDelete = async (record: ProductRow) => {
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
  };

  const columns: ColumnsType<ProductRow> = [
    {
      title: '主图',
      dataIndex: 'main_image_url',
      key: 'main_image_url',
      width: 70,
      align: 'center',
      render: (url: string) =>
        url ? (
          <Image
            src={url}
            alt="主图"
            width={44}
            height={44}
            style={{ objectFit: 'cover', borderRadius: 4 }}
          />
        ) : (
          <span style={{ color: '#9CA3AF' }}>—</span>
        ),
    },
    {
      title: '商品名称',
      dataIndex: 'name',
      key: 'name',
      minWidth: 200,
      render: (name: string, record) => (
        <div>
          <Typography.Text strong style={{ fontSize: 14 }}>
            {name}
          </Typography.Text>
          {record.subtitle && (
            <div style={{ color: '#6B7280', fontSize: 12, marginTop: 2 }}>{record.subtitle}</div>
          )}
        </div>
      ),
    },
    {
      title: '商品分类',
      dataIndex: 'category_name',
      key: 'category_name',
      width: 120,
      render: (catName: string) => catName || <span style={{ color: '#9CA3AF' }}>未分类</span>,
    },
    {
      title: '售价区间',
      key: 'price',
      width: 150,
      render: (_, record) => {
        if (!record.min_price && !record.max_price) {
          return <span style={{ color: '#9CA3AF' }}>—</span>;
        }
        if (record.min_price === record.max_price) {
          return <span style={{ color: '#EF4444', fontWeight: 600 }}>¥{record.min_price}</span>;
        }
        return (
          <span style={{ color: '#EF4444', fontWeight: 600 }}>
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
      title: '实际销量',
      dataIndex: 'sales_count',
      key: 'sales_count',
      width: 90,
      align: 'center',
    },
    {
      title: '虚拟销量',
      dataIndex: 'virtual_sales',
      key: 'virtual_sales',
      width: 90,
      align: 'center',
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 110,
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
      width: 170,
      render: (t: string) => formatTime(t),
    },
    {
      title: '操作',
      key: 'action',
      width: 220,
      fixed: 'right',
      render: (_, record) => (
        <Space size="small">
          <Button
            type="link"
            size="small"
            onClick={() => navigate(`/product/product/${record.id}`)}
          >
            查看
          </Button>
          {canUpdate && (
            <Button
              type="link"
              size="small"
              onClick={() => navigate(`/product/product/${record.id}/edit`)}
            >
              编辑
            </Button>
          )}
          {canViewInventory && (
            <Button
              type="link"
              size="small"
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
              <Button type="link" size="small" danger>
                删除
              </Button>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ];

  return (
    <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
      <Card>
        <Space wrap style={{ width: '100%', justifyContent: 'space-between' }}>
          <Space wrap size="middle">
            <Input
              placeholder="搜索商品名称/副标题/SKU编码/条码"
              allowClear
              style={{ width: 260 }}
              value={searchParams.keyword}
              onChange={(e) =>
                setSearchParams((prev) => ({ ...prev, keyword: e.target.value }))
              }
              onPressEnter={handleSearch}
            />

            <TreeSelect
              placeholder="全部分类"
              allowClear
              style={{ width: 180 }}
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
              style={{ width: 120 }}
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

            <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
              查询
            </Button>
            <Button icon={<ReloadOutlined />} onClick={handleReset}>
              重置
            </Button>
          </Space>

          {canCreate && (
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => navigate('/product/product/create')}
            >
              新建商品
            </Button>
          )}
        </Space>
      </Card>

      <Card>
        <Table<ProductRow>
          rowKey="id"
          loading={loading}
          columns={columns}
          dataSource={rows}
          scroll={{ x: 1200 }}
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
      </Card>
    </Space>
  );
}
