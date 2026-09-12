import { http } from "@/services/request"

export interface CreatePaymentRequest {
  order_id: number
  channel?: string
}

export interface MockPayRequest {
  payment_no?: string
  order_id?: number
  channel?: string
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

export function createPayment(data: CreatePaymentRequest) {
  return http.post<OrderPaymentResponse>("/api/miniapp/payments/create", data)
}

export function mockPay(data: MockPayRequest) {
  return http.post<OrderPaymentResponse>("/api/miniapp/payments/mock-pay", data)
}
