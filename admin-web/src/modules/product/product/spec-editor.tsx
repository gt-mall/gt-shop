import React, { useMemo, useRef, useState } from 'react';
import {
  Button,
  Input,
  message,
  Popconfirm,
  Tag,
  type InputRef,
} from 'antd';
import {
  DeleteOutlined,
  HolderOutlined,
  PlusOutlined,
  ThunderboltFilled,
} from '@ant-design/icons';
import {
  closestCenter,
  DndContext,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
} from '@dnd-kit/core';
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';

import type { ProductSKURequest, ProductSpec, ProductSpecValue } from '@/framework/http';
import { generateSKUCombinations, getSpecValueKey } from '../types';

interface SpecEditorProps {
  specs: ProductSpec[];
  skus: ProductSKURequest[];
  disabled?: boolean;
  onSpecsChange: (specs: ProductSpec[]) => void;
  onGenerateSKUs: (newSKUs: ProductSKURequest[]) => void;
}

interface SortableSpecCardProps {
  spec: ProductSpec;
  index: number;
  disabled?: boolean;
  savedUsage: { savedSpecs: Set<number>; savedVals: Set<string> };
  newOptionValue: string;
  onNewOptionChange: (val: string) => void;
  onAddOption: () => void;
  onRemoveOption: (valId: number) => void;
  onSpecNameChange: (name: string) => void;
  onRemoveSpec: () => void;
  onFocusInput: () => void;
  setInputRef: (el: InputRef | null) => void;
}

