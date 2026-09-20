<template>
  <view
    v-if="rendered"
    class="cart-drawer-root"
    :class="{ 'is-active': rendered }"
  >
    <!-- 半透明背景遮罩 -->
    <view
      class="drawer-mask"
      :class="{ 'mask-visible': visible }"
      @tap="handleClose"
      @touchmove.stop.prevent="noop"
    />

    <!-- 底部滑出抽屉面板 -->
    <view
      class="drawer-panel"
      :class="{ 'panel-visible': visible }"
      :style="{ height: height, maxHeight: height }"
      @tap.stop="noop"
    >
      <!-- 1. 顶部固定标题栏 -->
      <view class="drawer-header" @touchmove.stop.prevent="noop">
        <view class="drawer-header-left">
          <text class="drawer-title">购物车</text>
          <text v-if="isLoggedIn && totalQuantity > 0" class="drawer-count">
            (共 {{ totalQuantity }} 件)
          </text>
        </view>

        <view class="drawer-header-right">
          <!-- 管理 / 完成 切换按钮 (仅登录且有商品时显示) -->
          <text
            v-if="isLoggedIn && cartList.length > 0"
            class="drawer-action-btn"
            @tap="toggleEditMode"
          >
            {{ isEditMode ? '完成' : '管理' }}
          </text>

          <!-- 关闭按钮 -->
          <view
            v-if="showClose"
            class="drawer-close-btn"
            @tap="handleClose"
          >
            <text class="close-icon">✕</text>
          </view>
        </view>
      </view>

      <!-- 2. 中间独立可滚动区域 -->
      <scroll-view
        scroll-y
        class="drawer-scroll-view"
        :enable-flex="true"
        @touchmove.stop="noop"
      >
        <!-- A. 未登录状态 -->
        <view v-if="!isLoggedIn" class="drawer-state-box">
          <view class="state-icon-wrap auth-icon-wrap">
            <text class="state-emoji">🔒</text>
          </view>
          <text class="state-title">您还未登录</text>
          <text class="state-sub">登录后即可查看购物车中的商品</text>
          <button class="state-btn primary-btn" @tap="handleGoLogin">立即登录</button>
        </view>

        <!-- B. 首次加载中状态 (无数据时) -->
        <view v-else-if="loading && cartList.length === 0" class="drawer-state-box">
          <text class="loading-spinner">⏳</text>
          <text class="state-sub">加载购物车中...</text>
        </view>

        <!-- C. 加载失败状态 (无旧数据且出错) -->
        <view v-else-if="cartList.length === 0 && loadError" class="drawer-state-box">
          <view class="state-icon-wrap error-icon-wrap">
            <text class="state-emoji">⚠️</text>
          </view>
          <text class="state-title">购物车加载失败</text>
          <text class="state-sub">{{ loadError || '网络异常，请检查后重试' }}</text>
          <button class="state-btn retry-btn" @tap="handleRefresh">重新加载</button>
        </view>

        <!-- D. 购物车空状态 -->
        <view v-else-if="cartList.length === 0" class="drawer-state-box">
          <view class="state-icon-wrap">
            <text class="state-emoji">🛒</text>
          </view>
          <text class="state-title">购物车空空如也</text>
          <text class="state-sub">快去挑选心仪的商品吧</text>
          <button class="state-btn primary-btn" @tap="handleClose">去挑选商品</button>
        </view>

        <!-- E. 购物车商品列表内容 -->
        <view v-else class="drawer-content-inner">
          <!-- 有效商品分组 -->
          <view v-if="validItems.length > 0" class="cart-section">
            <CartItemCard
              v-for="item in validItems"
              :key="item.id"
              :item="item"
              :selected="isSelected(item.id)"
              :updating="busy"
              :is-edit-mode="isEditMode"
              @toggle-select="toggleSelect"
              @change-quantity="handleChangeQuantity"
              @delete="handleItemDelete"
              @click-product="handleClickProduct"
            />
          </view>

          <!-- 失效商品分组 -->
          <view v-if="invalidItems.length > 0" class="invalid-section">
            <view class="invalid-section-header">
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
              @delete="handleItemDelete"
              @click-product="handleClickProduct"
            />
          </view>

          <!-- 底部占位空白，防止被底部操作栏遮盖 -->
          <view class="drawer-bottom-placeholder" />
        </view>
      </scroll-view>

      <!-- 3. 底部固定结算/操作栏 (仅登录且有商品时展示) -->
      <view
        v-if="isLoggedIn && cartList.length > 0"
        class="drawer-footer"
        @touchmove.stop.prevent="noop"
      >
        <!-- 全选复选框 -->
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
            @tap="handleDrawerCheckout"
          >
            去结算 ({{ selectedValidCount }})
          </button>
        </template>

        <!-- 管理/批量删除模式 -->
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
  </view>
