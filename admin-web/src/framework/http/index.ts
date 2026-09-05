export * from "./generated_api";
import axios from "axios";
import { Api, type FullRequestParams } from "./generated_api";

let readToken = () => "";
let onAuthFailure = () => {};

export function configureAuthSession(tokenReader: () => string, failureHandler: () => void) {
  readToken = tokenReader;
  onAuthFailure = failureHandler;
}

export interface ApiResponse<T> {
  code?: number;
  data?: T;
  message?: string;
  total?: number;
}

class Req extends Api<any> {
  public request = async <T = any, _E = any>({
    secure,
    path,
    type,
    query,
    format,
    body,
    ...params
  }: FullRequestParams) => {
    const token = readToken();
    const httpResponse = await this.instance({
      url: path,
      params: query,
      responseType: format,
      data: body,
      ...params,
      headers: { ...params.headers, token, "Content-Type": type },
    });
    const data = httpResponse.data as ApiResponse<unknown>;
    if (typeof data?.code !== "number") {
      return { code: 0, message: "服务响应格式错误" } as T;
    }
    if (data.code < 0 && token === readToken()) onAuthFailure();
    return data as T;
  };
}

export const api = new Req({ baseURL: "/" });

api.instance.interceptors.response.use(
  (response) => response,
  (error: unknown) => {
    const response = axios.isAxiosError(error) ? error.response : undefined;
    return {
      data: {
        code: response?.status === 401 ? -1 : 0,
        message: response?.data?.message || (response ? `请求失败（${response.status}）` : "网络异常，请稍后重试"),
      },
    };
  },
);
