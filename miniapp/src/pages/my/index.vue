<template>
  <view class="profile-container">
    <!-- 用户信息头部卡片 -->
    <view class="user-card">
      <!-- 微信头像选择按钮 (open-type="chooseAvatar") -->
      <button class="avatar-btn" open-type="chooseAvatar" @chooseavatar="onChooseAvatar">
        <image v-if="avatarUrl" class="avatar-img" :src="avatarUrl" mode="aspectFill" />
        <view v-else class="avatar-placeholder">
          <text class="avatar-emoji">👤</text>
        </view>
        <view class="avatar-badge">
          <text class="badge-text">换头像</text>
        </view>
      </button>

      <view class="user-meta">
        <!-- 微信昵称输入框 (type="nickname" + 手动输入双兼容) -->
        <view class="nickname-row">
          <input
            type="nickname"
            class="nickname-input"
            v-model="nickName"
            placeholder="点击填入微信昵称"
            placeholder-class="nickname-placeholder"
            @blur="onNicknameBlur"
            @input="onNicknameInput"
          />
          <text class="edit-icon">✏️</text>
        </view>
        <text class="status-desc">
          {{ isLoggedIn ? `登录状态: 已登录 (时间: ${loginTime})` : "可点击头像更换，点击昵称快速填写" }}
        </text>
      </view>

      <view class="login-badge" :class="{ active: isLoggedIn }">
        <text>{{ isLoggedIn ? "已登录" : "未登录" }}</text>
      </view>
    </view>

    <!-- 快捷头像与昵称设置卡片 -->
    <view class="debug-card">
      <view class="debug-header">
        <text class="debug-title">头像与昵称快捷设置</text>
        <text class="debug-tag">防受限</text>
      </view>
      <view class="debug-actions">
        <!-- 微信原生选择 -->
        <button class="action-btn native-avatar-btn" open-type="chooseAvatar" @chooseavatar="onChooseAvatar">
          <text class="btn-text">微信头像选择 (chooseAvatar)</text>
        </button>
        <!-- 本地图片选择降级 -->
        <button class="action-btn local-avatar-btn" @tap="handleChooseLocalImage">
          <text class="btn-text">从本地相册选择头像</text>
        </button>
        <!-- 一键设置测试昵称 -->
        <button class="action-btn test-name-btn" @tap="handleSetDefaultNickname">
          <text class="btn-text">一键填入测试昵称</text>
        </button>
      </view>
    </view>

    <!-- 登录操作区域 -->
    <view class="action-card">
      <view class="section-title">
        <text class="title-text">小程序登录</text>
      </view>

      <view class="action-content">
        <!-- 微信一键登录按钮 -->
        <button
          class="wechat-login-btn"
          :class="{ 'btn-relogin': isLoggedIn }"
          @tap="handleWechatLogin"
        >
          <text class="wechat-icon">🟢</text>
          <text class="btn-text">
            {{ isLoggedIn ? "重新获取登录凭证 (刷新 Code)" : "微信一键登录" }}
          </text>
        </button>

        <button v-if="isLoggedIn" class="logout-btn" @tap="handleLogout">
          <text class="btn-text">退出登录</text>
        </button>
      </view>
    </view>

    <!-- 商城与公共服务信息卡片 (来自 Bootstrap API) -->
    <view v-if="bootstrapData" class="shop-service-card">
      <view class="card-header">
        <text class="card-title">商城与官方服务</text>
        <text class="service-tag">{{ bootstrapData.shop?.name || "官方自营" }}</text>
      </view>

      <view class="service-list">
        <!-- 小程序名称 -->
        <view v-if="bootstrapData.mp?.name" class="service-row">
          <text class="service-label">小程序名称</text>
          <text class="service-val">{{ bootstrapData.mp.name }}</text>
        </view>

        <!-- 原始 ID -->
        <view v-if="bootstrapData.mp?.original_id" class="service-row">
          <text class="service-label">原始 ID</text>
          <text class="service-val code-font">{{ bootstrapData.mp.original_id }}</text>
        </view>

        <!-- 客服电话 -->
        <view
          v-if="bootstrapData.shop?.service_phone"
          class="service-row tapable"
          @tap="handleCallPhone(bootstrapData.shop.service_phone)"
        >
          <text class="service-label">客服电话</text>
          <view class="val-with-action">
            <text class="service-val link-red">{{ bootstrapData.shop.service_phone }}</text>
            <text class="mini-btn">拨打</text>
          </view>
        </view>

        <!-- 客服微信 -->
        <view
          v-if="bootstrapData.shop?.service_wechat"
          class="service-row tapable"
          @tap="handleCopyWechat(bootstrapData.shop.service_wechat)"
        >
          <text class="service-label">客服微信</text>
          <view class="val-with-action">
            <text class="service-val link-red">{{ bootstrapData.shop.service_wechat }}</text>
            <text class="mini-btn">复制</text>
          </view>
        </view>
      </view>
    </view>

    <!-- 登录与用户信息打印展示面板 -->
    <view class="log-card">
      <view class="log-header">
        <text class="log-title">登录信息输出 (Console Log)</text>
        <text v-if="loginResult" class="log-tag">已捕获</text>
      </view>

      <view v-if="loginResult" class="log-body">
        <view class="log-tip">
          <text class="tip-dot">●</text>
          <text class="tip-text">完整数据已通过 console.log 打印至开发者工具控制台：</text>
        </view>
        <view class="code-box">
          <text class="code-content">{{ formattedLoginInfo }}</text>
        </view>
      </view>

      <view v-else class="empty-log">
        <text class="empty-log-text">暂无登录信息，点击上方【微信一键登录】后将在此处与控制台打印</text>
      </view>
    </view>

    <!-- 底部版权信息 (来自 Bootstrap Site) -->
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

