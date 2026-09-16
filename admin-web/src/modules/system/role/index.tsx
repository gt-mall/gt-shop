import { useCallback, useEffect, useState } from "react";
import {
  Button,
  Form,
  Input,
  InputNumber,
  message,
  Modal,
  Pagination,
  Popconfirm,
  Select,
  Space,
  Table,
  Tag,
  Tooltip,
  theme,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import {
  PlusOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
  SearchOutlined,
} from "@ant-design/icons";
import { useNavigate } from "react-router-dom";

import { api, type ApiResponse } from "@/framework/http";
import type { RoleRow } from "@/framework/http/generated_api";
import { usePermission } from "@/framework/store";
import { ContentCard, PageHeader, adminListVars } from "@/shared/admin-ui";

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
  const result = await task;
  if (result.code === 1) return result;
  message.error(result.message || fallback);
}

export default function RolePage() {
  const navigate = useNavigate();
  const { token } = theme.useToken();
  const hasPermission = usePermission();
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
  const [keywordInput, setKeywordInput] = useState("");
  const [statusFilterInput, setStatusFilterInput] = useState<number | undefined>(undefined);
  const [loading, setLoading] = useState(false);
  const [editingId, setEditingId] = useState<number>();
  const [editingCode, setEditingCode] = useState<string>();
  const [deletingId, setDeletingId] = useState<number>();
  const [modalOpen, setModalOpen] = useState(false);
  const [saving, setSaving] = useState(false);

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

  const openPermissions = (role: RoleRow) => {
    if (!role.id) return;
    navigate(`/system/role/${role.id}/permission`);
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

  const handleSearch = () => {
    setQuery((current) => ({
      ...current,
      current: 1,
      keyword: keywordInput.trim(),
      status: statusFilterInput,
    }));
  };

  const handleReset = () => {
    setKeywordInput("");
    setStatusFilterInput(undefined);
    setQuery((current) => ({
      ...current,
      current: 1,
      keyword: "",
      status: undefined,
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
          title="权限管理"
          description="管理系统管理人员的角色定义、操作权限分配与启用状态"
          extra={
            canCreate && (
              <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
                新增角色
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
            placeholder="搜索角色名称或编码"
            value={keywordInput}
            onChange={(e) => setKeywordInput(e.target.value)}
            style={{ width: 220 }}
          />
          <Select
            allowClear
            placeholder="状态"
            value={statusFilterInput}
            options={statusOptions}
            style={{ width: 120 }}
            onChange={setStatusFilterInput}
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
              <SafetyCertificateOutlined style={{ color: token.colorPrimary }} />
              角色列表
            </h2>
            <span className="text-sm" style={{ color: token.colorTextTertiary }}>
              共 {total} 个角色
            </span>
          </div>
          <Tooltip title="刷新列表">
            <Button
              type="text"
              icon={<ReloadOutlined spin={loading} />}
              onClick={() => void loadRoles()}
              style={{ color: token.colorTextSecondary }}
            >
              刷新
            </Button>
          </Tooltip>
        </div>

        {/* 表格数据 */}
        <div className="px-4">
          <Table<RoleRow>
            rowKey="id"
            size="middle"
            loading={loading}
            columns={columns}
            dataSource={rows}
            scroll={{ x: 980 }}
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
    </div>
  );
}
