<template>
  <view class="checkout-container">
    <!-- 顶部收货地址卡片 -->
    <view class="address-section" @tap="handleSelectAddress">
      <view v-if="address" class="address-card">
        <view class="address-icon-wrap">
          <text class="address-icon">📍</text>
        </view>
        <view class="address-content">
          <view class="user-row">
            <text class="user-name">{{ address.name }}</text>
            <text class="user-phone">{{ address.mobile }}</text>
            <text v-if="address.is_default === 1" class="default-badge">默认</text>
          </view>
          <view class="detail-row">
            <text class="detail-text">
              {{ address.province }} {{ address.city }} {{ address.district }} {{ address.address }}
            </text>
          </view>
        </view>
        <view class="address-arrow">
          <text class="arrow-text">›</text>
        </view>
      </view>

      <view v-else class="empty-address-card">
        <view class="empty-icon-wrap">
          <text class="empty-icon">➕</text>
        </view>
        <view class="empty-content">
          <text class="empty-title">添加收货地址</text>
          <text class="empty-sub">请选择或新建您的配送收货地址</text>
        </view>
        <view class="address-arrow">
          <text class="arrow-text">›</text>
        </view>
      </view>

      <!-- 彩色信封装饰条 -->
      <view class="envelope-line"></view>
    </view>

    <!-- 商品清单卡片 -->
    <view class="items-card">
      <view class="card-header">
        <text class="header-icon">🏬</text>
        <text class="header-title">购物清单</text>
        <text class="header-tag">正品保障 · 顺丰速运</text>
      </view>

      <view class="items-list">
        <view v-for="item in previewData?.items || []" :key="item.sku_id" class="item-row">
          <image class="item-image" :src="item.image_url || '/static/logo.png'" mode="aspectFill" />
          <view class="item-info">
            <text class="item-name">{{ item.product_name }}</text>
            <text v-if="formatSpecs(item.sku_spec_values)" class="item-specs">
              {{ formatSpecs(item.sku_spec_values) }}
            </text>
            <view class="price-qty-row">
              <view class="price-box">
                <text class="currency">¥</text>
                <text class="price-num">{{ item.unit_price }}</text>
              </view>
              <text class="quantity-num">x{{ item.quantity }}</text>
            </view>
          </view>
        </view>
      </view>
    </view>

    <!-- 买家备注卡片 -->
    <view class="remark-card">
      <view class="remark-header">
        <text class="remark-label">订单备注</text>
      </view>
      <input
        class="remark-input"
        v-model="orderRemark"
        placeholder="选填，请先与商家协商一致（50字以内）"
        maxlength="50"
      />
    </view>

    <!-- 费用明细卡片 -->
    <view class="cost-card">
      <view class="cost-row">
        <text class="cost-label">商品总额</text>
        <text class="cost-value">¥{{ previewData?.total_amount || "0.00" }}</text>
      </view>
      <view class="cost-row">
        <text class="cost-label">运费</text>
        <text class="cost-value free-shipping">免运费</text>
      </view>
      <view class="cost-row">
        <text class="cost-label">活动优惠</text>
        <text class="cost-value">¥{{ previewData?.discount_amount || "0.00" }}</text>
      </view>
    </view>

    <!-- 底部固定提交栏 -->
    <view class="bottom-submit-bar">
      <view class="price-summary">
        <text class="total-desc">共 {{ previewData?.total_quantity || 0 }} 件，实付款:</text>
        <view class="total-price-box">
          <text class="currency-symbol">¥</text>
          <text class="total-amount-integer">{{ priceParts.integer }}</text>
          <text class="total-amount-decimal">.{{ priceParts.decimal }}</text>
        </view>
      </view>
      <button
        class="submit-btn"
        :class="{ disabled: isSubmitting || !address }"
        :loading="isSubmitting"
        @tap="handleSubmitOrder"
      >
        <text class="submit-btn-text">{{ isSubmitting ? "正在提交..." : "提交订单" }}</text>
      </button>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { checkoutPreview, createOrder } from "@/api/order"
import type {
  CheckoutPreviewResponse,
  CheckoutAddressResponse,
  CheckoutItemRequest,
} from "@/api/order"
import { getStoredToken } from "@/services/request"

