<template>
  <view class="home-container">
    <!-- 顶部商城品牌信息区 (来自 Bootstrap Shop / Site 配置) -->
    <view class="shop-header">
      <view class="shop-brand-row">
        <image
          v-if="bootstrapData?.shop?.logo && !logoLoadFailed"
          class="shop-logo"
          :src="bootstrapData.shop.logo"
          mode="aspectFit"
          @error="logoLoadFailed = true"
        />
        <view v-else class="shop-logo-placeholder">
          <text class="logo-emoji">🛍️</text>
        </view>

        <view class="shop-info">
          <text class="shop-name">{{ shopName }}</text>
          <text class="shop-desc">{{ shopDesc }}</text>
        </view>
      </view>

      <!-- 搜索栏快捷入口 (点击跳转分类搜索页) -->
      <view class="search-box" @tap="goToCategory">
        <text class="search-icon">🔍</text>
        <text class="search-placeholder">搜索心仪好物、热门品类</text>
      </view>
    </view>

    <!-- 分类金刚区 (动态渲染 Bootstrap Categories) -->
    <view class="category-section">
      <view class="section-header">
        <text class="section-title">热门分类</text>
        <text class="more-link" @tap="goToCategory">查看全部 &gt;</text>
      </view>

      <view v-if="displayCategories.length > 0" class="category-grid">
        <view
          v-for="cat in displayCategories"
          :key="cat.id"
          class="category-item"
          @tap="goToCategoryWithId(cat.id)"
        >
          <view class="cat-icon-wrap">
            <image
              v-if="cat.image_url && !failedCatImages[cat.id]"
              class="cat-img"
              :src="cat.image_url"
              mode="aspectFill"
              @error="failedCatImages[cat.id] = true"
            />
            <text v-else class="cat-emoji">🎁</text>
          </view>
          <text class="cat-name">{{ cat.name }}</text>
        </view>
      </view>

      <view v-else class="empty-category">
        <text class="empty-text">暂无分类数据，请检查后台配置</text>
      </view>
    </view>

    <!-- 推荐商品 / 猜你喜欢瀑布流 -->
    <view class="recommend-section">
      <view class="section-header">
        <text class="section-title">推荐商品</text>
        <text class="sub-hint">精选好物 · 品质保障</text>
      </view>

      <!-- 加载中状态 -->
      <view v-if="productsLoading && recommendedProducts.length === 0" class="loading-box">
        <text class="loading-text">正在加载推荐商品...</text>
      </view>

      <!-- 加载失败 -->
      <view v-else-if="productsError && recommendedProducts.length === 0" class="error-box">
        <text class="error-text">{{ productsError }}</text>
        <button class="retry-btn" @tap="loadRecommendedProducts">重新加载</button>
      </view>

      <!-- 空数据 -->
      <view v-else-if="recommendedProducts.length === 0" class="empty-box">
        <text class="empty-text">暂无推荐商品</text>
      </view>

      <!-- 两列商品瀑布流 -->
      <view v-else class="product-grid">
        <ProductCard
          v-for="product in recommendedProducts"
          :key="product.id"
          :item="product"
          layout="grid"
        />
      </view>
    </view>

    <!-- 官方客服与服务支持卡片 (来自 Bootstrap Shop Service 配置) -->
    <view v-if="hasServiceInfo" class="service-card">
      <view class="card-header">
        <text class="card-title">官方服务与支持</text>
        <text class="service-badge">{{ bootstrapData?.shop?.service_name || "专属客服" }}</text>
      </view>

      <view class="service-grid">
        <!-- 客服电话一键拨打 -->
        <view
          v-if="bootstrapData?.shop?.service_phone"
          class="service-item"
          @tap="handleCallPhone(bootstrapData.shop.service_phone)"
        >
          <view class="service-icon-box phone-bg">
            <text class="service-icon">📞</text>
          </view>
          <view class="service-meta">
            <text class="service-label">客服电话</text>
            <text class="service-value highlight">{{ bootstrapData.shop.service_phone }}</text>
          </view>
          <text class="action-tag">拨打</text>
        </view>

        <!-- 客服微信一键复制 -->
        <view
          v-if="bootstrapData?.shop?.service_wechat"
          class="service-item"
          @tap="handleCopyWechat(bootstrapData.shop.service_wechat)"
        >
          <view class="service-icon-box wechat-bg">
            <text class="service-icon">💬</text>
          </view>
          <view class="service-meta">
            <text class="service-label">客服微信</text>
            <text class="service-value highlight">{{ bootstrapData.shop.service_wechat }}</text>
          </view>
          <text class="action-tag">复制</text>
        </view>
      </view>
    </view>

    <!-- 站点版权与 ICP 备案底部 (来自 Bootstrap Site) -->
    <view class="site-footer">
      <text v-if="bootstrapData?.site?.copyright" class="footer-text">
        {{ bootstrapData.site.copyright }}
      </text>
      <text v-else class="footer-text">© 2026 AI Shop. All Rights Reserved.</text>

      <text v-if="bootstrapData?.site?.icp" class="footer-text icp-text">
        {{ bootstrapData.site.icp }}
      </text>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from "vue"
