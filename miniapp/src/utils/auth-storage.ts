/**
 * 统一会员认证持久化管理 (Token & Member Info)
 * 兼容 H5 与微信小程序环境，统一使用 uni 存储 API
 */

export const TOKEN_STORAGE_KEY = 'token'
export const MEMBER_STORAGE_KEY = 'member_info'

/**
 * 安全获取当前存储的会员 Token 纯字符串
 * 兼容原生 string 存储与历史 storage.set 包装对象格式 { data: string, ... }
 */
export function getToken(): string {
  try {
    const raw = uni.getStorageSync(TOKEN_STORAGE_KEY)
    if (!raw) return ''
    if (typeof raw === 'string') return raw.trim()
    if (typeof raw === 'object' && raw !== null && 'data' in raw) {
      const data = (raw as Record<string, unknown>).data
      if (typeof data === 'string') return data.trim()
    }
    return ''
  } catch {
    return ''
  }
}

/**
 * 统一设置会员 Token
 */
export function setToken(token: string): void {
  try {
    const cleanToken = typeof token === 'string' ? token.trim() : ''
    uni.setStorageSync(TOKEN_STORAGE_KEY, cleanToken)
  } catch (err) {
    console.error('[AuthStorage] setToken error:', err)
  }
}

/**
 * 统一移除会员 Token
 */
export function removeToken(): void {
  try {
    uni.removeStorageSync(TOKEN_STORAGE_KEY)
  } catch (err) {
    console.error('[AuthStorage] removeToken error:', err)
  }
}

/**
 * 读取本地缓存的会员基础资料
 */
export function getStoredMember<T = unknown>(): T | null {
  try {
    const raw = uni.getStorageSync(MEMBER_STORAGE_KEY)
    if (!raw) return null
    if (typeof raw === 'object' && raw !== null) {
      if ('data' in raw && (raw as Record<string, unknown>).data !== undefined) {
        return (raw as Record<string, unknown>).data as T
      }
      return raw as T
    }
    if (typeof raw === 'string') {
      try {
        return JSON.parse(raw) as T
      } catch {
        return null
      }
    }
    return null
  } catch {
    return null
  }
}

/**
 * 写入本地缓存的会员基础资料
 */
export function setStoredMember<T = unknown>(member: T): void {
  try {
    uni.setStorageSync(MEMBER_STORAGE_KEY, member)
  } catch (err) {
    console.error('[AuthStorage] setStoredMember error:', err)
  }
}

/**
 * 移除本地缓存的会员基础资料
 */
export function removeStoredMember(): void {
  try {
    uni.removeStorageSync(MEMBER_STORAGE_KEY)
  } catch (err) {
    console.error('[AuthStorage] removeStoredMember error:', err)
  }
}

/**
 * 统一清空所有认证存储 (Token + 会员资料)
 */
export function clearAuthStorage(): void {
  removeToken()
  removeStoredMember()
}
