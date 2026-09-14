import { http } from "@/services/request"

export interface MiniappApplyAfterSaleRequest {
  order_id: number
  type: number
  reason: string
  description?: string
  proof_file_ids?: number[]
  request_id: string
}

export interface MiniappCancelAfterSaleRequest {
  after_sale_id: number
}

export interface MiniappReturnLogisticsRequest {
  after_sale_id: number
  company: string
  tracking_no: string
}

export interface MiniappAfterSaleListRequest {
  page?: number
  page_size?: number
  status?: number
}

export interface MiniappAfterSaleItemResponse {
  id: number
  product_id: number
  sku_id: number
  quantity: number
  refund_amount: string
  product_name: string
  sku_text: string
  sku_code: string
  sku_image_url: string
}

export interface MiniappAfterSaleLogRow {
  action: string
  operator_type: string
  from_status_text: string
  to_status_text: string
  remark: string
  created_at: string
}

export interface MiniappAfterSaleRefundInfo {
  status: number
  status_text: string
  refund_amount: string
  refunded_at?: string | null
}

export interface MiniappAfterSaleProofFile {
  id: number
  file_type: string
  original_name: string
  mime_type: string
  size: number
  url: string
  width: number
  height: number
  duration_ms: number
  cover_url: string
}

export interface MiniappAfterSaleListItemResponse {
  id: number
  after_sale_no: string
  order_id: number
  type: number
  type_text: string
  status: number
  status_text: string
  refund_amount: string
  reason: string
  created_at: string
  items: MiniappAfterSaleItemResponse[]
}

export interface MiniappAfterSaleDetailResponse {
  id: number
  after_sale_no: string
  order_id: number
  type: number
  type_text: string
  status: number
  status_text: string
  reason: string
  description: string
  proof_urls: string[]
  proof_files: MiniappAfterSaleProofFile[]
  refund_amount: string
  return_receiver: string
  return_phone: string
  return_address: string
  buyer_return_company: string
  buyer_return_no: string
  reject_reason: string
  applied_at: string
  audited_at?: string | null
  buyer_returned_at?: string | null
  merchant_received_at?: string | null
  completed_at?: string | null
  cancelled_at?: string | null
  created_at: string
  items: MiniappAfterSaleItemResponse[]
  refund?: MiniappAfterSaleRefundInfo | null
  logs: MiniappAfterSaleLogRow[]
}

export interface MiniappAfterSaleProofUploadResponse {
  id: number
  file_type: string
  original_name: string
  mime_type: string
  size: number
  url: string
  width: number
  height: number
  duration_ms: number
}

/**
 * 上传售后凭证文件 (图片 ≤10MB 或 视频 ≤100MB)
 */
export async function uploadAfterSaleProof(filePath: string) {
  const res = await http.upload<MiniappAfterSaleProofUploadResponse>(
    "/api/miniapp/after-sales/upload",
    filePath,
    "file"
  )
  return res.data
}

/**
 * 申请售后
 */
export async function applyAfterSale(data: MiniappApplyAfterSaleRequest) {
  const res = await http.post<MiniappAfterSaleDetailResponse>(
    "/api/miniapp/after-sales/apply",
    data as any
  )
  return res.data
}

/**
 * 售后单列表
 */
export async function getAfterSaleList(data: MiniappAfterSaleListRequest = {}) {
  const res = await http.post<MiniappAfterSaleListItemResponse[]>(
    "/api/miniapp/after-sales/list",
    data as any
  )
  return res.data
}

/**
 * 售后详情
 */
export async function getAfterSaleDetail(afterSaleID: number) {
  const res = await http.post<MiniappAfterSaleDetailResponse>(
    "/api/miniapp/after-sales/detail",
    { after_sale_id: afterSaleID }
  )
  return res.data
}

/**
 * 撤销售后申请
 */
export async function cancelAfterSale(afterSaleID: number) {
  const res = await http.post<null>(
    "/api/miniapp/after-sales/cancel",
    { after_sale_id: afterSaleID }
  )
  return res.data
}

/**
 * 买家填写退货物流
 */
export async function fillReturnLogistics(data: MiniappReturnLogisticsRequest) {
  const res = await http.post<null>(
    "/api/miniapp/after-sales/return-logistics",
    data as any
  )
  return res.data
}
