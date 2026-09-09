<template>
  <view class="category-container">
    <!-- 顶部搜索栏 -->
    <view class="search-bar">
      <view class="search-input">
        <text class="search-icon">🔍</text>
        <text class="placeholder-text">搜索商品、品类</text>
      </view>
    </view>

    <!-- 分类主体内容 -->
    <view class="category-body">
      <!-- 左侧分类侧边栏 (优先渲染 Bootstrap Categories) -->
      <scroll-view scroll-y class="sidebar">
        <view
          v-for="(item, index) in categoryList"
          :key="item.id"
          class="sidebar-item"
          :class="{ active: currentCategoryIndex === index }"
          @tap="currentCategoryIndex = index"
        >
          <text class="item-text">{{ item.name }}</text>
        </view>
      </scroll-view>

      <!-- 右侧商品/子分类网格 -->
      <scroll-view scroll-y class="sub-category-list">
        <view class="sub-title">
          <text>{{ currentCategoryName }}</text>
        </view>

        <!-- 当前分类商品展示占位 -->
        <view class="grid-list">
          <view v-for="i in 6" :key="i" class="grid-item">
            <view class="item-icon-box">
              <image
                v-if="currentCategory?.image_url"
                class="item-real-img"
                :src="currentCategory.image_url"
                mode="aspectFill"
              />
              <text v-else class="item-placeholder-emoji">📦</text>
            </view>
            <text class="item-name">{{ currentCategoryName }} {{ i }}</text>
            <text class="item-price">¥{{ (i * 19.9).toFixed(2) }}</text>
          </view>
        </view>
      </scroll-view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref, computed } from "vue"
import { onShow } from "@dcloudio/uni-app"
import { useBootstrap } from "@/services/bootstrap"
import type { BootstrapCategory } from "@/api/common"

const { bootstrapData, fetchBootstrap } = useBootstrap()

const currentCategoryIndex = ref<number>(0)

const defaultCategories: BootstrapCategory[] = [
  { id: 1, name: "数码电子", image_url: "", parent_id: 0, sort: 1 },
  { id: 2, name: "服装服饰", image_url: "", parent_id: 0, sort: 2 },
  { id: 3, name: "食品生鲜", image_url: "", parent_id: 0, sort: 3 },
  { id: 4, name: "家居日用", image_url: "", parent_id: 0, sort: 4 },
  { id: 5, name: "美妆个护", image_url: "", parent_id: 0, sort: 5 },
  { id: 6, name: "运动户外", image_url: "", parent_id: 0, sort: 6 },
  { id: 7, name: "母婴玩具", image_url: "", parent_id: 0, sort: 7 },
  { id: 8, name: "图书文具", image_url: "", parent_id: 0, sort: 8 },
]

const categoryList = computed<BootstrapCategory[]>(() => {
  const cats = bootstrapData.value?.categories
  if (cats && cats.length > 0) {
    return cats
  }
  return defaultCategories
})

const currentCategory = computed<BootstrapCategory | undefined>(() => {
  return categoryList.value[currentCategoryIndex.value] || categoryList.value[0]
})

const currentCategoryName = computed<string>(() => {
  return currentCategory.value?.name || "分类"
})

onShow(() => {
  // 如果尚未加载启动数据，则请求一次
  if (!bootstrapData.value) {
    fetchBootstrap()
  }

  // 检查是否有从首页点击指定分类传递的 active_category_id
  try {
    const targetId = uni.getStorageSync("active_category_id")
    if (targetId) {
      uni.removeStorageSync("active_category_id")
      const foundIdx = categoryList.value.findIndex((c) => c.id === Number(targetId))
      if (foundIdx !== -1) {
        currentCategoryIndex.value = foundIdx
      }
    }
  } catch {}
})
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
  gap: 8rpx;
}

.item-icon-box {
  width: 120rpx;
  height: 120rpx;
  background-color: #f8f9fa;
  border-radius: 16rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}

.item-real-img {
  width: 120rpx;
  height: 120rpx;
}

.item-placeholder-emoji {
  font-size: 48rpx;
}

.item-name {
  font-size: 24rpx;
  color: #303133;
  text-align: center;
}

.item-price {
  font-size: 22rpx;
  color: #e02e24;
  font-weight: 600;
}
</style>
