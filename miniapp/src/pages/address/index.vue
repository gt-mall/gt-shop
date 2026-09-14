<template>
  <view class="address-page">
    <!-- 顶部提示条 (当从结算页跳转选择地址时展示) -->
    <view v-if="isSelectMode" class="select-tip-bar">
      <text class="tip-icon">ℹ️</text>
      <text class="tip-text">点击选择要使用的收货地址</text>
    </view>

    <!-- 地址列表 -->
    <view v-if="addresses.length > 0" class="address-list">
      <view
        v-for="addr in addresses"
        :key="addr.id"
        class="address-card"
        :class="{ 'is-selected': isSelectMode && selectedId === addr.id }"
        @tap="handleCardClick(addr)"
      >
        <!-- 主体信息 -->
        <view class="addr-header">
          <view class="user-info">
            <text class="user-name">{{ addr.name }}</text>
            <text class="user-phone">{{ addr.mobile }}</text>
          </view>
          <text v-if="addr.is_default === 1" class="default-badge">默认</text>
        </view>

        <view class="addr-detail">
          <text class="detail-text">
            {{ addr.province }} {{ addr.city }} {{ addr.district }} {{ addr.address }}
          </text>
        </view>

        <!-- 卡片底部操作栏 (参考拼多多样式) -->
        <view class="addr-footer" @tap.stop>
          <!-- 设为默认单选框 -->
          <view class="default-radio" @tap="handleSetDefault(addr)">
            <view class="radio-circle" :class="{ checked: addr.is_default === 1 }">
              <text v-if="addr.is_default === 1" class="radio-inner">✓</text>
            </view>
            <text class="radio-label" :class="{ 'is-active': addr.is_default === 1 }">
              {{ addr.is_default === 1 ? "已设为默认" : "设为默认" }}
            </text>
          </view>

          <!-- 右侧操作按钮组 -->
          <view class="action-buttons">
            <view class="action-btn edit" @tap="handleEditAddress(addr.id)">
              <text class="btn-icon">✎</text>
              <text class="btn-text">编辑</text>
            </view>
            <view class="action-btn delete" @tap="handleDeleteAddress(addr.id)">
              <text class="btn-icon">🗑</text>
              <text class="btn-text">删除</text>
            </view>
          </view>
        </view>
      </view>
    </view>

    <!-- 空状态 -->
    <!-- 未登录状态提示 -->
    <view v-if="!token" class="empty-state">
      <view class="empty-icon-wrap">
        <text class="empty-icon">🔒</text>
      </view>
      <text class="empty-title">您尚未登录</text>
      <text class="empty-subtitle">登录后可查看与管理您的专属收货地址</text>
      <button class="empty-add-btn" @tap="handleGoLogin">
        立即去登录
      </button>
    </view>

    <!-- 空状态 (已登录但无地址) -->
    <view v-else-if="!pageLoading" class="empty-state">
      <view class="empty-icon-wrap">
        <text class="empty-icon">📍</text>
      </view>
      <text class="empty-title">暂无收货地址</text>
      <text class="empty-subtitle">添加收货地址，下单更轻松便捷</text>
      <button class="empty-add-btn" @tap="handleCreateAddress">
        + 新建收货地址
      </button>
      <button class="empty-quick-btn" @tap="handleQuickAddTestAddress">
        ⚡ 快捷添加默认测试地址
      </button>
      <!-- #ifdef MP-WEIXIN -->
      <button class="empty-wx-btn" @tap="handleImportWechatAddress">
        <text class="wx-icon">🟢</text>
        <text>一键导入微信收货地址</text>
      </button>
      <!-- #endif -->
    </view>

    <!-- 底部悬浮固定操作区 (拼多多经典双按钮/主按钮吸底) -->
    <view class="bottom-action-bar">
      <!-- #ifdef MP-WEIXIN -->
      <button class="action-bar-btn import-wx-btn" @tap="handleImportWechatAddress">
        <text class="wx-logo">🟢</text>
        <text class="btn-label">导入微信地址</text>
      </button>
      <!-- #endif -->
      <button class="action-bar-btn create-btn" @tap="handleCreateAddress">
        <text class="plus-sign">+</text>
        <text class="btn-label">新建收货地址</text>
      </button>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue"
import { onShow, onLoad } from "@dcloudio/uni-app"
import { storeToRefs } from "pinia"
import { useMemberStore } from "@/stores/member"
import { resolveRegionCodes } from "@/utils"
import type { AddressResponse } from "@/api"

