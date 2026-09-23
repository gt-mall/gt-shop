<template>
  <view class="auth-container">
    <!-- 头部品牌/标题 -->
    <view class="auth-header">
      <!-- #ifdef MP-WEIXIN -->
      <text class="auth-title">微信快捷登录</text>
      <text class="auth-subtitle">登录后即可购物、查看订单和会员服务</text>
      <!-- #endif -->
      <!-- #ifndef MP-WEIXIN -->
      <text class="auth-title">会员登录</text>
      <text class="auth-subtitle">欢迎回到商城，登录后享受更多权益</text>
      <!-- #endif -->
    </view>

    <!-- #ifdef MP-WEIXIN -->
    <view class="auth-card">
      <text class="wechat-hint">使用当前微信账号一键进入商城</text>
      <button class="submit-btn wechat-submit" :loading="wechatLoading" :disabled="wechatLoading" @tap="handleWechatLogin">
        微信一键登录 / 注册
      </button>
      <text class="wechat-note">首次使用将自动创建会员账号，无需填写手机号或密码</text>
    </view>
    <!-- #endif -->
    <!-- #ifndef MP-WEIXIN -->
    <!-- 登录表单卡片 -->
    <view class="auth-card">
      <!-- 手机号 -->
      <view class="input-group">
        <text class="input-label">手机号</text>
        <view class="input-wrap">
          <text class="input-icon">📱</text>
          <input
            class="input-field"
            type="number"
            maxlength="11"
            v-model="mobile"
            placeholder="请输入11位手机号"
            placeholder-class="placeholder-style"
          />
        </view>
      </view>

      <!-- 密码 -->
      <view class="input-group">
        <text class="input-label">密码</text>
        <view class="input-wrap">
          <text class="input-icon">🔒</text>
          <input
            class="input-field"
            :password="!showPassword"
            maxlength="64"
            v-model="password"
            placeholder="请输入登录密码"
            placeholder-class="placeholder-style"
          />
          <text class="eye-btn" @tap="showPassword = !showPassword">
            {{ showPassword ? "👁️" : "🙈" }}
          </text>
        </view>
      </view>

      <!-- 登录按钮 -->
      <button class="submit-btn" :loading="loading" @tap="handleLogin">
        登 录
      </button>

      <!-- 底部跳转 -->
      <view class="auth-links">
        <text class="link-text" @tap="goToRegister">没有账号？立即注册</text>
      </view>
    </view>
    <!-- #endif -->
  </view>
</template>

<script setup lang="ts">
import { ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { useMemberStore } from "@/stores/member"

const memberStore = useMemberStore()
const { loginWithPassword, loginWithWechat } = memberStore

const mobile = ref<string>("")
const password = ref<string>("")
const showPassword = ref<boolean>(false)
const loading = ref<boolean>(false)
const wechatLoading = ref<boolean>(false)
const redirectUrl = ref<string>("")

const mobileReg = /^1[3-9]\d{9}$/

onLoad((query: any) => {
  if (query?.redirect) {
    redirectUrl.value = decodeURIComponent(query.redirect)
  }
})

async function handleLogin() {
  const trimmedMobile = mobile.value.trim()
  if (!trimmedMobile) {
    uni.showToast({ title: "请输入手机号", icon: "none" })
    return
  }
  if (!mobileReg.test(trimmedMobile)) {
    uni.showToast({ title: "请输入有效的11位手机号", icon: "none" })
    return
  }
  if (!password.value) {
    uni.showToast({ title: "请输入密码", icon: "none" })
    return
  }

  loading.value = true
  try {
    await loginWithPassword({
      mobile: trimmedMobile,
      password: password.value,
    })
    uni.showToast({ title: "登录成功", icon: "success" })
    navigateAfterLogin()
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : "登录失败"
    uni.showToast({ title: msg, icon: "none" })
  } finally {
    loading.value = false
  }
}

function navigateAfterLogin() {
  setTimeout(() => {
    if (redirectUrl.value) {
      const tabbarPages = [
        "/pages/index/index",
        "/pages/category/index",
        "/pages/cart/index",
        "/pages/my/index",
      ]
      const isTab = tabbarPages.some((p) => redirectUrl.value.startsWith(p))
      if (isTab) {
        uni.switchTab({ url: redirectUrl.value })
      } else {
        uni.redirectTo({ url: redirectUrl.value })
      }
    } else {
      uni.switchTab({ url: "/pages/my/index" })
    }
  }, 500)
}

// #ifdef MP-WEIXIN
async function handleWechatLogin() {
  if (wechatLoading.value) return
  wechatLoading.value = true
  const [result] = await Promise.allSettled([loginWithWechat()])
  wechatLoading.value = false
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
  navigateAfterLogin()
}
// #endif

function goToRegister() {
  uni.navigateTo({ url: "/pages/auth/register" })
}
</script>

<style scoped>
.auth-container {
  min-height: 100vh;
  background-color: #f7f8fa;
  padding: 60rpx 40rpx;
  box-sizing: border-box;
}

.auth-header {
  margin-bottom: 48rpx;
  padding: 0 10rpx;
}

.auth-title {
  font-size: 48rpx;
  font-weight: bold;
  color: #303133;
  display: block;
  margin-bottom: 12rpx;
}

.auth-subtitle {
  font-size: 26rpx;
  color: #909399;
  display: block;
}

.auth-card {
  background-color: #ffffff;
  border-radius: 24rpx;
  padding: 40rpx 32rpx 48rpx;
  box-shadow: 0 4rpx 20rpx rgba(0, 0, 0, 0.05);
}

.wechat-hint,
.wechat-note {
  display: block;
  text-align: center;
  color: #606266;
  font-size: 28rpx;
}

.wechat-note {
  color: #909399;
  font-size: 24rpx;
  line-height: 1.6;
}

.wechat-submit {
  margin-top: 36rpx;
  margin-bottom: 24rpx;
}

.input-group {
  margin-bottom: 32rpx;
}

.input-label {
  font-size: 26rpx;
  font-weight: 500;
  color: #606266;
  margin-bottom: 12rpx;
  display: block;
}

.input-wrap {
  display: flex;
  align-items: center;
  background-color: #f8f9fa;
  border: 1rpx solid #ebeef5;
  border-radius: 16rpx;
  padding: 0 24rpx;
  height: 92rpx;
}

.input-icon {
  font-size: 32rpx;
  margin-right: 16rpx;
}

.input-field {
  flex: 1;
  font-size: 28rpx;
  color: #303133;
}

.placeholder-style {
  font-size: 28rpx;
  color: #c0c4cc;
}

.eye-btn {
  font-size: 32rpx;
  padding: 10rpx;
}

.submit-btn {
  background-color: #e02e24;
  color: #ffffff;
  font-size: 32rpx;
  font-weight: 500;
  height: 92rpx;
  line-height: 92rpx;
  border-radius: 46rpx;
  margin-top: 48rpx;
  margin-bottom: 32rpx;
  box-shadow: 0 6rpx 20rpx rgba(224, 46, 36, 0.28);
}

.submit-btn::after {
  border: none;
}

.auth-links {
  display: flex;
  justify-content: center;
  align-items: center;
  margin-top: 16rpx;
}

.link-text {
  font-size: 26rpx;
  color: #e02e24;
  font-weight: 500;
}
</style>
