<template>
  <view class="category-container">
    <!-- 顶部搜索栏占位 -->
    <view class="search-bar">
      <view class="search-input">
        <text class="search-icon">🔍</text>
        <text class="placeholder-text">搜索商品、品类</text>
      </view>
    </view>

    <!-- 分类主体内容 -->
    <view class="category-body">
      <!-- 左侧分类侧边栏 -->
      <scroll-view scroll-y class="sidebar">
        <view
          v-for="(item, index) in categories"
          :key="index"
          class="sidebar-item"
          :class="{ active: currentCategoryIndex === index }"
          @tap="currentCategoryIndex = index"
        >
          <text class="item-text">{{ item }}</text>
        </view>
      </scroll-view>

      <!-- 右侧商品网格占位 -->
      <scroll-view scroll-y class="sub-category-list">
        <view class="sub-title">
          <text>{{ categories[currentCategoryIndex] }}</text>
        </view>
        <view class="grid-list">
          <view v-for="i in 6" :key="i" class="grid-item">
            <view class="item-icon-box">
              <text class="item-placeholder-emoji">📦</text>
            </view>
            <text class="item-name">{{ categories[currentCategoryIndex] }} {{ i }}</text>
          </view>
        </view>
      </scroll-view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref } from 'vue'

const currentCategoryIndex = ref<number>(0)
const categories = ref<string[]>([
  '数码电子',
  '服装服饰',
  '食品生鲜',
  '家居日用',
  '美妆个护',
  '运动户外',
  '母婴玩具',
  '图书文具',
])
</script>

<style scoped>
.category-container {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background-color: #f7f8fa;
}

.search-bar {
  padding: 16rpx 24rpx;
  background-color: #ffffff;
  border-bottom: 1rpx solid #ebeef5;
}

.search-input {
  display: flex;
  align-items: center;
  background-color: #f2f3f5;
  border-radius: 32rpx;
  padding: 14rpx 24rpx;
  gap: 12rpx;
}

.search-icon {
  font-size: 28rpx;
}

.placeholder-text {
  font-size: 26rpx;
  color: #909399;
}

.category-body {
  display: flex;
  flex: 1;
  overflow: hidden;
}

.sidebar {
  width: 190rpx;
  background-color: #f5f6f8;
  height: 100%;
}

.sidebar-item {
  padding: 32rpx 16rpx;
  text-align: center;
  font-size: 28rpx;
  color: #606266;
  border-left: 6rpx solid transparent;
}

.sidebar-item.active {
  background-color: #ffffff;
  color: #e02e24;
  font-weight: bold;
  border-left-color: #e02e24;
}

.sub-category-list {
  flex: 1;
  background-color: #ffffff;
  padding: 24rpx;
  height: 100%;
  box-sizing: border-box;
}

.sub-title {
  font-size: 30rpx;
  font-weight: 600;
  color: #303133;
  margin-bottom: 24rpx;
  padding-left: 12rpx;
  border-left: 6rpx solid #e02e24;
}

.grid-list {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 24rpx;
}

.grid-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12rpx;
}

.item-icon-box {
  width: 120rpx;
  height: 120rpx;
  background-color: #f8f9fa;
  border-radius: 16rpx;
  display: flex;
  align-items: center;
  justify-content: center;
}

.item-placeholder-emoji {
  font-size: 48rpx;
}

.item-name {
  font-size: 24rpx;
  color: #606266;
  text-align: center;
}
</style>

