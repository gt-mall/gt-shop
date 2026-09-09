<template>
  <view class="cart-container">
    <!-- 空购物车占位状态 -->
    <view class="empty-state">
      <view class="empty-icon-wrap">
        <text class="empty-emoji">🛒</text>
      </view>
      <text class="empty-title">购物车空空如也</text>
      <text class="empty-subtitle">快去挑选心仪的商品吧</text>
      <button class="go-home-btn" @tap="goToHome">去逛逛</button>
    </view>
  </view>
</template>

<script setup lang="ts">
function goToHome() {
  uni.switchTab({
    url: '/pages/index/index',
  })
}
</script>

<style scoped>
.cart-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 80vh;
  padding: 40rpx;
  box-sizing: border-box;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
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
</style>

