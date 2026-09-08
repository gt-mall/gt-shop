import { useMemo, useState } from 'react';
import {
  Button,
  Card,
  Input,
  InputNumber,
  message,
  Modal,
  Popconfirm,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  Tooltip,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { CheckOutlined, CloseOutlined, DeleteOutlined, EditOutlined } from '@ant-design/icons';

import type { ProductSKURequest, ProductSpec } from '@/framework/http';
import { ImagePicker } from '@/modules/system/file/picker';
import { getSKUComboKey, getSpecValueKey, normalizeMoney } from '../types';

interface SKUTableProps {
  specs: ProductSpec[];
  skus: ProductSKURequest[];
  disabled?: boolean;
  onSKUsChange: (skus: ProductSKURequest[]) => void;
}

export function SKUTable({ specs, skus, disabled, onSKUsChange }: SKUTableProps) {
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);

  // Batch edit modals
  const [batchModalType, setBatchModalType] = useState<
    'price' | 'market_price' | 'cost_price' | 'unit' | null
  >(null);
  const [batchInputValue, setBatchInputValue] = useState<string>('');

  // Value IDs are unique within a spec, so names must use both IDs.
  const nameMap = useMemo(() => {
    const specName = new Map<number, string>();
    const valName = new Map<string, string>();
    for (const s of specs) {
      if (s.id) specName.set(s.id, s.name ?? '');
      for (const v of s.values || []) {
        if (s.id && v.id) valName.set(getSpecValueKey(s.id, v.id), v.name ?? '');
      }
    }
    return { specName, valName };
  }, [specs]);

  const rowKeyOf = (record: ProductSKURequest, index: number): string => {
    if (record.id && record.id > 0) return `saved_${record.id}`;
    if (record.spec_values && record.spec_values.length > 0) {
      return `combo_${getSKUComboKey(record.spec_values)}`;
    }
    return `idx_${index}`;
  };

  const updateSKUField = <K extends keyof ProductSKURequest>(
    index: number,
    field: K,
    value: ProductSKURequest[K],
  ) => {
    const next = [...skus];
    next[index] = { ...next[index], [field]: value };
    onSKUsChange(next);
  };

  // Remove unsaved SKU
  const handleRemoveUnsavedSKU = (index: number) => {
    const target = skus[index];
    if (target.id && target.id > 0) {
      message.warning('已保存的SKU不能直接删除，可通过禁用状态使其停止销售');
      return;
    }
    const next = skus.filter((_, i) => i !== index);
    onSKUsChange(next);
  };

  // Batch update
  const applyBatchUpdate = (
    field: 'price' | 'market_price' | 'cost_price' | 'unit' | 'status',
    val: any,
  ) => {
    if (selectedRowKeys.length === 0) {
      message.warning('请先勾选需要批量操作的SKU行');
      return;
    }
    const keySet = new Set(selectedRowKeys);
    const next = skus.map((sku, idx) => {
      const k = rowKeyOf(sku, idx);
      if (keySet.has(k)) {
        if (field === 'price' || field === 'market_price' || field === 'cost_price') {
          return { ...sku, [field]: normalizeMoney(val) };
        }
        return { ...sku, [field]: val };
      }
      return sku;
    });
    onSKUsChange(next);
    message.success(`已批量修改 ${selectedRowKeys.length} 个SKU`);
  };

  const handleConfirmBatchModal = () => {
    if (!batchModalType) return;
    if (batchModalType === 'unit') {
      applyBatchUpdate('unit', batchInputValue.trim() || undefined);
    } else {
      applyBatchUpdate(batchModalType, batchInputValue);
    }
    setBatchModalType(null);
    setBatchInputValue('');
  };

  const columns: ColumnsType<ProductSKURequest> = [
    {
      title: '规格组合',
      key: 'specs',
      width: 180,
      render: (_, record) => {
        if (!record.spec_values || record.spec_values.length === 0) {
          return <Tag color="default">默认规格</Tag>;
        }
        return (
          <Space wrap size={[4, 4]}>
            {record.spec_values.map((sv) => {
              const valId = sv.value_id ?? 0;
              return (
                <Tag key={`${sv.spec_id ?? 0}-${valId}`} color="blue">
                  {nameMap.valName.get(getSpecValueKey(sv.spec_id ?? 0, valId)) || (valId ? `ID:${valId}` : '')}
                </Tag>
              );
            })}
          </Space>
        );
      },
    },
    {
      title: 'SKU图片',
      key: 'image_id',
      width: 100,
      render: (_, record, index) => (
        <ImagePicker
          multiple={false}
          disabled={disabled}
          value={record.image_id ?? undefined}
          onChange={(id) => updateSKUField(index, 'image_id', (Array.isArray(id) ? id[0] : id) || undefined)}
        />
      ),
    },
    {
      title: '销售价(元) *',
      key: 'price',
      width: 130,
      render: (_, record, index) => (
        <InputNumber
          style={{ width: '100%' }}
          stringMode
          precision={2}
          min="0"
          step="0.01"
          disabled={disabled}
          placeholder="0.00"
          value={record.price}
          onChange={(v) => updateSKUField(index, 'price', v ? String(v) : '0.00')}
        />
      ),
    },
    {
      title: '划线价(元)',
      key: 'market_price',
      width: 130,
      render: (_, record, index) => (
        <InputNumber
          style={{ width: '100%' }}
          stringMode
          precision={2}
          min="0"
          step="0.01"
          disabled={disabled}
          placeholder="0.00"
          value={record.market_price}
          onChange={(v) => updateSKUField(index, 'market_price', v ? String(v) : '0.00')}
        />
      ),
    },
    {
      title: '成本价(元)',
      key: 'cost_price',
      width: 130,
      render: (_, record, index) => (
        <InputNumber
          style={{ width: '100%' }}
          stringMode
          precision={2}
          min="0"
          step="0.01"
          disabled={disabled}
          placeholder="0.00"
          value={record.cost_price}
          onChange={(v) => updateSKUField(index, 'cost_price', v ? String(v) : '0.00')}
        />
      ),
    },
    {
      title: '商家SKU编码',
      key: 'sku_code',
      width: 150,
      render: (_, record, index) => (
        <Input
          placeholder="可选，全局唯一"
          disabled={disabled}
          value={record.sku_code ?? ''}
          onChange={(e) => updateSKUField(index, 'sku_code', e.target.value || undefined)}
        />
      ),
    },
    {
      title: '条形码',
      key: 'barcode',
      width: 140,
      render: (_, record, index) => (
        <Input
          placeholder="可选"
          disabled={disabled}
          value={record.barcode ?? ''}
          onChange={(e) => updateSKUField(index, 'barcode', e.target.value || undefined)}
        />
      ),
    },
    {
      title: '单位',
      key: 'unit',
      width: 90,
      render: (_, record, index) => (
        <Input
          placeholder="件/个"
          disabled={disabled}
          value={record.unit ?? ''}
          onChange={(e) => updateSKUField(index, 'unit', e.target.value || undefined)}
        />
      ),
    },
    {
      title: '重量(克)',
      key: 'weight_gram',
      width: 110,
      render: (_, record, index) => (
        <InputNumber
          style={{ width: '100%' }}
          min={0}
          disabled={disabled}
          placeholder="克"
          value={record.weight_gram ?? undefined}
          onChange={(v) => updateSKUField(index, 'weight_gram', typeof v === 'number' ? v : undefined)}
        />
      ),
    },
    {
      title: '状态',
      key: 'status',
      width: 90,
      align: 'center',
      render: (_, record, index) => (
        <Switch
          checked={record.status === 1}
          checkedChildren="启用"
          unCheckedChildren="禁用"
          disabled={disabled}
          onChange={(checked) => updateSKUField(index, 'status', checked ? 1 : 0)}
        />
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 80,
      align: 'center',
      render: (_, record, index) => {
        const isSaved = record.id && record.id > 0;
        if (isSaved) {
          return (
            <Tooltip title="已入库的SKU不可物理删除，可切换启用/禁用状态">
              <span style={{ color: '#9CA3AF', fontSize: 12 }}>已入库</span>
            </Tooltip>
          );
        }
        if (disabled) return null;
        return (
          <Popconfirm
            title="确定移除该新组合？"
            okText="确定"
            cancelText="取消"
            onConfirm={() => handleRemoveUnsavedSKU(index)}
          >
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>
              移除
            </Button>
          </Popconfirm>
        );
      },
    },
  ];

  const hasSelected = selectedRowKeys.length > 0;

  return (
    <Card size="small" style={{ marginTop: 12 }}>
      {!disabled && (
        <div style={{ marginBottom: 12 }}>
          <Space wrap align="center">
            <span style={{ color: '#6B7280', fontSize: 13 }}>
              {hasSelected ? `已选中 ${selectedRowKeys.length} 项` : '勾选表格可批量设置：'}
            </span>
            <Button
              size="small"
              disabled={!hasSelected}
              onClick={() => {
                setBatchModalType('price');
                setBatchInputValue('0.00');
              }}
            >
              批量销售价
            </Button>
            <Button
              size="small"
              disabled={!hasSelected}
              onClick={() => {
                setBatchModalType('market_price');
                setBatchInputValue('0.00');
              }}
            >
              批量划线价
            </Button>
            <Button
              size="small"
              disabled={!hasSelected}
              onClick={() => {
                setBatchModalType('cost_price');
                setBatchInputValue('0.00');
              }}
            >
              批量成本价
            </Button>
            <Button
              size="small"
              disabled={!hasSelected}
              onClick={() => {
                setBatchModalType('unit');
                setBatchInputValue('件');
              }}
            >
              批量单位
            </Button>
            <Button
              size="small"
              disabled={!hasSelected}
              icon={<CheckOutlined />}
              onClick={() => applyBatchUpdate('status', 1)}
            >
              批量启用
            </Button>
            <Button
              size="small"
              disabled={!hasSelected}
              icon={<CloseOutlined />}
              onClick={() => applyBatchUpdate('status', 0)}
            >
              批量禁用
            </Button>
          </Space>
        </div>
      )}

      <Table<ProductSKURequest>
        rowKey={(record, idx) => rowKeyOf(record, idx ?? 0)}
        rowSelection={
          disabled
            ? undefined
            : {
                selectedRowKeys,
                onChange: (keys) => setSelectedRowKeys(keys),
              }
        }
        columns={columns}
        dataSource={skus}
        pagination={false}
        scroll={{ x: 1350 }}
        size="small"
        locale={{ emptyText: '暂无SKU数据，请在上方添加规格并生成组合' }}
      />

      {/* Batch input modal */}
      <Modal
        title={
          batchModalType === 'price'
            ? '批量设置销售价格 (元)'
            : batchModalType === 'market_price'
            ? '批量设置划线价格 (元)'
            : batchModalType === 'cost_price'
            ? '批量设置成本价格 (元)'
            : '批量设置销售单位'
        }
        open={batchModalType !== null}
        onOk={handleConfirmBatchModal}
        onCancel={() => {
          setBatchModalType(null);
          setBatchInputValue('');
        }}
        destroyOnClose
      >
        <div style={{ padding: '16px 0' }}>
          {batchModalType === 'unit' ? (
            <Input
              placeholder="请输入单位名称，如件、个、瓶"
              maxLength={32}
              value={batchInputValue}
              onChange={(e) => setBatchInputValue(e.target.value)}
            />
          ) : (
            <InputNumber
              style={{ width: '100%' }}
              stringMode
              precision={2}
              min="0"
              step="0.01"
              placeholder="0.00"
              value={batchInputValue}
              onChange={(v) => setBatchInputValue(v ? String(v) : '0.00')}
            />
          )}
        </div>
      </Modal>
    </Card>
  );
}
