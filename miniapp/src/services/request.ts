import type { ApiResponse, RequestOptions, RequestData } from "@/types/api";
import { getToken, clearAuthStorage } from "@/utils";
import { isCurrentSessionToken } from "./auth-session";

/**
 * 获取默认请求 Base URL
 */
function getDefaultBaseUrl(): string {
  // #ifdef H5
  // H5 开发环境走相对路径触发 Vite 代理避免跨域，生产环境读取配置
  if (import.meta.env.DEV) {
    return "";
  }
  return (import.meta.env.VITE_API_BASE_URL as string) || "";
  // #endif

  // #ifndef H5
  // 小程序端使用完整绝对地址
  return (
    (import.meta.env.VITE_API_BASE_URL as string) || "https://gt-shop.xi-soft.cn"
  );
  // #endif
}

/**
 * 安全获取当前存储的会员 Token 纯字符串
 * 兼容处理原生 string 存储与 storage 包装对象格式（委托给统一 auth-storage）
 */
export function getStoredToken(): string {
  return getToken();
}

/**
 * 统一处理认证失效：仅当触发失效的请求 Token 仍代表当前会话时才执行全局登出。
 * 若 Storage 中的 Token 已变化（用户已 logout、切换账号或重新登录），
 * 旧请求的认证失败必须被忽略，不能影响当前账号。
 * App.vue 监听 auth-expired 事件后触发 memberStore.logout() 完成 Pinia 与 Storage 联动清理。
 */
export function handleAuthExpired(requestToken: string): void {
  if (!isCurrentSessionToken(requestToken, getStoredToken())) {
    return;
  }
  clearAuthStorage();
  uni.$emit("auth-expired");
}

/**
 * 统一网络请求封装
 */
export async function request<T>(
  options: RequestOptions,
): Promise<ApiResponse<T>> {
  const baseURL = options.baseURL ?? getDefaultBaseUrl();
  const rawUrl = options.url.startsWith("http")
    ? options.url
    : `${baseURL}${options.url}`;
  const method = options.method || "GET";
  const timeout = options.timeout ?? 15000;

  const header: Record<string, string> = {
    "Content-Type": "application/json",
    ...options.header,
  };

  // 请求创建阶段捕获本次请求使用的 Token：Authorization 与响应后的认证失效判断
  // 都必须基于该 Token，不能在响应返回后重新读取 Storage 误把当前 Token 当作本请求 Token
  const requestToken = getStoredToken();
  if (requestToken && !header["Authorization"]) {
    header["Authorization"] = `Bearer ${requestToken}`;
  }

  // 处理 URL 查询参数
  let finalUrl = rawUrl;
  if (options.params && Object.keys(options.params).length > 0) {
    const queryParams: string[] = [];
    for (const [key, value] of Object.entries(options.params)) {
      if (value !== undefined && value !== null) {
        queryParams.push(
          `${encodeURIComponent(key)}=${encodeURIComponent(String(value))}`,
        );
      }
    }
    if (queryParams.length > 0) {
      finalUrl += (finalUrl.includes("?") ? "&" : "?") + queryParams.join("&");
    }
  }

  let response: UniNamespace.RequestSuccessCallbackResult;
  try {
    response = await uni.request({
      url: finalUrl,
      method,
      data: options.data,
      header,
      timeout,
    });
  } catch (err: unknown) {
    const errorMsg =
      err instanceof Error ? err.message : "网络请求失败，请检查网络连接";
    console.error(`[Request Error] ${method} ${finalUrl}:`, errorMsg);
    throw new Error(errorMsg);
  }

  // HTTP 状态码校验
  if (response.statusCode === 401) {
    handleAuthExpired(requestToken);
    const authErrorMsg = "请先登录或登录凭证已失效";
    console.warn(`[Auth Error] ${method} ${finalUrl}: HTTP 401`);
    throw new Error(authErrorMsg);
  }

  if (response.statusCode < 200 || response.statusCode >= 300) {
    const httpErrorMsg = `HTTP 错误: ${response.statusCode}`;
    console.error(
      `[HTTP Error] ${method} ${finalUrl}:`,
      httpErrorMsg,
      response.data,
    );
    throw new Error(httpErrorMsg);
  }

  const result = response.data as ApiResponse<T>;

  // 业务状态码校验 (项目规范: code === 1 代表成功)
  if (result.code !== 1) {
    // 捕获未登录或凭证过期错误
    if (
      result.code === -1 ||
      (typeof result.message === "string" &&
        result.message.toLowerCase().includes("authentication required"))
    ) {
      handleAuthExpired(requestToken);
      const authErrorMsg = "请先登录或登录凭证已失效";
      console.warn(`[Auth Error] ${method} ${finalUrl}:`, result.message);
      throw new Error(authErrorMsg);
    }

    const bizErrorMsg = result.message || `业务错误(code: ${result.code})`;
    console.warn(`[Biz Error] ${method} ${finalUrl}:`, bizErrorMsg);
    throw new Error(bizErrorMsg);
  }

  return result;
}

/**
 * 常用 HTTP 方法封装
 */
export const http = {
  get<T>(
    url: string,
    params?: Record<string, unknown>,
    config?: Omit<RequestOptions, "url" | "method" | "params">,
  ) {
    return request<T>({ url, method: "GET", params, ...config });
  },
  post<T>(
    url: string,
    data?: RequestData,
    config?: Omit<RequestOptions, "url" | "method" | "data">,
  ) {
    return request<T>({ url, method: "POST", data, ...config });
  },
  put<T>(
    url: string,
    data?: RequestData,
    config?: Omit<RequestOptions, "url" | "method" | "data">,
  ) {
    return request<T>({ url, method: "PUT", data, ...config });
  },
  delete<T>(
    url: string,
    params?: Record<string, unknown>,
    config?: Omit<RequestOptions, "url" | "method" | "params">,
  ) {
    return request<T>({ url, method: "DELETE", params, ...config });
  },
  upload<T>(
    url: string,
    filePath: string,
    name = "file",
    formData?: Record<string, unknown>,
  ) {
    const baseURL = getDefaultBaseUrl();
    const rawUrl = url.startsWith("http") ? url : `${baseURL}${url}`;
    // 请求创建阶段捕获本次上传使用的 Token，响应认证失败时用于会话一致性判断
    const requestToken = getStoredToken();
    const header: Record<string, string> = {};
    if (requestToken) {
      header["Authorization"] = `Bearer ${requestToken}`;
    }

    return new Promise<ApiResponse<T>>((resolve, reject) => {
      uni.uploadFile({
        url: rawUrl,
        filePath,
        name,
        formData,
        header,
        success: (res) => {
          if (res.statusCode === 401) {
            handleAuthExpired(requestToken);
            reject(new Error("请先登录或登录凭证已失效"));
            return;
          }
          if (res.statusCode < 200 || res.statusCode >= 300) {
            reject(new Error(`HTTP 错误: ${res.statusCode}`));
            return;
          }
          try {
            const data = JSON.parse(res.data) as ApiResponse<T>;
            if (
              data.code === -1 ||
              (typeof data.message === "string" &&
                data.message.toLowerCase().includes("authentication required"))
            ) {
              handleAuthExpired(requestToken);
              reject(new Error("请先登录或登录凭证已失效"));
              return;
            }
            if (data.code !== 1) {
              reject(new Error(data.message || "上传失败"));
            } else {
              resolve(data);
            }
          } catch (e) {
            reject(new Error("解析响应失败"));
          }
        },
        fail: (err) => {
          reject(new Error(err.errMsg || "文件上传失败"));
        },
      });
    });
  },
};
