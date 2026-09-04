import dayjs from "dayjs";

export const formatTime = (value?: string) =>
  value ? dayjs(value).format("YYYY-MM-DD HH:mm:ss") : "-";
