<template>
  <view class="review-detail-page">
    <view v-if="loading" class="state-container">
      <text class="state-text">正在加载评价详情...</text>
    </view>

    <view v-else-if="!review" class="state-container">
      <text class="state-text">未找到评价详情</text>
      <button class="back-btn" @tap="goBack">返回</button>
    </view>

    <view v-else class="content-container">
      <!-- 1. 状态横幅 -->
      <view class="status-banner" :class="'status-banner-' + review.status">
        <view class="banner-icon-wrap">
          <text class="banner-icon">{{ statusIcon }}</text>
        </view>
        <view class="banner-content">
          <text class="status-title">{{ review.status_text || statusTitle }}</text>
          <text class="status-desc">{{ statusDescription }}</text>
        </view>
      </view>

      <!-- 驳回原因重点展示 -->
      <view v-if="review.status === 30 && review.audit_reason" class="card reject-card">
        <text class="reject-title">审核驳回原因：</text>
        <text class="reject-text">{{ review.audit_reason }}</text>
      </view>

      <!-- 2. 商品快照卡片 -->
      <view class="card product-card" @tap="goToProduct(review.product_id)">
        <view class="product-info">
          <text class="product-name">{{ review.product_name }}</text>
          <text v-if="review.sku_spec_values" class="sku-spec">{{ review.sku_spec_values }}</text>
        </view>
        <text class="arrow-icon">›</text>
      </view>

      <!-- 3. 评价详情卡片 -->
      <view class="card detail-card">
        <view class="rating-row">
          <RatingStars :value="review.rating" :size="32" :show-score="true" />
          <text v-if="review.is_anonymous" class="anon-badge">匿名评价</text>
        </view>
        <text class="create-time">发表时间：{{ formatDate(review.created_at) }}</text>

        <view class="review-text-box">
          <text class="review-text">{{ review.content }}</text>
        </view>

        <!-- 晒图展示 -->
        <view v-if="review.images && review.images.length > 0" class="image-grid">
          <image
            v-for="(img, idx) in review.images"
            :key="idx"
            class="review-image"
            :src="img"
            mode="aspectFill"
            @tap="previewImage(idx)"
          />
        </view>
      </view>

      <!-- 4. 商家官方回复 -->
      <view v-if="review.reply_content" class="card reply-card">
        <view class="reply-header">
          <text class="reply-badge">商家回复</text>
          <text v-if="review.reply_at" class="reply-time">{{ formatDate(review.reply_at) }}</text>
        </view>
        <text class="reply-text">{{ review.reply_content }}</text>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref, computed } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { getReviewDetail } from "@/api/review"
import type { MiniappMineReviewItemResponse } from "@/api/review"
import RatingStars from "@/components/RatingStars.vue"

const reviewId = ref(0)
const loading = ref(true)
const review = ref<MiniappMineReviewItemResponse | null>(null)

onLoad((options: any) => {
  if (options?.id) {
    reviewId.value = Number(options.id)
    void fetchDetail()
  } else {
    loading.value = false
  }
})

async function fetchDetail() {
  if (!reviewId.value) return
  loading.value = true
  try {
    const res = await getReviewDetail(reviewId.value)
    if (res) {
      review.value = res
    }
  } catch (err: any) {
    uni.showToast({ title: err?.message || "获取评价详情失败", icon: "none" })
  } finally {
    loading.value = false
  }
}

const statusIcon = computed(() => {
  switch (review.value?.status) {
    case 10:
      return "⏳"
    case 20:
      return "✓"
    case 30:
      return "✕"
    case 40:
      return "👁️"
    default:
      return "📋"
  }
})

const statusTitle = computed(() => {
  switch (review.value?.status) {
    case 10:
      return "审核中"
    case 20:
      return "已公开"
    case 30:
      return "审核未通过"
    case 40:
      return "已隐藏"
    default:
      return "评价详情"
  }
})

const statusDescription = computed(() => {
  switch (review.value?.status) {
    case 10:
      return "您的评价已提交，平台正在审核中。审核通过后将对所有买家公开展示。"
    case 20:
      return "您的评价已审核通过，已公开展示在商品详情中，感谢您的分享！"
    case 30:
      return "很抱歉，您的评价未能通过审核，请查看下方具体驳回原因。"
    case 40:
      return "该评价已被隐藏，仅作者自己可见。"
    default:
      return ""
  }
})

function formatDate(dateStr?: string | null) {
  if (!dateStr) return ""
  try {
    const d = new Date(dateStr)
    if (isNaN(d.getTime())) return dateStr
    const y = d.getFullYear()
    const m = String(d.getMonth() + 1).padStart(2, "0")
    const day = String(d.getDate()).padStart(2, "0")
    const hh = String(d.getHours()).padStart(2, "0")
    const mm = String(d.getMinutes()).padStart(2, "0")
    return `${y}-${m}-${day} ${hh}:${mm}`
  } catch {
    return dateStr
  }
}

