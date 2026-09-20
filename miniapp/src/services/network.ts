import { ref, readonly } from "vue"

const isOnlineRef = ref<boolean>(true)
const networkTypeRef = ref<string>("unknown")

type OnlineListener = () => void
const onlineListeners = new Set<OnlineListener>()

let isListening = false
let listenerRefCount = 0

function handleNetworkStatusChange(res: { isConnected: boolean; networkType: string }) {
  const wasOffline = !isOnlineRef.value
  isOnlineRef.value = res.isConnected
  networkTypeRef.value = res.networkType

  // 离线恢复为在线时，通知所有注册的只读刷新回调
  if (wasOffline && res.isConnected) {
    onlineListeners.forEach((fn) => {
      try {
        fn()
      } catch (err) {
        console.error("Network recovery listener error:", err)
      }
    })
  }
}

/**
 * 启动网络状态监听（按需引用计数管理）
 */
export function startNetworkListener(): void {
  listenerRefCount++
  if (isListening) return

  isListening = true
  // 初次检测网络状态
  if (typeof uni !== "undefined" && typeof uni.getNetworkType === "function") {
    uni.getNetworkType({
      success(res) {
        const connected = res.networkType !== "none"
        isOnlineRef.value = connected
        networkTypeRef.value = res.networkType
      },
      fail() {
        // fail 不代表离线，保持默认
      },
    })
  }

  if (typeof uni !== "undefined" && typeof uni.onNetworkStatusChange === "function") {
    uni.onNetworkStatusChange(handleNetworkStatusChange)
  }
}

/**
 * 释放网络状态监听
 */
export function stopNetworkListener(): void {
  listenerRefCount = Math.max(0, listenerRefCount - 1)
  if (listenerRefCount > 0) return

  if (isListening) {
    if (typeof uni !== "undefined" && typeof uni.offNetworkStatusChange === "function") {
      uni.offNetworkStatusChange(handleNetworkStatusChange)
    }
    isListening = false
  }
}

/**
 * 注册网络恢复在线监听（仅供当前活跃页面的只读刷新，严禁重放写请求）
 */
export function onNetworkOnline(listener: OnlineListener): () => void {
  onlineListeners.add(listener)
  return () => {
    onlineListeners.delete(listener)
  }
}

/**
 * 导出只读响应式网络状态
 */
export const isOnline = readonly(isOnlineRef)
export const networkType = readonly(networkTypeRef)

/**
 * 主动检查一次当前网络状态
 */
export async function checkNetworkStatus(): Promise<{ isConnected: boolean; networkType: string }> {
  return new Promise((resolve) => {
    if (typeof uni === "undefined" || typeof uni.getNetworkType !== "function") {
      resolve({ isConnected: true, networkType: "unknown" })
      return
    }

    uni.getNetworkType({
      success(res) {
        const connected = res.networkType !== "none"
        isOnlineRef.value = connected
        networkTypeRef.value = res.networkType
        resolve({ isConnected: connected, networkType: res.networkType })
      },
      fail() {
        resolve({ isConnected: isOnlineRef.value, networkType: networkTypeRef.value })
      },
    })
  })
}

/**
 * 供测试调用的网络状态模拟触发
 */
export function triggerNetworkStatusChangeForTest(res: { isConnected: boolean; networkType: string }): void {
  handleNetworkStatusChange(res)
}
