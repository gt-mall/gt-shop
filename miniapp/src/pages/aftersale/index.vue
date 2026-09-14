<template>
  <view class="aftersale-list-page">
    <!-- 顶部状态分类 Tab 栏 -->
    <view class="tab-bar">
      <view
        v-for="tab in tabs"
        :key="tab.value"
        class="tab-item"
        :class="{ active: currentTab === tab.value }"
        @tap="handleTabChange(tab.value)"
      >
        <text class="tab-text">{{ tab.name }}</text>
        <view class="tab-line" v-if="currentTab === tab.value"></view>
      </view>
    </view>

    <!-- 列表主体 -->
    <view v-if="list.length > 0" class="cards-list">
      <view
        v-for="item in list"
        :key="item.id"
        class="as-card"
        @tap="goToDetail(item.id)"
      >
        <!-- 卡片头部 -->
        <view class="card-header">
          <view class="as-no-wrap">
            <text class="as-no-label">售后号:</text>
            <text class="as-no-val">{{ item.after_sale_no }}</text>
          </view>
          <text class="status-badge" :class="'status-' + item.status">
            {{ item.status_text }}
          </text>
        </view>

        <!-- 商品明细 -->
        <view class="goods-list">
          <view v-for="g in item.items" :key="g.id" class="goods-row">
            <image class="goods-img" :src="g.sku_image_url" mode="aspectFill" />
            <view class="goods-info">
              <text class="goods-name">{{ g.product_name }}</text>
              <text v-if="g.sku_text" class="goods-spec">{{ g.sku_text }}</text>
              <view class="goods-bottom">
                <text class="goods-type">{{ item.type_text }}</text>
                <text class="goods-qty">x{{ g.quantity }}</text>
              </view>
            </view>
          </view>
        </view>

        <!-- 原因与退款金额 -->
        <view class="card-summary">
          <view class="reason-text truncate">原因: {{ item.reason }}</view>
          <view class="amount-wrap">
            <text class="amount-label">退款金额:</text>
            <text class="amount-val">¥{{ item.refund_amount }}</text>
          </view>
        </view>

        <!-- 卡片底部操作栏 -->
        <view class="card-footer">
          <button class="detail-btn" @tap.stop="goToDetail(item.id)">
            查看详情
          </button>
        </view>
      </view>

      <!-- 触底加载状态 -->
      <view class="load-more-status">
        <text v-if="loading">加载中...</text>
        <text v-else-if="finished">没有更多售后单了</text>
      </view>
    </view>

    <!-- 空数据状态 -->
    <view v-else-if="!loading" class="empty-wrap">
      <text class="empty-icon">📋</text>
      <text class="empty-text">暂无相关售后申请记录</text>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref } from "vue"
import { onShow, onPullDownRefresh, onReachBottom } from "@dcloudio/uni-app"
import {
  getAfterSaleList,
  type MiniappAfterSaleListItemResponse,
} from "@/api/aftersale"

const tabs = [
  { name: "全部", value: 0 },
  { name: "待审核", value: 10 },
  { name: "待退货", value: 20 },
  { name: "待收货", value: 30 },
  { name: "已完成", value: 50 },
]

const currentTab = ref(0)
const list = ref<MiniappAfterSaleListItemResponse[]>([])
const page = ref(1)
const pageSize = 10
const loading = ref(false)
const finished = ref(false)
let requestSeq = 0

onShow(() => {
  resetAndFetch()
})

onPullDownRefresh(async () => {
  await resetAndFetch()
  uni.stopPullDownRefresh()
})

onReachBottom(() => {
  if (!finished.value && !loading.value) {
    page.value++
    const seq = requestSeq
    loading.value = true
    void fetchList(seq)
  }
})

function handleTabChange(val: number) {
  if (currentTab.value === val) return
  currentTab.value = val
  resetAndFetch()
}

async function resetAndFetch() {
  const seq = ++requestSeq
  page.value = 1
  finished.value = false
  list.value = []
  loading.value = true
  await fetchList(seq)
}

async function fetchList(seq: number) {
  try {
    const res = await getAfterSaleList({
      page: page.value,
      page_size: pageSize,
      status: currentTab.value > 0 ? currentTab.value : undefined,
    })

    if (seq !== requestSeq) {
      return
    }

    const rows = res || []
    if (page.value === 1) {
      list.value = rows
    } else {
      list.value = [...list.value, ...rows]
    }

    if (rows.length < pageSize) {
      finished.value = true
    }
  } catch (err: any) {
    if (seq !== requestSeq) {
      return
    }
    uni.showToast({ title: err.message || "获取列表失败", icon: "none" })
  } finally {
    if (seq === requestSeq) {
      loading.value = false
    }
  }
}

