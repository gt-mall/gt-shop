<template>
  <view class="product-reviews-page">
    <NetworkBanner @retry="fetchData(true)" />

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

    <!-- 页面三态 -->
    <PageState
      :state="pageState"
      :error-text="firstError"
      empty-text="暂无相关评价"
      empty-sub-text="该筛选条件下暂无公开评价"
      @retry="fetchData(true)"
    >
      <template #loading>
        <view class="skeleton-list">
          <view v-for="n in 3" :key="n" class="skeleton-item">
            <Skeleton type="text" width="40%" height="28rpx" style="margin-bottom: 16rpx;" />
            <Skeleton type="text" width="90%" height="24rpx" style="margin-bottom: 12rpx;" />
            <Skeleton type="text" width="70%" height="24rpx" style="margin-bottom: 16rpx;" />
            <Skeleton type="rect" width="160rpx" height="160rpx" radius="12rpx" />
          </view>
        </view>
      </template>

      <!-- 评价列表 -->
      <view class="review-list">
        <ReviewCard
          v-for="item in items"
          :key="item.id"
          :review="item"
          mode="product"
        />

        <!-- 分页底部状态 -->
        <view class="load-more-state">
          <text v-if="loadingMore">正在加载更多...</text>
          <view v-else-if="moreError" class="more-error" @tap="fetchData(false)">
            <text class="more-error-text">{{ moreError }}，点击重试</text>
          </view>
          <text v-else-if="!hasMore">— 没有更多评价了 —</text>
        </view>
      </view>
    </PageState>
  </view>
</template>

<script setup lang="ts">
import { ref, computed } from "vue"
import { onLoad, onPullDownRefresh, onReachBottom } from "@dcloudio/uni-app"
import { getProductReviews } from "@/api/review"
import type {
  MiniappProductReviewItemResponse,
  MiniappProductReviewSummary,
} from "@/api/review"
import RatingStars from "@/components/RatingStars.vue"
import ReviewCard from "@/components/ReviewCard.vue"
import Skeleton from "@/components/Skeleton.vue"
import PageState from "@/components/PageState.vue"
import NetworkBanner from "@/components/NetworkBanner.vue"
import { useNetwork } from "@/composables/useNetwork"

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
const firstError = ref("")
const moreError = ref("")

// 请求时序序列号，防止旧的慢请求覆盖新的筛选结果
let requestSeq = 0

const pageState = computed(() => {
  if (loading.value && items.value.length === 0) return "loading"
  if (firstError.value && items.value.length === 0) return "error"
  if (!loading.value && items.value.length === 0) return "empty"
  return "content"
})

useNetwork({
  onOnlineRefresh: () => {
    return fetchData(true)
  },
})

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
  if (!hasMore.value || loadingMore.value || loading.value) return
  void loadMore()
})

function switchFilter(filter: FilterType) {
  if (currentFilter.value === filter) return
  currentFilter.value = filter
  void fetchData(true)
}

async function fetchData(reset = false) {
  if (!productId.value) return
  if (loadingMore.value || (!reset && !hasMore.value)) return

  const targetPage = reset ? 1 : page.value + 1
  const currentSeq = ++requestSeq

  if (reset) {
    loading.value = true
    firstError.value = ""
  } else {
    loadingMore.value = true
    moreError.value = ""
  }

  try {
    const res = await getProductReviews(productId.value, {
      page: targetPage,
      page_size: pageSize,
      filter: currentFilter.value,
    })

    // 防乱序检查：如果在这期间发起了更新的请求，则放弃本次结果
    if (currentSeq !== requestSeq) return

    if (res) {
      page.value = targetPage // 提交成功页码
      if (res.summary) {
        summary.value = res.summary
      }
      const list = res.items || []
      if (reset) {
        items.value = list
      } else {
        const existingIds = new Set(items.value.map((i) => i.id))
        const newItems = list.filter((i) => !existingIds.has(i.id))
        items.value.push(...newItems)
      }
      hasMore.value = list.length >= pageSize
    }
  } catch (err: any) {
    if (currentSeq !== requestSeq) return
    const msg = err?.message || "获取评价列表失败"
    if (reset && items.value.length === 0) {
      firstError.value = msg
    } else if (!reset) {
      moreError.value = msg
    } else {
      uni.showToast({ title: msg, icon: "none" })
    }
  } finally {
    if (currentSeq === requestSeq) {
      loading.value = false
      loadingMore.value = false
    }
  }
}

async function loadMore() {
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

  .more-error {
    color: #e02e24;
    cursor: pointer;
    display: inline-block;
    padding: 8rpx 20rpx;
    background: #fff2f0;
    border-radius: 20rpx;
  }
}

.skeleton-list {
  display: flex;
  flex-direction: column;
  gap: 20rpx;

  .skeleton-item {
    background: #ffffff;
    border-radius: 16rpx;
    padding: 24rpx;
  }
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
