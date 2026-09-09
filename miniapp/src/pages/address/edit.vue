<template>
  <view class="address-edit-page">
    <!-- 拼多多特色：智能粘贴识别区域 (可折叠/展开) -->
    <view class="smart-paste-card">
      <view class="paste-header" @tap="showPasteBox = !showPasteBox">
        <view class="header-left">
          <text class="sparkle-icon">✨</text>
          <text class="paste-title">智能粘贴地址</text>
          <text class="paste-tag">一键识别</text>
        </view>
        <text class="toggle-arrow">{{ showPasteBox ? "▲ 收起" : "▼ 展开" }}</text>
      </view>

      <view v-if="showPasteBox" class="paste-body">
        <textarea
          class="paste-textarea"
          v-model="pasteText"
          placeholder="粘贴整段地址文本，自动识别姓名、电话、省市区与详细地址（如：张三，13800000000，北京市海淀区中关村南大街1号）"
          maxlength="200"
        />
        <view class="paste-actions">
          <button class="clear-btn" @tap="pasteText = ''">清空</button>
          <button class="parse-btn" :disabled="!pasteText.trim()" @tap="handleSmartParse">
            智能识别填入
          </button>
        </view>
      </view>
    </view>

    <!-- 主表单卡片 -->
    <view class="form-card">
      <!-- 收货人 -->
      <view class="form-row">
        <text class="field-label">收货人</text>
        <view class="field-control">
          <input
            class="field-input"
            v-model="form.name"
            placeholder="请填写收件人姓名"
            maxlength="25"
            placeholder-class="input-placeholder"
          />
          <text v-if="form.name" class="clear-icon" @tap="form.name = ''">✕</text>
        </view>
      </view>

      <!-- 手机号码 -->
      <view class="form-row">
        <text class="field-label">手机号码</text>
        <view class="field-control">
          <input
            class="field-input"
            type="number"
            v-model="form.mobile"
            placeholder="请填写11位手机号码"
            maxlength="11"
            placeholder-class="input-placeholder"
          />
          <text v-if="form.mobile" class="clear-icon" @tap="form.mobile = ''">✕</text>
        </view>
      </view>

      <!-- 所在地区 (原生 picker mode="region") -->
      <view class="form-row">
        <text class="field-label">所在地区</text>
        <picker
          mode="region"
          class="region-picker"
          :value="regionPickerValue"
          @change="handleRegionChange"
        >
          <view class="picker-value-wrap">
            <text v-if="hasSelectedRegion" class="region-text">
              {{ form.province }} {{ form.city }} {{ form.district }}
            </text>
            <text v-else class="region-placeholder">
              点击选择省、市、区/县
            </text>
            <text class="arrow-right">›</text>
          </view>
        </picker>
      </view>

      <!-- 详细地址 -->
      <view class="form-row textarea-row">
        <text class="field-label">详细地址</text>
        <view class="field-control textarea-wrap">
          <textarea
            class="field-textarea"
            v-model="form.address"
            placeholder="街道门牌、小区楼栋号、单元房间等详细信息"
            maxlength="120"
            placeholder-class="input-placeholder"
          />
          <text class="word-count">{{ form.address.length }}/120</text>
        </view>
      </view>

      <!-- 设为默认地址开关 -->
      <view class="form-row switch-row">
        <view class="switch-info">
          <text class="switch-title">设为默认收货地址</text>
          <text class="switch-desc">每次下单时将优先使用该地址</text>
        </view>
        <switch
          :checked="form.is_default === 1"
          color="#e02e24"
          @change="(e: any) => form.is_default = e.detail.value ? 1 : 0"
        />
      </view>
    </view>

    <!-- 编辑模式下的删除操作 -->
    <view v-if="isEdit" class="delete-section">
      <button class="delete-btn" @tap="handleDelete">
        <text class="delete-icon">🗑</text>
        <text class="delete-text">删除该收货地址</text>
      </button>
    </view>

    <!-- 底部固定保存按钮 (拼多多样式吸底) -->
    <view class="bottom-bar">
      <button class="submit-btn" :loading="submitting" @tap="handleSave">
        保存并使用
      </button>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref, reactive, computed } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { useMember } from "@/services/member"
import {
  parseAddressText,
  resolveRegionCodes,
} from "@/utils"
import type { AddressResponse } from "@/api"

