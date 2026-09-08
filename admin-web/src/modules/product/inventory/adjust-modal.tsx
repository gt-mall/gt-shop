import React, { useCallback, useEffect, useRef, useState } from 'react';
import {
  Alert,
  Button,
  Descriptions,
  Form,
  Input,
  InputNumber,
  message,
  Modal,
  Space,
  Spin,
  Tag,
  Typography,
} from 'antd';
import { api } from '@/framework/http';
import type {
  ProductInventoryAdjustRequest,
  ProductInventoryRow,
  ProductInventoryWarningRequest,
} from '@/framework/http/generated_api';
import { useUser } from '@/framework/store';

export type AdjustMode = 'manual_in' | 'manual_out' | 'stocktake' | 'warning';

interface AdjustModalProps {
  open: boolean;
  mode: AdjustMode;
  skuId: number | null;
  onClose: () => void;
  onSuccess: () => void;
}

interface PendingMutation {
  userId: number;
  skuId: number;
  requestId: string;
  expectedVersion: number;
  changeType: 'manual_in' | 'manual_out' | 'stocktake' | 'warning_set';
  quantity: number;
  remark: string;
  mode: AdjustMode;
  timestamp: number;
}

const getStorageKey = (userId?: number) =>
  userId ? `mall_inventory_pending_mutation_${userId}` : '';

const generateUUID = (): string => {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID().toLowerCase();
  }
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0;
    const v = c === 'x' ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
};

const modeTitles: Record<AdjustMode, string> = {
  manual_in: '人工入库',
  manual_out: '人工出库',
  stocktake: '盘点校正',
  warning: '设置预警阈值',
};