const memberStore = useMemberStore()
const { token, addresses } = storeToRefs(memberStore)
const {
  fetchAddresses,
  removeAddress,
  makeDefaultAddress,
  addAddress,
} = memberStore

const pageLoading = ref<boolean>(false)
const isSelectMode = ref<boolean>(false)
const selectedId = ref<number | null>(null)

onLoad((query: any) => {
  if (query?.select === "1" || query?.mode === "select") {
    isSelectMode.value = true
  }
  if (query?.id) {
    selectedId.value = Number(query.id)
  }
})

onShow(async () => {
  if (token.value) {
    pageLoading.value = true
    try {
      await fetchAddresses()
    } finally {
      pageLoading.value = false
    }
  }
})

onMounted(async () => {
  if (token.value && addresses.value.length === 0) {
    pageLoading.value = true
    try {
      await fetchAddresses()
    } finally {
      pageLoading.value = false
    }
  }
})

/**
 * 点击地址卡片 (若为选择模式则回调并返回)
 */
function handleCardClick(addr: AddressResponse): void {
  if (!isSelectMode.value) return

  // 触发全局选择事件或事件通道
  uni.$emit("selectAddress", addr)

  try {
    const pages = getCurrentPages()
    const prevPage = pages[pages.length - 2]
    if (prevPage && (prevPage as any).$vm?.onAddressSelected) {
      ;(prevPage as any).$vm.onAddressSelected(addr)
    }
  } catch {}

  uni.navigateBack()
}

/**
 * 跳转登录页
 */
function handleGoLogin(): void {
  uni.switchTab({
    url: "/pages/my/index",
  })
}

/**
 * 跳转新增地址
 */
function handleCreateAddress(): void {
  if (!token.value) {
    uni.showToast({ title: "请先登录后新增收货地址", icon: "none" })
    setTimeout(() => {
      uni.switchTab({ url: "/pages/my/index" })
    }, 800)
    return
  }
  uni.navigateTo({
    url: `/pages/address/edit?fromSelect=${isSelectMode.value ? "1" : "0"}`,
  })
}

/**
 * 快捷添加默认测试收货地址 (特别方便快速下单测试)
 */
async function handleQuickAddTestAddress(): Promise<void> {
  if (!token.value) {
    uni.showToast({ title: "请先登录", icon: "none" })
    return
  }

  uni.showLoading({ title: "正在添加测试地址...", mask: true })
  try {
    const res = await addAddress({
      name: "张三",
      mobile: "13800138000",
      province: "北京市",
      province_code: "110000",
      city: "北京市",
      city_code: "110100",
      district: "东城区",
      district_code: "110101",
      address: "长安街 1 号东方广场 A 座 801",
      postal_code: "100000",
      is_default: 1,
    })
    uni.showToast({ title: "测试地址添加成功", icon: "success" })
    const list = await fetchAddresses()
    if (isSelectMode.value) {
      const added = res && res.data ? res.data : list[0]
      if (added) {
        handleCardClick(added)
      }
    }
  } catch (err: any) {
    uni.showToast({ title: err?.message || "添加测试地址失败", icon: "none" })
  } finally {
    uni.hideLoading()
  }
}

/**
 * 跳转编辑地址
 */
function handleEditAddress(id: number): void {
  if (!token.value) {
    uni.showToast({ title: "请先登录", icon: "none" })
    setTimeout(() => {
      uni.switchTab({ url: "/pages/my/index" })
    }, 800)
    return
  }
  uni.navigateTo({
    url: `/pages/address/edit?id=${id}`,
  })
}

/**
 * 设为默认地址
 */
async function handleSetDefault(addr: AddressResponse): Promise<void> {
  if (!token.value) {
    uni.showToast({ title: "请先登录", icon: "none" })
    return
  }
  if (addr.is_default === 1) return
  uni.showLoading({ title: "设置中...", mask: true })
  try {
    await makeDefaultAddress(addr.id)
    uni.showToast({ title: "已设为默认地址", icon: "success" })
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : "设置失败"
    uni.showToast({ title: msg, icon: "none" })
  } finally {
    uni.hideLoading()
  }
}

/**
 * 删除地址
 */
