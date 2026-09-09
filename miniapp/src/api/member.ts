import { http } from "@/services/request"
import type { MemberInfo } from "./auth"

export interface UpdateProfileRequest {
  nickname?: string
  mobile?: string
}

export interface AvatarResponse {
  avatar_id: number
  avatar_url: string
}

export interface AddressResponse {
  id: number
  member_id: number
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
  is_default: number // 1 是, 0 否
  created_at: string
}

export interface AddressCreateRequest {
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
  is_default?: number
}

export interface AddressUpdateRequest extends AddressCreateRequest {
  id: number
}

/**
 * 获取当前登录会员信息
 */
export async function getMemberMe() {
  return await http.get<MemberInfo>("/api/miniapp/member/me")
}

/**
 * 更新当前会员资料 (昵称、手机号)
 */
export async function updateMemberProfile(data: UpdateProfileRequest) {
  return await http.post<MemberInfo>("/api/miniapp/member/profile", data)
}

/**
 * 上传并更新会员头像
 */
export async function uploadMemberAvatar(filePath: string) {
  return await http.upload<AvatarResponse>("/api/miniapp/member/avatar", filePath, "file")
}

/**
 * 获取会员收货地址列表
 */
export async function getMemberAddresses() {
  return await http.get<AddressResponse[]>("/api/miniapp/member/addresses")
}

/**
 * 创建收货地址
 */
export async function createMemberAddress(data: AddressCreateRequest) {
  return await http.post<AddressResponse>("/api/miniapp/member/addresses/create", data)
}

/**
 * 更新收货地址
 */
export async function updateMemberAddress(data: AddressUpdateRequest) {
  return await http.post<AddressResponse>("/api/miniapp/member/addresses/update", data)
}

/**
 * 删除收货地址
 */
export async function deleteMemberAddress(id: number) {
  return await http.post<null>("/api/miniapp/member/addresses/delete", { id })
}

/**
 * 设置默认收货地址
 */
export async function setDefaultMemberAddress(id: number) {
  return await http.post<null>("/api/miniapp/member/addresses/default", { id })
}
