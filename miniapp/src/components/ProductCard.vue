<template>
  <view
    class="product-card"
    :class="[layout === 'row' ? 'card-row' : 'card-grid']"
    @tap="handleClick"
  >
    <!-- 商品图片 -->
    <view class="img-wrapper">
      <image
        v-if="item.main_image_url && !imageFailed"
        class="product-img"
        :src="item.main_image_url"
        mode="aspectFill"
        lazy-load
        @error="imageFailed = true"
      />
      <view v-else class="img-placeholder">
        <text class="placeholder-icon">商品</text>
      </view>
    </view>

    <!-- 商品信息区 -->
    <view class="info-wrapper">
      <view class="title-area">
        <text class="product-name">{{ item.name }}</text>
        <text v-if="item.subtitle" class="product-subtitle">{{ item.subtitle }}</text>
      </view>

      <view class="bottom-area">
        <view class="price-box">
          <text class="currency">¥</text>
          <text class="price-val">{{ item.price }}</text>
          <text v-if="showMarketPrice" class="market-price">¥{{ item.market_price }}</text>
        </view>
        <text class="sales-text">已售 {{ item.sales }}</text>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref, computed, watch } from "vue"
import type { ProductListItem } from "@/api/product"
import { yuanToFen } from "@/utils/money"

const props = withDefaults(
  defineProps<{
    item: ProductListItem
    layout?: "grid" | "row"
  }>(),
  {
    layout: "grid",
  }
)

const imageFailed = ref(false)

watch(
  () => [props.item.id, props.item.main_image_url],
  () => {
    imageFailed.value = false
  }
)

const showMarketPrice = computed(() => {
  if (!props.item.market_price) return false
  return yuanToFen(props.item.market_price) > yuanToFen(props.item.price)
})

function handleClick(): void {
  uni.navigateTo({
    url: `/pages/product/detail?id=${props.item.id}`,
  })
}
</script>

<style scoped>
.product-card {
  background-color: #ffffff;
  border-radius: 16rpx;
  overflow: hidden;
  box-shadow: 0 2rpx 10rpx rgba(0, 0, 0, 0.04);
}

/* 垂直卡片网格布局 (用于首页瀑布流) */
.card-grid {
  display: flex;
  flex-direction: column;
}

.card-grid .img-wrapper {
  width: 100%;
  height: 330rpx;
  background-color: #f7f8fa;
  position: relative;
}

.card-grid .product-img {
  width: 100%;
  height: 100%;
  display: block;
}

.card-grid .info-wrapper {
  padding: 16rpx 16rpx 20rpx;
  display: flex;
  flex-direction: column;
  flex: 1;
  justify-content: space-between;
}

.card-grid .title-area {
  margin-bottom: 12rpx;
}

.card-grid .product-name {
  font-size: 28rpx;
  color: #303133;
  font-weight: 500;
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  word-break: break-all;
}

.card-grid .product-subtitle {
  font-size: 22rpx;
  color: #909399;
  line-height: 1.3;
  margin-top: 6rpx;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  display: block;
}

.card-grid .bottom-area {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
}

/* 水平单行列表布局 (用于分类页右侧商品栏) */
.card-row {
  display: flex;
  flex-direction: row;
  padding: 16rpx;
  margin-bottom: 16rpx;
}

.card-row .img-wrapper {
  width: 160rpx;
  height: 160rpx;
  border-radius: 12rpx;
  overflow: hidden;
  background-color: #f7f8fa;
  flex-shrink: 0;
}

.card-row .product-img {
  width: 100%;
  height: 100%;
  display: block;
}

.card-row .info-wrapper {
  flex: 1;
  margin-left: 20rpx;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  min-width: 0;
}

.card-row .product-name {
  font-size: 26rpx;
  color: #303133;
  font-weight: 500;
  line-height: 1.35;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  word-break: break-all;
}

.card-row .product-subtitle {
  font-size: 22rpx;
  color: #909399;
  margin-top: 4rpx;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  display: block;
}

.card-row .bottom-area {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin-top: 8rpx;
}

/* 通用价格与销量样式 */
.price-box {
  display: flex;
  align-items: baseline;
}

.currency {
  font-size: 22rpx;
  color: #e02e24;
  font-weight: bold;
}

.price-val {
  font-size: 32rpx;
  color: #e02e24;
  font-weight: bold;
  line-height: 1;
}

.market-price {
  font-size: 20rpx;
  color: #c0c4cc;
  text-decoration: line-through;
  margin-left: 8rpx;
}

.sales-text {
  font-size: 20rpx;
  color: #909399;
}

/* 图片占位 */
.img-placeholder {
  width: 100%;
  height: 100%;
  background-color: #f2f3f5;
  display: flex;
  align-items: center;
  justify-content: center;
}

.placeholder-icon {
  font-size: 24rpx;
  color: #c0c4cc;
}
</style>
