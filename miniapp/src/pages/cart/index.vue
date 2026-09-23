<template>
  <view class="cart-page">
    <NetworkBanner @retry="loadCartData" />
    <!-- 顶部状态栏操作区 (有数据时展示) -->
    <view v-if="cartList.length > 0" class="cart-header">
      <text class="cart-header-title">共 {{ totalQuantity }} 件商品</text>
      <text class="cart-header-action" @tap="toggleEditMode">
        {{ isEditMode ? "完成" : "管理" }}
      </text>
    </view>

    <!-- 加载中状态 -->
    <view v-if="loading && cartList.length === 0" class="loading-state">
      <Skeleton v-for="n in 3" :key="n" type="rect" width="100%" height="220rpx" radius="16rpx" />
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
        <CartItemCard
          v-for="item in validItems"
          :key="item.id"
          :item="item"
          :selected="isSelected(item.id)"
          :updating="busy"
          :is-edit-mode="isEditMode"
          @toggle-select="toggleSelect"
          @change-quantity="handleChangeQuantity"
          @delete="handleSingleDelete"
          @click-product="goToProduct"
        />
      </view>

      <!-- 失效商品分组 -->
      <view v-if="invalidItems.length > 0" class="invalid-group">
        <view class="invalid-header">
          <text class="invalid-title">失效商品 ({{ invalidItems.length }})</text>
          <text class="invalid-clear-btn" @tap="handleClearInvalid">清空失效商品</text>
        </view>

        <CartItemCard
          v-for="item in invalidItems"
          :key="item.id"
          :item="item"
          :selected="isSelected(item.id)"
          :updating="busy"
          :is-edit-mode="isEditMode"
          @toggle-select="toggleSelect"
          @delete="handleSingleDelete"
          @click-product="goToProduct"
        />
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
          :disabled="busy || selectedValidCount === 0"
          :class="{ disabled: busy || selectedValidCount === 0 }"
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
            :disabled="busy || selectedIds.size === 0"
            :class="{ disabled: busy || selectedIds.size === 0 }"
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
import { onShow, onPullDownRefresh } from "@dcloudio/uni-app"
import CartItemCard from "@/components/CartItemCard.vue"
import { useCartManage } from "@/composables/useCartManage"
import { getToken } from "@/utils"
import type { CartItem } from "@/api/cart"
import Skeleton from "@/components/Skeleton.vue"
import NetworkBanner from "@/components/NetworkBanner.vue"
import { useNetwork } from "@/composables/useNetwork"

const {
  cartStore,
  cartList,
  loading,
  busy,
  loadError,
  totalQuantity,
  selectedIds,
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
  syncSelection,
  changeQuantity,
  singleDelete,
  batchDelete,
  clearInvalid,
  proceedCheckout,
} = useCartManage()

useNetwork(() => loadCartData())

// 加载购物车数据
async function loadCartData(): Promise<void> {
  const token = getToken()
  if (!token) {
    cartStore.reset()
    syncSelection([])
    uni.stopPullDownRefresh()
    return
  }

  try {
    const prevLen = cartList.value.length
    const list = await cartStore.fetchCart()
    syncSelection(list)
    if (cartStore.error && prevLen > 0) {
      uni.showToast({ title: "刷新失败，请稍后重试", icon: "none" })
    }
  } catch (err: unknown) {
    console.error("加载购物车失败:", err)
    if (cartList.value.length > 0) {
      uni.showToast({ title: "刷新失败，请稍后重试", icon: "none" })
    }
  } finally {
    uni.stopPullDownRefresh()
  }
}

// 修改商品数量
async function handleChangeQuantity(item: CartItem, delta: number): Promise<void> {
  await changeQuantity(item, delta)
}

// 单个删除
function handleSingleDelete(item: CartItem): void {
  singleDelete(item)
}

// 批量删除
function handleBatchDelete(): void {
  batchDelete()
}

// 清空失效商品
function handleClearInvalid(): void {
  clearInvalid()
}

// 去结算
function handleCheckout(): void {
  proceedCheckout({
    redirectFrom: "/pages/cart/index",
  })
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
  void loadCartData()
})

onPullDownRefresh(() => {
  void loadCartData()
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
