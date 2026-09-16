import { useCallback, useEffect, useState } from "react";
import {
  Button,
  Form,
  Input,
  InputNumber,
  Modal,
  Pagination,
  Popconfirm,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  Tooltip,
  message,
  theme,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import {
  EditOutlined,
  PlusOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
  SearchOutlined,
} from "@ant-design/icons";

import { api } from "@/framework/http";
import type { ProductDictServiceRow } from "@/framework/http";
import { usePermission } from "@/framework/store";
import { ContentCard, PageHeader, adminListVars } from "@/shared/admin-ui";

interface ServiceFormValues {
  name: string;
  description?: string;
  sort?: number;
  status: boolean;
}

export default function ProductDictServicePage() {
  const { token } = theme.useToken();
  const hasPermission = usePermission();

  const canCreate = hasPermission("product:dict_service:create");
  const canUpdate = hasPermission("product:dict_service:update");
  const canDelete = hasPermission("product:dict_service:delete");

  const [loading, setLoading] = useState(false);
  const [rows, setRows] = useState<ProductDictServiceRow[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  // Filters
  const [keywordInput, setKeywordInput] = useState("");
  const [statusFilter, setStatusFilter] = useState<number | undefined>(undefined);
  const [activeKeyword, setActiveKeyword] = useState("");
  const [activeStatus, setActiveStatus] = useState<number | undefined>(undefined);

  // Modal
  const [modalOpen, setModalOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [editingRecord, setEditingRecord] = useState<ProductDictServiceRow | null>(null);
  const [form] = Form.useForm<ServiceFormValues>();

  // Row action loading
  const [statusLoadingId, setStatusLoadingId] = useState<number | undefined>(undefined);
  const [deletingId, setDeletingId] = useState<number | undefined>(undefined);

  const loadData = useCallback(
    async (currentPage = page, size = pageSize, kw = activeKeyword, st = activeStatus) => {
      setLoading(true);
      const res = await api.api.listProductDictServices({
        page: currentPage,
        page_size: size,
        keyword: kw.trim() || undefined,
        status: st,
      });
      setLoading(false);

      if (res.code === 1 && res.data) {
        setRows(res.data);
        setTotal(res.total || 0);
      } else {
        message.error(res.message || "加载服务保障列表失败");
      }
    },
    [page, pageSize, activeKeyword, activeStatus],
  );

  useEffect(() => {
    void loadData();
  }, [loadData]);

  const handleSearch = () => {
    setActiveKeyword(keywordInput);
    setActiveStatus(statusFilter);
    setPage(1);
    void loadData(1, pageSize, keywordInput, statusFilter);
  };

  const handleReset = () => {
    setKeywordInput("");
    setStatusFilter(undefined);
    setActiveKeyword("");
    setActiveStatus(undefined);
    setPage(1);
    void loadData(1, pageSize, "", undefined);
  };

  const handleOpenCreate = () => {
    setEditingRecord(null);
    form.resetFields();
    form.setFieldsValue({
      sort: 0,
      status: true,
    });
    setModalOpen(true);
  };

  const handleOpenEdit = (record: ProductDictServiceRow) => {
    setEditingRecord(record);
    form.resetFields();
    form.setFieldsValue({
      name: record.name,
      description: record.description,
      sort: record.sort,
      status: record.status === 1,
    });
    setModalOpen(true);
  };

  const handleToggleStatus = async (record: ProductDictServiceRow, checked: boolean) => {
    if (!record.id) return;
    setStatusLoadingId(record.id);
    const nextStatus = checked ? 1 : 0;
    const res = await api.api.updateProductDictService(record.id, {
      name: record.name ?? "",
      description: record.description,
      sort: record.sort,
      status: nextStatus,
    });
    setStatusLoadingId(undefined);

    if (res.code === 1) {
      message.success("状态已更新");
      void loadData();
    } else {
      message.error(res.message || "更新状态失败");
    }
  };

  const handleDelete = async (id?: number) => {
    if (!id) return;
    setDeletingId(id);
    const res = await api.api.deleteProductDictService(id);
    setDeletingId(undefined);

    if (res.code === 1) {
      message.success("服务保障已删除");
      if (rows.length === 1 && page > 1) {
        setPage(page - 1);
        return;
      }
      void loadData();
    } else {
      message.error(res.message || "删除失败");
    }
  };

  const handleSave = async (values: ServiceFormValues) => {
    setSaving(true);
    const statusVal = values.status ? 1 : 0;
    const sortVal = values.sort ?? 0;

    if (editingRecord?.id) {
      const res = await api.api.updateProductDictService(editingRecord.id, {
        name: values.name.trim(),
        description: values.description?.trim() ?? "",
        sort: sortVal,
        status: statusVal,
      });
      setSaving(false);

      if (res.code === 1) {
        message.success("服务保障已更新");
        setModalOpen(false);
        void loadData();
      } else {
        message.error(res.message || "更新服务保障失败");
      }
    } else {
      const res = await api.api.createProductDictService({
        name: values.name.trim(),
        description: values.description?.trim() ?? "",
        sort: sortVal,
        status: statusVal,
      });
      setSaving(false);

      if (res.code === 1) {
        message.success("服务保障已创建");
        setModalOpen(false);
        void loadData();
      } else {
        message.error(res.message || "创建服务保障失败");
      }
    }
  };

  const columns: ColumnsType<ProductDictServiceRow> = [
    {
      title: "ID",
      dataIndex: "id",
      key: "id",
      width: 80,
      align: "center",
      render: (val: number) => (
        <span style={{ color: token.colorTextTertiary }}>{val}</span>
      ),
    },
    {
      title: "保障名称",
      dataIndex: "name",
      key: "name",
      width: 180,
      render: (val: string) => (
        <span className="font-medium" style={{ color: token.colorTextHeading }}>
          {val}
        </span>
      ),
    },
    {
      title: "说明",
      dataIndex: "description",
      key: "description",
      ellipsis: true,
      render: (val: string) =>
        val ? (
          <span style={{ color: token.colorTextSecondary }}>{val}</span>
        ) : (
          <span style={{ color: token.colorTextQuaternary }}>—</span>
        ),
    },
    {
      title: "排序",
      dataIndex: "sort",
      key: "sort",
      width: 100,
      align: "center",
      render: (val: number) => (
        <span style={{ color: token.colorTextSecondary }}>{val}</span>
      ),
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 110,
      align: "center",
      render: (status: number, record) =>
        canUpdate ? (
          <Switch
            checked={status === 1}
            checkedChildren="启用"
            unCheckedChildren="禁用"
            loading={statusLoadingId === record.id}
            onChange={(checked) => void handleToggleStatus(record, checked)}
          />
        ) : (
          <Tag color={status === 1 ? "success" : "default"}>
            {status === 1 ? "启用" : "禁用"}
          </Tag>
        ),
    },
    {
      title: "操作",
      key: "actions",
      width: 130,
      align: "center",
      fixed: "right",
      render: (_, record) => (
        <Space size="small">
          {canUpdate && (
            <Button
              type="link"
              size="small"
              icon={<EditOutlined />}
              onClick={() => handleOpenEdit(record)}
            >
              编辑
            </Button>
          )}
          {canDelete && (
            <Popconfirm
              title="确定要删除该服务保障吗？"
              description="删除后不可恢复，请确认。"
              okText="删除"
              cancelText="取消"
              okButtonProps={{ danger: true, loading: deletingId === record.id }}
              onConfirm={() => void handleDelete(record.id)}
            >
              <Button type="link" size="small" danger>
                删除
              </Button>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ];

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
          title="服务保障"
          description="维护商品详情及售后展示的服务保障选项（如7天无理由退货、退货包运费、正品保障等）"
          extra={
            canCreate && (
              <Button
                type="primary"
                icon={<PlusOutlined />}
                onClick={handleOpenCreate}
              >
                新增服务保障
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
            placeholder="搜索保障名称或说明"
            allowClear
            value={keywordInput}
            onChange={(e) => setKeywordInput(e.target.value)}
            style={{ width: 220 }}
          />
          <Select
            placeholder="状态"
            allowClear
            value={statusFilter}
            onChange={setStatusFilter}
            style={{ width: 120 }}
            options={[
              { label: "启用", value: 1 },
              { label: "禁用", value: 0 },
            ]}
          />
          <Button
            type="primary"
            icon={<SearchOutlined />}
            onClick={handleSearch}
          >
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
              <SafetyCertificateOutlined
                style={{ color: token.colorPrimary }}
              />
              服务保障列表
            </h2>
            <span
              className="text-sm"
              style={{ color: token.colorTextTertiary }}
            >
              共 {total} 项
            </span>
          </div>
          <Tooltip title="刷新列表">
            <Button
              type="text"
              icon={<ReloadOutlined spin={loading} />}
              onClick={() => void loadData()}
              style={{ color: token.colorTextSecondary }}
            >
              刷新
            </Button>
          </Tooltip>
        </div>

        {/* 表格数据 */}
        <div className="px-4">
          <Table<ProductDictServiceRow>
            rowKey="id"
            loading={loading}
            columns={columns}
            dataSource={rows}
            size="middle"
            className="admin-list-table"
            pagination={false}
            locale={{ emptyText: "暂无服务保障数据" }}
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
              value={pageSize}
              options={[10, 20, 50, 100].map((v) => ({
                value: v,
                label: v,
              }))}
              onChange={(val) => {
                setPage(1);
                setPageSize(val);
              }}
            />
            <span>条</span>
          </div>
          <Pagination
            className="admin-list-pagination"
            current={page}
            pageSize={pageSize}
            total={total}
            showSizeChanger={false}
            showQuickJumper
            locale={{ jump_to: "前往", page: "页" }}
            onChange={(p) => setPage(p)}
          />
        </div>
      </ContentCard>

      {/* 新增 / 编辑弹窗 */}
      <Modal
        title={editingRecord ? "编辑服务保障" : "新增服务保障"}
        open={modalOpen}
        onCancel={() => setModalOpen(false)}
        onOk={() => form.submit()}
        confirmLoading={saving}
        destroyOnClose
        width={500}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={(values) => void handleSave(values)}
          autoComplete="off"
          className="pt-2"
        >
          <Form.Item
            name="name"
            label="保障名称"
            rules={[
              { required: true, message: "请输入保障名称" },
              { max: 64, message: "保障名称最多 64 个字符" },
            ]}
          >
            <Input placeholder="例如：7天无理由退货、正品保障、极速退款" />
          </Form.Item>

          <Form.Item
            name="description"
            label="保障说明"
            rules={[{ max: 255, message: "说明最多 255 个字符" }]}
          >
            <Input.TextArea
              rows={3}
              placeholder="输入保障详细说明，展示于商品详情页或保障弹窗中"
            />
          </Form.Item>

          <Form.Item
            name="sort"
            label="排序值"
            extra="数值越小越靠前显示，默认为 0"
          >
            <InputNumber min={0} max={99999} style={{ width: "100%" }} />
          </Form.Item>

          <Form.Item
            name="status"
            label="状态"
            valuePropName="checked"
          >
            <Switch checkedChildren="启用" unCheckedChildren="禁用" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
