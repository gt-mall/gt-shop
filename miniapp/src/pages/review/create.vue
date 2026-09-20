<template>
  <view class="review-create-page">
    <!-- 提交成功状态界面 -->
    <view v-if="submitSuccess" class="success-wrap">
      <view class="success-icon-box">
        <text class="success-icon">✓</text>
      </view>
      <text class="success-title">评价已提交，待审核</text>
      <text class="success-desc">感谢您的评价！审核通过后将公开展示，帮助更多买家做出决策。</text>
      <view class="success-actions">
        <button class="action-btn primary" @tap="goToMineReviews">查看我的评价</button>
        <button class="action-btn outline" @tap="goToOrderDetail">返回订单详情</button>
      </view>
    </view>

    <!-- 加载中 -->
    <view v-else-if="loading" class="state-container">
      <text class="state-text">正在加载商品信息...</text>
    </view>

    <!-- 不可评价状态 -->
    <view v-else-if="!canReview" class="state-container blocked-container">
      <view class="blocked-icon">⚠️</view>
      <text class="blocked-title">{{ blockedTitle }}</text>
      <text class="blocked-desc">{{ blockedReason }}</text>
      <view class="blocked-actions">
        <button v-if="existingReviewId" class="action-btn primary" @tap="goToExistingReview">
          查看已发表评价
        </button>
        <button class="action-btn outline" @tap="goToOrderDetail">返回订单</button>
      </view>
    </view>

    <!-- 正常评价表单 -->
    <view v-else class="form-container">
      <!-- 1. 商品快照卡片 -->
      <view class="card product-card">
        <image
          class="product-img"
          :src="orderItem?.image_url || '/static/logo.png'"
          mode="aspectFill"
        />
        <view class="product-info">
          <text class="product-title">{{ orderItem?.product_name }}</text>
          <text v-if="orderItem?.sku_spec_values" class="product-spec">
            {{ orderItem.sku_spec_values }}
          </text>
          <view class="price-qty-row">
            <text class="product-price">¥{{ orderItem?.unit_price }}</text>
            <text class="product-qty">购买数量: {{ orderItem?.quantity }}</text>
          </view>
        </view>
      </view>

      <!-- 2. 星级评分卡片 -->
      <view class="card rating-card">
        <view class="card-label">商品评分</view>
        <view class="rating-interactive-row">
          <RatingStars
            v-model="rating"
            :interactive="true"
            :size="48"
            @change="handleRatingChange"
          />
          <text class="rating-text-label">{{ ratingText }}</text>
        </view>
      </view>

      <!-- 3. 评价正文输入卡片 -->
      <view class="card content-card">
        <view class="card-label-row">
          <text class="card-label">评价内容</text>
          <text class="char-count" :class="{ 'char-limit': content.length >= 500 }">
            {{ content.length }}/500
          </text>
        </view>
        <textarea
          class="content-textarea"
          v-model="content"
          placeholder="从多个角度评价宝贝，可以帮助更多想买的人～（1-500字）"
          maxlength="500"
          :disabled="submitting"
        />
      </view>

      <!-- 4. 晒图凭证卡片 -->
      <view class="card image-card">
        <view class="card-label-row">
          <text class="card-label">晒图晒单 (选填)</text>
          <text class="card-sub">{{ images.length }}/6</text>
        </view>
        <text class="image-tip">支持 JPG、PNG、WebP，单张图片不超过 5MB</text>

        <view class="image-grid">
          <!-- 已添加图片项 -->
          <view
            v-for="(img, idx) in images"
            :key="img.localId"
            class="image-item"
            @tap="handlePreview(idx)"
          >
            <image class="image-thumb" :src="img.localPath" mode="aspectFill" />

            <!-- 上传中遮罩 -->
            <view v-if="img.status === 'uploading'" class="status-overlay uploading">
              <text class="overlay-text">上传中...</text>
            </view>

            <!-- 上传失败遮罩与重试 -->
            <view
              v-else-if="img.status === 'error'"
              class="status-overlay error"
              @tap.stop="retryUpload(img)"
            >
              <text class="overlay-icon">⚠️</text>
              <text class="overlay-text">重试</text>
            </view>

            <!-- 删除按钮 -->
            <view class="remove-btn" @tap.stop="removeImage(idx)">
              <text class="remove-x">✕</text>
            </view>
          </view>

          <!-- 添加图片按钮 -->
          <view
            v-if="images.length < 6"
            class="add-image-box"
            @tap="chooseImages"
          >
            <text class="add-icon">📷</text>
            <text class="add-text">添加图片</text>
          </view>
        </view>

        <view v-if="hasUploading" class="uploading-tip">
          <text class="tip-text">⏳ 图片正在上传中，请稍候提交...</text>
        </view>
      </view>

      <!-- 5. 匿名评价选项 -->
      <view class="card anon-card" @tap="toggleAnonymous">
        <view class="anon-info">
          <text class="anon-title">匿名评价</text>
          <text class="anon-desc">开启后，公开评价将隐藏您的真实头像与昵称</text>
        </view>
        <switch
          :checked="isAnonymous === 1"
          color="#e02e24"
          @change="onSwitchChange"
        />
      </view>

      <!-- 底部固定提交栏 -->
      <view class="bottom-bar">
        <button
          class="submit-btn"
          :class="{ disabled: isSubmitDisabled }"
          :disabled="isSubmitDisabled"
          :loading="submitting"
          @tap="handleSubmit"
        >
          {{ submitBtnText }}
        </button>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref, computed, watch, onBeforeUnmount } from "vue"
