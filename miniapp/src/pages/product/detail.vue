<template>
  <view class="page-container">
    <!-- 全屏加载中状态 -->
    <view v-if="loading" class="center-state-box">
      <text class="state-icon">⏳</text>
      <text class="state-text">商品加载中...</text>
    </view>

    <!-- 错误 / 商品不存在或下架状态 -->
    <view v-else-if="errorMsg || !product" class="center-state-box">
      <text class="state-icon">🛍️</text>
      <text class="error-msg">{{ errorMsg || "商品不存在或已下架" }}</text>
      <view class="state-btn-row">
        <button class="state-btn back" @tap="goBack">返回上一页</button>
        <button class="state-btn retry" @tap="fetchDetail">重新加载</button>
      </view>
    </view>

    <!-- 商品详情主体 -->
    <view v-else class="detail-body">
      <!-- 1. 顶部商品轮播图 -->
      <view class="swiper-container">
        <swiper
          class="swiper-box"
          circular
          :autoplay="swiperImages.length > 1"
          :interval="4000"
          :duration="500"
          @change="handleSwiperChange"
        >
          <swiper-item
            v-for="(imgUrl, idx) in swiperImages"
            :key="idx"
            class="swiper-item"
          >
            <image
              class="swiper-img"
              :src="imgUrl"
              mode="aspectFill"
            />
          </swiper-item>
        </swiper>

        <!-- 页码指示器 -->
        <view v-if="swiperImages.length > 1" class="swiper-indicator">
          <text class="indicator-text">{{ currentSwiperIndex + 1 }} / {{ swiperImages.length }}</text>
        </view>
      </view>

      <!-- 2~6. 价格与商品标题信息卡片 -->
      <view class="info-card">
        <view class="price-row">
          <view class="price-box">
            <text class="currency">¥</text>
            <text class="price-val">{{ displayPrice }}</text>
            <text v-if="showMarketPrice" class="market-price">¥{{ displayMarketPrice }}</text>
          </view>
          <text class="sales-badge">已售 {{ product.sales }}</text>
        </view>

        <view class="title-row">
          <text class="product-title">{{ product.name }}</text>
          <text v-if="product.subtitle" class="product-subtitle">{{ product.subtitle }}</text>
        </view>
      </view>

      <!-- 7. SKU 规格选择区 -->
      <view v-if="product.specs && product.specs.length > 0" class="spec-card">
        <view
          v-for="spec in product.specs"
          :key="spec.id"
          class="spec-group"
        >
          <view class="spec-header">
            <text class="spec-name">{{ spec.name }}</text>
            <text v-if="!selectedSpecs[spec.id]" class="spec-state-tip">请选择</text>
          </view>

          <view class="spec-tags-wrap">
            <view
              v-for="val in spec.values"
              :key="val.id"
              class="spec-tag"
              :class="{
                active: selectedSpecs[spec.id] === val.id,
                disabled: isValueDisabled(spec.id, val.id)
              }"
              @tap="handleSelectSpec(spec.id, val.id)"
            >
              <text class="tag-text">{{ val.name }}</text>
            </view>
          </view>
        </view>

        <!-- 规格选择汇总提示 -->
        <view class="selected-summary">
          <text class="summary-label">已选规格：</text>
          <text class="summary-content">{{ selectedSpecsSummary }}</text>
        </view>
      </view>

      <!-- 10. 数量选择区 -->
      <view class="quantity-card">
        <view class="quantity-header">
          <text class="quantity-label">购买数量</text>
          <text class="stock-hint" :class="{ 'out-of-stock': isOutOfStock }">
            {{ isOutOfStock ? "当前缺货" : `剩余库存: ${displayStock}` }}
          </text>
        </view>

        <view class="stepper">
          <button
            class="step-btn minus"
            :disabled="quantity <= 1 || isOutOfStock"
            @tap="handleMinus"
          >
            -
          </button>
          <view class="step-num-box">
            <text class="step-num">{{ quantity }}</text>
          </view>
          <button
            class="step-btn plus"
            :disabled="quantity >= displayStock || isOutOfStock"
            @tap="handlePlus"
          >
            +
          </button>
        </view>
      </view>

      <!-- 8. 商品参数区 -->
      <view v-if="product.params && product.params.length > 0" class="param-card">
        <view class="card-section-title">
          <text>商品参数</text>
        </view>
        <view class="param-list">
          <view
            v-for="(param, idx) in product.params"
            :key="idx"
            class="param-row"
          >
            <text class="param-name">{{ param.name }}</text>
            <text class="param-value">{{ param.value }}</text>
          </view>
        </view>
      </view>

      <!-- 9. 商品图文详情 (支持兼容微信小程序 rich-text) -->
      <view class="detail-card">
        <view class="card-section-title">
          <text>商品详情</text>
        </view>
        <view class="rich-text-wrapper">
          <rich-text :nodes="formattedDetailNodes" />
        </view>
      </view>

      <!-- 底部操作栏占位空间 -->
      <view class="bottom-bar-placeholder" />

      <!-- 10. 底部固定操作栏 -->
      <view class="bottom-bar">
        <view class="nav-action" @tap="goToHome">
          <text class="nav-icon">🏠</text>
          <text class="nav-text">首页</text>
        </view>
        <view class="nav-action" @tap="goToCart">
          <text class="nav-icon">🛒</text>
          <text class="nav-text">购物车</text>
        </view>

        <button
          class="add-cart-btn"
          :class="{ disabled: isOutOfStock }"
          :disabled="isOutOfStock"
          @tap="handleAddToCart"
        >
          加入购物车
        </button>

        <button
          class="buy-now-btn"
          :class="{ disabled: isOutOfStock }"
          :disabled="isOutOfStock"
          @tap="handleBuyNow"
        >
          {{ isOutOfStock ? "暂时缺货" : "立即购买" }}
        </button>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref, computed } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import {
  getProductDetail,
  type ProductDetailData,
  type ProductSKUItem,
} from "@/api/product"
import { addToCart } from "@/api/cart"
import { getStoredToken } from "@/services/request"
import { yuanToFen } from "@/utils/money"

