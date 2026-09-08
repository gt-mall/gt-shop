import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Button,
  Card,
  Form,
  Image,
  Input,
  InputNumber,
  message,
  Modal,
  Popconfirm,
  Radio,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  TreeSelect,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { AppstoreOutlined, PlusOutlined, ReloadOutlined } from '@ant-design/icons';

import { api } from '@/framework/http';
import type { ProductCategoryRow } from '@/framework/http';
import { usePermission } from '@/framework/store';
import { ImagePicker } from '@/modules/system/file/picker';
import { formatTime } from '@/shared/utils/format';
import { buildCategoryTree, type CategoryTreeNode } from '../types';

type CategoryFormData = {
  parent_id: number;
  name: string;
  image_id?: number | null;
  sort?: number;
  status: number;
};

export default function CategoryPage() {
  const hasPermission = usePermission();
  const canCreate = hasPermission('product:category:create');
  const canUpdate = hasPermission('product:category:update');
  const canDelete = hasPermission('product:category:delete');

  const [form] = Form.useForm<CategoryFormData>();
  const [flatList, setFlatList] = useState<ProductCategoryRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [searchKeyword, setSearchKeyword] = useState('');
  const [searchStatus, setSearchStatus] = useState<number | undefined>(undefined);

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

  // Find all descendant IDs of a category
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
    const kw = searchKeyword.trim().toLowerCase();

    if (!kw && searchStatus === undefined) {
      return rawTree;
    }

    const filterNode = (node: CategoryTreeNode): CategoryTreeNode | null => {
      const nameMatch = !kw || (node.name ?? '').toLowerCase().includes(kw);
      const statusMatch = searchStatus === undefined || node.status === searchStatus;
      const isSelfMatch = nameMatch && statusMatch;

      let matchedChildren: CategoryTreeNode[] = [];
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
  }, [flatList, searchKeyword, searchStatus]);

  // Options tree for Parent TreeSelect
  const parentSelectTreeData = useMemo(() => {
    const disabledIDs = editingCategory?.id ? getDescendantIDs(editingCategory.id) : new Set<number>();

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
  const openCreate = (parentID: number = 0) => {
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
  };

  // Open Edit modal
  const openEdit = (record: ProductCategoryRow) => {
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
  };

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
  const handleToggleStatus = async (record: ProductCategoryRow) => {
    if (!record.id) return;
    const nextStatus = record.status === 1 ? 0 : 1;
    const res = await api.api.updateProductCategory({
      id: record.id,
      status: nextStatus,
    });
    if (res.code === 1) {
      message.success(nextStatus === 1 ? '分类已启用' : '分类已禁用');
      void loadCategories();
    } else {
      message.error(res.message || '切换状态失败');
    }
  };

  // Handle delete
  const handleDelete = async (record: ProductCategoryRow) => {
    if (!record.id) return;
    const res = await api.api.deleteProductCategory({ id: record.id });
    if (res.code === 1) {
      message.success('分类删除成功');
      void loadCategories();
    } else {
      message.error(res.message || '删除分类失败');
    }
  };

  const columns: ColumnsType<CategoryTreeNode> = [
    {
      title: '分类名称',
      dataIndex: 'name',
      key: 'name',
      width: 280,
      render: (name: string, record) => (
        <Space>
          <span style={{ fontWeight: record.parent_id === 0 ? 600 : 400 }}>{name}</span>
          {record.status === 0 && <Tag color="error">已禁用</Tag>}
        </Space>
      ),
    },
    {
      title: '分类图标',
      dataIndex: 'image_url',
      key: 'image_url',
      width: 100,
      align: 'center',
      render: (url: string) =>
        url ? (
          <Image
            src={url}
            alt="分类图标"
            width={40}
            height={40}
            style={{ objectFit: 'cover', borderRadius: 4 }}
          />
        ) : (
          <span style={{ color: '#9CA3AF' }}>—</span>
        ),
    },
    {
      title: '排序值',
      dataIndex: 'sort',
      key: 'sort',
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
            checkedChildren="启用"
            unCheckedChildren="禁用"
            onChange={() => handleToggleStatus(record)}
          />
        ) : status === 1 ? (
          <Tag color="success">启用</Tag>
        ) : (
          <Tag color="default">禁用</Tag>
        ),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 170,
      render: (t: string) => formatTime(t),
    },
    {
      title: '操作',
      key: 'action',
      width: 220,
      render: (_, record) => (
        <Space orientation="horizontal" size="small">
          {canCreate && (
            <Button type="link" size="small" onClick={() => openCreate(record.id ?? 0)}>
              添加子分类
            </Button>
          )}
          {canUpdate && (
            <Button type="link" size="small" onClick={() => openEdit(record)}>
              编辑
            </Button>
          )}
          {canDelete && (
            <Popconfirm
              title="确定删除该分类？"
              description="若存在子分类或被商品引用则无法删除。"
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
          <Space wrap>
            <Input.Search
              placeholder="按分类名称搜索"
              allowClear
              style={{ width: 220 }}
              onSearch={setSearchKeyword}
              onChange={(e) => {
                if (!e.target.value) setSearchKeyword('');
              }}
            />
            <Select
              placeholder="状态筛选"
              allowClear
              style={{ width: 120 }}
              value={searchStatus}
              onChange={setSearchStatus}
              options={[
                { label: '全部状态', value: undefined },
                { label: '启用', value: 1 },
                { label: '禁用', value: 0 },
              ]}
            />
            <Button icon={<ReloadOutlined />} onClick={loadCategories}>
              刷新
            </Button>
          </Space>

          {canCreate && (
            <Button type="primary" icon={<PlusOutlined />} onClick={() => openCreate(0)}>
              新增顶级分类
            </Button>
          )}
        </Space>
      </Card>

      <Card>
        <Table<CategoryTreeNode>
          rowKey="id"
          loading={loading}
          columns={columns}
          dataSource={displayTree}
          pagination={false}
          expandable={{ defaultExpandAllRows: true }}
          locale={{ emptyText: '暂无商品分类' }}
        />
      </Card>

      <Modal
        title={
          <Space>
            <AppstoreOutlined />
            <span>{editingCategory ? '编辑商品分类' : '新增商品分类'}</span>
          </Space>
        }
        open={modalOpen}
        onCancel={() => {
          setModalOpen(false);
          setEditingCategory(null);
        }}
        confirmLoading={saving}
        onOk={() => form.submit()}
        destroyOnClose
      >
        <Form<CategoryFormData> form={form} layout="vertical" onFinish={handleFinish}>
          <Form.Item
            name="parent_id"
            label="上级分类"
            rules={[{ required: true, message: '请选择上级分类' }]}
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

          <Form.Item name="image_id" label="分类图标" extra="支持单张分类图标">
            <ImagePicker multiple={false} />
          </Form.Item>

          <Form.Item name="sort" label="排序值" extra="数字越小越靠前，默认 0">
            <InputNumber min={0} max={99999} style={{ width: '100%' }} placeholder="请输入排序值" />
          </Form.Item>

          <Form.Item name="status" label="状态" rules={[{ required: true }]}>
            <Radio.Group>
              <Radio value={1}>启用</Radio>
              <Radio value={0}>禁用</Radio>
            </Radio.Group>
          </Form.Item>
        </Form>
      </Modal>
    </Space>
  );
}
