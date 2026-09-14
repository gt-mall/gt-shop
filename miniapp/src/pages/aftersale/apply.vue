<template>
  <view class="aftersale-apply-page">
    <!-- 商品信息卡片 -->
    <view v-if="order" class="card goods-card">
      <view class="card-title">售后商品</view>
      <view v-for="item in order.items" :key="item.id" class="goods-item">
        <image class="goods-img" :src="item.image_url" mode="aspectFill" />
        <view class="goods-info">
          <text class="goods-name">{{ item.product_name }}</text>
          <text v-if="item.sku_spec_values" class="goods-spec">{{ item.sku_spec_values }}</text>
          <view class="goods-price-row">
            <text class="goods-price">¥{{ item.unit_price }}</text>
            <text class="goods-qty">x{{ item.quantity }}</text>
          </view>
        </view>
      </view>
      <view class="refund-hint-row">
        <text class="hint-label">最多可退金额：</text>
        <text class="refund-max-val">¥{{ order.pay_amount }}</text>
      </view>
    </view>

    <!-- 售后类型选择 -->
    <view class="card">
      <view class="card-title">售后类型</view>
      <view class="type-selector">
        <view
          v-if="canRefundOnly"
          class="type-btn"
          :class="{ active: form.type === 1 }"
          @tap="form.type = 1"
        >
          <text class="type-icon">💰</text>
          <view class="type-texts">
            <text class="type-title">仅退款</text>
            <text class="type-desc">未收到货，或与商家协商一致无需退货</text>
          </view>
        </view>

        <view
          v-if="canReturnRefund"
          class="type-btn"
          :class="{ active: form.type === 2 }"
          @tap="form.type = 2"
        >
          <text class="type-icon">📦</text>
          <view class="type-texts">
            <text class="type-title">退货退款</text>
            <text class="type-desc">已收到货，需将商品寄回后退款</text>
          </view>
        </view>
      </view>
    </view>

    <!-- 申请原因 -->
    <view class="card">
      <view class="card-title">申请原因 <text class="required">*</text></view>
      <picker :range="reasons" :value="reasonIndex" @change="onReasonChange">
        <view class="picker-selector">
          <text :class="reasonIndex >= 0 ? 'selected-val' : 'placeholder-val'">
            {{ reasonIndex >= 0 ? reasons[reasonIndex] : "请选择申请原因" }}
          </text>
          <text class="arrow">›</text>
        </view>
      </picker>
    </view>

    <!-- 问题描述 -->
    <view class="card">
      <view class="card-title">问题描述</view>
      <textarea
        class="desc-input"
        v-model="form.description"
        placeholder="请详细描述您遇到的商品或售后问题（255字以内）..."
        maxlength="255"
      />
      <view class="char-count">{{ form.description.length }}/255</view>
    </view>

    <!-- 凭证附件上传区 (图片 ≤ 9张，视频 ≤ 1个，总计 ≤ 10) -->
    <view class="card attachments-card">
      <view class="card-title-row">
        <view class="card-title">凭证附件</view>
        <view class="card-subtitle">
          图片 ≤ 9张(每张≤10MB)，视频 ≤ 1个(≤100MB)
        </view>
      </view>

      <view class="attachments-grid">
        <!-- 附件列表项 -->
        <view
          v-for="(item, idx) in attachments"
          :key="item.localId"
          class="attachment-item"
          @tap="handlePreview(item)"
        >
          <!-- 图片显示 -->
          <image
            v-if="item.fileType === 'image'"
            class="attachment-thumb"
            :src="item.localPath"
            mode="aspectFill"
          />

          <!-- 视频显示 -->
          <view v-else class="video-thumb-wrap">
            <image
              v-if="item.coverUrl"
              class="attachment-thumb"
              :src="item.coverUrl"
              mode="aspectFill"
            />
            <view v-else class="video-placeholder">
              <text class="video-icon">🎬</text>
            </view>
            <view class="play-badge">▶</view>
          </view>

          <!-- 视频角标 -->
          <text v-if="item.fileType === 'video'" class="video-type-tag">视频</text>

          <!-- 上传中遮罩 -->
          <view v-if="item.status === 'uploading'" class="status-mask uploading">
            <text class="mask-text">上传中...</text>
          </view>

          <!-- 失败遮罩并提供重试 -->
          <view
            v-if="item.status === 'error'"
            class="status-mask error"
            @tap.stop="retryUpload(item)"
          >
            <text class="mask-icon">⚠️</text>
            <text class="mask-text">重试</text>
          </view>

          <!-- 删除按钮 -->
          <view class="remove-btn" @tap.stop="removeAttachment(idx)">
            <text class="remove-x">✕</text>
          </view>
        </view>

        <!-- 添加图片按钮 -->
        <view
          v-if="canAddImage"
          class="add-btn"
          @tap="chooseImage"
        >
          <text class="add-icon">📷</text>
          <text class="add-label">上传图片</text>
          <text class="add-count">{{ imageCount }}/9</text>
        </view>

        <!-- 添加视频按钮 -->
        <view
          v-if="canAddVideo"
          class="add-btn"
          @tap="chooseVideo"
        >
          <text class="add-icon">📹</text>
          <text class="add-label">上传视频</text>
          <text class="add-count">{{ videoCount }}/1</text>
        </view>
      </view>

      <view v-if="uploadingCount > 0" class="uploading-tip">
        <text class="tip-icon">⏳</text>
        <text class="tip-text">正在上传凭证附件 ({{ uploadingCount }}个文件进行中)，请稍候提交...</text>
      </view>
    </view>

    <!-- 底部固定提交栏 -->
    <view class="bottom-bar">
      <button
        class="submit-btn"
        :disabled="submitting || isUploading"
        :class="{ disabled: submitting || isUploading }"
        @tap="handleSubmit"
      >
        <text v-if="isUploading">附件上传中，请稍候...</text>
        <text v-else-if="submitting">正在提交申请...</text>
        <text v-else>提交售后申请</text>
      </button>
    </view>

    <!-- 视频预览弹窗 (禁止自动播放) -->
    <view v-if="previewVideoUrl" class="video-modal-mask" @tap="previewVideoUrl = ''">
      <view class="video-modal-content" @tap.stop>
        <video
          class="preview-video-player"
          :src="previewVideoUrl"
          controls
          :autoplay="false"
        />
        <view class="video-close-btn" @tap="previewVideoUrl = ''">关闭播放</view>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref, computed, reactive } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { getOrderDetail, type OrderDetailResponse } from "@/api/order"
