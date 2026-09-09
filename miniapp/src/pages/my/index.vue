<template>
  <view class="profile-container">
    <!-- 用户/会员头部卡片 -->
    <view class="user-card">
      <!-- 微信头像选择按钮 (open-type="chooseAvatar") -->
      <button class="avatar-btn" open-type="chooseAvatar" @chooseavatar="onChooseAvatar">
        <image
          v-if="currentAvatarUrl && !avatarFailed"
          class="avatar-img"
          :src="currentAvatarUrl"
          mode="aspectFill"
          @error="avatarFailed = true"
        />
        <view v-else class="avatar-placeholder">
          <text class="avatar-emoji">👤</text>
        </view>
        <view class="avatar-badge">
          <text class="badge-text">{{ isLoggedIn ? "改头像" : "选头像" }}</text>
        </view>
      </button>

      <view class="user-meta">
        <!-- 微信昵称输入框 -->
        <view class="nickname-row">
          <input
            type="nickname"
            class="nickname-input"
            v-model="displayNickname"
            placeholder="点击填入微信昵称"
            placeholder-class="nickname-placeholder"
            @blur="onNicknameBlur"
            @input="onNicknameInput"
          />
          <text class="edit-icon">✏️</text>
        </view>

        <!-- 会员身份标识 -->
        <view class="member-sub-row">
          <text v-if="member" class="member-id-tag">会员号: #{{ member.id }}</text>
          <text class="status-desc">
            {{ isLoggedIn ? "会员身份已认证" : "登录后开启会员专属权益" }}
          </text>
        </view>
      </view>

      <view class="login-badge" :class="{ active: isLoggedIn }">
        <text>{{ isLoggedIn ? "已登录" : "未登录" }}</text>
      </view>
    </view>

    <!-- 快捷头像与昵称设置卡片 -->
    <view class="tool-card">
      <view class="card-header">
        <text class="card-title">资料快捷设置与上传</text>
        <text class="tool-tag">实时同步后端</text>
      </view>
      <view class="tool-actions">
        <!-- 微信原生选择头像 -->
        <button class="action-btn native-btn" open-type="chooseAvatar" @chooseavatar="onChooseAvatar">
          <text class="btn-text">微信头像选择 (chooseAvatar)</text>
        </button>
        <!-- 本地相册选图并上传 -->
        <button class="action-btn local-btn" @tap="handleChooseLocalImage">
          <text class="btn-text">从本地相册选择头像 (上传后端)</text>
        </button>
        <!-- 一键设置测试昵称 -->
        <button class="action-btn test-btn" @tap="handleSetDefaultNickname">
          <text class="btn-text">填入测试昵称 (同步后端)</text>
        </button>
      </view>
    </view>

    <!-- 登录与账号操作区域 -->
    <view class="action-card">
      <view class="section-title">
        <text class="title-text">会员认证</text>
      </view>

      <view class="action-content">
        <!-- 微信一键登录按钮 -->
        <button
          class="wechat-login-btn"
          :class="{ 'btn-relogin': isLoggedIn }"
          :loading="authLoading"
          @tap="handleWechatLogin"
        >
          <text class="wechat-icon">🟢</text>
          <text class="btn-text">
            {{ isLoggedIn ? "重新获取凭证 (刷新会员 Token)" : "微信一键登录 (换取会员 Token)" }}
          </text>
        </button>

        <button v-if="isLoggedIn" class="logout-btn" @tap="handleLogout">
          <text class="btn-text">退出登录</text>
        </button>
      </view>
    </view>

    <!-- 我的服务与常用工具 (拼多多风格金刚区与快捷横向条目) -->
    <view class="service-nav-card">
      <view class="card-header">
        <text class="card-title">我的服务与工具</text>
        <text class="service-tip">官方保障 · 极速售后</text>
      </view>

      <!-- 四宫格金刚区 -->
      <view class="service-grid">
        <!-- 收货地址入口 -->
        <view class="grid-item" @tap="handleNavigateToAddress">
          <view class="item-icon-wrap red-gradient">
            <text class="item-icon">📍</text>
            <text v-if="isLoggedIn && addresses.length > 0" class="badge-count">
              {{ addresses.length }}
            </text>
          </view>
          <text class="item-name">收货地址</text>
          <text class="item-sub">
            {{ isLoggedIn ? (addresses.length > 0 ? `${addresses.length}个地址` : "添加地址") : "去管理" }}
          </text>
        </view>

        <!-- 官方客服电话入口 -->
        <view
          v-if="bootstrapData?.shop?.service_phone"
          class="grid-item"
          @tap="handleCallPhone(bootstrapData.shop.service_phone)"
        >
          <view class="item-icon-wrap blue-gradient">
            <text class="item-icon">📞</text>
          </view>
          <text class="item-name">官方客服</text>
          <text class="item-sub">电话咨询</text>
        </view>

        <!-- 官方客服微信入口 -->
        <view
          v-if="bootstrapData?.shop?.service_wechat"
          class="grid-item"
          @tap="handleCopyWechat(bootstrapData.shop.service_wechat)"
        >
          <view class="item-icon-wrap green-gradient">
            <text class="item-icon">💬</text>
          </view>
          <text class="item-name">微信客服</text>
          <text class="item-sub">复制微信号</text>
        </view>

        <!-- 会员权益入口 -->
        <view class="grid-item" @tap="handleEquityNotice">
          <view class="item-icon-wrap gold-gradient">
            <text class="item-icon">👑</text>
          </view>
          <text class="item-name">会员权益</text>
          <text class="item-sub">{{ isLoggedIn ? "尊享权益" : "登录解锁" }}</text>
        </view>
      </view>

      <!-- 快捷横向条目：收货地址 -->
      <view class="service-cell" @tap="handleNavigateToAddress">
        <view class="cell-left">
          <view class="cell-icon-wrap">
            <text class="cell-icon">📍</text>
          </view>
          <view class="cell-texts">
            <text class="cell-title">我的收货地址</text>
            <text class="cell-sub">
              {{ isLoggedIn ? (addresses.length > 0 ? `已保存 ${addresses.length} 个地址` : "暂未添加收货地址") : "登录后管理您的收货地址" }}
            </text>
          </view>
        </view>
        <view class="cell-right">
          <text class="cell-action-text">{{ isLoggedIn ? "去管理" : "去登录" }}</text>
          <text class="cell-arrow">›</text>
        </view>
      </view>
    </view>

    <!-- 商城与官方服务信息卡片 (来自 Bootstrap API) -->
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
        <text class="log-title">会员数据与控制台输出 (Console Log)</text>
        <text v-if="member" class="log-tag">会员已联通</text>
      </view>

      <view v-if="logSummary" class="log-body">
        <view class="log-tip">
          <text class="tip-dot">●</text>
          <text class="tip-text">数据已输出至控制台并与 Go 后端会员服务同步：</text>
        </view>
        <view class="code-box">
          <text class="code-content">{{ formattedLogInfo }}</text>
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
import { onShow } from "@dcloudio/uni-app"
import { useBootstrap } from "@/services/bootstrap"
import { useMember } from "@/services/member"
import { resolveImageUrl } from "@/utils"

