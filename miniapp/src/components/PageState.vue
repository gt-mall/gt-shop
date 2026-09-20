<template>
  <view class="page-state-wrapper" :style="{ minHeight: minHeight }">
    <!-- 1. 加载态 -->
    <view v-if="state === 'loading'" class="state-container loading-state">
      <slot name="loading">
        <view class="loading-spinner" />
        <text v-if="loadingText" class="state-title">{{ loadingText }}</text>
      </slot>
    </view>

    <!-- 2. 空态 (明确真实空数据) -->
    <view v-else-if="state === 'empty'" class="state-container empty-state">
      <slot name="empty">
        <text class="state-icon">{{ emptyIcon }}</text>
        <text class="state-title">{{ emptyText }}</text>
        <text v-if="emptySubText" class="state-sub">{{ emptySubText }}</text>
        <button
          v-if="emptyActionText"
          class="state-btn empty-btn"
          @tap="emit('empty-action')"
        >
          {{ emptyActionText }}
        </button>
      </slot>
    </view>

    <!-- 3. 错误态 (网络异常/服务端错误，支持重试) -->
    <view v-else-if="state === 'error'" class="state-container error-state">
      <slot name="error">
        <text class="state-icon">{{ errorIcon }}</text>
        <text class="state-title">{{ errorText }}</text>
        <text v-if="errorSubText" class="state-sub">{{ errorSubText }}</text>
        <button
          v-if="retryText"
          class="state-btn retry-btn"
          @tap="emit('retry')"
        >
          {{ retryText }}
        </button>
      </slot>
    </view>

    <!-- 4. 内容态 (正常渲染) -->
    <view v-else class="content-view">
      <slot />
    </view>
  </view>
</template>

<script setup lang="ts">
interface Props {
  state?: "loading" | "empty" | "error" | "content"
  loadingText?: string
  emptyText?: string
  emptySubText?: string
  emptyIcon?: string
  emptyActionText?: string
  errorText?: string
  errorSubText?: string
  errorIcon?: string
  retryText?: string
  minHeight?: string
}

withDefaults(defineProps<Props>(), {
  state: "content",
  loadingText: "加载中...",
  emptyText: "暂无相关数据",
  emptySubText: "",
  emptyIcon: "📦",
  emptyActionText: "",
  errorText: "网络连接异常，请重试",
  errorSubText: "",
  errorIcon: "⚠️",
  retryText: "点击重试",
  minHeight: "400rpx",
})

const emit = defineEmits<{
  (e: "retry"): void
  (e: "empty-action"): void
}>()
</script>

<style lang="scss" scoped>
.page-state-wrapper {
  width: 100%;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
}

.state-container {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 80rpx 40rpx;
  box-sizing: border-box;
}

.state-icon {
  font-size: 72rpx;
  margin-bottom: 20rpx;
  line-height: 1;
}

.state-title {
  font-size: 28rpx;
  color: #666666;
  margin-bottom: 12rpx;
  text-align: center;
}

.state-sub {
  font-size: 24rpx;
  color: #999999;
  margin-bottom: 24rpx;
  text-align: center;
  line-height: 1.4;
}

.state-btn {
  height: 64rpx;
  line-height: 62rpx;
  padding: 0 36rpx;
  font-size: 26rpx;
  border-radius: 32rpx;
  margin-top: 16rpx;
  box-sizing: border-box;

  &::after {
    border: none;
  }

  &.retry-btn {
    background-color: #e02e24;
    color: #ffffff;
  }

  &.empty-btn {
    background-color: #ffffff;
    color: #333333;
    border: 1rpx solid #dddddd;
  }
}

.loading-spinner {
  width: 48rpx;
  height: 48rpx;
  border: 4rpx solid #f0f0f0;
  border-top-color: #e02e24;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
  margin-bottom: 20rpx;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.content-view {
  width: 100%;
}
</style>