const { addresses, fetchAddresses, addAddress, editAddress, removeAddress } = useMember()
const { addresses, fetchAddresses, addAddress, editAddress, removeAddress, token } = useMember()

const addressId = ref<number | null>(null)
const isEdit = computed(() => !!addressId.value)
const submitting = ref<boolean>(false)

// 智能识别文本
const showPasteBox = ref<boolean>(false)
const pasteText = ref<string>("")

// 表单响应式数据
const form = reactive({
  name: "",
  mobile: "",
  province: "",
  province_code: "",
  city: "",
  city_code: "",
  district: "",
  district_code: "",
  address: "",
  postal_code: "",
  is_default: 0,
})

// 当前 picker 显示值
const regionPickerValue = computed<string[]>(() => {
  if (form.province && form.city && form.district) {
    return [form.province, form.city, form.district]
  }
  return ["北京市", "北京市", "海淀区"]
})

const hasSelectedRegion = computed<boolean>(() => {
  return !!(form.province && form.city && form.district)
})

onLoad(async (query: any) => {
  if (!token.value) {
    uni.showToast({ title: "请先登录后操作收货地址", icon: "none" })
    setTimeout(() => {
      uni.switchTab({ url: "/pages/my/index" })
    }, 1000)
    return
  }

  if (query?.id) {
    addressId.value = Number(query.id)
    uni.setNavigationBarTitle({ title: "编辑收货地址" })
    await initEditData(addressId.value)
  } else {
    uni.setNavigationBarTitle({ title: "新增收货地址" })
    // 如果之前没有收货地址，默认开启“设为默认”
    if (addresses.value.length === 0) {
      form.is_default = 1
    }
  }
})

/**
 * 编辑模式初始化数据回显
 */
async function initEditData(id: number): Promise<void> {
  let target = addresses.value.find((item) => item.id === id)
  if (!target) {
    uni.showLoading({ title: "加载中..." })
    try {
      const list = await fetchAddresses()
      target = list.find((item) => item.id === id)
    } finally {
      uni.hideLoading()
    }
  }

  if (target) {
    populateForm(target)
  } else {
    uni.showToast({ title: "未找到该地址", icon: "none" })
    setTimeout(() => uni.navigateBack(), 1000)
  }
}

function populateForm(data: AddressResponse): void {
  form.name = data.name || ""
  form.mobile = data.mobile || ""
  form.province = data.province || ""
  form.province_code = data.province_code || ""
  form.city = data.city || ""
  form.city_code = data.city_code || ""
  form.district = data.district || ""
  form.district_code = data.district_code || ""
  form.address = data.address || ""
  form.postal_code = data.postal_code || ""
  form.is_default = data.is_default || 0
}

/**
 * 智能识别文本并填入
 */
function handleSmartParse(): void {
  const parsed = parseAddressText(pasteText.value)
  let filledCount = 0

  if (parsed.name) {
    form.name = parsed.name
    filledCount++
  }
  if (parsed.mobile) {
    form.mobile = parsed.mobile
    filledCount++
  }
  if (parsed.province) {
    form.province = parsed.province
    form.province_code = parsed.province_code
    filledCount++
  }
  if (parsed.city) {
    form.city = parsed.city
    form.city_code = parsed.city_code
  }
  if (parsed.district) {
    form.district = parsed.district
    form.district_code = parsed.district_code
  }
  if (parsed.address) {
    form.address = parsed.address
    filledCount++
  }

  if (filledCount > 0) {
    uni.showToast({ title: "已智能填入表单", icon: "success" })
    showPasteBox.value = false
  } else {
    uni.showToast({ title: "未能识别有效信息，请手动输入", icon: "none" })
  }
}

/**
 * 原生地区选择器回调
 */
function handleRegionChange(e: any): void {
  const vals: string[] = e.detail?.value || []
  const codes: string[] = e.detail?.code || []

  if (vals.length >= 3) {
    form.province = vals[0]
    form.city = vals[1]
    form.district = vals[2]

    const resolved = resolveRegionCodes(vals[0], vals[1], vals[2], {
      province_code: codes[0],
      city_code: codes[1],
      district_code: codes[2],
    })

    form.province_code = resolved.province_code
    form.city_code = resolved.city_code
    form.district_code = resolved.district_code
  }
}

/**
 * 表单校验
 */
