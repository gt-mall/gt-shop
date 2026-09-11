<template>
  <view class="cart-page">
    <!-- 顶部状态栏操作区 (有数据时展示) -->
    <view v-if="cartList.length > 0" class="cart-header">
      <text class="cart-header-title">共 {{ cartList.length }} 件商品</text>
      <text class="cart-header-action" @tap="toggleEditMode">
        {{ isEditMode ? "完成" : "管理" }}
      </text>
    </view>

    <!-- 加载中状态 -->
    <view v-if="loading && cartList.length === 0" class="loading-state">
      <text class="loading-text">加载购物车中...</text>
    </view>

    <!-- 加载失败错误状态 (无旧数据且有错误) -->
    <view v-else-if="cartList.length === 0 && loadError" class="empty-state error-state">
      <view class="empty-icon-wrap error-icon-wrap">
        <text class="empty-emoji">⚠️</text>
      </view>
      <text class="empty-title">购物车加载失败</text>
      <text class="empty-subtitle">请检查网络后重试</text>
      <button class="retry-btn" @tap="loadCartData">重新加载</button>
    </view>

    <!-- 空购物车占位状态 (非加载中、无错误且无数据) -->
    <view v-else-if="!loading && !loadError && cartList.length === 0" class="empty-state">
      <view class="empty-icon-wrap">
        <text class="empty-emoji">🛒</text>
      </view>
      <text class="empty-title">购物车空空如也</text>
      <text class="empty-subtitle">快去挑选心仪的商品吧</text>
      <button class="go-home-btn" @tap="goToHome">去逛逛</button>
    </view>

    <!-- 购物车商品列表 -->
    <view v-else class="cart-content">
      <!-- 有效商品分组 -->
      <view v-if="validItems.length > 0" class="cart-group">
        <view
          v-for="item in validItems"
          :key="item.id"
          class="cart-item-card"
        >
          <!-- 勾选框 -->
          <view class="item-checkbox" @tap="toggleSelect(item.id)">
            <view
              class="checkbox-round"
              :class="{ checked: isSelected(item.id) }"
            >
              <text v-if="isSelected(item.id)" class="checkbox-tick">✓</text>
            </view>
          </view>

          <!-- 商品图片 -->
          <image
            class="item-img"
            :src="item.image_url || '/static/logo.png'"
            mode="aspectFill"
            @tap="goToProduct(item.product_id)"
          />

          <!-- 商品信息 -->
          <view class="item-info">
            <view class="item-title-row">
              <text class="item-name" @tap="goToProduct(item.product_id)">
                {{ item.product_name }}
              </text>
              <text class="item-delete-btn" @tap="handleSingleDelete(item)">✕</text>
            </view>

            <!-- 规格标签 -->
            <view v-if="formatSpecs(item.spec_values)" class="item-specs-tag">
              <text class="specs-text">{{ formatSpecs(item.spec_values) }}</text>
            </view>

            <!-- 库存不足提示 (quantity > stock) -->
            <view v-if="item.quantity > item.stock" class="stock-warning">
              <text class="stock-warning-text">库存不足，当前仅剩 {{ item.stock }} 件，请调整数量</text>
            </view>

            <!-- 价格与数量加减 -->
            <view class="item-bottom-row">
              <view class="price-box">
                <text class="currency">¥</text>
                <text class="price-val">{{ item.price }}</text>
              </view>

              <view class="stepper">
                <button
                  class="step-btn"
                  :disabled="item.quantity <= 1 || updatingIds.has(item.id)"
                  :class="{ disabled: item.quantity <= 1 || updatingIds.has(item.id) }"
                  @tap.stop="changeQuantity(item, -1)"
                >
                  -
                </button>
                <text class="step-num">{{ item.quantity }}</text>
                <button
                  class="step-btn"
                  :disabled="item.quantity >= item.stock || item.quantity >= 999 || updatingIds.has(item.id)"
                  :class="{ disabled: item.quantity >= item.stock || item.quantity >= 999 || updatingIds.has(item.id) }"
                  @tap.stop="changeQuantity(item, 1)"
                >
                  +
                </button>
              </view>
            </view>
          </view>
        </view>
      </view>

      <!-- 失效商品分组 -->
      <view v-if="invalidItems.length > 0" class="invalid-group">
        <view class="invalid-header">
          <text class="invalid-title">失效商品 ({{ invalidItems.length }})</text>
          <text class="invalid-clear-btn" @tap="handleClearInvalid">清空失效商品</text>
        </view>

        <view
          v-for="item in invalidItems"
          :key="item.id"
          class="cart-item-card invalid-card"
        >
          <!-- 失效徽标 -->
          <view class="item-checkbox disabled">
            <text class="invalid-badge">失效</text>
          </view>

          <!-- 商品图片 -->
          <image
            class="item-img invalid-img"
            :src="item.image_url || '/static/logo.png'"
            mode="aspectFill"
          />

          <!-- 商品信息 -->
          <view class="item-info">
            <view class="item-title-row">
              <text class="item-name invalid-text">{{ item.product_name }}</text>
              <text class="item-delete-btn" @tap="handleSingleDelete(item)">✕</text>
            </view>

            <!-- 失效原因提示 -->
            <view class="invalid-reason-tag">
              <text class="reason-text">{{ item.unavailable_reason || "商品已下架" }}</text>
            </view>

            <view class="item-bottom-row">
              <view class="price-box invalid-text">
                <text class="currency">¥</text>
                <text class="price-val">{{ item.price }}</text>
              </view>
              <text class="invalid-qty">x{{ item.quantity }}</text>
            </view>
          </view>
        </view>
      </view>

      <!-- 底部操作栏占位空间 -->
      <view class="bottom-bar-placeholder" />
    </view>

    <!-- 底部固定结算/操作栏 -->
    <view v-if="cartList.length > 0" class="bottom-bar">
      <!-- 全选 -->
      <view class="select-all-box" @tap="toggleSelectAll">
        <view class="checkbox-round" :class="{ checked: isAllSelected }">
          <text v-if="isAllSelected" class="checkbox-tick">✓</text>
        </view>
        <text class="select-all-label">全选</text>
      </view>

      <!-- 结算模式 -->
      <template v-if="!isEditMode">
        <view class="total-box">
          <text class="total-label">合计:</text>
          <text class="total-symbol">¥</text>
          <text class="total-amount">{{ totalAmount }}</text>
        </view>
        <button
          class="checkout-btn"
          :disabled="selectedValidCount === 0"
          :class="{ disabled: selectedValidCount === 0 }"
          @tap="handleCheckout"
        >
          去结算 ({{ selectedValidCount }})
        </button>
      </template>

      <!-- 管理/删除模式 -->
      <template v-else>
        <view class="edit-actions">
          <button
            class="batch-delete-btn"
            :disabled="selectedIds.size === 0"
            :class="{ disabled: selectedIds.size === 0 }"
            @tap="handleBatchDelete"
          >
            删除 ({{ selectedIds.size }})
          </button>
        </view>
      </template>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref, computed } from "vue"
