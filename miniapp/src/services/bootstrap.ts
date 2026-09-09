import { ref } from "vue"
import { getBootstrap, type BootstrapData } from "@/api/common"
import { storage } from "@/utils"

const BOOTSTRAP_STORAGE_KEY = "app_bootstrap_data"
const BOOTSTRAP_TTL_SECONDS = 3600 // 缓存 1 小时

// 全局响应式单例，保证所有页面共享同一份数据
const bootstrapData = ref<BootstrapData | null>(storage.get<BootstrapData>(BOOTSTRAP_STORAGE_KEY))
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
        bootstrapData.value = res.data
        storage.set(BOOTSTRAP_STORAGE_KEY, res.data, BOOTSTRAP_TTL_SECONDS)
        console.log("【Bootstrap 启动配置加载成功】:", res.data)
        return res.data
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
