import { http } from "@/services/request"

export interface CartSpecValue {
  spec_id: number
  spec_name: string
  value_id: number
  value_name: string
}

export interface CartItem {
  id: number
  product_id: number
  sku_id: number
  product_name: string
  spec_values: CartSpecValue[]
  image_url: string
  price: string
  market_price: string
  quantity: number
  stock: number
  available: boolean
  unavailable_reason: string
}

export interface CartAddRequest {
  sku_id: number
  quantity: number
}

export interface CartUpdateQuantityRequest {
  id: number
  quantity: number
}

export interface CartDeleteRequest {
  ids: number[]
}

/**
 * 获取购物车商品列表
 */
export async function getCartList() {
  return await http.get<CartItem[]>("/api/miniapp/cart")
}

/**
 * 加入购物车
 */
export async function addToCart(data: CartAddRequest) {
  return await http.post<null>("/api/miniapp/cart/add", data)
}

/**
 * 修改购物车商品数量
 */
export async function updateCartQuantity(data: CartUpdateQuantityRequest) {
  return await http.post<null>("/api/miniapp/cart/quantity", data)
}

/**
 * 批量删除购物车商品
 */
export async function deleteCartItems(data: CartDeleteRequest) {
  return await http.post<null>("/api/miniapp/cart/delete", data)
}
