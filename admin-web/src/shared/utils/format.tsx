import type React from "react";
import dayjs from "dayjs";

export const formatTime = (value?: string | null) =>
  value ? dayjs(value).format("YYYY-MM-DD HH:mm:ss") : "-";

export const formatTimeTwoLines = (
  value?: string | null,
  options?: { className?: string; style?: React.CSSProperties }
): React.ReactNode => {
  if (!value) return "-";
  const d = dayjs(value);
  if (!d.isValid()) return "-";
  return (
    <div
      className={`flex flex-col leading-tight tabular-nums ${options?.className || ""}`.trim()}
      style={options?.style}
    >
      <span>{d.format("YYYY-MM-DD")}</span>
      <span className="text-xs opacity-70 mt-0.5">{d.format("HH:mm:ss")}</span>
    </div>
  );
};
