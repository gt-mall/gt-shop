<template>
  <view class="mine-reviews-page">
    <NetworkBanner @retry="loadData(true)" />

    <PageState
      :state="pageState"
      :error-text="firstError"
      empty-text="暂无评价记录"
      empty-sub-text="您购买的商品确认收货后，快来发表评价吧～"
      empty-action-text="查看订单"
      @retry="loadData(true)"
      @empty-action="goToOrders"
    >
      <template #loading>
        <view class="skeleton-list">
          <view v-for="n in 3" :key="n" class="skeleton-item">
            <Skeleton type="text" width="50%" height="32rpx" style="margin-bottom: 16rpx;" />
            <Skeleton type="text" width="90%" height="24rpx" style="margin-bottom: 12rpx;" />
            <Skeleton type="text" width="60%" height="24rpx" style="margin-bottom: 20rpx;" />
            <Skeleton type="rect" width="100%" height="80rpx" radius="8rpx" />
          </view>
        </view>
      </template>

      <!-- 评价列表 -->
      <view class="review-list">
        <view
          v-for="item in items"
          :key="item.id"
          class="review-item"
          @tap="goToDetail(item.id)"
        >
          <!-- 头部: 商品名称快照 + 状态标签 -->
          <view class="item-header">
            <text class="product-title">{{ item.product_name }}</text>
            <view class="status-badge" :class="'status-' + item.status">
              {{ item.status_text || formatStatus(item.status) }}
            </view>
          </view>

          <!-- 规格信息 (若有) -->
          <view v-if="item.sku_spec_values" class="sku-spec-row">
            <text class="sku-spec-text">规格：{{ item.sku_spec_values }}</text>
          </view>

          <!-- 评分与时间 -->
          <view class="rating-date-row">
            <RatingStars :value="item.rating" :size="24" />
            <text class="created-at">{{ formatDate(item.created_at) }}</text>
          </view>

          <!-- 评价内容 -->
          <view class="content-box">
            <text class="content-text">{{ item.content }}</text>
          </view>

          <!-- 晒图缩略图 -->
          <view v-if="item.images && item.images.length > 0" class="image-row">
            <image
              v-for="(img, idx) in item.images"
              :key="idx"
              class="thumb-img"
              :src="img"
              mode="aspectFill"
              @tap.stop="previewImage(item.images, idx)"
            />
          </view>

          <!-- 审核驳回原因提示 (仅状态为30且有原因时) -->
          <view v-if="item.status === 30 && item.audit_reason" class="reject-box">
            <text class="reject-title">驳回原因：</text>
            <text class="reject-desc">{{ item.audit_reason }}</text>
          </view>

          <!-- 掌柜回复 -->
          <view v-if="item.reply_content" class="reply-box">
            <view class="reply-header">
              <text class="reply-tag">掌柜回复</text>
              <text v-if="item.reply_at" class="reply-date">{{ formatDate(item.reply_at) }}</text>
            </view>
            <text class="reply-content">{{ item.reply_content }}</text>
          </view>
        </view>

        <!-- 加载更多状态 -->
        <view class="load-more-state">
          <text v-if="loadingMore">正在加载更多...</text>
          <view v-else-if="moreError" class="more-error" @tap="loadData(false)">
            <text class="more-error-text">{{ moreError }}，点击重试</text>
          </view>
          <text v-else-if="!hasMore">— 没有更多评价了 —</text>
        </view>
      </view>
    </PageState>
  </view>
</template>

<script setup lang="ts">
import { ref, computed, watch } from "vue"
import { onLoad, onPullDownRefresh, onReachBottom } from "@dcloudio/uni-app"
import { getMineReviews } from "@/api/review"
import type { MiniappMineReviewItemResponse } from "@/api/review"
import RatingStars from "@/components/RatingStars.vue"
import Skeleton from "@/components/Skeleton.vue"
import PageState from "@/components/PageState.vue"
import NetworkBanner from "@/components/NetworkBanner.vue"
import { useNetwork } from "@/composables/useNetwork"
import { useMemberStore } from "@/stores/member"

const memberStore = useMemberStore()
const items = ref<MiniappMineReviewItemResponse[]>([])
const loading = ref(true)
const loadingMore = ref(false)
const page = ref(1)
const pageSize = 10
const hasMore = ref(true)
const firstError = ref("")
const moreError = ref("")

// 请求序号与会话隔离快照
let requestSeq = 0

const pageState = computed(() => {
  if (loading.value && items.value.length === 0) return "loading"
  if (firstError.value && items.value.length === 0) return "error"
  if (!loading.value && items.value.length === 0) return "empty"
  return "content"
})

useNetwork({
  onOnlineRefresh: () => {
    return loadData(true)
  },
})

