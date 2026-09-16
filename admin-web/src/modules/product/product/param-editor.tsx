import React, { useCallback } from 'react';
import {
  AutoComplete,
  Button,
  Empty,
  Input,
  Popconfirm,
  Space,
  Table,
  Typography,
} from 'antd';
import {
  ArrowDownOutlined,
  ArrowUpOutlined,
  DeleteOutlined,
  PlusOutlined,
} from '@ant-design/icons';
import type { ProductParam } from '@/framework/http';

const COMMON_PARAM_OPTIONS = [
  { value: '品牌' },
  { value: '型号' },
  { value: '产地' },
  { value: '材质/成分' },
  { value: '保质期' },
  { value: '净含量' },
  { value: '包装规格' },
  { value: '适用季节' },
  { value: '适用人群' },
  { value: '储藏方法' },
  { value: '执行标准' },
  { value: '生产日期' },
  { value: '功能特点' },
  { value: '尺寸/规格' },
  { value: '商品毛重' },
  { value: '适用场景' },
  { value: '颜色分类' },
  { value: '包装形式' },
  { value: '生产许可证编号' },
];

interface ParamEditorProps {
  params: ProductParam[];
  onChange: (params: ProductParam[]) => void;
  disabled?: boolean;
}

export const ParamEditor: React.FC<ParamEditorProps> = ({
  params,
  onChange,
  disabled = false,
}) => {
  const handleAddRow = useCallback(() => {
    onChange([...params, { name: '', value: '' }]);
  }, [params, onChange]);

  const handleUpdate = useCallback(
    (index: number, field: 'name' | 'value', text: string) => {
      const next = params.map((item, i) => {
        if (i !== index) return item;
        return { ...item, [field]: text };
      });
      onChange(next);
    },
    [params, onChange],
  );

  const handleDelete = useCallback(
    (index: number) => {
      const next = params.filter((_, i) => i !== index);
      onChange(next);
    },
    [params, onChange],
  );

  const handleMove = useCallback(
    (index: number, direction: 'up' | 'down') => {
      const targetIndex = direction === 'up' ? index - 1 : index + 1;
      if (targetIndex < 0 || targetIndex >= params.length) return;
      const next = [...params];
      const temp = next[index];
      next[index] = next[targetIndex];
      next[targetIndex] = temp;
      onChange(next);
    },
    [params, onChange],
  );

  const handleClearAll = useCallback(() => {
    onChange([]);
  }, [onChange]);

  const columns = [
    {
      title: '序号',
      dataIndex: 'index',
      width: 60,
      align: 'center' as const,
      render: (_: unknown, __: ProductParam, index: number) => (
        <Typography.Text type="secondary">{index + 1}</Typography.Text>
      ),
    },
    {
      title: (
        <Space>
          <span style={{ color: '#EF4444' }}>*</span>
          <span>参数名称</span>
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            (支持输入或下拉选择常用参数)
          </Typography.Text>
        </Space>
      ),
      dataIndex: 'name',
      width: '40%',
      render: (name: string | undefined, _: ProductParam, index: number) => (
        <AutoComplete
          value={name ?? ''}
          options={COMMON_PARAM_OPTIONS}
          placeholder="请输入或选择参数名称"
          maxLength={100}
          disabled={disabled}
          allowClear={!disabled}
          status={!(name ?? '').trim() && params.length > 0 ? 'warning' : undefined}
          onChange={(val) => handleUpdate(index, 'name', val)}
          onSelect={(val) => handleUpdate(index, 'name', val)}
          filterOption={(inputValue, option) =>
            !inputValue ||
            (option?.value ?? '')
              .toLowerCase()
              .includes((inputValue ?? '').trim().toLowerCase())
          }
          style={{ width: '100%' }}
        />
      ),
    },
    {
      title: (
        <Space>
          <span>参数值</span>
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            (如：浙江杭州、12个月、100%纯棉)
          </Typography.Text>
        </Space>
      ),
      dataIndex: 'value',
      render: (value: string | undefined, _: ProductParam, index: number) => (
        <Input
          value={value ?? ''}
          placeholder="请输入参数值"
          maxLength={500}
          disabled={disabled}
          onChange={(e) => handleUpdate(index, 'value', e.target.value)}
        />
      ),
    },
    {
      title: '操作',
      dataIndex: 'action',
      width: 140,
      align: 'center' as const,
      render: (_: unknown, __: ProductParam, index: number) => (
        <Space orientation="horizontal" size={2}>
          <Button
            type="text"
            size="small"
            icon={<ArrowUpOutlined />}
            disabled={disabled || index === 0}
            title="上移"
            onClick={() => handleMove(index, 'up')}
          />
          <Button
            type="text"
            size="small"
            icon={<ArrowDownOutlined />}
            disabled={disabled || index === params.length - 1}
            title="下移"
            onClick={() => handleMove(index, 'down')}
          />
          <Button
            type="text"
            danger
            size="small"
            icon={<DeleteOutlined />}
            disabled={disabled}
            title="删除"
            onClick={() => handleDelete(index)}
          />
        </Space>
      ),
    },
  ];

  const dataSource = params.map((item, index) => ({
    ...item,
    key: `param-${index}`,
  }));

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <Table
        size="small"
        bordered
        pagination={false}
        columns={columns}
        dataSource={dataSource}
        locale={{
          emptyText: (
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description="暂未添加商品参数，点击下方按钮添加"
            />
          ),
        }}
      />

      {!disabled && (
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Button
            type="dashed"
            icon={<PlusOutlined />}
            onClick={handleAddRow}
            style={{ width: 160 }}
          >
            添加自定义参数
          </Button>

          {params.length > 0 && (
            <Popconfirm
              title="确定要清空全部参数吗？"
              okText="确定"
              cancelText="取消"
              onConfirm={handleClearAll}
            >
              <Button type="link" danger size="small">
                清空全部
              </Button>
            </Popconfirm>
          )}
        </div>
      )}
    </div>
  );
};
