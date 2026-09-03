/* eslint-disable */
/* tslint:disable */
// @ts-nocheck
/*
 * ---------------------------------------------------------------
 * ## THIS FILE WAS GENERATED VIA SWAGGER-TYPESCRIPT-API        ##
 * ##                                                           ##
 * ## AUTHOR: acacode                                           ##
 * ## SOURCE: https://github.com/acacode/swagger-typescript-api ##
 * ---------------------------------------------------------------
 */

export interface CreateRequest {
  /** @maxLength 100 */
  nickname?: string;
  /**
   * @minLength 6
   * @maxLength 72
   */
  password: string;
  status?: number;
  /** @maxLength 64 */
  username: string;
}

export interface DemoData {
  /** @example "not_configured" */
  database?: string;
  /** @example "Mall API is running" */
  message?: string;
}

export interface HealthData {
  /** @example "not_configured" */
  database?: string;
  /** @example "mall-admin-core" */
  service?: string;
  /** @example "ok" */
  status?: string;
}

export interface HelloData {
  /** @example "Hello from plugin" */
  message?: string;
  /** @example "hello" */
  plugin?: string;
}

export interface IDRequest {
  id: number;
}

export interface ListRequest {
  page?: number;
  page_size?: number;
  username?: string;
}

export interface ResponseUserRow {
  code?: number;
  data?: UserRow;
  message?: string;
  total?: number;
}

export interface ResponseAny {
  code?: number;
  data?: any;
  message?: string;
  total?: number;
}

export interface ResponseArrayUserRow {
  code?: number;
  data?: UserRow[];
  message?: string;
  total?: number;
}

export interface ResponseDemoDemoData {
  code?: number;
  data?: DemoData;
  message?: string;
  total?: number;
}

export interface ResponseHealthHealthData {
  code?: number;
  data?: HealthData;
  message?: string;
  total?: number;
}

export interface ResponseHelloHelloData {
  code?: number;
  data?: HelloData;
  message?: string;
  total?: number;
}

export interface UpdateRequest {
  id: number;
  nickname?: string;
  password?: string;
  status?: number;
  username?: string;
}

export interface UserRow {
  created_at?: string;
  id?: number;
  nickname?: string;
  status?: number;
  updated_at?: string;
  username?: string;
}

import type {
  AxiosInstance,
  AxiosRequestConfig,
  HeadersDefaults,
  ResponseType,
} from "axios";
import axios from "axios";

export type QueryParamsType = Record<string | number, any>;

export interface FullRequestParams
  extends Omit<AxiosRequestConfig, "data" | "params" | "url" | "responseType"> {
  /** set parameter to `true` for call `securityWorker` for this request */
  secure?: boolean;
  /** request path */
  path: string;
  /** content type of request body */
  type?: ContentType;
  /** query params */
  query?: QueryParamsType;
  /** format of response (i.e. response.json() -> format: "json") */
  format?: ResponseType;
  /** request body */
  body?: unknown;
}

export type RequestParams = Omit<
  FullRequestParams,
  "body" | "method" | "query" | "path"
>;

export interface ApiConfig<SecurityDataType = unknown>
  extends Omit<AxiosRequestConfig, "data" | "cancelToken"> {
  securityWorker?: (
    securityData: SecurityDataType | null,
  ) => Promise<AxiosRequestConfig | void> | AxiosRequestConfig | void;
  secure?: boolean;
  format?: ResponseType;
}

export enum ContentType {
  Json = "application/json",
  JsonApi = "application/vnd.api+json",
  FormData = "multipart/form-data",
  UrlEncoded = "application/x-www-form-urlencoded",
  Text = "text/plain",
}

export class HttpClient<SecurityDataType = unknown> {
  public instance: AxiosInstance;
  private securityData: SecurityDataType | null = null;
  private securityWorker?: ApiConfig<SecurityDataType>["securityWorker"];
  private secure?: boolean;
  private format?: ResponseType;

  constructor({
    securityWorker,
    secure,
    format,
    ...axiosConfig
  }: ApiConfig<SecurityDataType> = {}) {
    this.instance = axios.create({
      ...axiosConfig,
      baseURL: axiosConfig.baseURL || "//localhost:8080",
    });
    this.secure = secure;
    this.format = format;
    this.securityWorker = securityWorker;
  }

  public setSecurityData = (data: SecurityDataType | null) => {
    this.securityData = data;
  };

  protected mergeRequestParams(
    params1: AxiosRequestConfig,
    params2?: AxiosRequestConfig,
  ): AxiosRequestConfig {
    const method = params1.method || (params2 && params2.method);

    return {
      ...this.instance.defaults,
      ...params1,
      ...(params2 || {}),
      headers: {
        ...((method &&
          this.instance.defaults.headers[
            method.toLowerCase() as keyof HeadersDefaults
          ]) ||
          {}),
        ...(params1.headers || {}),
        ...((params2 && params2.headers) || {}),
      },
    };
  }