import { onShow, onPullDownRefresh } from "@dcloudio/uni-app"
import {
  getCartList,
  updateCartQuantity,
  deleteCartItems,
  type CartItem,
  type CartSpecValue,
} from "@/api/cart"
import { getStoredToken } from "@/services/request"
import { yuanToFen, fenToYuan, generateRequestId } from "@/utils"

const loading = ref<boolean>(false)
const loadError = ref<string>("")
const cartList = ref<CartItem[]>([])
const selectedIds = ref<Set<number>>(new Set())
const updatingIds = ref<Set<number>>(new Set())
const isEditMode = ref<boolean>(false)

/**
 * 校验购物车项是否可参与结算
 * 必须同时满足：商品状态有效 (available) 且 购买数量不超过库存 (quantity <= stock)
 */
function isItemCheckoutAvailable(item: CartItem): boolean {
  return item.available && item.quantity <= item.stock
}

// 有效商品 (包含数量超过库存的商品，保留在正常区域并展示警告)
const validItems = computed(() => {
  return cartList.value.filter((item) => item.available)
})

// 失效商品
const invalidItems = computed(() => {
  return cartList.value.filter((item) => !item.available)
})

// 选中且有效的数量
const selectedValidCount = computed(() => {
  let count = 0
  for (const item of validItems.value) {
    if (selectedIds.value.has(item.id)) {
      count++
    }
  }
  return count
})

// 是否全选 (非管理模式下针对有效且库存充足的商品；管理模式下针对全部购物车商品)
const isAllSelected = computed(() => {
  const targetItems = isEditMode.value
    ? cartList.value
    : validItems.value.filter(isItemCheckoutAvailable)
  if (targetItems.length === 0) return false
  return targetItems.every((item) => selectedIds.value.has(item.id))
})

