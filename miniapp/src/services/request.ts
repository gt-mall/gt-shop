import type { ApiResponse, RequestOptions, RequestData } from "@/types/api";

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
    (import.meta.env.VITE_API_BASE_URL as string) || "http://localhost:8081"
  );
  // #endif
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

  // Token 预留：若本地存储中存在 token 则携带 Authorization 请求头
  const token = uni.getStorageSync("token");
  if (token && typeof token === "string" && !header["Authorization"]) {
    header["Authorization"] = `Bearer ${token}`;
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
    const token = uni.getStorageSync("token");
    const header: Record<string, string> = {};
    if (token && typeof token === "string") {
      header["Authorization"] = `Bearer ${token}`;
    }

    return new Promise<ApiResponse<T>>((resolve, reject) => {
      uni.uploadFile({
        url: rawUrl,
        filePath,
        name,
        formData,
        header,
        success: (res) => {
          try {
            const data = JSON.parse(res.data) as ApiResponse<T>;
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