function handleDeleteAddress(id: number): void {
  if (!token.value) {
    uni.showToast({ title: "请先登录", icon: "none" })
    return
  }
  uni.showModal({
    title: "确认删除",
    content: "确定要删除该收货地址吗？删除后不可恢复。",
    confirmText: "删除",
    confirmColor: "#e02e24",
    cancelText: "取消",
    success: async (res) => {
      if (res.confirm) {
        uni.showLoading({ title: "删除中...", mask: true })
        try {
          await removeAddress(id)
          uni.showToast({ title: "已删除", icon: "success" })
        } catch (err: unknown) {
          const msg = err instanceof Error ? err.message : "删除失败"
          uni.showToast({ title: msg, icon: "none" })
        } finally {
          uni.hideLoading()
        }
      }
    },
  })
}

/**
 * 从微信导入收货地址 (MP-WEIXIN 专用，参考拼多多)
 */
function handleImportWechatAddress(): void {
  if (!token.value) {
    uni.showToast({ title: "请先登录后导入收货地址", icon: "none" })
    setTimeout(() => {
      uni.switchTab({ url: "/pages/my/index" })
    }, 800)
    return
  }
  // #ifdef MP-WEIXIN
  uni.chooseAddress({
    success: async (res) => {
      console.log("【获取微信收货地址成功】:", res)
      if (!res.userName || !res.telNumber) {
        uni.showToast({ title: "微信地址信息不完整", icon: "none" })
        return
      }

      uni.showLoading({ title: "正在导入...", mask: true })
      try {
        const codes = resolveRegionCodes(res.provinceName, res.cityName, res.countyName)
        await addAddress({
          name: res.userName.trim(),
          mobile: res.telNumber.trim(),
          province_code: codes.province_code,
          province: res.provinceName.trim(),
          city_code: codes.city_code,
          city: res.cityName.trim(),
          district_code: codes.district_code,
          district: res.countyName.trim(),
          address: res.detailInfo.trim(),
          postal_code: res.postalCode || null,
          is_default: addresses.value.length === 0 ? 1 : 0,
        })
        uni.showToast({ title: "导入地址成功", icon: "success" })
      } catch (err: unknown) {
        const msg = err instanceof Error ? err.message : "导入失败"
        uni.showToast({ title: msg, icon: "none" })
      } finally {
        uni.hideLoading()
      }
    },
    fail: (err) => {
      if (err?.errMsg && !err.errMsg.includes("cancel")) {
        console.warn("读取微信收货地址失败:", err)
        uni.showToast({ title: "未能读取微信地址", icon: "none" })
      }
    },
  })
  // #endif

  // #ifndef MP-WEIXIN
  uni.showToast({ title: "当前环境不支持微信地址导入", icon: "none" })
  // #endif
}
</script>

<style scoped>
.address-page {
  min-height: 100vh;
  background-color: #f7f8fa;
  padding: 20rpx 24rpx;
  /* 留足底部悬浮按钮的高度与安全区距离 */
  padding-bottom: calc(140rpx + env(safe-area-inset-bottom));
  box-sizing: border-box;
}

/* 顶部选择提示 */
.select-tip-bar {
  display: flex;
  align-items: center;
  gap: 12rpx;
  background-color: #fff8e6;
  border-radius: 12rpx;
  padding: 16rpx 24rpx;
  margin-bottom: 20rpx;
}

.tip-icon {
  font-size: 28rpx;
}

.tip-text {
  font-size: 26rpx;
  color: #d48806;
}

/* 地址列表 */
.address-list {
  display: flex;
  flex-direction: column;
  gap: 20rpx;
}

/* 地址卡片 */
.address-card {
  background-color: #ffffff;
  border-radius: 18rpx;
  padding: 28rpx;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.03);
  transition: all 0.2s ease;
  border: 2rpx solid transparent;
}

.address-card.is-selected {
  border-color: #e02e24;
  background-color: #fffdfd;
}

/* 头部收件人信息 */
.addr-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 14rpx;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 16rpx;
}

.user-name {
  font-size: 32rpx;
  font-weight: 600;
  color: #222222;
  max-width: 260rpx;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.user-phone {
  font-size: 28rpx;
  color: #555555;
  font-weight: 500;
}

.default-badge {
  font-size: 20rpx;
  color: #ffffff;
  background-color: #e02e24;
  padding: 2rpx 12rpx;
  border-radius: 6rpx;
  font-weight: 500;
  line-height: 1.4;
}

/* 详细地址 */
.addr-detail {
  padding-bottom: 20rpx;
  border-bottom: 1rpx solid #f2f3f5;
}

.detail-text {
  font-size: 26rpx;
  color: #666666;
  line-height: 1.5;
  word-break: break-all;
}

/* 卡片底部操作栏 (拼多多样式) */
.addr-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-top: 20rpx;
}