// 合计金额 (使用安全的分计算，只计算已勾选且可结算的商品)
const totalAmount = computed(() => {
  let totalFen = 0
  for (const item of validItems.value) {
    if (selectedIds.value.has(item.id) && isItemCheckoutAvailable(item)) {
      const priceFen = yuanToFen(item.price)
      totalFen += priceFen * item.quantity
    }
  }
  return fenToYuan(totalFen)
})

function isSelected(id: number): boolean {
  return selectedIds.value.has(id)
}

function toggleSelect(id: number): void {
  const newSet = new Set(selectedIds.value)
  if (newSet.has(id)) {
    newSet.delete(id)
  } else {
    newSet.add(id)
  }
  selectedIds.value = newSet
}

function toggleSelectAll(): void {
  const targetItems = isEditMode.value
    ? cartList.value
    : validItems.value.filter(isItemCheckoutAvailable)
  if (targetItems.length === 0) return

  if (isAllSelected.value) {
    // 全部取消当前目标集合
    const newSet = new Set(selectedIds.value)
    for (const item of targetItems) {
      newSet.delete(item.id)
    }
    selectedIds.value = newSet
  } else {
    // 全部勾选当前目标集合
    const newSet = new Set(selectedIds.value)
    for (const item of targetItems) {
      newSet.add(item.id)
    }
    selectedIds.value = newSet
  }
}

function toggleEditMode(): void {
  isEditMode.value = !isEditMode.value
}

function formatSpecs(specs?: CartSpecValue[]): string {
  if (!specs || specs.length === 0) return ""
  return specs
    .map((s) => s.value_name || "")
    .filter(Boolean)
    .join(" · ")
}

// 加载购物车数据
async function loadCartData(): Promise<void> {
  const token = getStoredToken()
  if (!token) {
    cartList.value = []
    selectedIds.value = new Set()
    updateTabBarBadge(0)
    loadError.value = ""
    return
  }

  loadError.value = ""
  loading.value = true
  try {
    const res = await getCartList()
    if (res && res.code === 1 && res.data) {
      cartList.value = res.data
      loadError.value = ""
      // 默认全选可结算商品 (available 且 quantity <= stock)
      const defaultSelected = new Set<number>()
      let totalQuantity = 0
      for (const item of res.data) {
        totalQuantity += item.quantity
        if (isItemCheckoutAvailable(item)) {
          defaultSelected.add(item.id)
        }
      }
      selectedIds.value = defaultSelected
      updateTabBarBadge(totalQuantity)
    } else {
      cartList.value = []
      updateTabBarBadge(0)
    }
  } catch (err: any) {
    console.error("加载购物车失败:", err)
    loadError.value = "购物车加载失败，请稍后重试"
    // 请求失败不要清空旧 cartList，也不要把 badge 清 0
    if (cartList.value.length > 0) {
      uni.showToast({ title: "刷新失败，请稍后重试", icon: "none" })
    }
  } finally {
    loading.value = false
    uni.stopPullDownRefresh()
  }
}

// 修改商品数量
async function changeQuantity(item: CartItem, delta: number): Promise<void> {
  if (updatingIds.value.has(item.id)) return

  const targetQty = item.quantity + delta
  if (targetQty < 1) {
    return
  }
  // 只有增加数量时才拦截超过库存；减少数量时必须允许（如 10 -> 9 -> ... -> 5）
  if (delta > 0 && targetQty > item.stock) {
    uni.showToast({ title: "已达最大可用库存", icon: "none" })
    return
  }
  if (targetQty > 999) {
    uni.showToast({ title: "单品最大支持999件", icon: "none" })
    return
  }

  // 临时记录防连续点击
  const newUpdating = new Set(updatingIds.value)
  newUpdating.add(item.id)
  updatingIds.value = newUpdating

  try {
    const res = await updateCartQuantity({
      id: item.id,
      quantity: targetQty,
    })
    if (res && res.code === 1) {
      item.quantity = targetQty
      // 更新 TabBar 角标
      const totalQty = cartList.value.reduce((sum, it) => sum + it.quantity, 0)
      updateTabBarBadge(totalQty)
    } else {
      uni.showToast({ title: res?.message || "更新数量失败", icon: "none" })
    }
  } catch (err: any) {
    uni.showToast({ title: err?.message || "网络异常，修改失败", icon: "none" })
  } finally {
    const resetUpdating = new Set(updatingIds.value)
    resetUpdating.delete(item.id)
    updatingIds.value = resetUpdating
  }
}