const skuID = ref<number>(0)
const quantity = ref<number>(1)
const address = ref<CheckoutAddressResponse | null>(null)
const previewData = ref<CheckoutPreviewResponse | null>(null)
const orderRemark = ref<string>("")
const isSubmitting = ref<boolean>(false)

// 金额拆分整数与小数
const priceParts = computed(() => {
  const pay = previewData.value?.pay_amount || "0.00"
  const [integer, decimal = "00"] = pay.split(".")
  return { integer, decimal }
})

function parseQueryParams(query?: any) {
  let sId = query?.sku_id
  let qty = query?.quantity

  if (!sId) {
    try {
      const pages = typeof getCurrentPages === "function" ? getCurrentPages() : []
      if (pages && pages.length > 0) {
        const curPage = pages[pages.length - 1] as any
        const opt = curPage?.options || curPage?.$page?.options
        if (opt?.sku_id) sId = opt.sku_id
        if (opt?.quantity) qty = opt.quantity
      }
    } catch {}
  }

  if (typeof window !== "undefined" && window.location) {
    if (!sId && window.location.search) {
      const sp = new URLSearchParams(window.location.search)
      if (sp.has("sku_id")) sId = sp.get("sku_id")
      if (sp.has("quantity")) qty = sp.get("quantity")
    }
    if (!sId && window.location.hash && window.location.hash.includes("?")) {
      const hp = new URLSearchParams(window.location.hash.split("?")[1])
      if (hp.has("sku_id")) sId = hp.get("sku_id")
      if (hp.has("quantity")) qty = hp.get("quantity")
    }
  }

  if (sId) {
    skuID.value = Number(sId)
  }
  if (qty) {
    quantity.value = Number(qty) || 1
  }
}

let isInitialized = false
let isListenerBound = false

async function initCheckout(query?: any): Promise<void> {
  parseQueryParams(query)

  if (!isListenerBound) {
    uni.$on("selectAddress", onAddressSelected)
    isListenerBound = true
  }

  if (skuID.value <= 0) {
    return
  }

  if (isInitialized) return
  isInitialized = true

  const token = getStoredToken()
  if (!token) {
    uni.showToast({ title: "请先登录", icon: "none" })
    setTimeout(() => {
      uni.navigateTo({
        url: `/pages/auth/login?redirect=${encodeURIComponent(
          `/pages/order/checkout?sku_id=${skuID.value}&quantity=${quantity.value}`
        )}`,
      })
    }, 800)
    return
  }

  await loadCheckoutPreview()
}

onLoad((query: any) => {
  initCheckout(query)
})

onMounted(async () => {
  await initCheckout()
  if (!isInitialized && skuID.value <= 0) {
    setTimeout(() => {
      initCheckout()
      if (skuID.value <= 0) {
        uni.showToast({ title: "购买商品无效", icon: "none" })
      }
    }, 100)
  }
})

onUnmounted(() => {
  if (isListenerBound) {
    uni.$off("selectAddress", onAddressSelected)
    isListenerBound = false
  }
})

function onAddressSelected(addr: any): void {
  if (!addr || !addr.id) return
  address.value = addr as CheckoutAddressResponse
  // 使用新选择的地址重新计算预览
  loadCheckoutPreview(addr.id)
}

async function loadCheckoutPreview(addrId?: number): Promise<void> {
  if (skuID.value <= 0 || quantity.value <= 0) {
    uni.showToast({ title: "购买商品无效", icon: "none" })
    return
  }

  const items: CheckoutItemRequest[] = [
    {
      sku_id: skuID.value,
      quantity: quantity.value,
    },
  ]

  uni.showLoading({ title: "计算订单明细...", mask: true })
  try {
    const res = await checkoutPreview({
      address_id: addrId || address.value?.id || null,
      items,
    })
    if (res && res.code === 1 && res.data) {
      previewData.value = res.data
      if (res.data.address) {
        address.value = res.data.address
      }
    }
  } catch (err: any) {
    uni.showToast({
      title: err?.message || "订单结算初始化失败",
      icon: "none",
      duration: 2500,
    })
  } finally {
    uni.hideLoading()
  }
}

function handleSelectAddress(): void {
  const currentId = address.value?.id || 0
  uni.navigateTo({
    url: `/pages/address/index?select=1&id=${currentId}`,
  })
}