const productId = ref<number>(0)
const loading = ref<boolean>(true)
const errorMsg = ref<string>("")
const product = ref<ProductDetailData | null>(null)

// 选中的规格: spec_id -> value_id
const selectedSpecs = ref<Record<number, number>>({})

// 购买数量 (默认 1)
const quantity = ref<number>(1)

// 轮播图当前索引
const currentSwiperIndex = ref<number>(0)

// 轮播图列表 (主图 + 画册)
const swiperImages = computed<string[]>(() => {
  if (!product.value) return []
  const list: string[] = []
  if (product.value.main_image_url) {
    list.push(product.value.main_image_url)
  }
  if (product.value.gallery_urls && product.value.gallery_urls.length > 0) {
    for (const url of product.value.gallery_urls) {
      if (url && !list.includes(url)) {
        list.push(url)
      }
    }
  }
  return list
})

// 匹配当前规格完全对应的 SKU
const currentSKU = computed<ProductSKUItem | null>(() => {
  if (!product.value) return null
  const { specs, skus } = product.value
  if (!specs || specs.length === 0) {
    return skus && skus.length > 0 ? skus[0] : null
  }

  // 必须所有规格维度都有选定值
  const selectedKeys = Object.keys(selectedSpecs.value)
  if (selectedKeys.length !== specs.length) {
    return null
  }

  return (
    skus.find((sku) =>
      specs.every((spec) => {
        const sv = sku.spec_values.find((v) => v.spec_id === spec.id)
        return sv && sv.value_id === selectedSpecs.value[spec.id]
      })
    ) || null
  )
})

// 显示售价
const displayPrice = computed<string>(() => {
  if (currentSKU.value) {
    return currentSKU.value.price
  }
  return product.value?.price || "0.00"
})

// 显示划线价
const displayMarketPrice = computed<string>(() => {
  if (currentSKU.value) {
    return currentSKU.value.market_price
  }
  return product.value?.market_price || ""
})

// 是否显示划线价
const showMarketPrice = computed<boolean>(() => {
  if (!displayMarketPrice.value) return false
  return yuanToFen(displayMarketPrice.value) > yuanToFen(displayPrice.value)
})

// 当前规格或商品的总可用库存
const displayStock = computed<number>(() => {
  if (currentSKU.value) {
    return currentSKU.value.stock
  }
  if (!product.value?.skus) return 0
  return product.value.skus.reduce((sum, s) => sum + (s.stock > 0 ? s.stock : 0), 0)
})

