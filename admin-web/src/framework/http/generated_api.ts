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

export interface CurrentRole {
  code?: string;
  id?: number;
  name?: string;
  status?: number;
}

export interface CurrentUser {
  created_at?: string;
  id?: number;
  nickname?: string;
  status?: number;
  updated_at?: string;
  username?: string;
}

export interface CurrentUserResponse {
  permissions?: string[];
  roles?: CurrentRole[];
  user?: CurrentUser;
}

export interface FileGroupCounts {
  groups: FileGroupRow[];
  total: number;
  total_storage_size: number;
  ungrouped: number;
  ungrouped_storage_size: number;
}

export interface FileGroupIDRequest {
  id: number;
}

export interface FileGroupRow {
  count: number;
  id: number;
  name: string;
  storage_size: number;
}

export interface FileGroupSaveRequest {
  /** @min 0 */
  id?: number;
  /** @maxLength 64 */
  name: string;
}

export interface FileIDsRequest {
  /**
   * @maxItems 100
   * @minItems 1
   */
  ids: number[];
}

export interface FileListRequest {
  /** @min 0 */
  group_id?: number;
  /** @maxLength 255 */
  keyword?: string;
  /** @min 1 */
  page?: number;
  /**
   * @min 1
   * @max 100
   */
  page_size?: number;
}

export interface FileLocalConfig {
  domain?: string;
  path: string;
}

export interface FileMoveRequest {
  /** @min 0 */
  group_id?: number;
  /**
   * @maxItems 100
   * @minItems 1
   */
  ids: number[];
}

export interface FileObjectConfig {
  access_key_id?: string;
  access_key_secret?: string;
  bucket?: string;
  domain?: string;
  endpoint?: string;
  region?: string;
  secret_set?: boolean;
  use_path_style?: boolean;
}

export interface FileRenameRequest {
  id: number;
  /** @maxLength 255 */
  name: string;
}

export interface FileRow {
  created_at: string;
  ext: string;
  group_id: number;
  height: number;
  id: number;
  name: string;
  path: string;
  size: number;
  storage: string;
  url: string;
  width: number;
}

export interface FileStorageConfig {
  cos: FileObjectConfig;
  default_storage: "local" | "oss" | "cos" | "s3";
  local: FileLocalConfig;
  oss: FileObjectConfig;
  s3: FileObjectConfig;
}

export interface LoginRequest {
  /** @maxLength 72 */
  password: string;
  remember?: boolean;
  /** @maxLength 64 */
  username: string;
}

export interface LoginResponse {
  expires_at?: string;
  token?: string;
}

export interface PermissionDefinition {
  code?: string;
  module?: string;
  name?: string;
}

export interface ProductCategoryCreateRequest {
  image_id?: number | null;
  /** @maxLength 255 */
  name: string;
  parent_id?: number;
  sort?: number;
  status?: number;
}

export interface ProductCategoryDeleteRequest {
  id: number;
}

export interface ProductCategoryOption {
  id?: number;
  name?: string;
  parent_id?: number;
  sort?: number;
  status?: number;
}

export interface ProductCategoryRow {
  created_at?: string;
  id?: number;
  image_id?: number | null;
  image_url?: string;
  name?: string;
  parent_id?: number;
  sort?: number;
  status?: number;
  updated_at?: string;
}

export interface ProductCategoryUpdateRequest {
  id: number;
  image_id?: number | null;
  name?: string;
  parent_id?: number;
  sort?: number;
  status?: number;
}

export interface ProductCreateRequest {
  category_id?: number;
  detail?: string;
  gallery_file_ids?: number[];
  main_image_id?: number;
  /** @maxLength 255 */
  name: string;
  params?: ProductParam[];
  skus: ProductSKURequest[];
  sort?: number;
  specs?: ProductSpec[];
  /** 0:下架, 1:上架 */
  status?: number;
  /** @maxLength 255 */
  subtitle?: string;
  virtual_sales?: number;
}