import { applyAfterSale, uploadAfterSaleProof } from "@/api/aftersale"
import { generateRequestId } from "@/utils/tool"

interface AttachmentItem {
  localId: string
  localPath: string
  fileType: "image" | "video"
  status: "uploading" | "success" | "error"
  fileId?: number
  url?: string
  coverUrl?: string
  errorMsg?: string
}

const orderID = ref<number>(0)
const order = ref<OrderDetailResponse | null>(null)
const submitting = ref(false)
const previewVideoUrl = ref("")
const requestId = ref(generateRequestId())

const canRefundOnly = computed(() => {
  if (!order.value) return true
  return order.value.status === 20 || order.value.status === 30
})

const canReturnRefund = computed(() => {
  if (!order.value) return true
  return order.value.status === 30 || order.value.status === 40
})

const reasons = [
  "多拍/拍错/不想要",
  "商品质量问题",
  "商品破损/少件/污渍",
  "尺寸/规格与描述不符",
  "卖家发错货",
  "其它原因",
]
const reasonIndex = ref<number>(-1)

const form = reactive({
  type: 1, // 1 仅退款, 2 退货退款
  reason: "",
  description: "",
})

const attachments = ref<AttachmentItem[]>([])

const imageCount = computed(() => attachments.value.filter((a) => a.fileType === "image").length)
const videoCount = computed(() => attachments.value.filter((a) => a.fileType === "video").length)
const totalCount = computed(() => attachments.value.length)

const canAddImage = computed(() => imageCount.value < 9 && totalCount.value < 10)
const canAddVideo = computed(() => videoCount.value < 1 && totalCount.value < 10)

const uploadingCount = computed(() => attachments.value.filter((a) => a.status === "uploading").length)
const isUploading = computed(() => uploadingCount.value > 0)

onLoad((options) => {
  if (options && options.order_id) {
    orderID.value = Number(options.order_id)
    void fetchOrder()
  }
})