export const AdjustModal: React.FC<AdjustModalProps> = ({
  open,
  mode,
  skuId,
  onClose,
  onSuccess,
}) => {
  const user = useUser();
  const [form] = Form.useForm();

  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [detail, setDetail] = useState<ProductInventoryRow | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);

  // Pending unconfirmed request state
  const [pendingRequest, setPendingRequest] = useState<PendingMutation | null>(null);
  const submittingRef = useRef(false);

  // Form values watcher for real-time delta preview
  const watchedQuantity = Form.useWatch('quantity', form);

  // Load detail or recover pending request
  const loadData = useCallback(async () => {
    if (!skuId || !open) return;
    setLoading(true);
    setLoadError(null);

    // 1. Check if there is an unconfirmed pending mutation for current user and SKU
    const storageKey = getStorageKey(user?.id);
    let recoveredPending: PendingMutation | null = null;
    if (storageKey) {
      const stored = sessionStorage.getItem(storageKey);
      if (stored) {
        const parsed = JSON.parse(stored) as PendingMutation;
        if (parsed.userId === user?.id && parsed.skuId === skuId) {
          recoveredPending = parsed;
        }
      }
    }

    // 2. Fetch latest SKU inventory
    const res = await api.api.getProductInventory({ sku_id: skuId });
    setLoading(false);

    if (res.code !== 1 || !res.data) {
      setLoadError(res.message || '获取SKU库存详情失败');
      return;
    }

    const inventory = res.data as ProductInventoryRow;
    if (!inventory.initialized) {
      setLoadError('该SKU库存尚未初始化，无法进行调整');
      return;
    }

    setDetail(inventory);

    if (recoveredPending) {
      setPendingRequest(recoveredPending);
      form.setFieldsValue({
        quantity: recoveredPending.quantity,
        remark: recoveredPending.remark,
      });
    } else {
      setPendingRequest(null);
      form.resetFields();
    }
  }, [skuId, open, user?.id, form]);

  useEffect(() => {
    if (open) {
      void loadData();
    } else {
      setDetail(null);
      setLoadError(null);
      setPendingRequest(null);
      form.resetFields();
      submittingRef.current = false;
    }
  }, [open, loadData, form]);

  // Handle Form submit
  const handleSubmit = async () => {
    if (!detail || !skuId || !user?.id) return;
    if (submittingRef.current) return;

    const values = await form.validateFields();
    const qty = Number(values.quantity);
    const remark = String(values.remark || '').trim();

    if (!remark) {
      message.error('操作原因不能为空');
      return;
    }

    // Check if non-change before submitting
    if (!pendingRequest) {
      if (mode === 'stocktake' && qty === detail.stock) {
        message.warning('盘点目标总库存与当前总库存一致，无需保存');
        return;
      }
      if (mode === 'warning' && qty === detail.warning_stock) {
        message.warning('预警阈值与当前预警阈值一致，无需保存');
        return;
      }
    }

    // Prepare payload
    const storageKey = getStorageKey(user.id);
    const requestId = pendingRequest ? pendingRequest.requestId : generateUUID();
    const expectedVersion = pendingRequest ? pendingRequest.expectedVersion : (detail.version ?? 0);
    const changeType =
      mode === 'manual_in'
        ? 'manual_in'
        : mode === 'manual_out'
          ? 'manual_out'
          : mode === 'stocktake'
            ? 'stocktake'
            : 'warning_set';

    const payload: PendingMutation = {
      userId: user.id,
      skuId,
      requestId,
      expectedVersion,
      changeType,
      quantity: qty,
      remark,
      mode,
      timestamp: Date.now(),
    };

    // Save pending mutation in sessionStorage to guard against timeout / refresh
    if (storageKey) {
      sessionStorage.setItem(storageKey, JSON.stringify(payload));
    }

    submittingRef.current = true;
    setSubmitting(true);

    let res;
    if (mode === 'warning') {
      const warningReq: ProductInventoryWarningRequest = {
        sku_id: skuId,
        request_id: requestId,
        expected_version: expectedVersion,
        quantity: qty,
        remark,
      };
      res = await api.api.setProductInventoryWarning(warningReq);
    } else {
      const adjustReq: ProductInventoryAdjustRequest = {
        sku_id: skuId,
        request_id: requestId,
        expected_version: expectedVersion,
        change_type: changeType,
        quantity: qty,
        remark,
      };
      res = await api.api.adjustProductInventory(adjustReq);
    }

    submittingRef.current = false;
    setSubmitting(false);

    // 1. Success
    if (res.code === 1) {
      if (storageKey) {
        sessionStorage.removeItem(storageKey);
      }
      setPendingRequest(null);
      if (res.data?.replayed) {
        message.info('该操作此前已处理完成（幂等回放）');
      } else {
        message.success('操作成功');
      }
      onSuccess();
      onClose();
      return;
    }

    // 2. Failure: handle error codes
    const errorCode = res.data?.error_code;
    const errorMsg = res.message || '操作失败';

    if (errorCode === 'VERSION_CONFLICT') {
      if (storageKey) {
        sessionStorage.removeItem(storageKey);
      }
      setPendingRequest(null);
      message.error(`${errorMsg}，已自动刷新最新库存，请核对后重新提交`);
      // Reload latest detail
      void loadData();
      return;
    }

    if (
      errorCode === 'INSUFFICIENT_STOCK' ||
      errorCode === 'NO_CHANGE' ||
      errorCode === 'INVALID_ARGUMENT' ||
      errorCode === 'NOT_FOUND' ||
      errorCode === 'NOT_INITIALIZED' ||
      errorCode === 'IDEMPOTENCY_CONFLICT'
    ) {
      if (storageKey) {
        sessionStorage.removeItem(storageKey);
      }
      setPendingRequest(null);
      message.error(errorMsg);
      return;
    }

    // Network timeout, RESULT_UNKNOWN, RETRYABLE or unexpected error:
    // Keep pendingRequest active in sessionStorage and modal
    setPendingRequest(payload);
    message.warning(
      `提交结果尚未确定（${errorMsg}）。操作已保留，请点击【确认上次操作结果 / 重试】再次确认，请勿更改参数。`,
    );
  };

  const handleCancel = () => {
    if (submitting) return;
    onClose();
  };

  // Preview calculations
  const currentStock = detail?.stock ?? 0;
  const currentLocked = detail?.locked_stock ?? 0;
  const currentAvailable = detail?.available_stock ?? 0;
  const currentWarning = detail?.warning_stock ?? 0;

  const numQty = typeof watchedQuantity === 'number' ? watchedQuantity : undefined;

  let previewText = '';
  if (numQty !== undefined) {
    if (mode === 'manual_in') {
      previewText = `入库后预计总库存: ${currentStock + numQty}，预计可用量: ${currentAvailable + numQty}`;
    } else if (mode === 'manual_out') {
      previewText = `出库后预计总库存: ${currentStock - numQty}，预计可用量: ${currentAvailable - numQty}`;
    } else if (mode === 'stocktake') {
      const delta = numQty - currentStock;
      const deltaStr = delta > 0 ? `+${delta}` : `${delta}`;
      previewText = `盘点调整差额: ${deltaStr}，盘点后预计可用量: ${numQty - currentLocked}`;
    } else if (mode === 'warning') {
      previewText = numQty === 0 ? '设为 0 将关闭低库存预警' : `当可用库存 ≤ ${numQty} 时将触发预警`;
    }
  }

  const renderSpecs = () => {
    if (!detail?.specs || detail.specs.length === 0) {
      return <Tag color="default">默认规格</Tag>;
    }
    return detail.specs.map((s) => (
      <Tag key={`${s.spec_id}-${s.value_id}`} color="blue">
        {s.spec_name}: {s.value_name}
      </Tag>
    ));
  };

  return (
    <Modal
      title={
        <Space>
          <span>{modeTitles[mode]}</span>
          {pendingRequest && <Tag color="warning">待确认操作 / 重试</Tag>}
        </Space>
      }
      open={open}
      onCancel={handleCancel}
      footer={[
        <Button key="cancel" disabled={submitting} onClick={handleCancel}>
          取消
        </Button>,
        <Button
          key="submit"
          type="primary"
          loading={submitting}
          disabled={loading || Boolean(loadError)}
          onClick={handleSubmit}
        >
          {pendingRequest ? '确认上次操作结果 / 重试' : '确认提交'}
        </Button>,
      ]}
      destroyOnClose
      width={640}
    >
      <Spin spinning={loading}>
        {loadError && (
          <Alert
            type="error"
            message="无法进行库存操作"
            description={loadError}
            showIcon
            style={{ marginBottom: 16 }}
          />
        )}

        {pendingRequest && (
          <Alert
            type="warning"
            showIcon
            message="检测到未完成确认的操作"
            description="可能由于网络超时或连接中断，上次提交的结果尚未在页面确认。参数已自动锁定，请点击重试以核对服务端结果，避免重复入库或扣减。"
            style={{ marginBottom: 16 }}
          />
        )}

        {detail && (
          <Space direction="vertical" size="middle" style={{ width: '100%' }}>
            {/* Readonly SKU & Inventory Info */}
            <Descriptions
              bordered
              size="small"
              column={{ xs: 1, sm: 2 }}
              style={{ background: '#fafafa' }}
            >
              <Descriptions.Item label="商品名称" span={2}>
                <Typography.Text strong>{detail.product_name}</Typography.Text>
              </Descriptions.Item>
              <Descriptions.Item label="SKU 规格">{renderSpecs()}</Descriptions.Item>
              <Descriptions.Item label="单位 / 编码">
                {detail.unit || '件'} / {detail.sku_code || '—'}
              </Descriptions.Item>
              <Descriptions.Item label="当前总库存">
                <Typography.Text strong>{currentStock}</Typography.Text> 件
              </Descriptions.Item>
              <Descriptions.Item label="当前锁定库存">
                <Typography.Text type={currentLocked > 0 ? 'warning' : 'secondary'}>
                  {currentLocked}
                </Typography.Text>{' '}
                件
              </Descriptions.Item>
              <Descriptions.Item label="当前可用库存">
                <Typography.Text
                  strong
                  type={currentAvailable === 0 ? 'danger' : 'success'}
                  style={{ fontSize: 15 }}
                >
                  {currentAvailable}
                </Typography.Text>{' '}
                件
              </Descriptions.Item>
              <Descriptions.Item label="当前预警阈值">
                {currentWarning === 0 ? (
                  <Tag color="default">未开启</Tag>
                ) : (
                  <Tag color="cyan">{currentWarning} 件</Tag>
                )}
              </Descriptions.Item>
            </Descriptions>

            {/* Form */}
            <Form form={form} layout="vertical" disabled={submitting || Boolean(pendingRequest)}>
              {mode === 'manual_in' && (
                <Form.Item
                  name="quantity"
                  label="本次入库数量"
                  extra={previewText}
                  rules={[
                    { required: true, message: '请输入入库数量' },
                    {
                      type: 'integer',
                      min: 1,
                      max: 2147483647,
                      message: '入库数量必须为 1 到 2,147,483,647 之间的正整数',
                    },
                  ]}
                >
                  <InputNumber
                    min={1}
                    max={2147483647}
                    precision={0}
                    style={{ width: '100%' }}
                    placeholder="请输入正整数入库件数"
                  />
                </Form.Item>
              )}

              {mode === 'manual_out' && (
                <Form.Item
                  name="quantity"
                  label="本次出库数量"
                  extra={previewText}
                  rules={[
                    { required: true, message: '请输入出库数量' },
                    {
                      type: 'integer',
                      min: 1,
                      max: Math.max(1, currentAvailable),
                      message: `出库数量必须为 1 到 ${currentAvailable} 之间的正整数（不能超过可用库存）`,
                    },
                  ]}
                >
                  <InputNumber
                    min={1}
                    max={currentAvailable}
                    precision={0}
                    style={{ width: '100%' }}
                    placeholder={`请输入出库件数，最大 ${currentAvailable}`}
                  />
                </Form.Item>
              )}

              {mode === 'stocktake' && (
                <Form.Item
                  name="quantity"
                  label="盘点后总库存"
                  extra={previewText}
                  rules={[
                    { required: true, message: '请输入盘点后总库存' },
                    {
                      type: 'integer',
                      min: currentLocked,
                      max: 2147483647,
                      message: `盘点后总库存必须大于或等于锁定库存 (${currentLocked})`,
                    },
                  ]}
                >
                  <InputNumber
                    min={currentLocked}
                    max={2147483647}
                    precision={0}
                    style={{ width: '100%' }}
                    placeholder="请输入盘点后的总实物库存件数（非差额）"
                  />
                </Form.Item>
              )}

              {mode === 'warning' && (
                <Form.Item
                  name="quantity"
                  label="预警阈值"
                  extra={previewText}
                  rules={[
                    { required: true, message: '请输入预警阈值' },
                    {
                      type: 'integer',
                      min: 0,
                      max: 2147483647,
                      message: '预警阈值必须为 0 到 2,147,483,647 之间的非负整数',
                    },
                  ]}
                >
                  <InputNumber
                    min={0}
                    max={2147483647}
                    precision={0}
                    style={{ width: '100%' }}
                    placeholder="输入 0 表示关闭低库存预警"
                  />
                </Form.Item>
              )}

              <Form.Item
                name="remark"
                label="操作原因"
                rules={[
                  { required: true, message: '请输入操作原因' },
                  {
                    validator: (_, val) => {
                      const trimmed = String(val || '').trim();
                      if (trimmed.length < 1 || trimmed.length > 500) {
                        return Promise.reject(new Error('操作原因长度必须在 1 到 500 个字符之间'));
                      }
                      return Promise.resolve();
                    },
                  },
                ]}
              >
                <Input.TextArea
                  rows={3}
                  maxLength={500}
                  showCount
                  placeholder="请输入操作原因说明，如：采购到货入库、调拨出库、盘点损耗校正等"
                />
              </Form.Item>
            </Form>
          </Space>
        )}
      </Spin>
    </Modal>
  );
};