function goToDetail(id: number) {
  uni.navigateTo({
    url: `/pages/aftersale/detail?id=${id}`,
  })
}
</script>

<style lang="scss" scoped>
.aftersale-list-page {
  min-height: 100vh;
  background-color: #f7f7f7;
  padding-bottom: 40rpx;
}

.tab-bar {
  position: sticky;
  top: 0;
  left: 0;
  right: 0;
  height: 88rpx;
  background: #ffffff;
  display: flex;
  align-items: center;
  border-bottom: 1rpx solid #eeeeee;
  z-index: 50;

  .tab-item {
    flex: 1;
    height: 100%;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    position: relative;

    .tab-text {
      font-size: 28rpx;
      color: #666666;
    }

    &.active {
      .tab-text {
        font-size: 28rpx;
        font-weight: 600;
        color: #e02e24;
      }
    }

    .tab-line {
      position: absolute;
      bottom: 0;
      width: 48rpx;
      height: 4rpx;
      background: #e02e24;
      border-radius: 2rpx;
    }
  }
}

.cards-list {
  padding: 24rpx;
}

.as-card {
  background: #ffffff;
  border-radius: 16rpx;
  padding: 24rpx;
  margin-bottom: 20rpx;

  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding-bottom: 16rpx;
    border-bottom: 1rpx solid #f5f5f5;

    .as-no-wrap {
      font-size: 24rpx;
      color: #999999;

      .as-no-val {
        color: #333333;
        margin-left: 8rpx;
        font-weight: 500;
      }
    }

    .status-badge {
      font-size: 24rpx;
      font-weight: 600;

      &.status-10 {
        color: #fa8c16;
      }
      &.status-20 {
        color: #1890ff;
      }
      &.status-30 {
        color: #13c2c2;
      }
      &.status-40 {
        color: #08979c;
      }
      &.status-50 {
        color: #52c41a;
      }
      &.status-60 {
        color: #f5222d;
      }
      &.status-70 {
        color: #8c8c8c;
      }
      &.status-80 {
        color: #eb2f96;
      }
    }
  }

  .goods-list {
    padding: 16rpx 0;

    .goods-row {
      display: flex;
      gap: 16rpx;
      margin-bottom: 16rpx;

      &:last-child {
        margin-bottom: 0;
      }

      .goods-img {
        width: 120rpx;
        height: 120rpx;
        border-radius: 8rpx;
        background: #f5f5f5;
      }

      .goods-info {
        flex: 1;
        display: flex;
        flex-direction: column;
        justify-content: space-between;

        .goods-name {
          font-size: 26rpx;
          color: #333333;
          line-height: 34rpx;
          overflow: hidden;
          text-overflow: ellipsis;
          display: -webkit-box;
          -webkit-line-clamp: 2;
          -webkit-box-orient: vertical;
        }

        .goods-spec {
          font-size: 22rpx;
          color: #999999;
        }

        .goods-bottom {
          display: flex;
          justify-content: space-between;
          align-items: center;

          .goods-type {
            font-size: 22rpx;
            color: #666666;
            background: #f5f5f5;
            padding: 2rpx 10rpx;
            border-radius: 4rpx;
          }

          .goods-qty {
            font-size: 22rpx;
            color: #999999;
          }
        }
      }
    }
  }

  .card-summary {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16rpx 0;
    border-top: 1rpx solid #f5f5f5;
    font-size: 24rpx;

    .reason-text {
      color: #666666;
      max-width: 50%;
    }

    .amount-wrap {
      .amount-label {
        color: #666666;
      }

      .amount-val {
        font-size: 30rpx;
        font-weight: 700;
        color: #e02e24;
        margin-left: 6rpx;
      }
    }
  }

  .card-footer {
    display: flex;
    justify-content: flex-end;
    padding-top: 16rpx;
    border-top: 1rpx solid #f5f5f5;

    .detail-btn {
      margin: 0;
      padding: 0 28rpx;
      height: 56rpx;
      line-height: 54rpx;
      font-size: 24rpx;
      border-radius: 28rpx;
      border: 2rpx solid #dddddd;
      background: #ffffff;
      color: #666666;

      &::after {
        border: none;
      }
    }
  }
}

.load-more-status {
  text-align: center;
  padding: 30rpx 0;
  font-size: 24rpx;
  color: #999999;
}

.empty-wrap {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 160rpx 0;

  .empty-icon {
    font-size: 80rpx;
    margin-bottom: 20rpx;
  }

  .empty-text {
    font-size: 28rpx;
    color: #999999;
  }
}
</style>