function formatSpecs(specValStr: string | null | undefined): string {
  if (!specValStr) return ""
  try {
    const arr = JSON.parse(specValStr)
    if (Array.isArray(arr)) {
      return arr.map((item: any) => item.value_name || item.name || "").filter(Boolean).join(" · ")
    }
  } catch {}
  return specValStr
}

function generateRequestId(): string {
  const timestamp = Date.now().toString(36)
  const randomPart = Math.random().toString(36).substring(2, 10)
  return `req-${timestamp}-${randomPart}`
}

async function handleSubmitOrder(): Promise<void> {
  if (isSubmitting.value) return

  if (!address.value || !address.value.id) {
    uni.showToast({ title: "请先添加或选择收货地址", icon: "none" })
    return
  }

  if (!skuID.value || quantity.value <= 0) {
    uni.showToast({ title: "订单商品规格无效", icon: "none" })
    return
  }

  isSubmitting.value = true
  uni.showLoading({ title: "正在提交订单...", mask: true })

  try {
    const reqID = generateRequestId()
    const res = await createOrder({
      request_id: reqID,
      address_id: address.value.id,
      items: [
        {
          sku_id: skuID.value,
          quantity: quantity.value,
        },
      ],
      remark: orderRemark.value.trim(),
    })

    if (!res || res.code !== 1 || !res.data) {
      throw new Error(res?.message || "提交订单失败")
    }

    const order = res.data

    uni.showToast({
      title: "下单成功",
      icon: "success",
    })

    // 跳转至订单详情页
    setTimeout(() => {
      uni.redirectTo({
        url: `/pages/order/detail?id=${order.id}`,
      })
    }, 600)
  } catch (err: any) {
    uni.showToast({
      title: err?.message || "提交订单失败，请稍后重试",
      icon: "none",
      duration: 3000,
    })
  } finally {
    isSubmitting.value = false
    uni.hideLoading()
  }
}
</script>

<style lang="scss" scoped>
.checkout-container {
  min-height: 100vh;
  background-color: #f7f7f7;
  padding: 20rpx 24rpx 140rpx;
  box-sizing: border-box;
}

/* 地址区域 */
.address-section {
  background-color: #ffffff;
  border-radius: 20rpx;
  overflow: hidden;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.04);
  margin-bottom: 20rpx;
  position: relative;

  .address-card {
    display: flex;
    align-items: center;
    padding: 28rpx 24rpx;

    .address-icon-wrap {
      width: 64rpx;
      height: 64rpx;
      border-radius: 50%;
      background: #fff1f0;
      display: flex;
      align-items: center;
      justify-content: center;
      margin-right: 20rpx;
      flex-shrink: 0;

      .address-icon {
        font-size: 32rpx;
      }
    }

    .address-content {
      flex: 1;

      .user-row {
        display: flex;
        align-items: center;
        margin-bottom: 8rpx;

        .user-name {
          font-size: 30rpx;
          font-weight: bold;
          color: #333333;
          margin-right: 16rpx;
        }

        .user-phone {
          font-size: 26rpx;
          color: #666666;
          margin-right: 16rpx;
        }

        .default-badge {
          font-size: 20rpx;
          background: #e02e24;
          color: #ffffff;
          padding: 2rpx 12rpx;
          border-radius: 6rpx;
        }
      }

      .detail-row {
        .detail-text {
          font-size: 26rpx;
          color: #444444;
          line-height: 1.4;
        }
      }
    }

    .address-arrow {
      margin-left: 16rpx;
      .arrow-text {
        font-size: 40rpx;
        color: #cccccc;
      }
    }
  }

  .empty-address-card {
    display: flex;
    align-items: center;
    padding: 32rpx 24rpx;

    .empty-icon-wrap {
      width: 64rpx;
      height: 64rpx;
      border-radius: 50%;
      background: #f5f5f5;
      display: flex;
      align-items: center;
      justify-content: center;
      margin-right: 20rpx;

      .empty-icon {
        font-size: 28rpx;
        color: #999999;
      }
    }

    .empty-content {
      flex: 1;
      .empty-title {
        font-size: 30rpx;
        font-weight: bold;
        color: #333333;
        display: block;
        margin-bottom: 6rpx;
      }
      .empty-sub {
        font-size: 24rpx;
        color: #999999;
      }
    }

    .address-arrow {
      margin-left: 16rpx;
      .arrow-text {
        font-size: 40rpx;
        color: #cccccc;
      }
    }
  }

  /* 经典彩色信封横条 */
  .envelope-line {
    height: 6rpx;
    background: repeating-linear-gradient(
      -45deg,
      #e02e24 0,
      #e02e24 16rpx,
      transparent 16rpx,
      transparent 24rpx,
      #1890ff 24rpx,
      #1890ff 40rpx,
      transparent 40rpx,
      transparent 48rpx
    );
  }
}