// 是否缺货
const isOutOfStock = computed<boolean>(() => {
  if (currentSKU.value) {
    return currentSKU.value.stock <= 0
  }
  return displayStock.value <= 0
})

// 已选规格文案摘要
const selectedSpecsSummary = computed<string>(() => {
  if (!product.value?.specs || product.value.specs.length === 0) {
    return "统一规格"
  }
  const parts: string[] = []
  for (const spec of product.value.specs) {
    const valId = selectedSpecs.value[spec.id]
    if (valId) {
      const val = spec.values.find((v) => v.id === valId)
      if (val) {
        parts.push(`${spec.name}: ${val.name}`)
      }
    } else {
      parts.push(`未选 ${spec.name}`)
    }
  }
  return parts.join("；")
})

// 兼容各端富文本图片展示
const formattedDetailNodes = computed<string>(() => {
  const raw = product.value?.detail || ""
  if (!raw.trim()) {
    return '<p style="color:#909399;font-size:26rpx;text-align:center;padding:40rpx 0;">暂无商品介绍</p>'
  }
  return raw
    .replace(/<img[^>]*>/gi, (match) => {
      if (/style\s*=\s*"/i.test(match)) {
        return match.replace(
          /style\s*=\s*"([^"]*)"/i,
          'style="$1;max-width:100%!important;height:auto!important;display:block;"'
        )
      }
      return match.replace(
        /<img/i,
        '<img style="max-width:100%!important;height:auto!important;display:block;"'
      )
    })
    .replace(/<p[^>]*>/gi, (match) => {
      if (/style\s*=\s*"/i.test(match)) {
        return match.replace(/style\s*=\s*"([^"]*)"/i, 'style="$1;margin:0;padding:0;"')
      }
      return match.replace(/<p/i, '<p style="margin:0;padding:0;"')
    })
})

// 检查某一选项是否不可售 (在当前其他已选规格的前提下，是否存在任意有效 SKU)
function isValueDisabled(specId: number, valId: number): boolean {
  if (!product.value?.skus) return true

  return !product.value.skus.some((sku) => {
    // 该 SKU 必须包含当前规格候选值
    const matchThis = sku.spec_values.some(
      (sv) => sv.spec_id === specId && sv.value_id === valId
    )
    if (!matchThis) return false

    // 同时该 SKU 必须满足其他已选中的规格维度
    for (const [sIdStr, vId] of Object.entries(selectedSpecs.value)) {
      const otherSpecId = Number(sIdStr)
      if (otherSpecId === specId) continue
      const matchOther = sku.spec_values.some(
        (sv) => sv.spec_id === otherSpecId && sv.value_id === vId
      )
      if (!matchOther) return false
    }

    return true
  })
}

// 用户点击规格选项
function handleSelectSpec(specId: number, valId: number): void {
  if (isValueDisabled(specId, valId)) return

  if (selectedSpecs.value[specId] === valId) {
    // 再次点击取消选择
    delete selectedSpecs.value[specId]
  } else {
    selectedSpecs.value[specId] = valId
  }

  // 数量自适应：不能超过新选中规格的库存
  if (currentSKU.value && currentSKU.value.stock > 0) {
    if (quantity.value > currentSKU.value.stock) {
      quantity.value = currentSKU.value.stock
    }
  }
}

// 数量递减
function handleMinus(): void {
  if (quantity.value > 1) {
    quantity.value -= 1
  }
}

// 数量递增
function handlePlus(): void {
  const max = currentSKU.value ? currentSKU.value.stock : displayStock.value
  if (quantity.value < max) {
    quantity.value += 1
  } else {
    uni.showToast({ title: "已达最大可用库存", icon: "none" })
  }
}

// 轮播切换
function handleSwiperChange(e: any): void {
  currentSwiperIndex.value = e.detail.current || 0
}

