<template>
  <view class="container">
    <view class="header">
      <text class="app-title">AI Shop</text>
    </view>

    <view class="content-card">
      <!-- 加载中状态 -->
      <view v-if="state === 'loading'" class="status-box loading">
        <text class="status-text">加载中...</text>
      </view>

      <!-- 成功状态 -->
      <view v-else-if="state === 'success'" class="status-box success">
        <text class="result-text">{{ helloText }}</text>
      </view>

      <!-- 失败状态 -->
      <view v-else-if="state === 'error'" class="status-box error">
        <text class="error-text">数据加载失败</text>
        <button class="retry-btn" @tap="fetchHello">重试</button>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getHello } from '@/api/common'

type FetchState = 'loading' | 'success' | 'error'

const state = ref<FetchState>('loading')
const helloText = ref<string>('')

async function fetchHello(): Promise<void> {
  state.value = 'loading'
  try {
    const res = await getHello()
    helloText.value = res.data.text
    state.value = 'success'
  } catch (err: unknown) {
    console.error('获取 Hello 数据失败:', err)
    state.value = 'error'
  }
}

onMounted(async () => {
  await fetchHello()
})
</script>

<style scoped>
.container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 80vh;
  padding: 40rpx;
}

.header {
  margin-bottom: 60rpx;
}

.app-title {
  font-size: 52rpx;
  font-weight: bold;
  color: #1a1a1a;
  letter-spacing: 2rpx;
}

.content-card {
  width: 90%;
  max-width: 500rpx;
  background-color: #ffffff;
  border-radius: 16rpx;
  padding: 48rpx 32rpx;
  box-shadow: 0 4rpx 20rpx rgba(0, 0, 0, 0.06);
  display: flex;
  flex-direction: column;
  align-items: center;
}

.status-box {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 24rpx;
}

.status-text {
  font-size: 32rpx;
  color: #909399;
}

.result-text {
  font-size: 38rpx;
  font-weight: 600;
  color: #2c3e50;
}

.error-text {
  font-size: 32rpx;
  color: #f56c6c;
}

.retry-btn {
  font-size: 28rpx;
  color: #409eff;
  background-color: #ecf5ff;
  border-radius: 8rpx;
  padding: 8rpx 32rpx;
  border: 1rpx solid #d9ecff;
  line-height: 2;
}

.retry-btn::after {
  border: none;
}
</style>
