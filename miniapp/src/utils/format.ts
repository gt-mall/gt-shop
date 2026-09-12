/**
 * 业务数据脱敏与文本格式化工具
 */

/**
 * 手机号脱敏
 * 例如: '13812345678' -> '138****5678'
 */
export function maskPhone(phone?: string | null): string {
  if (!phone) return "";
  const trimmed = String(phone).trim();
  if (trimmed.length < 7) return trimmed;
  return trimmed.replace(/^(\d{3})\d+(\d{4})$/, "$1****$2");
}

/**
 * 姓名脱敏
 * 2个字: '张三' -> '张*'
 * 3个字: '李四五' -> '李*五'
 * 4个字以上: '欧阳六七' -> '欧**七'
 */
export function maskName(name?: string | null): string {
  if (!name) return "";
  const trimmed = String(name).trim();
  const len = trimmed.length;
  if (len <= 1) return trimmed;
  if (len === 2) return `${trimmed[0]}*`;
  return `${trimmed[0]}${"*".repeat(len - 2)}${trimmed[len - 1]}`;
}

/**
 * 文本超长截断并追加省略号
 */
export function truncate(
  str?: string | null,
  maxLength = 20,
  suffix = "...",
): string {
  if (!str) return "";
  if (str.length <= maxLength) return str;
  return str.slice(0, maxLength) + suffix;
}

/**
 * 解析 URL 参数串为键值对象
 * 例如: '?id=123&from=share' -> { id: '123', from: 'share' }
 */
export function parseQuery(queryStr?: string | null): Record<string, string> {
  if (!queryStr) return {};
  const cleanStr = queryStr.includes("?") ? queryStr.split("?")[1] : queryStr;
  const params: Record<string, string> = {};

  for (const pair of cleanStr.split("&")) {
    if (!pair) continue;
    const [key, value = ""] = pair.split("=");
    if (key) {
      params[decodeURIComponent(key)] = decodeURIComponent(value);
    }
  }

  return params;
}

/**
 * 解析并拼接完整的图片访问 URL
 * 兼容处理相对路径（如 media/2026/09/xxx.png 或 /media/...）
 */
export function resolveImageUrl(url?: string | null): string {
  if (!url) return "";
  let trimmed = String(url).trim();
  if (!trimmed) return "";

  // 智能清洗历史残留 IP 或旧端口的后端媒体文件地址 (例如 http://192.168.1.60:8080/media/...)
  if (trimmed.includes("/media/")) {
    const mediaIdx = trimmed.indexOf("/media/");
    trimmed = trimmed.slice(mediaIdx);
  } else if (
    trimmed.startsWith("http://") ||
    trimmed.startsWith("https://") ||
    trimmed.startsWith("data:") ||
    trimmed.startsWith("wxfile://") ||
    trimmed.startsWith("blob:")
  ) {
    return trimmed;
  }

  // 本地静态资源路径
  if (trimmed.startsWith("/static/") || trimmed.startsWith("static/")) {
    return trimmed.startsWith("/") ? trimmed : `/${trimmed}`;
  }

  // 拼接后端基础地址
  let baseUrl = "";
  // #ifdef H5
  if (import.meta.env.DEV) {
    // H5 开发模式下走 Vite 代理，使用相对路径 / 开头
    return trimmed.startsWith("/") ? trimmed : `/${trimmed}`;
  }
  baseUrl = (import.meta.env.VITE_API_BASE_URL as string) || "";
  // #endif

  // #ifndef H5
  // 小程序端必须使用完整 HTTP/HTTPS 绝对地址
  baseUrl =
    (import.meta.env.VITE_API_BASE_URL as string) || "http://192.168.1.60:8080";
  // #endif

  const cleanBase = baseUrl.replace(/\/+$/, "");
  const cleanPath = trimmed.startsWith("/") ? trimmed : `/${trimmed}`;

  return `${cleanBase}${cleanPath}`;
}

/**
 * 格式化商品规格 JSON 字符串为展示文本
 * 例如: '[{"name":"颜色","value_name":"白色"},{"name":"尺寸","value_name":"XL"}]' -> '白色 · XL'
 */
export function formatSpecs(specValStr: string | null | undefined): string {
  if (!specValStr) return "";
  try {
    const arr = JSON.parse(specValStr);
    if (Array.isArray(arr)) {
      return arr
        .map((item: any) => item.value_name || item.name || "")
        .filter(Boolean)
        .join(" · ");
    }
  } catch {}
  return specValStr;
}