export interface ProductDetail {
  category_id?: number;
  category_name?: string;
  created_at?: string;
  detail?: string;
  gallery_file_ids?: number[];
  gallery_image_urls?: string[];
  id?: number;
  main_image_id?: number;
  main_image_url?: string;
  name?: string;
  params?: ProductParam[];
  sales_count?: number;
  skus?: ProductSKU[];
  sort?: number;
  specs?: ProductSpec[];
  status?: number;
  subtitle?: string;
  updated_at?: string;
  virtual_sales?: number;
}

export interface ProductIDRequest {
  id: number;
}

export interface ProductIDResponse {
  id?: number;
}

export interface ProductInventoryAdjustRequest {
  change_type: string;
  expected_version?: number;
  quantity?: number;
  remark?: string;
  request_id: string;
  sku_id: number;
}

export interface ProductInventoryDetailRequest {
  sku_id: number;
}

export interface ProductInventoryListRequest {
  keyword?: string;
  page?: number;
  page_size?: number;
  product_id?: number;
  product_status?: number;
  sku_id?: number;
  sku_status?: number;
  /** all, low, empty, normal, uninitialized */
  stock_filter?: string;
}

export interface ProductInventoryLogListRequest {
  change_type?: string;
  /** RFC3339 */
  created_from?: string;
  /** RFC3339 */
  created_to?: string;
  operator_id?: number;
  page?: number;
  page_size?: number;
  product_id?: number;
  request_id?: string;
  sku_id?: number;
}

export interface ProductInventoryLogRow {
  available_after?: number;
  available_before?: number;
  change_type?: string;
  created_at?: string;
  id?: number;
  inventory_id?: number;
  locked_after?: number;
  locked_before?: number;
  locked_delta?: number;
  operator_id?: number;
  operator_name?: string;
  product_id?: number;
  quantity?: number;
  remark?: string;
  request_id?: string;
  sku_id?: number;
  sku_snapshot?: ProductInventorySKUSnapshot;
  stock_after?: number;
  stock_before?: number;
  stock_delta?: number;
  version_after?: number;
  version_before?: number;
  warning_after?: number;
  warning_before?: number;
}

export interface ProductInventoryMutationResult {
  error_code?: string | null;
  log?: ProductInventoryLogRow | null;
  replayed?: boolean;
}

export interface ProductInventoryRow {
  available_stock?: number | null;
  barcode?: string | null;
  image_url?: string;
  initialized?: boolean;
  inventory_id?: number | null;
  is_low_stock?: boolean | null;
  is_out_of_stock?: boolean | null;
  locked_stock?: number | null;
  product_id?: number;
  product_name?: string;
  product_status?: number;
  sku_code?: string | null;
  sku_id?: number;
  sku_status?: number;
  specs?: ProductInventorySpecOption[];
  stock?: number | null;
  unit?: string | null;
  updated_at?: string | null;
  version?: number | null;
  warning_stock?: number | null;
}

export interface ProductInventorySKUSnapshot {
  product_name?: string;
  sku_code?: string | null;
  specs?: ProductInventorySpecOption[];
  unit?: string | null;
}

export interface ProductInventorySpecOption {
  spec_id?: number;
  spec_name?: string;
  value_id?: number;
  value_name?: string;
}

export interface ProductInventoryWarningRequest {
  expected_version?: number;
  quantity?: number;
  remark?: string;
  request_id: string;
  sku_id: number;
}

export interface ProductListRequest {
  category_id?: number;
  keyword?: string;
  page?: number;
  page_size?: number;
  status?: number;
}

export interface ProductParam {
  name?: string;
  value?: string;
}

export interface ProductRow {
  category_id?: number;
  category_name?: string;
  created_at?: string;
  id?: number;
  main_image_id?: number;
  main_image_url?: string;
  max_price?: string;
  min_price?: string;
  name?: string;
  sales_count?: number;
  sku_count?: number;
  sort?: number;
  status?: number;
  subtitle?: string;
  updated_at?: string;
  virtual_sales?: number;
}

export interface ProductSKU {
  barcode?: string;
  cost_price?: string;
  created_at?: string;
  id?: number;
  image_id?: number;
  image_url?: string;
  market_price?: string;
  price?: string;
  product_id?: number;
  sku_code?: string;
  sort?: number;
  spec_values?: ProductSKUValue[];
  status?: number;
  unit?: string;
  updated_at?: string;
  weight_gram?: number;
}