import { useBootstrap } from "@/services/bootstrap"
import { getProductList, type ProductListItem } from "@/api/product"
import ProductCard from "@/components/ProductCard.vue"

const { bootstrapData, fetchBootstrap } = useBootstrap()

const logoLoadFailed = ref<boolean>(false)
const failedCatImages = ref<Record<number, boolean>>({})

// 推荐商品状态
const recommendedProducts = ref<ProductListItem[]>([])
const productsLoading = ref<boolean>(false)
const productsError = ref<string>("")

const shopName = computed(() => {
  return (
    bootstrapData.value?.shop?.name ||
    bootstrapData.value?.site?.title ||
    "AI Shop 商城"
  )
})

const shopDesc = computed(() => {
  return (
    bootstrapData.value?.shop?.description ||
    bootstrapData.value?.site?.description ||
    "正品好货 · 极速物流 · 贴心售后"
  )
})

const displayCategories = computed(() => {
  const cats = bootstrapData.value?.categories
  if (cats && cats.length > 0) {
    return cats.slice(0, 8)
  }
  return []
})

const hasServiceInfo = computed(() => {
  const shop = bootstrapData.value?.shop
  return !!(shop?.service_phone || shop?.service_wechat)
})

function goToCategory(): void {
  uni.switchTab({
    url: "/pages/category/index",
  })
}

function goToCategoryWithId(id: number): void {
  try {
    uni.setStorageSync("active_category_id", id)
  } catch {}
  uni.switchTab({
    url: "/pages/category/index",
  })
}

function handleCallPhone(phoneNumber: string): void {
  uni.makePhoneCall({
    phoneNumber,
    fail: (err) => {
      console.warn("拨打电话失败或取消:", err)
    },
  })
}

function handleCopyWechat(wechat: string): void {
  uni.setClipboardData({
    data: wechat,
    success: () => {
      uni.showToast({ title: "客服微信已复制", icon: "success" })
    },
  })
}

async function loadRecommendedProducts(): Promise<void> {
  productsLoading.value = true
  productsError.value = ""
  try {
    const res = await getProductList({
      page: 1,
      page_size: 20,
      sort: "default",
    })
    recommendedProducts.value = res.data || []
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : "加载推荐商品失败"
    productsError.value = msg
  } finally {
    productsLoading.value = false
  }
}

onMounted(async () => {
  if (!bootstrapData.value) {
    await fetchBootstrap()
  }
  await loadRecommendedProducts()
})
</script>

<style scoped>
.home-container {
  min-height: 100vh;
  background-color: #f7f8fa;
  padding-bottom: 40rpx;
  max-width: 1000rpx;
  margin: 0 auto;
}

