<template>
  <view class="order-detail-page">
    <!-- 顶部状态横幅 (拼多多风格渐变状态头) -->
    <view class="status-banner" :class="'banner-status-' + (order?.status || 0)">
      <view class="banner-content">
        <text class="status-title">{{ order?.status_text || "加载中..." }}</text>
        <text class="status-desc">{{ statusDescription }}</text>
      </view>
      <view class="banner-icon-wrap">
        <text class="banner-icon">{{ statusIcon }}</text>
      </view>
    </view>

    <!-- 待付款支付倒计时横幅 / 超时确认状态 -->
    <view v-if="showCountdownSection" class="countdown-card">
      <OrderPaymentCountdown
        :expires-at="order?.payment_expires_at"
        :server-time="order?.server_time"
        :status="order?.status"
        :is-confirming="isConfirming"
        :confirm-failed="confirmFailed"
        @timeout="handleOrderTimeout"
        @retry="handleManualRetry"
      />
    </view>

    <!-- 物流信息卡片 (仅已发货/已完成状态展示) -->
    <view v-if="order?.express_company && order?.express_no" class="express-card">
      <view class="express-icon-wrap">
        <text class="express-icon">🚚</text>
      </view>
      <view class="express-info">
        <text class="express-company">{{ order.express_company }}</text>
        <view class="express-no-row">
          <text class="express-no">运单号: {{ order.express_no }}</text>
          <text class="copy-btn" @tap="handleCopy(order.express_no)">复制</text>
        </view>
      </view>
    </view>

    <!-- 收货地址快照卡片 -->
    <view v-if="order?.address" class="address-card">
      <view class="address-icon-wrap">
        <text class="address-icon">📍</text>
      </view>
      <view class="address-content">
        <view class="user-row">
          <text class="user-name">{{ order.address.name }}</text>
          <text class="user-phone">{{ order.address.mobile }}</text>
        </view>
        <text class="address-text">
          {{ order.address.province }} {{ order.address.city }} {{ order.address.district }} {{ order.address.address }}
        </text>
      </view>
    </view>

    <!-- 商品清单快照卡片 -->
    <view class="items-card">
      <view class="card-header">
        <text class="header-icon">🏬</text>
        <text class="header-title">商品明细 (下单快照)</text>
      </view>

      <OrderGoodsList :items="order?.items || []" />

      <view v-if="canApplyAfterSale" class="aftersale-row">
        <button class="aftersale-btn" @tap="handleApplyAfterSale">申请售后</button>
      </view>
    </view>

    <!-- 费用明细卡片 -->
    <OrderAmountSummary
      :total-amount="order?.total_amount"
      :freight-amount="order?.freight_amount"
      :discount-amount="order?.discount_amount"
      :pay-amount="order?.pay_amount"
    />

    <!-- 订单信息卡片 -->
    <view class="order-meta-card">
      <view class="meta-row">
        <text class="meta-label">订单编号</text>
        <view class="meta-val-row">
          <text class="meta-val">{{ order?.order_no }}</text>
          <text class="copy-btn" @tap="handleCopy(order?.order_no || '')">复制</text>
        </view>
      </view>
      <view class="meta-row">
        <text class="meta-label">创建时间</text>
        <text class="meta-val">{{ order?.created_at ? formatDate(order.created_at) : "-" }}</text>
      </view>
      <view v-if="order?.paid_at" class="meta-row">
        <text class="meta-label">付款时间</text>
        <text class="meta-val">{{ formatDate(order.paid_at) }}</text>
      </view>
      <view v-if="order?.shipped_at" class="meta-row">
        <text class="meta-label">发货时间</text>
        <text class="meta-val">{{ formatDate(order.shipped_at) }}</text>
      </view>
      <view v-if="order?.completed_at" class="meta-row">
        <text class="meta-label">成交时间</text>
        <text class="meta-val">{{ formatDate(order.completed_at) }}</text>
      </view>
      <view v-if="order?.cancelled_at" class="meta-row">
        <text class="meta-label">取消时间</text>
        <text class="meta-val">{{ formatDate(order.cancelled_at) }}</text>
      </view>
      <view v-if="order?.remark" class="meta-row">
        <text class="meta-label">订单备注</text>
        <text class="meta-val remark-text">{{ order.remark }}</text>
      </view>
    </view>

    <!-- 订单动态流卡片 -->
    <view v-if="order?.logs && order.logs.length > 0" class="logs-card">
      <view class="logs-header">
        <text class="logs-title">订单日志</text>
      </view>
      <view class="logs-timeline">
        <view v-for="(log, idx) in order.logs" :key="idx" class="log-item">
          <view class="log-dot"></view>
          <view class="log-info">
            <text class="log-remark">{{ log.remark }}</text>
            <text class="log-time">{{ formatDate(log.created_at) }}</text>
          </view>
        </view>
      </view>
    </view>

    <!-- 底部操作悬浮栏 -->
    <OrderActionBar
      mode="bottom"
      :status="order?.status"
      :can-pay="canPay"
      @contact="handleContact"
      @cancel="handleCancel"
      @pay="handlePay"
      @confirm="handleConfirm"
    />
  </view>