// 单个删除
function handleSingleDelete(item: CartItem): void {
  uni.showModal({
    title: "提示",
    content: "确定要从购物车中删除该商品吗？",
    success: async (res) => {
      if (res.confirm) {
        await executeDelete([item.id])
      }
    },
  })
}

// 批量删除
function handleBatchDelete(): void {
  if (selectedIds.value.size === 0) {
    uni.showToast({ title: "请选择要删除的商品", icon: "none" })
    return
  }

  const ids = Array.from(selectedIds.value)
  uni.showModal({
    title: "提示",
    content: `确定删除所选的 ${ids.length} 件商品吗？`,
    success: async (res) => {
      if (res.confirm) {
        await executeDelete(ids)
      }
    },
  })
}

// 清空失效商品
function handleClearInvalid(): void {
  const ids = invalidItems.value.map((it) => it.id)
  if (ids.length === 0) return

  uni.showModal({
    title: "提示",
    content: `确定清理全部 ${ids.length} 件失效商品吗？`,
    success: async (res) => {
      if (res.confirm) {
        await executeDelete(ids)
      }
    },
  })
}

// 执行删除接口并更新本地状态
async function executeDelete(ids: number[]): Promise<void> {
  uni.showLoading({ title: "正在删除...", mask: true })
  try {
    const res = await deleteCartItems({ ids })
    uni.hideLoading()
    if (res && res.code === 1) {
      uni.showToast({ title: "删除成功", icon: "success" })
      // 本地剔除
      const idSet = new Set(ids)
      cartList.value = cartList.value.filter((item) => !idSet.has(item.id))
      const newSelected = new Set(selectedIds.value)
      for (const id of ids) {
        newSelected.delete(id)
      }
      selectedIds.value = newSelected
      // 更新角标
      const totalQty = cartList.value.reduce((sum, it) => sum + it.quantity, 0)
      updateTabBarBadge(totalQty)
    } else {
      uni.showToast({ title: res?.message || "删除失败", icon: "none" })
    }
  } catch (err: any) {
    uni.hideLoading()
    uni.showToast({ title: err?.message || "网络异常，删除失败", icon: "none" })
  }
}

// 去结算
function handleCheckout(): void {
  const token = getStoredToken()
  if (!token) {
    uni.showToast({ title: "请先登录", icon: "none" })
    setTimeout(() => {
      uni.navigateTo({
        url: `/pages/auth/login?redirect=${encodeURIComponent("/pages/cart/index")}`,
      })
    }, 800)
    return
  }

  // 筛选已选中的有效商品
  const selectedItems = validItems.value.filter((item) => selectedIds.value.has(item.id))
  if (selectedItems.length === 0) {
    uni.showToast({ title: "请选择要结算的商品", icon: "none" })
    return
  }

  // 如果用户选中的商品里存在 quantity > stock，不要静默忽略，提示并阻止结算
  const hasOverStock = selectedItems.some((item) => item.quantity > item.stock)
  if (hasOverStock) {
    uni.showToast({ title: "库存不足，请先调整购买数量", icon: "none" })
    return
  }

  // 最终结算商品满足：selected && available && quantity <= stock
  const checkoutCartItems = selectedItems.filter(isItemCheckoutAvailable)
  if (checkoutCartItems.length === 0) {
    uni.showToast({ title: "所选商品暂不可结算", icon: "none" })
    return
  }

  const items = checkoutCartItems.map((item) => ({
    sku_id: item.sku_id,
    quantity: item.quantity,
  }))
  const cartItemIds = checkoutCartItems.map((item) => item.id)

  // 存储临时结算载荷，带上本次结算的唯一 request_id
  uni.setStorageSync("checkout_payload", {
    source: "cart",
    request_id: generateRequestId(),
    cart_item_ids: cartItemIds,
    items,
  })

  uni.navigateTo({
    url: "/pages/order/checkout",
  })
}

// 更新 TabBar 角标 (商品总件数)
function updateTabBarBadge(count: number): void {
  try {
    if (count > 0) {
      uni.setTabBarBadge({
        index: 2,
        text: count > 99 ? "99+" : String(count),
      })
    } else {
      uni.removeTabBarBadge({ index: 2 })
    }
  } catch {}
}

function goToHome(): void {
  uni.switchTab({ url: "/pages/index/index" })
}

function goToProduct(productId: number): void {
  if (productId > 0) {
    uni.navigateTo({ url: `/pages/product/detail?id=${productId}` })
  }
}

onShow(() => {
  loadCartData()
})

onPullDownRefresh(() => {
  loadCartData()
})
</script>

