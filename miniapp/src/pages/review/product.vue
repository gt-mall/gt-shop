<template>
  <view class="product-reviews-page">
    <!-- 头部摘要与统计 -->
    <view class="summary-header">
      <view class="rating-overview">
        <view class="score-box">
          <text class="score-num">{{ summary.average_rating || '5.0' }}</text>
          <view class="score-meta">
            <RatingStars :value="parseFloat(summary.average_rating || '5')" :size="26" />
            <text class="good-rate">好评度 {{ summary.good_rate || '100%' }}</text>
          </view>
        </view>
      </view>

      <!-- 筛选标签栏 (按服务端契约筛选，防本地假过滤) -->
      <view class="filter-chips">
        <view
          class="chip-item"
          :class="{ active: currentFilter === 'all' }"
          @tap="switchFilter('all')"
        >
          全部 ({{ summary.total_count || 0 }})
        </view>
        <view
          class="chip-item"
          :class="{ active: currentFilter === 'has_image' }"
          @tap="switchFilter('has_image')"
        >
          有图 ({{ summary.has_image_count || 0 }})
        </view>
        <view
          class="chip-item"
          :class="{ active: currentFilter === 'good' }"
          @tap="switchFilter('good')"
        >
          好评 ({{ summary.good_count || 0 }})
        </view>
        <view
          class="chip-item"
          :class="{ active: currentFilter === 'medium' }"
          @tap="switchFilter('medium')"
        >
          中评 ({{ summary.medium_count || 0 }})
        </view>
        <view
          class="chip-item"
          :class="{ active: currentFilter === 'bad' }"
          @tap="switchFilter('bad')"
        >
          差评 ({{ summary.bad_count || 0 }})
        </view>
      </view>
    </view>

    <!-- 评价列表 -->
    <view v-if="items.length > 0" class="review-list">
      <ReviewCard
        v-for="item in items"
        :key="item.id"
        :review="item"
        mode="product"
      />

      <view class="load-more-state">
        <text v-if="loadingMore">正在加载更多...</text>
        <text v-else-if="!hasMore">— 没有更多评价了 —</text>
      </view>
    </view>

    <!-- 加载中 -->
    <view v-else-if="loading" class="empty-state">
      <text class="empty-tip">加载中...</text>
    </view>

    <!-- 空状态 -->
    <view v-else class="empty-state">
      <view class="empty-icon-box">
        <text class="empty-icon">💬</text>
      </view>
      <text class="empty-title">暂无相关评价</text>
      <text class="empty-sub">该筛选条件下暂无公开评价</text>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref } from "vue"
import { onLoad, onPullDownRefresh, onReachBottom } from "@dcloudio/uni-app"
import { getProductReviews } from "@/api/review"
import type {
  MiniappProductReviewItemResponse,
  MiniappProductReviewSummary,
} from "@/api/review"
import RatingStars from "@/components/RatingStars.vue"
import ReviewCard from "@/components/ReviewCard.vue"

type FilterType = "all" | "has_image" | "good" | "medium" | "bad"

const productId = ref(0)
const currentFilter = ref<FilterType>("all")
const summary = ref<MiniappProductReviewSummary>({
  total_count: 0,
  has_image_count: 0,
  good_count: 0,
  medium_count: 0,
  bad_count: 0,
  average_rating: "5.0",
  good_rate: "100%",
})
const items = ref<MiniappProductReviewItemResponse[]>([])
const loading = ref(true)
const loadingMore = ref(false)
const page = ref(1)
const pageSize = 10
const hasMore = ref(true)

// 请求时序序列号，防止旧的慢请求覆盖新的筛选结果
let requestSeq = 0

onLoad((options: any) => {
  if (options?.id) {
    productId.value = Number(options.id)
    void fetchData(true)
  } else {
    loading.value = false
  }
})

onPullDownRefresh(async () => {
  await fetchData(true)
  uni.stopPullDownRefresh()
})

onReachBottom(() => {
  if (!hasMore.value || loadingMore.value) return
  void loadMore()
})

function switchFilter(filter: FilterType) {
  if (currentFilter.value === filter) return
  currentFilter.value = filter
  void fetchData(true)
}

async function fetchData(reset = false) {
  if (!productId.value) return
  const currentSeq = ++requestSeq

  if (reset) {
    page.value = 1
    loading.value = true
  }

  try {
    const res = await getProductReviews(productId.value, {
      page: page.value,
      page_size: pageSize,
      filter: currentFilter.value,
    })

    // 防乱序检查：如果在这期间发起了更新的请求，则放弃本次结果
    if (currentSeq !== requestSeq) return

    if (res) {
      if (res.summary) {
        summary.value = res.summary
      }
      const list = res.items || []
      if (reset) {
        items.value = list
      } else {
        items.value = [...items.value, ...list]
      }
      hasMore.value = list.length >= pageSize
    }
  } catch (err: any) {
    if (currentSeq !== requestSeq) return
    uni.showToast({ title: err?.message || "获取评价列表失败", icon: "none" })
  } finally {
    if (currentSeq === requestSeq) {
      loading.value = false
      loadingMore.value = false
    }
  }
}

async function loadMore() {
  page.value += 1
  loadingMore.value = true
  await fetchData(false)
}
</script>

<style lang="scss" scoped>
.product-reviews-page {
  min-height: 100vh;
  background-color: #f7f8fa;
  padding: 20rpx;
  box-sizing: border-box;
}

.summary-header {
  background: #ffffff;
  border-radius: 16rpx;
  padding: 24rpx;
  margin-bottom: 20rpx;
  box-shadow: 0 2rpx 8rpx rgba(0, 0, 0, 0.02);

  .rating-overview {
    margin-bottom: 20rpx;

    .score-box {
      display: flex;
      align-items: center;
      gap: 20rpx;

      .score-num {
        font-size: 52rpx;
        font-weight: bold;
        color: #ff9800;
        line-height: 1;
      }

      .score-meta {
        display: flex;
        flex-direction: column;
        gap: 6rpx;

        .good-rate {
          font-size: 22rpx;
          color: #666666;
        }
      }
    }
  }

  .filter-chips {
    display: flex;
    flex-wrap: wrap;
    gap: 12rpx;

    .chip-item {
      font-size: 24rpx;
      padding: 10rpx 20rpx;
      border-radius: 28rpx;
      background: #f5f5f5;
      color: #666666;
      transition: all 0.2s ease;

      &.active {
        background: #fee7e5;
        color: #e02e24;
        font-weight: 500;
      }
    }
  }
}

.review-list {
  display: flex;
  flex-direction: column;
}

.load-more-state {
  text-align: center;
  padding: 30rpx 0;
  font-size: 24rpx;
  color: #999999;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 120rpx 40rpx;

  .empty-icon-box {
    width: 120rpx;
    height: 120rpx;
    border-radius: 50%;
    background: #f0f0f0;
    display: flex;
    align-items: center;
    justify-content: center;
    margin-bottom: 24rpx;

    .empty-icon {
      font-size: 56rpx;
    }
  }

  .empty-title {
    font-size: 28rpx;
    font-weight: 600;
    color: #333333;
    margin-bottom: 8rpx;
  }

  .empty-sub {
    font-size: 24rpx;
    color: #999999;
  }
}
</style>
