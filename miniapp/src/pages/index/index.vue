<template>
  <view class="home-container">
    <!-- 顶部商城品牌信息区 (来自 Bootstrap Shop / Site 配置) -->
    <view class="shop-header">
      <view class="shop-brand-row">
        <image
          v-if="bootstrapData?.shop?.logo"
          class="shop-logo"
          :src="bootstrapData.shop.logo"
          mode="aspectFit"
        />
        <view v-else class="shop-logo-placeholder">
          <text class="logo-emoji">🛍️</text>
        </view>

        <view class="shop-info">
          <text class="shop-name">{{ shopName }}</text>
          <text class="shop-desc">{{ shopDesc }}</text>
        </view>
      </view>

      <!-- 搜索栏快捷入口 -->
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
              v-if="cat.image_url"
              class="cat-img"
              :src="cat.image_url"
              mode="aspectFill"
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

    <!-- 启动接口与系统调试状态卡片 -->
    <view class="debug-card">
      <view class="debug-header">
        <view class="title-with-dot">
          <text class="status-dot" :class="{ ready: !!bootstrapData, fail: !!bootstrapError }"></text>
          <text class="debug-title">启动配置 (Bootstrap Status)</text>
        </view>
        <button class="refresh-btn" :loading="loading" @tap="handleRefresh">
          刷新配置
        </button>
      </view>

      <view v-if="loading" class="state-hint loading">
        <text>正在同步最新启动配置...</text>
      </view>
      <view v-else-if="bootstrapError" class="state-hint error">
        <text class="error-msg">加载失败: {{ bootstrapError }}</text>
        <text class="hint-msg">可点击右上角刷新，或确认本地 Go 后端是否启动</text>
      </view>
      <view v-else-if="bootstrapData" class="state-hint success">
        <text class="info-row">✓ 商城: {{ bootstrapData.shop?.name || "已加载" }}</text>
        <text class="info-row">✓ 小程序: {{ bootstrapData.mp?.name || "已关联" }}</text>
        <text class="info-row">✓ 分类项: 共 {{ bootstrapData.categories?.length || 0 }} 个分类</text>
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
import { computed, onMounted } from "vue"
import { useBootstrap } from "@/services/bootstrap"

const { bootstrapData, loading, error: bootstrapError, fetchBootstrap } = useBootstrap()

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
  return [
    { id: 1, name: "数码电子", image_url: "", parent_id: 0, sort: 1 },
    { id: 2, name: "服装服饰", image_url: "", parent_id: 0, sort: 2 },
    { id: 3, name: "食品生鲜", image_url: "", parent_id: 0, sort: 3 },
    { id: 4, name: "家居日用", image_url: "", parent_id: 0, sort: 4 },
    { id: 5, name: "美妆个护", image_url: "", parent_id: 0, sort: 5 },
    { id: 6, name: "运动户外", image_url: "", parent_id: 0, sort: 6 },
    { id: 7, name: "母婴玩具", image_url: "", parent_id: 0, sort: 7 },
    { id: 8, name: "图书文具", image_url: "", parent_id: 0, sort: 8 },
  ]
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

async function handleRefresh(): Promise<void> {
  uni.showLoading({ title: "同步中..." })
  try {
    await fetchBootstrap(true)
    uni.showToast({ title: "配置已更新", icon: "success" })
  } catch {
    uni.showToast({ title: "同步失败", icon: "none" })
  } finally {
    uni.hideLoading()
  }
}

onMounted(async () => {
  if (!bootstrapData.value) {
    await fetchBootstrap()
  }
})
</script>

<style scoped>
.home-container {
  min-height: 100vh;
  background-color: #f7f8fa;
  padding-bottom: 40rpx;
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

/* 调试卡片 */
.debug-card {
  background-color: #ffffff;
  margin: 24rpx 28rpx;
  border-radius: 20rpx;
  padding: 24rpx;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.04);
}

.debug-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16rpx;
}

.title-with-dot {
  display: flex;
  align-items: center;
  gap: 10rpx;
}

.status-dot {
  width: 14rpx;
  height: 14rpx;
  border-radius: 7rpx;
  background-color: #909399;
}

.status-dot.ready {
  background-color: #67c23a;
}

.status-dot.fail {
  background-color: #f56c6c;
}

.debug-title {
  font-size: 26rpx;
  font-weight: 600;
  color: #606266;
}

.refresh-btn {
  font-size: 22rpx;
  color: #e02e24;
  background-color: #fff1f0;
  border-radius: 20rpx;
  height: 52rpx;
  line-height: 52rpx;
  padding: 0 20rpx;
  margin: 0;
}

.refresh-btn::after {
  border: none;
}

.state-hint {
  padding: 12rpx 16rpx;
  border-radius: 8rpx;
  font-size: 24rpx;
}

.state-hint.loading {
  background-color: #f4f4f5;
  color: #909399;
}

.state-hint.error {
  background-color: #fef0f0;
  color: #f56c6c;
  display: flex;
  flex-direction: column;
  gap: 6rpx;
}

.hint-msg {
  font-size: 20rpx;
  color: #909399;
}

.state-hint.success {
  background-color: #f0f9eb;
  color: #67c23a;
  display: flex;
  flex-direction: column;
  gap: 6rpx;
}

.info-row {
  font-size: 22rpx;
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
