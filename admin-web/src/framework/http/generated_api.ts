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

export interface AdminAfterSaleDetailRequest {
  id: number;
}

export interface AdminAfterSaleDetailResponse {
  after_sale_no?: string;
  applied_at?: string;
  audited_at?: string | null;
  buyer_return_company?: string;
  buyer_return_no?: string;
  buyer_returned_at?: string | null;
  cancelled_at?: string | null;
  completed_at?: string | null;
  created_at?: string;
  description?: string;
  id?: number;
  items?: AfterSaleItemRow[];
  logs?: AfterSaleLogRow[];
  member_id?: number;
  member_mobile?: string;
  member_nickname?: string;
  merchant_received_at?: string | null;
  order_id?: number;
  order_no?: string;
  order_status_at_apply?: number;
  order_status_at_apply_text?: string;
  proof_file_ids?: number[];
  proof_files?: AfterSaleProofFile[];
  proof_urls?: string[];
  reason?: string;
  refund?: OrderRefundRow | null;
  refund_amount?: string;
  refund_started_at?: string | null;
  reject_reason?: string;
  request_id?: string;
  return_address?: string;
  return_phone?: string;
  return_receiver?: string;
  status?: number;
  status_text?: string;
  type?: number;
  type_text?: string;
  updated_at?: string;
}

export interface AdminAfterSaleListRequest {
  after_sale_no?: string;
  end_time?: string;
  keyword?: string;
  order_no?: string;
  page?: number;
  page_size?: number;
  start_time?: string;
  status?: number;
  tab?: string;
  type?: number;
}

export interface AdminAfterSaleListRow {
  after_sale_no?: string;
  applied_at?: string;
  created_at?: string;
  id?: number;
  member_id?: number;
  member_mobile?: string;
  member_nickname?: string;
  order_id?: number;
  order_no?: string;
  reason?: string;
  refund_amount?: string;
  status?: number;
  status_text?: string;
  type?: number;
  type_text?: string;
}

export interface AdminAfterSaleTabCountsResponse {
  all?: number;
  cancelled?: number;
  completed?: number;
  pending_review?: number;
  refund_failed?: number;
  refunding?: number;
  rejected?: number;
  waiting_buyer_return?: number;
  waiting_merchant_receive?: number;
}

export interface AdminApproveRequest {
  id: number;
  return_address?: string;
  return_address_id?: number;
  return_phone?: string;
  return_receiver?: string;
}

export interface AdminConfirmReceiveItem {
  after_sale_item_id: number;
  restock_quantity?: number;
}

export interface AdminConfirmReceiveRequest {
  id: number;
  items: AdminConfirmReceiveItem[];
}

export interface AdminRejectRequest {
  id: number;
  reject_reason: string;
}

export interface AdminRetryRefundRequest {
  id: number;
}

export interface AdminSyncRefundRequest {
  id: number;
}

export interface AfterSaleItemRow {
  after_sale_id?: number;
  id?: number;
  order_item_id?: number;
  product_id?: number;
  product_name?: string;
  quantity?: number;
  refund_amount?: string;
  restock_quantity?: number;
  sku_code?: string;
  sku_id?: number;
  sku_image_url?: string;
  sku_text?: string;
}

export interface AfterSaleLogRow {
  action?: string;
  created_at?: string;
  from_status?: number;
  from_status_text?: string;
  id?: number;
  operator_id?: number;
  operator_name?: string;
  operator_type?: number;
  operator_type_text?: string;
  remark?: string;
  to_status?: number;
  to_status_text?: string;
}

export interface AfterSaleProofFile {
  cover_url?: string;
  duration_ms?: number;
  file_type?: string;
  height?: number;
  id?: number;
  mime_type?: string;
  original_name?: string;
  size?: number;
  url?: string;
  width?: number;
}

export interface AgreementBasicConfig {
  /** 售后服务政策富文本 */
  after_sale_policy?: string;
  /** 隐私政策富文本 */
  privacy_policy?: string;
  /** 用户服务协议富文本 */
  user_agreement?: string;
}

export interface AnalyticsChannelDistributionItem {
  /** @example "wechat" */
  channel?: string;
  /** @example "微信支付" */
  channel_name?: string;
  /** @example "14000.00" */
  pay_amount?: string;
  /** @example 70 */
  pay_count?: number;
  /** @example "87.50%" */
  percentage?: string;
}

export interface AnalyticsCustomerGroupAnalysis {
  /** @example 18 */
  new_customer_count?: number;
  /** @example 20 */
  new_customer_order_count?: number;
  /** @example "4500.00" */
  new_customer_pay_amount?: string;
  /** @example "56.25%" */
  new_customer_percentage?: string;
  /** @example 14 */
  old_customer_count?: number;
  /** @example 28 */
  old_customer_order_count?: number;
  /** @example "6700.00" */
  old_customer_pay_amount?: string;
  /** @example "43.75%" */
  old_customer_percentage?: string;
}

export interface AnalyticsFinanceFlowRow {
  /**
   * 正向十进制金额
   * @example "199.00"
   */
  amount?: string;
  /** @example "PAY202609170001" */
  biz_no?: string;
  /** @example "wechat" */
  channel?: string;
  created_at?: string;
  /**
   * income 或 expense
   * @example "income"
   */
  direction?: string;
  /**
   * payment 或 refund
   * @example "payment"
   */
  flow_type?: string;
  /** @example 1 */
  id?: number;
  occurred_at?: string;
  /** @example 100 */
  order_id?: number;
  /** @example "ORD202609170001" */
  order_no?: string;
  /** @example 20 */
  status?: number;
  /** @example "支付成功" */
  status_text?: string;
  /** @example "WX_TRANS_123456" */
  transaction_no?: string;
}

export interface AnalyticsFlowListRequest {
  /** @example "PAY202609170001" */
  biz_no?: string;
  /** @example "wechat" */
  channel?: string;
  /** @example "2026-09-17 23:59:59" */
  end_time?: string;
  /**
   * all, payment, refund
   * @example "all"
   */
  flow_type?: string;
  /** @example "ORD202609170001" */
  order_no?: string;
  /** @example 1 */
  page?: number;
  /** @example 10 */
  page_size?: number;
  /** @example "2026-09-01 00:00:00" */
  start_time?: string;
  /** @example 20 */
  status?: number;
}

export interface AnalyticsFlowListResponse {
  items?: AnalyticsFinanceFlowRow[];
  /** @example "17300.00" */
  net_flow_amount?: string;
  /** @example 150 */
  total?: number;
  /** @example "1200.00" */
  total_expense_amount?: string;
  /** @example "18500.00" */
  total_income_amount?: string;
}

export interface AnalyticsMemberMetric {
  /** @example "1.50" */
  average_member_order_count?: string;
  /** @example "350.00" */
  average_member_spending?: string;
  /** @example 45 */
  new_member_count?: number;
  /** @example 18 */
  new_paid_member_count?: number;
  /** @example 32 */
  paid_member_count?: number;
  /** @example 520 */
  total_member_count?: number;
}

export interface AnalyticsMemberRankingItem {
  /** @example "https://example.com/a.jpg" */
  avatar_url?: string;
  last_paid_at?: string;
  /** @example 1 */
  member_id?: number;
  /** @example "13800138000" */
  mobile?: string;
  /** @example "张三" */
  nickname?: string;
  /** @example 5 */
  paid_order_count?: number;
  /** @example "1280.00" */
  total_pay_amount?: string;
}

export interface AnalyticsMemberResponse {
  customer_segment?: AnalyticsCustomerGroupAnalysis;
  metrics?: AnalyticsMemberMetric;
  top_members?: AnalyticsMemberRankingItem[];
  trends?: AnalyticsMemberTrendItem[];
}

export interface AnalyticsMemberTrendItem {
  /** @example "2026-09-10" */
  date?: string;
  /** @example 6 */
  new_member_count?: number;
  /** @example 8 */
  paid_member_count?: number;
}

export interface AnalyticsOrderStatusDistributionItem {
  /** @example 25 */
  count?: number;
  /** @example "25.00%" */
  percentage?: string;
  /** @example 20 */
  status?: number;
  /** @example "待发货" */
  status_text?: string;
}

export interface AnalyticsOrderTrendItem {
  /** @example "2026-09-10" */
  date?: string;
  /**
   * 下单订单数
   * @example 20
   */
  order_count?: number;
  /**
   * 支付订单数
   * @example 16
   */
  paid_order_count?: number;
}

