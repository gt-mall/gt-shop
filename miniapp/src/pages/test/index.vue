<template>
  <view class="test-page">
    <view class="test-card">
      <text class="card-title">测试工具</text>
      <text class="description">用于验证会员资料与收货地址功能。以下写入操作会修改当前账号的真实数据，请使用测试账号。</text>
      <text class="session-status">{{ isLoggedIn ? "当前已登录" : "请先登录后使用资料和地址测试" }}</text>
      <button v-if="!isLoggedIn" class="action-button" @tap="goToLogin">去登录</button>
      <!-- #ifdef MP-WEIXIN -->
      <button :disabled="busy" :loading="busy && activeAction === '微信认证'" class="action-button" @tap="testWechatLogin">
        {{ isLoggedIn ? "重新获取微信登录凭证" : "微信登录测试" }}
      </button>
      <!-- #endif -->
    </view>

    <view class="test-card">
      <text class="card-title">会员资料</text>
      <text class="description">测试昵称和头像将保存到当前账号，个人中心会同步显示。</text>
      <button class="action-button" :disabled="busy || !isLoggedIn" @tap="testNickname">设置测试昵称</button>
      <!-- #ifdef MP-WEIXIN -->
      <button class="action-button" :disabled="busy || !isLoggedIn" open-type="chooseAvatar" @chooseavatar="onChooseAvatar">微信头像选择测试</button>
      <!-- #endif -->
      <button class="action-button" :disabled="busy || !isLoggedIn" @tap="chooseLocalAvatar">本地头像上传测试</button>
    </view>

    <view class="test-card">
      <text class="card-title">测试收货地址</text>
      <text class="description">选择地区后新建一条测试地址，不替换已有默认地址。测试完成后可在地址管理中删除。</text>
      <picker :range="presetCities" range-key="name" :value="selectedCity" :disabled="busy" @change="onCityChange">
        <view class="city-picker">{{ currentCity.name }} <text>切换地区 ›</text></view>
      </picker>
      <button class="action-button" :disabled="busy || !isLoggedIn" @tap="testAddress">添加测试地址</button>
      <button class="secondary-button" :disabled="busy || !isLoggedIn" @tap="goToAddresses">管理收货地址</button>
    </view>

    <view class="test-card">
      <text class="card-title">当前会员状态</text>
      <text class="description">仅展示必要字段，不展示登录凭证和联系方式。</text>
      <text class="diagnostic-data" selectable>{{ diagnosticInfo }}</text>
    </view>

    <view v-if="busy || resultMessage" class="test-card">
      <text class="card-title">{{ busy ? activeAction + "进行中…" : "操作结果" }}</text>
      <text v-if="resultMessage" class="result-message" :class="{ 'result-error': resultFailed }">{{ resultMessage }}</text>
    </view>
  </view>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { storeToRefs } from "pinia"
import { useMemberStore } from "@/stores/member"

const memberStore = useMemberStore()
const { token, member, addresses, isLoggedIn } = storeToRefs(memberStore)
const busy = ref(false)
const activeAction = ref("")
const resultMessage = ref("")
const resultFailed = ref(false)
const selectedCity = ref(0)

const presetCities = [
  { name: "北京东城", province: "北京市", province_code: "110000", city: "北京市", city_code: "110100", district: "东城区", district_code: "110101" },
  { name: "上海黄浦", province: "上海市", province_code: "310000", city: "上海市", city_code: "310100", district: "黄浦区", district_code: "310101" },
  { name: "广州天河", province: "广东省", province_code: "440000", city: "广州市", city_code: "440100", district: "天河区", district_code: "440106" },
  { name: "深圳南山", province: "广东省", province_code: "440000", city: "深圳市", city_code: "440300", district: "南山区", district_code: "440305" },
  { name: "杭州西湖", province: "浙江省", province_code: "330000", city: "杭州市", city_code: "330100", district: "西湖区", district_code: "330106" },
  { name: "成都锦江", province: "四川省", province_code: "510000", city: "成都市", city_code: "510100", district: "锦江区", district_code: "510104" },
]

const currentCity = computed(() => presetCities[selectedCity.value] || presetCities[0])
const diagnosticInfo = computed(() => JSON.stringify({
  logged_in: isLoggedIn.value,
  member_id: member.value?.id ?? null,
  nickname: member.value?.nickname ?? "",
  has_avatar: !!member.value?.avatar_url,
  loaded_address_count: addresses.value.length,
}, null, 2))

watch(token, () => {
  resultMessage.value = ""
  resultFailed.value = false
})