// 立即购买
function handleBuyNow(): void {
  if (!product.value) return

  // 1. 规格校验
  if (product.value.specs && product.value.specs.length > 0) {
    for (const spec of product.value.specs) {
      if (!selectedSpecs.value[spec.id]) {
        uni.showToast({ title: `请选择 ${spec.name}`, icon: "none" })
        return
      }
    }
  }

  // 2. 匹配有效 SKU 校验
  if (!currentSKU.value) {
    uni.showToast({ title: "请选择完整规格", icon: "none" })
    return
  }

  // 3. 库存校验
  if (currentSKU.value.stock <= 0) {
    uni.showToast({ title: "该规格暂无库存", icon: "none" })
    return
  }

  // 4. 购买件数校验
  if (quantity.value <= 0 || quantity.value > currentSKU.value.stock) {
    uni.showToast({ title: "购买数量无效", icon: "none" })
    return
  }

  uni.setStorageSync("checkout_payload", {
    source: "buy_now",
    items: [
      {
        sku_id: currentSKU.value.id,
        quantity: quantity.value,
      },
    ],
  })

  uni.navigateTo({
    url: `/pages/order/checkout?sku_id=${currentSKU.value.id}&quantity=${quantity.value}`,
  })
}

// 加入购物车
async function handleAddToCart(): Promise<void> {
  if (!product.value) return

  // 1. 规格校验
  if (product.value.specs && product.value.specs.length > 0) {
    for (const spec of product.value.specs) {
      if (!selectedSpecs.value[spec.id]) {
        uni.showToast({ title: `请选择 ${spec.name}`, icon: "none" })
        return
      }
    }
  }

  // 2. 匹配完整有效 SKU
  if (!currentSKU.value) {
    uni.showToast({ title: "请选择规格", icon: "none" })
    return
  }

  // 3. 购买件数校验
  if (quantity.value <= 0) {
    uni.showToast({ title: "购买数量无效", icon: "none" })
    return
  }

  // 4. 会员登录校验
  const token = getStoredToken()
  if (!token) {
    uni.showToast({ title: "请先登录", icon: "none" })
    setTimeout(() => {
      uni.navigateTo({
        url: `/pages/auth/login?redirect=${encodeURIComponent(
          `/pages/product/detail?id=${productId.value}`
        )}`,
      })
    }, 800)
    return
  }

  try {
    uni.showLoading({ title: "正在加入...", mask: true })
    const res = await addToCart({
      sku_id: currentSKU.value.id,
      quantity: quantity.value,
    })
    uni.hideLoading()
    if (res && res.code === 1) {
      uni.showToast({ title: "已加入购物车", icon: "success" })
    } else {
      uni.showToast({ title: res?.message || "加入购物车失败", icon: "none" })
    }
  } catch (err: any) {
    uni.hideLoading()
    uni.showToast({ title: err?.message || "加入购物车失败", icon: "none" })
  }
}

// 导航操作
function goToHome(): void {
  uni.switchTab({ url: "/pages/index/index" })
}

function goToCart(): void {
  uni.switchTab({ url: "/pages/cart/index" })
}

function goToCategory(): void {
  uni.switchTab({ url: "/pages/category/index" })
}

function goBack(): void {
  const pages = getCurrentPages()
  if (pages && pages.length > 1) {
    uni.navigateBack()
  } else {
    uni.switchTab({ url: "/pages/index/index" })
  }
}

// 初始化规格选择
function initSpecSelection(): void {
  selectedSpecs.value = {}
  quantity.value = 1
  currentSwiperIndex.value = 0

  if (!product.value) return
  const { specs, skus } = product.value

  // 无规格或只有一个 SKU 时自动选中唯一有效 SKU
  if (!specs || specs.length === 0 || (skus && skus.length === 1)) {
    if (skus && skus[0]?.spec_values) {
      for (const sv of skus[0].spec_values) {
        selectedSpecs.value[sv.spec_id] = sv.value_id
      }
    }
  }
}

// 请求商品详情
async function fetchDetail(): Promise<void> {
  if (!productId.value) {
    errorMsg.value = "商品不存在或已下架"
    loading.value = false
    return
  }

  loading.value = true
  errorMsg.value = ""

  try {
    const res = await getProductDetail(productId.value)
    product.value = res.data
    initSpecSelection()
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : "获取商品详情失败"
    errorMsg.value = msg
  } finally {
    loading.value = false
  }
}

onLoad((query: Record<string, any> | undefined) => {
  if (query && query.id) {
    productId.value = Number(query.id)
    fetchDetail()
  } else {
    errorMsg.value = "参数错误：商品ID缺失"
    loading.value = false
  }
})
</script>