function previewImage(idx: number) {
  if (!review.value?.images) return
  uni.previewImage({
    current: review.value.images[idx],
    urls: review.value.images,
  })
}

function goToProduct(productId?: number) {
  if (!productId) return
  uni.navigateTo({
    url: `/pages/product/detail?id=${productId}`,
  })
}

function goBack() {
  uni.navigateBack()
}
</script>

<style lang="scss" scoped>
.review-detail-page {
  min-height: 100vh;
  background-color: #f7f8fa;
  padding-bottom: 60rpx;
}

.state-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 120rpx 40rpx;

  .state-text {
    font-size: 28rpx;
    color: #999999;
    margin-bottom: 24rpx;
  }

  .back-btn {
    height: 64rpx;
    line-height: 62rpx;
    padding: 0 32rpx;
    font-size: 26rpx;
    border-radius: 32rpx;
    background: #ffffff;
    border: 1rpx solid #dddddd;
    color: #666666;
  }
}

.status-banner {
  display: flex;
  align-items: center;
  padding: 32rpx 28rpx;
  gap: 20rpx;

  .banner-icon-wrap {
    width: 72rpx;
    height: 72rpx;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 36rpx;
    flex-shrink: 0;
  }

  .banner-content {
    display: flex;
    flex-direction: column;
    gap: 6rpx;

    .status-title {
      font-size: 32rpx;
      font-weight: bold;
    }

    .status-desc {
      font-size: 24rpx;
      line-height: 1.4;
      opacity: 0.9;
    }
  }

  &.status-banner-10 {
    background: #fff7e6;
    color: #d46b08;
    .banner-icon-wrap {
      background: #ffd591;
    }
  }

  &.status-banner-20 {
    background: #f6ffed;
    color: #389e0d;
    .banner-icon-wrap {
      background: #b7eb8f;
    }
  }

  &.status-banner-30 {
    background: #fff1f0;
    color: #cf1322;
    .banner-icon-wrap {
      background: #ffa39e;
    }
  }

  &.status-banner-40 {
    background: #f5f5f5;
    color: #595959;
    .banner-icon-wrap {
      background: #d9d9d9;
    }
  }
}

.content-container {
  padding: 20rpx;
}

.card {
  background: #ffffff;
  border-radius: 16rpx;
  padding: 24rpx;
  margin-bottom: 20rpx;
  box-shadow: 0 2rpx 8rpx rgba(0, 0, 0, 0.02);
}

.reject-card {
  background: #fff2f0;
  border: 1rpx solid #ffccc7;

  .reject-title {
    font-size: 26rpx;
    font-weight: 600;
    color: #cf1322;
    margin-bottom: 8rpx;
    display: block;
  }

  .reject-text {
    font-size: 26rpx;
    color: #cf1322;
    line-height: 1.5;
  }
}

.product-card {
  display: flex;
  align-items: center;
  justify-content: space-between;

  .product-info {
    display: flex;
    flex-direction: column;
    gap: 6rpx;

    .product-name {
      font-size: 28rpx;
      font-weight: 500;
      color: #333333;
    }

    .sku-spec {
      font-size: 22rpx;
      color: #999999;
    }
  }

  .arrow-icon {
    font-size: 32rpx;
    color: #cccccc;
  }
}

.detail-card {
  .rating-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12rpx;

    .anon-badge {
      font-size: 22rpx;
      color: #999999;
      background: #f5f5f5;
      padding: 4rpx 12rpx;
      border-radius: 6rpx;
    }
  }

  .create-time {
    font-size: 22rpx;
    color: #999999;
    margin-bottom: 24rpx;
    display: block;
  }

  .review-text-box {
    margin-bottom: 20rpx;

    .review-text {
      font-size: 28rpx;
      color: #333333;
      line-height: 1.6;
      word-break: break-all;
    }
  }

  .image-grid {
    display: flex;
    flex-wrap: wrap;
    gap: 16rpx;

    .review-image {
      width: 216rpx;
      height: 216rpx;
      border-radius: 12rpx;
      background: #f5f5f5;
    }
  }
}

.reply-card {
  background: #f8f9fa;
  border-left: 6rpx solid #e02e24;

  .reply-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12rpx;

    .reply-badge {
      font-size: 24rpx;
      font-weight: bold;
      color: #e02e24;
    }

    .reply-time {
      font-size: 22rpx;
      color: #999999;
    }
  }

  .reply-text {
    font-size: 26rpx;
    color: #555555;
    line-height: 1.5;
    word-break: break-all;
  }
}
</style>
