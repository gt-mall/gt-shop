import { http } from "@/services/request"

export interface MiniappSearchConfigResponse {
  hot_keywords: string[]
}

/**
 * 获取小程序搜索公开配置（热门搜索关键词列表）
 */
export async function getSearchConfig() {
  return await http.get<MiniappSearchConfigResponse>("/api/miniapp/search/config")
}
