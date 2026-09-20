import { ref, onMounted, onBeforeUnmount } from "vue"
import { onShow, onHide } from "@dcloudio/uni-app"
import {
  isOnline,
  networkType,
  startNetworkListener,
  stopNetworkListener,
  onNetworkOnline,
} from "@/services/network"

export function useNetwork(
  options?:
    | { onOnlineRefresh?: () => void | Promise<void> }
    | (() => void | Promise<void>)
) {
  const isVisible = ref(true)
  let cleanupOnline: (() => void) | null = null
  let isRefreshing = false

  const refreshFn = typeof options === "function" ? options : options?.onOnlineRefresh

  // UniApp lifecycle injection is unavailable when this composable is mounted
  // by a plain Vue renderer (component tests / H5 pre-render). Network cleanup
  // still follows the Vue mount lifecycle in that environment.
  try {
    onShow(() => { isVisible.value = true })
    onHide(() => { isVisible.value = false })
  } catch {
    isVisible.value = true
  }

  onMounted(() => {
    startNetworkListener()

    if (refreshFn) {
      cleanupOnline = onNetworkOnline(async () => {
        // 关键保护：仅当前前台可见的页面执行只读刷新；同一时间只允许一个刷新，避免多个组件/隐藏Tab同时发起高频重试
        if (!isVisible.value) return
        if (isRefreshing) return

        isRefreshing = true
        await Promise.allSettled([Promise.resolve().then(refreshFn)])
        isRefreshing = false
      })
    }
  })

  onBeforeUnmount(() => {
    isVisible.value = false
    if (cleanupOnline) {
      cleanupOnline()
      cleanupOnline = null
    }
    stopNetworkListener()
  })

  return {
    isOnline,
    networkType,
    isVisible,
  }
}