async function fetchOrder() {
  if (!orderID.value) return
  uni.showLoading({ title: "加载中..." })
  try {
    const res = await getOrderDetail(orderID.value)
    if (res && res.code === 1 && res.data) {
      order.value = res.data
      if (res.data.status === 40) {
        form.type = 2
      } else {
        form.type = 1
      }
    }
  } catch (err: any) {
    uni.showToast({ title: err.message || "获取订单失败", icon: "none" })
  } finally {
    uni.hideLoading()
  }
}

function onReasonChange(e: any) {
  const idx = Number(e.detail.value)
  reasonIndex.value = idx
  form.reason = reasons[idx]
}

function chooseImage() {
  const remain = Math.min(9 - imageCount.value, 10 - totalCount.value)
  if (remain <= 0) return

  uni.chooseImage({
    count: remain,
    sizeType: ["compressed", "original"],
    sourceType: ["album", "camera"],
    success: (res) => {
      const paths = res.tempFilePaths || []
      const files = (res.tempFiles || []) as any[]

      for (let i = 0; i < paths.length; i++) {
        const p = paths[i]
        const f = files[i]
        // 校验图片大小 10MB
        if (f && f.size && f.size > 10 * 1024 * 1024) {
          uni.showToast({ title: "单张图片大小不能超过10MB", icon: "none" })
          continue
        }

        const item: AttachmentItem = {
          localId: `img_${Date.now()}_${Math.random()}`,
          localPath: p,
          fileType: "image",
          status: "uploading",
        }
        attachments.value.push(item)
        void doUpload(item)
      }
    },
  })
}

function chooseVideo() {
  if (!canAddVideo.value) return

  uni.chooseVideo({
    sourceType: ["album", "camera"],
    maxDuration: 60,
    success: (res) => {
      if (res.size && res.size > 100 * 1024 * 1024) {
        uni.showToast({ title: "视频大小不能超过100MB", icon: "none" })
        return
      }

      const item: AttachmentItem = {
        localId: `vid_${Date.now()}_${Math.random()}`,
        localPath: res.tempFilePath,
        fileType: "video",
        status: "uploading",
      }
      attachments.value.push(item)
      void doUpload(item)
    },
  })
}

async function doUpload(item: AttachmentItem) {
  const target = attachments.value.find((a) => a.localId === item.localId)
  if (target) {
    target.status = "uploading"
  }
  try {
    const res = await uploadAfterSaleProof(item.localPath)
    const cur = attachments.value.find((a) => a.localId === item.localId)
    if (cur) {
      cur.status = "success"
      cur.fileId = res.id
      cur.url = res.url
      cur.fileType = (res.file_type as "image" | "video") || cur.fileType
    }
  } catch (err: any) {
    const cur = attachments.value.find((a) => a.localId === item.localId)
    if (cur) {
      cur.status = "error"
      cur.errorMsg = err.message || "上传失败"
    }
    uni.showToast({ title: (err && err.message) || "上传失败", icon: "none" })
  }
}

function retryUpload(item: AttachmentItem) {
  const target = attachments.value.find((a) => a.localId === item.localId)
  if (target) {
    void doUpload(target)
  }
}

function removeAttachment(index: number) {
  attachments.value.splice(index, 1)
}

function handlePreview(item: AttachmentItem) {
  if (item.fileType === "image") {
    const imgUrls = attachments.value
      .filter((a) => a.fileType === "image")
      .map((a) => a.url || a.localPath)
    const current = item.url || item.localPath
    uni.previewImage({
      urls: imgUrls,
      current,
    })
  } else if (item.fileType === "video") {
    previewVideoUrl.value = item.url || item.localPath
  }
}