<style scoped>
.cart-page {
  min-height: 100vh;
  background-color: #f8f8f8;
  display: flex;
  flex-direction: column;
}

/* 顶部状态栏 */
.cart-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 24rpx 32rpx 16rpx;
  background-color: #f8f8f8;
}

.cart-header-title {
  font-size: 26rpx;
  color: #909399;
}

.cart-header-action {
  font-size: 28rpx;
  color: #303133;
  font-weight: 500;
  padding: 8rpx 16rpx;
}

/* 加载状态 */
.loading-state {
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 120rpx 0;
}

.loading-text {
  font-size: 28rpx;
  color: #909399;
}

/* 空状态 */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 140rpx 40rpx;
}

.empty-icon-wrap {
  width: 160rpx;
  height: 160rpx;
  background-color: #fff1f0;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 32rpx;
}

.empty-emoji {
  font-size: 72rpx;
}

.empty-title {
  font-size: 34rpx;
  font-weight: 600;
  color: #303133;
  margin-bottom: 12rpx;
}

.empty-subtitle {
  font-size: 26rpx;
  color: #909399;
  margin-bottom: 48rpx;
}

.go-home-btn {
  background-color: #e02e24;
  color: #ffffff;
  font-size: 30rpx;
  font-weight: 500;
  padding: 0 64rpx;
  height: 80rpx;
  line-height: 80rpx;
  border-radius: 40rpx;
  box-shadow: 0 6rpx 16rpx rgba(224, 46, 36, 0.3);
}

.go-home-btn::after {
  border: none;
}

.retry-btn {
  background-color: #e02e24;
  color: #ffffff;
  font-size: 30rpx;
  font-weight: 500;
  padding: 0 64rpx;
  height: 80rpx;
  line-height: 80rpx;
  border-radius: 40rpx;
  box-shadow: 0 6rpx 16rpx rgba(224, 46, 36, 0.3);
}

.retry-btn::after {
  border: none;
}

.error-icon-wrap {
  background-color: #fffbe6;
}

/* 列表内容区 */
.cart-content {
  padding: 0 24rpx;
}

.cart-group {
  margin-bottom: 24rpx;
}

.cart-item-card {
  background-color: #ffffff;
  border-radius: 20rpx;
  padding: 24rpx;
  margin-bottom: 20rpx;
  display: flex;
  align-items: center;
  box-shadow: 0 2rpx 8rpx rgba(0, 0, 0, 0.04);
}

/* 勾选框 */
.item-checkbox {
  padding-right: 20rpx;
  display: flex;
  align-items: center;
  justify-content: center;
}

.checkbox-round {
  width: 40rpx;
  height: 40rpx;
  border-radius: 50%;
  border: 2rpx solid #dcdfe6;
  display: flex;
  align-items: center;
  justify-content: center;
  box-sizing: border-box;
}

.checkbox-round.checked {
  background-color: #e02e24;
  border-color: #e02e24;
}

.checkbox-tick {
  color: #ffffff;
  font-size: 24rpx;
  font-weight: bold;
}

/* 商品图片 */
.item-img {
  width: 176rpx;
  height: 176rpx;
  border-radius: 12rpx;
  flex-shrink: 0;
  background-color: #f5f7fa;
}

/* 商品信息 */
.item-info {
  flex: 1;
  min-width: 0;
  margin-left: 20rpx;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  height: 176rpx;
}

.item-title-row {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
}

.item-name {
  font-size: 28rpx;
  color: #303133;
  line-height: 1.4;
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  flex: 1;
}

.item-delete-btn {
  font-size: 28rpx;
  color: #c0c4cc;
  padding: 0 8rpx 8rpx 16rpx;
}

.item-specs-tag {
  align-self: flex-start;
  background-color: #f5f7fa;
  padding: 4rpx 12rpx;
  border-radius: 6rpx;
  margin-top: 4rpx;
}

.specs-text {
  font-size: 22rpx;
  color: #909399;
}

.stock-warning {
  align-self: flex-start;
  background-color: #fff7e6;
  padding: 4rpx 12rpx;
  border-radius: 6rpx;
  margin-top: 6rpx;
}

.stock-warning-text {
  font-size: 22rpx;
  color: #fa8c16;
  font-weight: 500;
}

.item-bottom-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: auto;
}

.price-box {
  display: flex;
  align-items: baseline;
}

.currency {
  font-size: 24rpx;
  color: #e02e24;
  font-weight: 600;
}