// 会话变化监听：退出或切换账号时清空本地评价列表
watch(
  () => memberStore.member?.id,
  (newId, oldId) => {
    if (newId !== oldId) {
      items.value = []
      page.value = 1
      hasMore.value = true
      requestSeq++
      if (newId) {
        void loadData(true)
      } else {
        loading.value = false
      }
    }
  }
)

onLoad(() => {
  void loadData(true)
})

onPullDownRefresh(async () => {
  await loadData(true)
  uni.stopPullDownRefresh()
})

onReachBottom(() => {
  if (!hasMore.value || loadingMore.value || loading.value) return
  void loadMore()
})

async function loadData(reset = false) {
  if (loadingMore.value || (!reset && !hasMore.value)) return

  const currentMemberId = memberStore.member?.id || 0
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
    const res = await getMineReviews(targetPage, pageSize)

    // 会话与请求序号保护：旧请求或已切换账号放弃写入
    if (currentSeq !== requestSeq || (memberStore.member?.id || 0) !== currentMemberId) {
      return
    }

    if (res) {
      page.value = targetPage // 提交成功页码
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
  await loadData(false)
}

function formatStatus(status: number) {
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
      return "待处理"
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

function previewImage(urls: string[], currentIdx: number) {
  uni.previewImage({
    current: urls[currentIdx],
    urls,
  })
}

function goToDetail(id: number) {
  uni.navigateTo({
    url: `/pages/review/detail?id=${id}`,
  })
}

function goToOrders() {
  uni.navigateTo({
    url: "/pages/order/index?status=40",
  })
}
</script>

<style lang="scss" scoped>
.mine-reviews-page {
  min-height: 100vh;
  background-color: #f7f8fa;
  padding: 20rpx;
  box-sizing: border-box;
}

.review-list {
  display: flex;
  flex-direction: column;
  gap: 20rpx;
}

.review-item-card {
  background: #ffffff;
  border-radius: 16rpx;
  padding: 24rpx;
  box-shadow: 0 2rpx 8rpx rgba(0, 0, 0, 0.02);

  .item-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 16rpx;
    margin-bottom: 12rpx;

    .product-title-wrap {
      flex: 1;
      display: flex;
      flex-direction: column;
      gap: 4rpx;

      .product-title {
        font-size: 28rpx;
        font-weight: 500;
        color: #333333;
        line-height: 1.4;
      }

      .sku-spec {
        font-size: 22rpx;
        color: #999999;
      }
    }

    .status-badge {
      font-size: 22rpx;
      padding: 4rpx 12rpx;
      border-radius: 6rpx;
      flex-shrink: 0;
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

  .rating-date-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 16rpx;

    .date-text {
      font-size: 22rpx;
      color: #999999;
    }
  }

  .content-box {
    margin-bottom: 16rpx;
    line-height: 1.5;

    .content-text {
      font-size: 27rpx;
      color: #333333;
      word-break: break-all;
    }
  }

  .image-grid {
    display: flex;
    flex-wrap: wrap;
    gap: 12rpx;
    margin-bottom: 16rpx;

    .thumb-img {
      width: 154rpx;
      height: 154rpx;
      border-radius: 8rpx;
      background: #f5f5f5;
    }
  }

  .reject-box {
    background: #fff2f0;
    border: 1rpx solid #ffccc7;
    border-radius: 8rpx;
    padding: 12rpx 16rpx;
    margin-bottom: 14rpx;
    font-size: 24rpx;

    .reject-tag {
      color: #cf1322;
      font-weight: 500;
    }

    .reject-reason {
      color: #cf1322;
    }
  }

  .reply-box {
    background: #f8f9fa;
    border-left: 6rpx solid #e02e24;
    border-radius: 6rpx;
    padding: 12rpx 16rpx;
    margin-top: 10rpx;

    .reply-header {
      display: flex;
      justify-content: space-between;
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

    .reply-content {
      font-size: 24rpx;
      color: #555555;
      line-height: 1.45;
      word-break: break-all;
    }
  }
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

  .empty-icon-wrap {
    width: 120rpx;
    height: 120rpx;
    border-radius: 50%;
    background: #f0f0f0;
    display: flex;
    align-items: center;
    justify-content: center;
    margin-bottom: 24rpx;

    .empty-icon {
      font-size: 60rpx;
    }
  }

  .empty-title {
    font-size: 30rpx;
    font-weight: 600;
    color: #333333;
    margin-bottom: 12rpx;
  }

  .empty-sub {
    font-size: 24rpx;
    color: #999999;
    margin-bottom: 40rpx;
    text-align: center;
  }

  .empty-btn {
    height: 72rpx;
    line-height: 70rpx;
    padding: 0 40rpx;
    font-size: 26rpx;
    border-radius: 36rpx;
    background: #ffffff;
    border: 1rpx solid #e02e24;
    color: #e02e24;
  }
}
</style>