export interface ProductSKURequest {
  barcode?: string;
  /** e.g. "199.90" */
  cost_price?: string;
  /** 0 for new SKU, >0 for existing SKU */
  id?: number;
  image_id?: number;
  /** e.g. "199.90" */
  market_price?: string;
  /** e.g. "199.90" */
  price?: string;
  sku_code?: string;
  sort?: number;
  spec_values?: ProductSKUValue[];
  /** 0:禁用, 1:启用 */
  status?: number;
  unit?: string;
  weight_gram?: number;
}

export interface ProductSKUValue {
  spec_id?: number;
  value_id?: number;
}

export interface ProductSpec {
  id?: number;
  name?: string;
  values?: ProductSpecValue[];
}

export interface ProductSpecValue {
  id?: number;
  name?: string;
}

export interface ProductStatusRequest {
  id: number;
  /** 0:下架, 1:上架 */
  status?: number;
}

export interface ProductUpdateRequest {
  category_id?: number;
  detail?: string;
  gallery_file_ids?: number[];
  id: number;
  main_image_id?: number;
  /** @maxLength 255 */
  name: string;
  params?: ProductParam[];
  skus: ProductSKURequest[];
  sort?: number;
  specs?: ProductSpec[];
  status?: number;
  /** @maxLength 255 */
  subtitle?: string;
  virtual_sales?: number;
}

export interface ResponseCurrentUserResponse {
  code?: number;
  data?: CurrentUserResponse;
  message?: string;
  total?: number;
}

export interface ResponseFileGroupCounts {
  code?: number;
  data?: FileGroupCounts;
  message?: string;
  total?: number;
}

export interface ResponseFileRow {
  code?: number;
  data?: FileRow;
  message?: string;
  total?: number;
}

export interface ResponseFileStorageConfig {
  code?: number;
  data?: FileStorageConfig;
  message?: string;
  total?: number;
}

export interface ResponseLoginResponse {
  code?: number;
  data?: LoginResponse;
  message?: string;
  total?: number;
}

export interface ResponseProductCategoryRow {
  code?: number;
  data?: ProductCategoryRow;
  message?: string;
  total?: number;
}

export interface ResponseProductDetail {
  code?: number;
  data?: ProductDetail;
  message?: string;
  total?: number;
}

export interface ResponseProductIDResponse {
  code?: number;
  data?: ProductIDResponse;
  message?: string;
  total?: number;
}

export interface ResponseProductInventoryMutationResult {
  code?: number;
  data?: ProductInventoryMutationResult;
  message?: string;
  total?: number;
}

export interface ResponseProductInventoryRow {
  code?: number;
  data?: ProductInventoryRow;
  message?: string;
  total?: number;
}

export interface ResponseRoleRow {
  code?: number;
  data?: RoleRow;
  message?: string;
  total?: number;
}

export interface ResponseShopBasicConfig {
  code?: number;
  data?: ShopBasicConfig;
  message?: string;
  total?: number;
}

