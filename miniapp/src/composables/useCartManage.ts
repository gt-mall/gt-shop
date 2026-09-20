import { ref, computed, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useCartStore } from '@/stores/cart'
import { useMemberStore } from '@/stores/member'
import type { CartItem } from '@/api/cart'
import {
  getToken,
  generateRequestId,
  isItemCheckoutAvailable,
  calculateCartTotalAmount,
  calculateSelectedValidCount,
  validateCheckoutPrerequisites,
} from '@/utils'

export function useCartManage() {
  const cartStore = useCartStore()
  const memberStore = useMemberStore()
  const { items: cartList, loading, error: loadError, totalQuantity } = storeToRefs(cartStore)

  const selectedIds = ref<Set<number>>(new Set())
  const updatingIds = ref<Set<number>>(new Set())
  const isEditMode = ref<boolean>(false)
  let lastToken = getToken()
  let selectionInitialized = false
  let confirmingDelete = false
  const busy = computed(() => loading.value || cartStore.mutating || updatingIds.value.size > 0)

  // 有效商品列表
  const validItems = computed<CartItem[]>(() => {
    return cartList.value.filter((item) => item.available)
  })

  // 失效商品列表
  const invalidItems = computed<CartItem[]>(() => {
    return cartList.value.filter((item) => !item.available)
  })

  // 选中且符合结算条件的商品种类数
  const selectedValidCount = computed<number>(() => {
    return calculateSelectedValidCount(cartList.value, selectedIds.value)
  })

  // 是否全选 (正常模式针对有效且库存充足商品；管理模式针对全部购物车商品)
  const isAllSelected = computed<boolean>(() => {
    const targetItems = isEditMode.value
      ? cartList.value
      : validItems.value.filter(isItemCheckoutAvailable)
    if (targetItems.length === 0) return false
    return targetItems.every((item) => selectedIds.value.has(item.id))
  })

  // 选中商品合计金额（精确到分计算，转为元字符串）
  const totalAmount = computed<string>(() => {
    return calculateCartTotalAmount(cartList.value, selectedIds.value)
  })

  function isSelected(id: number): boolean {
    return selectedIds.value.has(id)
  }

  function toggleSelect(id: number): void {
    selectionInitialized = true
    const next = new Set(selectedIds.value)
    if (next.has(id)) {
      next.delete(id)
    } else {
      next.add(id)
    }
    selectedIds.value = next
  }

  function toggleSelectAll(): void {
    selectionInitialized = true
    const targetItems = isEditMode.value
      ? cartList.value
      : validItems.value.filter(isItemCheckoutAvailable)
    if (targetItems.length === 0) return

    const next = new Set(selectedIds.value)
    if (isAllSelected.value) {
      for (const item of targetItems) {
        next.delete(item.id)
      }
    } else {
      for (const item of targetItems) {
        next.add(item.id)
      }
    }
    selectedIds.value = next
  }

  function toggleEditMode(): void {
    isEditMode.value = !isEditMode.value
  }

  function clearSelection(): void {
    selectedIds.value = new Set()
    selectionInitialized = true
  }

  /**
   * 列表数据变动时同步勾选状态
   * 1. 账号变动时彻底清空勾选项
   * 2. 移除列表中已不存在的 ID
   * 3. 仅首次取得非空列表时默认全选，后续保留用户主动取消的选择
   */
  function syncSelection(items: CartItem[] = cartList.value): void {
    const currentToken = getToken()
    if (currentToken !== lastToken) {
      lastToken = currentToken
      selectedIds.value = new Set()
      selectionInitialized = false
      isEditMode.value = false
    }

    if (!items || items.length === 0) {
      selectedIds.value = new Set()
      return
    }

    const currentItemIds = new Set(items.map((it) => it.id))
    const preservedSelected = new Set<number>()

    // 保留依然存在的选中项
    for (const id of selectedIds.value) {
      if (currentItemIds.has(id)) {
        preservedSelected.add(id)
      }
    }

    // 仅初始化时默认勾选，删除最后一个已选项不能导致其他商品被自动选中。
    if (!selectionInitialized) {
      for (const item of items) {
        if (isItemCheckoutAvailable(item)) {
          preservedSelected.add(item.id)
        }
      }
    }

    selectedIds.value = preservedSelected
    selectionInitialized = true
  }

  // 数据刷新、删除和账号变化都同步 UI；用户主动取消全选后保持为空。
  watch(cartList, () => syncSelection(), { deep: true })
  watch(() => memberStore.token, () => {
    lastToken = getToken()
    selectedIds.value = new Set()
    selectionInitialized = false
    isEditMode.value = false
    updatingIds.value = new Set()
  }, { flush: 'sync' })

  /**
   * 修改商品数量
   */
  async function changeQuantity(item: CartItem, delta: number): Promise<boolean> {
    if (busy.value || !item.available || updatingIds.value.has(item.id)) return false

    const targetQty = item.quantity + delta
    if (targetQty < 1) {
      return false
    }

    // 只有增加数量时拦截超过库存；减少数量时必须允许（如库存从 10 减少到 5）
    if (delta > 0 && targetQty > item.stock) {
      uni.showToast({ title: '已达最大可用库存', icon: 'none' })
      return false
    }
    if (targetQty > 999) {
      uni.showToast({ title: '单品最大支持999件', icon: 'none' })
      return false
    }

    const nextUpdating = new Set(updatingIds.value)
    nextUpdating.add(item.id)
    updatingIds.value = nextUpdating

    try {
      const res = await cartStore.updateQuantity({
        id: item.id,
        quantity: targetQty,
      })
      if (res && res.code === 1) {
        return true
      } else {
        uni.showToast({ title: res?.message || '更新数量失败', icon: 'none' })
        return false
      }
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : '网络异常，修改失败'
      uni.showToast({ title: msg, icon: 'none' })
      return false
    } finally {
      const resetUpdating = new Set(updatingIds.value)
      resetUpdating.delete(item.id)
      updatingIds.value = resetUpdating
    }
  }

  /**
   * 执行删除
   */
  async function executeDelete(ids: number[]): Promise<boolean> {
    if (busy.value || !ids || ids.length === 0) return false

    uni.showLoading({ title: '正在删除...', mask: true })
    try {
      const res = await cartStore.removeItem(ids)
      uni.hideLoading()
      if (res && res.code === 1) {
        uni.showToast({ title: '删除成功', icon: 'success' })
        const nextSelected = new Set(selectedIds.value)
        for (const id of ids) {
          nextSelected.delete(id)
        }
        selectedIds.value = nextSelected
        return true
      } else {
        uni.showToast({ title: res?.message || '删除失败', icon: 'none' })
        return false
      }
    } catch (err: unknown) {
      uni.hideLoading()
      const msg = err instanceof Error ? err.message : '网络异常，删除失败'
      uni.showToast({ title: msg, icon: 'none' })
      return false
    }
  }

  /**
   * 单项删除确认
   */
  function singleDelete(item: CartItem, onSuccess?: () => void): void {
    if (busy.value || confirmingDelete) return
    const token = getToken()
    confirmingDelete = true
    uni.showModal({
      title: '提示',
      content: '确定要从购物车中删除该商品吗？',
      success: async (res) => {
        if (res.confirm && token === getToken()) {
          const ok = await executeDelete([item.id])
          if (ok && onSuccess) onSuccess()
        }
      },
      complete: () => { confirmingDelete = false },
    })
  }

  /**
   * 批量删除确认
   */
  function batchDelete(onSuccess?: () => void): void {
    if (busy.value || confirmingDelete) return
    if (selectedIds.value.size === 0) {
      uni.showToast({ title: '请选择要删除的商品', icon: 'none' })
      return
    }

    const ids = Array.from(selectedIds.value)
    const token = getToken()
    confirmingDelete = true
    uni.showModal({
      title: '提示',
      content: `确定删除所选的 ${ids.length} 件商品吗？`,
      success: async (res) => {
        if (res.confirm && token === getToken()) {
          const ok = await executeDelete(ids)
          if (ok && onSuccess) onSuccess()
        }
      },
      complete: () => { confirmingDelete = false },
    })
  }

  /**
   * 清空失效商品确认
   */
  function clearInvalid(onSuccess?: () => void): void {
    if (busy.value || confirmingDelete) return
    const ids = invalidItems.value.map((it) => it.id)
    if (ids.length === 0) return
    const token = getToken()
    confirmingDelete = true

    uni.showModal({
      title: '提示',
      content: `确定清理全部 ${ids.length} 件失效商品吗？`,
      success: async (res) => {
        if (res.confirm && token === getToken()) {
          const ok = await executeDelete(ids)
          if (ok && onSuccess) onSuccess()
        }
      },
      complete: () => { confirmingDelete = false },
    })
  }

  /**
   * 去结算
   */
  let isNavigating = false
  function proceedCheckout(options?: {
    redirectFrom?: string
    onBeforeNavigate?: () => void
    onNavigated?: () => void
  }): void {
    if (isNavigating || busy.value) return

    const token = getToken()
    if (!token) {
      uni.showToast({ title: '请先登录', icon: 'none' })
      isNavigating = true
      setTimeout(() => {
        isNavigating = false
        const redirect = options?.redirectFrom || '/pages/cart/index'
        if (options?.onBeforeNavigate) options.onBeforeNavigate()
        uni.navigateTo({
          url: `/pages/auth/login?redirect=${encodeURIComponent(redirect)}`,
        })
      }, 800)
      return
    }

    // 校验结算前置条件
    const validation = validateCheckoutPrerequisites(cartList.value, selectedIds.value)
    if (!validation.ok) {
      uni.showToast({ title: validation.error || '无法结算', icon: 'none' })
      return
    }

    const checkoutCartItems = validation.checkoutItems || []
    const items = checkoutCartItems.map((item) => ({
      sku_id: item.sku_id,
      quantity: item.quantity,
    }))
    const cartItemIds = checkoutCartItems.map((item) => item.id)

    // 存储临时结算载荷
    uni.setStorageSync('checkout_payload', {
      source: 'cart',
      request_id: generateRequestId(),
      cart_item_ids: cartItemIds,
      items,
    })

    if (options?.onBeforeNavigate) {
      options.onBeforeNavigate()
    }

    isNavigating = true
    uni.navigateTo({
      url: '/pages/order/checkout',
      success: () => options?.onNavigated?.(),
      fail: () => uni.showToast({ title: '打开结算页失败，请重试', icon: 'none' }),
      complete: () => {
        setTimeout(() => {
          isNavigating = false
        }, 500)
      },
    })
  }

  return {
    cartStore,
    cartList,
    loading,
    loadError,
    busy,
    totalQuantity,
    selectedIds,
    updatingIds,
    isEditMode,
    validItems,
    invalidItems,
    selectedValidCount,
    isAllSelected,
    totalAmount,
    isSelected,
    toggleSelect,
    toggleSelectAll,
    toggleEditMode,
    clearSelection,
    syncSelection,
    changeQuantity,
    executeDelete,
    singleDelete,
    batchDelete,
    clearInvalid,
    proceedCheckout,
  }
}