  protected stringifyFormItem(formItem: unknown) {
    if (typeof formItem === "object" && formItem !== null) {
      return JSON.stringify(formItem);
    } else {
      return `${formItem}`;
    }
  }

  protected createFormData(input: Record<string, unknown>): FormData {
    if (input instanceof FormData) {
      return input;
    }
    return Object.keys(input || {}).reduce((formData, key) => {
      const property = input[key];
      const propertyContent: any[] =
        property instanceof Array ? property : [property];

      for (const formItem of propertyContent) {
        const isFileType = formItem instanceof Blob || formItem instanceof File;
        formData.append(
          key,
          isFileType ? formItem : this.stringifyFormItem(formItem),
        );
      }

      return formData;
    }, new FormData());
  }

  public request = async <T = any, _E = any>({
    secure,
    path,
    type,
    query,
    format,
    body,
    ...params
  }: FullRequestParams): Promise<T> => {
    const secureParams =
      ((typeof secure === "boolean" ? secure : this.secure) &&
        this.securityWorker &&
        (await this.securityWorker(this.securityData))) ||
      {};
    const requestParams = this.mergeRequestParams(params, secureParams);
    const responseFormat = format || this.format || undefined;

    if (
      type === ContentType.FormData &&
      body &&
      body !== null &&
      typeof body === "object"
    ) {
      body = this.createFormData(body as Record<string, unknown>);
    }

    if (
      type === ContentType.Text &&
      body &&
      body !== null &&
      typeof body !== "string"
    ) {
      body = JSON.stringify(body);
    }

    return this.instance
      .request({
        ...requestParams,
        headers: {
          ...(requestParams.headers || {}),
          ...(type ? { "Content-Type": type } : {}),
        },
        params: query,
        responseType: responseFormat,
        data: body,
        url: path,
      })
      .then((response) => response.data);
  };
}

/**
 * @title Mall Admin Core API
 * @version 0.1.0
 * @baseUrl //localhost:8080
 * @contact
 *
 * Simple commerce administration backend.
 */
export class Api<
  SecurityDataType extends unknown,
> extends HttpClient<SecurityDataType> {
  api = {
    /**
     * No description
     *
     * @tags demo
     * @name GetDemo
     * @summary Get backend demo data
     * @request GET:/api/demo
     */
    getDemo: (params: RequestParams = {}) =>
      this.request<ResponseDemoDemoData, any>({
        path: `/api/demo`,
        method: "GET",
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags plugins
     * @name GetHelloPlugin
     * @summary Get hello plugin data
     * @request GET:/api/plugins/hello
     */
    getHelloPlugin: (params: RequestParams = {}) =>
      this.request<ResponseHelloHelloData, any>({
        path: `/api/plugins/hello`,
        method: "GET",
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags system-user
     * @name CreateSystemUser
     * @summary Create a system user
     * @request POST:/api/system/users/create
     */
    createSystemUser: (request: CreateRequest, params: RequestParams = {}) =>
      this.request<ResponseUserRow, any>({
        path: `/api/system/users/create`,
        method: "POST",
        body: request,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags system-user
     * @name DeleteSystemUser
     * @summary Delete a system user
     * @request POST:/api/system/users/delete
     */
    deleteSystemUser: (request: IDRequest, params: RequestParams = {}) =>
      this.request<ResponseAny, any>({
        path: `/api/system/users/delete`,
        method: "POST",
        body: request,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags system-user
     * @name GetSystemUser
     * @summary Get a system user
     * @request POST:/api/system/users/detail
     */
    getSystemUser: (request: IDRequest, params: RequestParams = {}) =>
      this.request<ResponseUserRow, any>({
        path: `/api/system/users/detail`,
        method: "POST",
        body: request,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags system-user
     * @name ListSystemUsers
     * @summary List system users
     * @request POST:/api/system/users/list
     */
    listSystemUsers: (request: ListRequest, params: RequestParams = {}) =>
      this.request<ResponseArrayUserRow, any>({
        path: `/api/system/users/list`,
        method: "POST",
        body: request,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags system-user
     * @name UpdateSystemUser
     * @summary Update a system user
     * @request POST:/api/system/users/update
     */
    updateSystemUser: (request: UpdateRequest, params: RequestParams = {}) =>
      this.request<ResponseUserRow, any>({
        path: `/api/system/users/update`,
        method: "POST",
        body: request,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),
  };
  health = {
    /**
     * No description
     *
     * @tags system
     * @name GetHealth
     * @summary Check service health
     * @request GET:/health
     */
    getHealth: (params: RequestParams = {}) =>
      this.request<ResponseHealthHealthData, any>({
        path: `/health`,
        method: "GET",
        format: "json",
        ...params,
      }),
  };
}