.price-val {
  font-size: 32rpx;
  color: #e02e24;
  font-weight: bold;
}

/* 步进器 Stepper */
.stepper {
  display: flex;
  align-items: center;
  border: 1rpx solid #e4e7ed;
  border-radius: 8rpx;
  overflow: hidden;
}

.step-btn {
  width: 56rpx;
  height: 52rpx;
  line-height: 52rpx;
  text-align: center;
  background-color: #fafafa;
  color: #303133;
  font-size: 28rpx;
  font-weight: bold;
  padding: 0;
  margin: 0;
  border: none;
  border-radius: 0;
}

.step-btn::after {
  border: none;
}

.step-btn.disabled,
.step-btn[disabled] {
  color: #c0c4cc;
  background-color: #f5f7fa;
}

.step-num {
  min-width: 64rpx;
  height: 52rpx;
  line-height: 52rpx;
  text-align: center;
  font-size: 26rpx;
  color: #303133;
  background-color: #ffffff;
  padding: 0 8rpx;
}

/* 失效商品分组 */
.invalid-group {
  margin-top: 16rpx;
  margin-bottom: 24rpx;
}

.invalid-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16rpx 8rpx;
}

.invalid-title {
  font-size: 26rpx;
  color: #909399;
}

.invalid-clear-btn {
  font-size: 26rpx;
  color: #e02e24;
}

.invalid-card {
  background-color: #fafafa;
}

.invalid-badge {
  font-size: 20rpx;
  color: #909399;
  background-color: #ebeef5;
  padding: 4rpx 10rpx;
  border-radius: 6rpx;
}

.invalid-img {
  opacity: 0.5;
}

.invalid-text {
  color: #909399 !important;
}

.invalid-reason-tag {
  align-self: flex-start;
  background-color: #fff1f0;
  padding: 4rpx 12rpx;
  border-radius: 6rpx;
  margin-top: 4rpx;
}

.reason-text {
  font-size: 22rpx;
  color: #f5222d;
}

.invalid-qty {
  font-size: 26rpx;
  color: #909399;
}

/* 占位底栏 */
.bottom-bar-placeholder {
  height: 130rpx;
  /* #ifdef H5 */
  height: calc(130rpx + var(--window-bottom, 50px));
  /* #endif */
}

/* 底部操作栏 */
.bottom-bar {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  /* #ifdef H5 */
  bottom: var(--window-bottom, 50px);
  /* #endif */
  max-width: 1000rpx;
  margin: 0 auto;
  height: 110rpx;
  background-color: #ffffff;
  border-top: 1rpx solid #ebeef5;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24rpx;
  z-index: 99;
  box-sizing: border-box;
  box-shadow: 0 -2rpx 10rpx rgba(0, 0, 0, 0.04);
}

.select-all-box {
  display: flex;
  align-items: center;
  gap: 16rpx;
}

.select-all-label {
  font-size: 28rpx;
  color: #606266;
}

.total-box {
  display: flex;
  align-items: baseline;
  margin-left: auto;
  margin-right: 20rpx;
}

.total-label {
  font-size: 26rpx;
  color: #303133;
}

.total-symbol {
  font-size: 26rpx;
  color: #e02e24;
  font-weight: 600;
  margin-left: 8rpx;
}

.total-amount {
  font-size: 36rpx;
  color: #e02e24;
  font-weight: bold;
}

.checkout-btn {
  background: linear-gradient(135deg, #e02e24 0%, #ff5b4f 100%);
  color: #ffffff;
  font-size: 28rpx;
  font-weight: 500;
  height: 76rpx;
  line-height: 76rpx;
  padding: 0 40rpx;
  border-radius: 38rpx;
  margin: 0;
  border: none;
}

.checkout-btn::after {
  border: none;
}

.checkout-btn.disabled,
.checkout-btn[disabled] {
  background: #c0c4cc;
  color: #ffffff;
  opacity: 0.8;
}

.edit-actions {
  display: flex;
  align-items: center;
}

.batch-delete-btn {
  background-color: #ffffff;
  color: #e02e24;
  border: 2rpx solid #e02e24;
  font-size: 28rpx;
  font-weight: 500;
  height: 72rpx;
  line-height: 72rpx;
  padding: 0 36rpx;
  border-radius: 36rpx;
  margin: 0;
}

.batch-delete-btn::after {
  border: none;
}

.batch-delete-btn.disabled,
.batch-delete-btn[disabled] {
  border-color: #dcdfe6;
  color: #c0c4cc;
}
</style>
