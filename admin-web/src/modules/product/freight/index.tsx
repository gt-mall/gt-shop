import { useCallback, useEffect, useState } from 'react';
import { Button, Card, Input, message, Popconfirm, Space, Switch, Table, Tag } from 'antd';
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';

import { api } from '@/framework/http';
import type { FreightTemplateRow } from '@/framework/http';
import { usePermission } from '@/framework/store';

export default function FreightListPage() {
  const navigate = useNavigate();
  const hasPermission = usePermission();

  const canCreate = hasPermission('product:freight:create');
  const canUpdate = hasPermission('product:freight:update');
  const canDelete = hasPermission('product:freight:delete');

  const [loading, setLoading] = useState(false);
  const [list, setList] = useState<FreightTemplateRow[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [keyword, setKeyword] = useState('');

  const loadData = useCallback(
    async (currentPage = page, size = pageSize, kw = keyword) => {
      setLoading(true);
      const res = await api.api.listFreightTemplates({
        page: currentPage,
        page_size: size,
        keyword: kw.trim() || undefined,
      });
      setLoading(false);
      if (res.code === 1 && res.data) {
        setList(res.data);
        setTotal(res.total || 0);
      } else {
        message.error(res.message || '加载运费模板失败');
      }
    },
    [page, pageSize, keyword],
  );

  useEffect(() => {
    loadData();
  }, [loadData]);

  const handleSearch = (value: string) => {
    setKeyword(value);
    setPage(1);
    loadData(1, pageSize, value);
  };

  const handleStatusChange = async (record: FreightTemplateRow, checked: boolean) => {
    const nextStatus = checked ? 1 : 0;
    const res = await api.api.updateFreightTemplateStatus({
      id: record.id ?? 0,
      status: nextStatus,
    });
    if (res.code === 1) {
      message.success(checked ? '模板已启用' : '模板已禁用');
      setList((prev) =>
        prev.map((item) => (item.id === record.id ? { ...item, status: nextStatus } : item)),
      );
    } else {
      message.error(res.message || '更新状态失败');
    }
  };

  const handleDelete = async (id: number) => {
    const res = await api.api.deleteFreightTemplate({ id });
    if (res.code === 1) {
      message.success('删除成功');
      loadData();
    } else {
      message.error(res.message || '删除失败');
    }
  };

  const columns = [
    {
      title: '模板名称',
      dataIndex: 'name',
      key: 'name',
      render: (name: string, record: FreightTemplateRow) => (
        <span className="font-medium text-gray-800">{name}</span>
      ),
    },
    {
      title: '默认运费',
      dataIndex: 'base_fee',
      key: 'base_fee',
      render: (fee: string) => {
        if (!fee || fee === '0' || fee === '0.00') {
          return <Tag color="green">包邮</Tag>;
        }
        return <span className="text-gray-700 font-semibold">¥{fee}</span>;
      },
    },
    {
      title: '地区附加',
      key: 'extra_fees',
      render: (_: any, record: FreightTemplateRow) => {
        if (!record.extra_fee_count || record.extra_fee_count <= 0) {
          return <span className="text-gray-400">无</span>;
        }
        return <Tag color="blue">{record.extra_fee_count} 个地区</Tag>;
      },
    },
    {
      title: '不配送',
      key: 'no_delivery',
      render: (_: any, record: FreightTemplateRow) => {
        if (!record.no_delivery_count || record.no_delivery_count <= 0) {
          return <span className="text-gray-400">无</span>;
        }
        return <Tag color="orange">{record.no_delivery_count} 个地区</Tag>;
      },
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: number, record: FreightTemplateRow) => (
        <Switch
          checked={status === 1}
          disabled={!canUpdate}
          checkedChildren="启用"
          unCheckedChildren="禁用"
          onChange={(checked) => handleStatusChange(record, checked)}
        />
      ),
    },
    {
      title: '操作',
      key: 'action',
      render: (_: any, record: FreightTemplateRow) => (
        <Space size="middle">
          {canUpdate && (
            <Button
              type="link"
              size="small"
              onClick={() => navigate(`/product/freight/${record.id}/edit`)}
            >
              编辑
            </Button>
          )}
          {canDelete && (
            <Popconfirm
              title="确定要删除该运费模板吗？"
              description="删除后无法恢复，正在被商品使用的模板不可删除"
              onConfirm={() => handleDelete(record.id ?? 0)}
              okText="删除"
              cancelText="取消"
              okButtonProps={{ danger: true }}
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
    <Card
      title="运费模板"
      extra={
        <Space>
          <Input.Search
            placeholder="搜索模板名称"
            allowClear
            onSearch={handleSearch}
            style={{ width: 220 }}
          />
          <Button icon={<ReloadOutlined />} onClick={() => loadData()}>
            刷新
          </Button>
          {canCreate && (
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => navigate('/product/freight/create')}
            >
              新建模板
            </Button>
          )}
        </Space>
      }
    >
      <Table
        rowKey="id"
        columns={columns}
        dataSource={list}
        loading={loading}
        pagination={{
          current: page,
          pageSize,
          total,
          showSizeChanger: true,
          showQuickJumper: true,
          showTotal: (t) => `共 ${t} 条记录`,
          onChange: (p, s) => {
            setPage(p);
            setPageSize(s);
            loadData(p, s);
          },
        }}
      />
    </Card>
  );
}