async function handleSubmit() {
  if (isUploading.value) {
    uni.showToast({ title: "附件正在上传中，请稍候", icon: "none" })
    return
  }

  if (order.value?.status === 20 && form.type !== 1) {
    uni.showToast({ title: "待发货订单仅支持仅退款", icon: "none" })
    return
  }
  if (order.value?.status === 40 && form.type !== 2) {
    uni.showToast({ title: "已完成订单仅支持退货退款", icon: "none" })
    return
  }

  if (!form.reason) {
    uni.showToast({ title: "请选择申请原因", icon: "none" })
    return
  }

  const failedUploads = attachments.value.filter((a) => a.status === "error")
  if (failedUploads.length > 0) {
    uni.showToast({ title: "有附件上传失败，请重试或删除", icon: "none" })
    return
  }

  // 严格只提交后端返回的已上传附件 ID
  const proofFileIds = attachments.value
    .filter((a) => a.status === "success" && a.fileId)
    .map((a) => a.fileId!)

  submitting.value = true
  uni.showLoading({ title: "正在提交..." })

  try {
    const res = await applyAfterSale({
      order_id: orderID.value,
      type: form.type,
      reason: form.reason,
      description: form.description.trim(),
      proof_file_ids: proofFileIds,
      request_id: requestId.value,
    })

    uni.showToast({ title: "售后申请提交成功", icon: "success" })
    requestId.value = generateRequestId()
    setTimeout(() => {
      uni.redirectTo({
        url: `/pages/aftersale/detail?id=${res.id}`,
      })
    }, 1200)
  } catch (err: any) {
    uni.showToast({ title: err.message || "提交失败", icon: "none" })
  } finally {
    submitting.value = false
    uni.hideLoading()
  }
}
</script>

<style lang="scss" scoped>
.aftersale-apply-page {
  min-height: 100vh;
  background-color: #f7f7f7;
  padding: 24rpx;
  padding-bottom: 160rpx;
  box-sizing: border-box;
}

.card {
  background: #ffffff;
  border-radius: 16rpx;
  padding: 28rpx;
  margin-bottom: 20rpx;

  .card-title {
    font-size: 30rpx;
    font-weight: 600;
    color: #333333;
    margin-bottom: 20rpx;

    .required {
      color: #e02e24;
      margin-left: 6rpx;
    }
  }

  .card-title-row {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    margin-bottom: 20rpx;

    .card-title {
      margin-bottom: 0;
    }

    .card-subtitle {
      font-size: 22rpx;
      color: #999999;
    }
  }
}

.goods-card {
  .goods-item {
    display: flex;
    gap: 20rpx;
    margin-bottom: 20rpx;

    .goods-img {
      width: 140rpx;
      height: 140rpx;
      border-radius: 8rpx;
      background: #f5f5f5;
    }

    .goods-info {
      flex: 1;
      display: flex;
      flex-direction: column;
      justify-content: space-between;

      .goods-name {
        font-size: 26rpx;
        color: #333333;
        line-height: 36rpx;
        overflow: hidden;
        text-overflow: ellipsis;
        display: -webkit-box;
        -webkit-line-clamp: 2;
        -webkit-box-orient: vertical;
      }

      .goods-spec {
        font-size: 22rpx;
        color: #999999;
      }

      .goods-price-row {
        display: flex;
        justify-content: space-between;
        align-items: center;

        .goods-price {
          font-size: 28rpx;
          font-weight: 600;
          color: #e02e24;
        }

        .goods-qty {
          font-size: 24rpx;
          color: #999999;
        }
      }
    }
  }

  .refund-hint-row {
    display: flex;
    justify-content: flex-end;
    align-items: center;
    border-top: 1rpx solid #f0f0f0;
    padding-top: 18rpx;
    font-size: 24rpx;

    .hint-label {
      color: #666666;
    }

    .refund-max-val {
      font-size: 30rpx;
      font-weight: 700;
      color: #e02e24;
    }
  }
}

.type-selector {
  display: flex;
  flex-direction: column;
  gap: 16rpx;

  .type-btn {
    display: flex;
    align-items: center;
    padding: 24rpx;
    border-radius: 12rpx;
    border: 2rpx solid #eeeeee;
    background: #fafafa;
    transition: all 0.2s ease;

    &.active {
      border-color: #e02e24;
      background: #fff5f5;
    }

    .type-icon {
      font-size: 40rpx;
      margin-right: 20rpx;
    }

    .type-texts {
      display: flex;
      flex-direction: column;

      .type-title {
        font-size: 28rpx;
        font-weight: 600;
        color: #333333;
      }

      .type-desc {
        font-size: 22rpx;
        color: #888888;
        margin-top: 4rpx;
      }
    }
  }
}

.picker-selector {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20rpx 0;
  border-bottom: 1rpx solid #f0f0f0;

  .selected-val {
    font-size: 28rpx;
    color: #333333;
  }

  .placeholder-val {
    font-size: 28rpx;
    color: #999999;
  }

  .arrow {
    font-size: 36rpx;
    color: #cccccc;
  }
}

