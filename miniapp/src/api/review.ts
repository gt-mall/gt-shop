import { http } from "@/services/request"

export interface MiniappReviewUploadResponse {
  file_id: number
  url: string
}

export interface MiniappReviewCreateRequest {
  order_item_id: number
  rating: number
  content: string
  image_ids?: number[]
  is_anonymous?: number
  request_id: string
}

export interface MiniappReviewCreateResponse {
  id: number
  status: number
  status_text: string
}

export interface MiniappMineReviewItemResponse {
  id: number
  order_id: number
  order_item_id: number
  product_id: number
  product_name: string
  sku_id: number
  sku_spec_values?: string | null
  rating: number
  content: string
  is_anonymous: number
  status: number
  status_text: string
  audit_reason: string
  has_image: number
  images: string[]
  reply_content: string
  reply_at?: string | null
  created_at: string
}

export interface MiniappReviewDetailResponse {
  id: number
  product_id: number
  product_name: string
  sku_id: number
  sku_spec_values?: string | null
  rating: number
  content: string
  is_anonymous: number
  status: number
  status_text: string
  has_image: number
  images: string[]
  reply_content: string
  reply_at?: string | null
  created_at: string
  is_author: boolean
  user_nickname?: string
  user_avatar?: string
  order_id?: number | null
  order_item_id?: number | null
  audit_reason?: string | null
}

export interface MiniappProductReviewSummary {
  total_count: number
  has_image_count: number
  good_count: number
  medium_count: number
  bad_count: number
  average_rating: string
  good_rate: string
}

export interface MiniappProductReviewItemResponse {
  id: number
  rating: number
  content: string
  sku_spec_values?: string | null
  is_anonymous: number
  user_nickname: string
  user_avatar: string
  has_image: number
  images: string[]
  reply_content: string
  reply_at?: string | null
  created_at: string
}

export interface MiniappProductReviewsResponse {
  summary: MiniappProductReviewSummary
  items: MiniappProductReviewItemResponse[]
}

export interface MiniappProductReviewsQuery {
  page?: number
  page_size?: number
  filter?: "all" | "has_image" | "good" | "medium" | "bad"
  rating?: number
}

/**
 * 上传评价图片
 */
export async function uploadReviewImage(filePath: string) {
  const res = await http.upload<MiniappReviewUploadResponse>(
    "/api/miniapp/reviews/upload",
    filePath,
    "file"
  )
  return res.data
}

/**
 * 提交商品评价
 */
export async function createReview(data: MiniappReviewCreateRequest) {
  const res = await http.post<MiniappReviewCreateResponse>(
    "/api/miniapp/reviews/create",
    data as any
  )
  return res.data
}

/**
 * 获取我的评价列表
 */
export async function getMineReviews(page = 1, pageSize = 10) {
  const res = await http.get<MiniappMineReviewItemResponse[]>(
    "/api/miniapp/reviews/mine",
    { page, page_size: pageSize }
  )
  return {
    items: res.data || [],
    total: res.total || 0,
  }
}

/**
 * 获取单条评价详情
 */
export async function getReviewDetail(id: number) {
  const res = await http.get<MiniappReviewDetailResponse>(
    `/api/miniapp/reviews/${id}`
  )
  return res.data
}

/**
 * 获取商品公开评价列表与摘要
 */
export async function getProductReviews(
  productId: number,
  query: MiniappProductReviewsQuery = {}
) {
  const res = await http.get<MiniappProductReviewsResponse>(
    `/api/miniapp/products/${productId}/reviews`,
    query as any
  )
  return res.data
}