import { onLoad, onUnload } from "@dcloudio/uni-app"
import { getOrderDetail } from "@/api/order"
import type { OrderItemResponse } from "@/api/order"
import { uploadReviewImage, createReview } from "@/api/review"
import { useMemberStore } from "@/stores/member"
import RatingStars from "@/components/RatingStars.vue"

interface ImageItem {
  localId: string
  localPath: string
  fileId?: number
  url?: string
  status: "uploading" | "success" | "error"
  errorMsg?: string
}

const memberStore = useMemberStore()
let isUnmounted = false
const initialUserId = ref<number | null>(memberStore.member?.id || null)

const orderId = ref(0)
const orderItemId = ref(0)
const loading = ref(true)
const submitting = ref(false)
const submitSuccess = ref(false)

const orderItem = ref<OrderItemResponse | null>(null)
const canReview = ref(true)
const existingReviewId = ref(0)
const blockedTitle = ref("暂不可评价")
const blockedReason = ref("")

const rating = ref(5)
const content = ref("")
const isAnonymous = ref(0) // 0: 实名, 1: 匿名
const images = ref<ImageItem[]>([])

// 幂等保护：同一个表单只生成一次 request_id，重试保持不变
const requestId = ref("")

function resetForm() {
  rating.value = 5
  content.value = ""
  images.value = []
  isAnonymous.value = 0
  requestId.value = ""
}

watch(
  () => memberStore.member?.id,
  (newId, oldId) => {
    if (oldId !== undefined && newId !== oldId) {
      resetForm()
      uni.showToast({ title: "登录账号已变更，已清空评价表单", icon: "none" })
      setTimeout(() => {
        if (!isUnmounted) {
          uni.navigateBack()
        }
      }, 800)
    }
  }
)

onBeforeUnmount(() => {
  isUnmounted = true
})

onUnload(() => {
  isUnmounted = true
})

const ratingTextMap: Record<number, string> = {
  1: "非常差",
  2: "差",
  3: "一般",
  4: "满意",
  5: "非常满意",
}

const ratingText = computed(() => ratingTextMap[rating.value] || "满意")

const hasUploading = computed(() => images.value.some((img) => img.status === "uploading"))
const hasUploadError = computed(() => images.value.some((img) => img.status === "error"))

const isSubmitDisabled = computed(() => {
  return (
    submitting.value ||
    hasUploading.value ||
    content.value.trim().length === 0 ||
    content.value.trim().length > 500
  )
})

const submitBtnText = computed(() => {
  if (submitting.value) return "正在提交..."
  if (hasUploading.value) return "图片上传中..."
  if (content.value.trim().length === 0) return "请填写评价内容"
  return "提交评价"
})

onLoad((options: any) => {
  if (options) {
    if (options.order_id) orderId.value = Number(options.order_id)
    if (options.order_item_id) orderItemId.value = Number(options.order_item_id)
  }
  void initEligibility()
})

