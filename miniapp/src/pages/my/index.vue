<template>
  <view class="profile-page">
    <NetworkBanner />
    <view class="profile-header">
      <!-- #ifdef MP-WEIXIN -->
      <button v-if="isLoggedIn" class="avatar-button" open-type="chooseAvatar" @chooseavatar="onChooseAvatar">
        <image
          v-if="currentAvatarUrl && !avatarFailed"
          class="avatar-image"
          :src="currentAvatarUrl"
          mode="aspectFill"
          @error="avatarFailed = true"
        />
        <view v-else class="avatar-placeholder">
          <image class="avatar-placeholder-icon" :src="userIcon" mode="aspectFit" />
        </view>
      </button>
      <view v-else class="avatar-button" @tap="goToLogin">
        <view class="avatar-placeholder">
          <image class="avatar-placeholder-icon" :src="userIcon" mode="aspectFit" />
        </view>
      </view>
      <!-- #endif -->
      <!-- #ifndef MP-WEIXIN -->
      <view class="avatar-button" @tap="handleAvatarClick">
        <image
          v-if="currentAvatarUrl && !avatarFailed"
          class="avatar-image"
          :src="currentAvatarUrl"
          mode="aspectFill"
          @error="avatarFailed = true"
        />
        <view v-else class="avatar-placeholder">
          <image class="avatar-placeholder-icon" :src="userIcon" mode="aspectFit" />
        </view>
      </view>
      <!-- #endif -->

      <view class="profile-main">
        <view v-if="isLoggedIn" class="nickname-row">
          <input
            v-model="displayNickname"
            class="nickname-input"
            type="nickname"
            placeholder="设置昵称"
            placeholder-class="nickname-placeholder"
            @blur="onNicknameBlur"
          />
          <image class="edit-icon" :src="editIcon" mode="aspectFit" />
        </view>
        <text v-else class="login-title" @tap="goToLogin">登录 / 注册</text>
        <text class="member-note">
          {{ isLoggedIn ? `会员号 ${member?.id || "--"}` : "登录后查看订单和会员服务" }}
        </text>
      </view>

      <!-- #ifdef MP-WEIXIN -->
      <button
        v-if="!isLoggedIn"
        class="header-login-button"
        :loading="authLoading"
        :disabled="authLoading"
        @tap="handleWechatLogin"
      >
        微信一键登录
      </button>
      <!-- #endif -->
      <!-- #ifndef MP-WEIXIN -->
      <button v-if="!isLoggedIn" class="header-login-button" @tap="goToLogin">立即登录</button>
      <!-- #endif -->
    </view>

    <view class="section-card order-section">
      <view class="section-heading" @tap="handleNavigateToOrders(0)">
        <text class="section-title">我的订单</text>
        <view class="heading-link">
          <text>全部订单</text>
          <image class="arrow-icon" :src="arrowIcon" mode="aspectFit" />
        </view>
      </view>

      <view class="order-grid">
        <view
          v-for="item in orderEntries"
          :key="item.label"
          class="shortcut-item"
          @tap="item.status === -1 ? handleNavigateToAfterSale() : handleNavigateToOrders(item.status)"
        >
          <image class="order-icon" :src="item.icon" mode="aspectFit" />
          <text class="shortcut-label">{{ item.label }}</text>
        </view>
      </view>
    </view>

    <view class="section-card service-section">
      <view class="section-heading static-heading">
        <text class="section-title">常用服务</text>
        <text class="service-slogan">安心购物 · 贴心售后</text>
      </view>

      <view class="service-grid">
        <view
          v-for="item in visibleServiceEntries"
          :key="item.action"
          class="shortcut-item service-item"
          @tap="handleServiceAction(item.action)"
        >
          <image class="service-icon" :src="item.icon" mode="aspectFit" />
          <text class="shortcut-label">{{ item.label }}</text>
          <text v-if="item.action === 'address' && isLoggedIn && addresses.length" class="shortcut-note">
            {{ addresses.length }} 个
          </text>
        </view>
      </view>
      <button v-if="isLoggedIn" class="logout-link" @tap="handleServiceAction('logout')">
        退出当前账号
      </button>
    </view>

    <view class="recommend-section">
      <view class="recommend-heading">
        <view class="heading-line"></view>
        <text class="recommend-title">猜你喜欢</text>
        <view class="heading-line"></view>
      </view>

      <view v-if="productsLoading && recommendedProducts.length === 0" class="state-box">
        <text>正在加载好物...</text>
      </view>
      <view v-else-if="productsError && recommendedProducts.length === 0" class="state-box error-state">
        <text>{{ productsError }}</text>
        <button class="retry-button" @tap="loadRecommendedProducts(true)">重新加载</button>
      </view>
      <view v-else-if="recommendedProducts.length === 0" class="state-box">
        <text>暂无推荐商品</text>
      </view>
      <view v-else class="product-grid">
        <ProductCard
          v-for="product in recommendedProducts"
          :key="product.id"
          :item="product"
          layout="grid"
        />
      </view>

      <view v-if="loadingMore" class="load-more-state">正在加载更多...</view>
      <view v-else-if="moreError" class="load-more-state error-state" @tap="loadRecommendedProducts(false)">
        <text>{{ moreError }}，点击重试</text>
      </view>
      <view v-else-if="!hasMore && recommendedProducts.length > 0" class="load-more-state">
        已经到底了
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { onPullDownRefresh, onReachBottom, onShow } from "@dcloudio/uni-app"
import { storeToRefs } from "pinia"
import ProductCard from "@/components/ProductCard.vue"
import NetworkBanner from "@/components/NetworkBanner.vue"
import { getProductList, type ProductListItem } from "@/api/product"
import { useAppStore } from "@/stores/app"
import { useCartStore } from "@/stores/cart"
import { useMemberStore } from "@/stores/member"
import { useNetwork } from "@/composables/useNetwork"
import { resolveImageUrl } from "@/utils"
import userIcon from "remixicon/icons/User & Faces/user-3-line.svg"
import editIcon from "remixicon/icons/Design/edit-line.svg"
import arrowIcon from "remixicon/icons/Arrows/arrow-right-s-line.svg"
import paymentIcon from "remixicon/icons/Finance/wallet-3-line.svg"
import shipmentIcon from "remixicon/icons/Finance/shopping-bag-3-line.svg"
import receiptIcon from "remixicon/icons/Map/truck-line.svg"
import completedIcon from "remixicon/icons/System/checkbox-circle-line.svg"
import refundIcon from "remixicon/icons/Finance/refund-2-line.svg"
import categoryIcon from "remixicon/icons/Buildings/store-2-line.svg"
import cartIcon from "remixicon/icons/Finance/shopping-cart-2-line.svg"
import serviceIcon from "remixicon/icons/Business/customer-service-2-line.svg"
import addressIcon from "remixicon/icons/Map/map-pin-line.svg"
import settingsIcon from "remixicon/icons/System/settings-3-line.svg"
import reviewIcon from "remixicon/icons/System/star-line.svg"