</template>

<script setup lang="ts">
import { ref, computed, watch, onBeforeUnmount, nextTick } from 'vue'
import { onHide } from '@dcloudio/uni-app'
import CartItemCard from './CartItemCard.vue'
import { useCartManage } from '@/composables/useCartManage'
import { getToken } from '@/utils'
import { useMemberStore } from '@/stores/member'
import type { CartItem } from '@/api/cart'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    height?: string
    showClose?: boolean
  }>(),
  {
    height: '75vh',
    showClose: true,
  }
)

const emit = defineEmits<{
  (e: 'update:modelValue', val: boolean): void
  (e: 'close'): void
  (e: 'checkout'): void
  (e: 'scroll-lock', locked: boolean): void
}>()

// 动画与挂载状态管理
const rendered = ref<boolean>(false)
const visible = ref<boolean>(false)
const memberStore = useMemberStore()
const isLoggedIn = computed(() => Boolean(memberStore.token))
let closeTimer: ReturnType<typeof setTimeout> | null = null
let openTimer: ReturnType<typeof setTimeout> | null = null
let animationVersion = 0
let scrollLocked = false
let restoreScroll: (() => void) | undefined

// 购物车状态管理 Composable
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

function noop() {}

/**
 * 跨端背景滚动锁
 */
function lockScroll(locked: boolean) {
  if (scrollLocked === locked) return
  scrollLocked = locked
  emit('scroll-lock', locked)
  // #ifdef H5
  if (typeof document !== 'undefined' && document.body) {
    if (locked) {
      const elements = [document.body, document.documentElement]
      const previous = elements.map((element) => element.style.overflow)
      elements.forEach((element) => { element.style.overflow = 'hidden' })
      restoreScroll = () => elements.forEach((element, index) => { element.style.overflow = previous[index] })
    } else {
      restoreScroll?.()
      restoreScroll = undefined
    }
  }
  // #endif
}

/**
 * 获取当前页面完整路径（含参数），用于未登录重定向
 */
function getCurrentPagePath(): string {
  try {
    const pages = getCurrentPages()
    if (pages && pages.length > 0) {
      const currentPage = pages[pages.length - 1]
      const route = currentPage.route ? `/${currentPage.route}` : '/pages/index/index'
      // @ts-ignore
      const options = currentPage.options || currentPage.$page?.options || {}
      const queryStr = Object.keys(options)
        .map((k) => `${encodeURIComponent(k)}=${encodeURIComponent(options[k])}`)
        .join('&')
      return queryStr ? `${route}?${queryStr}` : route
    }
  } catch {}
  return '/pages/index/index'
}

/**
 * 刷新购物车数据
 */
async function handleRefresh(): Promise<void> {
  const token = getToken()
  if (!token) {
    cartStore.reset()
    syncSelection([])
    return
  }

  // 避免并发重复请求
  if (busy.value) return

  const prevLen = cartList.value.length
  const list = await cartStore.fetchCart()
  if (token !== getToken() || !props.modelValue) return
  syncSelection(list)

  // 已有数据刷新失败给出轻提示，不刷空列表
  if (cartStore.error && prevLen > 0) {
    uni.showToast({ title: '刷新失败，请稍后重试', icon: 'none' })
  }
}