async function initEligibility() {
  if (!orderId.value || !orderItemId.value) {
    loading.value = false
    canReview.value = false
    blockedTitle.value = "参数缺失"
    blockedReason.value = "缺少有效的订单或商品明细参数"
    return
  }

  const currentUserId = memberStore.member?.id || 0
  loading.value = true
  try {
    const res = await getOrderDetail(orderId.value)
    if (isUnmounted || (memberStore.member?.id || 0) !== currentUserId) return
    if (res && res.code === 1 && res.data) {
      const order = res.data
      const item = order.items?.find((i: OrderItemResponse) => i.id === orderItemId.value)
      if (!item) {
        canReview.value = false
        blockedTitle.value = "未找到该明细"
        blockedReason.value = "订单中不存在指定的商品明细"
      } else {
        orderItem.value = item
        if (item.review_id) {
          canReview.value = false
          existingReviewId.value = item.review_id
          blockedTitle.value = "该商品已评价"
          blockedReason.value = "每件购买的商品明细仅可评价一次，您可以直接查看已发表的评价内容。"
        } else if (!item.can_review) {
          canReview.value = false
          blockedTitle.value = "暂无法发表评价"
          blockedReason.value = item.review_block_reason || "当前订单状态暂不支持评价"
        } else {
          canReview.value = true
        }
      }
    } else {
      canReview.value = false
      blockedTitle.value = "加载订单失败"
      blockedReason.value = (res && res.message) || "无法获取订单信息"
    }
  } catch (err: any) {
    if (isUnmounted || (memberStore.member?.id || 0) !== currentUserId) return
    canReview.value = false
    blockedTitle.value = "加载订单失败"
    blockedReason.value = err?.message || "网络请求失败，请稍后重试"
  } finally {
    if (!isUnmounted && (memberStore.member?.id || 0) === currentUserId) {
      loading.value = false
    }
  }
}

function handleRatingChange(val: number) {
  rating.value = val
}

function toggleAnonymous() {
  isAnonymous.value = isAnonymous.value === 1 ? 0 : 1
}

function onSwitchChange(e: any) {
  isAnonymous.value = e.detail.value ? 1 : 0
}

function chooseImages() {
  const remain = 6 - images.value.length
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
        // 5MB 限制
        if (f && f.size && f.size > 5 * 1024 * 1024) {
          uni.showToast({ title: "单张图片大小不能超过 5MB", icon: "none" })
          continue
        }

        const item: ImageItem = {
          localId: `rev_img_${Date.now()}_${Math.random()}`,
          localPath: p,
          status: "uploading",
        }
        images.value.push(item)
        void doUpload(item)
      }
    },
  })
}

async function doUpload(item: ImageItem) {
  const currentUserId = memberStore.member?.id || 0
  const target = images.value.find((img) => img.localId === item.localId)
  if (target) {
    target.status = "uploading"
  }

  try {
    const res = await uploadReviewImage(item.localPath)
    if (isUnmounted || (memberStore.member?.id || 0) !== currentUserId) return
    const cur = images.value.find((img) => img.localId === item.localId)
    if (cur) {
      cur.status = "success"
      cur.fileId = res.file_id
      cur.url = res.url
    }
  } catch (err: any) {
    if (isUnmounted || (memberStore.member?.id || 0) !== currentUserId) return
    const cur = images.value.find((img) => img.localId === item.localId)
    if (cur) {
      cur.status = "error"
      cur.errorMsg = err?.message || "上传失败"
    }
    uni.showToast({ title: err?.message || "图片上传失败", icon: "none" })
  }
}

function retryUpload(item: ImageItem) {
  void doUpload(item)
}

function removeImage(index: number) {
  images.value.splice(index, 1)
}

function handlePreview(index: number) {
  const urls = images.value.map((img) => img.url || img.localPath)
  uni.previewImage({
    current: urls[index],
    urls,
  })
}

