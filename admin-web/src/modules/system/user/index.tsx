import { useCallback, useEffect, useState } from "react";
import {
  Button,
  Checkbox,
  Form,
  Input,
  message,
  Modal,
  Pagination,
  Popconfirm,
  Select,
  Space,
  Spin,
  Table,
  Tag,
  Tooltip,
  theme,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import {
  PlusOutlined,
  ReloadOutlined,
  SearchOutlined,
  UserOutlined,
} from "@ant-design/icons";

import { api, type ApiResponse } from "@/framework/http";
import type { CurrentRole, RoleRow, UserRow } from "@/framework/http/generated_api";
import { usePermission } from "@/framework/store";
import { ContentCard, PageHeader, adminListVars } from "@/shared/admin-ui";
import { formatTime } from "@/shared/utils/format";

type UserForm = {
  username: string;
  nickname?: string;
  password?: string;
  status: number;
};

type Query = {
  current: number;
  pageSize: number;
  username: string;
};

const statusOptions = [
  { label: "启用", value: 1 },
  { label: "禁用", value: 0 },
];

async function requestApi<T>(task: Promise<ApiResponse<T>>, fallback: string) {
  const result = await task;
  if (result.code === 1) return result;
  message.error(result.message || fallback);
}

export default function UserPage() {
  const { token } = theme.useToken();
  const hasPermission = usePermission();
  const canCreate = hasPermission("system:user:create");
  const canUpdate = hasPermission("system:user:update");
  const canDelete = hasPermission("system:user:delete");
  const canAssignRoles =
    hasPermission("system:user:role") && hasPermission("system:role:list");
  const [form] = Form.useForm<UserForm>();
  const [rows, setRows] = useState<UserRow[]>([]);
  const [total, setTotal] = useState(0);
  const [query, setQuery] = useState<Query>({
    current: 1,
    pageSize: 20,
    username: "",
  });
  const [usernameInput, setUsernameInput] = useState("");
  const [loading, setLoading] = useState(false);
  const [editingId, setEditingId] = useState<number>();
  const [deletingId, setDeletingId] = useState<number>();
  const [modalOpen, setModalOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [roleUser, setRoleUser] = useState<UserRow>();
  const [roleOptions, setRoleOptions] = useState<RoleRow[]>([]);
  const [checkedRoleIDs, setCheckedRoleIDs] = useState<number[]>([]);
  const [roleLoading, setRoleLoading] = useState(false);
  const [roleSaving, setRoleSaving] = useState(false);

  const loadUsers = useCallback(async () => {
    setLoading(true);
    const result = await requestApi<UserRow[]>(
      api.api.listSystemUsers({
        page: query.current,
        page_size: query.pageSize,
        username: query.username || undefined,
      }),
      "获取用户列表失败",
    );
    setRows(result?.data ?? []);
    setTotal(result?.total ?? 0);
    setLoading(false);
  }, [query]);

  useEffect(() => {
    void loadUsers();
  }, [loadUsers]);

  const closeModal = () => {
    setModalOpen(false);
    setEditingId(undefined);
  };

  const openCreate = () => {
    setEditingId(undefined);
    form.resetFields();
    setModalOpen(true);
  };

  const openEdit = (user: UserRow) => {
    if (!user.id) return;
    setEditingId(user.id);
    form.setFieldsValue({
      username: user.username,
      nickname: user.nickname,
      password: "",
      status: user.status ?? 1,
    });
    setModalOpen(true);
  };

  const saveUser = async (values: UserForm) => {
    const isEdit = editingId !== undefined;
    const base = {
      username: values.username.trim(),
      nickname: values.nickname?.trim() ?? "",
      status: values.status,
    };
    const task = isEdit
      ? api.api.updateSystemUser({
          id: editingId,
          ...base,
          ...(values.password ? { password: values.password } : {}),
        })
      : api.api.createSystemUser({
          ...base,
          password: values.password ?? "",
        });

    setSaving(true);
    const result = await requestApi<UserRow>(
      task,
      isEdit ? "更新用户失败" : "新增用户失败",
    );
    setSaving(false);
    if (!result) return;

    message.success(isEdit ? "用户信息已更新" : "用户已新增");
    closeModal();
    await loadUsers();
  };

  const deleteUser = async (id?: number) => {
    if (!id) return;
    setDeletingId(id);
    const result = await requestApi(
      api.api.deleteSystemUser({ id }),
      "删除用户失败",
    );
    setDeletingId(undefined);
    if (!result) return;

    message.success("用户已删除");
    if (rows.length === 1 && query.current > 1) {
      setQuery((value) => ({ ...value, current: value.current - 1 }));
      return;
    }
    await loadUsers();
  };

  const openRoles = async (user: UserRow) => {
    if (!user.id) return;
    setRoleUser(user);
    setRoleLoading(true);
    const [all, assigned] = await Promise.all([
      requestApi<RoleRow[]>(
        api.api.listSystemRoles({ page: 1, page_size: 100 }),
        "获取角色列表失败",
      ),
      requestApi<CurrentRole[]>(api.api.getSystemUserRoles({ id: user.id }), "获取用户角色失败"),
    ]);
    setRoleOptions(all?.data ?? []);
    setCheckedRoleIDs((assigned?.data ?? []).flatMap((role) => (role.id ? [role.id] : [])));
    setRoleLoading(false);
  };

  const saveRoles = async () => {
    if (!roleUser?.id) return;
    setRoleSaving(true);
    const result = await requestApi(
      api.api.saveSystemUserRoles({ id: roleUser.id, role_ids: checkedRoleIDs }),
      "保存用户角色失败",
    );
    setRoleSaving(false);
    if (!result) return;
    message.success("用户角色已保存");
    setRoleUser(undefined);
  };

  const columns: ColumnsType<UserRow> = [
    { title: "用户名", dataIndex: "username", ellipsis: true },
    {
      title: "昵称",
      dataIndex: "nickname",
      ellipsis: true,
      render: (value) => value || "-",
    },
    {
      title: "状态",
      dataIndex: "status",
      width: 90,
      render: (value) =>
        value === 1 ? <Tag color="success">启用</Tag> : <Tag>禁用</Tag>,
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
          {canAssignRoles ? (
            <Button type="link" size="small" onClick={() => openRoles(record)}>
              分配角色
            </Button>
          ) : null}
          {canUpdate ? (
            <Button type="link" size="small" onClick={() => openEdit(record)}>
              编辑
            </Button>
          ) : null}
          {canDelete ? <Popconfirm
            title="删除这个用户？"
            description="删除后无法恢复"
            okText="删除"
            cancelText="取消"
            okButtonProps={{ danger: true, loading: deletingId === record.id }}
            onConfirm={() => deleteUser(record.id)}
          >
            <Button type="link" danger size="small">
              删除
            </Button>
          </Popconfirm> : null}
        </Space>
      ),
    },
  ];

  const handleSearch = () => {
    setQuery((current) => ({
      ...current,
      current: 1,
      username: usernameInput.trim(),
    }));
  };

  const handleReset = () => {
    setUsernameInput("");
    setQuery((current) => ({
      ...current,
      current: 1,
      username: "",
    }));
  };

  return (
    <div
      className="admin-list-page overflow-hidden rounded-lg"
      style={{
        backgroundColor: token.colorBgContainer,
        ...adminListVars(token),
      }}
    >
      <div className="admin-list-header">
        <PageHeader
          title="用户管理"
          description="管理系统后台操作员账户、关联角色与启用状态"
          extra={
            canCreate && (
              <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
                新增用户
              </Button>
            )
          }
        />
      </div>

      <ContentCard>
        {/* 筛选区 */}
        <form
          className="admin-list-filters"
          onSubmit={(e) => {
            e.preventDefault();
            handleSearch();
          }}
        >
          <Input
            prefix={<SearchOutlined style={{ color: token.colorTextSecondary }} />}
            allowClear
            placeholder="搜索用户名"
            value={usernameInput}
            onChange={(e) => setUsernameInput(e.target.value)}
            style={{ width: 220 }}
          />
          <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
            查询
          </Button>
          <Button onClick={handleReset}>重置</Button>
        </form>

        {/* 表格工具栏 */}
        <div className="admin-list-toolbar">
          <div className="flex flex-wrap items-center gap-3">
            <h2
              className="m-0 flex items-center gap-2 text-base font-semibold"
              style={{ color: token.colorTextHeading }}
            >
              <UserOutlined style={{ color: token.colorPrimary }} />
              用户列表
            </h2>
            <span className="text-sm" style={{ color: token.colorTextTertiary }}>
              共 {total} 位用户
            </span>
          </div>
          <Tooltip title="刷新列表">
            <Button
              type="text"
              icon={<ReloadOutlined spin={loading} />}
              onClick={() => void loadUsers()}
              style={{ color: token.colorTextSecondary }}
            >
              刷新
            </Button>
          </Tooltip>
        </div>

        {/* 表格数据 */}
        <div className="px-4">
          <Table<UserRow>
            rowKey="id"
            size="middle"
            loading={loading}
            columns={columns}
            dataSource={rows}
            scroll={{ x: 860 }}
            className="admin-list-table"
            pagination={false}
          />
        </div>

        {/* 分页 */}
        <div className="flex flex-wrap items-center justify-between gap-4 px-6 py-4">
          <div
            className="flex items-center gap-2 text-sm"
            style={{ color: token.colorTextSecondary }}
          >
            <span>每页显示</span>
            <Select
              aria-label="每页条数"
              className="w-18"
              size="small"
              value={query.pageSize}
              options={[10, 20, 50, 100].map((v) => ({
                value: v,
                label: v,
              }))}
              onChange={(pageSize) =>
                setQuery((value) => ({ ...value, current: 1, pageSize }))
              }
            />
            <span>条</span>
          </div>
          <Pagination
            className="admin-list-pagination"
            current={query.current}
            pageSize={query.pageSize}
            total={total}
            showSizeChanger={false}
            showQuickJumper
            locale={{ jump_to: "前往", page: "页" }}
            onChange={(current) =>
              setQuery((value) => ({ ...value, current }))
            }
          />
        </div>
      </ContentCard>

      <Modal
        title={editingId === undefined ? "新增用户" : "编辑用户"}
        open={modalOpen}
        width={460}
        destroyOnHidden
        okText="保存"
        cancelText="取消"
        confirmLoading={saving}
        onCancel={closeModal}
        onOk={() => form.submit()}
      >
        <Form<UserForm>
          form={form}
          layout="vertical"
          requiredMark={false}
          preserve={false}
          initialValues={{ status: 1 }}
          onFinish={saveUser}
        >
          <Form.Item
            name="username"
            label="用户名"
            rules={[
              { required: true, whitespace: true, message: "请输入用户名" },
              { max: 64, message: "用户名不能超过 64 个字符" },
            ]}
          >
            <Input
              disabled={editingId !== undefined}
              maxLength={64}
              placeholder="请输入用户名"
            />
          </Form.Item>
          <Form.Item name="nickname" label="昵称">
            <Input maxLength={64} placeholder="请输入昵称" />
          </Form.Item>
          <Form.Item
            name="password"
            label="密码"
            rules={[
              ...(editingId === undefined
                ? [{ required: true, message: "请输入密码" }]
                : []),
              { min: 6, max: 72, message: "密码长度为 6-72 位" },
            ]}
          >
            <Input.Password
              placeholder={editingId === undefined ? "请输入密码" : "不修改请留空"}
            />
          </Form.Item>
          <Form.Item
            name="status"
            label="状态"
            rules={[{ required: true, message: "请选择状态" }]}
          >
            <Select options={statusOptions} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={`为 ${roleUser?.username ?? "用户"} 分配角色`}
        open={Boolean(roleUser)}
        width={520}
        okText="保存"
        cancelText="取消"
        confirmLoading={roleSaving}
        okButtonProps={{ disabled: roleLoading }}
        onCancel={() => setRoleUser(undefined)}
        onOk={saveRoles}
      >
        <Spin spinning={roleLoading}>
          <Checkbox.Group
            value={checkedRoleIDs}
            onChange={(values) => setCheckedRoleIDs(values as number[])}
            style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 12 }}
          >
            {roleOptions.map((role) => (
              <Checkbox key={role.id} value={role.id} disabled={role.status !== 1}>
                {role.name} ({role.code})
              </Checkbox>
            ))}
          </Checkbox.Group>
        </Spin>
      </Modal>
    </div>
  );
}
