<template>
  <view class="review-card" :class="'mode-' + mode">
    <!-- 头部信息 -->
    <view class="card-header">
      <view class="user-info">
        <image
          class="user-avatar"
          :src="displayAvatar"
          mode="aspectFill"
        />
        <view class="user-meta">
          <view class="name-row">
            <text class="user-name">{{ displayName }}</text>
            <text v-if="review.is_anonymous" class="anon-tag">匿名</text>
          </view>
          <view class="sub-row">
            <RatingStars :value="review.rating" :size="24" />
            <text class="review-date">{{ formatDate(review.created_at) }}</text>
          </view>
        </view>
      </view>

      <!-- 我的评价模式下：状态标签 -->
      <view v-if="mode === 'mine'" class="status-wrap">
        <text class="status-badge" :class="'status-' + review.status">
          {{ review.status_text || getStatusText(review.status) }}
        </text>
      </view>
    </view>

    <!-- 商品与规格描述 (在我的评价或需要时展示) -->
    <view v-if="showProductInfo && (review.product_name || review.sku_spec_values)" class="product-snippet">
      <text v-if="review.product_name" class="product-title">{{ review.product_name }}</text>
      <text v-if="review.sku_spec_values" class="sku-specs">{{ review.sku_spec_values }}</text>
    </view>
    <view v-else-if="review.sku_spec_values" class="spec-line">
      <text class="spec-label">规格：{{ review.sku_spec_values }}</text>
    </view>

    <!-- 评价文字内容 -->
    <view class="review-content">
      <text class="content-text">{{ review.content }}</text>
    </view>

    <!-- 图片列表 (最多6张) -->
    <view v-if="review.images && review.images.length > 0" class="image-grid">
      <image
        v-for="(img, idx) in review.images"
        :key="idx"
        class="review-thumb"
        :src="img"
        mode="aspectFill"
        @tap="handlePreviewImage(idx)"
      />
    </view>

    <!-- 驳回原因提示 (仅我的评价且驳回状态) -->
    <view v-if="mode === 'mine' && review.status === 30 && review.audit_reason" class="reject-alert">
      <text class="reject-label">驳回原因：</text>
      <text class="reject-desc">{{ review.audit_reason }}</text>
    </view>

    <!-- 商家回复 -->
    <view v-if="review.reply_content" class="merchant-reply">
      <view class="reply-header">
        <text class="reply-tag">商家回复</text>
        <text v-if="review.reply_at" class="reply-date">{{ formatDate(review.reply_at) }}</text>
      </view>
      <text class="reply-text">{{ review.reply_content }}</text>
    </view>
  </view>
</template>

<script setup lang="ts">
import { computed } from "vue"
import RatingStars from "./RatingStars.vue"

export interface ReviewCardItem {
  id: number
  rating: number
  content: string
  created_at: string
  is_anonymous?: number
  user_nickname?: string
  user_avatar?: string
  product_name?: string
  sku_spec_values?: string | null
  images?: string[]
  reply_content?: string
  reply_at?: string | null
  status?: number
  status_text?: string
  audit_reason?: string
}

interface Props {
  review: ReviewCardItem
  mode?: "product" | "mine"
  showProductInfo?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  mode: "product",
  showProductInfo: false,
})

const defaultAvatar = "/static/logo.png"

const displayAvatar = computed(() => {
  if (props.review.is_anonymous) {
    return defaultAvatar
  }
  return props.review.user_avatar || defaultAvatar
})

const displayName = computed(() => {
  if (props.mode === "mine") {
    return props.review.user_nickname || "我"
  }
  if (props.review.is_anonymous) {
    const raw = props.review.user_nickname || "用户"
    if (raw.length <= 1) return raw + "***"
    return raw.charAt(0) + "***" + raw.charAt(raw.length - 1)
  }
  return props.review.user_nickname || "买家"
})

function getStatusText(status?: number) {
  switch (status) {
    case 10:
      return "待审核"
    case 20:
      return "已公开"
    case 30:
      return "审核未通过"
    case 40:
      return "已隐藏"
    default:
      return ""
  }
}

function formatDate(dateStr?: string | null) {
  if (!dateStr) return ""
  try {
    const d = new Date(dateStr)
    if (isNaN(d.getTime())) return dateStr
    const y = d.getFullYear()
    const m = String(d.getMonth() + 1).padStart(2, "0")
    const day = String(d.getDate()).padStart(2, "0")
    return `${y}-${m}-${day}`
  } catch {
    return dateStr
  }
}