export interface AnalyticsOverviewMetric {
  /** @example "+3.85%" */
  aov_growth_rate?: string;
  /** @example "200.00" */
  average_order_value?: string;
  /** @example "12150.00" */
  net_sales_amount?: string;
  /** @example "+14.20%" */
  net_sales_growth_rate?: string;
  /** @example 52 */
  paid_member_count?: number;
  /** @example "+6.12%" */
  paid_member_count_growth_rate?: string;
  /** @example 64 */
  paid_order_count?: number;
  /** @example "+8.33%" */
  paid_order_count_growth_rate?: string;
  /** @example "12800.00" */
  pay_amount?: string;
  /** @example "+12.50%" */
  pay_amount_growth_rate?: string;
  /** @example "650.00" */
  refund_amount?: string;
  /** @example "-5.00%" */
  refund_amount_growth_rate?: string;
}

export interface AnalyticsOverviewResponse {
  metrics?: AnalyticsOverviewMetric;
  order_trends?: AnalyticsOrderTrendItem[];
  recent_flows?: AnalyticsRecentFlowItem[];
  sales_trends?: AnalyticsSalesTrendItem[];
  top_products?: AnalyticsTopProductItem[];
}

export interface AnalyticsProductAnalysisRow {
  /** @example 75 */
  buyer_count?: number;
  /** @example "https://example.com/p1.jpg" */
  image_url?: string;
  /** @example 80 */
  paid_order_count?: number;
  /** @example 1 */
  product_id?: number;
  /** @example "纯棉圆领T恤" */
  product_name?: string;
  /** @example "396.00" */
  refund_amount?: string;
  /** @example 4 */
  refund_quantity?: number;
  /** @example "3.33%" */
  refund_rate?: string;
  /** @example "11880.00" */
  sales_amount?: string;
  /** @example 120 */
  sales_quantity?: number;
  /** @example "TSHIRT-BLK-L" */
  sku_code?: string;
  /** @example 10 */
  sku_id?: number;
  /** @example "黑色 / L" */
  sku_text?: string;
}

export interface AnalyticsProductListRequest {
  /**
   * "product" 或 "sku"，默认 "product"
   * @example "product"
   */
  dimension?: string;
  /** @example "2026-09-17 23:59:59" */
  end_time?: string;
  /** @example "T恤" */
  keyword?: string;
  /** @example 1 */
  page?: number;
  /** @example 10 */
  page_size?: number;
  /** @example 1 */
  product_id?: number;
  /** @example "TSHIRT-BLK-L" */
  sku_code?: string;
  /**
   * sales_amount, sales_quantity, refund_amount, refund_quantity, paid_order_count
   * @example "sales_amount"
   */
  sort_by?: string;
  /**
   * desc, asc
   * @example "desc"
   */
  sort_order?: string;
  /** @example "2026-09-01 00:00:00" */
  start_time?: string;
  /** @example "last_7_days" */
  time_range?: string;
}

export interface AnalyticsProductRankingItem {
  /** @example "https://example.com/p1.jpg" */
  image_url?: string;
  /** @example 1 */
  product_id?: number;
  /** @example "纯棉圆领T恤" */
  product_name?: string;
  /** @example "495.00" */
  refund_amount?: string;
  /** @example 5 */
  refund_quantity?: number;
  /** @example "3.33%" */
  refund_rate?: string;
  /** @example "14850.00" */
  sales_amount?: string;
  /** @example 150 */
  sales_quantity?: number;
}

export interface AnalyticsProductRankingsResponse {
  /** 退款排行 TOP 10 */
  refund_rankings?: AnalyticsProductRankingItem[];
  /** 销售额排行 TOP 10 */
  sales_amount_rankings?: AnalyticsProductRankingItem[];
  /** 销量排行 TOP 10 */
  sales_quantity_rankings?: AnalyticsProductRankingItem[];
}

export interface AnalyticsRecentFlowItem {
  /** @example "199.00" */
  amount?: string;
  /** @example "PAY202609170001" */
  biz_no?: string;
  /** @example "wechat" */
  channel?: string;
  /**
   * income 或 expense
   * @example "income"
   */
  direction?: string;
  /**
   * payment 或 refund
   * @example "payment"
   */
  flow_type?: string;
  /** @example 101 */
  id?: number;
  occurred_at?: string;
  /** @example 1 */
  order_id?: number;
  /** @example "ORD202609170001" */
  order_no?: string;
  /** @example 20 */
  status?: number;
  /** @example "支付成功" */
  status_text?: string;
  /** @example "WX_TRANS_123456" */
  transaction_no?: string;
}

export interface AnalyticsSalesTrendItem {
  /** @example "2026-09-10" */
  date?: string;
  /** @example "1050.00" */
  net_sales_amount?: string;
  /** @example "1200.00" */
  pay_amount?: string;
  /** @example "150.00" */
  refund_amount?: string;
}

export interface AnalyticsTopProductItem {
  /** @example "https://example.com/p1.png" */
  image_url?: string;
  /** @example 85 */
  paid_order_count?: number;
  /** @example 1 */
  product_id?: number;
  /** @example "经典纯棉T恤" */
  product_name?: string;
  /** @example 120 */
  quantity?: number;
  /** @example "11880.00" */
  sales_amount?: string;
}

export interface AnalyticsTradeAmountTrendItem {
  /** @example "2026-09-10" */
  date?: string;
  /** @example "2200.00" */
  net_sales_amount?: string;
  /** @example "2400.00" */
  pay_amount?: string;
  /** @example "200.00" */
  refund_amount?: string;
}

export interface AnalyticsTradeMetric {
  /**
   * 客单价
   * @example "200.00"
   */
  average_order_value?: string;
  /**
   * 取消订单数
   * @example 15
   */
  cancelled_order_count?: number;
  /**
   * 优惠金额
   * @example "800.00"
   */
  discount_amount?: string;
  /**
   * 运费金额
   * @example "400.00"
   */
  freight_amount?: string;
  /**
   * 下单订单数
   * @example 100
   */
  order_count?: number;
  /**
   * 支付订单数
   * @example 80
   */
  paid_order_count?: number;
  /**
   * 支付金额
   * @example "16000.00"
   */
  pay_amount?: string;
}

export interface AnalyticsTradeOrderTrendItem {
  /** @example 2 */
  cancelled_order_count?: number;
  /** @example "2026-09-10" */
  date?: string;
  /** @example 15 */
  order_count?: number;
  /** @example 12 */
  paid_order_count?: number;
}

export interface AnalyticsTradeResponse {
  amount_trends?: AnalyticsTradeAmountTrendItem[];
  channel_distributions?: AnalyticsChannelDistributionItem[];
  metrics?: AnalyticsTradeMetric;
  order_status_distributions?: AnalyticsOrderStatusDistributionItem[];
  order_trends?: AnalyticsTradeOrderTrendItem[];
}

export interface ChangePasswordRequest {
  /** @maxLength 72 */
  current_password: string;
  /**
   * @minLength 6
   * @maxLength 72
   */
  new_password: string;
}

export interface Config {
  api_v3_key?: string;
  mch_id?: string;
  merchant_private_key?: string;
  merchant_serial_no?: string;
  notify_base_url?: string;
  public_key?: string;
  public_key_id?: string;
}

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

export interface CreateReturnAddressRequest {
  address: string;
  city_code?: string;
  district_code?: string;
  is_default?: number;
  name: string;
  phone: string;
  province_code?: string;
  receiver: string;
  sort?: number;
  status?: number;
}

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
  must_change_password?: boolean;
  permissions?: string[];
  roles?: CurrentRole[];
  user?: CurrentUser;
}

export interface DashboardOverviewResponse {
  /**
   * 低库存SKU数
   * @example 3
   */
  low_stock_sku_count?: number;
  /**
   * 会员总数
   * @example 500
   */
  member_count?: number;
  /**
   * 在售商品数
   * @example 80
   */
  on_sale_product_count?: number;
  /**
   * 待处理售后数（待审核/待收货/退款失败）
   * @example 2
   */
  pending_aftersale_count?: number;
  /**
   * 待发货订单数（已支付且未发货）
   * @example 8
   */
  pending_delivery_count?: number;
  /**
   * 待支付订单数
   * @example 5
   */
  pending_payment_count?: number;
  /**
   * 商品总数
   * @example 100
   */
  product_count?: number;
  /**
   * 今日新增会员数
   * @example 6
   */
  today_new_member_count?: number;
  /** @example 12 */
  today_order_count?: number;
  /** @example 10 */
  today_paid_order_count?: number;
  /**
   * 今日实际支付销售额（单位：元，十进制字符串保留精度）
   * @example "1280.00"
   */
  today_sales_amount?: string;
}

export interface DashboardTrendResponse {
  /**
   * 连续自然日列表
   * @example ["2026-09-08","2026-09-09"]
   */
  dates?: string[];
  /**
   * 每日订单量
   * @example [12,18]
   */
  order_counts?: number[];
  /**
   * 每日销售额（单位：元，十进制字符串保留精度）
   * @example ["1200.00","1850.00"]
   */
  sales_amounts?: string[];
}

export interface DemoData {
  /** @example "not_configured" */
  database?: string;
  /** @example "Mall API is running" */
  message?: string;
}