function SortableSpecCard({
  spec,
  index,
  disabled,
  newOptionValue,
  onNewOptionChange,
  onAddOption,
  onRemoveOption,
  onSpecNameChange,
  onRemoveSpec,
  onFocusInput,
  setInputRef,
}: SortableSpecCardProps) {
  const specId = spec.id ?? 0;
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({
    id: specId,
    disabled: disabled,
  });

  const style: React.CSSProperties = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.6 : 1,
    background: '#ffffff',
    border: isDragging ? '1px dashed #6366f1' : '1px solid #e2e8f0',
    borderRadius: 8,
    padding: '16px 20px',
    marginBottom: 16,
    boxShadow: isDragging ? '0 10px 15px -3px rgba(0, 0, 0, 0.1)' : 'none',
    position: 'relative',
    zIndex: isDragging ? 999 : undefined,
  };

  const values = spec.values || [];

  return (
    <div ref={setNodeRef} style={style}>
      {/* 顶部行：拖拽把手、规格维度标题、名称输入框、删除按钮 */}
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          gap: 12,
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, flex: 1, minWidth: 0 }}>
          <span
            {...attributes}
            {...listeners}
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              justifyContent: 'center',
              cursor: disabled ? 'default' : isDragging ? 'grabbing' : 'grab',
              color: '#94a3b8',
              padding: '2px',
              userSelect: 'none',
              touchAction: 'none',
            }}
            title={disabled ? undefined : '按住拖拽以调整规格顺序'}
          >
            <HolderOutlined style={{ fontSize: 18 }} />
          </span>

          <span style={{ fontSize: 14, fontWeight: 600, color: '#0f172a', whiteSpace: 'nowrap' }}>
            规格维度 {index + 1}
          </span>

          <Input
            style={{ width: 180, borderRadius: 6 }}
            value={spec.name}
            disabled={disabled}
            placeholder="如颜色、尺码"
            maxLength={20}
            showCount
            onChange={(e) => onSpecNameChange(e.target.value)}
          />
        </div>

        {!disabled && (
          <Popconfirm
            title="确定删除该规格维度？"
            description="若已有已保存的SKU使用该维度则无法删除。"
            okText="确定"
            cancelText="取消"
            onConfirm={onRemoveSpec}
          >
            <Button
              type="text"
              danger
              icon={<DeleteOutlined />}
              style={{
                display: 'inline-flex',
                alignItems: 'center',
                gap: 4,
                padding: '4px 8px',
                fontSize: 14,
              }}
            >
              删除
            </Button>
          </Popconfirm>
        )}
      </div>

      {/* 选项行：规格选项数量及标签列表 */}
      <div style={{ marginTop: 14 }}>
        <div style={{ fontSize: 13, color: '#64748b', marginBottom: 10 }}>
          规格选项 ({values.length}/50)
        </div>

        <div style={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 10 }}>
          {values.map((val) => {
            const valId = val.id ?? 0;
            return (
              <Tag
                key={valId}
                closable={!disabled}
                onClose={(e) => {
                  e.preventDefault();
                  onRemoveOption(valId);
                }}
                style={{
                  margin: 0,
                  padding: '4px 12px',
                  fontSize: 13,
                  lineHeight: '22px',
                  background: '#eff6ff',
                  borderColor: '#bfdbfe',
                  color: '#2563eb',
                  borderRadius: 6,
                  display: 'inline-flex',
                  alignItems: 'center',
                  gap: 6,
                }}
              >
                {val.name}
              </Tag>
            );
          })}

          {!disabled && values.length < 50 && (
            <div style={{ display: 'inline-flex', alignItems: 'center', gap: 10 }}>
              <Input
                ref={setInputRef}
                placeholder="+ 输入选项，按回车添加"
                maxLength={30}
                value={newOptionValue}
                onChange={(e) => onNewOptionChange(e.target.value)}
                onPressEnter={onAddOption}
                style={{
                  width: 180,
                  height: 32,
                  borderRadius: 6,
                  borderStyle: 'dashed',
                  borderColor: '#cbd5e1',
                  background: '#ffffff',
                  fontSize: 13,
                }}
              />

              <Button
                onClick={() => {
                  if (newOptionValue.trim()) {
                    onAddOption();
                  } else {
                    onFocusInput();
                  }
                }}
                icon={<PlusOutlined style={{ fontSize: 12 }} />}
                style={{
                  height: 32,
                  borderRadius: 6,
                  borderColor: '#c7d2fe',
                  background: '#f5f3ff',
                  color: '#4f46e5',
                  fontWeight: 500,
                  fontSize: 13,
                }}
              >
                添加选项
              </Button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
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
  const inputRefs = useRef<{ [specId: number]: InputRef | null }>({});
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

  // Add an option value to a spec (max 50)
  const handleAddOption = (specID: number) => {
    const text = (newOptionInput[specID] || '').trim();
    if (!text) return;

    const targetSpec = specs.find((s) => s.id === specID);
    if (!targetSpec) return;

    const values = targetSpec.values || [];
    if (values.length >= 50) {
      message.warning('每个规格最多只能添加 50 个选项');
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

  // Drag & drop sensors and handler
  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: {
        distance: 5,
      },
    }),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    }),
  );

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (over && active.id !== over.id) {
      const oldIndex = specs.findIndex((s) => s.id === active.id);
      const newIndex = specs.findIndex((s) => s.id === over.id);
      if (oldIndex !== -1 && newIndex !== -1) {
        onSpecsChange(arrayMove(specs, oldIndex, newIndex));
      }
    }
  };

  return (
    <div style={{ width: '100%' }}>
      <DndContext
        sensors={sensors}
        collisionDetection={closestCenter}
        onDragEnd={handleDragEnd}
      >
        <SortableContext
          items={specs.map((s) => s.id ?? 0)}
          strategy={verticalListSortingStrategy}
        >
          {specs.map((spec, index) => {
            const specId = spec.id ?? 0;
            return (
              <SortableSpecCard
                key={specId}
                spec={spec}
                index={index}
                disabled={disabled}
                savedUsage={savedUsage}
                newOptionValue={newOptionInput[specId] || ''}
                onNewOptionChange={(val) =>
                  setNewOptionInput((prev) => ({ ...prev, [specId]: val }))
                }
                onAddOption={() => handleAddOption(specId)}
                onRemoveOption={(valId) => handleRemoveOption(specId, valId)}
                onSpecNameChange={(name) => handleSpecNameChange(specId, name)}
                onRemoveSpec={() => handleRemoveSpec(specId)}
                onFocusInput={() => inputRefs.current[specId]?.focus()}
                setInputRef={(el) => {
                  inputRefs.current[specId] = el;
                }}
              />
            );
          })}
        </SortableContext>
      </DndContext>

      {!disabled && (
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 16,
            marginTop: 20,
            flexWrap: 'wrap',
          }}
        >
          <Button
            icon={<PlusOutlined />}
            onClick={handleAddSpec}
            disabled={hasSavedSKUs || specs.length >= 3}
            style={{
              height: 38,
              borderRadius: 6,
              borderStyle: 'dashed',
              borderColor: '#818cf8',
              background: '#f5f3ff',
              color: '#4f46e5',
              fontWeight: 500,
              fontSize: 14,
            }}
          >
            添加规格维度 ({specs.length}/3)
          </Button>

          {specs.length > 0 && (
            <div style={{ display: 'inline-flex', alignItems: 'center', gap: 12 }}>
              <Button
                type="primary"
                icon={<ThunderboltFilled />}
                onClick={handleGenerate}
                style={{
                  height: 38,
                  borderRadius: 6,
                  background: '#4f46e5',
                  borderColor: '#4f46e5',
                  fontWeight: 500,
                  fontSize: 14,
                }}
              >
                生成 / 补充 SKU 组合
              </Button>
              <span style={{ fontSize: 13, color: '#64748b' }}>
                根据当前规格选项，自动生成全部 SKU 组合
              </span>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