export interface ResponseSiteBasicConfig {
  code?: number;
  data?: SiteBasicConfig;
  message?: string;
  total?: number;
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

export interface ResponseArrayCurrentRole {
  code?: number;
  data?: CurrentRole[];
  message?: string;
  total?: number;
}

export interface ResponseArrayFileGroupRow {
  code?: number;
  data?: FileGroupRow[];
  message?: string;
  total?: number;
}

export interface ResponseArrayFileRow {
  code?: number;
  data?: FileRow[];
  message?: string;
  total?: number;
}

export interface ResponseArrayPermissionDefinition {
  code?: number;
  data?: PermissionDefinition[];
  message?: string;
  total?: number;
}

export interface ResponseArrayProductCategoryOption {
  code?: number;
  data?: ProductCategoryOption[];
  message?: string;
  total?: number;
}

export interface ResponseArrayProductCategoryRow {
  code?: number;
  data?: ProductCategoryRow[];
  message?: string;
  total?: number;
}

export interface ResponseArrayProductInventoryLogRow {
  code?: number;
  data?: ProductInventoryLogRow[];
  message?: string;
  total?: number;
}

export interface ResponseArrayProductInventoryRow {
  code?: number;
  data?: ProductInventoryRow[];
  message?: string;
  total?: number;
}

export interface ResponseArrayProductRow {
  code?: number;
  data?: ProductRow[];
  message?: string;
  total?: number;
}

export interface ResponseArrayRoleRow {
  code?: number;
  data?: RoleRow[];
  message?: string;
  total?: number;
}

export interface ResponseArrayUserRow {
  code?: number;
  data?: UserRow[];
  message?: string;
  total?: number;
}

export interface ResponseArrayString {
  code?: number;
  data?: string[];
  message?: string;
  total?: number;
}

export interface ResponseDemoDemoData {
  code?: number;
  data?: DemoDemoData;
  message?: string;
  total?: number;
}

export interface ResponseHealthHealthData {
  code?: number;
  data?: HealthHealthData;
  message?: string;
  total?: number;
}

export interface ResponseHelloHelloData {
  code?: number;
  data?: HelloHelloData;
  message?: string;
  total?: number;
}

export interface RoleCreateRequest {
  /** @maxLength 64 */
  name: string;
  /** @maxLength 255 */
  remark?: string;
  sort?: number;
  status?: number;
}

export interface RoleIDRequest {
  id: number;
}

export interface RoleListRequest {
  keyword?: string;
  page?: number;
  page_size?: number;
  status?: number;
}

export interface RolePermissionRequest {
  id: number;
}

export interface RoleRow {
  code?: string;
  created_at?: string;
  id?: number;
  name?: string;
  remark?: string;
  sort?: number;
  status?: number;
  updated_at?: string;
}

export interface RoleSavePermissionRequest {
  id: number;
  permissions?: string[];
}

export interface RoleUpdateRequest {
  id: number;
  name?: string;
  remark?: string;
  sort?: number;
  status?: number;
}

export interface ShopBasicConfig {
  /** 商城简介 */
  description?: string;
  /** 浏览器图标地址 */
  favicon?: string;
  /** 浏览器图标图片 ID */
  favicon_id?: number;
  /** 商城 Logo 地址 */
  logo?: string;
  /** 商城 Logo 图片 ID */
  logo_id?: number;
  /** 商城名称 */
  name?: string;
  /** 客服名称 */
  service_name?: string;
  /** 客服电话 */
  service_phone?: string;
  /** 客服微信 */
  service_wechat?: string;
  /** 客服微信二维码地址 */
  service_wechat_qrcode?: string;
  /** 客服微信二维码图片 ID */
  service_wechat_qrcode_id?: number;
}

export interface SiteBasicConfig {
  /** 版权信息 */
  copyright?: string;
  /** 网站描述 */
  description?: string;
  /** ICP备案号 */
  icp?: string;
  /** 网站关键词 */
  keywords?: string;
  /** 网站标题 */
  title?: string;
}

export interface UserRoleRequest {
  id: number;
}

export interface UserRow {
  created_at?: string;
  id?: number;
  nickname?: string;
  status?: number;
  updated_at?: string;
  username?: string;
}

export interface UserSaveRoleRequest {
  id: number;
  role_ids?: number[];
}

export interface DemoDemoData {
  /** @example "not_configured" */
  database?: string;
  /** @example "Mall API is running" */
  message?: string;
}

export interface HealthHealthData {
  /** @example "not_configured" */
  database?: string;
  /** @example "mall-admin-core" */
  service?: string;
  /** @example "ok" */
  status?: string;
}

export interface HelloHelloData {
  /** @example "Hello from plugin" */
  message?: string;
  /** @example "hello" */
  plugin?: string;
}

export interface UserCreateRequest {
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

export interface UserIDRequest {
  id: number;
}

export interface UserListRequest {
  page?: number;
  page_size?: number;
  username?: string;
}

export interface UserUpdateRequest {
  id: number;
  nickname?: string;
  password?: string;
  status?: number;
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