</template>

<script setup lang="ts">
import { ref, computed, onBeforeUnmount } from "vue"
import { onLoad, onShow, onUnload } from "@dcloudio/uni-app"
import { getOrderDetail, cancelOrder, confirmOrder } from "@/api/order"
import { createPayment, mockPay } from "@/api/payment"
import type { OrderDetailResponse } from "@/api/order"
import OrderGoodsList from "@/components/OrderGoodsList.vue"
import OrderAmountSummary from "@/components/OrderAmountSummary.vue"
import OrderActionBar from "@/components/OrderActionBar.vue"
import OrderPaymentCountdown from "@/components/OrderPaymentCountdown.vue"
import { syncOrderCountdownClock } from "@/composables/useOrderCountdown"

const orderID = ref<number>(0)
const order = ref<OrderDetailResponse | null>(null)

// 倒计时归零后退避确认状态
const isConfirming = ref<boolean>(false)
const confirmFailed = ref<boolean>(false)
let retryTimer: ReturnType<typeof setTimeout> | null = null
let isUnmounted = false
const retryDelays = [2000, 5000, 10000]
let retryAttempt = 0

const canPay = computed(() => {
  if (isConfirming.value || confirmFailed.value) return false
  if (!order.value) return false
  return order.value.can_pay ?? (order.value.status === 10)
})

const showCountdownSection = computed(() => {
  if (!order.value) return false
  if (isConfirming.value || confirmFailed.value) return true
  return order.value.status === 10 && !!order.value.payment_expires_at
})

const canApplyAfterSale = computed(() => {
  const s = order.value?.status
  return s === 20 || s === 30 || s === 40
})

function handleApplyAfterSale(): void {
  if (!order.value) return
  uni.navigateTo({
    url: `/pages/aftersale/apply?order_id=${order.value.id}`,
  })
}

const statusDescription = computed(() => {
  switch (order.value?.status) {
    case 10:
      return "请尽快完成支付"
    case 20:
      return "商家正在为您打包配货，请耐心等待"
    case 30:
      return "商品已发货，请留意物流配送信息"
    case 40:
      return "订单已完成，感谢您的惠顾"
    case 50:
      return "订单已取消，已扣减库存已恢复"
    default:
      return ""
  }
})

const statusIcon = computed(() => {
  switch (order.value?.status) {
    case 10:
      return "💳"
    case 20:
      return "📦"
    case 30:
      return "🚚"
    case 40:
      return "🎉"
    case 50:
      return "❌"
    default:
      return "📋"
  }
})

onLoad((query: any) => {
  if (query?.id) {
    orderID.value = Number(query.id)
    loadDetail()
  }
})

onShow(() => {
  syncOrderCountdownClock()
  if (orderID.value > 0 && order.value?.status === 10) {
    loadDetail(false)
  }
})

onUnload(() => {
  isUnmounted = true
  clearRetryTimer()
})

