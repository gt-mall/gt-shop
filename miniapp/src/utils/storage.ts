/**
 * 小程序本地缓存安全封装
 * 支持 TTL 过期时间控制，防止缓存无限期堆积
 */

interface StoragePayload<T> {
  data: T
  expireAt: number | null // 时间戳毫秒，null 表示永不过期
}

export const storage = {
  /**
   * 写入缓存
   * @param key 存储键名
   * @param value 存储值
   * @param expireSeconds 有效期（秒），缺省为永久有效
   */
  set<T>(key: string, value: T, expireSeconds?: number): boolean {
    try {
      const payload: StoragePayload<T> = {
        data: value,
        expireAt: expireSeconds && expireSeconds > 0 ? Date.now() + expireSeconds * 1000 : null,
      }
      uni.setStorageSync(key, payload)
      return true
    } catch (err) {
      console.error(`[Storage Set Error] key=${key}:`, err)
      return false
    }
  },

  /**
   * 读取缓存
   * 若已过期则自动清除并返回 defaultValue
   */
  get<T>(key: string, defaultValue: T | null = null): T | null {
    try {
      const payload = uni.getStorageSync(key) as StoragePayload<T> | ''
      if (!payload || typeof payload !== 'object' || !('data' in payload)) {
        // 兼容非包装的原生遗留数据
        return (payload as unknown as T) ?? defaultValue
      }

      if (payload.expireAt && Date.now() > payload.expireAt) {
        uni.removeStorageSync(key)
        return defaultValue
      }

      return payload.data
    } catch (err) {
      console.error(`[Storage Get Error] key=${key}:`, err)
      return defaultValue
    }
  },

  /**
   * 移除指定缓存
   */
  remove(key: string): void {
    try {
      uni.removeStorageSync(key)
    } catch (err) {
      console.error(`[Storage Remove Error] key=${key}:`, err)
    }
  },

  /**
   * 清空所有本地缓存
   */
  clear(): void {
    try {
      uni.clearStorageSync()
    } catch (err) {
      console.error('[Storage Clear Error]:', err)
    }
  },
}