type ServiceAction = "category" | "cart" | "customer" | "address" | "tools" | "logout" | "review"

interface ServiceEntry {
  label: string
  icon: string
  action: ServiceAction
  loggedInOnly?: boolean
}

const appStore = useAppStore()
const cartStore = useCartStore()
const { bootstrapData } = storeToRefs(appStore)
const { fetchBootstrap } = appStore

const memberStore = useMemberStore()
const { token, member, addresses, isLoggedIn, loading: authLoading } = storeToRefs(memberStore)
const {
  loginWithWechat,
  refreshMember,
  updateNickname,
  updateAvatar,
  fetchAddresses,
  logout,
} = memberStore

const orderEntries = [
  { label: "待付款", status: 10, icon: paymentIcon },
  { label: "待发货", status: 20, icon: shipmentIcon },
  { label: "待收货", status: 30, icon: receiptIcon },
  { label: "已完成", status: 40, icon: completedIcon },
  { label: "退款售后", status: -1, icon: refundIcon },
]

const serviceEntries: ServiceEntry[] = [
  { label: "商品分类", icon: categoryIcon, action: "category" },
  { label: "购物车", icon: cartIcon, action: "cart" },
  { label: "我的评价", icon: reviewIcon, action: "review", loggedInOnly: true },
  { label: "专属客服", icon: serviceIcon, action: "customer" },
  { label: "收货地址", icon: addressIcon, action: "address" },
  { label: "测试工具", icon: settingsIcon, action: "tools" },
]

const visibleServiceEntries = computed(() => {
  return serviceEntries.filter((item) => !item.loggedInOnly || isLoggedIn.value)
})