const { bootstrapData, fetchBootstrap } = useBootstrap()

const isLoggedIn = ref<boolean>(false)
const loginResult = ref<Record<string, unknown> | null>(null)
const loginTime = ref<string>("")

// 用户头像与昵称
const avatarUrl = ref<string>("")
const nickName = ref<string>("")

const formattedLoginInfo = computed(() => {
  if (!loginResult.value) return ""
  try {
    return JSON.stringify(loginResult.value, null, 2)
  } catch {
    return String(loginResult.value)
  }
})

onMounted(() => {
  // 读取本地缓存
  try {
    const cachedAvatar = uni.getStorageSync("user_avatar")
    const cachedNick = uni.getStorageSync("user_nickname")
    if (cachedAvatar && typeof cachedAvatar === "string") avatarUrl.value = cachedAvatar
    if (cachedNick && typeof cachedNick === "string") nickName.value = cachedNick
  } catch (e) {
    console.warn("读取本地缓存失败:", e)
  }

  // 获取启动配置
  if (!bootstrapData.value) {
    fetchBootstrap()
  }
})

function handleCallPhone(phoneNumber: string): void {
  uni.makePhoneCall({
    phoneNumber,
    fail: (err) => {
      console.warn("拨打电话取消或失败:", err)
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

/**
 * 微信官方 chooseAvatar 回调
 */
function onChooseAvatar(e: any): void {
  try {
    const selectedUrl = e?.detail?.avatarUrl
    if (selectedUrl) {
      avatarUrl.value = selectedUrl
      try {
        uni.setStorageSync("user_avatar", selectedUrl)
      } catch {}

      console.log("【微信获取头像成功 (chooseAvatar)】avatarUrl:", selectedUrl)
      printFullInfo("微信原生头像更新")
      uni.showToast({ title: "头像已更新", icon: "success" })
    }
  } catch (err) {
    console.warn("onChooseAvatar 处理异常:", err)
  }
}

/**
 * 降级方案：从本地相册选择头像
 */
function handleChooseLocalImage(): void {
  uni.chooseImage({
    count: 1,
    sizeType: ["compressed"],
    sourceType: ["album", "camera"],
    success: (res) => {
      if (res.tempFilePaths && res.tempFilePaths.length > 0) {
        const filePath = res.tempFilePaths[0]
        avatarUrl.value = filePath
        try {
          uni.setStorageSync("user_avatar", filePath)
        } catch {}

        console.log("【本地选图成功】avatarUrl:", filePath)
        printFullInfo("本地头像更新")
        uni.showToast({ title: "头像已选定", icon: "success" })
      }
    },
    fail: (err) => {
      console.log("取消选择或选择失败:", err)
    },
  })
}

/**
 * 快捷设置测试昵称
 */
function handleSetDefaultNickname(): void {
  const names = ["AI 商城用户", "微信小体验官", "拼购达人", "阳光快乐少年"]
  const randomName = names[Math.floor(Math.random() * names.length)]
  nickName.value = randomName
  try {
    uni.setStorageSync("user_nickname", randomName)
  } catch {}

  console.log("【测试昵称填入成功】nickName:", randomName)
  printFullInfo("测试昵称填入")
  uni.showToast({ title: "昵称已填入", icon: "none" })
}

/**
 * 微信官方 type="nickname" 失焦回调
 */
function onNicknameBlur(e: any): void {
  try {
    const value = e?.detail?.value
    if (value) {
      nickName.value = value
      try {
        uni.setStorageSync("user_nickname", value)
      } catch {}

      console.log("【微信获取昵称成功】nickName:", value)
      printFullInfo("昵称填入")
    }
  } catch (err) {
    console.warn("onNicknameBlur 异常:", err)
  }
}

/**
 * 昵称输入事件
 */
function onNicknameInput(e: any): void {
  try {
    const value = e?.detail?.value
    if (value !== undefined) {
      nickName.value = value
      try {
        uni.setStorageSync("user_nickname", value)
      } catch {}
    }
  } catch (err) {
    console.warn("onNicknameInput 异常:", err)
  }
}

/**
 * 打印并汇总当前全部登录与用户信息
 */
function printFullInfo(triggerReason = "常规操作"): void {
  const codeVal =
    (loginResult.value?.code as string) ||
    (loginResult.value?.loginRes as any)?.code ||
    ""

  const fullData: Record<string, unknown> = {
    trigger: triggerReason,
    time: new Date().toLocaleTimeString(),
    code: codeVal || "请点击下方【微信一键登录】获取 Code",
    userInfo: {
      nickName: nickName.value || "未设置 (可直接在上方输入或点击下方快速填入)",
      avatarUrl: avatarUrl.value || "未设置 (可点击头像或下方相册选择按钮)",
    },
    rawLoginRes: loginResult.value?.loginRes || loginResult.value || null,
  }

  loginResult.value = fullData

  console.log("====================================================")
  console.log(`【小程序登录与用户信息汇总 - ${triggerReason}】`)
  console.log("登录凭证 Code (用于后端换取 OpenID):", codeVal || "暂未获取")
  console.log("用户昵称 (nickName):", nickName.value || "未设置")
  console.log("用户头像 (avatarUrl):", avatarUrl.value || "未设置")
  console.log("完整数据对象:", fullData)
  console.log("====================================================")
}

/**
 * 微信小程序登录
 */
function handleWechatLogin(): void {
  uni.showLoading({ title: "正在登录...", mask: true })

  // #ifdef MP-WEIXIN
  try {
    uni.login({
      provider: "weixin",
      success: (res) => {
        isLoggedIn.value = true
        loginTime.value = new Date().toLocaleTimeString()

        loginResult.value = {
          code: res.code,
          errMsg: res.errMsg,
          loginRes: res,
        }

        printFullInfo("微信一键登录成功")

        uni.showToast({
          title: "登录成功",
          icon: "success",
        })
      },
      fail: (err) => {
        console.error("【微信小程序登录失败】:", err)
        uni.showToast({
          title: "登录失败",
          icon: "none",
        })
      },
      complete: () => {
        uni.hideLoading()
      },
    })
  } catch (err) {
    uni.hideLoading()
    console.error("uni.login 异常:", err)
  }
  // #endif

  // #ifndef MP-WEIXIN
  setTimeout(() => {
    const mockRes = {
      errMsg: "login:ok",
      code: "mock_code_" + Math.random().toString(36).substring(2, 10),
    }

    if (!nickName.value) nickName.value = "测试用户"

    isLoggedIn.value = true
    loginTime.value = new Date().toLocaleTimeString()

    loginResult.value = {
      code: mockRes.code,
      errMsg: mockRes.errMsg,
      loginRes: mockRes,
    }

    printFullInfo("模拟登录成功")

    uni.hideLoading()
    uni.showToast({
      title: "模拟登录成功",
      icon: "success",
    })
  }, 400)
  // #endif
}

/**
 * 退出登录
 */
function handleLogout(): void {
  isLoggedIn.value = false
  loginResult.value = null
  loginTime.value = ""
  console.log("【小程序状态】用户已退出登录")
  uni.showToast({
    title: "已退出登录",
    icon: "none",
  })
}
</script>

<style scoped>
.profile-container {
  min-height: 100vh;
  background-color: #f7f8fa;
  padding: 32rpx 28rpx 48rpx;
  box-sizing: border-box;
}

/* 用户卡片 */
.user-card {
  display: flex;
  align-items: center;
  background: linear-gradient(135deg, #ffffff 0%, #fef5f5 100%);
  border-radius: 20rpx;
  padding: 36rpx 28rpx;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.05);
  margin-bottom: 24rpx;
}

.avatar-btn {
  position: relative;
  width: 120rpx;
  height: 120rpx;
  border-radius: 60rpx;
  padding: 0;
  margin: 0 24rpx 0 0;
  background: none;
  border: none;
  overflow: visible;
  line-height: normal;
}

.avatar-btn::after {
  border: none;
}

.avatar-img {
  width: 120rpx;
  height: 120rpx;
  border-radius: 60rpx;
  border: 4rpx solid #ffffff;
  box-shadow: 0 4rpx 10rpx rgba(224, 46, 36, 0.15);
}

.avatar-placeholder {
  width: 120rpx;
  height: 120rpx;
  border-radius: 60rpx;
  background-color: #fdeeed;
  display: flex;
  align-items: center;
  justify-content: center;
}

.avatar-emoji {
  font-size: 56rpx;
}

.avatar-badge {
  position: absolute;
  bottom: -6rpx;
  left: 50%;
  transform: translateX(-50%);
  background-color: #e02e24;
  color: #ffffff;
  font-size: 18rpx;
  padding: 2rpx 12rpx;
  border-radius: 12rpx;
  white-space: nowrap;
}

.user-meta {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.nickname-row {
  display: flex;
  align-items: center;
  gap: 8rpx;
  margin-bottom: 8rpx;
}

.nickname-input {
  font-size: 32rpx;
  font-weight: 600;
  color: #303133;
  width: 280rpx;
}

.nickname-placeholder {
  font-size: 28rpx;
  color: #909399;
  font-weight: normal;
}

.edit-icon {
  font-size: 24rpx;
}

.status-desc {
  font-size: 22rpx;
  color: #909399;
  line-height: 1.4;
}

.login-badge {
  font-size: 22rpx;
  padding: 6rpx 16rpx;
  border-radius: 20rpx;
  background-color: #eef0f4;
  color: #909399;
}

.login-badge.active {
  background-color: #fff1f0;
  color: #e02e24;
  font-weight: 500;
}

/* 调试卡片 */
.debug-card {
  background-color: #ffffff;
  border-radius: 20rpx;
  padding: 24rpx 28rpx;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.04);
  margin-bottom: 24rpx;
}

.debug-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 18rpx;
}

.debug-title {
  font-size: 28rpx;
  font-weight: 600;
  color: #303133;
}

.debug-tag {
  font-size: 20rpx;
  color: #e02e24;
  background-color: #fff1f0;
  padding: 4rpx 12rpx;
  border-radius: 6rpx;
}

.debug-actions {
  display: flex;
  flex-direction: column;
  gap: 14rpx;
}

.action-btn {
  font-size: 26rpx;
  height: 72rpx;
  line-height: 72rpx;
  border-radius: 36rpx;
  padding: 0 24rpx;
  text-align: center;
}

.action-btn::after {
  border: none;
}

.native-avatar-btn {
  background-color: #f7f8fa;
  color: #606266;
  border: 1rpx solid #e4e7ed;
}

.local-avatar-btn {
  background-color: #fff1f0;
  color: #e02e24;
  border: 1rpx solid #fcdad7;
  font-weight: 500;
}

.test-name-btn {
  background-color: #f0f7ff;
  color: #409eff;
  border: 1rpx solid #d9ecff;
}

/* 操作区域 */
.action-card {
  background-color: #ffffff;
  border-radius: 20rpx;
  padding: 32rpx 28rpx;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.04);
  margin-bottom: 24rpx;
}

.section-title {
  margin-bottom: 24rpx;
}

.title-text {
  font-size: 30rpx;
  font-weight: 600;
  color: #303133;
  padding-left: 12rpx;
  border-left: 6rpx solid #e02e24;
}

.action-content {
  display: flex;
  flex-direction: column;
  gap: 20rpx;
}

.wechat-login-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12rpx;
  background-color: #e02e24;
  color: #ffffff;
  font-size: 30rpx;
  font-weight: 500;
  height: 88rpx;
  line-height: 88rpx;
  border-radius: 44rpx;
  box-shadow: 0 6rpx 16rpx rgba(224, 46, 36, 0.28);
}

.wechat-login-btn.btn-relogin {
  background-color: #f75444;
  box-shadow: 0 6rpx 16rpx rgba(247, 84, 68, 0.28);
}

.wechat-login-btn::after {
  border: none;
}

.wechat-icon {
  font-size: 28rpx;
}

.btn-text {
  font-size: 28rpx;
}

.logout-btn {
  background-color: #f5f6f8;
  color: #909399;
  font-size: 28rpx;
  height: 80rpx;
  line-height: 80rpx;
  border-radius: 40rpx;
}

.logout-btn::after {
  border: none;
}

/* 商城与官方服务信息卡片 */
.shop-service-card {
  background-color: #ffffff;
  border-radius: 20rpx;
  padding: 28rpx;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.04);
  margin-bottom: 24rpx;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 20rpx;
}

