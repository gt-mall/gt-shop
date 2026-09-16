import type { Key } from 'react';
import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Button,
  Form,
  Image,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Radio,
  Select,
  Switch,
  Table,
  Tag,
  Tooltip,
  TreeSelect,
  message,
  theme,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  AppstoreOutlined,
  PlusOutlined,
  ReloadOutlined,
  SearchOutlined,
} from '@ant-design/icons';

import { api } from '@/framework/http';
import type { ProductCategoryRow } from '@/framework/http';
import { usePermission } from '@/framework/store';
import { ImagePicker } from '@/modules/system/file/picker';
import { ContentCard, PageHeader, adminListVars } from '@/shared/admin-ui';
import { buildCategoryTree, type CategoryTreeNode } from '../types';
import './category.less';

type CategoryFormData = {
  parent_id: number;
  name: string;
  image_id?: number | null;
  sort?: number;
  status: number;
};

interface SearchState {
  keyword: string;
  status?: number;
}

export default function CategoryPage() {
  const { token } = theme.useToken();
  const hasPermission = usePermission();
  const canCreate = hasPermission('product:category:create');
  const canUpdate = hasPermission('product:category:update');
  const canDelete = hasPermission('product:category:delete');

  const [form] = Form.useForm<CategoryFormData>();
  const [flatList, setFlatList] = useState<ProductCategoryRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [togglingId, setTogglingId] = useState<number | null>(null);

  // Search input state
  const [searchParams, setSearchParams] = useState<SearchState>({
    keyword: '',
    status: undefined,
  });

  // Active query state for filtering
  const [activeQuery, setActiveQuery] = useState<SearchState>({
    keyword: '',
    status: undefined,
  });

  // Tree row expansion
  const [expandedRowKeys, setExpandedRowKeys] = useState<Key[]>([]);
  const [hasInitializedExpand, setHasInitializedExpand] = useState(false);

  // Modal state
  const [modalOpen, setModalOpen] = useState(false);
  const [editingCategory, setEditingCategory] = useState<ProductCategoryRow | null>(null);
  const [saving, setSaving] = useState(false);

  // Load all categories
  const loadCategories = useCallback(async () => {
    setLoading(true);
    const res = await api.api.listProductCategories();
    setLoading(false);
    if (res.code === 1) {
      setFlatList(res.data ?? []);
    } else {
      message.error(res.message || '加载商品分类失败');
    }
  }, []);

  useEffect(() => {
    void loadCategories();
  }, [loadCategories]);

  // Find all descendant IDs of a category to prevent cycle selection
  const getDescendantIDs = useCallback(
    (targetID: number): Set<number> => {
      const descendants = new Set<number>();
      descendants.add(targetID);

      const findChildren = (parentID: number) => {
        for (const item of flatList) {
          const itemId = item.id ?? 0;
          if (item.parent_id === parentID && !descendants.has(itemId)) {
            descendants.add(itemId);
            findChildren(itemId);
          }
        }
      };
      findChildren(targetID);
      return descendants;
    },
    [flatList],
  );

  // Build tree with filter while preserving ancestors of matched nodes
  const displayTree = useMemo(() => {
    const rawTree = buildCategoryTree(flatList);
    const kw = activeQuery.keyword.trim().toLowerCase();
    const searchStatus = activeQuery.status;

    if (!kw && searchStatus === undefined) {
      return rawTree;
    }

    const filterNode = (node: CategoryTreeNode): CategoryTreeNode | null => {
      const nameMatch = !kw || (node.name ?? '').toLowerCase().includes(kw);
      const statusMatch = searchStatus === undefined || node.status === searchStatus;
      const isSelfMatch = nameMatch && statusMatch;

      const matchedChildren: CategoryTreeNode[] = [];
      if (node.children) {
        for (const child of node.children) {
          const res = filterNode(child);
          if (res) matchedChildren.push(res);
        }
      }

      if (isSelfMatch || matchedChildren.length > 0) {
        return {
          ...node,
          children: matchedChildren.length > 0 ? matchedChildren : undefined,
        };
      }
      return null;
    };

    const filtered: CategoryTreeNode[] = [];
    for (const root of rawTree) {
      const res = filterNode(root);
      if (res) filtered.push(res);
    }
    return filtered;
  }, [flatList, activeQuery]);

  // All expandable tree node keys
  const allExpandableKeys = useMemo(() => {
    const keys: Key[] = [];
    const collect = (nodes: CategoryTreeNode[]) => {
      for (const node of nodes) {
        if (node.children && node.children.length > 0 && node.id) {
          keys.push(node.id);
          collect(node.children);
        }
      }
    };
    collect(displayTree);
    return keys;
  }, [displayTree]);

  // Expand all nodes by default on initial data load
  useEffect(() => {
    if (!hasInitializedExpand && allExpandableKeys.length > 0) {
      setExpandedRowKeys(allExpandableKeys);
      setHasInitializedExpand(true);
    }
  }, [allExpandableKeys, hasInitializedExpand]);

  // Auto-expand all matching paths when search query is active
  useEffect(() => {
    if (activeQuery.keyword.trim() || activeQuery.status !== undefined) {
      setExpandedRowKeys(allExpandableKeys);
    }
  }, [activeQuery, allExpandableKeys]);

  const isAllExpanded =
    allExpandableKeys.length > 0 &&
    allExpandableKeys.every((key) => expandedRowKeys.includes(key));

  const handleToggleExpandAll = () => {
    if (isAllExpanded) {
      setExpandedRowKeys([]);
    } else {
      setExpandedRowKeys(allExpandableKeys);
    }
  };

  // Search and reset handlers
  const handleSearch = () => {
    setActiveQuery({ ...searchParams });
  };

  const handleReset = () => {
    const empty: SearchState = {
      keyword: '',
      status: undefined,
    };
    setSearchParams(empty);
    setActiveQuery(empty);
  };

  // Category counts
  const totalCategoryCount = flatList.length;
  const rootCount = useMemo(() => {
    return flatList.filter((item) => (item.parent_id ?? 0) === 0).length;
  }, [flatList]);

  const displayCount = useMemo(() => {
    const countNodes = (nodes: CategoryTreeNode[]): number => {
      let count = 0;
      for (const node of nodes) {
        count += 1;
        if (node.children && node.children.length > 0) {
          count += countNodes(node.children);
        }
      }
      return count;
    };
    return countNodes(displayTree);
  }, [displayTree]);

  const isFiltering = Boolean(activeQuery.keyword.trim() || activeQuery.status !== undefined);

  // Options tree for Parent TreeSelect
  const parentSelectTreeData = useMemo(() => {
    const disabledIDs = editingCategory?.id
      ? getDescendantIDs(editingCategory.id)
      : new Set<number>();

    const convert = (nodes: CategoryTreeNode[]): any[] => {
      return nodes.map((n) => {
        const nid = n.id ?? 0;
        return {
          key: nid,
          value: nid,
          title: (n.name ?? '') + (n.status === 0 ? ' (已禁用)' : ''),
          disabled: disabledIDs.has(nid),
          children: n.children ? convert(n.children) : undefined,
        };
      });
    };

    return [
      {
        key: 0,
        value: 0,
        title: '顶级分类',
        children: convert(buildCategoryTree(flatList)),
      },
    ];
  }, [flatList, editingCategory, getDescendantIDs]);

  // Open Create modal
  const openCreate = useCallback(
    (parentID: number = 0) => {
      setEditingCategory(null);
      form.resetFields();
      form.setFieldsValue({
        parent_id: parentID,
        name: '',
        image_id: undefined,
        sort: 0,
        status: 1,
      });
      setModalOpen(true);
    },
    [form],
  );

  // Open Edit modal
  const openEdit = useCallback(
    (record: ProductCategoryRow) => {
      setEditingCategory(record);
      form.resetFields();
      form.setFieldsValue({
        parent_id: record.parent_id ?? 0,
        name: record.name ?? '',
        image_id: record.image_id ?? undefined,
        sort: record.sort ?? 0,
        status: record.status ?? 1,
      });
      setModalOpen(true);
    },
    [form],
  );

  // Handle submit
  const handleFinish = async (values: CategoryFormData) => {
    setSaving(true);
    const imageIdVal = values.image_id || null;

    if (editingCategory && editingCategory.id) {
      const res = await api.api.updateProductCategory({
        id: editingCategory.id,
        parent_id: values.parent_id,
        name: values.name.trim(),
        image_id: imageIdVal,
        sort: values.sort ?? 0,
        status: values.status as 0 | 1,
      });
      setSaving(false);
      if (res.code === 1) {
        message.success('分类更新成功');
        setModalOpen(false);
        setEditingCategory(null);
        void loadCategories();
      } else {
        message.error(res.message || '更新分类失败');
      }
    } else {
      const res = await api.api.createProductCategory({
        parent_id: values.parent_id,
        name: values.name.trim(),
        image_id: imageIdVal,
        sort: values.sort ?? 0,
        status: values.status as 0 | 1,
      });
      setSaving(false);
      if (res.code === 1) {
        message.success('分类创建成功');
        setModalOpen(false);
        void loadCategories();
      } else {
        message.error(res.message || '创建分类失败');
      }
    }
  };

  // Handle status quick toggle
  const handleToggleStatus = useCallback(
    async (record: ProductCategoryRow) => {
      if (!record.id) return;
      setTogglingId(record.id);
      const nextStatus = record.status === 1 ? 0 : 1;
      const res = await api.api.updateProductCategory({
        id: record.id,
        status: nextStatus,
      });
      setTogglingId(null);
      if (res.code === 1) {
        message.success(nextStatus === 1 ? '分类已启用' : '分类已禁用');
        void loadCategories();
      } else {
        message.error(res.message || '切换状态失败');
      }
    },
    [loadCategories],
  );

  // Handle delete
  const handleDelete = useCallback(
    async (record: ProductCategoryRow) => {
      if (!record.id) return;
      const res = await api.api.deleteProductCategory({ id: record.id });
      if (res.code === 1) {
        message.success('分类删除成功');
        void loadCategories();
      } else {
        message.error(res.message || '删除分类失败');
      }
    },
    [loadCategories],
  );

  const columns: ColumnsType<CategoryTreeNode> = useMemo(
    () => [
      {
        title: '分类名称',
        dataIndex: 'name',
        key: 'name',
        minWidth: 260,
        render: (name: string, record) => {
          const isRoot = (record.parent_id ?? 0) === 0;
          const childCount = record.children?.length ?? 0;
          return (
            <div className="inline-flex items-center gap-2 flex-wrap">
              <span
                className={isRoot ? 'font-semibold text-sm' : 'text-sm'}
                style={{
                  color: isRoot ? token.colorTextHeading : token.colorText,
                }}
              >
                {name}
              </span>
              <span
                className="text-xs font-mono"
                style={{ color: token.colorTextQuaternary }}
              >
                #{record.id}
              </span>
              {childCount > 0 && (
                <Tag
                  bordered={false}
                  className="text-xs m-0"
                  style={{
                    backgroundColor: token.colorFillTertiary,
                    color: token.colorTextSecondary,
                  }}
                >
                  {childCount} 个子分类
                </Tag>
              )}
              {record.status === 0 && (
                <Tag bordered={false} color="error" className="m-0">
                  已禁用
                </Tag>
              )}
            </div>
          );
        },
      },
      {
        title: '分类图标',
        dataIndex: 'image_url',
        key: 'image_url',
        width: 100,
        align: 'center',
        render: (url: string) =>
          url ? (
            <div className="flex justify-center">
              <Image
                src={url}
                alt="分类图标"
                width={36}
                height={36}
                style={{
                  objectFit: 'cover',
                  borderRadius: token.borderRadiusSM,
                  border: `1px solid ${token.colorBorderSecondary}`,
                }}
              />
            </div>
          ) : (
            <span style={{ color: token.colorTextQuaternary }}>—</span>
          ),
      },
      {
        title: '排序值',
        dataIndex: 'sort',
        key: 'sort',
        width: 90,
        align: 'center',
        render: (sort: number) => (
          <span
            className="font-mono text-sm"
            style={{ color: token.colorTextSecondary }}
          >
            {sort ?? 0}
          </span>
        ),
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
              size="small"
              checked={status === 1}
              checkedChildren="启用"
              unCheckedChildren="禁用"
              loading={togglingId === record.id}
              onChange={() => void handleToggleStatus(record)}
            />
          ) : (
            <Tag bordered={false} color={status === 1 ? 'success' : 'default'}>
              {status === 1 ? '启用' : '禁用'}
            </Tag>
          ),
      },
      {
        title: '操作',
        key: 'action',
        width: 220,
        align: 'right',
        render: (_, record) => (
          <div className="admin-list-actions flex items-center justify-end gap-3">
            {canCreate && (
              <Button
                type="link"
                size="small"
                onClick={() => openCreate(record.id ?? 0)}
              >
                添加子分类
              </Button>
            )}
            {canUpdate && (
              <Button
                type="link"
                size="small"
                onClick={() => openEdit(record)}
              >
                编辑
              </Button>
            )}
            {canDelete && (
              <Popconfirm
                title="确定删除该分类？"
                description="若存在子分类或被商品引用则无法删除。"
                okText="确定"
                cancelText="取消"
                okButtonProps={{ danger: true }}
                onConfirm={() => void handleDelete(record)}
              >
                <Button type="link" size="small" danger>
                  删除
                </Button>
              </Popconfirm>
            )}
          </div>
        ),
      },
    ],
    [
      token,
      canCreate,
      canUpdate,
      canDelete,
      togglingId,
      handleToggleStatus,
      handleDelete,
      openCreate,
      openEdit,
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
          title="商品分类"
          description="管理多级商品分类层级、图标与启用状态"
          extra={
            canCreate && (
              <Button
                type="primary"
                icon={<PlusOutlined />}
                onClick={() => openCreate(0)}
              >
                新增顶级分类
              </Button>
            )
          }
        />
      </div>

      <ContentCard>
        <form
          className="admin-list-filters"
          aria-label="分类筛选"
          onSubmit={(event) => {
            event.preventDefault();
            handleSearch();
          }}
        >
          <Input
            prefix={
              <SearchOutlined style={{ color: token.colorTextSecondary }} />
            }
            placeholder="按分类名称搜索..."
            aria-label="搜索分类"
            allowClear
            className="category-list-keyword"
            value={searchParams.keyword}
            onChange={(e) =>
              setSearchParams((prev) => ({ ...prev, keyword: e.target.value }))
            }
          />
          <Select
            placeholder="全部状态"
            aria-label="分类状态"
            allowClear
            className="w-36"
            value={searchParams.status}
            onChange={(val) =>
              setSearchParams((prev) => ({ ...prev, status: val }))
            }
            options={[
              { label: '全部状态', value: undefined },
              { label: '启用', value: 1 },
              { label: '禁用', value: 0 },
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
              <AppstoreOutlined
                className="text-xl"
                style={{ color: token.colorPrimary }}
              />
              分类列表
            </h2>
            <span
              className="text-sm"
              style={{ color: token.colorTextTertiary }}
            >
              {isFiltering
                ? `共 ${totalCategoryCount} 个分类 · 筛选命中 ${displayCount} 个`
                : `共 ${totalCategoryCount} 个分类（顶级 ${rootCount} 个）`}
            </span>
          </div>

          <div className="flex items-center gap-2">
            {allExpandableKeys.length > 0 && (
              <Button
                type="text"
                size="small"
                onClick={handleToggleExpandAll}
                style={{ color: token.colorTextSecondary }}
              >
                {isAllExpanded ? '全部折叠' : '全部展开'}
              </Button>
            )}
            <Tooltip title="刷新列表">
              <Button
                type="text"
                icon={<ReloadOutlined spin={loading} />}
                onClick={() => void loadCategories()}
                style={{ color: token.colorTextSecondary }}
              >
                刷新
              </Button>
            </Tooltip>
          </div>
        </div>

        <div className="px-4 pb-4">
          <Table<CategoryTreeNode>
            rowKey="id"
            loading={loading}
            columns={columns}
            dataSource={displayTree}
            size="middle"
            tableLayout="auto"
            scroll={{ x: 960 }}
            className="admin-list-table"
            pagination={false}
            expandable={{
              expandedRowKeys,
              onExpandedRowsChange: (keys) => setExpandedRowKeys([...keys]),
            }}
            locale={{ emptyText: '暂无商品分类' }}
          />
        </div>
      </ContentCard>

      <Modal
        title={
          <div className="flex items-center gap-2.5">
            <AppstoreOutlined style={{ color: token.colorPrimary, fontSize: 18 }} />
            <span
              className="font-semibold text-base"
              style={{ color: token.colorTextHeading }}
            >
              {editingCategory ? '编辑商品分类' : '新增商品分类'}
            </span>
          </div>
        }
        open={modalOpen}
        onCancel={() => {
          setModalOpen(false);
          setEditingCategory(null);
        }}
        confirmLoading={saving}
        onOk={() => form.submit()}
        okText={editingCategory ? '保存修改' : '立即创建'}
        cancelText="取消"
        destroyOnClose
        width={540}
      >
        <Form<CategoryFormData>
          form={form}
          layout="vertical"
          onFinish={handleFinish}
          className="pt-2"
        >
          <Form.Item
            name="parent_id"
            label="上级分类"
            rules={[{ required: true, message: '请选择上级分类' }]}
            extra="选择“顶级分类”表示作为一级类目"
          >
            <TreeSelect
              placeholder="请选择上级分类"
              treeData={parentSelectTreeData}
              treeDefaultExpandAll
              showSearch
              treeNodeFilterProp="title"
            />
          </Form.Item>

          <Form.Item
            name="name"
            label="分类名称"
            rules={[
              { required: true, message: '请输入分类名称' },
              { max: 255, message: '分类名称最多255个字符' },
            ]}
          >
            <Input placeholder="请输入分类名称" maxLength={255} showCount />
          </Form.Item>

          <Form.Item
            name="image_id"
            label="分类图标"
            extra="建议上传 1:1 正方形图标（如 200×200px），支持 JPG、PNG、WEBP"
          >
            <ImagePicker multiple={false} />
          </Form.Item>

          <Form.Item
            name="sort"
            label="排序值"
            extra="数值越小越靠前显示，默认为 0"
          >
            <InputNumber
              min={0}
              max={99999}
              style={{ width: '100%' }}
              placeholder="请输入排序值"
            />
          </Form.Item>

          <Form.Item
            name="status"
            label="启用状态"
            rules={[{ required: true, message: '请选择状态' }]}
          >
            <Radio.Group>
              <Radio value={1}>启用</Radio>
              <Radio value={0}>禁用</Radio>
            </Radio.Group>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