const { bootstrapData, fetchBootstrap } = useBootstrap()
const {
  token,
  member,
  addresses,
  isLoggedIn,
  loading: authLoading,
  loginWithCode,
  refreshMember,
  updateNickname,
  updateAvatar,
  fetchAddresses,
  logout,
} = useMember()

const avatarFailed = ref<boolean>(false)
const inputNickname = ref<string>("")
const loginTime = ref<string>("")
const lastLoginResult = ref<any>(null)

// 当前展示的头像
const currentAvatarUrl = computed(() => {
  if (member.value?.avatar_url) {
    return resolveImageUrl(member.value.avatar_url)
  }
  const cached = uni.getStorageSync("user_avatar")
  return cached ? resolveImageUrl(cached) : ""
})

// 当前展示的昵称
const displayNickname = computed({
  get() {
    if (inputNickname.value) return inputNickname.value
    if (member.value?.nickname) return member.value.nickname
    const cached = uni.getStorageSync("user_nickname")
    return cached || ""
  },
  set(val: string) {
    inputNickname.value = val
  },
})

// 日志面板展示对象
const logSummary = computed(() => {
  if (lastLoginResult.value) return lastLoginResult.value
  if (member.value) {
    return {
      token: token.value ? `${token.value.substring(0, 20)}...` : "",
      member: member.value,
      addressesCount: addresses.value.length,
    }
  }
  return null
})