/* 设为默认单选框 */
.default-radio {
  display: flex;
  align-items: center;
  gap: 12rpx;
  padding: 6rpx 0;
}

.radio-circle {
  width: 32rpx;
  height: 32rpx;
  border-radius: 50%;
  border: 2rpx solid #c0c4cc;
  display: flex;
  align-items: center;
  justify-content: center;
  box-sizing: border-box;
  transition: all 0.2s;
}

.radio-circle.checked {
  border-color: #e02e24;
  background-color: #e02e24;
}

.radio-inner {
  color: #ffffff;
  font-size: 22rpx;
  line-height: 1;
  font-weight: bold;
}

.radio-label {
  font-size: 26rpx;
  color: #666666;
}

.radio-label.is-active {
  color: #e02e24;
  font-weight: 500;
}

/* 右侧操作按钮组 */
.action-buttons {
  display: flex;
  align-items: center;
  gap: 28rpx;
}

.action-btn {
  display: flex;
  align-items: center;
  gap: 6rpx;
  font-size: 26rpx;
  color: #888888;
  padding: 6rpx 4rpx;
}

.action-btn:active {
  opacity: 0.7;
}

.action-btn.edit .btn-icon {
  font-size: 26rpx;
}

.action-btn.delete {
  color: #999999;
}

.action-btn.delete .btn-icon {
  font-size: 24rpx;
}

/* 空状态 */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding-top: 140rpx;
  padding-bottom: 60rpx;
}

.empty-icon-wrap {
  width: 140rpx;
  height: 140rpx;
  border-radius: 70rpx;
  background-color: #fdeeed;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 30rpx;
}

.empty-icon {
  font-size: 64rpx;
}

.empty-title {
  font-size: 32rpx;
  font-weight: 600;
  color: #333333;
  margin-bottom: 12rpx;
}

.empty-subtitle {
  font-size: 26rpx;
  color: #999999;
  margin-bottom: 48rpx;
}

.empty-add-btn {
  width: 380rpx;
  height: 84rpx;
  line-height: 84rpx;
  background-color: #e02e24;
  color: #ffffff;
  font-size: 28rpx;
  font-weight: 500;
  border-radius: 42rpx;
  box-shadow: 0 6rpx 16rpx rgba(224, 46, 36, 0.25);
  margin-bottom: 24rpx;
}

.empty-add-btn::after {
  border: none;
}

.empty-quick-btn {
  width: 380rpx;
  height: 84rpx;
  line-height: 84rpx;
  background-color: #fff1f0;
  color: #e02e24;
  font-size: 28rpx;
  font-weight: 500;
  border-radius: 42rpx;
  border: 1rpx solid #ffccc7;
  margin-bottom: 24rpx;
}

.empty-quick-btn::after {
  border: none;
}

.empty-wx-btn {
  width: 380rpx;
  height: 84rpx;
  line-height: 84rpx;
  background-color: #ffffff;
  color: #07c160;
  border: 1rpx solid #07c160;
  font-size: 28rpx;
  font-weight: 500;
  border-radius: 42rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12rpx;
}

.empty-wx-btn::after {
  border: none;
}

/* 底部悬浮固定操作区 */
.bottom-action-bar {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  background-color: #ffffff;
  padding: 18rpx 28rpx;
  padding-bottom: calc(18rpx + env(safe-area-inset-bottom));
  box-shadow: 0 -4rpx 16rpx rgba(0, 0, 0, 0.05);
  display: flex;
  align-items: center;
  gap: 20rpx;
  z-index: 100;
}

.action-bar-btn {
  flex: 1;
  height: 84rpx;
  line-height: 84rpx;
  border-radius: 42rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10rpx;
  font-size: 30rpx;
  font-weight: 500;
  margin: 0;
}

.action-bar-btn::after {
  border: none;
}

.import-wx-btn {
  background-color: #f2faf4;
  color: #07c160;
  border: 1rpx solid #c7eddb;
}

.wx-logo {
  font-size: 28rpx;
}

.create-btn {
  background-color: #e02e24;
  color: #ffffff;
  box-shadow: 0 6rpx 16rpx rgba(224, 46, 36, 0.28);
}

.plus-sign {
  font-size: 36rpx;
  font-weight: bold;
  line-height: 1;
}
</style>
