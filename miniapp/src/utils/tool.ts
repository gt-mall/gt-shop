/**
 * 核心通用工具函数（防抖、节流、轻量深拷贝、空值校验）
 * 自研轻量实现，零外部依赖，严格保证小程序包体积
 */

/**
 * 防抖函数
 * 适用于搜索输入联想、按钮防重复点击等场景
 */
export function debounce<T extends (...args: any[]) => any>(
  fn: T,
  delay = 300
): (...args: Parameters<T>) => void {
  let timer: ReturnType<typeof setTimeout> | null = null

  return function (this: any, ...args: Parameters<T>) {
    if (timer) {
      clearTimeout(timer)
    }
    timer = setTimeout(() => {
      fn.apply(this, args)
      timer = null
    }, delay)
  }
}

/**
 * 节流函数
 * 适用于页面滚动监听 (onPageScroll)、触底加载 (onReachBottom) 等高频触发场景
 */
export function throttle<T extends (...args: any[]) => any>(
  fn: T,
  interval = 300
): (...args: Parameters<T>) => void {
  let lastTime = 0

  return function (this: any, ...args: Parameters<T>) {
    const now = Date.now()
    if (now - lastTime >= interval) {
      lastTime = now
      fn.apply(this, args)
    }
  }
}

/**
 * 轻量安全深拷贝
 * 优先调用环境原生 structuredClone，兜底采用类型递归拷贝
 */
export function deepClone<T>(source: T): T {
  if (source === null || typeof source !== 'object') {
    return source
  }

  // 环境支持 structuredClone 时直接使用原生高性能 API
  if (typeof structuredClone === 'function') {
    try {
      return structuredClone(source)
    } catch {
      // 降级到自实现递归
    }
  }

  if (source instanceof Date) {
    return new Date(source.getTime()) as any
  }

  if (source instanceof RegExp) {
    return new RegExp(source.source, source.flags) as any
  }

  if (Array.isArray(source)) {
    return source.map((item) => deepClone(item)) as any
  }

  const result: Record<string, any> = {}
  for (const key of Object.keys(source as Record<string, any>)) {
    result[key] = deepClone((source as Record<string, any>)[key])
  }
  return result as T
}

/**
 * 判空函数
 * 支持 null, undefined, '', [], {} 的空值判定
 */
export function isEmpty(val: unknown): boolean {
  if (val === null || val === undefined) return true
  if (typeof val === 'string' || Array.isArray(val)) return val.length === 0
  if (typeof val === 'object') return Object.keys(val).length === 0
  return false
}