/* 顶部商城品牌区 */
.shop-header {
  background: linear-gradient(135deg, #e02e24 0%, #ff5b4f 100%);
  padding: 40rpx 32rpx 32rpx;
  border-bottom-left-radius: 36rpx;
  border-bottom-right-radius: 36rpx;
  box-shadow: 0 8rpx 24rpx rgba(224, 46, 36, 0.2);
}

.shop-brand-row {
  display: flex;
  align-items: center;
  gap: 24rpx;
  margin-bottom: 28rpx;
}

.shop-logo {
  width: 108rpx;
  height: 108rpx;
  border-radius: 20rpx;
  background-color: #ffffff;
  border: 4rpx solid rgba(255, 255, 255, 0.8);
}

.shop-logo-placeholder {
  width: 108rpx;
  height: 108rpx;
  border-radius: 20rpx;
  background-color: #ffffff;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 4rpx solid rgba(255, 255, 255, 0.8);
}

.logo-emoji {
  font-size: 56rpx;
}

.shop-info {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.shop-name {
  font-size: 38rpx;
  font-weight: bold;
  color: #ffffff;
  margin-bottom: 8rpx;
  letter-spacing: 1rpx;
}

.shop-desc {
  font-size: 24rpx;
  color: rgba(255, 255, 255, 0.88);
  line-height: 1.4;
}

.search-box {
  display: flex;
  align-items: center;
  gap: 12rpx;
  background-color: #ffffff;
  border-radius: 36rpx;
  padding: 16rpx 28rpx;
  box-shadow: 0 4rpx 12rpx rgba(0, 0, 0, 0.06);
}

.search-icon {
  font-size: 28rpx;
}

.search-placeholder {
  font-size: 26rpx;
  color: #909399;
}

/* 分类金刚区 */
.category-section {
  background-color: #ffffff;
  margin: 24rpx 28rpx;
  border-radius: 20rpx;
  padding: 28rpx 24rpx;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.04);
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 24rpx;
}

.section-title {
  font-size: 30rpx;
  font-weight: 600;
  color: #303133;
  padding-left: 12rpx;
  border-left: 6rpx solid #e02e24;
}

.more-link {
  font-size: 24rpx;
  color: #909399;
}

.sub-hint {
  font-size: 22rpx;
  color: #909399;
}

.category-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  row-gap: 24rpx;
  column-gap: 16rpx;
}

.category-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12rpx;
}

.cat-icon-wrap {
  width: 96rpx;
  height: 96rpx;
  border-radius: 48rpx;
  background-color: #fff1f0;
  display: flex;
  align-items: center;
  justify-content: center;
}

.cat-img {
  width: 96rpx;
  height: 96rpx;
  border-radius: 48rpx;
}

.cat-emoji {
  font-size: 44rpx;
}

.cat-name {
  font-size: 24rpx;
  color: #303133;
  text-align: center;
}

.empty-category {
  padding: 40rpx 0;
  text-align: center;
}

.empty-text {
  font-size: 26rpx;
  color: #909399;
}

/* 推荐商品区 */
.recommend-section {
  margin: 24rpx 28rpx;
}

.product-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 20rpx;
}

.loading-box,
.empty-box,
.error-box {
  padding: 60rpx 0;
  text-align: center;
  background-color: #ffffff;
  border-radius: 20rpx;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16rpx;
}

.loading-text {
  font-size: 26rpx;
  color: #909399;
}

.error-text {
  font-size: 26rpx;
  color: #f56c6c;
}

.retry-btn {
  font-size: 24rpx;
  color: #ffffff;
  background-color: #e02e24;
  border-radius: 30rpx;
  padding: 0 32rpx;
  height: 56rpx;
  line-height: 56rpx;
  margin: 0;
}

.retry-btn::after {
  border: none;
}

/* 客服与支持卡片 */
.service-card {
  background-color: #ffffff;
  margin: 24rpx 28rpx;
  border-radius: 20rpx;
  padding: 28rpx 24rpx;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.04);
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 20rpx;
}

.card-title {
  font-size: 30rpx;
  font-weight: 600;
  color: #303133;
  padding-left: 12rpx;
  border-left: 6rpx solid #e02e24;
}

.service-badge {
  font-size: 22rpx;
  color: #e02e24;
  background-color: #fff1f0;
  padding: 4rpx 14rpx;
  border-radius: 8rpx;
}

.service-grid {
  display: flex;
  flex-direction: column;
  gap: 16rpx;
}

.service-item {
  display: flex;
  align-items: center;
  background-color: #f8f9fa;
  border-radius: 14rpx;
  padding: 16rpx 20rpx;
  gap: 16rpx;
}

.service-icon-box {
  width: 68rpx;
  height: 68rpx;
  border-radius: 34rpx;
  display: flex;
  align-items: center;
  justify-content: center;
}

.phone-bg {
  background-color: #fef0f0;
}

.wechat-bg {
  background-color: #f0f9eb;
}

.service-icon {
  font-size: 32rpx;
}

.service-meta {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.service-label {
  font-size: 22rpx;
  color: #909399;
}

.service-value {
  font-size: 28rpx;
  color: #303133;
  font-weight: 500;
}

.service-value.highlight {
  color: #e02e24;
}

.action-tag {
  font-size: 22rpx;
  color: #ffffff;
  background-color: #e02e24;
  padding: 6rpx 20rpx;
  border-radius: 20rpx;
}

/* 底部版权信息 */
.site-footer {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-top: 36rpx;
  gap: 8rpx;
}

.footer-text {
  font-size: 22rpx;
  color: #c0c4cc;
}

.icp-text {
  font-size: 20rpx;
}
</style>
