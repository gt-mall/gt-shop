<template>
  <view class="order-goods-list" :class="{ 'layout-card': layout === 'card' }">
    <view
      v-for="item in items"
      :key="item.id || item.sku_id"
      class="item-row"
    >
      <image
        class="item-img"
        :src="item.image_url || '/static/logo.png'"
        mode="aspectFill"
      />
      <view class="item-meta">
        <text class="item-title">{{ item.product_name }}</text>
        <text v-if="formatSpecs(item.sku_spec_values)" class="item-specs">
          {{ formatSpecs(item.sku_spec_values) }}
        </text>
        <!-- 不配送提示标签（Checkout 页面场景） -->
        <view v-if="item.deliverable === false" class="undeliverable-badge">
          <text class="undeliverable-badge-text">{{ item.undeliverable_reason || "当前收货地区暂不配送" }}</text>
        </view>
        <!-- 详情 / 结算页的价格与数量展示 (左下红字价格，右下数量) -->
        <view v-if="layout !== 'card'" class="price-qty-row">
          <view class="price-box">
            <text class="currency">¥</text>
            <text class="price-num">{{ item.unit_price }}</text>
          </view>
          <text class="quantity-num">x{{ item.quantity }}</text>
        </view>
        <!-- 操作区插槽 (如评价晒单操作) -->
        <view v-if="$slots.action" class="item-action-slot">
          <slot name="action" :item="item" />
        </view>
      </view>
      <!-- 订单列表卡片模式的价格与数量展示 (右侧独立列) -->
      <view v-if="layout === 'card'" class="item-price-side">
        <text class="item-price">¥{{ item.unit_price }}</text>
        <text class="item-quantity">x{{ item.quantity }}</text>
        <view v-if="$slots.action" class="item-action-slot">
          <slot name="action" :item="item" />
        </view>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { formatSpecs } from "@/utils"

export interface OrderGoodsItem {
  id?: number
  order_item_id?: number
  product_id?: number
  sku_id?: number
  product_name: string
  image_url?: string | null
  sku_spec_values?: string | null
  unit_price: string
  quantity: number
  deliverable?: boolean
  undeliverable_reason?: string
  can_review?: boolean
  review_id?: number
  review_status?: number
  review_block_reason?: string
}

interface Props {
  items?: OrderGoodsItem[]
  layout?: "default" | "card"
}

withDefaults(defineProps<Props>(), {
  items: () => [],
  layout: "default",
})
</script>

<style lang="scss" scoped>
.order-goods-list {
  /* 默认明细排版 (结算 checkout 与 详情 detail) */
  .item-row {
    display: flex;
    padding: 16rpx 0;
    border-bottom: 2rpx solid #f9f9f9;

    &:last-child {
      border-bottom: none;
      padding-bottom: 0;
    }

    .item-img {
      width: 150rpx;
      height: 150rpx;
      border-radius: 12rpx;
      background: #f5f5f5;
      margin-right: 20rpx;
      flex-shrink: 0;
    }

    .item-meta {
      flex: 1;
      display: flex;
      flex-direction: column;
      justify-content: space-between;

      .item-title {
        font-size: 28rpx;
        color: #333333;
        font-weight: 500;
        line-height: 1.4;
        display: -webkit-box;
        -webkit-line-clamp: 2;
        -webkit-box-orient: vertical;
        overflow: hidden;
      }

      .item-specs {
        font-size: 22rpx;
        color: #999999;
        background: #f8f8f8;
        padding: 4rpx 12rpx;
        border-radius: 6rpx;
        align-self: flex-start;
        margin-top: 6rpx;
      }

      .undeliverable-badge {
        margin-top: 8rpx;
        align-self: flex-start;
        background: #fff1f0;
        border: 1rpx solid #ffa39e;
        border-radius: 6rpx;
        padding: 2rpx 10rpx;

        .undeliverable-badge-text {
          font-size: 20rpx;
          color: #cf1322;
        }
      }

      .price-qty-row {
        display: flex;
        align-items: center;
        justify-content: space-between;
        margin-top: 10rpx;

        .price-box {
          display: flex;
          align-items: baseline;

          .currency {
            font-size: 22rpx;
            color: #e02e24;
            font-weight: bold;
          }

          .price-num {
            font-size: 32rpx;
            color: #e02e24;
            font-weight: bold;
          }
        }

        .quantity-num {
          font-size: 24rpx;
          color: #999999;
        }
      }

      .item-action-slot {
        margin-top: 12rpx;
        display: flex;
        justify-content: flex-end;
      }
    }
  }

  /* 订单列表卡片紧凑排版 (index 列表页) */
  &.layout-card {
    padding: 12rpx 0;

    .item-row {
      display: flex;
      align-items: center;
      margin-bottom: 16rpx;
      padding: 0;
      border-bottom: none;

      &:last-child {
        margin-bottom: 0;
      }

      .item-img {
        width: 130rpx;
        height: 130rpx;
        border-radius: 10rpx;
        background: #f5f5f5;
        margin-right: 18rpx;
        flex-shrink: 0;
      }

      .item-meta {
        flex: 1;
        min-width: 0;
        margin-right: 16rpx;
        display: block;

        .item-title {
          font-size: 26rpx;
          color: #333333;
          line-height: 1.4;
          display: -webkit-box;
          -webkit-line-clamp: 2;
          -webkit-box-orient: vertical;
          overflow: hidden;
          margin-bottom: 6rpx;
        }

        .item-specs {
          font-size: 22rpx;
          color: #999999;
          background: #f8f8f8;
          padding: 2rpx 10rpx;
          border-radius: 6rpx;
          display: inline-block;
        }
      }

      .item-price-side {
        display: flex;
        flex-direction: column;
        align-items: flex-end;
        flex-shrink: 0;

        .item-price {
          font-size: 28rpx;
          font-weight: bold;
          color: #333333;
          margin-bottom: 6rpx;
        }

        .item-quantity {
          font-size: 22rpx;
          color: #999999;
        }
      }
    }
  }
}
</style>