const avatarFailed = ref(false)
const inputNickname = ref("")
const recommendedProducts = ref<ProductListItem[]>([])
const productsLoading = ref(false)
const loadingMore = ref(false)
const productsError = ref("")
const moreError = ref("")
const productPage = ref(1)
const hasMore = ref(true)
const PAGE_SIZE = 10
let requestSeq = 0

const currentAvatarUrl = computed(() => {
  if (!isLoggedIn.value || !member.value?.avatar_url) return ""
  return resolveImageUrl(member.value.avatar_url)
})

const displayNickname = computed({
  get() {
    if (inputNickname.value) return inputNickname.value
    if (member.value?.nickname) return member.value.nickname
    return "用户"
  },
  set(value: string) {
    inputNickname.value = value
  },
})

onMounted(async () => {
  const tasks: Promise<unknown>[] = [loadRecommendedProducts(true)]
  if (!bootstrapData.value) tasks.push(fetchBootstrap())
  if (token.value) {
    tasks.push(refreshMember())
    tasks.push(fetchAddresses())
  }
  await Promise.allSettled(tasks)
})

onShow(() => {
  cartStore.syncTabBarBadge()
  inputNickname.value = ""
  if (token.value) void fetchAddresses()
})

onPullDownRefresh(async () => {
  const tasks: Promise<unknown>[] = [loadRecommendedProducts(true), fetchBootstrap(true)]
  if (token.value) {
    tasks.push(refreshMember())
    tasks.push(fetchAddresses())
  }
  await Promise.allSettled(tasks)
  uni.stopPullDownRefresh()
})

useNetwork(() => {
  return loadRecommendedProducts(true)
})

onReachBottom(() => {
  if (!hasMore.value || productsLoading.value || loadingMore.value) return
  void loadRecommendedProducts(false)
})

async function loadRecommendedProducts(refresh = false): Promise<void> {
  const currentSeq = ++requestSeq
  const targetPage = refresh ? 1 : productPage.value + 1

  if (refresh) {
    hasMore.value = true
    productsError.value = ""
    productsLoading.value = true
  } else {
    loadingMore.value = true
    moreError.value = ""
  }

  const [result] = await Promise.allSettled([
    getProductList({ page: targetPage, page_size: PAGE_SIZE, sort: "default" }),
  ])

  if (currentSeq !== requestSeq) return

  productsLoading.value = false
  loadingMore.value = false
  if (result.status === "rejected") {
    const msg = result.reason instanceof Error ? result.reason.message : "加载推荐商品失败"
    if (refresh || recommendedProducts.value.length === 0) {
      productsError.value = msg
    } else {
      moreError.value = msg
    }
    return
  }

  const list = result.value.code === 1 && Array.isArray(result.value.data) ? result.value.data : []
  productPage.value = targetPage
  if (refresh) {
    recommendedProducts.value = list
  } else {
    const existingIds = new Set(recommendedProducts.value.map((p) => p.id))
    recommendedProducts.value = [
      ...recommendedProducts.value,
      ...list.filter((p) => !existingIds.has(p.id)),
    ]
  }
  hasMore.value = list.length === PAGE_SIZE
}

async function handleWechatLogin(): Promise<void> {
  const [result] = await Promise.allSettled([loginWithWechat()])
  if (result.status === "rejected") {
    uni.showToast({
      title: result.reason instanceof Error ? result.reason.message : "微信登录失败",
      icon: "none",
    })
    return
  }
  uni.showToast({
    title: result.value.is_new_member ? "注册成功" : "登录成功",
    icon: "success",
  })
}

function onChooseAvatar(event: { detail?: { avatarUrl?: string } }): void {
  const filePath = event.detail?.avatarUrl
  if (!filePath) return
  void saveAvatar(filePath)
}

function handleAvatarClick(): void {
  if (!isLoggedIn.value) {
    goToLogin()
    return
  }
  uni.chooseImage({
    count: 1,
    sizeType: ["compressed"],
    sourceType: ["album", "camera"],
    success: (result) => {
      const filePath = result.tempFilePaths[0]
      if (filePath) void saveAvatar(filePath)
    },
  })
}

