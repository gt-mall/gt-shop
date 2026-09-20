import { useEffect, useState } from 'react';
import {
  Alert,
  Button,
  Form,
  Input,
  InputNumber,
  Popconfirm,
  Space,
  Spin,
  Switch,
  Table,
  Typography,
  message,
} from 'antd';
import { PlusOutlined, DeleteOutlined } from '@ant-design/icons';
import { api, type HotKeywordItem } from '@/framework/http';
import { usePermission } from '@/framework/store';

const { Text } = Typography;

interface TableRowItem extends HotKeywordItem {
  key: string;
}

export default function SearchSettingPage() {
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [items, setItems] = useState<TableRowItem[]>([]);

  const hasPermission = usePermission();
  const canUpdate = hasPermission('system:setting:update');

  const loadData = async () => {
    setLoading(true);
    const result = await api.api.getSystemSearchSetting();
    setLoading(false);
    if (result.code !== 1 || !result.data) {
      message.error(result.message || '获取搜索设置失败');
      return;
    }
    const rawList = result.data.hot_keywords || [];
    const formatted: TableRowItem[] = rawList.map((item, idx) => ({
      key: `row_${Date.now()}_${idx}_${Math.random().toString(36).slice(2, 7)}`,
      keyword: item.keyword || '',
      sort: typeof item.sort === 'number' ? item.sort : 10,
      enabled: item.enabled ?? true,
    }));
    setItems(formatted);
  };

  useEffect(() => {
    let active = true;
    const run = async () => {
      setLoading(true);
      const result = await api.api.getSystemSearchSetting();
      if (!active) return;
      setLoading(false);
      if (result.code !== 1 || !result.data) {
        message.error(result.message || '获取搜索设置失败');
        return;
      }
      const rawList = result.data.hot_keywords || [];
      const formatted: TableRowItem[] = rawList.map((item, idx) => ({
        key: `row_${Date.now()}_${idx}_${Math.random().toString(36).slice(2, 7)}`,
        keyword: item.keyword || '',
        sort: typeof item.sort === 'number' ? item.sort : 10,
        enabled: item.enabled ?? true,
      }));
      setItems(formatted);
    };
    void run();
    return () => {
      active = false;
    };
  }, []);

  const handleAdd = () => {
    if (items.length >= 20) {
      message.warning('最多只支持配置 20 条热门搜索词');
      return;
    }
    const newItem: TableRowItem = {
      key: `row_${Date.now()}_${Math.random().toString(36).slice(2, 7)}`,
      keyword: '',
      sort: (items.length + 1) * 10,
      enabled: true,
    };
    setItems([...items, newItem]);
  };

  const handleDelete = (index: number) => {
    const next = [...items];
    next.splice(index, 1);
    setItems(next);
  };

  const handleKeywordChange = (index: number, val: string) => {
    const next = [...items];
    next[index] = { ...next[index], keyword: val };
    setItems(next);
  };

  const handleSortChange = (index: number, val: number | null) => {
    const next = [...items];
    next[index] = { ...next[index], sort: val ?? 0 };
    setItems(next);
  };

  const handleEnabledChange = (index: number, val: boolean) => {
    const next = [...items];
    next[index] = { ...next[index], enabled: val };
    setItems(next);
  };

  const handleSave = async () => {
    // 数据校验
    const cleanedKeywords: HotKeywordItem[] = [];
    const seen = new Set<string>();

    for (let i = 0; i < items.length; i++) {
      const row = items[i];
      const trimmed = (row.keyword || '').trim().replace(/\s+/g, ' ');
      if (!trimmed) {
        message.error(`第 ${i + 1} 行搜索词不能为空`);
        return;
      }
      if (trimmed.length > 50) {
        message.error(`第 ${i + 1} 行搜索词超出 50 个字符上限`);
        return;
      }
      const lower = trimmed.toLowerCase();
      if (seen.has(lower)) {
        message.error(`存在重复的热门搜索词：${trimmed}`);
        return;
      }
      seen.add(lower);
      cleanedKeywords.push({
        keyword: trimmed,
        sort: typeof row.sort === 'number' ? row.sort : 10,
        enabled: !!row.enabled,
      });
    }

    if (cleanedKeywords.length > 20) {
      message.error('热门搜索词数量超过 20 条上限');
      return;
    }

    setSaving(true);
    const result = await api.api.saveSystemSearchSetting({
      hot_keywords: cleanedKeywords,
    });
    setSaving(false);

    if (result.code !== 1) {
      message.error(result.message || '保存搜索设置失败');
      return;
    }

    message.success('搜索设置已保存');
    await loadData();
  };

  const columns = [
    {
      title: '序号',
      key: 'index',
      width: 70,
      render: (_: unknown, __: unknown, index: number) => index + 1,
    },
    {
      title: (
        <Space>
          <span>推荐搜索词</span>
          <Text type="secondary" style={{ fontSize: 12 }}>
            (最多 50 字)
          </Text>
        </Space>
      ),
      dataIndex: 'keyword',
      key: 'keyword',
      render: (val: string, _: unknown, index: number) => (
        <Input
          value={val}
          maxLength={50}
          placeholder="请输入热门推荐关键词，如：轻薄笔记本"
          onChange={(e) => handleKeywordChange(index, e.target.value)}
          disabled={!canUpdate}
          style={{ maxWidth: 360 }}
        />
      ),
    },
    {
      title: (
        <Space>
          <span>排序权重</span>
          <Text type="secondary" style={{ fontSize: 12 }}>
            (越小越靠前)
          </Text>
        </Space>
      ),
      dataIndex: 'sort',
      key: 'sort',
      width: 140,
      render: (val: number, _: unknown, index: number) => (
        <InputNumber
          value={val}
          min={0}
          max={99999}
          precision={0}
          onChange={(v) => handleSortChange(index, v)}
          disabled={!canUpdate}
          style={{ width: 100 }}
        />
      ),
    },
    {
      title: '启用状态',
      dataIndex: 'enabled',
      key: 'enabled',
      width: 110,
      render: (val: boolean, _: unknown, index: number) => (
        <Switch
          checked={val}
          checkedChildren="启用"
          unCheckedChildren="停用"
          onChange={(v) => handleEnabledChange(index, v)}
          disabled={!canUpdate}
        />
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 100,
      render: (_: unknown, __: unknown, index: number) => (
        <Popconfirm
          title="确定删除此条热门搜索词？"
          onConfirm={() => handleDelete(index)}
          disabled={!canUpdate}
        >
          <Button
            type="link"
            danger
            icon={<DeleteOutlined />}
            disabled={!canUpdate}
          >
            删除
          </Button>
        </Popconfirm>
      ),
    },
  ];

  return (
    <Spin spinning={loading}>
      <div style={{ maxWidth: 960 }}>
        <Alert
          type="info"
          showIcon
          message="热门搜索 / 推荐搜索词配置说明"
          description="在此配置小程序搜索页发现态展示的热门搜索标签。最多支持 20 条，按排序权重升序排列。仅已启用且非空的词会对小程序端用户展示，保存后代际缓存立即生效。"
          style={{ marginBottom: 20 }}
        />

        <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Space>
            <Button
              type="dashed"
              icon={<PlusOutlined />}
              onClick={handleAdd}
              disabled={!canUpdate || items.length >= 20}
            >
              添加热门搜索词 ({items.length}/20)
            </Button>
          </Space>
        </div>

        <Table
          dataSource={items}
          columns={columns}
          pagination={false}
          rowKey="key"
          size="middle"
          bordered
          locale={{ emptyText: '暂无配置热门搜索词，点击上方按钮添加' }}
        />

        <div style={{ marginTop: 24 }}>
          <Space>
            <Button
              type="primary"
              onClick={handleSave}
              loading={saving}
              disabled={!canUpdate}
            >
              保存设置
            </Button>
            <Button onClick={loadData} disabled={loading || saving}>
              重置
            </Button>
          </Space>
        </div>
      </div>
    </Spin>
  );
}
