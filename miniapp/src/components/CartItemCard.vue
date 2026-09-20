<template>
  <view
    class="cart-item-card"
    :class="{ 'invalid-card': !item.available }"
  >
    <!-- 勾选区 / 失效状态 -->
    <view
      v-if="item.available || isEditMode"
      class="item-checkbox"
      @tap="handleToggleSelect"
    >
      <view
        class="checkbox-round"
        :class="{ checked: selected }"
      >
        <text v-if="selected" class="checkbox-tick">✓</text>
      </view>
    </view>
    <view v-else class="item-checkbox disabled">
      <text class="invalid-badge">失效</text>
    </view>

    <!-- 商品图片 -->
    <image
      class="item-img"
      :class="{ 'invalid-img': !item.available }"
      :src="item.image_url || '/static/logo.png'"
      mode="aspectFill"
      @tap="handleClickProduct"
    />

    <!-- 商品信息 -->
    <view class="item-info">
      <view class="item-title-row">
        <text
          class="item-name"
          :class="{ 'invalid-text': !item.available }"
          @tap="handleClickProduct"
        >
          {{ item.product_name }}
        </text>
        <text
          v-if="showDelete"
          class="item-delete-btn"
          @tap.stop="handleDelete"
        >
          ✕
        </text>
      </view>

      <!-- 规格标签 -->
      <view v-if="formattedSpecs" class="item-specs-tag">
        <text class="specs-text">{{ formattedSpecs }}</text>
      </view>

      <!-- 有效商品且库存不足提示 (quantity > stock) -->
      <view v-if="item.available && item.quantity > item.stock" class="stock-warning">
        <text class="stock-warning-text">库存不足，当前仅剩 {{ item.stock }} 件，请调整数量</text>
      </view>

      <!-- 失效商品原因提示 -->
      <view v-if="!item.available" class="invalid-reason-tag">
        <text class="reason-text">{{ item.unavailable_reason || "商品已下架" }}</text>
      </view>

      <!-- 价格与数量行 -->
      <view class="item-bottom-row">
        <view class="price-box" :class="{ 'invalid-text': !item.available }">
          <text class="currency">¥</text>
          <text class="price-val">{{ item.price }}</text>
        </view>

        <!-- 有效商品步进器 -->
        <view v-if="item.available" class="stepper">
          <button
            class="step-btn"
            :disabled="item.quantity <= 1 || updating"
            :class="{ disabled: item.quantity <= 1 || updating }"
            @tap.stop="handleChangeQty(-1)"
          >
            -
          </button>
          <text class="step-num">{{ item.quantity }}</text>
          <button
            class="step-btn"
            :disabled="item.quantity >= item.stock || item.quantity >= 999 || updating"
            :class="{ disabled: item.quantity >= item.stock || item.quantity >= 999 || updating }"
            @tap.stop="handleChangeQty(1)"
          >
            +
          </button>
        </view>

        <!-- 失效商品纯文本数量 -->
        <text v-else class="invalid-qty">x{{ item.quantity }}</text>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { CartItem } from '@/api/cart'
import { formatCartSpecs } from '@/utils'

const props = withDefaults(
  defineProps<{
    item: CartItem
    selected?: boolean
    updating?: boolean
    isEditMode?: boolean
    showDelete?: boolean
  }>(),
  {
    selected: false,
    updating: false,
    isEditMode: false,
    showDelete: true,
  }
)

const emit = defineEmits<{
  (e: 'toggle-select', id: number): void
  (e: 'change-quantity', item: CartItem, delta: number): void
  (e: 'delete', item: CartItem): void
  (e: 'click-product', productId: number): void
}>()

const formattedSpecs = computed(() => {
  return formatCartSpecs(props.item.spec_values)
})

function handleToggleSelect() {
  emit('toggle-select', props.item.id)
}

function handleChangeQty(delta: number) {
  emit('change-quantity', props.item, delta)
}

function handleDelete() {
  if (props.updating) return
  emit('delete', props.item)
}

function handleClickProduct() {
  if (props.item.product_id > 0) {
    emit('click-product', props.item.product_id)
  }
}
</script>