// 当前请求层会 reject，统一在此操作边界收集结果和恢复按钮状态。
async function runTest(
  label: string,
  operation: () => Promise<string>,
  requiresLogin = true,
  changesSession = false,
): Promise<void> {
  if (busy.value) return
  if (requiresLogin && !isLoggedIn.value) {
    uni.showToast({ title: "请先登录", icon: "none" })
    return
  }
  const requestToken = token.value
  busy.value = true
  activeAction.value = label
  resultMessage.value = ""
  resultFailed.value = false
  const [result] = await Promise.allSettled([operation()])
  busy.value = false
  if (!changesSession && requestToken !== token.value) return
  resultFailed.value = result.status === "rejected"
  resultMessage.value = result.status === "fulfilled"
    ? result.value
    : result.reason instanceof Error ? result.reason.message : label + "失败，请重试"
}

async function testNickname(): Promise<void> {
  await runTest("更新昵称", async () => {
    const names = ["AI 商城达人", "微信拼购客", "尊贵会员小张", "数码极客"]
    const nickname = names[Math.floor(Math.random() * names.length)]
    const updated = await memberStore.updateNickname(nickname)
    if (!updated || updated.nickname !== nickname) throw new Error("昵称未更新，请重新登录后重试")
    return "昵称已保存：" + updated.nickname
  })
}

async function uploadAvatar(filePath: string): Promise<void> {
  if (!filePath) return
  await runTest("上传头像", async () => {
    const updated = await memberStore.updateAvatar(filePath)
    if (!updated) throw new Error("头像未更新，请重新登录后重试")
    return "头像已上传，返回个人中心即可查看"
  })
}

function onChooseAvatar(event: { detail: { avatarUrl: string } }): void {
  void uploadAvatar(event.detail.avatarUrl)
}

function chooseLocalAvatar(): void {
  if (busy.value || !isLoggedIn.value) return
  uni.chooseImage({
    count: 1,
    sizeType: ["compressed"],
    sourceType: ["album", "camera"],
    success: (res) => {
      if (res.tempFilePaths[0]) void uploadAvatar(res.tempFilePaths[0])
    },
  })
}

function onCityChange(event: { detail: { value: string | number } }): void {
  selectedCity.value = Number(event.detail.value)
}

async function testAddress(): Promise<void> {
  await runTest("添加地址", async () => {
    const { name: cityLabel, ...region } = currentCity.value
    const response = await memberStore.addAddress({
      ...region,
      name: "测试收件人",
      mobile: "13800138000",
      address: "测试地址，请勿用于实际配送",
      is_default: 0,
    })
    if (response.code !== 1 || !response.data) throw new Error(response.message || "添加地址失败")
    return cityLabel + "测试地址已创建，可前往地址管理查看或删除"
  })
}

// #ifdef MP-WEIXIN
async function testWechatLogin(): Promise<void> {
  await runTest("微信认证", async () => {
    const result = await new Promise<UniNamespace.LoginRes>((resolve, reject) => {
      uni.login({ provider: "weixin", success: resolve, fail: reject })
    })
    if (!result.code) throw new Error("未获取到微信登录凭证")
    await memberStore.loginWithCode(result.code)
    return "微信登录成功，会员状态已同步"
  }, false, true)
}
// #endif

function goToLogin(): void {
  // #ifdef MP-WEIXIN
  uni.switchTab({ url: "/pages/my/index" })
  // #endif
  // #ifndef MP-WEIXIN
  uni.navigateTo({ url: "/pages/auth/login" })
  // #endif
}

function goToAddresses(): void {
  uni.navigateTo({ url: "/pages/address/index" })
}
</script>

<style scoped>
.test-page {
  min-height: 100vh;
  padding: 24rpx 24rpx 48rpx;
  box-sizing: border-box;
  background: #f7f8fa;
}
.test-card {
  margin-bottom: 24rpx;
  padding: 28rpx;
  border-radius: 20rpx;
  background: #fff;
}
.card-title, .description, .session-status, .result-message, .diagnostic-data {
  display: block;
}
.card-title {
  margin-bottom: 16rpx;
  font-size: 30rpx;
  font-weight: 600;
  color: #303133;
}
.description, .session-status, .result-message {
  font-size: 25rpx;
  line-height: 1.7;
  color: #606266;
}
.session-status {
  margin-top: 16rpx;
}
.action-button, .secondary-button {
  margin-top: 20rpx;
  font-size: 27rpx;
  border-radius: 12rpx;
}
.action-button {
  background: #fff1f0;
  color: #e02e24;
}
.secondary-button {
  background: #f7f8fa;
  color: #606266;
}
.city-picker {
  display: flex;
  justify-content: space-between;
  padding: 24rpx 0;
  font-size: 27rpx;
  color: #303133;
}
.city-picker text {
  color: #909399;
}
.diagnostic-data {
  margin-top: 20rpx;
  padding: 20rpx;
  background: #f7f8fa;
  white-space: pre-wrap;
  word-break: break-all;
  font-family: monospace;
  font-size: 24rpx;
  color: #606266;
}
.result-error {
  color: #e02e24;
}
</style>
