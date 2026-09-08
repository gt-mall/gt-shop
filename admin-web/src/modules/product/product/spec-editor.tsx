import { useMemo, useRef, useState } from 'react';
import {
  Alert,
  Button,
  Card,
  Divider,
  Input,
  message,
  Popconfirm,
  Space,
  Tag,
  Typography,
} from 'antd';
import { DeleteOutlined, PlusOutlined, ThunderboltOutlined } from '@ant-design/icons';

import type { ProductSKURequest, ProductSpec, ProductSpecValue } from '@/framework/http';
import { generateSKUCombinations, getSpecValueKey } from '../types';

interface SpecEditorProps {
  specs: ProductSpec[];
  skus: ProductSKURequest[];
  disabled?: boolean;
  onSpecsChange: (specs: ProductSpec[]) => void;
  onGenerateSKUs: (newSKUs: ProductSKURequest[]) => void;
}

export function SpecEditor({
  specs,
  skus,
  disabled,
  onSpecsChange,
  onGenerateSKUs,
}: SpecEditorProps) {
  // Compute initial max ID to avoid collisions
  const maxSpecIDRef = useRef(
    specs.reduce((max, s) => Math.max(max, s.id || 0), 0),
  );
  const maxValIDRef = useRef(
    specs.reduce(
      (max, s) =>
        Math.max(max, (s.values || []).reduce((vm, v) => Math.max(vm, v.id || 0), 0)),
      0,
    ),
  );

  const [newOptionInput, setNewOptionInput] = useState<{ [specId: number]: string }>({});
  const hasSavedSKUs = skus.some((sku) => (sku.id ?? 0) > 0);

  // Set of value IDs and spec IDs used by saved SKUs (id > 0)
  const savedUsage = useMemo(() => {
    const savedSpecs = new Set<number>();
    const savedVals = new Set<string>();
    for (const sku of skus) {
      if (sku.id && sku.id > 0 && sku.spec_values) {
        for (const sv of sku.spec_values) {
          if (sv.spec_id) savedSpecs.add(sv.spec_id);
          if (sv.spec_id && sv.value_id) savedVals.add(getSpecValueKey(sv.spec_id, sv.value_id));
        }
      }
    }
    return { savedSpecs, savedVals };
  }, [skus]);

  // Add a new spec dimension (max 3)
  const handleAddSpec = () => {
    if (hasSavedSKUs) {
      message.warning('商品已有已保存的 SKU，不能新增规格维度，可以在现有维度中添加选项。');
      return;
    }
    if (specs.length >= 3) {
      message.warning('最多只能添加 3 个规格维度');
      return;
    }
    const nextID = ++maxSpecIDRef.current;
    const newSpec: ProductSpec = {
      id: nextID,
      name: `规格${specs.length + 1}`,
      values: [],
    };
    onSpecsChange([...specs, newSpec]);
  };

  // Remove a spec dimension
  const handleRemoveSpec = (specID: number) => {
    if (savedUsage.savedSpecs.has(specID)) {
      message.error(
        '该规格已被已保存的SKU引用，无法删除。如需下架相关组合，请在下方SKU列表中将其禁用。',
      );
      return;
    }
    onSpecsChange(specs.filter((s) => s.id !== specID));
  };

  // Update spec name
  const handleSpecNameChange = (specID: number, name: string) => {
    onSpecsChange(
      specs.map((s) => (s.id === specID ? { ...s, name: name } : s)),
    );
  };

  // Add an option value to a spec (max 20)
  const handleAddOption = (specID: number) => {
    const text = (newOptionInput[specID] || '').trim();
    if (!text) return;

    const targetSpec = specs.find((s) => s.id === specID);
    if (!targetSpec) return;

    const values = targetSpec.values || [];
    if (values.length >= 20) {
      message.warning('每个规格最多只能添加 20 个选项');
      return;
    }

    if (values.some((v) => (v.name ?? '').toLowerCase() === text.toLowerCase())) {
      message.warning(`规格选项 "${text}" 已存在`);
      return;
    }

    const nextValID = ++maxValIDRef.current;
    const newVal: ProductSpecValue = { id: nextValID, name: text };

    onSpecsChange(
      specs.map((s) =>
        s.id === specID ? { ...s, values: [...(s.values || []), newVal] } : s,
      ),
    );

    setNewOptionInput((prev) => ({ ...prev, [specID]: '' }));
  };

  // Remove an option value from a spec
  const handleRemoveOption = (specID: number, valID: number) => {
    if (savedUsage.savedVals.has(getSpecValueKey(specID, valID))) {
      message.error(
        '该规格选项已被已保存的SKU使用，无法删除。如不销售该组合，请在下方SKU表格中将其状态设为禁用。',
      );
      return;
    }

    onSpecsChange(
      specs.map((s) =>
        s.id === specID
          ? { ...s, values: (s.values || []).filter((v) => v.id !== valID) }
          : s,
      ),
    );
  };

  // Generate / replenish SKU combinations
  const handleGenerate = () => {
    const validSpecs = specs.filter((s) => s.values && s.values.length > 0);
    if (validSpecs.length === 0) {
      message.warning('请先为规格添加至少一个有效选项');
      return;
    }

    const { skus: generated, totalCount } = generateSKUCombinations(specs, skus);
    if (totalCount > 200) {
      message.error(`生成的SKU组合数量(${totalCount})超过系统上限(200)，请减少规格选项`);
      return;
    }

    onGenerateSKUs(generated);
    message.success(`已生成 ${generated.length} 条SKU组合`);
  };

  return (
    <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
      <Alert
        type="info"
        showIcon
        title="商品多规格设置说明"
        description="支持最多 3 个规格维度，每个维度最多 20 个选项。已有 SKU 的规格身份具有唯一性，删除已被已保存 SKU 使用的选项将被拦截以保证历史数据一致性。"
      />

      {specs.map((spec, index) => {
        const specId = spec.id ?? 0;
        return (
          <Card
            key={specId}
            size="small"
            title={
              <Space>
                <Typography.Text strong>规格维度 {index + 1}：</Typography.Text>
                <Input
                  style={{ width: 160 }}
                  value={spec.name}
                  disabled={disabled}
                  placeholder="如颜色、尺码"
                  maxLength={64}
                  onChange={(e) => handleSpecNameChange(specId, e.target.value)}
                />
              </Space>
            }
            extra={
              !disabled && (
                <Popconfirm
                  title="确定删除该规格维度？"
                  description="若已有已保存的SKU使用该维度则无法删除。"
                  okText="确定"
                  cancelText="取消"
                  onConfirm={() => handleRemoveSpec(specId)}
                >
                  <Button type="link" danger icon={<DeleteOutlined />}>
                    删除规格
                  </Button>
                </Popconfirm>
              )
            }
          >
            <div>
              <Typography.Text type="secondary" style={{ fontSize: 13, marginBottom: 8, display: 'block' }}>
                规格选项列表：
              </Typography.Text>
              <Space wrap size={[8, 8]} align="center">
                {(spec.values || []).map((val) => {
                  const valId = val.id ?? 0;
                  return (
                    <Tag
                      key={valId}
                      closable={!disabled}
                      color="blue"
                      onClose={(e) => {
                        e.preventDefault();
                        handleRemoveOption(specId, valId);
                      }}
                      style={{ padding: '4px 10px', fontSize: 13 }}
                    >
                      {val.name}
                    </Tag>
                  );
                })}

                {!disabled && (spec.values || []).length < 20 && (
                  <Space.Compact style={{ width: 200 }}>
                    <Input
                      placeholder="输入选项按回车"
                      maxLength={64}
                      value={newOptionInput[specId] || ''}
                      onChange={(e) =>
                        setNewOptionInput((prev) => ({ ...prev, [specId]: e.target.value }))
                      }
                      onPressEnter={() => handleAddOption(specId)}
                    />
                    <Button
                      type="primary"
                      icon={<PlusOutlined />}
                      onClick={() => handleAddOption(specId)}
                    >
                      添加
                    </Button>
                  </Space.Compact>
                )}
              </Space>
            </div>
          </Card>
        );
      })}

      {!disabled && (
        <Space style={{ marginTop: 8 }}>
          {specs.length < 3 && (
            <Button icon={<PlusOutlined />} onClick={handleAddSpec} disabled={hasSavedSKUs}>
              添加规格维度 ({specs.length}/3)
            </Button>
          )}

          {specs.length > 0 && (
            <Button
              type="primary"
              icon={<ThunderboltOutlined />}
              onClick={handleGenerate}
            >
              生成 / 补充 SKU 组合
            </Button>
          )}
        </Space>
      )}
    </Space>
  );
}
