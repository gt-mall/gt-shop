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
  AppstoreOutlined,
  EditOutlined,
  PlusOutlined,
  ReloadOutlined,
  SearchOutlined,
} from "@ant-design/icons";

import { api } from "@/framework/http";
import type { ProductDictUnitRow } from "@/framework/http";
import { usePermission } from "@/framework/store";
import { ContentCard, PageHeader, adminListVars } from "@/shared/admin-ui";
import { formatTime } from "@/shared/utils/format";

interface UnitFormValues {
  name: string;
  description?: string;
  sort?: number;
  status: boolean;
}

export default function ProductDictUnitPage() {
  const { token } = theme.useToken();
  const hasPermission = usePermission();

  const canCreate = hasPermission("product:dict_unit:create");
  const canUpdate = hasPermission("product:dict_unit:update");
  const canDelete = hasPermission("product:dict_unit:delete");

  const [loading, setLoading] = useState(false);
  const [rows, setRows] = useState<ProductDictUnitRow[]>([]);
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
  const [editingRecord, setEditingRecord] = useState<ProductDictUnitRow | null>(null);
  const [form] = Form.useForm<UnitFormValues>();

  // Row action loading
  const [statusLoadingId, setStatusLoadingId] = useState<number | undefined>(undefined);
  const [deletingId, setDeletingId] = useState<number | undefined>(undefined);

  const loadData = useCallback(
    async (currentPage = page, size = pageSize, kw = activeKeyword, st = activeStatus) => {
      setLoading(true);
      const res = await api.api.listProductDictUnits({
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
        message.error(res.message || "加载商品单位列表失败");
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

  const handleOpenEdit = (record: ProductDictUnitRow) => {
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

  const handleToggleStatus = async (record: ProductDictUnitRow, checked: boolean) => {
    if (!record.id) return;
    setStatusLoadingId(record.id);
    const nextStatus = checked ? 1 : 0;
    const res = await api.api.updateProductDictUnit(record.id, {
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
    const res = await api.api.deleteProductDictUnit(id);
    setDeletingId(undefined);

    if (res.code === 1) {
      message.success("商品单位已删除");
      if (rows.length === 1 && page > 1) {
        setPage(page - 1);
        return;
      }
      void loadData();
    } else {
      message.error(res.message || "删除失败");
    }
  };

  const handleSave = async (values: UnitFormValues) => {
    setSaving(true);
    const statusVal = values.status ? 1 : 0;
    const sortVal = values.sort ?? 0;

    if (editingRecord?.id) {
      const res = await api.api.updateProductDictUnit(editingRecord.id, {
        name: values.name.trim(),
        description: values.description?.trim() ?? "",
        sort: sortVal,
        status: statusVal,
      });
      setSaving(false);

      if (res.code === 1) {
        message.success("商品单位已更新");
        setModalOpen(false);
        void loadData();
      } else {
        message.error(res.message || "更新商品单位失败");
      }
    } else {
      const res = await api.api.createProductDictUnit({
        name: values.name.trim(),
        description: values.description?.trim() ?? "",
        sort: sortVal,
        status: statusVal,
      });
      setSaving(false);

      if (res.code === 1) {
        message.success("商品单位已创建");
        setModalOpen(false);
        void loadData();
      } else {
        message.error(res.message || "创建商品单位失败");
      }
    }
  };

  const columns: ColumnsType<ProductDictUnitRow> = [
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
      title: "单位名称",
      dataIndex: "name",
      key: "name",
      width: 160,
      render: (val: string) => (
        <span
          className="inline-flex items-center px-2 py-0.5 rounded text-sm font-medium"
          style={{
            backgroundColor: token.colorPrimaryBg,
            color: token.colorPrimary,
            border: `1px solid ${token.colorPrimaryBorder}`,
          }}
        >
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
      title: "创建时间",
      dataIndex: "created_at",
      key: "created_at",
      width: 180,
      render: (val?: string) => (
        <span style={{ color: token.colorTextTertiary }}>{formatTime(val)}</span>
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
              title="确定要删除该商品单位吗？"
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
          title="商品单位"
          description="维护商品及规格使用的计量单位（如：个、件、台、箱、包、套等）"
          extra={
            canCreate && (
              <Button
                type="primary"
                icon={<PlusOutlined />}
                onClick={handleOpenCreate}
              >
                新增商品单位
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
            placeholder="搜索单位名称或说明"
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
              <AppstoreOutlined
                style={{ color: token.colorPrimary }}
              />
              商品单位列表
            </h2>
            <span
              className="text-sm"
              style={{ color: token.colorTextTertiary }}
            >
              共 {total} 个
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
          <Table<ProductDictUnitRow>
            rowKey="id"
            loading={loading}
            columns={columns}
            dataSource={rows}
            size="middle"
            className="admin-list-table"
            pagination={false}
            locale={{ emptyText: "暂无商品单位数据" }}
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
        title={editingRecord ? "编辑商品单位" : "新增商品单位"}
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
            label="单位名称"
            rules={[
              { required: true, message: "请输入单位名称" },
              { max: 32, message: "单位名称最多 32 个字符" },
            ]}
          >
            <Input placeholder="例如：个、件、台、箱、包、套" />
          </Form.Item>

          <Form.Item
            name="description"
            label="单位说明"
            rules={[{ max: 255, message: "说明最多 255 个字符" }]}
          >
            <Input.TextArea
              rows={3}
              placeholder="输入单位说明，可描述适用场景或包装特性"
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