    const response = await this.instance.request({
      ...requestParams,
      headers: {
        ...(requestParams.headers || {}),
        ...(type ? { "Content-Type": type } : {}),
      },
      params: query,
      responseType: responseFormat,
      data: body,
      url: path,
    });
    return response.data;
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
     * @tags auth
     * @name Login
     * @summary Login to the admin API
     * @request POST:/api/auth/login
     */
    login: (request: LoginRequest, params: RequestParams = {}) =>
      this.request<ResponseLoginResponse, any>({
        path: `/api/auth/login`,
        method: "POST",
        body: request,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags auth
     * @name GetCurrentUser
     * @summary Get the current login user, roles, and permissions
     * @request GET:/api/auth/me
     * @secure
     */
    getCurrentUser: (params: RequestParams = {}) =>
      this.request<ResponseCurrentUserResponse, any>({
        path: `/api/auth/me`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

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
     * @tags product-category
     * @name CreateProductCategory
     * @summary Create a product category
     * @request POST:/api/product/categories/create
     * @secure
     */
    createProductCategory: (
      request: ProductCategoryCreateRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseProductCategoryRow, any>({
        path: `/api/product/categories/create`,
        method: "POST",
        body: request,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags product-category
     * @name DeleteProductCategory
     * @summary Delete a product category
     * @request POST:/api/product/categories/delete
     * @secure
     */
    deleteProductCategory: (
      request: ProductCategoryDeleteRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseAny, any>({
        path: `/api/product/categories/delete`,
        method: "POST",
        body: request,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags product-category
     * @name ListProductCategories
     * @summary List all product categories (flat array)
     * @request POST:/api/product/categories/list
     * @secure
     */
    listProductCategories: (params: RequestParams = {}) =>
      this.request<ResponseArrayProductCategoryRow, any>({
        path: `/api/product/categories/list`,
        method: "POST",
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags product-category
     * @name GetProductCategoryOptions
     * @summary Get lightweight product category options
     * @request POST:/api/product/categories/options
     * @secure
     */
    getProductCategoryOptions: (params: RequestParams = {}) =>
      this.request<ResponseArrayProductCategoryOption, any>({
        path: `/api/product/categories/options`,
        method: "POST",
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags product-category
     * @name UpdateProductCategory
     * @summary Update a product category
     * @request POST:/api/product/categories/update
     * @secure
     */
    updateProductCategory: (
      request: ProductCategoryUpdateRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseProductCategoryRow, any>({
        path: `/api/product/categories/update`,
        method: "POST",
        body: request,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description 支持人工入库、人工出库、盘点校正
     *
     * @tags 商品库存
     * @name AdjustProductInventory
     * @summary 调整商品库存
     * @request POST:/api/product/inventories/adjust
     */
    adjustProductInventory: (
      request: ProductInventoryAdjustRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseProductInventoryMutationResult, any>({
        path: `/api/product/inventories/adjust`,
        method: "POST",
        body: request,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description 打开库存调整框前获取SKU最新库存信息
     *
     * @tags 商品库存
     * @name GetProductInventory
     * @summary 获取单个SKU最新库存
     * @request POST:/api/product/inventories/detail
     */
    getProductInventory: (
      request: ProductInventoryDetailRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseAny, any>({
        path: `/api/product/inventories/detail`,
        method: "POST",
        body: request,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description 支持商品、SKU、状态及库存状态过滤
     *
     * @tags 商品库存
     * @name ListProductInventories
     * @summary 按SKU分页查询商品库存
     * @request POST:/api/product/inventories/list
     */
    listProductInventories: (
      request: ProductInventoryListRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseAny, any>({
        path: `/api/product/inventories/list`,
        method: "POST",
        body: request,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description 只读查询SKU库存与预警历史变更流水
     *
     * @tags 商品库存
     * @name ListProductInventoryLogs
     * @summary 查询库存变更流水
     * @request POST:/api/product/inventories/logs/list
     */
    listProductInventoryLogs: (
      request: ProductInventoryLogListRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseAny, any>({
        path: `/api/product/inventories/logs/list`,
        method: "POST",
        body: request,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description 设置SKU可用库存预警阈值，0表示关闭预警
     *
     * @tags 商品库存
     * @name SetProductInventoryWarning
     * @summary 设置库存预警阈值
     * @request POST:/api/product/inventories/warning
     */
    setProductInventoryWarning: (
      request: ProductInventoryWarningRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseProductInventoryMutationResult, any>({
        path: `/api/product/inventories/warning`,
        method: "POST",
        body: request,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags product
     * @name CreateProduct
     * @summary Create a product with skus
     * @request POST:/api/product/products/create
     * @secure
     */
    createProduct: (
      request: ProductCreateRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseProductIDResponse, any>({
        path: `/api/product/products/create`,
        method: "POST",
        body: request,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags product
     * @name DeleteProduct
     * @summary Soft delete a product and its skus
     * @request POST:/api/product/products/delete
     * @secure
     */
    deleteProduct: (request: ProductIDRequest, params: RequestParams = {}) =>
      this.request<ResponseAny, any>({
        path: `/api/product/products/delete`,
        method: "POST",
        body: request,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags product
     * @name GetProductDetail
     * @summary Get product detail with all SKUs
     * @request POST:/api/product/products/detail
     * @secure
     */
    getProductDetail: (request: ProductIDRequest, params: RequestParams = {}) =>
      this.request<ResponseProductDetail, any>({
        path: `/api/product/products/detail`,
        method: "POST",
        body: request,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags product
     * @name ListProducts
     * @summary List products with pagination and filters
     * @request POST:/api/product/products/list
     * @secure
     */
    listProducts: (request: ProductListRequest, params: RequestParams = {}) =>
      this.request<ResponseArrayProductRow, any>({
        path: `/api/product/products/list`,
        method: "POST",
        body: request,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags product
     * @name UpdateProductStatus
     * @summary Modify product online/offline status
     * @request POST:/api/product/products/status
     * @secure
     */
    updateProductStatus: (
      request: ProductStatusRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseAny, any>({
        path: `/api/product/products/status`,
        method: "POST",
        body: request,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags product
     * @name UpdateProduct
     * @summary Update a product and all its skus
     * @request POST:/api/product/products/update
     * @secure
     */
    updateProduct: (
      request: ProductUpdateRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseProductIDResponse, any>({
        path: `/api/product/products/update`,
        method: "POST",
        body: request,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags system-file
     * @name GetFileStorageConfig
     * @summary Get storage settings with secrets masked
     * @request GET:/api/system/files/config
     * @secure
     */
    getFileStorageConfig: (params: RequestParams = {}) =>
      this.request<ResponseFileStorageConfig, any>({
        path: `/api/system/files/config`,
        method: "GET",
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags system-file
     * @name SaveFileStorageConfig
     * @summary Save storage settings; empty secrets retain existing values
     * @request POST:/api/system/files/config/save
     * @secure
     */
    saveFileStorageConfig: (
      request: FileStorageConfig,
      params: RequestParams = {},
    ) =>
      this.request<ResponseAny, any>({
        path: `/api/system/files/config/save`,
        method: "POST",
        body: request,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags system-file
     * @name DeleteSystemFiles
     * @summary Archive images without breaking existing business file IDs
     * @request POST:/api/system/files/delete
     * @secure
     */
    deleteSystemFiles: (request: FileIDsRequest, params: RequestParams = {}) =>
      this.request<ResponseAny, any>({
        path: `/api/system/files/delete`,
        method: "POST",
        body: request,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags system-file
     * @name ListSystemFileGroups
     * @summary List image groups and counts
     * @request GET:/api/system/files/groups
     * @secure
     */
    listSystemFileGroups: (params: RequestParams = {}) =>
      this.request<ResponseArrayFileGroupRow, any>({
        path: `/api/system/files/groups`,
        method: "GET",
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags system-file
     * @name GetSystemFileGroupCounts
     * @summary Get image counts for all, ungrouped, and each group
     * @request GET:/api/system/files/groups/counts
     * @secure
     */
    getSystemFileGroupCounts: (params: RequestParams = {}) =>
      this.request<ResponseFileGroupCounts, any>({
        path: `/api/system/files/groups/counts`,
        method: "GET",
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags system-file
     * @name DeleteSystemFileGroup
     * @summary Delete a group and move its images to ungrouped
     * @request POST:/api/system/files/groups/delete
     * @secure
     */
    deleteSystemFileGroup: (
      request: FileGroupIDRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseAny, any>({
        path: `/api/system/files/groups/delete`,
        method: "POST",
        body: request,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags system-file
     * @name SaveSystemFileGroup
     * @summary Create or rename an image group
     * @request POST:/api/system/files/groups/save
     * @secure
     */
    saveSystemFileGroup: (
      request: FileGroupSaveRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseAny, any>({
        path: `/api/system/files/groups/save`,
        method: "POST",
        body: request,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags system-file
     * @name ListSystemFiles
     * @summary List and search images
     * @request POST:/api/system/files/list
     * @secure
     */
    listSystemFiles: (request: FileListRequest, params: RequestParams = {}) =>
      this.request<ResponseArrayFileRow, any>({
        path: `/api/system/files/list`,
        method: "POST",
        body: request,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags system-file
     * @name MoveSystemFiles
     * @summary Move images to a group
     * @request POST:/api/system/files/move
     * @secure
     */
    moveSystemFiles: (request: FileMoveRequest, params: RequestParams = {}) =>
      this.request<ResponseAny, any>({
        path: `/api/system/files/move`,
        method: "POST",
        body: request,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags system-file
     * @name RenameSystemFile
     * @summary Rename an image display name
     * @request POST:/api/system/files/rename
     * @secure
     */
    renameSystemFile: (
      request: FileRenameRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseAny, any>({
        path: `/api/system/files/rename`,
        method: "POST",
        body: request,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags system-file
     * @name ResolveSystemFiles
     * @summary Resolve business file IDs into current URLs (including archived images)
     * @request POST:/api/system/files/resolve
     * @secure
     */
    resolveSystemFiles: (request: FileIDsRequest, params: RequestParams = {}) =>
      this.request<ResponseArrayFileRow, any>({
        path: `/api/system/files/resolve`,
        method: "POST",
        body: request,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags system-file
     * @name UploadSystemFile
     * @summary Upload one image (10 MB maximum); call for each file in a batch
     * @request POST:/api/system/files/upload
     * @secure
     */
    uploadSystemFile: (
      data: {
        /**
         * Image file
         * @format binary
         */
        file: File;
        /** Group ID; zero means ungrouped */
        group_id?: number;
      },
      params: RequestParams = {},
    ) =>
      this.request<ResponseFileRow, any>({
        path: `/api/system/files/upload`,
        method: "POST",
        body: data,
        secure: true,
        type: ContentType.FormData,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags system-permission
     * @name ListAllPermissions
     * @summary List all permission definitions
     * @request GET:/api/system/permissions/list
     * @secure
     */
    listAllPermissions: (params: RequestParams = {}) =>
      this.request<ResponseArrayPermissionDefinition, any>({
        path: `/api/system/permissions/list`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags system-role
     * @name CreateSystemRole
     * @summary Create a system role
     * @request POST:/api/system/roles/create
     * @secure
     */
    createSystemRole: (
      request: RoleCreateRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseRoleRow, any>({
        path: `/api/system/roles/create`,
        method: "POST",
        body: request,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags system-role
     * @name DeleteSystemRole
     * @summary Delete a system role
     * @request POST:/api/system/roles/delete
     * @secure
     */
    deleteSystemRole: (request: RoleIDRequest, params: RequestParams = {}) =>
      this.request<ResponseAny, any>({
        path: `/api/system/roles/delete`,
        method: "POST",
        body: request,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags system-role
     * @name GetSystemRole
     * @summary Get a system role
     * @request POST:/api/system/roles/detail
     * @secure
     */
    getSystemRole: (request: RoleIDRequest, params: RequestParams = {}) =>
      this.request<ResponseRoleRow, any>({
        path: `/api/system/roles/detail`,
        method: "POST",
        body: request,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags system-role
     * @name ListSystemRoles
     * @summary List system roles
     * @request POST:/api/system/roles/list
     * @secure
     */
    listSystemRoles: (request: RoleListRequest, params: RequestParams = {}) =>
      this.request<ResponseArrayRoleRow, any>({
        path: `/api/system/roles/list`,
        method: "POST",
        body: request,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags system-role
     * @name GetSystemRolePermissions
     * @summary Get permissions assigned to a role
     * @request POST:/api/system/roles/permissions
     * @secure
     */
    getSystemRolePermissions: (
      request: RolePermissionRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseArrayString, any>({
        path: `/api/system/roles/permissions`,
        method: "POST",
        body: request,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags system-role
     * @name SaveSystemRolePermissions
     * @summary Replace all permissions assigned to a role
     * @request POST:/api/system/roles/permissions/save
     * @secure
     */
    saveSystemRolePermissions: (
      request: RoleSavePermissionRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseAny, any>({
        path: `/api/system/roles/permissions/save`,
        method: "POST",
        body: request,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags system-role
     * @name UpdateSystemRole
     * @summary Update a system role
     * @request POST:/api/system/roles/update
     * @secure
     */
    updateSystemRole: (
      request: RoleUpdateRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseRoleRow, any>({
        path: `/api/system/roles/update`,
        method: "POST",
        body: request,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags system-setting
     * @name GetSystemShopSetting
     * @summary 获取商城基础设置
     * @request GET:/api/system/settings/shop
     * @secure
     */
    getSystemShopSetting: (params: RequestParams = {}) =>
      this.request<ResponseShopBasicConfig, any>({
        path: `/api/system/settings/shop`,
        method: "GET",
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags system-setting
     * @name SaveSystemShopSetting
     * @summary 保存商城基础设置
     * @request POST:/api/system/settings/shop/save
     * @secure
     */
    saveSystemShopSetting: (
      request: ShopBasicConfig,
      params: RequestParams = {},
    ) =>
      this.request<ResponseAny, any>({
        path: `/api/system/settings/shop/save`,
        method: "POST",
        body: request,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags system-setting
     * @name GetSystemSiteSetting
     * @summary 获取站点信息设置
     * @request GET:/api/system/settings/site
     * @secure
     */
    getSystemSiteSetting: (params: RequestParams = {}) =>
      this.request<ResponseSiteBasicConfig, any>({
        path: `/api/system/settings/site`,
        method: "GET",
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags system-setting
     * @name SaveSystemSiteSetting
     * @summary 保存站点信息设置
     * @request POST:/api/system/settings/site/save
     * @secure
     */
    saveSystemSiteSetting: (
      request: SiteBasicConfig,
      params: RequestParams = {},
    ) =>
      this.request<ResponseAny, any>({
        path: `/api/system/settings/site/save`,
        method: "POST",
        body: request,
        secure: true,
        type: ContentType.Json,
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
     * @secure
     */
    createSystemUser: (
      request: UserCreateRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseUserRow, any>({
        path: `/api/system/users/create`,
        method: "POST",
        body: request,
        secure: true,
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
     * @secure
     */
    deleteSystemUser: (request: UserIDRequest, params: RequestParams = {}) =>
      this.request<ResponseAny, any>({
        path: `/api/system/users/delete`,
        method: "POST",
        body: request,
        secure: true,
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
     * @secure
     */
    getSystemUser: (request: UserIDRequest, params: RequestParams = {}) =>
      this.request<ResponseUserRow, any>({
        path: `/api/system/users/detail`,
        method: "POST",
        body: request,
        secure: true,
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
     * @secure
     */
    listSystemUsers: (request: UserListRequest, params: RequestParams = {}) =>
      this.request<ResponseArrayUserRow, any>({
        path: `/api/system/users/list`,
        method: "POST",
        body: request,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags system-user
     * @name GetSystemUserRoles
     * @summary Get roles assigned to a user
     * @request POST:/api/system/users/roles
     * @secure
     */
    getSystemUserRoles: (
      request: UserRoleRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseArrayCurrentRole, any>({
        path: `/api/system/users/roles`,
        method: "POST",
        body: request,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags system-user
     * @name SaveSystemUserRoles
     * @summary Replace all roles assigned to a user
     * @request POST:/api/system/users/roles/save
     * @secure
     */
    saveSystemUserRoles: (
      request: UserSaveRoleRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseAny, any>({
        path: `/api/system/users/roles/save`,
        method: "POST",
        body: request,
        secure: true,
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
     * @secure
     */
    updateSystemUser: (
      request: UserUpdateRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseUserRow, any>({
        path: `/api/system/users/update`,
        method: "POST",
        body: request,
        secure: true,
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
