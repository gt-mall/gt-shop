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
      v-if="status === 10"
      class="action-btn primary-btn"
      @tap="emit('pay', order)"
    >
      {{ computedPayText }}
    </button>

    <!-- 待收货 (30): 确认收货 -->
    <button
      v-if="status === 30"
      class="action-btn primary-btn"
      @tap="emit('confirm', order)"
    >
      确认收货
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
  showContact?: boolean
  showDetail?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  status: 0,
  mode: "bottom",
  order: undefined,
  payText: "",
  showContact: undefined,
  showDetail: undefined,
})

const emit = defineEmits<{
  (e: "cancel", order?: any): void
  (e: "pay", order?: any): void
  (e: "confirm", order?: any): void
  (e: "detail", order?: any): void
  (e: "contact", order?: any): void
}>()

const computedPayText = computed(() => {
  if (props.payText) return props.payText
  return props.mode === "bottom" ? "立即支付" : "去支付"
})
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
      }
    }
  }
}
</style>
