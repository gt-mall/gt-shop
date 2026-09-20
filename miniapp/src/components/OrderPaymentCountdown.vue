<template>
  <view v-if="hasCountdown || isConfirming || confirmFailed" class="order-countdown-wrap" :class="{ 'is-compact': compact }">
    <!-- 1. 正在向服务器确认订单超时取消状态 -->
    <view v-if="isConfirming" class="countdown-status-tip confirming" :class="{ 'tip-compact': compact }">
      <text class="tip-icon">⏳</text>
      <text class="tip-text">{{ compact ? '确认中...' : '支付已超时，正在确认订单状态...' }}</text>
    </view>

    <!-- 2. 确认状态失败（网络波动等），提供手动重试入口 -->
    <view v-else-if="confirmFailed" class="countdown-status-tip failed" :class="{ 'tip-compact': compact }">
      <text class="tip-icon">⚠️</text>
      <text class="tip-text">{{ compact ? '确认超时' : '订单状态确认超时' }}</text>
      <text class="retry-action" @tap.stop="handleRetry">刷新状态</text>
    </view>

    <!-- 3. 正常倒计时展示 (未归零) -->
    <template v-else-if="!isExpired">
      <!-- A. 列表卡片紧凑模式 (Compact) -->
      <view v-if="compact" class="countdown-compact">
        <text class="compact-icon">⏳</text>
        <text class="compact-label">剩余</text>
        <text class="compact-digits">{{ countdown.hours !== '00' ? countdown.hours + ':' : '' }}{{ countdown.minutes }}:{{ countdown.seconds }}</text>
      </view>

      <!-- B. 详情页置顶横幅模式 (Banner) -->
      <view v-else class="countdown-banner">
        <view class="banner-left">
          <text class="banner-icon">⏳</text>
          <text class="banner-title">等待买家付款</text>
        </view>
        <view class="banner-right">
          <text class="banner-text">剩 </text>
          <text class="banner-highlight">{{ countdown.hours !== '00' ? countdown.hours + '小时' : '' }}{{ countdown.minutes }}分{{ countdown.seconds }}秒</text>
          <text class="banner-text"> 订单自动关闭</text>
        </view>
      </view>
    </template>
  </view>
</template>

<script setup lang="ts">
import { watch } from 'vue'
import { useOrderCountdown } from '@/composables/useOrderCountdown'

interface Props {
  expiresAt?: string | null
  serverTime?: string | null
  status?: number
  compact?: boolean
  isConfirming?: boolean
  confirmFailed?: boolean
  paused?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  expiresAt: null,
  serverTime: null,
  status: 10,
  compact: false,
  isConfirming: false,
  confirmFailed: false,
  paused: false,
})

const emit = defineEmits<{
  (e: 'timeout'): void
  (e: 'retry'): void
}>()

const { countdown, isExpired, hasCountdown, resetTimeoutFlag } = useOrderCountdown(() => ({
  expiresAt: props.expiresAt,
  serverTime: props.serverTime,
  status: props.status,
  paused: props.paused,
  onTimeout: () => {
    emit('timeout')
  },
}))

// 当服务端刷新后更新了 expiresAt 或 status，重置 timeout 标志
watch(
  () => [props.expiresAt, props.status],
  () => {
    resetTimeoutFlag()
  }
)

function handleRetry() {
  emit('retry')
}
</script>

<style scoped>
.order-countdown-wrap {
  width: 100%;
  box-sizing: border-box;
}

/* 详情页模式横幅 */
.countdown-banner {
  background: linear-gradient(135deg, #fff7e6 0%, #ffe7ba 100%);
  border: 1rpx solid #ffd591;
  border-radius: 16rpx;
  padding: 20rpx 28rpx;
  margin-bottom: 20rpx;
  display: flex;
  align-items: center;
  justify-content: space-between;
  box-shadow: 0 2rpx 8rpx rgba(250, 140, 22, 0.08);
}

.banner-left {
  display: flex;
  align-items: center;
  gap: 12rpx;
}

.banner-icon {
  font-size: 32rpx;
}

.banner-title {
  font-size: 28rpx;
  font-weight: 600;
  color: #d46b08;
}

.banner-right {
  display: flex;
  align-items: baseline;
}

.banner-text {
  font-size: 24rpx;
  color: #873800;
}

.banner-highlight {
  font-size: 28rpx;
  font-weight: bold;
  color: #e02e24;
  margin: 0 4rpx;
  font-family: monospace;
}

/* 卡片紧凑模式 */
.countdown-compact {
  display: inline-flex;
  align-items: center;
  background-color: #fff7e6;
  border: 1rpx solid #ffd591;
  border-radius: 8rpx;
  padding: 4rpx 14rpx;
  gap: 8rpx;
}

.compact-icon {
  font-size: 22rpx;
  line-height: 1;
}

.compact-label {
  font-size: 22rpx;
  color: #d46b08;
}

.compact-digits {
  font-size: 22rpx;
  font-weight: bold;
  color: #e02e24;
  font-family: monospace;
}

/* 超时确认中 / 失败提示 */
.countdown-status-tip {
  display: flex;
  align-items: center;
  padding: 12rpx 20rpx;
  border-radius: 12rpx;
  font-size: 24rpx;

  &.tip-compact {
    padding: 4rpx 12rpx;
    font-size: 20rpx;
    border-radius: 8rpx;

    .tip-icon {
      font-size: 20rpx;
      margin-right: 6rpx;
    }

    .retry-action {
      font-size: 20rpx;
      margin-left: 8rpx;
    }
  }
}

.countdown-status-tip.confirming {
  background-color: #f0f5ff;
  border: 1rpx solid #adc6ff;
  color: #1d39c4;
}

.countdown-status-tip.failed {
  background-color: #fff1f0;
  border: 1rpx solid #ffa39e;
  color: #cf1322;
  justify-content: space-between;
}

.tip-icon {
  font-size: 26rpx;
  margin-right: 12rpx;
}

.tip-text {
  flex: 1;
}

.retry-action {
  color: #1890ff;
  font-weight: 500;
  margin-left: 16rpx;
  cursor: pointer;
  text-decoration: underline;
}
</style>