<style scoped>
.page-container {
  min-height: 100vh;
  background-color: #f7f8fa;
  max-width: 1000rpx;
  margin: 0 auto;
  position: relative;
}

/* 居中状态提示 */
.center-state-box {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 160rpx 40rpx;
  gap: 24rpx;
}

.state-icon {
  font-size: 72rpx;
}

.state-text {
  font-size: 28rpx;
  color: #909399;
}

.error-msg {
  font-size: 30rpx;
  color: #303133;
  font-weight: 500;
  text-align: center;
}

.state-btn-row {
  display: flex;
  gap: 24rpx;
  margin-top: 16rpx;
}

.state-btn {
  font-size: 26rpx;
  border-radius: 36rpx;
  padding: 0 36rpx;
  height: 64rpx;
  line-height: 64rpx;
  margin: 0;
}

.state-btn::after {
  border: none;
}

.state-btn.back {
  background-color: #ffffff;
  color: #606266;
  border: 1rpx solid #dcdfe6;
}

.state-btn.retry {
  background-color: #e02e24;
  color: #ffffff;
}

/* 详情内容流 */
.detail-body {
  display: flex;
  flex-direction: column;
}

/* 1. 轮播图 */
.swiper-container {
  width: 100%;
  height: 750rpx;
  position: relative;
  background-color: #ffffff;
}

.swiper-box {
  width: 100%;
  height: 100%;
}

.swiper-item {
  width: 100%;
  height: 100%;
}

.swiper-img {
  width: 100%;
  height: 100%;
  display: block;
}

.swiper-indicator {
  position: absolute;
  right: 28rpx;
  bottom: 28rpx;
  background-color: rgba(0, 0, 0, 0.45);
  color: #ffffff;
  font-size: 22rpx;
  padding: 4rpx 18rpx;
  border-radius: 20rpx;
}

/* 2~6. 价格与标题卡片 */
.info-card {
  background-color: #ffffff;
  padding: 28rpx 32rpx;
  margin-bottom: 20rpx;
}

.price-row {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin-bottom: 20rpx;
}

.price-box {
  display: flex;
  align-items: baseline;
}

.currency {
  font-size: 28rpx;
  color: #e02e24;
  font-weight: bold;
}

.price-val {
  font-size: 48rpx;
  color: #e02e24;
  font-weight: bold;
  line-height: 1;
}

.market-price {
  font-size: 26rpx;
  color: #909399;
  text-decoration: line-through;
  margin-left: 16rpx;
}

.sales-badge {
  font-size: 24rpx;
  color: #909399;
}

.title-row {
  display: flex;
  flex-direction: column;
  gap: 12rpx;
}

.product-title {
  font-size: 32rpx;
  color: #303133;
  font-weight: 600;
  line-height: 1.4;
  word-break: break-all;
}

.product-subtitle {
  font-size: 24rpx;
  color: #909399;
  line-height: 1.4;
}

/* 7. 规格卡片 */
.spec-card {
  background-color: #ffffff;
  padding: 28rpx 32rpx;
  margin-bottom: 20rpx;
}

.spec-group {
  margin-bottom: 28rpx;
}

.spec-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16rpx;
}

.spec-name {
  font-size: 28rpx;
  color: #303133;
  font-weight: 500;
}

.spec-state-tip {
  font-size: 22rpx;
  color: #e02e24;
}

.spec-tags-wrap {
  display: flex;
  flex-wrap: wrap;
  gap: 16rpx;
}

.spec-tag {
  padding: 12rpx 28rpx;
  background-color: #f7f8fa;
  border-radius: 12rpx;
  border: 2rpx solid transparent;
  transition: all 0.2s;
}

.spec-tag.active {
  background-color: #fff1f0;
  border-color: #e02e24;
}

.spec-tag.active .tag-text {
  color: #e02e24;
  font-weight: 500;
}

.spec-tag.disabled {
  background-color: #f2f3f5;
  opacity: 0.5;
}

.spec-tag.disabled .tag-text {
  color: #c0c4cc;
}

.tag-text {
  font-size: 26rpx;
  color: #606266;
}

.selected-summary {
  padding-top: 16rpx;
  border-top: 1rpx solid #f2f3f5;
  font-size: 24rpx;
  display: flex;
}

.summary-label {
  color: #909399;
}

