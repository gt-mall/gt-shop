import { getBootstrap, type BootstrapData } from "@/api/common"
import { storage, resolveImageUrl } from "@/utils"
import { useAppStore } from "@/stores/app"

export const BOOTSTRAP_STORAGE_KEY = "app_bootstrap_data"
export const BOOTSTRAP_TTL_SECONDS = 3600 // 缓存 1 小时

/**
 * 规范化启动数据中的图片地址（相对路径转换为完整可访问 URL）
 */
export function normalizeBootstrapData(data: BootstrapData): BootstrapData {
  if (!data) return data
  if (data.shop) {
    if (data.shop.logo) data.shop.logo = resolveImageUrl(data.shop.logo)
    if (data.shop.service_wechat_qrcode) {
      data.shop.service_wechat_qrcode = resolveImageUrl(data.shop.service_wechat_qrcode)
    }
  }
  if (data.mp && data.mp.qrcode) {
    data.mp.qrcode = resolveImageUrl(data.mp.qrcode)
  }
  if (Array.isArray(data.categories)) {
    data.categories.forEach((cat) => {
      if (cat.image_url) {
        cat.image_url = resolveImageUrl(cat.image_url)
      }
    })
  }
  return data
}

/**
 * 获取本地缓存的启动配置
 */
export function getCachedBootstrapData(): BootstrapData | null {
  return storage.get<BootstrapData>(BOOTSTRAP_STORAGE_KEY)
}

/**
 * 发起接口请求获取公共启动配置并写入本地存储
 */
export async function fetchBootstrapApi(): Promise<BootstrapData | null> {
  const res = await getBootstrap()
  if (res && res.code === 1 && res.data) {
    const normalized = normalizeBootstrapData(res.data)
    storage.set(BOOTSTRAP_STORAGE_KEY, normalized, BOOTSTRAP_TTL_SECONDS)
    console.log("【Bootstrap 启动配置加载成功】:", normalized)
    return normalized
  }
  return null
}

/**
 * 兼容旧代码调用 useBootstrap，状态完全委托至 appStore
 */
export function useBootstrap() {
  const store = useAppStore()
  return {
    bootstrapData: store.bootstrapData,
    loading: store.initializing,
    error: store.error,
    fetchBootstrap: store.fetchBootstrap,
  }
}
