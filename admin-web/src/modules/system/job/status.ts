export type JobStatusName =
  | "idle"
  | "running"
  | "retrying"
  | "success"
  | "failed"
  | string;

export interface JobStatusPresentation {
  label: string;
  color: string;
}

const statusPresentations: Record<string, JobStatusPresentation> = {
  idle: { label: "空闲", color: "default" },
  running: { label: "运行中", color: "processing" },
  retrying: { label: "等待重试", color: "warning" },
  success: { label: "成功", color: "success" },
  failed: { label: "最终失败", color: "error" },
};

export function getJobStatusPresentation(
  status?: JobStatusName,
): JobStatusPresentation {
  return statusPresentations[status ?? ""] ?? { label: "未知", color: "default" };
}

export function canRetryJob(
  status: JobStatusName | undefined,
  hasRetryPermission: boolean,
): boolean {
  return status === "failed" && hasRetryPermission;
}