export interface DictServiceCreateRequest {
  description?: string;
  name: string;
  sort?: number;
  /** 0: 禁用, 1: 启用, 默认为 1 */
  status?: number;
}

export interface DictServiceUpdateRequest {
  description?: string;
  name: string;
  sort?: number;
  /** 0: 禁用, 1: 启用 */
  status?: number;
}

export interface DictUnitCreateRequest {
  description?: string;
  name: string;
  sort?: number;
  /** 0: 禁用, 1: 启用, 默认为 1 */
  status?: number;
}

export interface DictUnitUpdateRequest {
  description?: string;
  name: string;
  sort?: number;
  /** 0: 禁用, 1: 启用 */
  status?: number;
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
  /** @maxLength 32 */
  file_type?: string;
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
  cover_file_id?: number;
  cover_url?: string;
  created_at: string;
  duration_ms?: number;
  ext: string;
  file_type?: string;
  group_id: number;
  height: number;
  id: number;
  mime_type?: string;
  name: string;
  original_name?: string;
  owner_id?: number;
  owner_type?: string;
  path: string;
  scene?: string;
  size: number;
  storage: string;
  url: string;
  visibility?: number;
  width: number;
}

export interface FileStorageConfig {
  cos: FileObjectConfig;
  default_storage: "local" | "oss" | "cos" | "s3";
  local: FileLocalConfig;
  oss: FileObjectConfig;
  s3: FileObjectConfig;
}

export interface FreightExtraFeeRule {
  city_code?: string;
  fee?: string;
  province_code?: string;
}

export interface FreightNoDeliveryRule {
  city_code?: string;
  province_code?: string;
}

export interface FreightTemplateCreateRequest {
  base_fee: string;
  extra_fees?: FreightExtraFeeRule[];
  /** @maxLength 100 */
  name: string;
  no_delivery?: FreightNoDeliveryRule[];
  /** 0:禁用, 1:启用, 默认1 */
  status?: number;
}

export interface FreightTemplateDetail {
  base_fee?: string;
  created_at?: string;
  extra_fees?: FreightExtraFeeRule[];
  id?: number;
  name?: string;
  no_delivery?: FreightNoDeliveryRule[];
  status?: number;
  updated_at?: string;
}

export interface FreightTemplateIDRequest {
  id: number;
}

export interface FreightTemplateListRequest {
  keyword?: string;
  page?: number;
  page_size?: number;
  status?: number;
}

export interface FreightTemplateOption {
  base_fee?: string;
  extra_fees?: FreightExtraFeeRule[];
  id?: number;
  name?: string;
  no_delivery?: FreightNoDeliveryRule[];
  status?: number;
}

export interface FreightTemplateRow {
  base_fee?: string;
  created_at?: string;
  extra_fee_count?: number;
  id?: number;
  name?: string;
  no_delivery_count?: number;
  status?: number;
  updated_at?: string;
}

export interface FreightTemplateStatusRequest {
  id: number;
  /** 0:禁用, 1:启用 */
  status?: number;
}

