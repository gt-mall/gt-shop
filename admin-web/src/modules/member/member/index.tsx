import { useCallback, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  Avatar,
  Button,
  Card,
  Form,
  Input,
  Select,
  Space,
  Table,
  Tag,
  message,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import { UserOutlined } from "@ant-design/icons";

import { api, type ApiResponse } from "@/framework/http";
import type { MemberRow } from "@/framework/http/generated_api";
import { formatTime } from "@/shared/utils/format";

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
      render: (time: string) => formatTime(time),
    },
    {
      title: "注册时间",
      dataIndex: "created_at",
      key: "created_at",
      render: (time: string) => formatTime(time),
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
    <Space direction="vertical" size={16} style={{ width: "100%" }}>
      <Card size="small">
        <Form
          form={form}
          layout="inline"
          onFinish={onSearch}
          initialValues={{ keyword: "", status: undefined }}
        >
          <Form.Item label="关键词" name="keyword">
            <Input
              placeholder="搜索 ID / 昵称 / 手机号 / OpenID"
              allowClear
              style={{ width: 260 }}
            />
          </Form.Item>
          <Form.Item label="状态" name="status">
            <Select
              options={statusOptions}
              placeholder="选择状态"
              allowClear
              style={{ width: 120 }}
            />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                查询
              </Button>
              <Button onClick={onReset}>重置</Button>
            </Space>
          </Form.Item>
        </Form>
      </Card>

      <Card size="small">
        <Table<MemberRow>
          rowKey="id"
          loading={loading}
          columns={columns}
          dataSource={rows}
          pagination={{
            current: query.current,
            pageSize: query.pageSize,
            total,
            showSizeChanger: true,
            showTotal: (totalCount) => `共 ${totalCount} 条`,
            onChange: (page, pageSize) => {
              setQuery((prev) => ({
                ...prev,
                current: page,
                pageSize,
              }));
            },
          }}
        />
      </Card>
    </Space>
  );
}
