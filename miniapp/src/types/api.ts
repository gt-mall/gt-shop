/**
 * 统一 API 响应包装接口
 */
export interface ApiResponse<T> {
  code: number
  message: string
  data: T
  total?: number
}

/**
 * 支持的 HTTP 请求方法
 */
export type HttpMethod = 'GET' | 'POST' | 'PUT' | 'DELETE' | 'OPTIONS' | 'HEAD'

/**
 * 请求体支持的数据类型
 */
export type RequestData = string | Record<string, any> | object | ArrayBuffer

/**
 * 统一请求参数配置
 */
export interface RequestOptions {
  url: string
  method?: HttpMethod
  data?: RequestData
  params?: Record<string, unknown>
  header?: Record<string, string>
  timeout?: number
  baseURL?: string
}
