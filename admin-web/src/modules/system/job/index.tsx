import { useCallback, useEffect, useRef, useState } from "react";
import {
  Alert,
  Button,
  Popconfirm,
  Space,
  Table,
  Tag,
  Tooltip,
  message,
  theme,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import { ReloadOutlined } from "@ant-design/icons";
import dayjs from "dayjs";

import { api, type SystemJobStatus } from "@/framework/http";
import { usePermission } from "@/framework/store";
import { ContentCard, PageHeader, adminListVars } from "@/shared/admin-ui";
import { canRetryJob, getJobStatusPresentation } from "./status";

const refreshInterval = 30_000;

function formatTime(value?: string) {
  return value ? dayjs(value).format("YYYY-MM-DD HH:mm:ss") : "—";
}

function formatDuration(value?: number) {
  if (value === undefined || value === null) return "—";
  if (value < 1000) return `${value} ms`;
  return `${(value / 1000).toFixed(2)} 秒`;
}

export default function SystemJobPage() {
  const { token } = theme.useToken();
  const hasPermission = usePermission();
  const canRetry = hasPermission("system:job:retry");
  const [rows, setRows] = useState<SystemJobStatus[]>([]);
  const [loading, setLoading] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string>();
  const [retryingJobs, setRetryingJobs] = useState<string[]>([]);
  const latestRequest = useRef(0);

  const loadJobs = useCallback(async () => {
    const requestId = ++latestRequest.current;
    setLoading(true);
    const result = await api.api.listSystemJobs();
    if (requestId !== latestRequest.current) return;
    setLoading(false);
    if (result.code === 1) {
      setRows(result.data ?? []);
      setErrorMessage(undefined);
      return;
    }
    setErrorMessage(result.message || "获取任务状态失败，请重试");
  }, []);

  useEffect(() => {
    void loadJobs();
    const timer = window.setInterval(() => void loadJobs(), refreshInterval);
    return () => {
      latestRequest.current++;
      window.clearInterval(timer);
    };
  }, [loadJobs]);

  const retryJob = async (record: SystemJobStatus) => {
    const jobName = record.job_name;
    if (!jobName) return;
    setRetryingJobs((current) => [...current, jobName]);
    const result = await api.api.retrySystemJob(jobName);
    setRetryingJobs((current) => current.filter((name) => name !== jobName));
    if (result.code === 1) {
      message.success("任务已接受，将在后台执行");
      await loadJobs();
      return;
    }
    message.error(result.message || "任务重试失败");
  };

  const columns: ColumnsType<SystemJobStatus> = [
    {
      title: "任务名称",
      dataIndex: "display_name",
      width: 150,
      render: (value: string, record) => (
        <div className="min-w-0">
          <div className="font-medium" style={{ color: token.colorTextHeading }}>
            {value || record.job_name || "—"}
          </div>
          <div className="text-xs" style={{ color: token.colorTextTertiary }}>
            {record.job_name || "—"}
          </div>
        </div>
      ),
    },
    {
      title: "状态",
      dataIndex: "status",
      width: 105,
      render: (status?: string) => {
        const presentation = getJobStatusPresentation(status);
        return <Tag color={presentation.color}>{presentation.label}</Tag>;
      },
    },
    {
      title: "最近开始",
      dataIndex: "last_started_at",
      width: 155,
      render: formatTime,
    },
    {
      title: "最近完成",
      dataIndex: "last_finished_at",
      width: 155,
      render: formatTime,
    },
    {
      title: "最近成功",
      dataIndex: "last_success_at",
      width: 155,
      render: formatTime,
    },
    {
      title: "最近耗时",
      dataIndex: "last_duration_ms",
      width: 105,
      align: "right",
      render: formatDuration,
    },
    {
      title: "处理 / 成功 / 失败",
      key: "counts",
      width: 145,
      align: "right",
      render: (_, record) =>
        `${record.last_processed_count ?? 0} / ${record.last_success_count ?? 0} / ${record.last_failed_count ?? 0}`,
    },
    {
      title: "尝试",
      dataIndex: "last_attempt_count",
      width: 74,
      align: "right",
      render: (value?: number) => value ?? 0,
    },
    {
      title: "连续失败",
      dataIndex: "consecutive_failures",
      width: 90,
      align: "right",
      render: (value?: number) => value ?? 0,
    },
    {
      title: "下次重试",
      dataIndex: "next_retry_at",
      width: 155,
      render: formatTime,
    },
    {
      title: "最近错误",
      dataIndex: "last_error",
      width: 210,
      ellipsis: true,
      render: (value?: string) =>
        value ? (
          <Tooltip title={value} placement="topLeft">
            <span className="block max-w-[190px] truncate">{value}</span>
          </Tooltip>
        ) : (
          <span style={{ color: token.colorTextTertiary }}>—</span>
        ),
    },
    {
      title: "操作",
      key: "actions",
      fixed: "right",
      width: 120,
      render: (_, record) => {
        const jobName = record.job_name;
        const active = record.status === "running" || record.status === "retrying";
        if (!canRetryJob(record.status, canRetry) || !jobName) return "—";
        return (
          <Popconfirm
            title="重新执行这个任务？"
            description="任务将在后台执行，可能会重复扫描未完成记录。"
            okText="重新执行"
            cancelText="取消"
            onConfirm={() => retryJob(record)}
          >
            <Button
              type="link"
              size="small"
              loading={retryingJobs.includes(jobName)}
              disabled={active || retryingJobs.includes(jobName)}
            >
              重新执行
            </Button>
          </Popconfirm>
        );
      },
    },
  ];

  return (
    <div
      className="admin-list-page"
      style={{
        ...adminListVars(token),
        minWidth: 0,
      }}
    >
      <div className="admin-list-header">
        <PageHeader
          title="任务监控"
          description="查看后台定时任务状态、最近执行情况和自动重试进度。"
          extra={
            <Button icon={<ReloadOutlined />} loading={loading} onClick={() => void loadJobs()}>
              刷新
            </Button>
          }
        />
      </div>

      <ContentCard>
        <div className="admin-list-toolbar flex items-center justify-between">
          <div className="font-medium" style={{ color: token.colorTextHeading }}>
            内置任务
          </div>
          <Space size={8}>
            <span className="text-xs" style={{ color: token.colorTextTertiary }}>
              每 30 秒自动刷新
            </span>
          </Space>
        </div>
        {errorMessage && (
          <Alert
            className="mx-5 mb-3"
            type="error"
            showIcon
            message={errorMessage}
            action={<Button size="small" onClick={() => void loadJobs()}>重试加载</Button>}
          />
        )}
        <Table<SystemJobStatus>
          className="admin-list-table"
          rowKey={(record) => record.job_name || "unknown"}
          columns={columns}
          dataSource={rows}
          loading={loading}
          pagination={false}
          scroll={{ x: 1460 }}
          locale={{ emptyText: errorMessage ? "任务状态加载失败" : "暂无任务状态" }}
          size="middle"
        />
      </ContentCard>
    </div>
  );
}
