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

export type AuthResponse = WechatLoginResponse

export interface RegisterRequest {
  mobile: string
  password: string
  nickname?: string
}

export interface LoginRequest {
  mobile: string
  password: string
}

/**
 * 微信小程序登录 (换取 JWT Token 与会员信息)
 */
export async function wechatLogin(code: string) {
  return await http.post<WechatLoginResponse>("/api/miniapp/auth/wechat-login", { code })
}

/**
 * 会员注册 (手机号+密码)
 */
export async function register(data: RegisterRequest) {
  return await http.post<AuthResponse>("/api/miniapp/auth/register", data)
}

/**
 * 会员登录 (手机号+密码)
 */
export async function login(data: LoginRequest) {
  return await http.post<AuthResponse>("/api/miniapp/auth/login", data)
}