<style scoped>
.cart-item-card {
  background-color: #ffffff;
  border-radius: 20rpx;
  padding: 24rpx;
  margin-bottom: 20rpx;
  display: flex;
  align-items: center;
  box-shadow: 0 2rpx 8rpx rgba(0, 0, 0, 0.04);
}

.invalid-card {
  background-color: #fafafa;
}

/* 勾选框 */
.item-checkbox {
  padding-right: 20rpx;
  display: flex;
  align-items: center;
  justify-content: center;
}

.checkbox-round {
  width: 40rpx;
  height: 40rpx;
  border-radius: 50%;
  border: 2rpx solid #dcdfe6;
  display: flex;
  align-items: center;
  justify-content: center;
  box-sizing: border-box;
}

.checkbox-round.checked {
  background-color: #e02e24;
  border-color: #e02e24;
}

.checkbox-tick {
  color: #ffffff;
  font-size: 24rpx;
  font-weight: bold;
}

.invalid-badge {
  font-size: 20rpx;
  color: #909399;
  background-color: #ebeef5;
  padding: 4rpx 10rpx;
  border-radius: 6rpx;
}

/* 商品图片 */
.item-img {
  width: 176rpx;
  height: 176rpx;
  border-radius: 12rpx;
  flex-shrink: 0;
  background-color: #f5f7fa;
}

.invalid-img {
  opacity: 0.5;
}

/* 商品信息 */
.item-info {
  flex: 1;
  min-width: 0;
  margin-left: 20rpx;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  min-height: 176rpx;
}

.item-title-row {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
}

.item-name {
  font-size: 28rpx;
  color: #303133;
  line-height: 1.4;
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  flex: 1;
}

.item-delete-btn {
  font-size: 28rpx;
  color: #c0c4cc;
  padding: 0 8rpx 8rpx 16rpx;
  cursor: pointer;
}

.item-specs-tag {
  align-self: flex-start;
  background-color: #f5f7fa;
  padding: 4rpx 12rpx;
  border-radius: 6rpx;
  margin-top: 6rpx;
}

.specs-text {
  font-size: 22rpx;
  color: #909399;
}

.stock-warning {
  align-self: flex-start;
  background-color: #fff7e6;
  padding: 4rpx 12rpx;
  border-radius: 6rpx;
  margin-top: 6rpx;
}

.stock-warning-text {
  font-size: 22rpx;
  color: #fa8c16;
  font-weight: 500;
}

.invalid-reason-tag {
  align-self: flex-start;
  background-color: #fff1f0;
  padding: 4rpx 12rpx;
  border-radius: 6rpx;
  margin-top: 6rpx;
}

.reason-text {
  font-size: 22rpx;
  color: #f5222d;
}

.item-bottom-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 12rpx;
  flex-wrap: wrap;
  gap: 12rpx;
}

.price-box {
  display: flex;
  align-items: baseline;
}

.currency {
  font-size: 24rpx;
  color: #e02e24;
  font-weight: 600;
}

.price-val {
  font-size: 32rpx;
  color: #e02e24;
  font-weight: bold;
}

.invalid-text {
  color: #909399 !important;
}

.invalid-qty {
  font-size: 26rpx;
  color: #909399;
}

/* 步进器 Stepper */
.stepper {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  border: 1rpx solid #e4e7ed;
  border-radius: 8rpx;
  overflow: hidden;
}

.step-btn {
  width: 56rpx;
  height: 52rpx;
  line-height: 52rpx;
  text-align: center;
  background-color: #fafafa;
  color: #303133;
  font-size: 28rpx;
  font-weight: bold;
  padding: 0;
  margin: 0;
  border: none;
  border-radius: 0;
}

.step-btn::after {
  border: none;
}

.step-btn.disabled,
.step-btn[disabled] {
  color: #c0c4cc;
  background-color: #f5f7fa;
}

.step-num {
  min-width: 64rpx;
  height: 52rpx;
  line-height: 52rpx;
  text-align: center;
  font-size: 26rpx;
  color: #303133;
  background-color: #ffffff;
  padding: 0 8rpx;
}
</style>
