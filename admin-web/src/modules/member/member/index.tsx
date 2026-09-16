import { useCallback, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  Avatar,
  Button,
  Form,
  Input,
  Pagination,
  Select,
  Space,
  Table,
  Tag,
  Tooltip,
  message,
  theme,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import {
  ReloadOutlined,
  SearchOutlined,
  TeamOutlined,
  UserOutlined,
} from "@ant-design/icons";

import { api, type ApiResponse } from "@/framework/http";
import type { MemberRow } from "@/framework/http/generated_api";
import { ContentCard, PageHeader, adminListVars } from "@/shared/admin-ui";
import { formatTimeTwoLines } from "@/shared/utils/format";

type Query = {
  current: number;
  pageSize: number;
  keyword: string;
  status?: number;
};

const statusOptions = [
  { label: "全部", value: undefined },
  { label: "正常", value: 1 },
  { label: "禁用", value: 0 },
];

async function requestApi<T>(task: Promise<ApiResponse<T>>, fallback: string) {
  const result = await task;
  if (result.code === 1) return result;
  message.error(result.message || fallback);
}

export default function MemberListPage() {
  const navigate = useNavigate();
  const { token } = theme.useToken();
  const [form] = Form.useForm();
  const [rows, setRows] = useState<MemberRow[]>([]);
  const [total, setTotal] = useState(0);
  const [query, setQuery] = useState<Query>({
    current: 1,
    pageSize: 20,
    keyword: "",
    status: undefined,
  });
  const [loading, setLoading] = useState(false);

  const loadMembers = useCallback(async () => {
    setLoading(true);
    const result = await requestApi<MemberRow[]>(
      api.api.listMembers({
        page: query.current,
        page_size: query.pageSize,
        keyword: query.keyword || undefined,
        status: query.status,
      }),
      "获取会员列表失败",
    );
    setRows(result?.data ?? []);
    setTotal(result?.total ?? 0);
    setLoading(false);
  }, [query]);

  useEffect(() => {
    void loadMembers();
  }, [loadMembers]);

  const onSearch = (values: { keyword?: string; status?: number }) => {
    setQuery((prev) => ({
      ...prev,
      current: 1,
      keyword: values.keyword?.trim() ?? "",
      status: values.status,
    }));
  };

  const onReset = () => {
    form.resetFields();
    setQuery({
      current: 1,
      pageSize: 20,
      keyword: "",
      status: undefined,
    });
  };

  const columns: ColumnsType<MemberRow> = [
    {
      title: "头像",
      dataIndex: "avatar_url",
      key: "avatar_url",
      width: 80,
      render: (url: string) => (
        <Avatar src={url} icon={<UserOutlined />} size={40} />
      ),
    },
    {
      title: "ID",
      dataIndex: "id",
      key: "id",
      width: 100,
    },
    {
      title: "昵称",
      dataIndex: "nickname",
      key: "nickname",
      render: (text: string) => text || "-",
    },
    {
      title: "手机号",
      dataIndex: "mobile",
      key: "mobile",
      render: (text: string) => text || "-",
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 100,
      render: (status: number) =>
        status === 1 ? (
          <Tag color="success">正常</Tag>
        ) : (
          <Tag color="error">禁用</Tag>
        ),
    },
    {
      title: "最后登录时间",
      dataIndex: "last_login_at",
      key: "last_login_at",
      width: 120,
      render: (time: string) => formatTimeTwoLines(time),
    },
    {
      title: "注册时间",
      dataIndex: "created_at",
      key: "created_at",
      width: 120,
      render: (time: string) => formatTimeTwoLines(time),
    },
    {
      title: "操作",
      key: "action",
      width: 90,
      render: (_, record) => (
        <Button
          type="link"
          onClick={() => navigate(`/member/member/${record.id}`)}
        >
          查看
        </Button>
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
          title="会员管理"
          description="查看注册会员信息、消费数据、最后登录及账号状态"
        />
      </div>

      <ContentCard>
        {/* 筛选区 */}
        <Form
          form={form}
          className="admin-list-filters"
          onFinish={onSearch}
          initialValues={{ keyword: "", status: undefined }}
        >
          <Form.Item name="keyword" noStyle>
            <Input
              prefix={<SearchOutlined style={{ color: token.colorTextSecondary }} />}
              placeholder="搜索 ID / 昵称 / 手机号 / OpenID"
              allowClear
              style={{ width: 260 }}
            />
          </Form.Item>
          <Form.Item name="status" noStyle>
            <Select
              options={statusOptions}
              placeholder="选择状态"
              allowClear
              style={{ width: 120 }}
            />
          </Form.Item>
          <Button type="primary" htmlType="submit" icon={<SearchOutlined />}>
            查询
          </Button>
          <Button onClick={onReset}>重置</Button>
        </Form>

        {/* 表格工具栏 */}
        <div className="admin-list-toolbar">
          <div className="flex flex-wrap items-center gap-3">
            <h2
              className="m-0 flex items-center gap-2 text-base font-semibold"
              style={{ color: token.colorTextHeading }}
            >
              <TeamOutlined style={{ color: token.colorPrimary }} />
              会员列表
            </h2>
            <span className="text-sm" style={{ color: token.colorTextTertiary }}>
              共 {total} 位会员
            </span>
          </div>
          <Tooltip title="刷新列表">
            <Button
              type="text"
              icon={<ReloadOutlined spin={loading} />}
              onClick={() => void loadMembers()}
              style={{ color: token.colorTextSecondary }}
            >
              刷新
            </Button>
          </Tooltip>
        </div>

        {/* 表格数据 */}
        <div className="px-4">
          <Table<MemberRow>
            rowKey="id"
            size="middle"
            loading={loading}
            columns={columns}
            dataSource={rows}
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
                setQuery((prev) => ({ ...prev, current: 1, pageSize }))
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
            onChange={(page) =>
              setQuery((prev) => ({
                ...prev,
                current: page,
              }))
            }
          />
        </div>
      </ContentCard>
    </div>
  );
}
