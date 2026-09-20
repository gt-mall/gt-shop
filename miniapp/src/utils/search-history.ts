import { storage } from './storage'

export const MAX_SEARCH_HISTORY_COUNT = 20
export const MAX_KEYWORD_LENGTH = 50

export const GUEST_SEARCH_HISTORY_KEY = 'search_history_guest'
export const USER_SEARCH_HISTORY_KEY_PREFIX = 'search_history_user_'

/**
 * 规范化搜索词：
 * 1. 去除首尾空白
 * 2. 将连续多个空白字符合并为单个空格
 * 3. 截断最多 50 个字符
 */
export function normalizeKeyword(keyword: unknown): string {
  if (typeof keyword !== 'string') return ''
  const trimmed = keyword.trim().replace(/\s+/g, ' ')
  return trimmed.slice(0, MAX_KEYWORD_LENGTH)
}

/**
 * 根据会员 ID 获取对应的本地存储 Key
 * 游客使用独立的 key，会员使用会员 ID 隔离 key，不使用 Token 作为 key
 */
export function getSearchHistoryKey(memberId?: number | string | null): string {
  if (memberId !== undefined && memberId !== null && memberId !== '') {
    const numericId = typeof memberId === 'number' ? memberId : Number(memberId)
    if (!Number.isNaN(numericId)) {
      if (numericId > 0) {
        return `${USER_SEARCH_HISTORY_KEY_PREFIX}${numericId}`
      }
      return GUEST_SEARCH_HISTORY_KEY
    }
    const strId = String(memberId).trim()
    if (strId) {
      return `${USER_SEARCH_HISTORY_KEY_PREFIX}${strId}`
    }
  }
  return GUEST_SEARCH_HISTORY_KEY
}

/**
 * 获取搜索历史列表（最近优先，最多 20 条）
 */
export function getSearchHistory(memberId?: number | string | null): string[] {
  const key = getSearchHistoryKey(memberId)
  const list = storage.get<string[]>(key, [])
  if (!Array.isArray(list)) {
    return []
  }
  return list
    .filter((item): item is string => typeof item === 'string' && item.trim().length > 0)
    .slice(0, MAX_SEARCH_HISTORY_COUNT)
}

/**
 * 添加搜索历史：
 * - 规范化去空并截断
 * - 去重并保持最近优先（后搜的排在最前）
 * - 最多保留 20 条
 * - 存储失败不阻断搜索，返回更新后的内存数组
 */
export function addSearchHistory(keyword: string, memberId?: number | string | null): string[] {
  const normalized = normalizeKeyword(keyword)
  if (!normalized) {
    return getSearchHistory(memberId)
  }

  const key = getSearchHistoryKey(memberId)
  const currentList = getSearchHistory(memberId)

  // 严格大小写或小写去重（保留本次输入的真实形态）
  const lowerNorm = normalized.toLowerCase()
  const filtered = currentList.filter(
    (item) => item.trim().toLowerCase() !== lowerNorm
  )

  const nextList = [normalized, ...filtered].slice(0, MAX_SEARCH_HISTORY_COUNT)
  storage.set(key, nextList)
  return nextList
}

/**
 * 删除单条搜索历史
 */
export function removeSearchHistoryItem(keyword: string, memberId?: number | string | null): string[] {
  const normalized = normalizeKeyword(keyword)
  if (!normalized) {
    return getSearchHistory(memberId)
  }

  const key = getSearchHistoryKey(memberId)
  const currentList = getSearchHistory(memberId)
  const lowerNorm = normalized.toLowerCase()
  const nextList = currentList.filter((item) => item.trim().toLowerCase() !== lowerNorm)
  storage.set(key, nextList)
  return nextList
}

/**
 * 清空指定会话（会员或游客）的搜索历史
 */
export function clearSearchHistory(memberId?: number | string | null): void {
  const key = getSearchHistoryKey(memberId)
  storage.remove(key)
}

/**
 * 清除指定会员的搜索历史（供登出时调用）
 */
export function clearUserSearchHistory(memberId: number | string): void {
  if (memberId !== undefined && memberId !== null && memberId !== '') {
    const key = getSearchHistoryKey(memberId)
    storage.remove(key)
  }
}
