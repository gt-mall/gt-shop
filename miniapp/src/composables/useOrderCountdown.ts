import { ref, computed, watch, onBeforeUnmount } from 'vue'
import { formatCountdown } from '@/utils/date'

/**
 * 安全解析时间字符串为毫秒时间戳
 */
export function parseDateToMs(input?: string | number | Date | null): number {
  if (!input) return 0
  if (input instanceof Date) {
    const t = input.getTime()
    return isNaN(t) ? 0 : t
  }
  if (typeof input === 'number') {
    return input.toString().length === 10 ? input * 1000 : input
  }
  if (typeof input === 'string') {
    // 兼容 iOS 将 "YYYY-MM-DD" 格式替换为 "YYYY/MM/DD"
    let str = input.trim()
    if (str.includes('-') && !str.includes('T')) {
      str = str.replace(/-/g, '/')
    }
    const d = new Date(str)
    const t = d.getTime()
    return isNaN(t) ? 0 : t
  }
  return 0
}

/**
 * 共享时钟 Ticker（避免为每个订单项单独创建 setInterval，实现全局唯一定时器）
 */
let sharedIntervalId: ReturnType<typeof setInterval> | null = null
let subscriberCount = 0
const globalCurrentTime = ref<number>(Date.now())

/**
 * 手动同步当前时钟（页面从后台切换回前台或 onShow 时调用）
 */
export function syncOrderCountdownClock() {
  globalCurrentTime.value = Date.now()
}

function retainTicker() {
  subscriberCount++
  if (!sharedIntervalId) {
    globalCurrentTime.value = Date.now()
    sharedIntervalId = setInterval(() => {
      globalCurrentTime.value = Date.now()
    }, 1000)
  }
}

function releaseTicker() {
  subscriberCount = Math.max(0, subscriberCount - 1)
  if (subscriberCount === 0 && sharedIntervalId) {
    clearInterval(sharedIntervalId)
    sharedIntervalId = null
  }
}

/**
 * 重置/清理共享定时器（测试环境或全局重置时使用）
 */
export function resetOrderCountdownTicker() {
  if (sharedIntervalId) {
    clearInterval(sharedIntervalId)
    sharedIntervalId = null
  }
  subscriberCount = 0
}

export interface UseOrderCountdownOptions {
  expiresAt?: string | null
  serverTime?: string | null
  status?: number
  paused?: boolean
  onTimeout?: () => void
}

export function useOrderCountdown(optionsGetter: () => UseOrderCountdownOptions) {
  let isSubscribed = false
  let timeoutTriggered = false
  const manualPaused = ref(false)

  // 记录最近一次根据 serverTime 计算的时钟偏差
  let lastServerTimeStr: string | null = null
  let serverOffsetMs = 0

  function getServerOffset(serverTime?: string | null): number {
    if (!serverTime) {
      lastServerTimeStr = null
      serverOffsetMs = 0
      return 0
    }
    if (serverTime === lastServerTimeStr) {
      return serverOffsetMs
    }
    lastServerTimeStr = serverTime
    const sTime = parseDateToMs(serverTime)
    if (sTime > 0) {
      serverOffsetMs = sTime - Date.now()
    } else {
      serverOffsetMs = 0
    }
    return serverOffsetMs
  }

  function subscribe() {
    if (!isSubscribed) {
      retainTicker()
      isSubscribed = true
    }
  }

  function unsubscribe() {
    if (isSubscribed) {
      releaseTicker()
      isSubscribed = false
    }
  }

  const isPaused = computed(() => manualPaused.value || !!optionsGetter().paused)

  watch(
    isPaused,
    (paused) => {
      if (paused) {
        unsubscribe()
      } else {
        subscribe()
      }
    },
    { immediate: true }
  )

  onBeforeUnmount(() => {
    unsubscribe()
  })

  function pause() {
    manualPaused.value = true
  }

  function resume() {
    manualPaused.value = false
  }

  // 截止时间毫秒戳
  const expiresAtMs = computed<number>(() => {
    const opts = optionsGetter()
    return parseDateToMs(opts.expiresAt)
  })

  // 剩余秒数（每次 tick 均基于服务端校准后的时间动态重新计算）
  const remainingSeconds = computed<number>(() => {
    const opts = optionsGetter()
    if (opts.status !== 10) {
      return 0
    }
    if (expiresAtMs.value <= 0) {
      return 0
    }

    const offset = getServerOffset(opts.serverTime)
    const estimatedServerNow = globalCurrentTime.value + offset
    const diffMs = expiresAtMs.value - estimatedServerNow
    const sec = Math.floor(diffMs / 1000)

    if (sec <= 0) {
      if (!timeoutTriggered && !isPaused.value) {
        timeoutTriggered = true
        if (opts.onTimeout) {
          opts.onTimeout()
        }
      }
      return 0
    }

    return sec
  })

  // 格式化倒计时拆解
  const countdown = computed(() => {
    return formatCountdown(remainingSeconds.value)
  })

  // 是否已超时
  const isExpired = computed<boolean>(() => {
    const opts = optionsGetter()
    if (opts.status !== 10) return false
    if (expiresAtMs.value <= 0) return false
    return remainingSeconds.value <= 0
  })

  // 是否有有效待付款倒计时
  const hasCountdown = computed<boolean>(() => {
    const opts = optionsGetter()
    return opts.status === 10 && expiresAtMs.value > 0
  })

  function resetTimeoutFlag() {
    timeoutTriggered = false
  }

  return {
    remainingSeconds,
    countdown,
    isExpired,
    hasCountdown,
    resetTimeoutFlag,
    pause,
    resume,
  }
}