onBeforeUnmount(() => {
  isUnmounted = true
  clearRetryTimer()
})

function clearRetryTimer() {
  if (retryTimer) {
    clearTimeout(retryTimer)
    retryTimer = null
  }
}

/**
 * 倒计时归零时触发：进入确认状态，启动有限退避重查（2s, 5s, 10s 最多3次）
 */
function handleOrderTimeout() {
  if (isUnmounted || !order.value || order.value.status !== 10) return
  if (isConfirming.value) return

  isConfirming.value = true
  confirmFailed.value = false
  retryAttempt = 0
  scheduleNextRetry()
}

/**
 * 手动点击“刷新状态”
 */
function handleManualRetry() {
  clearRetryTimer()
  isConfirming.value = true
  confirmFailed.value = false
  retryAttempt = 0
  checkOrderStatusDirectly()
}

function scheduleNextRetry() {
  clearRetryTimer()
  if (isUnmounted) return
  if (retryAttempt >= retryDelays.length) {
    isConfirming.value = false
    confirmFailed.value = true
    return
  }

  const delay = retryDelays[retryAttempt]
  retryAttempt++
  retryTimer = setTimeout(() => {
    checkOrderStatusDirectly()
  }, delay)
}

async function checkOrderStatusDirectly() {
  if (isUnmounted || orderID.value <= 0) return
  try {
    const res = await getOrderDetail(orderID.value)
    if (isUnmounted) return
    if (res && res.code === 1 && res.data) {
      order.value = res.data
      if (res.data.status !== 10) {
        clearRetryTimer()
        isConfirming.value = false
        confirmFailed.value = false
        return
      }
    }
  } catch (e) {
    // 忽略单次网络波动，由后续重试覆盖
  }

  scheduleNextRetry()
}

async function loadDetail(showMask = true): Promise<void> {
  if (orderID.value <= 0) return

  if (showMask) {
    uni.showLoading({ title: "加载中...", mask: true })
  }
  try {
    const res = await getOrderDetail(orderID.value)
    if (res && res.code === 1 && res.data) {
      order.value = res.data
      if (res.data.status !== 10) {
        isConfirming.value = false
        confirmFailed.value = false
        clearRetryTimer()
      }
    }
  } catch (err: any) {
    uni.showToast({ title: err?.message || "加载订单详情失败", icon: "none" })
  } finally {
    if (showMask) {
      uni.hideLoading()
    }
  }
}

function formatDate(dateStr: string): string {
  if (!dateStr) return "-"
  const d = new Date(dateStr)
  if (isNaN(d.getTime())) return dateStr
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, "0")
  const day = String(d.getDate()).padStart(2, "0")
  const hh = String(d.getHours()).padStart(2, "0")
  const mm = String(d.getMinutes()).padStart(2, "0")
  const ss = String(d.getSeconds()).padStart(2, "0")
  return `${y}-${m}-${day} ${hh}:${mm}:${ss}`
}

function handleCopy(text: string): void {
  if (!text) return
  uni.setClipboardData({
    data: text,
    success: () => {
      uni.showToast({ title: "复制成功", icon: "none" })
    },
  })
}

function handleCancel(): void {
  const currentOrder = order.value
  if (!currentOrder) return
  uni.showModal({
    title: "取消订单确认",
    content: "确定取消此订单吗？取消后库存将自动恢复。",
    confirmColor: "#e02e24",
    success: async (res) => {
      if (res.confirm) {
        uni.showLoading({ title: "正在取消...", mask: true })
        try {
          await cancelOrder(currentOrder.id)
          uni.showToast({ title: "订单已取消", icon: "success" })
          loadDetail()
        } catch (err: any) {
          uni.showToast({ title: err?.message || "取消失败", icon: "none" })
        } finally {
          uni.hideLoading()
        }
      }
    },
  })
}