/**
 * 打开抽屉动作
 */
function openDrawer() {
  const version = ++animationVersion
  if (openTimer) clearTimeout(openTimer)
  if (closeTimer) {
    clearTimeout(closeTimer)
    closeTimer = null
  }

  isEditMode.value = false
  rendered.value = true
  lockScroll(true)

  nextTick(() => {
    if (version !== animationVersion || !props.modelValue) return
    // 给初始离屏状态一次渲染机会，确保首次打开也执行过渡。
    openTimer = setTimeout(() => {
      openTimer = null
      if (version !== animationVersion || !props.modelValue) return
      visible.value = true
    }, 30)
    void handleRefresh()
  })
}

/**
 * 关闭抽屉动作
 */
function handleClose() {
  if (!rendered.value || closeTimer) return
  animationVersion++
  if (openTimer) { clearTimeout(openTimer); openTimer = null }
  visible.value = false
  emit('update:modelValue', false)
  emit('close')

  if (closeTimer) {
    clearTimeout(closeTimer)
  }
  // 等待向下滑出动画完成后再卸载 DOM
  closeTimer = setTimeout(() => {
    rendered.value = false
    closeTimer = null
    lockScroll(false)
  }, 320)
}

// 监听外部 v-model 变化
watch(
  () => props.modelValue,
  (val) => {
    if (val) {
      openDrawer()
    } else if (rendered.value) {
      handleClose()
    }
  },
  { immediate: true }
)

watch(() => memberStore.token, () => {
  if (props.modelValue) void handleRefresh()
})

onHide(() => {
  handleClose()
  lockScroll(false)
})

// 修改数量
async function handleChangeQuantity(item: CartItem, delta: number) {
  await changeQuantity(item, delta)
}

// 单项删除
function handleItemDelete(item: CartItem) {
  singleDelete(item)
}

// 批量删除
function handleBatchDelete() {
  batchDelete()
}

// 清理失效商品
function handleClearInvalid() {
  clearInvalid()
}

// 去结算
function handleDrawerCheckout() {
  proceedCheckout({
    redirectFrom: getCurrentPagePath(),
    onBeforeNavigate: () => {
      handleClose()
    },
    onNavigated: () => emit('checkout'),
  })
}

// 点击登录
function handleGoLogin() {
  const redirect = getCurrentPagePath()
  handleClose()
  uni.navigateTo({
    url: `/pages/auth/login?redirect=${encodeURIComponent(redirect)}`,
  })
}

// 点击商品跳转
function handleClickProduct(productId: number) {
  const currentPath = getCurrentPagePath()
  // 如果当前已经在该商品详情页，直接收起抽屉即可
  const [route, query = ''] = currentPath.split('?')
  const currentId = query.split('&').find((part) => part.startsWith('id='))?.slice(3)
  if (route === '/pages/product/detail' && Number(currentId) === productId) {
    handleClose()
    return
  }
  handleClose()
  uni.navigateTo({
    url: `/pages/product/detail?id=${productId}`,
  })
}

onBeforeUnmount(() => {
  animationVersion++
  if (openTimer) clearTimeout(openTimer)
  if (closeTimer) {
    clearTimeout(closeTimer)
    closeTimer = null
  }
  lockScroll(false)
})
</script>

<style scoped>
.cart-drawer-root {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 998;
  pointer-events: none;
}

.cart-drawer-root.is-active {
  pointer-events: auto;
}

/* 半透明遮罩 */
.drawer-mask {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.5);
  opacity: 0;
  transition: opacity 0.3s ease;
  z-index: 999;
}

.drawer-mask.mask-visible {
  opacity: 1;
}

