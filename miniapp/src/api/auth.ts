import { http } from "@/services/request"

export interface MemberInfo {
  id: number
  nickname: string
  avatar_id: number | null
  avatar_url: string
  mobile: string | null
  status: number
  last_login_at: string | null
  created_at: string
}

export interface WechatLoginRequest {
  code: string
}

export interface WechatLoginResponse {
  token: string
  expires_at: string
  is_new_member: boolean
  member: MemberInfo
}

/**
 * 微信小程序登录 (换取 JWT Token 与会员信息)
 */
export async function wechatLogin(code: string) {
  return await http.post<WechatLoginResponse>("/api/miniapp/auth/wechat-login", { code })
}
