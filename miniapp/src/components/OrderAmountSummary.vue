<template>
  <view class="order-amount-summary cost-card">
    <view class="cost-row">
      <text class="cost-label">商品总额</text>
      <text class="cost-value">¥{{ totalAmount || "0.00" }}</text>
    </view>
    <view class="cost-row">
      <text class="cost-label">运费</text>
      <text v-if="isCalculatingFreight" class="cost-value">计算中...</text>
      <text v-else-if="emptyFreight" class="cost-value">--</text>
      <text
        v-else-if="!freightAmount || freightAmount === '0.00' || freightAmount === '0'"
        class="cost-value free-shipping"
      >
        免运费
      </text>
      <text v-else class="cost-value">¥{{ freightAmount }}</text>
    </view>
    <view class="cost-row">
      <text class="cost-label">活动优惠</text>
      <text class="cost-value">¥{{ discountAmount || "0.00" }}</text>
    </view>
    <view v-if="shouldShowPayRow" class="cost-row total-row">
      <text class="total-label">{{ payLabel }}</text>
      <text class="total-amount">¥{{ payAmount || "0.00" }}</text>
    </view>
  </view>
</template>

<script setup lang="ts">
import { computed } from "vue"

interface Props {
  totalAmount?: string | number
  freightAmount?: string | number | null
  discountAmount?: string | number
  payAmount?: string | number | null
  isCalculatingFreight?: boolean
  emptyFreight?: boolean
  payLabel?: string
  showPayRow?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  totalAmount: "0.00",
  freightAmount: "0.00",
  discountAmount: "0.00",
  payAmount: undefined,
  isCalculatingFreight: false,
  emptyFreight: false,
  payLabel: "实付款",
  showPayRow: undefined,
})

const shouldShowPayRow = computed(() => {
  if (props.showPayRow !== undefined) return props.showPayRow
  return props.payAmount !== undefined && props.payAmount !== null
})
</script>

<style lang="scss" scoped>
.cost-card {
  background-color: #ffffff;
  border-radius: 20rpx;
  padding: 24rpx 28rpx;
  margin-bottom: 20rpx;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.04);

  .cost-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 18rpx;

    &:last-child {
      margin-bottom: 0;
    }

    .cost-label {
      font-size: 26rpx;
      color: #666666;
    }

    .cost-value {
      font-size: 26rpx;
      color: #333333;
      font-weight: 500;

      &.free-shipping {
        color: #52c41a;
      }
    }

    &.total-row {
      margin-top: 24rpx;
      padding-top: 20rpx;
      border-top: 2rpx solid #f5f5f5;
      margin-bottom: 0;

      .total-label {
        font-size: 28rpx;
        font-weight: bold;
        color: #333333;
      }

      .total-amount {
        font-size: 36rpx;
        font-weight: bold;
        color: #e02e24;
      }
    }
  }
}
</style>
