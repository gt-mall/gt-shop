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

  try {
    onShow(() => {
      isVisible.value = true
    })
  } catch {}

  try {
    onHide(() => {
      isVisible.value = false
    })
  } catch {}

  onMounted(() => {
    startNetworkListener()

    if (refreshFn) {
      cleanupOnline = onNetworkOnline(async () => {
        // 关键保护：仅当前前台可见的页面执行只读刷新；同一时间只允许一个刷新，避免多个组件/隐藏Tab同时发起高频重试
        if (!isVisible.value) return
        if (isRefreshing) return

        isRefreshing = true
        try {
          await refreshFn()
        } finally {
          isRefreshing = false
        }
      })
    }
  })

  onBeforeUnmount(() => {
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
