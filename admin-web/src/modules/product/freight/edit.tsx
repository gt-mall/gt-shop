import { useEffect, useState } from 'react';
import {
  Button,
  Card,
  Divider,
  Form,
  Input,
  InputNumber,
  message,
  Popconfirm,
  Radio,
  Space,
  Spin,
  Table,
  Tag,
  Typography,
} from 'antd';
import { ArrowLeftOutlined, DeleteOutlined, PlusOutlined, SaveOutlined } from '@ant-design/icons';
import { useNavigate, useParams } from 'react-router-dom';

import { api } from '@/framework/http';
import type { FreightExtraFeeRule, FreightNoDeliveryRule } from '@/framework/http';
import {
  formatRegionLabel,
  loadRegions,
  RegionCascader,
} from './region-cascader';

const { Title, Text } = Typography;

interface ExtraFeeRow {
  key: string;
  provinceCode: string;
  cityCode: string;
  fee: string;
}

interface NoDeliveryItem {
  key: string;
  provinceCode: string;
  cityCode: string;
}

export default function FreightEditPage() {
  const { id } = useParams<{ id?: string }>();
  const navigate = useNavigate();
  const isEdit = Boolean(id);

  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  // 表单状态
  const [name, setName] = useState('');
  const [baseFeeType, setBaseFeeType] = useState<'free' | 'fixed'>('free');
  const [fixedFeeValue, setFixedFeeValue] = useState<number | null>(5);
  const [extraFees, setExtraFees] = useState<ExtraFeeRow[]>([]);
  const [noDeliveries, setNoDeliveries] = useState<NoDeliveryItem[]>([]);

  // 添加不配送地区的选择器状态
  const [selectingNoDelivery, setSelectingNoDelivery] = useState(false);
  const [tempNoDeliveryValue, setTempNoDeliveryValue] = useState<[string, string] | [string] | undefined>(undefined);

  // 初始化加载地区与详情
  useEffect(() => {
    async function init() {
      setLoading(true);
      await loadRegions();
      if (isEdit && id) {
        const res = await api.api.getFreightTemplateDetail({ id: Number(id) });
        if (res.code === 1 && res.data) {
          const detail = res.data;
          setName(detail.name || '');
          const feeNum = parseFloat(detail.base_fee || '0');
          if (feeNum > 0) {
            setBaseFeeType('fixed');
            setFixedFeeValue(feeNum);
          } else {
            setBaseFeeType('free');
            setFixedFeeValue(null);
          }

          if (detail.extra_fees && detail.extra_fees.length > 0) {
            setExtraFees(
              detail.extra_fees.map((ef: FreightExtraFeeRule, index: number) => ({
                key: `${ef.province_code}_${ef.city_code}_${index}`,
                provinceCode: ef.province_code || '',
                cityCode: ef.city_code || '',
                fee: ef.fee || '0.00',
              })),
            );
          }

          if (detail.no_delivery && detail.no_delivery.length > 0) {
            setNoDeliveries(
              detail.no_delivery.map((nd: FreightNoDeliveryRule, index: number) => ({
                key: `${nd.province_code}_${nd.city_code}_${index}`,
                provinceCode: nd.province_code || '',
                cityCode: nd.city_code || '',
              })),
            );
          }
        } else {
          message.error(res.message || '加载运费模板详情失败');
          navigate('/product/freight');
        }
      }
      setLoading(false);
    }
    init();
  }, [id, isEdit, navigate]);

  // 添加附加运费行
  const handleAddExtraFee = () => {
    setExtraFees((prev) => [
      ...prev,
      {
        key: `new_${Date.now()}_${Math.random()}`,
        provinceCode: '',
        cityCode: '',
        fee: '5.00',
      },
    ]);
  };

  // 修改附加运费行地区
  const handleExtraRegionChange = (key: string, val: [string, string] | [string] | undefined) => {
    setExtraFees((prev) =>
      prev.map((row) => {
        if (row.key !== key) return row;
        if (!val) {
          return { ...row, provinceCode: '', cityCode: '' };
        }
        const cCode = val.length > 1 && val[1] ? val[1] : '';
        return {
          ...row,
          provinceCode: val[0],
          cityCode: cCode,
        };
      }),
    );
  };

  // 修改附加运费金额
  const handleExtraFeeChange = (key: string, val: number | null) => {
    const feeStr = val !== null && val >= 0 ? val.toFixed(2) : '0.00';
    setExtraFees((prev) =>
      prev.map((row) => (row.key === key ? { ...row, fee: feeStr } : row)),
    );
  };

  // 删除附加运费行
  const handleRemoveExtraFee = (key: string) => {
    setExtraFees((prev) => prev.filter((row) => row.key !== key));
  };

  // 确认添加不配送地区
  const handleConfirmNoDelivery = () => {
    if (!tempNoDeliveryValue) {
      message.warning('请先选择不配送省份或城市');
      return;
    }
    const pCode = tempNoDeliveryValue[0];
    const cCode = tempNoDeliveryValue.length > 1 && tempNoDeliveryValue[1] ? tempNoDeliveryValue[1] : '';

    // 检查是否已存在
    const exists = noDeliveries.some(
      (item) => item.provinceCode === pCode && item.cityCode === cCode,
    );
    if (exists) {
      message.warning('该地区已在不配送列表中');
      return;
    }

    setNoDeliveries((prev) => [
      ...prev,
      {
        key: `nd_${Date.now()}_${Math.random()}`,
        provinceCode: pCode,
        cityCode: cCode,
      },
    ]);
    setTempNoDeliveryValue(undefined);
    setSelectingNoDelivery(false);
  };

  // 删除不配送地区
  const handleRemoveNoDelivery = (key: string) => {
    setNoDeliveries((prev) => prev.filter((item) => item.key !== key));
  };

  // 提交保存
  const handleSave = async () => {
    const trimmedName = name.trim();
    if (!trimmedName) {
      message.error('请输入模板名称');
      return;
    }
    if (trimmedName.length > 100) {
      message.error('模板名称不能超过100个字符');
      return;
    }

    let finalBaseFee = '0.00';
    if (baseFeeType === 'fixed') {
      if (fixedFeeValue === null || fixedFeeValue < 0) {
        message.error('请输入正确的固定运费金额');
        return;
      }
      finalBaseFee = fixedFeeValue.toFixed(2);
    }

    // 校验附加运费
    const extraFeeKeySet = new Set<string>();
    const payloadExtraFees: FreightExtraFeeRule[] = [];
    for (const row of extraFees) {
      if (!row.provinceCode) {
        message.error('附加运费地区存在未选择地区的配置行');
        return;
      }
      const feeNum = parseFloat(row.fee);
      if (isNaN(feeNum) || feeNum < 0) {
        message.error('附加运费必须是非负合法金额');
        return;
      }
      const key = `${row.provinceCode}#${row.cityCode}`;
      if (extraFeeKeySet.has(key)) {
        message.error(`附加运费地区存在重复配置: ${formatRegionLabel(row.provinceCode, row.cityCode)}`);
        return;
      }
      extraFeeKeySet.add(key);
      payloadExtraFees.push({
        province_code: row.provinceCode,
        city_code: row.cityCode,
        fee: feeNum.toFixed(2),
      });
    }

    // 校验不配送地区
    const noDelKeySet = new Set<string>();
    const payloadNoDelivery: FreightNoDeliveryRule[] = [];
    for (const item of noDeliveries) {
      const key = `${item.provinceCode}#${item.cityCode}`;
      if (noDelKeySet.has(key)) {
        message.error(`不配送地区存在重复配置: ${formatRegionLabel(item.provinceCode, item.cityCode)}`);
        return;
      }
      noDelKeySet.add(key);

      // 冲突校验：同一省市不能同时配置附加运费和不配送
      if (extraFeeKeySet.has(key)) {
        message.error(
          `同一省市不能同时配置附加运费与不配送: ${formatRegionLabel(item.provinceCode, item.cityCode)}`,
        );
        return;
      }

      payloadNoDelivery.push({
        province_code: item.provinceCode,
        city_code: item.cityCode,
      });
    }

    setSubmitting(true);
    if (isEdit && id) {
      const res = await api.api.updateFreightTemplate({
        id: Number(id),
        name: trimmedName,
        base_fee: finalBaseFee,
        extra_fees: payloadExtraFees,
        no_delivery: payloadNoDelivery,
      });
      setSubmitting(false);
      if (res.code === 1) {
        message.success('运费模板更新成功');
        navigate('/product/freight');
      } else {
        message.error(res.message || '更新运费模板失败');
      }
    } else {
      const res = await api.api.createFreightTemplate({
        name: trimmedName,
        base_fee: finalBaseFee,
        extra_fees: payloadExtraFees,
        no_delivery: payloadNoDelivery,
      });
      setSubmitting(false);
      if (res.code === 1) {
        message.success('运费模板创建成功');
        navigate('/product/freight');
      } else {
        message.error(res.message || '创建运费模板失败');
      }
    }
  };

  const extraFeeColumns = [
    {
      title: '配送地区',
      key: 'region',
      width: 320,
      render: (_: any, record: ExtraFeeRow) => {
        const val: [string, string] | [string] | undefined = record.provinceCode
          ? record.cityCode
            ? [record.provinceCode, record.cityCode]
            : [record.provinceCode]
          : undefined;
        return (
          <RegionCascader
            value={val}
            onChange={(v) => handleExtraRegionChange(record.key, v)}
            placeholder="选择省份或城市"
          />
        );
      },
    },
    {
      title: '附加运费',
      key: 'fee',
      width: 200,
      render: (_: any, record: ExtraFeeRow) => (
        <InputNumber
          prefix="¥"
          min={0}
          precision={2}
          value={parseFloat(record.fee || '0')}
          onChange={(v) => handleExtraFeeChange(record.key, v)}
          style={{ width: '100%' }}
        />
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 80,
      render: (_: any, record: ExtraFeeRow) => (
        <Button
          type="text"
          danger
          icon={<DeleteOutlined />}
          onClick={() => handleRemoveExtraFee(record.key)}
        />
      ),
    },
  ];

  return (
    <Spin spinning={loading}>
      <Card
        title={
          <Space>
            <Button
              type="text"
              icon={<ArrowLeftOutlined />}
              onClick={() => navigate('/product/freight')}
            />
            <Title level={4} style={{ margin: 0 }}>
              {isEdit ? '编辑运费模板' : '新建运费模板'}
            </Title>
          </Space>
        }
      >
        <div style={{ maxWidth: 880, margin: '0 auto' }}>
          {/* 模板名称 */}
          <div className="mb-6">
            <div className="text-base font-semibold mb-2">模板名称</div>
            <Input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="请输入模板名称，如：普通商品模板、大件快递等"
              maxLength={100}
              showCount
              style={{ maxWidth: 400 }}
            />
          </div>

          <Divider />

          {/* 默认运费 */}
          <div className="mb-6">
            <div className="text-base font-semibold mb-2">默认运费</div>
            <Radio.Group
              value={baseFeeType}
              onChange={(e) => setBaseFeeType(e.target.value)}
            >
              <Space direction="vertical">
                <Radio value="free">
                  <Text strong>包邮</Text>
                  <Text type="secondary" className="ml-2">
                    （默认全国包邮，可通过下方地区附加运费增加偏远地区运费）
                  </Text>
                </Radio>
                <Radio value="fixed">
                  <Space>
                    <Text strong>固定运费</Text>
                    <InputNumber
                      prefix="¥"
                      min={0.01}
                      precision={2}
                      value={fixedFeeValue}
                      disabled={baseFeeType !== 'fixed'}
                      onChange={(v) => setFixedFeeValue(v)}
                      style={{ width: 140 }}
                    />
                  </Space>
                </Radio>
              </Space>
            </Radio.Group>
          </div>

          <Divider />

          {/* 地区附加运费 */}
          <div className="mb-6">
            <div className="flex items-center justify-between mb-3">
              <div>
                <span className="text-base font-semibold">地区附加运费</span>
                <span className="text-gray-400 text-sm ml-2">
                  （在默认运费基础上额外增加的运费，城市规则优先于省份规则且不叠加）
                </span>
              </div>
              <Button
                type="dashed"
                icon={<PlusOutlined />}
                onClick={handleAddExtraFee}
              >
                添加地区
              </Button>
            </div>

            {extraFees.length === 0 ? (
              <div className="p-4 bg-gray-50 border border-dashed border-gray-200 rounded text-center text-gray-400">
                暂未配置地区附加运费，默认按上方基础运费计算
              </div>
            ) : (
              <Table
                rowKey="key"
                columns={extraFeeColumns}
                dataSource={extraFees}
                pagination={false}
                size="middle"
              />
            )}
          </div>

          <Divider />

          {/* 不配送地区 */}
          <div className="mb-8">
            <div className="flex orientation-row items-center justify-between mb-3">
              <div>
                <span className="text-base font-semibold">不配送地区</span>
                <span className="text-gray-400 text-sm ml-2">
                  （收货地址匹配到该地区时，禁止下单，城市不配送优先于省份）
                </span>
              </div>
              {!selectingNoDelivery && (
                <Button
                  type="dashed"
                  icon={<PlusOutlined />}
                  onClick={() => setSelectingNoDelivery(true)}
                >
                  添加不配送地区
                </Button>
              )}
            </div>

            {selectingNoDelivery && (
              <Card size="small" className="mb-4 bg-blue-50 border-blue-200">
                <div className="flex items-center gap-3">
                  <div style={{ width: 280 }}>
                    <RegionCascader
                      value={tempNoDeliveryValue}
                      onChange={(v) => setTempNoDeliveryValue(v)}
                      placeholder="选择不配送省份或城市"
                    />
                  </div>
                  <Button type="primary" size="small" onClick={handleConfirmNoDelivery}>
                    确认添加
                  </Button>
                  <Button
                    size="small"
                    onClick={() => {
                      setSelectingNoDelivery(false);
                      setTempNoDeliveryValue(undefined);
                    }}
                  >
                    取消
                  </Button>
                </div>
              </Card>
            )}

            {noDeliveries.length === 0 ? (
              <div className="p-4 bg-gray-50 border border-dashed border-gray-200 rounded text-center text-gray-400">
                暂无不配送地区限制，全国均可配送
              </div>
            ) : (
              <div className="flex flex-wrap gap-2 p-3 bg-gray-50 border border-gray-200 rounded">
                {noDeliveries.map((item) => (
                  <Tag
                    key={item.key}
                    closable
                    onClose={() => handleRemoveNoDelivery(item.key)}
                    color="error"
                    style={{ fontSize: 13, padding: '4px 10px' }}
                  >
                    {formatRegionLabel(item.provinceCode, item.cityCode)}
                  </Tag>
                ))}
              </div>
            )}
          </div>

          <Divider />

          {/* 底部保存与取消 */}
          <div className="flex justify-end gap-3 pb-8">
            <Button onClick={() => navigate('/product/freight')}>取消</Button>
            <Button
              type="primary"
              icon={<SaveOutlined />}
              loading={submitting}
              onClick={handleSave}
            >
              保存模板
            </Button>
          </div>
        </div>
      </Card>
    </Spin>
  );
}
