import { useCallback, useEffect, useState } from "react";
import {
  Button,
  Card,
  Checkbox,
  Form,
  Input,
  InputNumber,
  message,
  Modal,
  Popconfirm,
  Select,
  Space,
  Spin,
  Table,
  Tag,
  Typography,
} from "antd";
import type { ColumnsType } from "antd/es/table";

import { api, type ApiResponse } from "@/framework/http";
import type { PermissionDefinition, RoleRow } from "@/framework/http/generated_api";
import { useAppStore } from "@/framework/store";
import { formatTime } from "@/shared/utils/format";

type RoleForm = {
  name: string;
  status: number;
  sort: number;
  remark?: string;
};

type Query = {
  current: number;
  pageSize: number;
  keyword: string;
  status?: number;
};

const statusOptions = [
  { label: "启用", value: 1 },
  { label: "禁用", value: 0 },
];

async function requestApi<T>(task: Promise<ApiResponse<T>>, fallback: string) {
  const result = await task.catch(() => undefined);
  if (result?.code === 1) return result;
  message.error(result?.message || fallback);
}

export default function RolePage() {
  const hasPermission = useAppStore((state) => state.hasPermission);
  const canCreate = hasPermission("system:role:create");
  const canUpdate = hasPermission("system:role:update");
  const canDelete = hasPermission("system:role:delete");
  const canGrant = hasPermission("system:role:permission");
  const [form] = Form.useForm<RoleForm>();
  const [rows, setRows] = useState<RoleRow[]>([]);
  const [total, setTotal] = useState(0);
  const [query, setQuery] = useState<Query>({
    current: 1,
    pageSize: 20,
    keyword: "",
  });
  const [loading, setLoading] = useState(false);
  const [editingId, setEditingId] = useState<number>();
  const [editingCode, setEditingCode] = useState<string>();
  const [deletingId, setDeletingId] = useState<number>();
  const [modalOpen, setModalOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [permissionRole, setPermissionRole] = useState<RoleRow>();
  const [permissionOptions, setPermissionOptions] = useState<PermissionDefinition[]>([]);
  const [checkedPermissions, setCheckedPermissions] = useState<string[]>([]);
  const [permissionLoading, setPermissionLoading] = useState(false);
  const [permissionSaving, setPermissionSaving] = useState(false);

  const loadRoles = useCallback(async () => {
    setLoading(true);
    const result = await requestApi<RoleRow[]>(
      api.api.listSystemRoles({
        page: query.current,
        page_size: query.pageSize,
        keyword: query.keyword || undefined,
        status: query.status,
      }),
      "获取角色列表失败",
    );
    setRows(result?.data ?? []);
    setTotal(result?.total ?? 0);
    setLoading(false);
  }, [query]);

  useEffect(() => {
    void loadRoles();
  }, [loadRoles]);

  const closeModal = () => {
    setModalOpen(false);
    setEditingId(undefined);
    setEditingCode(undefined);
  };

  const openCreate = () => {
    setEditingId(undefined);
    setEditingCode(undefined);
    form.resetFields();
    setModalOpen(true);
  };

  const openEdit = (role: RoleRow) => {
    if (!role.id) return;
    setEditingId(role.id);
    setEditingCode(role.code);
    form.setFieldsValue({
      name: role.name,
      status: role.status ?? 1,
      sort: role.sort ?? 0,
      remark: role.remark,
    });
    setModalOpen(true);
  };

  const saveRole = async (values: RoleForm) => {
    const isEdit = editingId !== undefined;
    const base = {
      name: values.name.trim(),
      status: values.status,
      sort: values.sort ?? 0,
      remark: values.remark?.trim() ?? "",
    };
    const task = isEdit
      ? api.api.updateSystemRole({ id: editingId, ...base })
      : api.api.createSystemRole(base);

    setSaving(true);
    const result = await requestApi<RoleRow>(
      task,
      isEdit ? "更新角色失败" : "新增角色失败",
    );
    setSaving(false);
    if (!result) return;

    message.success(isEdit ? "角色信息已更新" : "角色已新增");
    closeModal();
    await loadRoles();
  };

  const deleteRole = async (id?: number) => {
    if (!id) return;
    setDeletingId(id);
    const result = await requestApi(
      api.api.deleteSystemRole({ id }),
      "删除角色失败",
    );
    setDeletingId(undefined);
    if (!result) return;

    message.success("角色已删除");
    if (rows.length === 1 && query.current > 1) {
      setQuery((value) => ({ ...value, current: value.current - 1 }));
      return;
    }
    await loadRoles();
  };

  const openPermissions = async (role: RoleRow) => {
    if (!role.id) return;
    setPermissionRole(role);
    setPermissionLoading(true);
    const [all, assigned] = await Promise.all([
      requestApi<PermissionDefinition[]>(api.api.listAllPermissions(), "获取权限定义失败"),
      requestApi<string[]>(api.api.getSystemRolePermissions({ id: role.id }), "获取角色权限失败"),
    ]);
    setPermissionOptions(all?.data ?? []);
    setCheckedPermissions(assigned?.data ?? []);
    setPermissionLoading(false);
  };

  const savePermissions = async () => {
    if (!permissionRole?.id || permissionRole.code === "super_admin") return;
    setPermissionSaving(true);
    const result = await requestApi(
      api.api.saveSystemRolePermissions({
        id: permissionRole.id,
        permissions: checkedPermissions,
      }),
      "保存角色权限失败",
    );
    setPermissionSaving(false);
    if (!result) return;
    message.success("角色权限已保存");
    setPermissionRole(undefined);
  };

  const columns: ColumnsType<RoleRow> = [
    { title: "角色名称", dataIndex: "name", ellipsis: true },
    { title: "角色编码", dataIndex: "code", ellipsis: true },
    {
      title: "状态",
      dataIndex: "status",
      width: 90,
      render: (value) =>
        value === 1 ? <Tag color="success">启用</Tag> : <Tag>禁用</Tag>,
    },
    { title: "排序", dataIndex: "sort", width: 80 },
    {
      title: "备注",
      dataIndex: "remark",
      ellipsis: true,
      render: (value) => value || "-",
    },
    {
      title: "创建时间",
      dataIndex: "created_at",
      width: 170,
      render: formatTime,
    },
    {
      title: "操作",
      width: 190,
      render: (_, record) => (
        <Space size={4}>
          {canGrant ? (
            <Button type="link" size="small" onClick={() => openPermissions(record)}>
              {record.code === "super_admin" ? "查看权限" : "授权"}
            </Button>
          ) : null}
          {canUpdate ? (
            <Button type="link" size="small" onClick={() => openEdit(record)}>
              编辑
            </Button>
          ) : null}
          {canDelete && record.code !== "super_admin" ? <Popconfirm
            title="删除这个角色？"
            description="删除后无法恢复"
            okText="删除"
            cancelText="取消"
            okButtonProps={{ danger: true, loading: deletingId === record.id }}
            onConfirm={() => deleteRole(record.id)}
          >
            <Button type="link" danger size="small">
              删除
            </Button>
          </Popconfirm> : null}
        </Space>
      ),
    },
  ];

  return (
    <Card
      title="权限管理"
      extra={
        <Space wrap>
          <Input.Search
            allowClear
            placeholder="搜索角色名称或编码"
            style={{ width: 240 }}
            onSearch={(value) =>
              setQuery((current) => ({
                ...current,
                current: 1,
                keyword: value.trim(),
              }))
            }
          />
          <Select
            allowClear
            placeholder="状态"
            options={statusOptions}
            style={{ width: 100 }}
            onChange={(status) =>
              setQuery((current) => ({ ...current, current: 1, status }))
            }
          />
          {canCreate ? (
            <Button type="primary" onClick={openCreate}>
              新增角色
            </Button>
          ) : null}
        </Space>
      }
    >
      <Table<RoleRow>
        rowKey="id"
        size="small"
        loading={loading}
        columns={columns}
        dataSource={rows}
        scroll={{ x: 980 }}
        pagination={{
          current: query.current,
          pageSize: query.pageSize,
          total,
          showSizeChanger: true,
          showTotal: (count) => `共 ${count} 条`,
          onChange: (current, pageSize) =>
            setQuery((value) => ({ ...value, current, pageSize })),
        }}
      />

      <Modal
        title={editingId === undefined ? "新增角色" : "编辑角色"}
        open={modalOpen}
        width={460}
        destroyOnHidden
        okText="保存"
        cancelText="取消"
        confirmLoading={saving}
        onCancel={closeModal}
        onOk={() => form.submit()}
      >
        <Form<RoleForm>
          form={form}
          layout="vertical"
          requiredMark={false}
          preserve={false}
          initialValues={{ status: 1, sort: 0 }}
          onFinish={saveRole}
        >
          <Form.Item
            name="name"
            label="角色名称"
            rules={[{ required: true, whitespace: true, message: "请输入角色名称" }]}
          >
            <Input maxLength={64} placeholder="请输入角色名称" />
          </Form.Item>
          {editingCode ? (
            <Form.Item label="角色编码" extra="角色编码由系统生成，创建后不可修改">
              <Input value={editingCode} disabled />
            </Form.Item>
          ) : null}
          <Space align="start" style={{ display: "flex" }}>
            <Form.Item
              name="status"
              label="状态"
              rules={[{ required: true, message: "请选择状态" }]}
            >
              <Select options={statusOptions} style={{ width: 180 }} />
            </Form.Item>
            <Form.Item
              name="sort"
              label="排序"
              rules={[{ required: true, message: "请输入排序" }]}
            >
              <InputNumber min={0} precision={0} style={{ width: 180 }} />
            </Form.Item>
          </Space>
          <Form.Item name="remark" label="备注">
            <Input.TextArea maxLength={255} rows={3} placeholder="请输入备注" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={`${permissionRole?.name ?? "角色"}权限`}
        open={Boolean(permissionRole)}
        width={560}
        okText="保存"
        cancelText="取消"
        confirmLoading={permissionSaving}
        okButtonProps={{ disabled: permissionRole?.code === "super_admin" || permissionLoading }}
        onCancel={() => setPermissionRole(undefined)}
        onOk={savePermissions}
      >
        {permissionRole?.code === "super_admin" ? (
          <Typography.Text type="secondary">super_admin 自动拥有全部权限，无需保存策略。</Typography.Text>
        ) : null}
        <Spin spinning={permissionLoading}>
          <Checkbox.Group
            value={checkedPermissions}
            onChange={(values) => setCheckedPermissions(values as string[])}
            style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 12, marginTop: 16 }}
          >
            {permissionOptions.map((item) => (
              <Checkbox key={item.code} value={item.code}>
                {item.module} / {item.name}
              </Checkbox>
            ))}
          </Checkbox.Group>
        </Spin>
      </Modal>
    </Card>
  );
}