const formattedLogInfo = computed(() => {
  if (!logSummary.value) return ""
  try {
    return JSON.stringify(logSummary.value, null, 2)
  } catch {
    return String(logSummary.value)
  }
})

onMounted(async () => {
  if (!bootstrapData.value) {
    fetchBootstrap()
  }
  if (token.value) {
    await refreshMember()
    await fetchAddresses()
  }
})

onShow(async () => {
  // 页面唤醒时静默刷新地址列表以同步数量徽标
  if (token.value) {
    fetchAddresses().catch(() => {})
  }
})

/**
 * 微信一键登录 (换取 Code 并调用后端 /auth/wechat-login)
 */
function handleWechatLogin(): void {
  uni.showLoading({ title: "正在登录...", mask: true })

  // #ifdef MP-WEIXIN
  try {
    uni.login({
      provider: "weixin",
      success: async (res) => {
        console.log("【1. 微信原生登录获取 Code 成功】:", res.code)
        try {
          // 调用后端微服务进行用户创建或登录换取 JWT Token
          const loginData = await loginWithCode(res.code)
          loginTime.value = new Date().toLocaleTimeString()
          lastLoginResult.value = {
            code: res.code,
            token: loginData.token,
            is_new_member: loginData.is_new_member,
            member: loginData.member,
          }

          console.log("====================================================")
          console.log("【2. 后端会员认证成功】Token:", loginData.token)
          console.log("【会员信息】:", loginData.member)
          console.log("====================================================")

          // 如果之前有本地选择的昵称且会员昵称为空，则同步更新资料
          if (inputNickname.value && !loginData.member.nickname) {
            await updateNickname(inputNickname.value)
          }

          uni.showToast({ title: "登录成功", icon: "success" })
        } catch (err: unknown) {
          const msg = err instanceof Error ? err.message : "登录失败"
          console.error("【后端会员登录异常】:", msg)
          uni.showToast({ title: msg, icon: "none" })
        }
      },
      fail: (err) => {
        console.error("【微信登录失败】:", err)
        uni.showToast({ title: "微信登录取消或失败", icon: "none" })
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
  setTimeout(async () => {
    const mockCode = "mock_code_" + Math.random().toString(36).substring(2, 10)
    console.log("【非微信环境 - 模拟获取 Code】:", mockCode)
    try {
      const loginData = await loginWithCode(mockCode)
      loginTime.value = new Date().toLocaleTimeString()
      lastLoginResult.value = loginData
      uni.hideLoading()
      uni.showToast({ title: "模拟登录成功", icon: "success" })
    } catch {
      uni.hideLoading()
      uni.showToast({ title: "模拟登录失败", icon: "none" })
    }
  }, 500)
  // #endif
}

/**
 * 微信官方 chooseAvatar 回调
 */
async function onChooseAvatar(e: any): Promise<void> {
  const selectedUrl = e?.detail?.avatarUrl
  if (selectedUrl) {
    avatarFailed.value = false
    try {
      uni.setStorageSync("user_avatar", selectedUrl)
    } catch {}

    // 如果已登录，自动上传到后端会员接口
    if (isLoggedIn.value) {
      uni.showLoading({ title: "上传头像中..." })
      try {
        const res = await updateAvatar(selectedUrl)
        if (res) {
          uni.showToast({ title: "头像已同步至后端", icon: "success" })
          console.log("【会员头像更新成功】:", res)
        }
      } catch (err) {
        console.warn("头像上传后端失败:", err)
        uni.showToast({ title: "本地头像已选定", icon: "none" })
      } finally {
        uni.hideLoading()
      }
    } else {
      uni.showToast({ title: "头像已选定，登录后将自动关联", icon: "none" })
    }
  }
}

/**
 * 从本地相册选择头像并上传
 */
function handleChooseLocalImage(): void {
  uni.chooseImage({
    count: 1,
    sizeType: ["compressed"],
    sourceType: ["album", "camera"],
    success: async (res) => {
      if (res.tempFilePaths && res.tempFilePaths.length > 0) {
        const filePath = res.tempFilePaths[0]
        avatarFailed.value = false
        try {
          uni.setStorageSync("user_avatar", filePath)
        } catch {}

        if (isLoggedIn.value) {
          uni.showLoading({ title: "上传头像中..." })
          try {
            await updateAvatar(filePath)
            uni.showToast({ title: "头像上传成功", icon: "success" })
          } catch (err) {
            console.warn("头像上传失败:", err)
            uni.showToast({ title: "本地头像已设置", icon: "none" })
          } finally {
            uni.hideLoading()
          }
        } else {
          uni.showToast({ title: "本地头像已设置", icon: "none" })
        }
      }
    },
  })
}

/**
 * 昵称失焦后触发保存
 */
async function onNicknameBlur(e: any): Promise<void> {
  const value = e?.detail?.value || inputNickname.value
  if (value) {
    inputNickname.value = value
    try {
      uni.setStorageSync("user_nickname", value)
    } catch {}

    if (isLoggedIn.value && member.value && member.value.nickname !== value) {
      try {
        await updateNickname(value)
        console.log("【会员昵称已同步至后端】:", value)
      } catch (err) {
        console.warn("同步昵称失败:", err)
      }
    }
  }
}

function onNicknameInput(e: any): void {
  const value = e?.detail?.value
  if (value !== undefined) {
    inputNickname.value = value
  }
}

/**
 * 设置默认测试昵称
 */
async function handleSetDefaultNickname(): Promise<void> {
  const names = ["AI 商城达人", "微信拼购客", "尊贵会员小张", "数码极客"]
  const randomName = names[Math.floor(Math.random() * names.length)]
  inputNickname.value = randomName
  try {
    uni.setStorageSync("user_nickname", randomName)
  } catch {}

  if (isLoggedIn.value) {
    try {
      await updateNickname(randomName)
      uni.showToast({ title: "昵称已同步至后端", icon: "success" })
    } catch {
      uni.showToast({ title: "已填入", icon: "none" })
    }
  } else {
    uni.showToast({ title: "已填入", icon: "none" })
  }
}

/**
 * 点击收货地址入口：跳转至独立收货地址管理页面
 */
function handleNavigateToAddress(): void {
  if (!isLoggedIn.value) {
    uni.showToast({
      title: "请先登录后管理收货地址",
      icon: "none",
    })
    return
  }
  uni.navigateTo({
    url: "/pages/address/index",
  })
}

/**
 * 会员权益提示
 */
function handleEquityNotice(): void {
  if (isLoggedIn.value) {
    uni.showToast({
      title: "您已享有专属会员折扣与包邮权益",
      icon: "none",
    })
  } else {
    uni.showToast({
      title: "请先点击上方登录开通会员",
      icon: "none",
    })
  }
}

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
 * 退出登录
 */
function handleLogout(): void {
  logout()
  loginTime.value = ""
  lastLoginResult.value = null
  console.log("【会员状态】用户已退出登录并清除 Token")
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
  padding: 32rpx 28rpx 60rpx;
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

.member-sub-row {
  display: flex;
  align-items: center;
  gap: 12rpx;
}

.member-id-tag {
  font-size: 20rpx;
  color: #e02e24;
  background-color: #fff1f0;
  padding: 2rpx 10rpx;
  border-radius: 6rpx;
  font-family: monospace;
}

.status-desc {
  font-size: 22rpx;
  color: #909399;
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

/* 快捷设置卡片 */
.tool-card {
  background-color: #ffffff;
  border-radius: 20rpx;
  padding: 24rpx 28rpx;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.04);
  margin-bottom: 24rpx;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 18rpx;
}

.card-title {
  font-size: 28rpx;
  font-weight: 600;
  color: #303133;
}

.tool-tag {
  font-size: 20rpx;
  color: #e02e24;
  background-color: #fff1f0;
  padding: 4rpx 12rpx;
  border-radius: 6rpx;
}

.tool-actions {
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

.native-btn {
  background-color: #f7f8fa;
  color: #606266;
  border: 1rpx solid #e4e7ed;
}

.local-btn {
  background-color: #fff1f0;
  color: #e02e24;
  border: 1rpx solid #fcdad7;
  font-weight: 500;
}

.test-btn {
  background-color: #f0f7ff;
  color: #409eff;
  border: 1rpx solid #d9ecff;
}

/* 登录认证区域 */
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

/* 我的服务与常用工具 (拼多多风格金刚区与快捷入口) */
.service-nav-card {
  background-color: #ffffff;
  border-radius: 20rpx;
  padding: 28rpx;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.04);
  margin-bottom: 24rpx;
}

.service-tip {
  font-size: 22rpx;
  color: #909399;
}

.service-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16rpx;
  padding: 16rpx 0 24rpx;
  border-bottom: 1rpx solid #f2f3f5;
}

.grid-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}

.grid-item:active {
  opacity: 0.75;
}

.item-icon-wrap {
  position: relative;
  width: 88rpx;
  height: 88rpx;
  border-radius: 44rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 12rpx;
  box-shadow: 0 4rpx 10rpx rgba(0, 0, 0, 0.06);
}

.red-gradient {
  background: linear-gradient(135deg, #ff6b6b 0%, #e02e24 100%);
}

.blue-gradient {
  background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
}

.green-gradient {
  background: linear-gradient(135deg, #43e97b 0%, #38f9d7 100%);
}

.gold-gradient {
  background: linear-gradient(135deg, #f6d365 0%, #fda085 100%);
}

.item-icon {
  font-size: 40rpx;
}

.badge-count {
  position: absolute;
  top: -6rpx;
  right: -6rpx;
  background-color: #ff3b30;
  color: #ffffff;
  font-size: 20rpx;
  font-weight: bold;
  min-width: 32rpx;
  height: 32rpx;
  line-height: 32rpx;
  text-align: center;
  border-radius: 16rpx;
  border: 2rpx solid #ffffff;
  padding: 0 6rpx;
  box-sizing: border-box;
}

.item-name {
  font-size: 26rpx;
  font-weight: 500;
  color: #303133;
  margin-bottom: 4rpx;
}

.item-sub {
  font-size: 20rpx;
  color: #909399;
}

/* 快捷横向条目 */
.service-cell {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-top: 24rpx;
}

.service-cell:active {
  opacity: 0.7;
}

.cell-left {
  display: flex;
  align-items: center;
  gap: 16rpx;
}

.cell-icon-wrap {
  width: 56rpx;
  height: 56rpx;
  border-radius: 28rpx;
  background-color: #fff1f0;
  border: 1rpx solid #fcdad7;
  display: flex;
  align-items: center;
  justify-content: center;
}

.cell-icon {
  font-size: 28rpx;
}

.cell-texts {
  display: flex;
  flex-direction: column;
  gap: 4rpx;
}

.cell-title {
  font-size: 28rpx;
  font-weight: 600;
  color: #303133;
}

.cell-sub {
  font-size: 22rpx;
  color: #909399;
}

.cell-right {
  display: flex;
  align-items: center;
  gap: 8rpx;
}

.cell-action-text {
  font-size: 24rpx;
  color: #e02e24;
  font-weight: 500;
}

.cell-arrow {
  font-size: 32rpx;
  color: #c0c4cc;
  line-height: 1;
}


/* 商城与官方服务信息卡片 */
.shop-service-card {
  background-color: #ffffff;
  border-radius: 20rpx;
  padding: 28rpx;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.04);
  margin-bottom: 24rpx;
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