async function handleSubmit() {
  if (isSubmitDisabled.value || isUnmounted) return
  const currentUserId = memberStore.member?.id || 0

  if (!memberStore.isLoggedIn || !currentUserId) {
    uni.showToast({ title: "请先登录", icon: "none" })
    return
  }

  if (initialUserId.value !== null && initialUserId.value !== currentUserId) {
    uni.showToast({ title: "登录账号已变更，请重新进入评价页面", icon: "none" })
    return
  }

  if (hasUploadError.value) {
    uni.showModal({
      title: "提示",
      content: "存在上传失败的图片，请重试或删除失败图片后再提交",
      showCancel: false,
    })
    return
  }

  // 幂等 request_id：若之前已生成过，则保持相同值防重
  if (!requestId.value) {
    requestId.value = `rev_${Date.now()}_${Math.random().toString(36).slice(2, 10)}`
  }

  const successImageIds = images.value
    .filter((img) => img.status === "success" && img.fileId)
    .map((img) => img.fileId as number)

  submitting.value = true
  uni.showLoading({ title: "正在提交...", mask: true })

  try {
    const res = await createReview({
      order_item_id: orderItemId.value,
      rating: rating.value,
      content: content.value.trim(),
      image_ids: successImageIds,
      is_anonymous: isAnonymous.value,
      request_id: requestId.value,
    })

    if (isUnmounted || (memberStore.member?.id || 0) !== currentUserId) return

    if (res && res.id) {
      submitSuccess.value = true
    } else {
      uni.showToast({ title: "提交未成功，请重试", icon: "none" })
    }
  } catch (err: any) {
    if (isUnmounted || (memberStore.member?.id || 0) !== currentUserId) return
    // 保留已填写内容与 request_id，让用户可以直接重试
    uni.showToast({ title: err?.message || "提交失败，请重试", icon: "none" })
  } finally {
    if (!isUnmounted) {
      uni.hideLoading()
      submitting.value = false
    }
  }
}

function goToMineReviews() {
  uni.redirectTo({ url: "/pages/review/mine" })
}

function goToOrderDetail() {
  if (orderId.value > 0) {
    uni.navigateBack()
  } else {
    uni.switchTab({ url: "/pages/index/index" })
  }
}

function goToExistingReview() {
  if (existingReviewId.value > 0) {
    uni.redirectTo({ url: `/pages/review/detail?id=${existingReviewId.value}` })
  }
}
</script>

<style lang="scss" scoped>
.review-create-page {
  min-height: 100vh;
  background-color: #f7f8fa;
  padding: 24rpx;
  padding-bottom: 140rpx;
  box-sizing: border-box;
}

.state-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 100rpx 40rpx;

  .state-text {
    font-size: 28rpx;
    color: #999999;
  }

  &.blocked-container {
    background: #ffffff;
    border-radius: 16rpx;
    margin-top: 40rpx;

    .blocked-icon {
      font-size: 80rpx;
      margin-bottom: 24rpx;
    }

    .blocked-title {
      font-size: 32rpx;
      font-weight: 600;
      color: #333333;
      margin-bottom: 16rpx;
    }

    .blocked-desc {
      font-size: 26rpx;
      color: #666666;
      text-align: center;
      line-height: 1.5;
      margin-bottom: 40rpx;
    }

    .blocked-actions {
      display: flex;
      gap: 20rpx;
      width: 100%;

      .action-btn {
        flex: 1;
        height: 80rpx;
        line-height: 78rpx;
        font-size: 28rpx;
        border-radius: 40rpx;

        &.primary {
          background: #e02e24;
          color: #ffffff;
        }

        &.outline {
          background: #ffffff;
          border: 1rpx solid #dddddd;
          color: #666666;
        }
      }
    }
  }
}

.card {
  background: #ffffff;
  border-radius: 16rpx;
  padding: 28rpx 24rpx;
  margin-bottom: 20rpx;
  box-shadow: 0 2rpx 8rpx rgba(0, 0, 0, 0.02);

  .card-label {
    font-size: 28rpx;
    font-weight: 600;
    color: #333333;
  }

  .card-label-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12rpx;

    .card-sub,
    .char-count {
      font-size: 24rpx;
      color: #999999;

      &.char-limit {
        color: #e02e24;
      }
    }
  }
}

.product-card {
  display: flex;
  align-items: center;
  gap: 20rpx;

  .product-img {
    width: 120rpx;
    height: 120rpx;
    border-radius: 10rpx;
    background: #f5f5f5;
    flex-shrink: 0;
  }

  .product-info {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 8rpx;

    .product-title {
      font-size: 26rpx;
      font-weight: 500;
      color: #333333;
      display: -webkit-box;
      -webkit-line-clamp: 2;
      -webkit-box-orient: vertical;
      overflow: hidden;
      line-height: 1.4;
    }

    .product-spec {
      font-size: 22rpx;
      color: #999999;
      background: #f8f8f8;
      padding: 2rpx 10rpx;
      border-radius: 4rpx;
      align-self: flex-start;
    }

    .price-qty-row {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-top: 4rpx;

      .product-price {
        font-size: 26rpx;
        color: #e02e24;
        font-weight: 600;
      }

      .product-qty {
        font-size: 22rpx;
        color: #999999;
      }
    }
  }
}