function handlePay(): void {
  const currentOrder = order.value
  if (!currentOrder || !canPay.value) {
    uni.showToast({ title: "订单支付已超时，正在确认状态", icon: "none" })
    return
  }
  uni.showModal({
    title: "模拟支付确认",
    content: `确认支付订单 ¥${currentOrder.pay_amount} 吗？`,
    confirmText: "确认支付",
    confirmColor: "#e02e24",
    success: async (res) => {
      if (res.confirm) {
        uni.showLoading({ title: "正在支付...", mask: true })
        try {
          // 1. 创建或获取支付单
          const payRes = await createPayment({ order_id: currentOrder.id, channel: "mock" })
          const paymentNo = payRes?.data?.payment_no
          // 2. 模拟支付成功
          await mockPay({ payment_no: paymentNo, order_id: currentOrder.id, channel: "mock" })
          uni.showToast({ title: "支付成功", icon: "success" })
          setTimeout(() => {
            loadDetail()
          }, 600)
        } catch (err: any) {
          uni.showToast({ title: err?.message || "支付失败", icon: "none" })
          loadDetail(false)
        } finally {
          uni.hideLoading()
        }
      }
    },
  })
}

function handleConfirm(): void {
  const currentOrder = order.value
  if (!currentOrder) return
  uni.showModal({
    title: "确认收货",
    content: "确认已收到该订单的所有商品吗？",
    confirmColor: "#e02e24",
    success: async (res) => {
      if (res.confirm) {
        uni.showLoading({ title: "处理中...", mask: true })
        try {
          await confirmOrder(currentOrder.id)
          uni.showToast({ title: "已确认收货", icon: "success" })
          loadDetail()
        } catch (err: any) {
          uni.showToast({ title: err?.message || "确认收货失败", icon: "none" })
        } finally {
          uni.hideLoading()
        }
      }
    },
  })
}

function handleContact(): void {
  uni.showToast({ title: "客服热线：400-888-9999", icon: "none" })
}
</script>

<style lang="scss" scoped>
.order-detail-page {
  min-height: 100vh;
  background-color: #f7f7f7;
  padding-bottom: 140rpx;
}

/* 待付款倒计时卡片 */
.countdown-card {
  margin: 20rpx 24rpx 0;
}

