export * from "./generated_api";
import axios from "axios";
import {
  Api,
  FullRequestParams,
} from "./generated_api";

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

// 创建一个处理重定向的函数
const handleAuthFailure = () => {
  onAuthFailure();
};

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
    const httpResponse = await axios({
        url: path,
        params: query,
        responseType: format,
        data: body,
        headers: { token, "Content-Type": type },
        ...params,
      })
      .catch((error) => {
        if (axios.isAxiosError(error) && error.response?.status === 401) handleAuthFailure();
        return Promise.reject(error);
      });
    const data = httpResponse.data as ApiResponse<unknown>;
    if (data?.code != null && data.code < 0) handleAuthFailure();
    return httpResponse.data as T;
  };
}

export const api = new Req({
  secure: true,
  securityWorker: (data) => {
    return { headers: { token: readToken() } };
  },
});