.desc-input {
  width: 100%;
  height: 180rpx;
  font-size: 26rpx;
  color: #333333;
  line-height: 38rpx;
  box-sizing: border-box;
}

.char-count {
  text-align: right;
  font-size: 22rpx;
  color: #bbbbbb;
  margin-top: 10rpx;
}

.attachments-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 20rpx;

  .attachment-item {
    position: relative;
    width: 156rpx;
    height: 156rpx;
    border-radius: 12rpx;
    overflow: hidden;
    background: #f0f0f0;

    .attachment-thumb {
      width: 100%;
      height: 100%;
    }

    .video-thumb-wrap {
      width: 100%;
      height: 100%;
      position: relative;
      background: #1e293b;
      display: flex;
      align-items: center;
      justify-content: center;

      .video-placeholder {
        .video-icon {
          font-size: 48rpx;
        }
      }

      .play-badge {
        position: absolute;
        width: 44rpx;
        height: 44rpx;
        background: rgba(0, 0, 0, 0.6);
        color: #ffffff;
        border-radius: 50%;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 20rpx;
      }
    }

    .video-type-tag {
      position: absolute;
      bottom: 6rpx;
      left: 6rpx;
      background: rgba(0, 0, 0, 0.6);
      color: #ffffff;
      font-size: 18rpx;
      padding: 2rpx 8rpx;
      border-radius: 4rpx;
    }

    .status-mask {
      position: absolute;
      inset: 0;
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      font-size: 20rpx;
      color: #ffffff;

      &.uploading {
        background: rgba(0, 0, 0, 0.55);
      }

      &.error {
        background: rgba(224, 46, 36, 0.85);

        .mask-icon {
          font-size: 28rpx;
          margin-bottom: 4rpx;
        }
      }
    }

    .remove-btn {
      position: absolute;
      top: 6rpx;
      right: 6rpx;
      width: 36rpx;
      height: 36rpx;
      background: rgba(0, 0, 0, 0.65);
      border-radius: 50%;
      display: flex;
      align-items: center;
      justify-content: center;
      z-index: 10;

      .remove-x {
        color: #ffffff;
        font-size: 20rpx;
        line-height: 1;
      }
    }
  }

  .add-btn {
    width: 156rpx;
    height: 156rpx;
    border-radius: 12rpx;
    border: 2rpx dashed #d9d9d9;
    background: #fafafa;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    box-sizing: border-box;

    .add-icon {
      font-size: 40rpx;
      margin-bottom: 6rpx;
    }

    .add-label {
      font-size: 20rpx;
      color: #666666;
    }

    .add-count {
      font-size: 18rpx;
      color: #aaaaaa;
      margin-top: 2rpx;
    }
  }
}

.uploading-tip {
  display: flex;
  align-items: center;
  gap: 10rpx;
  background: #fff8e6;
  padding: 16rpx 20rpx;
  border-radius: 8rpx;
  margin-top: 20rpx;

  .tip-icon {
    font-size: 26rpx;
  }

  .tip-text {
    font-size: 22rpx;
    color: #d48806;
  }
}

.bottom-bar {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  height: 110rpx;
  background: #ffffff;
  display: flex;
  align-items: center;
  padding: 0 32rpx;
  box-shadow: 0 -4rpx 16rpx rgba(0, 0, 0, 0.06);
  z-index: 99;

  .submit-btn {
    flex: 1;
    height: 80rpx;
    line-height: 80rpx;
    background: #e02e24;
    color: #ffffff;
    font-size: 28rpx;
    font-weight: 600;
    border-radius: 40rpx;
    border: none;

    &.disabled {
      background: #f08b84;
      opacity: 0.8;
    }

    &::after {
      border: none;
    }
  }
}

.video-modal-mask {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.85);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 999;

  .video-modal-content {
    width: 90%;
    display: flex;
    flex-direction: column;
    align-items: center;

    .preview-video-player {
      width: 100%;
      height: 480rpx;
      border-radius: 12rpx;
    }

    .video-close-btn {
      margin-top: 30rpx;
      color: #ffffff;
      font-size: 28rpx;
      padding: 12rpx 36rpx;
      border: 2rpx solid rgba(255, 255, 255, 0.6);
      border-radius: 30rpx;
    }
  }
}
</style>
