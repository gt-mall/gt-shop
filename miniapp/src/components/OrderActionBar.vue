<template>
  <view
    class="order-action-bar"
    :class="mode === 'bottom' ? 'bottom-bar' : 'card-bar'"
    @tap.stop
  >
    <!-- 底部悬浮模式: 联系客服按钮 (居左) -->
    <button
      v-if="mode === 'bottom' && (showContact !== undefined ? showContact : true)"
      class="action-btn contact-btn"
      @tap="emit('contact', order)"
    >
      <text class="btn-icon">💬</text>
      <text class="btn-text">联系客服</text>
    </button>

    <!-- 卡片模式: 查看详情按钮 -->
    <button
      v-if="mode === 'card' && (showDetail !== undefined ? showDetail : true)"
      class="action-btn outline-btn"
      @tap="emit('detail', order)"
    >
      查看详情
    </button>

    <!-- 待支付 (10): 取消订单 / 支付 -->
    <button
      v-if="status === 10"
      class="action-btn warn-btn"
      @tap="emit('cancel', order)"
    >
      取消订单
    </button>
    <button
      v-if="status === 10 && showPay"
      class="action-btn primary-btn"
      :disabled="isPayDisabled"
      :class="{ disabled: isPayDisabled }"
      @tap="handlePay"
    >
      {{ computedPayText }}
    </button>

    <button
      v-if="status === 20 && canCancelPaid"
      class="action-btn warn-btn"
      @tap="emit('cancel', order)"
    >
      申请取消退款
    </button>

    <!-- 待收货 (30): 确认收货 -->
    <button
      v-if="status === 30"
      class="action-btn primary-btn"
      @tap="emit('confirm', order)"
    >
      确认收货
    </button>

    <!-- 已完成 (40): 去评价 -->
    <button
      v-if="status === 40 && showReview"
      class="action-btn primary-btn"
      @tap="emit('review', order)"
    >
      去评价
    </button>
  </view>
</template>

<script setup lang="ts">
import { computed } from "vue"

interface Props {
  status?: number
  mode?: "bottom" | "card"
  order?: any
  payText?: string
  canPay?: boolean
  showPay?: boolean
  canCancelPaid?: boolean
  showReview?: boolean
  showContact?: boolean
  showDetail?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  status: 0,
  mode: "bottom",
  order: undefined,
  payText: "",
  canPay: undefined,
  showPay: true,
  canCancelPaid: false,
  showReview: true,
  showContact: undefined,
  showDetail: undefined,
})

const emit = defineEmits<{
  (e: "cancel", order?: any): void
  (e: "pay", order?: any): void
  (e: "confirm", order?: any): void
  (e: "detail", order?: any): void
  (e: "contact", order?: any): void
  (e: "review", order?: any): void
}>()

const isPayDisabled = computed(() => {
  if (props.canPay !== undefined) {
    return !props.canPay
  }
  if (props.order) {
    if (props.status === 10 || props.order.status === 10) {
      if (props.order.can_pay === false) return true
      if (!props.order.payment_expires_at) return true
      if (props.order.can_pay === true) return false
      // 未明确标识可支付且未校准时间，必须先刷新状态
      return true
    }
  }
  return false
})

const computedPayText = computed(() => {
  if (isPayDisabled.value) {
    if (props.payText) return props.payText
    if (props.order && !props.order.payment_expires_at) return "需刷新状态"
    return "已超时"
  }
  if (props.payText) return props.payText
  return props.mode === "bottom" ? "立即支付" : "去支付"
})

function handlePay() {
  if (isPayDisabled.value) {
    uni.showToast({ title: "订单支付已失效或需刷新确认状态", icon: "none" })
    return
  }
  emit("pay", props.order)
}
</script>

<style lang="scss" scoped>
.order-action-bar {
  /* 底部固定操作栏 (detail.vue) */
  &.bottom-bar {
    position: fixed;
    bottom: 0;
    left: 0;
    right: 0;
    height: 110rpx;
    background-color: #ffffff;
    display: flex;
    align-items: center;
    justify-content: flex-end;
    padding: 0 32rpx;
    gap: 20rpx;
    box-shadow: 0 -4rpx 16rpx rgba(0, 0, 0, 0.05);
    z-index: 99;

    .action-btn {
      margin: 0;
      padding: 0 32rpx;
      height: 72rpx;
      line-height: 70rpx;
      font-size: 26rpx;
      border-radius: 36rpx;
      border: 2rpx solid transparent;
      box-sizing: border-box;

      &::after {
        border: none;
      }

      &.contact-btn {
        margin-right: auto;
        background: transparent;
        padding: 0;
        display: flex;
        align-items: center;
        color: #666666;
        font-size: 24rpx;

        .btn-icon {
          margin-right: 8rpx;
          font-size: 28rpx;
        }
      }

      &.warn-btn {
        background: #fff1f0;
        border-color: #ffa39e;
        color: #e02e24;
      }

      &.primary-btn {
        background: #e02e24;
        color: #ffffff;

        &.disabled,
        &[disabled] {
          background: #dcdfe6;
          color: #ffffff;
          border-color: transparent;
          opacity: 0.8;
          cursor: not-allowed;
        }
      }
    }
  }

  /* 订单列表卡片模式 (index.vue) */
  &.card-bar {
    display: flex;
    justify-content: flex-end;
    align-items: center;
    gap: 16rpx;
    padding-top: 16rpx;
    border-top: 2rpx solid #f5f5f5;

    .action-btn {
      margin: 0;
      padding: 0 24rpx;
      height: 56rpx;
      line-height: 54rpx;
      font-size: 24rpx;
      border-radius: 28rpx;
      border: 2rpx solid transparent;
      box-sizing: border-box;

      &::after {
        border: none;
      }

      &.outline-btn {
        background: #ffffff;
        border-color: #dddddd;
        color: #666666;
      }

      &.warn-btn {
        background: #fff1f0;
        border-color: #ffa39e;
        color: #e02e24;
      }

      &.primary-btn {
        background: #e02e24;
        color: #ffffff;

        &.disabled,
        &[disabled] {
          background: #dcdfe6;
          color: #ffffff;
          border-color: transparent;
          opacity: 0.8;
          cursor: not-allowed;
        }
      }
    }
  }
}
</style>
