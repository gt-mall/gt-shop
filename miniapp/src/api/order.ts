import { http } from "@/services/request"

export interface CheckoutItemRequest {
  sku_id: number
  quantity: number
}

export interface CheckoutPreviewRequest {
  address_id?: number | null
  items: CheckoutItemRequest[]
}

export interface CheckoutAddressResponse {
  id: number
  name: string
  mobile: string
  province_code: string
  province: string
  city_code: string
  city: string
  district_code: string
  district: string
  address: string
  postal_code?: string | null
  is_default: number
}

export interface CheckoutItemResponse {
  sku_id: number
  product_id: number
  product_name: string
  sku_code: string
  sku_spec_values: string
  image_url: string
  unit_price: string
  quantity: number
  total_amount: string
  stock: number
  deliverable?: boolean
  undeliverable_reason?: string
}

export interface CheckoutPreviewResponse {
  address?: CheckoutAddressResponse | null
  items: CheckoutItemResponse[]
  total_quantity: number
  total_amount: string
  freight_amount: string
  discount_amount: string
  pay_amount: string
  deliverable?: boolean
  undeliverable_reason?: string
}

export interface OrderCreateRequest {
  request_id: string
  address_id: number
  items: CheckoutItemRequest[]
  remark?: string
}

export interface OrderItemResponse {
  id: number
  product_id: number
  sku_id: number
  product_name: string
  sku_code: string
  sku_spec_values?: string | null
  image_url: string
  unit_price: string
  quantity: number
  total_amount: string
}

export interface OrderAddressResponse {
  name: string
  mobile: string
  province_code: string
  province: string
  city_code: string
  city: string
  district_code: string
  district: string
  address: string
  postal_code?: string | null
}

export interface OrderLogResponse {
  action: string
  operator_type: string
  operator_name: string
  remark: string
  created_at: string
}

export interface OrderDetailResponse {
  id: number
  order_no: string
  status: number
  status_text: string
  total_amount: string
  pay_amount: string
  freight_amount: string
  discount_amount: string
  express_company?: string | null
  express_no?: string | null
  remark: string
  paid_at?: string | null
  shipped_at?: string | null
  completed_at?: string | null
  cancelled_at?: string | null
  created_at: string
  address?: OrderAddressResponse | null
  items: OrderItemResponse[]
  logs?: OrderLogResponse[]
}

export interface OrderListItemResponse {
  id: number
  order_no: string
  status: number
  status_text: string
  total_amount: string
  pay_amount: string
  freight_amount: string
  total_quantity: number
  created_at: string
  express_company?: string | null
  express_no?: string | null
  items: OrderItemResponse[]
}

export interface OrderListParams extends Record<string, unknown> {
  page?: number
  page_size?: number
  status?: number
}

/**
 * 结账预览
 */
export async function checkoutPreview(data: CheckoutPreviewRequest) {
  return await http.post<CheckoutPreviewResponse>("/api/miniapp/checkout/preview", data)
}

/**
 * 创建订单
 */
export async function createOrder(data: OrderCreateRequest) {
  return await http.post<OrderDetailResponse>("/api/miniapp/orders/create", data)
}

/**
 * 获取我的订单列表
 */
export async function getOrderList(params?: OrderListParams) {
  return await http.get<OrderListItemResponse[]>("/api/miniapp/orders", params)
}

/**
 * 获取订单详情
 */
export async function getOrderDetail(id: number) {
  return await http.get<OrderDetailResponse>(`/api/miniapp/orders/${id}`)
}

/**
 * 取消订单
 */
export async function cancelOrder(id: number) {
  return await http.post<null>(`/api/miniapp/orders/${id}/cancel`)
}

/**
 * 确认收货
 */
export async function confirmOrder(id: number) {
  return await http.post<null>(`/api/miniapp/orders/${id}/confirm`)
}