function validateForm(): boolean {
  if (!form.name.trim()) {
    uni.showToast({ title: "请填写收货人姓名", icon: "none" })
    return false
  }
  if (form.name.trim().length < 2) {
    uni.showToast({ title: "姓名至少2个字符", icon: "none" })
    return false
  }

  const phone = form.mobile.trim()
  if (!phone) {
    uni.showToast({ title: "请填写手机号码", icon: "none" })
    return false
  }
  if (!/^1[3-9]\d{9}$/.test(phone)) {
    uni.showToast({ title: "请填写有效11位手机号", icon: "none" })
    return false
  }

  if (!form.province || !form.city || !form.district) {
    uni.showToast({ title: "请选择所在省市区", icon: "none" })
    return false
  }

  if (!form.address.trim()) {
    uni.showToast({ title: "请填写详细地址", icon: "none" })
    return false
  }
  if (form.address.trim().length < 4) {
    uni.showToast({ title: "详细地址过短，请补全门牌号", icon: "none" })
    return false
  }

  return true
}

/**
 * 保存收货地址
 */
async function handleSave(): Promise<void> {
  if (!token.value) {
    uni.showToast({ title: "请先登录", icon: "none" })
    setTimeout(() => {
      uni.switchTab({ url: "/pages/my/index" })
    }, 800)
    return
  }

  if (!validateForm()) return

  // 补全区划代码
  const resolved = resolveRegionCodes(form.province, form.city, form.district, {
    province_code: form.province_code,
    city_code: form.city_code,
    district_code: form.district_code,
  })

  submitting.value = true
  try {
    if (isEdit.value && addressId.value) {
      await editAddress({
        id: addressId.value,
        name: form.name.trim(),
        mobile: form.mobile.trim(),
        province: form.province.trim(),
        province_code: resolved.province_code,
        city: form.city.trim(),
        city_code: resolved.city_code,
        district: form.district.trim(),
        district_code: resolved.district_code,
        address: form.address.trim(),
        postal_code: form.postal_code ? form.postal_code.trim() : null,
        is_default: form.is_default,
      })
      uni.showToast({ title: "修改成功", icon: "success" })
    } else {
      await addAddress({
        name: form.name.trim(),
        mobile: form.mobile.trim(),
        province: form.province.trim(),
        province_code: resolved.province_code,
        city: form.city.trim(),
        city_code: resolved.city_code,
        district: form.district.trim(),
        district_code: resolved.district_code,
        address: form.address.trim(),
        postal_code: form.postal_code ? form.postal_code.trim() : null,
        is_default: form.is_default,
      })
      uni.showToast({ title: "添加成功", icon: "success" })
    }

    setTimeout(() => {
      uni.navigateBack()
    }, 600)
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : "保存失败"
    uni.showToast({ title: msg, icon: "none" })
  } finally {
    submitting.value = false
  }
}

/**
 * 删除收货地址
 */
