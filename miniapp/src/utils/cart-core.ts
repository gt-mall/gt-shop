import type { CartItem, CartSpecValue } from '@/api/cart'
import { yuanToFen, fenToYuan } from './money'

/**
 * 校验购物车项是否可参与结算
 * 必须同时满足：商品状态有效 (available) 且 购买数量不超过库存 (quantity <= stock)
 */
export function isItemCheckoutAvailable(item: CartItem | null | undefined): boolean {
  if (!item) return false
  return Boolean(item.available && item.quantity > 0 && item.quantity <= item.stock)
}

/**
 * 计算选中且符合结算条件的商品总金额（单位：分）
 * 严格使用整数分进行累加计算，杜绝浮点数失真
 */
export function calculateCartTotalFen(
  items: CartItem[],
  selectedIds: Set<number>
): number {
  if (!Array.isArray(items) || items.length === 0 || !selectedIds || selectedIds.size === 0) {
    return 0
  }

  let totalFen = 0
  for (const item of items) {
    if (selectedIds.has(item.id) && isItemCheckoutAvailable(item)) {
      const priceFen = yuanToFen(item.price)
      totalFen += priceFen * item.quantity
    }
  }
  return totalFen
}

/**
 * 计算选中且符合结算条件的商品总金额（单位：元，标准两位小数字符串）
 */
export function calculateCartTotalAmount(
  items: CartItem[],
  selectedIds: Set<number>
): string {
  const totalFen = calculateCartTotalFen(items, selectedIds)
  return fenToYuan(totalFen)
}

/**
 * 统计选中且符合结算条件的商品种类数
 */
export function calculateSelectedValidCount(
  items: CartItem[],
  selectedIds: Set<number>
): number {
  if (!Array.isArray(items) || items.length === 0 || !selectedIds || selectedIds.size === 0) {
    return 0
  }

  let count = 0
  for (const item of items) {
    if (selectedIds.has(item.id) && isItemCheckoutAvailable(item)) {
      count++
    }
  }
  return count
}

/**
 * 格式化规格值字符串
 */
export function formatCartSpecs(specs?: CartSpecValue[] | null): string {
  if (!specs || specs.length === 0) return ''
  return specs
    .map((s) => s?.value_name || '')
    .filter(Boolean)
    .join(' · ')
}

/**
 * 过滤出选中且符合结算条件的购物车项列表
 */
export function filterCheckoutCartItems(
  items: CartItem[],
  selectedIds: Set<number>
): CartItem[] {
  if (!Array.isArray(items) || items.length === 0 || !selectedIds || selectedIds.size === 0) {
    return []
  }
  return items.filter((item) => selectedIds.has(item.id) && isItemCheckoutAvailable(item))
}

export interface CheckoutValidationResult {
  ok: boolean
  error?: string
  checkoutItems?: CartItem[]
}

/**
 * 结算前置条件校验
 * @param allItems 当前购物车所有条目
 * @param selectedIds 用户选中的条目 ID 集合
 */
export function validateCheckoutPrerequisites(
  allItems: CartItem[],
  selectedIds: Set<number>
): CheckoutValidationResult {
  if (!selectedIds || selectedIds.size === 0) {
    return { ok: false, error: '请选择要结算的商品' }
  }

  const selectedItems = allItems.filter((item) => selectedIds.has(item.id))
  if (selectedItems.length === 0) {
    return { ok: false, error: '请选择要结算的商品' }
  }

  // 检查是否有选中的商品超库存（quantity > stock）
  const hasOverStock = selectedItems.some((item) => item.quantity > item.stock)
  if (hasOverStock) {
    return { ok: false, error: '库存不足，请先调整购买数量' }
  }

  // 检查是否有选中的商品已失效
  const checkoutItems = selectedItems.filter(isItemCheckoutAvailable)
  if (checkoutItems.length === 0) {
    return { ok: false, error: '所选商品暂不可结算' }
  }

  return { ok: true, checkoutItems }
}