export interface FreightTemplateUpdateRequest {
  base_fee: string;
  extra_fees?: FreightExtraFeeRule[];
  id: number;
  /** @maxLength 100 */
  name: string;
  no_delivery?: FreightNoDeliveryRule[];
  status?: number;
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

export interface HotKeywordItem {
  /** 是否启用 */
  enabled?: boolean;
  /** 搜索关键词 (最多50字符) */
  keyword?: string;
  /** 排序权重 (数字越小越靠前) */
  sort?: number;
}

export interface IDRequest {
  id: number;
}

export interface ListRequest {
  page?: number;
  page_size?: number;
  username?: string;
}

export interface ListReturnAddressesRequest {
  search?: string;
  status?: number;
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

export interface LogisticsBasicConfig {
  /** 默认发货详细地址 */
  sender_address?: string;
  /** 默认发货人姓名 */
  sender_name?: string;
  /** 默认发货人电话 */
  sender_phone?: string;
}

export interface MemberAddressRow {
  address?: string;
  city?: string;
  city_code?: string;
  created_at?: string;
  district?: string;
  district_code?: string;
  id?: number;
  is_default?: number;
  member_id?: number;
  mobile?: string;
  name?: string;
  postal_code?: string;
  province?: string;
  province_code?: string;
}

export interface MemberAuthRow {
  app_id?: string;
  auth_type?: string;
  created_at?: string;
  openid?: string;
  provider_data?: string;
  unionid?: string;
}

export interface MemberBasicConfig {
  /** 是否开启账号密码登录 */
  password_login_enabled?: boolean;
  /** 是否允许新用户注册 */
  register_enabled?: boolean;
  /** 是否开启微信小程序快捷登录 */
  wechat_login_enabled?: boolean;
}

export interface MemberDetailResponse {
  addresses?: MemberAddressRow[];
  auth?: MemberAuthRow[];
  member?: MemberRow;
}

export interface MemberIDRequest {
  id: number;
}

export interface MemberListRequest {
  keyword?: string;
  page?: number;
  page_size?: number;
  status?: number;
}

export interface MemberRow {
  avatar_id?: number;
  avatar_url?: string;
  created_at?: string;
  id?: number;
  last_login_at?: string;
  mobile?: string;
  nickname?: string;
  status?: number;
}

export interface MpConfig {
  /** AppID */
  app_id?: string;
  /** AppSecret */
  app_secret?: string;
  /** 业务域名 */
  business_domain?: string;
  /** downloadFile合法域名 */
  download_domain?: string;
  /** 小程序名称 */
  name?: string;
  /** 原始ID */
  original_id?: string;
  /** 小程序码图片地址 */
  qrcode?: string;
  /** 小程序码图片 ID */
  qrcode_id?: number;
  /** request合法域名 */
  request_domain?: string;
  /** socket合法域名 */
  socket_domain?: string;
  /** tcp合法域名 */
  tcp_domain?: string;
  /** udp合法域名 */
  udp_domain?: string;
  /** uploadFile合法域名 */
  upload_domain?: string;
  /** 代码上传密钥 */
  upload_private_key?: string;
}

export interface OrderAddressRow {
  address?: string;
  city?: string;
  city_code?: string;
  district?: string;
  district_code?: string;
  mobile?: string;
  name?: string;
  postal_code?: string | null;
  province?: string;
  province_code?: string;
}

export interface OrderDetailRequest {
  id: number;
}

export interface OrderDetailResponse {
  address?: OrderAddressRow | null;
  cancelled_at?: string | null;
  completed_at?: string | null;
  created_at?: string;
  discount_amount?: string;
  express_company?: string | null;
  express_no?: string | null;
  freight_amount?: string;
  id?: number;
  items?: OrderItemRow[];
  logs?: OrderLogRow[];
  member_id?: number;
  member_mobile?: string;
  member_nickname?: string;
  order_no?: string;
  paid_at?: string | null;
  pay_amount?: string;
  remark?: string;
  shipped_at?: string | null;
  status?: number;
  status_text?: string;
  total_amount?: string;
  updated_at?: string;
}

export interface OrderItemRow {
  id?: number;
  image_url?: string;
  product_id?: number;
  product_name?: string;
  quantity?: number;
  sku_code?: string;
  sku_id?: number;
  sku_spec_values?: string | null;
  total_amount?: string;
  unit_price?: string;
}

export interface OrderListRequest {
  end_time?: string;
  keyword?: string;
  member_id?: number;
  page?: number;
  page_size?: number;
  start_time?: string;
  status?: number;
  tab?: string;
}

export interface OrderListRow {
  address?: OrderAddressRow | null;
  cancelled_at?: string | null;
  completed_at?: string | null;
  created_at?: string;
  express_company?: string | null;
  express_no?: string | null;
  freight_amount?: string;
  id?: number;
  items?: OrderItemRow[];
  member_id?: number;
  member_mobile?: string;
  member_nickname?: string;
  order_no?: string;
  paid_at?: string | null;
  pay_amount?: string;
  shipped_at?: string | null;
  status?: number;
  status_text?: string;
  total_amount?: string;
  total_quantity?: number;
}

export interface OrderLogRow {
  action?: string;
  created_at?: string;
  from_status?: number;
  from_status_text?: string;
  id?: number;
  operator_name?: string;
  operator_type?: string;
  remark?: string;
  to_status?: number;
  to_status_text?: string;
}

export interface OrderRefundRow {
  id?: number;
  last_error?: string;
  provider?: string;
  provider_refund_no?: string | null;
  refund_amount?: string;
  refund_no?: string;
  refunded_at?: string | null;
  requested_at?: string | null;
  retry_count?: number;
  status?: number;
  status_text?: string;
}

export interface OrderShipRequest {
  /** @maxLength 64 */
  express_company: string;
  /** @maxLength 64 */
  express_no: string;
  id: number;
}

export interface OrderTabCountsResponse {
  all?: number;
  cancelled?: number;
  completed?: number;
  pending_payment?: number;
  pending_receipt?: number;
  pending_shipment?: number;
  refunded?: number;
}

export interface PaymentBasicConfig {
  /** 是否开启微信支付 */
  wechat_enabled?: boolean;
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
  /** 固定运费金额 */
  freight_fee?: string;
  /** 运费模板ID */
  freight_template_id?: number;
  /** 1:包邮, 2:固定运费, 3:运费模板 */
  freight_type?: number;
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
  freight_fee?: string;
  freight_template_id?: number;
  freight_template_name?: string;
  freight_type?: number;
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

export interface ProductDictServiceRow {
  created_at?: string;
  description?: string;
  id?: number;
  name?: string;
  sort?: number;
  /** 0: 禁用, 1: 启用 */
  status?: number;
  updated_at?: string;
}

export interface ProductDictUnitRow {
  created_at?: string;
  description?: string;
  id?: number;
  name?: string;
  sort?: number;
  /** 0: 禁用, 1: 启用 */
  status?: number;
  updated_at?: string;
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
  /** 创建时间结束 */
  created_at_end?: string;
  /** 创建时间开始 */
  created_at_start?: string;
  /** 运费类型：1包邮，2固定运费，3运费模板 */
  freight_type?: number;
  keyword?: string;
  /** 最高价格（元） */
  max_price?: string;
  /** 最高销量 */
  max_sales?: number;
  /** 最高库存 */
  max_stock?: number;
  /** 最低价格（元） */
  min_price?: string;
  /** 最低销量 */
  min_sales?: number;
  /** 最低库存 */
  min_stock?: number;
  page?: number;
  page_size?: number;
  /** 商品编码/ID */
  product_code?: string;
  /** SKU编码/条形码 */
  sku_code?: string;
  status?: number;
  /** all, selling, warning, sold_out, warehouse */
  tab?: string;
  /** 更新时间结束 */
  updated_at_end?: string;
  /** 更新时间开始 */
  updated_at_start?: string;
}

export interface ProductParam {
  name?: string;
  value?: string;
}

export interface ProductRow {
  category_id?: number;
  category_name?: string;
  created_at?: string;
  freight_fee?: string;
  freight_template_id?: number;
  freight_type?: number;
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

export interface ProductTabCountsResponse {
  all?: number;
  selling?: number;
  sold_out?: number;
  warehouse?: number;
  warning?: number;
}

export interface ProductUpdateRequest {
  category_id?: number;
  detail?: string;
  freight_fee?: string;
  freight_template_id?: number;
  freight_type?: number;
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

export interface PublicConfig {
  api_v3_key_configured?: boolean;
  mch_id?: string;
  merchant_private_key_configured?: boolean;
  merchant_serial_no?: string;
  notify_base_url?: string;
  public_key_configured?: boolean;
  public_key_id?: string;
}

export interface RegionNode {
  children?: RegionNode[];
  code?: string;
  name?: string;
}

export interface ResponseAdminAfterSaleDetailResponse {
  code?: number;
  data?: AdminAfterSaleDetailResponse;
  message?: string;
  total?: number;
}

export interface ResponseAdminAfterSaleTabCountsResponse {
  code?: number;
  data?: AdminAfterSaleTabCountsResponse;
  message?: string;
  total?: number;
}

export interface ResponseAgreementBasicConfig {
  code?: number;
  data?: AgreementBasicConfig;
  message?: string;
  total?: number;
}

export interface ResponseAnalyticsFlowListResponse {
  code?: number;
  data?: AnalyticsFlowListResponse;
  message?: string;
  total?: number;
}

export interface ResponseAnalyticsMemberResponse {
  code?: number;
  data?: AnalyticsMemberResponse;
  message?: string;
  total?: number;
}

export interface ResponseAnalyticsOverviewResponse {
  code?: number;
  data?: AnalyticsOverviewResponse;
  message?: string;
  total?: number;
}

export interface ResponseAnalyticsProductRankingsResponse {
  code?: number;
  data?: AnalyticsProductRankingsResponse;
  message?: string;
  total?: number;
}

export interface ResponseAnalyticsTradeResponse {
  code?: number;
  data?: AnalyticsTradeResponse;
  message?: string;
  total?: number;
}

export interface ResponseCurrentUserResponse {
  code?: number;
  data?: CurrentUserResponse;
  message?: string;
  total?: number;
}

export interface ResponseDashboardOverviewResponse {
  code?: number;
  data?: DashboardOverviewResponse;
  message?: string;
  total?: number;
}

export interface ResponseDashboardTrendResponse {
  code?: number;
  data?: DashboardTrendResponse;
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

export interface ResponseFreightTemplateDetail {
  code?: number;
  data?: FreightTemplateDetail;
  message?: string;
  total?: number;
}

export interface ResponseFreightTemplateIDRequest {
  code?: number;
  data?: FreightTemplateIDRequest;
  message?: string;
  total?: number;
}

export interface ResponseLoginResponse {
  code?: number;
  data?: LoginResponse;
  message?: string;
  total?: number;
}

export interface ResponseLogisticsBasicConfig {
  code?: number;
  data?: LogisticsBasicConfig;
  message?: string;
  total?: number;
}

export interface ResponseMemberBasicConfig {
  code?: number;
  data?: MemberBasicConfig;
  message?: string;
  total?: number;
}

export interface ResponseMemberDetailResponse {
  code?: number;
  data?: MemberDetailResponse;
  message?: string;
  total?: number;
}

export interface ResponseMpConfig {
  code?: number;
  data?: MpConfig;
  message?: string;
  total?: number;
}

export interface ResponseOrderDetailResponse {
  code?: number;
  data?: OrderDetailResponse;
  message?: string;
  total?: number;
}

export interface ResponseOrderTabCountsResponse {
  code?: number;
  data?: OrderTabCountsResponse;
  message?: string;
  total?: number;
}

export interface ResponsePaymentBasicConfig {
  code?: number;
  data?: PaymentBasicConfig;
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

export interface ResponseProductDictServiceRow {
  code?: number;
  data?: ProductDictServiceRow;
  message?: string;
  total?: number;
}

export interface ResponseProductDictUnitRow {
  code?: number;
  data?: ProductDictUnitRow;
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

export interface ResponseProductTabCountsResponse {
  code?: number;
  data?: ProductTabCountsResponse;
  message?: string;
  total?: number;
}

export interface ResponseReturnAddressItem {
  code?: number;
  data?: ReturnAddressItem;
  message?: string;
  total?: number;
}

export interface ResponseReviewBasicConfig {
  code?: number;
  data?: ReviewBasicConfig;
  message?: string;
  total?: number;
}

export interface ResponseReviewDetailResponse {
  code?: number;
  data?: ReviewDetailResponse;
  message?: string;
  total?: number;
}

export interface ResponseRoleRow {
  code?: number;
  data?: RoleRow;
  message?: string;
  total?: number;
}

export interface ResponseSearchBasicConfig {
  code?: number;
  data?: SearchBasicConfig;
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

export interface ResponseSystemJobRetryAccepted {
  code?: number;
  data?: SystemJobRetryAccepted;
  message?: string;
  total?: number;
}

export interface ResponseTradeBasicConfig {
  code?: number;
  data?: TradeBasicConfig;
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

export interface ResponseArrayAdminAfterSaleListRow {
  code?: number;
  data?: AdminAfterSaleListRow[];
  message?: string;
  total?: number;
}

export interface ResponseArrayAnalyticsProductAnalysisRow {
  code?: number;
  data?: AnalyticsProductAnalysisRow[];
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

export interface ResponseArrayFreightTemplateOption {
  code?: number;
  data?: FreightTemplateOption[];
  message?: string;
  total?: number;
}

export interface ResponseArrayFreightTemplateRow {
  code?: number;
  data?: FreightTemplateRow[];
  message?: string;
  total?: number;
}

export interface ResponseArrayMemberRow {
  code?: number;
  data?: MemberRow[];
  message?: string;
  total?: number;
}

export interface ResponseArrayOrderListRow {
  code?: number;
  data?: OrderListRow[];
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

export interface ResponseArrayProductDictServiceRow {
  code?: number;
  data?: ProductDictServiceRow[];
  message?: string;
  total?: number;
}

export interface ResponseArrayProductDictUnitRow {
  code?: number;
  data?: ProductDictUnitRow[];
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

export interface ResponseArrayReturnAddressItem {
  code?: number;
  data?: ReturnAddressItem[];
  message?: string;
  total?: number;
}

export interface ResponseArrayReviewListItemResponse {
  code?: number;
  data?: ReviewListItemResponse[];
  message?: string;
  total?: number;
}

export interface ResponseArrayRoleRow {
  code?: number;
  data?: RoleRow[];
  message?: string;
  total?: number;
}

export interface ResponseArraySystemJobStatus {
  code?: number;
  data?: SystemJobStatus[];
  message?: string;
  total?: number;
}

export interface ResponseArrayUserRow {
  code?: number;
  data?: UserRow[];
  message?: string;
  total?: number;
}

export interface ResponseArrayRegionRegionNode {
  code?: number;
  data?: RegionNode[];
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

export interface ResponseInt64 {
  code?: number;
  data?: number;
  message?: string;
  total?: number;
}

export interface ResponseString {
  code?: number;
  data?: string;
  message?: string;
  total?: number;
}

export interface ResponseVersionVersionData {
  code?: number;
  data?: VersionData;
  message?: string;
  total?: number;
}

export interface ResponseWechatpayPublicConfig {
  code?: number;
  data?: PublicConfig;
  message?: string;
  total?: number;
}

export interface ReturnAddressIDRequest {
  id: number;
}

export interface ReturnAddressItem {
  address?: string;
  city_code?: string;
  created_at?: string;
  district_code?: string;
  id?: number;
  is_default?: number;
  name?: string;
  phone?: string;
  province_code?: string;
  receiver?: string;
  sort?: number;
  status?: number;
  updated_at?: string;
}

export interface ReviewAuditRequest {
  /** "approve", "reject", "hide", "restore" */
  action: string;
  /** 期望旧状态（乐观锁防并发覆盖） */
  expected_status: number;
  id: number;
  /** 审核原因/驳回原因 */
  reason?: string;
}

export interface ReviewBasicConfig {
  /** 自动好评天数（天：1-30） */
  auto_review_days?: number;
  /** 是否开启超时自动好评 */
  auto_review_enabled?: boolean;
  /** 评价是否默认匿名 */
  default_anonymous?: boolean;
  /** 是否开启商品评价 */
  enabled?: boolean;
  /** 评价是否需要审核 */
  need_audit?: boolean;
}

export interface ReviewDetailRequest {
  id: number;
}

export interface ReviewDetailResponse {
  audit_reason?: string;
  audit_user_name?: string;
  audited_at?: string | null;
  content?: string;
  created_at?: string;
  has_image?: number;
  id?: number;
  images?: FileRow[];
  is_anonymous?: number;
  logs?: ReviewLogResponse[];
  member_avatar_url?: string;
  member_id?: number;
  member_mobile?: string;
  member_nickname?: string;
  order_id?: number;
  order_item_id?: number;
  order_no?: string;
  product_id?: number;
  product_name?: string;
  rating?: number;
  reply_at?: string | null;
  reply_content?: string;
  reply_user_name?: string;
  sku_id?: number;
  sku_spec_values?: string | null;
  status?: number;
  status_text?: string;
  updated_at?: string;
}

export interface ReviewListItemResponse {
  audit_reason?: string;
  audit_user_name?: string;
  audited_at?: string | null;
  content?: string;
  created_at?: string;
  has_image?: number;
  id?: number;
  images?: FileRow[];
  is_anonymous?: number;
  member_id?: number;
  member_mobile?: string;
  member_nickname?: string;
  order_id?: number;
  order_item_id?: number;
  product_id?: number;
  product_name?: string;
  rating?: number;
  reply_at?: string | null;
  reply_content?: string;
  reply_user_name?: string;
  sku_id?: number;
  sku_spec_values?: string | null;
  status?: number;
  status_text?: string;
}

export interface ReviewListRequest {
  /** YYYY-MM-DD */
  end_date?: string;
  /** 0:无图, 1:有图 */
  has_image?: number;
  /** 搜索评价内容、商品名称 */
  keyword?: string;
  member_id?: number;
  page?: number;
  page_size?: number;
  product_id?: number;
  /** 1-5 */
  rating?: number;
  /** YYYY-MM-DD */
  start_date?: string;
  /** 10:待审核, 20:公开, 30:驳回, 40:隐藏 */
  status?: number;
}

export interface ReviewLogResponse {
  action?: string;
  created_at?: string;
  from_status?: number;
  from_status_text?: string;
  id?: number;
  operator_id?: number;
  operator_name?: string;
  operator_type?: number;
  operator_type_name?: string;
  remark?: string;
  to_status?: number;
  to_status_text?: string;
}

export interface ReviewReplyRequest {
  /** 最多500字符 */
  content: string;
  id: number;
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

export interface SearchBasicConfig {
  /** 热门搜索关键词列表 (最多20条) */
  hot_keywords?: HotKeywordItem[];
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
  /** 公安联网备案号 */
  police_record?: string;
  /** 网站标题 */
  title?: string;
}

export interface SystemJobRetryAccepted {
  /** 已接受的任务名称 */
  job_name?: string;
  /** 当前任务状态 */
  status?: string;
}

export interface SystemJobStatus {
  /** 连续失败周期数 */
  consecutive_failures?: number;
  /** 任务中文名称 */
  display_name?: string;
  /** 内置任务唯一名称 */
  job_name?: string;
  /** 最近执行周期尝试次数 */
  last_attempt_count?: number;
  /** 最近执行耗时，单位毫秒 */
  last_duration_ms?: number;
  /** 最近错误摘要，已脱敏 */
  last_error?: string;
  /** 最近失败数量 */
  last_failed_count?: number;
  /** 最近结束时间 */
  last_finished_at?: string;
  /** 最近处理数量 */
  last_processed_count?: number;
  /** 最近开始时间 */
  last_started_at?: string;
  /** 最近成功时间 */
  last_success_at?: string;
  /** 最近成功数量 */
  last_success_count?: number;
  /** 人工重试次数 */
  manual_retry_count?: number;
  /** 下次自动重试时间 */
  next_retry_at?: string;
  /** idle、running、retrying、success 或 failed */
  status?: string;
}

export interface TradeBasicConfig {
  /** 订单完成后允许申请售后天数（天：0-90） */
  after_sale_days?: number;
  /** 待发货订单是否允许用户主动取消 */
  allow_cancel_unshipped?: boolean;
  /** 发货后自动确认收货天数（天：1-30） */
  auto_receive_days?: number;
  /** 取消待发货订单时是否自动退款 */
  auto_refund_on_cancel?: boolean;
  /** 待付款订单自动关闭时间（分钟：1-1440） */
  order_close_minutes?: number;
}

export interface UpdateRequest {
  id: number;
  nickname?: string;
  password?: string;
  status?: number;
  username?: string;
}

export interface UpdateReturnAddressRequest {
  address: string;
  city_code?: string;
  district_code?: string;
  id: number;
  is_default?: number;
  name: string;
  phone: string;
  province_code?: string;
  receiver: string;
  sort?: number;
  status?: number;
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

export interface VersionData {
  /** @example "20260923.1721" */
  build_time?: string;
  /** @example "a8f31c2" */
  git_commit?: string;
  /** @example "1.3.0" */
  version?: string;
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
      baseURL: axiosConfig.baseURL || "//localhost:8100",
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
 * @baseUrl //localhost:8100
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
     * @tags analytics
     * @name ListAnalyticsFlows
     * @summary 分页查询资金流水
     * @request POST:/api/analytics/flows/list
     */
    listAnalyticsFlows: (
      request: AnalyticsFlowListRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseAnalyticsFlowListResponse, ResponseAny>({
        path: `/api/analytics/flows/list`,
        method: "POST",
        body: request,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags analytics
     * @name GetAnalyticsMember
     * @summary 获取会员数据分析
     * @request GET:/api/analytics/member
     */
    getAnalyticsMember: (
      query?: {
        /** @example "2026-09-17 23:59:59" */
        end_time?: string;
        /** @example "2026-09-01 00:00:00" */
        start_time?: string;
        /** @example "last_7_days" */
        time_range?: string;
      },
      params: RequestParams = {},
    ) =>
      this.request<ResponseAnalyticsMemberResponse, ResponseAny>({
        path: `/api/analytics/member`,
        method: "GET",
        query: query,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags analytics
     * @name GetAnalyticsOverview
     * @summary 获取经营概览数据
     * @request GET:/api/analytics/overview
     */
    getAnalyticsOverview: (
      query?: {
        /** @example "2026-09-17 23:59:59" */
        end_time?: string;
        /** @example "2026-09-01 00:00:00" */
        start_time?: string;
        /**
         * today, yesterday, last_7_days, last_30_days, this_month, last_month, custom
         * @example "today"
         */
        time_range?: string;
      },
      params: RequestParams = {},
    ) =>
      this.request<ResponseAnalyticsOverviewResponse, ResponseAny>({
        path: `/api/analytics/overview`,
        method: "GET",
        query: query,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags analytics
     * @name ListAnalyticsProducts
     * @summary 分页查询商品/SKU数据分析
     * @request POST:/api/analytics/product/list
     */
    listAnalyticsProducts: (
      request: AnalyticsProductListRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseArrayAnalyticsProductAnalysisRow, ResponseAny>({
        path: `/api/analytics/product/list`,
        method: "POST",
        body: request,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags analytics
     * @name GetAnalyticsProductRankings
     * @summary 获取商品销售与退款排行
     * @request GET:/api/analytics/product/rankings
     */
    getAnalyticsProductRankings: (
      query?: {
        /** @example "2026-09-17 23:59:59" */
        end_time?: string;
        /** @example "2026-09-01 00:00:00" */
        start_time?: string;
        /** @example "last_7_days" */
        time_range?: string;
      },
      params: RequestParams = {},
    ) =>
      this.request<ResponseAnalyticsProductRankingsResponse, ResponseAny>({
        path: `/api/analytics/product/rankings`,
        method: "GET",
        query: query,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags analytics
     * @name GetAnalyticsTrade
     * @summary 获取交易分析数据
     * @request GET:/api/analytics/trade
     */
    getAnalyticsTrade: (
      query?: {
        /** @example "2026-09-17 23:59:59" */
        end_time?: string;
        /** @example "2026-09-01 00:00:00" */
        start_time?: string;
        /**
         * today, yesterday, last_7_days, last_30_days, this_month, last_month, custom
         * @example "last_7_days"
         */
        time_range?: string;
      },
      params: RequestParams = {},
    ) =>
      this.request<ResponseAnalyticsTradeResponse, ResponseAny>({
        path: `/api/analytics/trade`,
        method: "GET",
        query: query,
        format: "json",
        ...params,
      }),

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
     * @tags auth
     * @name ChangeCurrentUserPassword
     * @summary Change the current user's password
     * @request POST:/api/auth/password
     * @secure
     */
    changeCurrentUserPassword: (
      request: ChangePasswordRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseAny, any>({
        path: `/api/auth/password`,
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
     * @tags dashboard
     * @name GetDashboardOverview
     * @summary 获取数据看板概览数据
     * @request GET:/api/dashboard/overview
     */
    getDashboardOverview: (params: RequestParams = {}) =>
      this.request<ResponseDashboardOverviewResponse, ResponseAny>({
        path: `/api/dashboard/overview`,
        method: "GET",
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags dashboard
     * @name GetDashboardTrend
     * @summary 获取销售与订单趋势
     * @request GET:/api/dashboard/trend
     */
    getDashboardTrend: (
      query?: {
        /** 趋势天数 (7 或 30，默认 7) */
        days?: number;
      },
      params: RequestParams = {},
    ) =>
      this.request<ResponseDashboardTrendResponse, ResponseAny>({
        path: `/api/dashboard/trend`,
        method: "GET",
        query: query,
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
     * @tags member
     * @name GetMemberDetail
     * @summary Get member detail
     * @request POST:/api/member/members/detail
     * @secure
     */
    getMemberDetail: (request: MemberIDRequest, params: RequestParams = {}) =>
      this.request<ResponseMemberDetailResponse, any>({
        path: `/api/member/members/detail`,
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
     * @tags member
     * @name ListMembers
     * @summary List members
     * @request POST:/api/member/members/list
     * @secure
     */
    listMembers: (request: MemberListRequest, params: RequestParams = {}) =>
      this.request<ResponseArrayMemberRow, any>({
        path: `/api/member/members/list`,
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
     * @tags order
     * @name ApproveAfterSale
     * @summary 审核通过售后单
     * @request POST:/api/order/after-sales/approve
     */
    approveAfterSale: (
      request: AdminApproveRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseAny, any>({
        path: `/api/order/after-sales/approve`,
        method: "POST",
        body: request,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags order
     * @name GetAfterSaleDetail
     * @summary 获取售后单详情
     * @request POST:/api/order/after-sales/detail
     */
    getAfterSaleDetail: (
      request: AdminAfterSaleDetailRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseAdminAfterSaleDetailResponse, any>({
        path: `/api/order/after-sales/detail`,
        method: "POST",
        body: request,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags order
     * @name ListAfterSales
     * @summary 获取售后单列表
     * @request POST:/api/order/after-sales/list
     */
    listAfterSales: (
      request: AdminAfterSaleListRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseArrayAdminAfterSaleListRow, any>({
        path: `/api/order/after-sales/list`,
        method: "POST",
        body: request,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags order
     * @name ConfirmReturnReceived
     * @summary 商家确认收到退货
     * @request POST:/api/order/after-sales/receive
     */
    confirmReturnReceived: (
      request: AdminConfirmReceiveRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseAny, any>({
        path: `/api/order/after-sales/receive`,
        method: "POST",
        body: request,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags order
     * @name RejectAfterSale
     * @summary 审核拒绝售后单
     * @request POST:/api/order/after-sales/reject
     */
    rejectAfterSale: (
      request: AdminRejectRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseAny, any>({
        path: `/api/order/after-sales/reject`,
        method: "POST",
        body: request,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags order
     * @name RetryAfterSaleRefund
     * @summary 重试售后退款
     * @request POST:/api/order/after-sales/retry-refund
     */
    retryAfterSaleRefund: (
      request: AdminRetryRefundRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseAny, any>({
        path: `/api/order/after-sales/retry-refund`,
        method: "POST",
        body: request,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags order
     * @name SyncAfterSaleRefund
     * @summary 同步/恢复退款状态
     * @request POST:/api/order/after-sales/sync-refund
     */
    syncAfterSaleRefund: (
      request: AdminSyncRefundRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseAny, any>({
        path: `/api/order/after-sales/sync-refund`,
        method: "POST",
        body: request,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags order
     * @name GetAfterSaleTabCounts
     * @summary 获取售后单状态统计
     * @request POST:/api/order/after-sales/tab-counts
     */
    getAfterSaleTabCounts: (params: RequestParams = {}) =>
      this.request<ResponseAdminAfterSaleTabCountsResponse, any>({
        path: `/api/order/after-sales/tab-counts`,
        method: "POST",
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags order
     * @name GetOrderDetail
     * @summary 获取订单详情
     * @request POST:/api/order/orders/detail
     */
    getOrderDetail: (request: OrderDetailRequest, params: RequestParams = {}) =>
      this.request<ResponseOrderDetailResponse, any>({
        path: `/api/order/orders/detail`,
        method: "POST",
        body: request,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags order
     * @name ListOrders
     * @summary 获取订单列表
     * @request POST:/api/order/orders/list
     */
    listOrders: (request: OrderListRequest, params: RequestParams = {}) =>
      this.request<ResponseArrayOrderListRow, any>({
        path: `/api/order/orders/list`,
        method: "POST",
        body: request,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags order
     * @name ShipOrder
     * @summary 订单发货
     * @request POST:/api/order/orders/ship
     */
    shipOrder: (request: OrderShipRequest, params: RequestParams = {}) =>
      this.request<ResponseAny, any>({
        path: `/api/order/orders/ship`,
        method: "POST",
        body: request,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags order
     * @name GetOrderTabCounts
     * @summary 获取订单状态统计
     * @request POST:/api/order/orders/tab-counts
     */
    getOrderTabCounts: (params: RequestParams = {}) =>
      this.request<ResponseOrderTabCountsResponse, any>({
        path: `/api/order/orders/tab-counts`,
        method: "POST",
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags order-return-address
     * @name CreateOrderReturnAddress
     * @summary 新增退货地址
     * @request POST:/api/order/return-addresses/create
     * @secure
     */
    createOrderReturnAddress: (
      request: CreateReturnAddressRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseInt64, any>({
        path: `/api/order/return-addresses/create`,
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
     * @tags order-return-address
     * @name GetDefaultOrderReturnAddress
     * @summary 获取默认退货地址
     * @request GET:/api/order/return-addresses/default
     * @secure
     */
    getDefaultOrderReturnAddress: (params: RequestParams = {}) =>
      this.request<ResponseReturnAddressItem, any>({
        path: `/api/order/return-addresses/default`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags order-return-address
     * @name DeleteOrderReturnAddress
     * @summary 删除退货地址
     * @request POST:/api/order/return-addresses/delete
     * @secure
     */
    deleteOrderReturnAddress: (
      request: ReturnAddressIDRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseAny, any>({
        path: `/api/order/return-addresses/delete`,
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
     * @tags order-return-address
     * @name GetOrderReturnAddressDetail
     * @summary 获取退货地址详情
     * @request POST:/api/order/return-addresses/detail
     * @secure
     */
    getOrderReturnAddressDetail: (
      request: ReturnAddressIDRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseReturnAddressItem, any>({
        path: `/api/order/return-addresses/detail`,
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
     * @tags order-return-address
     * @name ListOrderReturnAddresses
     * @summary 获取退货地址列表
     * @request POST:/api/order/return-addresses/list
     * @secure
     */
    listOrderReturnAddresses: (
      request: ListReturnAddressesRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseArrayReturnAddressItem, any>({
        path: `/api/order/return-addresses/list`,
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
     * @tags order-return-address
     * @name SetDefaultOrderReturnAddress
     * @summary 设置默认退货地址
     * @request POST:/api/order/return-addresses/set-default
     * @secure
     */
    setDefaultOrderReturnAddress: (
      request: ReturnAddressIDRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseAny, any>({
        path: `/api/order/return-addresses/set-default`,
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
     * @tags order-return-address
     * @name UpdateOrderReturnAddress
     * @summary 更新退货地址
     * @request POST:/api/order/return-addresses/update
     * @secure
     */
    updateOrderReturnAddress: (
      request: UpdateReturnAddressRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseAny, any>({
        path: `/api/order/return-addresses/update`,
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
     * No description
     *
     * @tags product-dict-service
     * @name CreateProductDictService
     * @summary 创建服务保障
     * @request POST:/api/product/dict-service
     * @secure
     */
    createProductDictService: (
      request: DictServiceCreateRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseProductDictServiceRow, any>({
        path: `/api/product/dict-service`,
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
     * @tags product-dict-service
     * @name UpdateProductDictService
     * @summary 更新服务保障
     * @request PUT:/api/product/dict-service/{id}
     * @secure
     */
    updateProductDictService: (
      id: number,
      request: DictServiceUpdateRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseProductDictServiceRow, any>({
        path: `/api/product/dict-service/${id}`,
        method: "PUT",
        body: request,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags product-dict-service
     * @name DeleteProductDictService
     * @summary 删除服务保障
     * @request DELETE:/api/product/dict-service/{id}
     * @secure
     */
    deleteProductDictService: (id: number, params: RequestParams = {}) =>
      this.request<ResponseAny, any>({
        path: `/api/product/dict-service/${id}`,
        method: "DELETE",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags product-dict-service
     * @name ListProductDictServices
     * @summary 分页获取服务保障列表
     * @request GET:/api/product/dict-services
     * @secure
     */
    listProductDictServices: (
      query?: {
        /**
         * 页码
         * @default 1
         */
        page?: number;
        /**
         * 每页数量
         * @default 20
         */
        page_size?: number;
        /** 关键词 */
        keyword?: string;
        /** 状态: 0禁用, 1启用 */
        status?: number;
      },
      params: RequestParams = {},
    ) =>
      this.request<ResponseArrayProductDictServiceRow, any>({
        path: `/api/product/dict-services`,
        method: "GET",
        query: query,
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags product-dict-unit
     * @name CreateProductDictUnit
     * @summary 创建商品单位
     * @request POST:/api/product/dict-unit
     * @secure
     */
    createProductDictUnit: (
      request: DictUnitCreateRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseProductDictUnitRow, any>({
        path: `/api/product/dict-unit`,
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
     * @tags product-dict-unit
     * @name UpdateProductDictUnit
     * @summary 更新商品单位
     * @request PUT:/api/product/dict-unit/{id}
     * @secure
     */
    updateProductDictUnit: (
      id: number,
      request: DictUnitUpdateRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseProductDictUnitRow, any>({
        path: `/api/product/dict-unit/${id}`,
        method: "PUT",
        body: request,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags product-dict-unit
     * @name DeleteProductDictUnit
     * @summary 删除商品单位
     * @request DELETE:/api/product/dict-unit/{id}
     * @secure
     */
    deleteProductDictUnit: (id: number, params: RequestParams = {}) =>
      this.request<ResponseAny, any>({
        path: `/api/product/dict-unit/${id}`,
        method: "DELETE",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags product-dict-unit
     * @name ListProductDictUnits
     * @summary 分页获取商品单位列表
     * @request GET:/api/product/dict-units
     * @secure
     */
    listProductDictUnits: (
      query?: {
        /**
         * 页码
         * @default 1
         */
        page?: number;
        /**
         * 每页数量
         * @default 20
         */
        page_size?: number;
        /** 关键词 */
        keyword?: string;
        /** 状态: 0禁用, 1启用 */
        status?: number;
      },
      params: RequestParams = {},
    ) =>
      this.request<ResponseArrayProductDictUnitRow, any>({
        path: `/api/product/dict-units`,
        method: "GET",
        query: query,
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags product-freight
     * @name CreateFreightTemplate
     * @summary 创建运费模板
     * @request POST:/api/product/freight-templates/create
     * @secure
     */
    createFreightTemplate: (
      request: FreightTemplateCreateRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseFreightTemplateIDRequest, any>({
        path: `/api/product/freight-templates/create`,
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
     * @tags product-freight
     * @name DeleteFreightTemplate
     * @summary 删除运费模板
     * @request POST:/api/product/freight-templates/delete
     * @secure
     */
    deleteFreightTemplate: (
      request: FreightTemplateIDRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseAny, any>({
        path: `/api/product/freight-templates/delete`,
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
     * @tags product-freight
     * @name GetFreightTemplateDetail
     * @summary 获取运费模板详情
     * @request POST:/api/product/freight-templates/detail
     * @secure
     */
    getFreightTemplateDetail: (
      request: FreightTemplateIDRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseFreightTemplateDetail, any>({
        path: `/api/product/freight-templates/detail`,
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
     * @tags product-freight
     * @name ListFreightTemplates
     * @summary 分页获取运费模板列表
     * @request POST:/api/product/freight-templates/list
     * @secure
     */
    listFreightTemplates: (
      request: FreightTemplateListRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseArrayFreightTemplateRow, any>({
        path: `/api/product/freight-templates/list`,
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
     * @tags product-freight
     * @name GetFreightTemplateOptions
     * @summary 获取启用的运费模板选项
     * @request POST:/api/product/freight-templates/options
     * @secure
     */
    getFreightTemplateOptions: (params: RequestParams = {}) =>
      this.request<ResponseArrayFreightTemplateOption, any>({
        path: `/api/product/freight-templates/options`,
        method: "POST",
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags product-freight
     * @name GetFreightRegions
     * @summary 获取全国省市区三级行政区划树（用于运费模板地区配置）
     * @request POST:/api/product/freight-templates/regions
     * @secure
     */
    getFreightRegions: (params: RequestParams = {}) =>
      this.request<ResponseArrayRegionRegionNode, any>({
        path: `/api/product/freight-templates/regions`,
        method: "POST",
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags product-freight
     * @name UpdateFreightTemplateStatus
     * @summary 切换运费模板启用状态
     * @request POST:/api/product/freight-templates/status
     * @secure
     */
    updateFreightTemplateStatus: (
      request: FreightTemplateStatusRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseAny, any>({
        path: `/api/product/freight-templates/status`,
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
     * @tags product-freight
     * @name UpdateFreightTemplate
     * @summary 更新运费模板
     * @request POST:/api/product/freight-templates/update
     * @secure
     */
    updateFreightTemplate: (
      request: FreightTemplateUpdateRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseAny, any>({
        path: `/api/product/freight-templates/update`,
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
     * @name GetProductTabCounts
     * @summary Get product tab counts
     * @request POST:/api/product/products/tab-counts
     * @secure
     */
    getProductTabCounts: (params: RequestParams = {}) =>
      this.request<ResponseProductTabCountsResponse, any>({
        path: `/api/product/products/tab-counts`,
        method: "POST",
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
     * @description Approve, reject, hide or restore a product review with optimistic lock protection
     *
     * @tags product-review
     * @name AuditProductReview
     * @summary Audit or change product review status (admin)
     * @request POST:/api/product/reviews/audit
     * @secure
     */
    auditProductReview: (
      request: ReviewAuditRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseString, any>({
        path: `/api/product/reviews/audit`,
        method: "POST",
        body: request,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description Get full detail of a review including order, member, image URLs and audit history
     *
     * @tags product-review
     * @name GetProductReviewDetail
     * @summary Get product review detail (admin)
     * @request POST:/api/product/reviews/detail
     * @secure
     */
    getProductReviewDetail: (
      request: ReviewDetailRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseReviewDetailResponse, any>({
        path: `/api/product/reviews/detail`,
        method: "POST",
        body: request,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description Query product reviews with status, rating, has_image, product_id, keyword and pagination filters
     *
     * @tags product-review
     * @name ListProductReviews
     * @summary List product reviews (admin)
     * @request POST:/api/product/reviews/list
     * @secure
     */
    listProductReviews: (
      request: ReviewListRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseArrayReviewListItemResponse, any>({
        path: `/api/product/reviews/list`,
        method: "POST",
        body: request,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description Official merchant reply to a product review
     *
     * @tags product-review
     * @name ReplyProductReview
     * @summary Reply to product review (admin)
     * @request POST:/api/product/reviews/reply
     * @secure
     */
    replyProductReview: (
      request: ReviewReplyRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseString, any>({
        path: `/api/product/reviews/reply`,
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
     * @summary Archive files without breaking existing business file IDs
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
     * @summary List material groups and counts
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
     * @summary Get material counts for all, ungrouped, and each group
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
     * @summary Delete a group and move its files to ungrouped
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
     * @summary Create or rename a material group
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
     * @summary List and search admin material files
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
     * @summary Move files to a group
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
     * @summary Rename a file display name
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
     * @summary Upload material file (image, video, document); call for each file
     * @request POST:/api/system/files/upload
     * @secure
     */
    uploadSystemFile: (
      data: {
        /**
         * File resource
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
     * @description 返回四个内置定时任务的最近运行状态、执行计数和错误摘要。
     *
     * @tags system-job
     * @name ListSystemJobs
     * @summary 获取内置后台任务状态
     * @request GET:/api/system/jobs
     * @secure
     */
    listSystemJobs: (params: RequestParams = {}) =>
      this.request<ResponseArraySystemJobStatus, any>({
        path: `/api/system/jobs`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description 仅接受最终失败且租约已释放的内置任务，返回后由后台异步执行。
     *
     * @tags system-job
     * @name RetrySystemJob
     * @summary 人工重试最终失败的后台任务
     * @request POST:/api/system/jobs/{job_name}/retry
     * @secure
     */
    retrySystemJob: (jobName: string, params: RequestParams = {}) =>
      this.request<ResponseSystemJobRetryAccepted, any>({
        path: `/api/system/jobs/${jobName}/retry`,
        method: "POST",
        secure: true,
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
     * @name GetSystemAgreementSetting
     * @summary 获取政策协议设置
     * @request GET:/api/system/settings/agreement
     * @secure
     */
    getSystemAgreementSetting: (params: RequestParams = {}) =>
      this.request<ResponseAgreementBasicConfig, any>({
        path: `/api/system/settings/agreement`,
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
     * @name SaveSystemAgreementSetting
     * @summary 保存政策协议设置
     * @request POST:/api/system/settings/agreement/save
     * @secure
     */
    saveSystemAgreementSetting: (
      request: AgreementBasicConfig,
      params: RequestParams = {},
    ) =>
      this.request<ResponseAny, any>({
        path: `/api/system/settings/agreement/save`,
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
     * @name GetSystemLogisticsSetting
     * @summary 获取配送基础设置
     * @request GET:/api/system/settings/logistics
     * @secure
     */
    getSystemLogisticsSetting: (params: RequestParams = {}) =>
      this.request<ResponseLogisticsBasicConfig, any>({
        path: `/api/system/settings/logistics`,
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
     * @name SaveSystemLogisticsSetting
     * @summary 保存配送基础设置
     * @request POST:/api/system/settings/logistics/save
     * @secure
     */
    saveSystemLogisticsSetting: (
      request: LogisticsBasicConfig,
      params: RequestParams = {},
    ) =>
      this.request<ResponseAny, any>({
        path: `/api/system/settings/logistics/save`,
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
     * @name GetSystemMemberSetting
     * @summary 获取用户注册与登录设置
     * @request GET:/api/system/settings/member
     * @secure
     */
    getSystemMemberSetting: (params: RequestParams = {}) =>
      this.request<ResponseMemberBasicConfig, any>({
        path: `/api/system/settings/member`,
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
     * @name SaveSystemMemberSetting
     * @summary 保存用户注册与登录设置
     * @request POST:/api/system/settings/member/save
     * @secure
     */
    saveSystemMemberSetting: (
      request: MemberBasicConfig,
      params: RequestParams = {},
    ) =>
      this.request<ResponseAny, any>({
        path: `/api/system/settings/member/save`,
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
     * @name GetSystemMpSetting
     * @summary 获取小程序设置
     * @request GET:/api/system/settings/mp
     * @secure
     */
    getSystemMpSetting: (params: RequestParams = {}) =>
      this.request<ResponseMpConfig, any>({
        path: `/api/system/settings/mp`,
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
     * @name SaveSystemMpSetting
     * @summary 保存小程序设置
     * @request POST:/api/system/settings/mp/save
     * @secure
     */
    saveSystemMpSetting: (request: MpConfig, params: RequestParams = {}) =>
      this.request<ResponseAny, any>({
        path: `/api/system/settings/mp/save`,
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
     * @name GetSystemPaymentSetting
     * @summary 获取支付方式配置
     * @request GET:/api/system/settings/payment
     * @secure
     */
    getSystemPaymentSetting: (params: RequestParams = {}) =>
      this.request<ResponsePaymentBasicConfig, any>({
        path: `/api/system/settings/payment`,
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
     * @name SaveSystemPaymentSetting
     * @summary 保存支付方式配置
     * @request POST:/api/system/settings/payment/save
     * @secure
     */
    saveSystemPaymentSetting: (
      request: PaymentBasicConfig,
      params: RequestParams = {},
    ) =>
      this.request<ResponseAny, any>({
        path: `/api/system/settings/payment/save`,
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
     * @name GetSystemReviewSetting
     * @summary 获取商品评价设置
     * @request GET:/api/system/settings/review
     * @secure
     */
    getSystemReviewSetting: (params: RequestParams = {}) =>
      this.request<ResponseReviewBasicConfig, any>({
        path: `/api/system/settings/review`,
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
     * @name SaveSystemReviewSetting
     * @summary 保存商品评价设置
     * @request POST:/api/system/settings/review/save
     * @secure
     */
    saveSystemReviewSetting: (
      request: ReviewBasicConfig,
      params: RequestParams = {},
    ) =>
      this.request<ResponseAny, any>({
        path: `/api/system/settings/review/save`,
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
     * @name GetSystemSearchSetting
     * @summary 获取搜索与热门词设置
     * @request GET:/api/system/settings/search
     * @secure
     */
    getSystemSearchSetting: (params: RequestParams = {}) =>
      this.request<ResponseSearchBasicConfig, any>({
        path: `/api/system/settings/search`,
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
     * @name SaveSystemSearchSetting
     * @summary 保存搜索与热门词设置
     * @request POST:/api/system/settings/search/save
     * @secure
     */
    saveSystemSearchSetting: (
      request: SearchBasicConfig,
      params: RequestParams = {},
    ) =>
      this.request<ResponseAny, any>({
        path: `/api/system/settings/search/save`,
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
     */
    getSystemShopSetting: (params: RequestParams = {}) =>
      this.request<ResponseShopBasicConfig, any>({
        path: `/api/system/settings/shop`,
        method: "GET",
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
     */
    getSystemSiteSetting: (params: RequestParams = {}) =>
      this.request<ResponseSiteBasicConfig, any>({
        path: `/api/system/settings/site`,
        method: "GET",
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
     * @tags system-setting
     * @name GetSystemTradeSetting
     * @summary 获取交易基础设置
     * @request GET:/api/system/settings/trade
     * @secure
     */
    getSystemTradeSetting: (params: RequestParams = {}) =>
      this.request<ResponseTradeBasicConfig, any>({
        path: `/api/system/settings/trade`,
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
     * @name SaveSystemTradeSetting
     * @summary 保存交易基础设置
     * @request POST:/api/system/settings/trade/save
     * @secure
     */
    saveSystemTradeSetting: (
      request: TradeBasicConfig,
      params: RequestParams = {},
    ) =>
      this.request<ResponseAny, any>({
        path: `/api/system/settings/trade/save`,
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
     * @name GetSystemWechatPaySetting
     * @summary 获取微信支付配置状态（不返回密钥）
     * @request GET:/api/system/settings/wechat-pay
     * @secure
     */
    getSystemWechatPaySetting: (params: RequestParams = {}) =>
      this.request<ResponseWechatpayPublicConfig, any>({
        path: `/api/system/settings/wechat-pay`,
        method: "GET",
        secure: true,
        ...params,
      }),

    /**
     * No description
     *
     * @tags system-setting
     * @name SaveSystemWechatPaySetting
     * @summary 保存微信支付商户配置
     * @request POST:/api/system/settings/wechat-pay/save
     * @secure
     */
    saveSystemWechatPaySetting: (request: Config, params: RequestParams = {}) =>
      this.request<ResponseAny, any>({
        path: `/api/system/settings/wechat-pay/save`,
        method: "POST",
        body: request,
        secure: true,
        type: ContentType.Json,
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
    createSystemUser: (request: CreateRequest, params: RequestParams = {}) =>
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
    deleteSystemUser: (request: IDRequest, params: RequestParams = {}) =>
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
    getSystemUser: (request: IDRequest, params: RequestParams = {}) =>
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
    listSystemUsers: (request: ListRequest, params: RequestParams = {}) =>
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
    updateSystemUser: (request: UpdateRequest, params: RequestParams = {}) =>
      this.request<ResponseUserRow, any>({
        path: `/api/system/users/update`,
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
     * @tags system
     * @name GetSystemVersion
     * @summary 获取系统版本信息
     * @request GET:/api/system/version
     */
    getSystemVersion: (params: RequestParams = {}) =>
      this.request<ResponseVersionVersionData, any>({
        path: `/api/system/version`,
        method: "GET",
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