async function saveAvatar(filePath: string): Promise<void> {
  avatarFailed.value = false
  if (!isLoggedIn.value) return
  uni.showLoading({ title: "上传中..." })
  const [result] = await Promise.allSettled([updateAvatar(filePath)])
  uni.hideLoading()
  uni.showToast({
    title: result.status === "fulfilled" && result.value ? "头像已更新" : "头像上传失败",
    icon: result.status === "fulfilled" && result.value ? "success" : "none",
  })
}

async function onNicknameBlur(event: any): Promise<void> {
  const nickname = (event.detail?.value || inputNickname.value).trim()
  if (!nickname || nickname === member.value?.nickname) return
  inputNickname.value = nickname
  if (!isLoggedIn.value) return
  const [result] = await Promise.allSettled([updateNickname(nickname)])
  inputNickname.value = ""
  if (result.status === "rejected" || !result.value) {
    uni.showToast({ title: "昵称保存失败", icon: "none" })
  }
}

function handleNavigateToOrders(status: number): void {
  if (!isLoggedIn.value) {
    goToLogin()
    return
  }
  uni.navigateTo({
    url: status > 0 ? `/pages/order/index?status=${status}` : "/pages/order/index",
  })
}

function handleNavigateToAfterSale(): void {
  if (!isLoggedIn.value) {
    goToLogin()
    return
  }
  uni.navigateTo({ url: "/pages/aftersale/index" })
}

function handleServiceAction(action: ServiceAction): void {
  if (action === "category") {
    uni.switchTab({ url: "/pages/category/index" })
    return
  }
  if (action === "cart") {
    uni.switchTab({ url: "/pages/cart/index" })
    return
  }
  if (action === "customer") {
    handleCustomerService()
    return
  }
  if (action === "address") {
    if (!isLoggedIn.value) {
      goToLogin()
      return
    }
    uni.navigateTo({ url: "/pages/address/index" })
    return
  }
  if (action === "review") {
    if (!isLoggedIn.value) {
      goToLogin()
      return
    }
    uni.navigateTo({ url: "/pages/review/mine" })
    return
  }
  if (action === "tools") {
    uni.navigateTo({ url: "/pages/test/index" })
    return
  }
  uni.showModal({
    title: "退出登录",
    content: "确定退出当前账号吗？",
    success: (result) => {
      if (result.confirm) {
        logout()
        inputNickname.value = ""
        avatarFailed.value = false
        uni.showToast({ title: "已退出登录", icon: "success" })
      }
    },
  })
}

function handleCustomerService(): void {
  const phone = bootstrapData.value?.shop?.service_phone
  const wechat = bootstrapData.value?.shop?.service_wechat
  if (phone) {
    uni.makePhoneCall({ phoneNumber: phone })
    return
  }
  if (wechat) {
    uni.setClipboardData({ data: wechat })
    return
  }
  uni.showToast({ title: "客服信息暂未配置", icon: "none" })
}

function goToLogin(): void {
  uni.navigateTo({ url: "/pages/auth/login" })
}
</script>

<style scoped>
.profile-page {
  min-height: 100vh;
  box-sizing: border-box;
  padding-bottom: 48rpx;
  background: #f5f5f5;
  color: #202124;
}

.profile-header {
  display: flex;
  align-items: center;
  min-height: calc(210rpx + var(--status-bar-height));
  padding: calc(36rpx + var(--status-bar-height)) 32rpx 28rpx;
  box-sizing: border-box;
  background: #ffffff;
}

.avatar-button {
  width: 116rpx;
  height: 116rpx;
  flex-shrink: 0;
  margin: 0 24rpx 0 0;
  padding: 0;
  overflow: hidden;
  border-radius: 58rpx;
  background: #f1f2f3;
  line-height: normal;
}

.avatar-button::after,
.header-login-button::after,
.retry-button::after {
  border: 0;
}

.avatar-image,
.avatar-placeholder {
  width: 116rpx;
  height: 116rpx;
  border-radius: 58rpx;
}

.avatar-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f1f2f3;
}

.avatar-placeholder-icon {
  width: 58rpx;
  height: 58rpx;
  opacity: 0.45;
}

.profile-main {
  flex: 1;
  min-width: 0;
}

.nickname-row {
  display: flex;
  align-items: center;
  max-width: 340rpx;
}

.nickname-input,
.login-title {
  height: 54rpx;
  font-size: 38rpx;
  line-height: 54rpx;
  font-weight: 600;
  color: #202124;
}

