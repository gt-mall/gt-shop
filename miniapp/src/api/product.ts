import { http } from "@/services/request"

export type ProductSortType = "default" | "sales" | "price_asc" | "price_desc"

export interface ProductListParams {
  page?: number
  page_size?: number
  keyword?: string
  category_id?: number
  sort?: ProductSortType
}

export interface ProductListItem {
  id: number
  category_id: number
  name: string
  subtitle: string
  main_image_url: string
  price: string
  market_price: string
  sales: number
  stock: number
}

export interface ProductSpecValue {
  id: number
  name: string
}

export interface ProductSpec {
  id: number
  name: string
  values: ProductSpecValue[]
}

export interface ProductSKUValue {
  spec_id: number
  value_id: number
}

export interface ProductParam {
  name: string
  value: string
}

export interface ProductSKUItem {
  id: number
  spec_values: ProductSKUValue[]
  price: string
  market_price: string
  image_url: string
  unit: string | null
  weight_gram: number | null
  stock: number
}

export interface ProductDetailData {
  id: number
  category_id: number
  name: string
  subtitle: string
  main_image_url: string
  gallery_urls: string[]
  price: string
  market_price: string
  sales: number
  specs: ProductSpec[]
  params: ProductParam[]
  skus: ProductSKUItem[]
  detail: string
}

/**
 * 获取商品列表 (公开接口，支持分页、分类筛选、关键字搜索、排序)
 */
export async function getProductList(params?: ProductListParams) {
  return await http.get<ProductListItem[]>("/api/miniapp/products", params as Record<string, unknown>)
}

/**
 * 获取商品详情 (公开接口)
 */
export async function getProductDetail(id: number | string) {
  return await http.get<ProductDetailData>(`/api/miniapp/products/${id}`)
}

export interface ProductShareCodeData {
  product_id: number
  file_id: number
  url: string
}

/**
 * 获取商品分享小程序码 (需会员登录)
 */
export async function getProductShareCode(id: number | string) {
  return await http.post<ProductShareCodeData>(`/api/miniapp/products/${id}/share-code`)
}