.summary-content {
  color: #303133;
  font-weight: 500;
}

/* 数量选择卡片 */
.quantity-card {
  background-color: #ffffff;
  padding: 28rpx 32rpx;
  margin-bottom: 20rpx;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.quantity-header {
  display: flex;
  flex-direction: column;
  gap: 6rpx;
}

.quantity-label {
  font-size: 28rpx;
  color: #303133;
  font-weight: 500;
}

.stock-hint {
  font-size: 22rpx;
  color: #909399;
}

.stock-hint.out-of-stock {
  color: #e02e24;
  font-weight: 500;
}

.stepper {
  display: flex;
  align-items: center;
  border: 1rpx solid #dcdfe6;
  border-radius: 8rpx;
  overflow: hidden;
}

.step-btn {
  width: 64rpx;
  height: 64rpx;
  line-height: 64rpx;
  text-align: center;
  background-color: #f7f8fa;
  font-size: 32rpx;
  color: #303133;
  padding: 0;
  margin: 0;
  border-radius: 0;
}

.step-btn::after {
  border: none;
}

.step-btn[disabled] {
  color: #c0c4cc;
  background-color: #f2f3f5;
}

.step-num-box {
  min-width: 72rpx;
  height: 64rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: #ffffff;
}

.step-num {
  font-size: 28rpx;
  color: #303133;
  font-weight: 500;
}

/* 8. 参数卡片 */
.param-card {
  background-color: #ffffff;
  padding: 28rpx 32rpx;
  margin-bottom: 20rpx;
}

.card-section-title {
  font-size: 30rpx;
  font-weight: 600;
  color: #303133;
  padding-left: 12rpx;
  border-left: 6rpx solid #e02e24;
  margin-bottom: 24rpx;
}

.param-list {
  display: flex;
  flex-direction: column;
  gap: 16rpx;
}

.param-row {
  display: flex;
  font-size: 26rpx;
  line-height: 1.4;
}

.param-name {
  width: 160rpx;
  color: #909399;
  flex-shrink: 0;
}

.param-value {
  color: #303133;
  flex: 1;
}

/* 9. 图文详情卡片 */
.detail-card {
  background-color: #ffffff;
  padding: 28rpx 32rpx 40rpx;
  margin-bottom: 20rpx;
}

.rich-text-wrapper {
  font-size: 28rpx;
  color: #303133;
  line-height: 1.6;
}

/* 底部操作栏占位空间 */
.bottom-bar-placeholder {
  height: 120rpx;
}

/* 10. 底部固定操作栏 */
.bottom-bar {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  max-width: 1000rpx;
  margin: 0 auto;
  height: 108rpx;
  background-color: #ffffff;
  border-top: 1rpx solid #ebeef5;
  display: flex;
  align-items: center;
  padding: 0 24rpx;
  gap: 20rpx;
  z-index: 100;
  box-sizing: border-box;
}

.nav-action {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 0 16rpx;
}

.nav-icon {
  font-size: 32rpx;
  line-height: 1;
}

.nav-text {
  font-size: 20rpx;
  color: #606266;
  margin-top: 6rpx;
}

.add-cart-btn {
  flex: 1;
  background: linear-gradient(135deg, #ff9800 0%, #ffa726 100%);
  color: #ffffff;
  font-size: 28rpx;
  font-weight: 500;
  border-radius: 44rpx;
  height: 80rpx;
  line-height: 80rpx;
  text-align: center;
  margin: 0;
  border: none;
}

.add-cart-btn::after {
  border: none;
}

.add-cart-btn.disabled,
.add-cart-btn[disabled] {
  background: #c0c4cc;
  color: #ffffff;
  opacity: 0.8;
}

.buy-now-btn {
  flex: 1;
  background: linear-gradient(135deg, #e02e24 0%, #ff5b4f 100%);
  color: #ffffff;
  font-size: 28rpx;
  font-weight: 500;
  border-radius: 44rpx;
  height: 80rpx;
  line-height: 80rpx;
  text-align: center;
  margin: 0;
  border: none;
}

.buy-now-btn::after {
  border: none;
}

.buy-now-btn.disabled,
.buy-now-btn[disabled] {
  background: #c0c4cc;
  color: #ffffff;
  opacity: 0.8;
}
</style>
