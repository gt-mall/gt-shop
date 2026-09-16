import { useCallback, useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import {
  Avatar,
  Button,
  Card,
  Descriptions,
  Space,
  Table,
  Tag,
  message,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import { ArrowLeftOutlined, UserOutlined } from "@ant-design/icons";

import { api, type ApiResponse } from "@/framework/http";
import type {
  MemberAddressRow,
  MemberAuthRow,
  MemberDetailResponse,
} from "@/framework/http/generated_api";
import { formatTime, formatTimeTwoLines } from "@/shared/utils/format";

async function requestApi<T>(task: Promise<ApiResponse<T>>, fallback: string) {
  const result = await task;
  if (result.code === 1) return result;
  message.error(result.message || fallback);
}

export default function MemberDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [detail, setDetail] = useState<MemberDetailResponse>();
  const [loading, setLoading] = useState(false);

  const loadDetail = useCallback(async () => {
    const memberId = Number(id);
    if (!memberId || isNaN(memberId)) {
      message.error("无效的会员 ID");
      return;
    }
    setLoading(true);
    const result = await requestApi<MemberDetailResponse>(
      api.api.getMemberDetail({ id: memberId }),
      "获取会员详情失败",
    );
    if (result?.data) {
      setDetail(result.data);
    }
    setLoading(false);
  }, [id]);

  useEffect(() => {
    void loadDetail();
  }, [loadDetail]);

  const authColumns: ColumnsType<MemberAuthRow> = [
    {
      title: "认证类型 (auth_type)",
      dataIndex: "auth_type",
      key: "auth_type",
      width: 160,
      render: (val: string) => val || "-",
    },
    {
      title: "AppID (app_id)",
      dataIndex: "app_id",
      key: "app_id",
      width: 180,
      render: (val: string) => val || "-",
    },
    {
      title: "OpenID (openid)",
      dataIndex: "openid",
      key: "openid",
      width: 220,
      render: (val: string) => val || "-",
    },
    {
      title: "UnionID (unionid)",
      dataIndex: "unionid",
      key: "unionid",
      width: 180,
      render: (val: string) => val || "-",
    },
    {
      title: "扩展信息 (provider_data)",
      dataIndex: "provider_data",
      key: "provider_data",
      render: (val?: string) =>
        val ? (
          <pre
            style={{
              margin: 0,
              maxHeight: 150,
              overflow: "auto",
              fontSize: 12,
              background: "#fafafa",
              padding: "6px 10px",
              borderRadius: 4,
              border: "1px solid #f0f0f0",
              fontFamily: "monospace",
            }}
          >
            {val}
          </pre>
        ) : (
          "-"
        ),
    },
    {
      title: "创建时间",
      dataIndex: "created_at",
      key: "created_at",
      width: 120,
      render: (time: string) => formatTimeTwoLines(time),
    },
  ];

  const addressColumns: ColumnsType<MemberAddressRow> = [
    {
      title: "收货人",
      dataIndex: "name",
      key: "name",
      width: 120,
      render: (text: string) => text || "-",
    },
    {
      title: "手机号",
      dataIndex: "mobile",
      key: "mobile",
      width: 140,
      render: (text: string) => text || "-",
    },
    {
      title: "省/市/区",
      key: "region",
      width: 180,
      render: (_, r) =>
        [r.province, r.city, r.district].filter(Boolean).join(" / ") || "-",
    },
    {
      title: "地区编码",
      key: "region_code",
      width: 200,
      render: (_, r) =>
        [r.province_code, r.city_code, r.district_code]
          .filter(Boolean)
          .join(" / ") || "-",
    },
    {
      title: "详细地址",
      dataIndex: "address",
      key: "address",
      render: (text: string) => text || "-",
    },
    {
      title: "邮编",
      dataIndex: "postal_code",
      key: "postal_code",
      width: 100,
      render: (text: string) => text || "-",
    },
    {
      title: "默认地址",
      dataIndex: "is_default",
      key: "is_default",
      width: 100,
      render: (isDefault: number) =>
        isDefault === 1 ? (
          <Tag color="blue">默认地址</Tag>
        ) : (
          <Tag>普通地址</Tag>
        ),
    },
    {
      title: "创建时间",
      dataIndex: "created_at",
      key: "created_at",
      width: 120,
      render: (time: string) => formatTimeTwoLines(time),
    },
  ];

  return (
    <Space direction="vertical" size={16} style={{ width: "100%" }}>
      <Card size="small">
        <Space direction="horizontal" style={{ width: "100%", justifyContent: "space-between" }}>
          <Space>
            <Button
              icon={<ArrowLeftOutlined />}
              onClick={() => navigate("/member/member")}
            >
              返回会员列表
            </Button>
            <span style={{ fontSize: 16, fontWeight: 600 }}>会员详情</span>
          </Space>
        </Space>
      </Card>

      {/* 1. 基本信息 */}
      <Card title="基本信息" size="small" loading={loading}>
        <Descriptions bordered column={{ xs: 1, sm: 2, md: 3 }}>
          <Descriptions.Item label="会员 ID">
            {detail?.member?.id ?? "-"}
          </Descriptions.Item>
          <Descriptions.Item label="头像">
            <Avatar
              src={detail?.member?.avatar_url}
              icon={<UserOutlined />}
              size={56}
            />
          </Descriptions.Item>
          <Descriptions.Item label="昵称">
            {detail?.member?.nickname || "-"}
          </Descriptions.Item>
          <Descriptions.Item label="手机号">
            {detail?.member?.mobile || "-"}
          </Descriptions.Item>
          <Descriptions.Item label="状态">
            {detail?.member?.status === 1 ? (
              <Tag color="success">正常</Tag>
            ) : (
              <Tag color="error">禁用</Tag>
            )}
          </Descriptions.Item>
          <Descriptions.Item label="最后登录时间">
            {formatTime(detail?.member?.last_login_at)}
          </Descriptions.Item>
          <Descriptions.Item label="注册时间">
            {formatTime(detail?.member?.created_at)}
          </Descriptions.Item>
        </Descriptions>
      </Card>

      {/* 2. 登录身份 */}
      <Card title="登录身份" size="small" loading={loading}>
        <Table<MemberAuthRow>
          rowKey={(r) => `${r.auth_type}_${r.app_id}_${r.openid}`}
          columns={authColumns}
          dataSource={detail?.auth ?? []}
          pagination={false}
        />
      </Card>

      {/* 3. 收货地址 */}
      <Card title="收货地址" size="small" loading={loading}>
        <Table<MemberAddressRow>
          rowKey="id"
          columns={addressColumns}
          dataSource={detail?.addresses ?? []}
          pagination={false}
        />
      </Card>
    </Space>
  );
}