.nickname-input {
  min-width: 0;
  flex: 1;
}

.nickname-placeholder {
  color: #9a9a9a;
}

.edit-icon {
  width: 28rpx;
  height: 28rpx;
  margin-left: 10rpx;
  opacity: 0.45;
}

.member-note {
  display: block;
  margin-top: 10rpx;
  font-size: 23rpx;
  color: #999999;
}

.header-login-button {
  width: auto;
  min-width: 132rpx;
  height: 58rpx;
  margin: 0;
  padding: 0 22rpx;
  border-radius: 30rpx;
  background: #ef3027;
  color: #ffffff;
  font-size: 24rpx;
  line-height: 58rpx;
}

.section-card {
  margin-top: 18rpx;
  padding: 28rpx 28rpx 30rpx;
  background: #ffffff;
}

.section-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 30rpx;
}

.static-heading {
  margin-bottom: 34rpx;
}

.section-title {
  font-size: 34rpx;
  font-weight: 600;
  color: #202124;
}

.heading-link {
  display: flex;
  align-items: center;
  font-size: 25rpx;
  color: #9b9b9b;
}

.arrow-icon {
  width: 32rpx;
  height: 32rpx;
  margin-left: 6rpx;
  opacity: 0.35;
}

.service-slogan {
  font-size: 23rpx;
  color: #24a94b;
}

.order-grid,
.service-grid {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
}

.service-grid {
  row-gap: 36rpx;
}

.logout-link {
  width: auto;
  height: 52rpx;
  margin: 30rpx auto 0;
  padding: 0 24rpx;
  background: transparent;
  color: #aaaaaa;
  font-size: 22rpx;
  line-height: 52rpx;
}

.logout-link::after {
  border: 0;
}

.shortcut-item {
  position: relative;
  display: flex;
  align-items: center;
  flex-direction: column;
  justify-content: flex-start;
  min-width: 0;
}

.shortcut-item:active {
  opacity: 0.65;
}

.order-icon {
  width: 58rpx;
  height: 58rpx;
  margin-bottom: 14rpx;
  opacity: 0.54;
}

.service-icon {
  width: 58rpx;
  height: 58rpx;
  margin-bottom: 14rpx;
  filter: invert(26%) sepia(97%) saturate(3560%) hue-rotate(352deg) brightness(98%) contrast(89%);
}

.shortcut-label {
  max-width: 126rpx;
  overflow: hidden;
  font-size: 25rpx;
  line-height: 34rpx;
  color: #555555;
  text-align: center;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.shortcut-note {
  position: absolute;
  top: -18rpx;
  right: 10rpx;
  min-width: 36rpx;
  padding: 2rpx 8rpx;
  border: 2rpx solid #ef3027;
  border-radius: 18rpx;
  background: #ffffff;
  color: #ef3027;
  font-size: 18rpx;
  line-height: 28rpx;
  text-align: center;
}

.recommend-section {
  margin-top: 18rpx;
}

.recommend-heading {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 92rpx;
  background: #f5f5f5;
}

.heading-line {
  width: 62rpx;
  height: 2rpx;
  background: #d8d8d8;
}

.recommend-title {
  margin: 0 18rpx;
  font-size: 29rpx;
  font-weight: 600;
  color: #555555;
}

.product-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8rpx;
  padding: 0 8rpx;
}

.state-box {
  display: flex;
  align-items: center;
  flex-direction: column;
  justify-content: center;
  min-height: 240rpx;
  padding: 30rpx;
  box-sizing: border-box;
  background: #ffffff;
  color: #999999;
  font-size: 25rpx;
}

.error-state {
  color: #777777;
}

.retry-button {
  width: 180rpx;
  height: 60rpx;
  margin-top: 22rpx;
  border-radius: 30rpx;
  background: #ef3027;
  color: #ffffff;
  font-size: 24rpx;
  line-height: 60rpx;
}

.load-more-state {
  padding: 28rpx 0 12rpx;
  color: #aaaaaa;
  font-size: 22rpx;
  text-align: center;
}

:deep(.product-card) {
  border-radius: 0;
  box-shadow: none;
}

:deep(.card-grid .img-wrapper) {
  height: 360rpx;
}

@media screen and (min-width: 750px) {
  .profile-page {
    max-width: 750px;
    margin: 0 auto;
  }
}
</style>
