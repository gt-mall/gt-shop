import { ref } from "vue"
import { getBootstrap, type BootstrapData } from "@/api/common"
import { storage, resolveImageUrl } from "@/utils"

const BOOTSTRAP_STORAGE_KEY = "app_bootstrap_data"
const BOOTSTRAP_TTL_SECONDS = 3600 // 缓存 1 小时

/**
 * 规范化启动数据中的图片地址（相对路径转换为完整可访问 URL）
 */
function normalizeBootstrapData(data: BootstrapData): BootstrapData {
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

// 全局响应式单例，保证所有页面共享同一份数据
const cachedInitial = storage.get<BootstrapData>(BOOTSTRAP_STORAGE_KEY)
const bootstrapData = ref<BootstrapData | null>(cachedInitial ? normalizeBootstrapData(cachedInitial) : null)
const loading = ref<boolean>(false)
const error = ref<string | null>(null)

export function useBootstrap() {
  /**
   * 拉取小程序公共启动配置 (GET /api/miniapp/bootstrap)
   * @param force 是否强制跳过缓存重新请求
   */
  async function fetchBootstrap(force = false): Promise<BootstrapData | null> {
    if (bootstrapData.value && !force) {
      return bootstrapData.value
    }

    loading.value = true
    error.value = null

    try {
      const res = await getBootstrap()
      if (res && res.code === 1 && res.data) {
        const normalized = normalizeBootstrapData(res.data)
        bootstrapData.value = normalized
        storage.set(BOOTSTRAP_STORAGE_KEY, normalized, BOOTSTRAP_TTL_SECONDS)
        console.log("【Bootstrap 启动配置加载成功】:", normalized)
        return normalized
      }
      return bootstrapData.value
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "获取公共启动配置失败"
      error.value = msg
      console.warn("[Bootstrap Service] 加载启动配置失败:", msg)
      return bootstrapData.value
    } finally {
      loading.value = false
    }
  }

  return {
    bootstrapData,
    loading,
    error,
    fetchBootstrap,
  }
}