.card-title {
  font-size: 28rpx;
  font-weight: 600;
  color: #303133;
}

.service-tag {
  font-size: 22rpx;
  color: #e02e24;
  background-color: #fff1f0;
  padding: 4rpx 12rpx;
  border-radius: 8rpx;
}

.service-list {
  display: flex;
  flex-direction: column;
  gap: 16rpx;
}

.service-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10rpx 0;
  border-bottom: 1rpx solid #f2f3f5;
}

.service-row:last-child {
  border-bottom: none;
}

.service-label {
  font-size: 26rpx;
  color: #606266;
}

.service-val {
  font-size: 26rpx;
  color: #303133;
}

.code-font {
  font-family: monospace;
  color: #909399;
}

.link-red {
  color: #e02e24;
  font-weight: 500;
}

.val-with-action {
  display: flex;
  align-items: center;
  gap: 12rpx;
}

.mini-btn {
  font-size: 20rpx;
  color: #ffffff;
  background-color: #e02e24;
  padding: 4rpx 14rpx;
  border-radius: 16rpx;
}

/* 控制台输出信息卡片 */
.log-card {
  background-color: #ffffff;
  border-radius: 20rpx;
  padding: 32rpx 28rpx;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.04);
  margin-bottom: 24rpx;
}

.log-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 20rpx;
}

.log-title {
  font-size: 28rpx;
  font-weight: 600;
  color: #303133;
}

.log-tag {
  font-size: 22rpx;
  color: #67c23a;
  background-color: #f0f9eb;
  padding: 4rpx 14rpx;
  border-radius: 8rpx;
}

.log-body {
  display: flex;
  flex-direction: column;
  gap: 16rpx;
}

.log-tip {
  display: flex;
  align-items: center;
  gap: 10rpx;
}

.tip-dot {
  color: #67c23a;
  font-size: 20rpx;
}

.tip-text {
  font-size: 24rpx;
  color: #606266;
}

.code-box {
  background-color: #1e1e1e;
  border-radius: 12rpx;
  padding: 24rpx;
  overflow-x: auto;
}

.code-content {
  font-family: monospace;
  font-size: 24rpx;
  color: #00e676;
  white-space: pre-wrap;
  word-break: break-all;
}

.empty-log {
  padding: 40rpx 20rpx;
  text-align: center;
}

.empty-log-text {
  font-size: 24rpx;
  color: #c0c4cc;
}

/* 底部版权信息 */
.site-footer {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-top: 24rpx;
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
