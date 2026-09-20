<template>
  <view class="order-list-page">
    <!-- 顶部状态分类 Tab 栏 (拼多多风格吸顶) -->
    <view class="tab-bar">
      <view
        v-for="tab in statusTabs"
        :key="tab.value"
        class="tab-item"
        :class="{ active: currentStatus === tab.value }"
        @tap="handleTabChange(tab.value)"
      >
        <text class="tab-text">{{ tab.name }}</text>
        <view class="tab-line" v-if="currentStatus === tab.value"></view>
      </view>
    </view>

    <!-- 订单列表主体 -->
    <view v-if="orderList.length > 0" class="order-cards-list">
      <view
        v-for="order in orderList"
        :key="order.id"
        class="order-card"
        @tap="goToDetail(order.id)"
      >
        <!-- 卡片头部: 订单号与状态 -->
        <view class="card-header">
          <view class="order-no-wrap">
            <text class="order-no-label">订单号:</text>
            <text class="order-no-val">{{ order.order_no }}</text>
          </view>
          <text class="status-badge" :class="'status-' + order.status">
            {{ order.status_text }}
          </text>
        </view>

        <!-- 商品快照列表 -->
        <OrderGoodsList :items="order.items" layout="card" />

        <!-- 金额统计汇总 -->
        <view class="card-summary">
          <text class="total-qty">共 {{ order.total_quantity }} 件商品</text>
          <text class="pay-desc">实付款:</text>
          <text class="pay-amount">¥{{ order.pay_amount }}</text>
        </view>

        <!-- 物流简报 (若已发货) -->
        <view v-if="order.express_company && order.express_no" class="express-bar">
          <text class="express-icon">🚚</text>
          <text class="express-text">{{ order.express_company }}: {{ order.express_no }}</text>
        </view>

        <!-- 底部快捷动作按钮组 -->
        <OrderActionBar
          mode="card"
          :status="order.status"
          :order="order"
          @detail="goToDetail(order.id)"
          @cancel="handleCancelOrder(order)"
          @pay="goToDetail(order.id)"
          @confirm="handleConfirmReceipt(order)"
        />
      </view>

      <!-- 加载更多提示 -->
      <view class="load-more">
        <text class="load-text">{{ hasMore ? "上拉加载更多" : "— 没有更多订单了 —" }}</text>
      </view>
    </view>

    <!-- 空状态 -->
    <view v-else-if="!loading" class="empty-wrap">
      <view class="empty-icon-box">
        <text class="empty-icon">📦</text>
      </view>
      <text class="empty-title">暂无相关订单</text>
      <text class="empty-sub">快去选购心仪的商品吧</text>
      <button class="empty-btn" @tap="goToHome">去逛逛</button>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue"
import { onLoad, onPullDownRefresh, onReachBottom } from "@dcloudio/uni-app"
import { getOrderList, cancelOrder, confirmOrder } from "@/api/order"
import type { OrderListItemResponse } from "@/api/order"
import { getToken } from "@/utils"
import OrderGoodsList from "@/components/OrderGoodsList.vue"
import OrderActionBar from "@/components/OrderActionBar.vue"

interface StatusTab {
  name: string
  value: number
}

const statusTabs: StatusTab[] = [
  { name: "全部", value: 0 },
  { name: "待付款", value: 10 },
  { name: "待发货", value: 20 },
  { name: "待收货", value: 30 },
  { name: "已完成", value: 40 },
  { name: "已取消", value: 50 },
]

const currentStatus = ref<number>(0)
const orderList = ref<OrderListItemResponse[]>([])
const page = ref<number>(1)
const pageSize = 10
const total = ref<number>(0)
const hasMore = ref<boolean>(true)
const loading = ref<boolean>(false)

onLoad((query: any) => {
  if (query?.status) {
    currentStatus.value = Number(query.status)
  }
})

onMounted(() => {
  const token = getToken()
  if (!token) {
    uni.showToast({ title: "请先登录", icon: "none" })
    setTimeout(() => {
      uni.switchTab({ url: "/pages/my/index" })
    }, 600)
    return
  }
  loadOrders(true)
})

onPullDownRefresh(async () => {
  await loadOrders(true)
  uni.stopPullDownRefresh()
})

onReachBottom(() => {
  if (!hasMore.value || loading.value) return
  page.value += 1
  loadOrders(false)
})

async function loadOrders(isRefresh = false): Promise<void> {
  if (isRefresh) {
    page.value = 1
    hasMore.value = true
  }

  loading.value = true
  try {
    const params: any = {
      page: page.value,
      page_size: pageSize,
    }
    if (currentStatus.value > 0) {
      params.status = currentStatus.value
    }

    const res = await getOrderList(params)
    const list = res && res.code === 1 && Array.isArray(res.data) ? res.data : []

    if (isRefresh) {
      orderList.value = list
    } else {
      orderList.value = [...orderList.value, ...list]
    }

    if (list.length < pageSize) {
      hasMore.value = false
    }
  } catch (err: any) {
    uni.showToast({ title: err?.message || "获取订单失败", icon: "none" })
  } finally {
    loading.value = false
  }
}

function handleTabChange(statusVal: number): void {
  if (currentStatus.value === statusVal) return
  currentStatus.value = statusVal
  loadOrders(true)
}

function goToDetail(orderId: number): void {
  uni.navigateTo({
    url: `/pages/order/detail?id=${orderId}`,
  })
}

function goToHome(): void {
  uni.switchTab({ url: "/pages/index/index" })
}