function handlePreviewImage(index: number) {
  if (!props.review.images || props.review.images.length === 0) return
  uni.previewImage({
    current: props.review.images[index],
    urls: props.review.images,
  })
}
</script>

<style lang="scss" scoped>
.review-card {
  background: #ffffff;
  padding: 24rpx;
  border-radius: 16rpx;
  margin-bottom: 20rpx;
  box-shadow: 0 2rpx 8rpx rgba(0, 0, 0, 0.03);

  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 16rpx;

    .user-info {
      display: flex;
      align-items: center;
      gap: 16rpx;

      .user-avatar {
        width: 68rpx;
        height: 68rpx;
        border-radius: 50%;
        background: #f0f0f0;
        flex-shrink: 0;
      }

      .user-meta {
        display: flex;
        flex-direction: column;
        gap: 6rpx;

        .name-row {
          display: flex;
          align-items: center;
          gap: 10rpx;

          .user-name {
            font-size: 26rpx;
            font-weight: 500;
            color: #333333;
          }

          .anon-tag {
            font-size: 20rpx;
            color: #999999;
            background: #f2f3f5;
            padding: 2rpx 8rpx;
            border-radius: 4rpx;
          }
        }

        .sub-row {
          display: flex;
          align-items: center;
          gap: 12rpx;

          .review-date {
            font-size: 22rpx;
            color: #999999;
          }
        }
      }
    }

    .status-wrap {
      .status-badge {
        font-size: 22rpx;
        padding: 4rpx 14rpx;
        border-radius: 8rpx;
        font-weight: 500;

        &.status-10 {
          color: #fa8c16;
          background: #fff7e6;
          border: 1rpx solid #ffd591;
        }

        &.status-20 {
          color: #52c41a;
          background: #f6ffed;
          border: 1rpx solid #b7eb8f;
        }

        &.status-30 {
          color: #ff4d4f;
          background: #fff1f0;
          border: 1rpx solid #ffa39e;
        }

        &.status-40 {
          color: #8c8c8c;
          background: #f5f5f5;
          border: 1rpx solid #d9d9d9;
        }
      }
    }
  }

  .product-snippet {
    background: #f8f9fa;
    padding: 12rpx 16rpx;
    border-radius: 8rpx;
    margin-bottom: 14rpx;
    display: flex;
    flex-direction: column;
    gap: 4rpx;

    .product-title {
      font-size: 24rpx;
      font-weight: 500;
      color: #333333;
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }

    .sku-specs {
      font-size: 22rpx;
      color: #888888;
    }
  }

  .spec-line {
    margin-bottom: 12rpx;

    .spec-label {
      font-size: 22rpx;
      color: #999999;
    }
  }

  .review-content {
    margin-bottom: 16rpx;
    line-height: 1.5;

    .content-text {
      font-size: 27rpx;
      color: #222222;
      word-break: break-all;
    }
  }

  .image-grid {
    display: flex;
    flex-wrap: wrap;
    gap: 12rpx;
    margin-bottom: 16rpx;

    .review-thumb {
      width: 160rpx;
      height: 160rpx;
      border-radius: 8rpx;
      background: #f5f5f5;
    }
  }

  .reject-alert {
    background: #fff2f0;
    border: 1rpx solid #ffccc7;
    border-radius: 8rpx;
    padding: 12rpx 16rpx;
    margin-bottom: 16rpx;
    font-size: 24rpx;

    .reject-label {
      color: #cf1322;
      font-weight: 500;
    }

    .reject-desc {
      color: #cf1322;
    }
  }

  .merchant-reply {
    background: #f7f8fa;
    border-left: 6rpx solid #e02e24;
    border-radius: 6rpx;
    padding: 14rpx 18rpx;
    margin-top: 12rpx;

    .reply-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 6rpx;

      .reply-tag {
        font-size: 22rpx;
        font-weight: 600;
        color: #e02e24;
      }

      .reply-date {
        font-size: 20rpx;
        color: #999999;
      }
    }

    .reply-text {
      font-size: 24rpx;
      color: #555555;
      line-height: 1.45;
      word-break: break-all;
    }
  }
}
</style>
