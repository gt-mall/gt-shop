export * from "./generated_api";
import axios from "axios";
import {
  Api,
  FullRequestParams,
} from "./generated_api";
// import { userStore } from '../models/user';
// import { jwtAtom, mainStore } from '../models/authHook';

export interface ApiResponse<T> {
  code?: number;
  data?: T;
  message?: string;
  total?: number;
}

// 创建一个处理重定向的函数
const handleAuthFailure = () => {
  // 使用 window.location.hash 进行重定向
  window.location.hash = "#/login";
  // 或者使用完整路径
  // window.location.href = '/#/login';
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
    const jwt = "";

    try {
      let httpResponse = await axios({
        url: path,
        params: query,
        responseType: format,
        data: body,
        headers: { token: jwt, "Content-Type": type },
        ...params,
      });
      let data = httpResponse.data as ApiResponse<T>;
      if (data?.code != null && data.code < 0) {
        //message.error(data?.meta?.message || '请重新登录');
        handleAuthFailure();
        return data;
      }
      return httpResponse.data;
    } catch (error) {
      // 处理请求错误
      if (axios.isAxiosError(error) && error.response?.status === 401) {
        handleAuthFailure();
      }
      throw error;
    }
  };
}

export const api = new Req({
  secure: true,
  securityWorker: (data) => {
    const jwt = "jwt";
    return { headers: { token: jwt } };
  },
});