function handleCancelOrder(order: OrderListItemResponse): void {
  uni.showModal({
    title: "取消订单确认",
    content: `确定取消订单 ${order.order_no} 吗？取消后已扣减库存将自动恢复。`,
    confirmColor: "#e02e24",
    success: async (res) => {
      if (res.confirm) {
        uni.showLoading({ title: "正在取消...", mask: true })
        try {
          await cancelOrder(order.id)
          uni.showToast({ title: "订单已取消", icon: "success" })
          loadOrders(true)
        } catch (err: any) {
          uni.showToast({ title: err?.message || "取消失败", icon: "none" })
        } finally {
          uni.hideLoading()
        }
      }
    },
  })
}

function handleConfirmReceipt(order: OrderListItemResponse): void {
  uni.showModal({
    title: "确认收货",
    content: `确认已收到订单 ${order.order_no} 的所有商品吗？`,
    confirmColor: "#e02e24",
    success: async (res) => {
      if (res.confirm) {
        uni.showLoading({ title: "处理中...", mask: true })
        try {
          await confirmOrder(order.id)
          uni.showToast({ title: "已确认收货", icon: "success" })
          loadOrders(true)
        } catch (err: any) {
          uni.showToast({ title: err?.message || "确认收货失败", icon: "none" })
        } finally {
          uni.hideLoading()
        }
      }
    },
  })
}
</script>

<style lang="scss" scoped>
.order-list-page {
  min-height: 100vh;
  background-color: #f7f7f7;
  padding-bottom: 40rpx;
}

/* 顶部吸顶 Tab */
.tab-bar {
  position: sticky;
  top: 0;
  left: 0;
  right: 0;
  height: 88rpx;
  background-color: #ffffff;
  display: flex;
  align-items: center;
  justify-content: space-around;
  box-shadow: 0 2rpx 10rpx rgba(0, 0, 0, 0.04);
  z-index: 50;

  .tab-item {
    position: relative;
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 0 16rpx;

    .tab-text {
      font-size: 28rpx;
      color: #666666;
      transition: all 0.2s ease;
    }

    &.active {
      .tab-text {
        font-size: 30rpx;
        font-weight: bold;
        color: #e02e24;
      }

      .tab-line {
        position: absolute;
        bottom: 8rpx;
        left: 20%;
        right: 20%;
        height: 6rpx;
        border-radius: 4rpx;
        background: #e02e24;
      }
    }
  }
}

/* 订单卡片列表 */
.order-cards-list {
  padding: 20rpx 24rpx;

  .order-card {
    background-color: #ffffff;
    border-radius: 20rpx;
    padding: 24rpx;
    margin-bottom: 20rpx;
    box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.03);

    .card-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 20rpx;
      padding-bottom: 16rpx;
      border-bottom: 1rpx solid #f2f2f2;

      .order-no-wrap {
        display: flex;
        align-items: center;
        font-size: 24rpx;

        .order-no-label {
          color: #999999;
          margin-right: 8rpx;
        }

        .order-no-val {
          color: #333333;
          font-weight: 500;
        }
      }

      .status-badge {
        font-size: 24rpx;
        font-weight: 500;

        &.status-10 {
          color: #faad14;
        }
        &.status-20 {
          color: #1890ff;
        }
        &.status-30 {
          color: #e02e24;
        }
        &.status-40 {
          color: #52c41a;
        }
        &.status-50 {
          color: #999999;
        }
      }
    }

    .card-summary {
      display: flex;
      justify-content: flex-end;
      align-items: baseline;
      padding-top: 16rpx;
      margin-top: 16rpx;
      border-top: 1rpx solid #f8f8f8;

      .total-qty {
        font-size: 24rpx;
        color: #888888;
        margin-right: 12rpx;
      }

      .pay-desc {
        font-size: 24rpx;
        color: #333333;
        margin-right: 6rpx;
      }

      .pay-amount {
        font-size: 32rpx;
        font-weight: bold;
        color: #e02e24;
      }
    }

    .express-bar {
      display: flex;
      align-items: center;
      background: #fcf6f5;
      padding: 12rpx 16rpx;
      border-radius: 8rpx;
      margin-top: 16rpx;

      .express-icon {
        font-size: 24rpx;
        margin-right: 10rpx;
      }

      .express-text {
        font-size: 24rpx;
        color: #e02e24;
      }
    }
  }

  .load-more {
    text-align: center;
    padding: 24rpx 0;

    .load-text {
      font-size: 24rpx;
      color: #bbbbbb;
    }
  }
}

/* 空状态 */
.empty-wrap {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 160rpx 40rpx 0;

  .empty-icon-box {
    width: 140rpx;
    height: 140rpx;
    border-radius: 50%;
    background-color: #ffffff;
    display: flex;
    align-items: center;
    justify-content: center;
    margin-bottom: 24rpx;
    box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.05);

    .empty-icon {
      font-size: 64rpx;
    }
  }

  .empty-title {
    font-size: 32rpx;
    font-weight: bold;
    color: #333333;
    margin-bottom: 8rpx;
  }

  .empty-sub {
    font-size: 26rpx;
    color: #999999;
    margin-bottom: 40rpx;
  }

  .empty-btn {
    padding: 0 64rpx;
    height: 72rpx;
    line-height: 72rpx;
    background: #e02e24;
    color: #ffffff;
    font-size: 28rpx;
    border-radius: 36rpx;
    border: none;

    &::after {
      border: none;
    }
  }
}
</style>
