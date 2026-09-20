<template>
  <view v-if="!isOnline" class="network-banner">
    <view class="banner-left">
      <text class="banner-icon">📡</text>
      <text class="banner-text">当前网络不可用，请检查网络设置</text>
    </view>
    <button v-if="showRetry" class="retry-btn" @tap="handleRetry">
      重试
    </button>
  </view>
</template>

<script setup lang="ts">
import { isOnline, checkNetworkStatus } from "@/services/network"

interface Props {
  showRetry?: boolean
}

withDefaults(defineProps<Props>(), {
  showRetry: true,
})

const emit = defineEmits<{
  (e: "retry"): void
}>()

async function handleRetry() {
  await checkNetworkStatus()
  emit("retry")
}
</script>

<style lang="scss" scoped>
.network-banner {
  width: 100%;
  background-color: #fff2f0;
  border-bottom: 1rpx solid #ffccc7;
  padding: 16rpx 28rpx;
  box-sizing: border-box;
  display: flex;
  align-items: center;
  justify-content: space-between;
  z-index: 100;

  .banner-left {
    display: flex;
    align-items: center;
    gap: 12rpx;
    flex: 1;

    .banner-icon {
      font-size: 28rpx;
      line-height: 1;
    }

    .banner-text {
      font-size: 24rpx;
      color: #cf1322;
      line-height: 1.4;
    }
  }

  .retry-btn {
    margin: 0;
    padding: 0 20rpx;
    height: 48rpx;
    line-height: 46rpx;
    font-size: 22rpx;
    color: #cf1322;
    background-color: #ffffff;
    border: 1rpx solid #ffccc7;
    border-radius: 24rpx;

    &::after {
      border: none;
    }
  }
}
</style>
