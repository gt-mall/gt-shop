import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import {
  getCartList,
  addToCart,
  updateCartQuantity,
  deleteCartItems,
  type CartItem,
  type CartAddRequest,
  type CartUpdateQuantityRequest,
} from '@/api/cart'
import { getToken } from '@/utils'
import type { ApiResponse } from '@/types/api'

export const useCartStore = defineStore('cart', () => {
  // 购物车服务端为唯一事实来源，内存缓存不持久化到 storage，切换页面时不闪空
  const items = ref<CartItem[]>([])
  const loading = ref<boolean>(false)
  const loaded = ref<boolean>(false)
  const error = ref<string | null>(null)
  const mutating = ref(false)
  let sessionVersion = 0

  // 购物车商品种类数量 (SKU 款数)
  const count = computed<number>(() => items.value.length)

  // 购物车商品总件数 (所有商品 quantity 累加，用于 TabBar 角标及购物车数量)
  const totalQuantity = computed<number>(() => {
    return items.value.reduce((sum, item) => sum + (item.quantity || 0), 0)
  })

  // 有效商品
  const validItems = computed<CartItem[]>(() => {
    return items.value.filter((item) => item.available)
  })

  // 失效商品
  const invalidItems = computed<CartItem[]>(() => {
    return items.value.filter((item) => !item.available)
  })

  /**
   * 同步更新底部 TabBar 购物车徽标（index: 2 对应购物车）
   */
  function syncTabBarBadge(badgeCount?: number): void {
    const qty = badgeCount !== undefined ? badgeCount : totalQuantity.value
    try {
      if (qty > 0) {
        uni.setTabBarBadge({
          index: 2,
          text: qty > 99 ? '99+' : String(qty),
        })
      } else {
        uni.removeTabBarBadge({ index: 2 })
      }
    } catch {
      // 在非 TabBar 页面调用或 TabBar 未就绪时静默忽略
    }
  }

  // 请求序号追踪，用于解决乱序并发响应覆盖问题
  let fetchSeq = 0

  // 所有入口共享写操作锁；旧会话响应和修改前的列表请求不能覆盖新状态。
  function mutate(operation: () => Promise<ApiResponse<null>>, apply: () => void | Promise<void>) {
    const requestToken = getToken()
    const version = sessionVersion
    if (!requestToken || mutating.value) {
      return Promise.resolve({ code: 0, message: '请登录或等待购物车更新完成', data: null })
    }
    mutating.value = true
    fetchSeq++
    loading.value = false
    return operation().then(async (res) => {
      if (requestToken !== getToken() || version !== sessionVersion) {
        return { code: 0, message: '登录状态已变化，请刷新购物车', data: null }
      }
      fetchSeq++
      loading.value = false
      if (res.code === 1) await apply()
      return res
    }).finally(() => {
      if (requestToken === getToken() && version === sessionVersion) mutating.value = false
    })
  }

  /**
   * 从服务端拉取最新购物车列表
   */
  async function fetchCart(): Promise<CartItem[]> {
    const requestToken = getToken()
    const seq = ++fetchSeq

    if (!requestToken) {
      items.value = []
      loaded.value = true
      loading.value = false
      syncTabBarBadge(0)
      return []
    }

    loading.value = true
    error.value = null
    try {
      const res = await getCartList()
      // 跨账号或乱序请求防护：若 token 改变或不是最新请求序号，丢弃结果禁止写入 Store
      if (requestToken !== getToken() || seq !== fetchSeq) {
        return items.value
      }
      if (res && res.code === 1 && Array.isArray(res.data)) {
        items.value = res.data
        loaded.value = true
        syncTabBarBadge(totalQuantity.value)
        return res.data
      } else {
        const msg = res?.message || '获取购物车失败'
        error.value = msg
        return items.value
      }
    } catch (err: unknown) {
      if (requestToken !== getToken() || seq !== fetchSeq) {
        return items.value
      }
      const msg = err instanceof Error ? err.message : '获取购物车失败'
      error.value = msg
      console.warn('[CartStore] fetchCart error:', msg)
      return items.value
    } finally {
      if (requestToken === getToken() && seq === fetchSeq) {
        loading.value = false
      }
    }
  }

  /**
   * 添加商品到购物车
   */
  async function addItem(data: CartAddRequest) {
    return mutate(() => addToCart(data), async () => { await fetchCart() })
  }

  /**
   * 修改购物车商品数量
   */
  async function updateQuantity(data: CartUpdateQuantityRequest) {
    return mutate(() => updateCartQuantity(data), async () => {
      const found = items.value.find((it) => it.id === data.id)
      if (found) {
        found.quantity = data.quantity
        syncTabBarBadge(totalQuantity.value)
      } else {
        await fetchCart()
      }
    })
  }

  /**
   * 批量删除购物车项
   */
  async function removeItem(ids: number[]) {
    return mutate(() => deleteCartItems({ ids }), () => {
      const idSet = new Set(ids)
      items.value = items.value.filter((item) => !idSet.has(item.id))
      syncTabBarBadge(totalQuantity.value)
    })
  }

  /**
   * 清空失效商品或全部商品
   */
  async function clearCart() {
    const ids = items.value.map((it) => it.id)
    if (ids.length === 0) return null
    return await removeItem(ids)
  }

  /**
   * 退出登录或用户切换时彻底重置 Store 状态
   */
  function reset() {
    sessionVersion++
    mutating.value = false
    fetchSeq++ // 失效所有进行中的拉取请求，防止旧请求晚返回复活 Store
    items.value = []
    loading.value = false
    loaded.value = false
    error.value = null
    syncTabBarBadge(0)
  }

  return {
    items,
    loading,
    loaded,
    error,
    mutating,
    count,
    totalQuantity,
    validItems,
    invalidItems,
    syncTabBarBadge,
    fetchCart,
    addItem,
    updateQuantity,
    removeItem,
    clearCart,
    reset,
  }
})