/* 滑出面板 */
.drawer-panel {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  max-width: 1000rpx;
  margin: 0 auto;
  background-color: #f8f8f8;
  border-radius: 28rpx 28rpx 0 0;
  display: flex;
  flex-direction: column;
  transform: translateY(100%);
  transition: transform 0.32s cubic-bezier(0.25, 1, 0.5, 1);
  box-shadow: 0 -6rpx 24rpx rgba(0, 0, 0, 0.08);
  overflow: hidden;
  z-index: 1000;
  box-sizing: border-box;
}

.drawer-panel.panel-visible {
  transform: translateY(0);
}

/* 顶部标题栏 */
.drawer-header {
  height: 96rpx;
  background-color: #ffffff;
  border-bottom: 1rpx solid #f0f2f5;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 32rpx;
  flex-shrink: 0;
}

.drawer-header-left {
  display: flex;
  align-items: baseline;
  gap: 12rpx;
}

.drawer-title {
  font-size: 32rpx;
  font-weight: 600;
  color: #303133;
}

.drawer-count {
  font-size: 24rpx;
  color: #909399;
}

.drawer-header-right {
  display: flex;
  align-items: center;
  gap: 24rpx;
}

.drawer-action-btn {
  font-size: 28rpx;
  color: #606266;
  font-weight: 500;
  padding: 8rpx 12rpx;
  cursor: pointer;
}

.drawer-close-btn {
  width: 48rpx;
  height: 48rpx;
  border-radius: 24rpx;
  background-color: #f4f4f5;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
}

.close-icon {
  font-size: 26rpx;
  color: #909399;
  line-height: 1;
}

/* 中间滚动区 */
.drawer-scroll-view {
  flex: 1;
  min-height: 0;
  height: 0;
  overscroll-behavior: contain;
}

.drawer-content-inner {
  padding: 24rpx 24rpx 0;
  box-sizing: border-box;
}

/* 状态展示 */
.drawer-state-box {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 100rpx 40rpx;
}

.state-icon-wrap {
  width: 140rpx;
  height: 140rpx;
  background-color: #fff1f0;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 28rpx;
}

.auth-icon-wrap {
  background-color: #f4f4f5;
}

.error-icon-wrap {
  background-color: #fffbe6;
}

.state-emoji {
  font-size: 64rpx;
}

.loading-spinner {
  font-size: 64rpx;
  margin-bottom: 24rpx;
}

.state-title {
  font-size: 32rpx;
  font-weight: 600;
  color: #303133;
  margin-bottom: 12rpx;
}

.state-sub {
  font-size: 26rpx;
  color: #909399;
  margin-bottom: 40rpx;
  text-align: center;
}

.state-btn {
  font-size: 28rpx;
  font-weight: 500;
  padding: 0 56rpx;
  height: 72rpx;
  line-height: 72rpx;
  border-radius: 36rpx;
  border: none;
}

.state-btn::after {
  border: none;
}

.primary-btn {
  background-color: #e02e24;
  color: #ffffff;
  box-shadow: 0 4rpx 14rpx rgba(224, 46, 36, 0.25);
}

.retry-btn {
  background-color: #e02e24;
  color: #ffffff;
}

/* 失效商品分组 */
.invalid-section {
  margin-top: 16rpx;
  margin-bottom: 20rpx;
}

.invalid-section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12rpx 8rpx;
}

.invalid-title {
  font-size: 26rpx;
  color: #909399;
}

.invalid-clear-btn {
  font-size: 26rpx;
  color: #e02e24;
  cursor: pointer;
}

/* 底部占位 */
.drawer-bottom-placeholder {
  height: 24rpx;
}

/* 底部结算栏 */
.drawer-footer {
  background-color: #ffffff;
  border-top: 1rpx solid #ebeef5;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16rpx 24rpx;
  padding-bottom: calc(16rpx + constant(safe-area-inset-bottom));
  padding-bottom: calc(16rpx + env(safe-area-inset-bottom));
  flex-shrink: 0;
  box-sizing: border-box;
}

.select-all-box {
  display: flex;
  align-items: center;
  gap: 16rpx;
  cursor: pointer;
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
  min-width: 0;
  flex-wrap: wrap;
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
  flex-shrink: 0;
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