.rating-card {
  .rating-interactive-row {
    display: flex;
    align-items: center;
    gap: 24rpx;
    margin-top: 20rpx;

    .rating-text-label {
      font-size: 28rpx;
      color: #ff9800;
      font-weight: 600;
    }
  }
}

.content-card {
  .content-textarea {
    width: 100%;
    height: 220rpx;
    box-sizing: border-box;
    font-size: 28rpx;
    line-height: 1.5;
    color: #333333;
    padding: 16rpx 0;
  }
}

.image-card {
  .image-tip {
    font-size: 22rpx;
    color: #999999;
    margin-bottom: 20rpx;
    display: block;
  }

  .image-grid {
    display: flex;
    flex-wrap: wrap;
    gap: 16rpx;

    .image-item {
      position: relative;
      width: 154rpx;
      height: 154rpx;
      border-radius: 12rpx;
      overflow: hidden;

      .image-thumb {
        width: 100%;
        height: 100%;
      }

      .status-overlay {
        position: absolute;
        inset: 0;
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        background: rgba(0, 0, 0, 0.55);

        .overlay-text {
          font-size: 20rpx;
          color: #ffffff;
        }

        &.error {
          background: rgba(224, 46, 36, 0.7);

          .overlay-icon {
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
        border-radius: 50%;
        background: rgba(0, 0, 0, 0.6);
        display: flex;
        align-items: center;
        justify-content: center;

        .remove-x {
          color: #ffffff;
          font-size: 22rpx;
          line-height: 1;
        }
      }
    }

    .add-image-box {
      width: 154rpx;
      height: 154rpx;
      border-radius: 12rpx;
      border: 2rpx dashed #d9d9d9;
      background: #fafafa;
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      gap: 8rpx;

      .add-icon {
        font-size: 44rpx;
      }

      .add-text {
        font-size: 22rpx;
        color: #999999;
      }
    }
  }

  .uploading-tip {
    margin-top: 16rpx;

    .tip-text {
      font-size: 22rpx;
      color: #ff9800;
    }
  }
}

.anon-card {
  display: flex;
  align-items: center;
  justify-content: space-between;

  .anon-info {
    display: flex;
    flex-direction: column;
    gap: 6rpx;

    .anon-title {
      font-size: 28rpx;
      font-weight: 500;
      color: #333333;
    }

    .anon-desc {
      font-size: 22rpx;
      color: #999999;
    }
  }
}

.bottom-bar {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  background: #ffffff;
  padding: 16rpx 24rpx calc(16rpx + env(safe-area-inset-bottom));
  box-shadow: 0 -2rpx 10rpx rgba(0, 0, 0, 0.05);
  z-index: 100;

  .submit-btn {
    height: 84rpx;
    line-height: 84rpx;
    background: #e02e24;
    color: #ffffff;
    font-size: 30rpx;
    font-weight: 600;
    border-radius: 42rpx;
    border: none;

    &.disabled {
      background: #f0a39f;
      color: #ffffff;
    }
  }
}

.success-wrap {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 80rpx 40rpx;
  background: #ffffff;
  border-radius: 20rpx;
  margin-top: 60rpx;

  .success-icon-box {
    width: 120rpx;
    height: 120rpx;
    border-radius: 50%;
    background: #52c41a;
    display: flex;
    align-items: center;
    justify-content: center;
    margin-bottom: 32rpx;

    .success-icon {
      font-size: 64rpx;
      color: #ffffff;
      font-weight: bold;
    }
  }

  .success-title {
    font-size: 36rpx;
    font-weight: bold;
    color: #333333;
    margin-bottom: 16rpx;
  }

  .success-desc {
    font-size: 26rpx;
    color: #666666;
    text-align: center;
    line-height: 1.5;
    margin-bottom: 60rpx;
  }

  .success-actions {
    display: flex;
    flex-direction: column;
    gap: 24rpx;
    width: 100%;

    .action-btn {
      height: 84rpx;
      line-height: 82rpx;
      font-size: 28rpx;
      font-weight: 500;
      border-radius: 42rpx;
      width: 100%;

      &.primary {
        background: #e02e24;
        color: #ffffff;
      }

      &.outline {
        background: #ffffff;
        border: 1rpx solid #dddddd;
        color: #666666;
      }
    }
  }
}
</style>