/* 顶部状态横幅 */
.status-banner {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 40rpx 36rpx;
  color: #ffffff;

  &.banner-status-10 {
    background: linear-gradient(135deg, #faad14, #d48806);
  }
  &.banner-status-20 {
    background: linear-gradient(135deg, #ff6034, #ee0a24);
  }
  &.banner-status-30 {
    background: linear-gradient(135deg, #ff9900, #ff6000);
  }
  &.banner-status-40 {
    background: linear-gradient(135deg, #52c41a, #389e0d);
  }
  &.banner-status-50 {
    background: linear-gradient(135deg, #8c8c8c, #595959);
  }

  .banner-content {
    flex: 1;

    .status-title {
      font-size: 38rpx;
      font-weight: bold;
      display: block;
      margin-bottom: 10rpx;
    }

    .status-desc {
      font-size: 24rpx;
      opacity: 0.9;
    }
  }

  .banner-icon-wrap {
    margin-left: 20rpx;

    .banner-icon {
      font-size: 64rpx;
    }
  }
}

/* 物流卡片 */
.express-card {
  margin: -16rpx 24rpx 20rpx;
  background: #ffffff;
  border-radius: 20rpx;
  padding: 24rpx;
  display: flex;
  align-items: center;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.04);

  .express-icon-wrap {
    width: 64rpx;
    height: 64rpx;
    border-radius: 50%;
    background: #e6f7ff;
    display: flex;
    align-items: center;
    justify-content: center;
    margin-right: 20rpx;

    .express-icon {
      font-size: 32rpx;
    }
  }

  .express-info {
    flex: 1;

    .express-company {
      font-size: 28rpx;
      font-weight: bold;
      color: #333333;
      margin-bottom: 6rpx;
      display: block;
    }

    .express-no-row {
      display: flex;
      align-items: center;

      .express-no {
        font-size: 24rpx;
        color: #666666;
        margin-right: 16rpx;
      }
    }
  }
}

/* 地址卡片 */
.address-card {
  margin: 20rpx 24rpx;
  background: #ffffff;
  border-radius: 20rpx;
  padding: 28rpx 24rpx;
  display: flex;
  align-items: center;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.04);

  .address-icon-wrap {
    width: 64rpx;
    height: 64rpx;
    border-radius: 50%;
    background: #fff1f0;
    display: flex;
    align-items: center;
    justify-content: center;
    margin-right: 20rpx;
    flex-shrink: 0;

    .address-icon {
      font-size: 32rpx;
    }
  }

  .address-content {
    flex: 1;

    .user-row {
      display: flex;
      align-items: center;
      margin-bottom: 8rpx;

      .user-name {
        font-size: 28rpx;
        font-weight: bold;
        color: #333333;
        margin-right: 16rpx;
      }

      .user-phone {
        font-size: 26rpx;
        color: #666666;
      }
    }

    .address-text {
      font-size: 26rpx;
      color: #555555;
      line-height: 1.4;
    }
  }
}

/* 商品清单卡片 */
.items-card {
  margin: 20rpx 24rpx;
  background: #ffffff;
  border-radius: 20rpx;
  padding: 24rpx;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.04);

  .card-header {
    display: flex;
    align-items: center;
    margin-bottom: 24rpx;

    .header-icon {
      font-size: 28rpx;
      margin-right: 12rpx;
    }

    .header-title {
      font-size: 28rpx;
      font-weight: bold;
      color: #333333;
    }
  }

  .aftersale-row {
    display: flex;
    justify-content: flex-end;
    padding-top: 20rpx;
    margin-top: 10rpx;
    border-top: 1rpx solid #f5f5f5;

    .aftersale-btn {
      margin: 0;
      padding: 0 28rpx;
      height: 56rpx;
      line-height: 54rpx;
      font-size: 24rpx;
      border-radius: 28rpx;
      border: 1rpx solid #dddddd;
      background: #ffffff;
      color: #666666;

      &::after {
        border: none;
      }
    }
  }
}

/* 订单信息卡片 */
.order-meta-card {
  margin: 20rpx 24rpx;
  background: #ffffff;
  border-radius: 20rpx;
  padding: 24rpx;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.04);

  .meta-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 16rpx;

    &:last-child {
      margin-bottom: 0;
    }

    .meta-label {
      font-size: 24rpx;
      color: #888888;
      width: 140rpx;
      flex-shrink: 0;
    }

    .meta-val-row {
      display: flex;
      align-items: center;
    }

    .meta-val {
      font-size: 24rpx;
      color: #444444;

      &.remark-text {
        text-align: right;
      }
    }
  }
}

.copy-btn {
  font-size: 20rpx;
  color: #1890ff;
  border: 1rpx solid #91d5ff;
  background: #e6f7ff;
  padding: 2rpx 12rpx;
  border-radius: 6rpx;
  margin-left: 12rpx;
}

/* 日志动态卡片 */
.logs-card {
  margin: 20rpx 24rpx;
  background: #ffffff;
  border-radius: 20rpx;
  padding: 24rpx;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.04);

  .logs-header {
    margin-bottom: 16rpx;
    .logs-title {
      font-size: 26rpx;
      font-weight: bold;
      color: #333333;
    }
  }

  .logs-timeline {
    .log-item {
      display: flex;
      align-items: flex-start;
      margin-bottom: 16rpx;

      &:last-child {
        margin-bottom: 0;
      }

      .log-dot {
        width: 12rpx;
        height: 12rpx;
        border-radius: 50%;
        background: #e02e24;
        margin-top: 10rpx;
        margin-right: 16rpx;
        flex-shrink: 0;
      }

      .log-info {
        flex: 1;

        .log-remark {
          font-size: 24rpx;
          color: #333333;
          display: block;
          margin-bottom: 4rpx;
        }

        .log-time {
          font-size: 20rpx;
          color: #999999;
        }
      }
    }
  }
}
</style>