function handleDelete(): void {
  if (!addressId.value) return

  uni.showModal({
    title: "确认删除",
    content: "确定要删除该收货地址吗？",
    confirmText: "删除",
    confirmColor: "#e02e24",
    cancelText: "取消",
    success: async (res) => {
      if (res.confirm) {
        uni.showLoading({ title: "删除中...", mask: true })
        try {
          await removeAddress(addressId.value!)
          uni.showToast({ title: "已删除", icon: "success" })
          setTimeout(() => {
            uni.navigateBack()
          }, 600)
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
</script>

<style scoped>
.address-edit-page {
  min-height: 100vh;
  background-color: #f7f8fa;
  padding: 20rpx 24rpx;
  padding-bottom: calc(140rpx + env(safe-area-inset-bottom));
  box-sizing: border-box;
}

/* 智能粘贴卡片 */
.smart-paste-card {
  background-color: #ffffff;
  border-radius: 18rpx;
  padding: 24rpx 28rpx;
  margin-bottom: 20rpx;
  box-shadow: 0 4rpx 14rpx rgba(0, 0, 0, 0.03);
}

.paste-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 10rpx;
}

.sparkle-icon {
  font-size: 28rpx;
}

.paste-title {
  font-size: 28rpx;
  font-weight: 600;
  color: #333333;
}

.paste-tag {
  font-size: 20rpx;
  color: #e02e24;
  background-color: #fff1f0;
  padding: 2rpx 10rpx;
  border-radius: 6rpx;
}

.toggle-arrow {
  font-size: 24rpx;
  color: #999999;
}

.paste-body {
  margin-top: 18rpx;
}

.paste-textarea {
  width: 100%;
  height: 140rpx;
  background-color: #f9f9fb;
  border-radius: 12rpx;
  padding: 16rpx 20rpx;
  box-sizing: border-box;
  font-size: 26rpx;
  color: #333333;
  line-height: 1.4;
}

.paste-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 16rpx;
  margin-top: 16rpx;
}

.clear-btn {
  font-size: 24rpx;
  color: #888888;
  background-color: #f2f3f5;
  border-radius: 24rpx;
  padding: 0 24rpx;
  height: 52rpx;
  line-height: 52rpx;
  margin: 0;
}

.clear-btn::after {
  border: none;
}

.parse-btn {
  font-size: 24rpx;
  color: #ffffff;
  background-color: #e02e24;
  border-radius: 24rpx;
  padding: 0 28rpx;
  height: 52rpx;
  line-height: 52rpx;
  margin: 0;
}

.parse-btn::after {
  border: none;
}

.parse-btn[disabled] {
  opacity: 0.5;
}

/* 主表单卡片 */
.form-card {
  background-color: #ffffff;
  border-radius: 18rpx;
  padding: 8rpx 28rpx;
  box-shadow: 0 4rpx 14rpx rgba(0, 0, 0, 0.03);
}

.form-row {
  display: flex;
  align-items: center;
  padding: 28rpx 0;
  border-bottom: 1rpx solid #f2f3f5;
}

.form-row:last-child {
  border-bottom: none;
}

.field-label {
  width: 160rpx;
  font-size: 28rpx;
  color: #333333;
  font-weight: 500;
}

.field-control {
  flex: 1;
  display: flex;
  align-items: center;
  position: relative;
}

.field-input {
  flex: 1;
  font-size: 28rpx;
  color: #333333;
}

.input-placeholder {
  color: #c0c4cc;
  font-size: 28rpx;
}

.clear-icon {
  font-size: 24rpx;
  color: #c0c4cc;
  padding: 10rpx 14rpx;
}

/* 地区选择器 */
.region-picker {
  flex: 1;
}

.picker-value-wrap {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.region-text {
  font-size: 28rpx;
  color: #333333;
  flex: 1;
}

.region-placeholder {
  font-size: 28rpx;
  color: #c0c4cc;
}

.arrow-right {
  font-size: 36rpx;
  color: #c0c4cc;
  margin-left: 8rpx;
  line-height: 1;
}

/* 详细地址多行文本 */
.textarea-row {
  align-items: flex-start;
  padding: 28rpx 0 20rpx;
}

.textarea-wrap {
  display: flex;
  flex-direction: column;
}

.field-textarea {
  width: 100%;
  height: 140rpx;
  font-size: 28rpx;
  color: #333333;
  line-height: 1.4;
}

.word-count {
  align-self: flex-end;
  font-size: 22rpx;
  color: #c0c4cc;
  margin-top: 6rpx;
}

/* 设为默认开关 */
.switch-row {
  justify-content: space-between;
  padding: 24rpx 0;
}

.switch-info {
  display: flex;
  flex-direction: column;
  gap: 6rpx;
}

.switch-title {
  font-size: 28rpx;
  color: #333333;
  font-weight: 500;
}

.switch-desc {
  font-size: 22rpx;
  color: #999999;
}

/* 删除区域 */
.delete-section {
  margin-top: 32rpx;
  display: flex;
  justify-content: center;
}

.delete-btn {
  background-color: transparent;
  display: flex;
  align-items: center;
  gap: 10rpx;
  padding: 16rpx 40rpx;
  margin: 0;
}

.delete-btn::after {
  border: none;
}

.delete-icon {
  font-size: 28rpx;
  color: #e02e24;
}

.delete-text {
  font-size: 26rpx;
  color: #e02e24;
}

/* 底部固定保存栏 */
.bottom-bar {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  background-color: #ffffff;
  padding: 18rpx 28rpx;
  padding-bottom: calc(18rpx + env(safe-area-inset-bottom));
  box-shadow: 0 -4rpx 16rpx rgba(0, 0, 0, 0.05);
  z-index: 100;
}

.submit-btn {
  height: 84rpx;
  line-height: 84rpx;
  background-color: #e02e24;
  color: #ffffff;
  font-size: 30rpx;
  font-weight: 500;
  border-radius: 42rpx;
  box-shadow: 0 6rpx 16rpx rgba(224, 46, 36, 0.28);
}

.submit-btn::after {
  border: none;
}
</style>
