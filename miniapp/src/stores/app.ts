import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type {
  BootstrapData,
  BootstrapShop,
  BootstrapSite,
  BootstrapMp,
  BootstrapCategory,
} from '@/api/common'
import {
  fetchBootstrapApi,
  normalizeBootstrapData,
  getCachedBootstrapData,
} from '@/services/bootstrap'

export const useAppStore = defineStore('app', () => {
  const cached = getCachedBootstrapData()
  const bootstrapData = ref<BootstrapData | null>(
    cached ? normalizeBootstrapData(cached) : null
  )
  const initialized = ref<boolean>(!!bootstrapData.value)
  const initializing = ref<boolean>(false)
  const error = ref<string | null>(null)

  // 防止并发重复请求
  let pendingInitPromise: Promise<BootstrapData | null> | null = null

  const shop = computed<BootstrapShop | null>(() => bootstrapData.value?.shop ?? null)
  const site = computed<BootstrapSite | null>(() => bootstrapData.value?.site ?? null)
  const mp = computed<BootstrapMp | null>(() => bootstrapData.value?.mp ?? null)
  const categories = computed<BootstrapCategory[]>(() => bootstrapData.value?.categories ?? [])

  /**
   * 初始化商城公共配置
   * 防并发与防重复保护：若已初始化且非强制刷新，直接返回；若正在初始化，复用进行中的 Promise
   */
  async function init(force = false): Promise<BootstrapData | null> {
    if (initialized.value && !force && bootstrapData.value) {
      return bootstrapData.value
    }

    if (initializing.value && pendingInitPromise) {
      return pendingInitPromise
    }

    initializing.value = true
    error.value = null

    pendingInitPromise = (async () => {
      try {
        const data = await fetchBootstrapApi()
        if (data) {
          bootstrapData.value = data
          initialized.value = true
          return data
        }
        return bootstrapData.value
      } catch (err: unknown) {
        const msg = err instanceof Error ? err.message : '获取商城公共配置失败'
        error.value = msg
        console.warn('[AppStore] init error:', msg)
        return bootstrapData.value
      } finally {
        initializing.value = false
        pendingInitPromise = null
      }
    })()

    return pendingInitPromise
  }

  // 保持对旧代码 fetchBootstrap(force) 调用的兼容
  const fetchBootstrap = (force = false) => init(force)

  return {
    bootstrapData,
    initialized,
    initializing,
    error,
    shop,
    site,
    mp,
    categories,
    init,
    fetchBootstrap,
  }
})
