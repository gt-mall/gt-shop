import { http } from "@/services/request"

export interface WechatPrepayResponse {
  payment_no: string
  appId: string
  timeStamp: string
  nonceStr: string
  package: string
  signType: string
  paySign: string
}

export interface OrderPaymentResponse {
  id: number
  payment_no: string
  order_id: number
  channel: string
  amount: string
  status: number
  status_text: string
  transaction_no?: string | null
  paid_at?: string | null
  created_at: string
}

export function prepayOrder(orderID: number) {
  return http.post<WechatPrepayResponse>("/api/miniapp/payments/prepay", { order_id: orderID })
}

export function queryPayment(paymentNo: string) {
  return http.get<OrderPaymentResponse>(`/api/miniapp/payments/${encodeURIComponent(paymentNo)}`)
}