/* 商品清单卡片 */
.items-card {
  background-color: #ffffff;
  border-radius: 20rpx;
  padding: 24rpx;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.04);
  margin-bottom: 20rpx;

  .card-header {
    display: flex;
    align-items: center;
    margin-bottom: 24rpx;

    .header-icon {
      font-size: 28rpx;
      margin-right: 12rpx;
    }

    .header-title {
      font-size: 28rpx;
      font-weight: bold;
      color: #333333;
      flex: 1;
    }

    .header-tag {
      font-size: 22rpx;
      color: #999999;
    }
  }

  .items-list {
    .item-row {
      display: flex;
      align-items: center;
      margin-bottom: 24rpx;

      &:last-child {
        margin-bottom: 0;
      }

      .item-image {
        width: 140rpx;
        height: 140rpx;
        border-radius: 12rpx;
        background-color: #f5f5f5;
        margin-right: 20rpx;
        flex-shrink: 0;
      }

      .item-info {
        flex: 1;
        min-width: 0;

        .item-name {
          font-size: 28rpx;
          color: #333333;
          font-weight: 500;
          display: -webkit-box;
          -webkit-box-orient: vertical;
          -webkit-line-clamp: 1;
          overflow: hidden;
          margin-bottom: 8rpx;
        }

        .item-specs {
          font-size: 22rpx;
          color: #888888;
          background: #f7f7f7;
          padding: 4rpx 12rpx;
          border-radius: 6rpx;
          display: inline-block;
          margin-bottom: 12rpx;
        }

        .price-qty-row {
          display: flex;
          align-items: baseline;
          justify-content: space-between;

          .price-box {
            color: #e02e24;
            font-weight: bold;

            .currency {
              font-size: 22rpx;
            }
            .price-num {
              font-size: 32rpx;
            }
          }

          .quantity-num {
            font-size: 24rpx;
            color: #888888;
          }
        }
      }
    }
  }
}

/* 备注卡片 */
.remark-card {
  background-color: #ffffff;
  border-radius: 20rpx;
  padding: 24rpx;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.04);
  margin-bottom: 20rpx;

  .remark-header {
    margin-bottom: 12rpx;
    .remark-label {
      font-size: 26rpx;
      font-weight: bold;
      color: #333333;
    }
  }

  .remark-input {
    width: 100%;
    font-size: 26rpx;
    color: #333333;
    background: #f8f8f8;
    padding: 16rpx 20rpx;
    border-radius: 10rpx;
    box-sizing: border-box;
  }
}

/* 费用明细卡片 */
.cost-card {
  background-color: #ffffff;
  border-radius: 20rpx;
  padding: 24rpx;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.04);
  margin-bottom: 20rpx;

  .cost-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 16rpx;

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
  }
}

/* 底部吸底操作栏 */
.bottom-submit-bar {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  height: 110rpx;
  background-color: #ffffff;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 32rpx;
  box-shadow: 0 -4rpx 16rpx rgba(0, 0, 0, 0.06);
  z-index: 100;

  .price-summary {
    display: flex;
    align-items: baseline;

    .total-desc {
      font-size: 24rpx;
      color: #666666;
      margin-right: 10rpx;
    }

    .total-price-box {
      color: #e02e24;
      font-weight: bold;

      .currency-symbol {
        font-size: 24rpx;
      }
      .total-amount-integer {
        font-size: 38rpx;
      }
      .total-amount-decimal {
        font-size: 26rpx;
      }
    }
  }

  .submit-btn {
    margin: 0;
    padding: 0 48rpx;
    height: 80rpx;
    line-height: 80rpx;
    border-radius: 40rpx;
    background: linear-gradient(135deg, #ff6034, #ee0a24);
    color: #ffffff;
    font-size: 28rpx;
    font-weight: bold;
    border: none;

    &.disabled {
      opacity: 0.6;
    }

    &::after {
      border: none;
    }
  }
}
</style>
