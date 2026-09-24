import { useEffect, useState } from 'react';
import {
  Button,
  Form,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Space,
  Switch,
  Table,
  Tag,
  Typography,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  api,
  type CreateReturnAddressRequest,
  type ReturnAddressItem,
  type UpdateReturnAddressRequest,
} from '@/framework/http';
import { usePermission } from '@/framework/store';

export type ReturnAddressModalProps = {
  open: boolean;
  onClose: () => void;
  onDefaultChange?: (defaultAddress: ReturnAddressItem | null) => void;
};

export default function ReturnAddressModal({
  open,
  onClose,
  onDefaultChange,
}: ReturnAddressModalProps) {
  const [loading, setLoading] = useState(false);
  const [addresses, setAddresses] = useState<ReturnAddressItem[]>([]);
  const [editModalVisible, setEditModalVisible] = useState(false);
  const [editingItem, setEditingItem] = useState<ReturnAddressItem | null>(null);
  const [form] = Form.useForm();
  const [submitting, setSubmitting] = useState(false);
  const canUpdate = usePermission()('system:setting:update');

  const loadAddresses = async () => {
    setLoading(true);
    const res = await api.api.listOrderReturnAddresses({});
    setLoading(false);
    if (res.code !== 1 || !res.data) {
      message.error(res.message || '获取退货地址列表失败');
      return;
    }
    setAddresses(res.data);
    const def = res.data.find((a) => a.is_default === 1) || null;
    onDefaultChange?.(def);
  };

  useEffect(() => {
    if (open) {
      void loadAddresses();
    }
  }, [open]);

  const handleOpenCreate = () => {
    setEditingItem(null);
    form.resetFields();
    form.setFieldsValue({
      status: 1,
      is_default: addresses.length === 0 ? 1 : 0,
      sort: 10,
    });
    setEditModalVisible(true);
  };

  const handleOpenEdit = (item: ReturnAddressItem) => {
    setEditingItem(item);
    form.setFieldsValue({
      name: item.name,
      receiver: item.receiver,
      phone: item.phone,
      address: item.address,
      province_code: item.province_code,
      city_code: item.city_code,
      district_code: item.district_code,
      is_default: item.is_default,
      status: item.status,
      sort: item.sort,
    });
    setEditModalVisible(true);
  };

  const handleSave = async () => {
    const values = await form.validateFields();
    setSubmitting(true);

    if (editingItem && editingItem.id) {
      const payload: UpdateReturnAddressRequest = {
        id: editingItem.id,
        name: values.name?.trim(),
        receiver: values.receiver?.trim(),
        phone: values.phone?.trim(),
        address: values.address?.trim(),
        province_code: values.province_code || '',
        city_code: values.city_code || '',
        district_code: values.district_code || '',
        is_default: values.is_default ? 1 : 0,
        status: values.status ? 1 : 0,
        sort: values.sort ?? 10,
      };
      const res = await api.api.updateOrderReturnAddress(payload);
      setSubmitting(false);
      if (res.code !== 1) {
        message.error(res.message || '更新退货地址失败');
        return;
      }
      message.success('退货地址已更新');
    } else {
      const payload: CreateReturnAddressRequest = {
        name: values.name?.trim(),
        receiver: values.receiver?.trim(),
        phone: values.phone?.trim(),
        address: values.address?.trim(),
        province_code: values.province_code || '',
        city_code: values.city_code || '',
        district_code: values.district_code || '',
        is_default: values.is_default ? 1 : 0,
        status: values.status ? 1 : 0,
        sort: values.sort ?? 10,
      };
      const res = await api.api.createOrderReturnAddress(payload);
      setSubmitting(false);
      if (res.code !== 1) {
        message.error(res.message || '新增退货地址失败');
        return;
      }
      message.success('退货地址已添加');
    }

    setEditModalVisible(false);
    void loadAddresses();
  };

  const handleSetDefault = async (item: ReturnAddressItem) => {
    if (!item.id) return;
    const res = await api.api.setDefaultOrderReturnAddress({ id: item.id });
    if (res.code !== 1) {
      message.error(res.message || '设为默认地址失败');
      return;
    }
    message.success('已设为默认退货地址');
    void loadAddresses();
  };

  const handleDelete = async (id?: number) => {
    if (!id) return;
    const res = await api.api.deleteOrderReturnAddress({ id });
    if (res.code !== 1) {
      message.error(res.message || '删除退货地址失败');
      return;
    }
    message.success('退货地址已删除');
    void loadAddresses();
  };

  const columns: ColumnsType<ReturnAddressItem> = [
    {
      title: '仓库/地址名称',
      dataIndex: 'name',
      key: 'name',
      width: 140,
      render: (text, r) => (
        <Space orientation="vertical" size={2}>
          <Typography.Text strong>{text}</Typography.Text>
          {r.is_default === 1 && <Tag color="blue">默认地址</Tag>}
        </Space>
      ),
    },
    {
      title: '联系人',
      dataIndex: 'receiver',
      key: 'receiver',
      width: 110,
    },
    {
      title: '联系电话',
      dataIndex: 'phone',
      key: 'phone',
      width: 130,
    },
    {
      title: '详细退货地址',
      dataIndex: 'address',
      key: 'address',
      ellipsis: true,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 90,
      render: (status: number) =>
        status === 1 ? <Tag color="success">启用</Tag> : <Tag color="default">禁用</Tag>,
    },
    {
      title: '排序',
      dataIndex: 'sort',
      key: 'sort',
      width: 70,
    },
    {
      title: '操作',
      key: 'action',
      width: 180,
      render: (_, r) => (
        <Space size="middle">
          {r.is_default !== 1 && (
            <Button
              type="link"
              size="small"
              disabled={!canUpdate || r.status !== 1}
              onClick={() => void handleSetDefault(r)}
            >
              设为默认
            </Button>
          )}
          <Button
            type="link"
            size="small"
            disabled={!canUpdate}
            onClick={() => handleOpenEdit(r)}
          >
            编辑
          </Button>
          <Popconfirm
            title="确定删除该退货地址？"
            okText="确定"
            cancelText="取消"
            disabled={!canUpdate}
            onConfirm={() => void handleDelete(r.id)}
          >
            <Button type="link" size="small" danger disabled={!canUpdate}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <>
      <Modal
        title="退货地址库管理"
        open={open}
        onCancel={onClose}
        footer={[
          <Button key="close" onClick={onClose}>
            关闭
          </Button>,
        ]}
        width={880}
        destroyOnClose
      >
        <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Typography.Text type="secondary">
            维护商家售后退货地址。在买家申请退货退款审核通过时，系统将使用该地址引导买家邮寄退回。
          </Typography.Text>
          <Button type="primary" onClick={handleOpenCreate} disabled={!canUpdate}>
            新增退货地址
          </Button>
        </div>

        <Table
          dataSource={addresses}
          columns={columns}
          rowKey="id"
          loading={loading}
          pagination={false}
          size="middle"
          scroll={{ y: 360 }}
        />
      </Modal>

      <Modal
        title={editingItem ? '编辑退货地址' : '新增退货地址'}
        open={editModalVisible}
        onCancel={() => setEditModalVisible(false)}
        onOk={() => void handleSave()}
        confirmLoading={submitting}
        okText="保存"
        cancelText="取消"
        width={560}
        destroyOnClose
      >
        <Form
          form={form}
          layout="vertical"
          style={{ marginTop: 12 }}
        >
          <Form.Item
            name="name"
            label="仓库/地址名称"
            rules={[{ required: true, whitespace: true, max: 64, message: '请输入地址名称（如：华东总仓）' }]}
          >
            <Input placeholder="例：华东默认退货仓" maxLength={64} />
          </Form.Item>

          <Space style={{ display: 'flex' }} align="start">
            <Form.Item
              name="receiver"
              label="收件人姓名"
              style={{ width: 240 }}
              rules={[{ required: true, whitespace: true, max: 32, message: '请输入收件人姓名' }]}
            >
              <Input placeholder="例：张三" maxLength={32} />
            </Form.Item>

            <Form.Item
              name="phone"
              label="联系电话"
              style={{ width: 260 }}
              rules={[{ required: true, whitespace: true, max: 32, message: '请输入联系电话' }]}
            >
              <Input placeholder="例：13800000000" maxLength={32} />
            </Form.Item>
          </Space>

          <Form.Item
            name="address"
            label="详细退货地址"
            rules={[{ required: true, whitespace: true, max: 255, message: '请输入详细地址' }]}
          >
            <Input.TextArea rows={3} placeholder="例：上海市黄浦区人民大道100号A座5楼退货部" maxLength={255} showCount />
          </Form.Item>

          <Space size={32}>
            <Form.Item name="is_default" label="设为默认" valuePropName="checked">
              <Switch />
            </Form.Item>
            <Form.Item name="status" label="启用状态" valuePropName="checked">
              <Switch defaultChecked />
            </Form.Item>
            <Form.Item name="sort" label="排序权重">
              <InputNumber min={0} max={9999} />
            </Form.Item>
          </Space>
        </Form>
      </Modal>
    </>
  );
}
