<template>
  <view class="aftersale-detail-page" v-if="detail">
    <!-- 状态横幅 -->
    <view class="status-banner" :class="'status-bg-' + detail.status">
      <view class="status-header">
        <text class="status-title">{{ detail.status_text }}</text>
        <text class="status-type-tag">{{ detail.type_text }}</text>
      </view>
      <text class="status-desc">{{ statusTip }}</text>
    </view>

    <!-- 拒绝原因警示卡片 (若被拒绝) -->
    <view v-if="detail.status === 60 && detail.reject_reason" class="reject-alert-card">
      <view class="alert-title">
        <text class="alert-icon">⚠️</text>
        <text>商家拒绝说明</text>
      </view>
      <text class="alert-desc">{{ detail.reject_reason }}</text>
    </view>

    <!-- 退货地址卡片 (若为退货退款且审核通过待退货/收货) -->
    <view v-if="detail.type === 2 && detail.return_address" class="card return-address-card">
      <view class="card-header">
        <text class="card-title">商家退货收件信息</text>
        <button class="copy-address-btn" size="mini" @tap="copyReturnAddress">复制地址</button>
      </view>
      <view class="address-content">
        <view class="address-row">
          <text class="address-label">收件人：</text>
          <text class="address-val">{{ detail.return_receiver }} ({{ detail.return_phone }})</text>
        </view>
        <view class="address-row">
          <text class="address-label">退货地址：</text>
          <text class="address-val">{{ detail.return_address }}</text>
        </view>
      </view>
    </view>

    <!-- 买家填写退货物流卡片 (待买家退货 20) -->
    <view v-if="detail.status === 20 && detail.type === 2" class="card return-logistics-card">
      <view class="card-title">填写退货运单信息</view>
      <view class="form-row">
        <text class="form-label">物流公司</text>
        <input
          class="form-input"
          v-model="logisticsForm.company"
          placeholder="如：顺丰速运、中通快递"
        />
      </view>
      <view class="form-row">
        <text class="form-label">物流单号</text>
        <input
          class="form-input"
          v-model="logisticsForm.trackingNo"
          placeholder="请输入退货快递单号"
        />
      </view>
      <button
        class="submit-logistics-btn"
        :disabled="logisticsSubmitting"
        @tap="submitLogistics"
      >
        {{ logisticsSubmitting ? "正在提交..." : "提交退货物流单号" }}
      </button>
    </view>

    <!-- 已提交退货物流信息展示 -->
    <view v-if="detail.buyer_return_no" class="card">
      <view class="card-title">买家退货物流</view>
      <view class="meta-row">
        <text class="meta-label">物流公司</text>
        <text class="meta-val">{{ detail.buyer_return_company }}</text>
      </view>
      <view class="meta-row">
        <text class="meta-label">物流单号</text>
        <view class="copy-val-row">
          <text class="meta-val">{{ detail.buyer_return_no }}</text>
          <text class="copy-action" @tap="copyText(detail.buyer_return_no)">复制</text>
        </view>
      </view>
      <view v-if="detail.buyer_returned_at" class="meta-row">
        <text class="meta-label">发货时间</text>
        <text class="meta-val">{{ formatDateTime(detail.buyer_returned_at) }}</text>
      </view>
    </view>

    <!-- 售后商品明细 -->
    <view class="card">
      <view class="card-title">退款商品</view>
      <view v-for="g in detail.items" :key="g.id" class="goods-row">
        <image class="goods-img" :src="g.sku_image_url" mode="aspectFill" />
        <view class="goods-info">
          <text class="goods-name">{{ g.product_name }}</text>
          <text v-if="g.sku_text" class="goods-spec">{{ g.sku_text }}</text>
          <view class="goods-meta">
            <text class="goods-qty">申请数量: {{ g.quantity }} 件</text>
            <text class="goods-amount">退款: ¥{{ g.refund_amount }}</text>
          </view>
        </view>
      </view>
    </view>

    <!-- 凭证附件区 (清晰区分图片与视频) -->
    <view v-if="detail.proof_files && detail.proof_files.length > 0" class="card">
      <view class="card-title">凭证附件</view>

      <!-- 图片凭证 -->
      <view v-if="imageProofs.length > 0" class="proof-section">
        <view class="proof-section-title">
          <text>图片凭证 ({{ imageProofs.length }}张，点击放大)</text>
        </view>
        <view class="proofs-grid">
          <view
            v-for="(img, idx) in imageProofs"
            :key="img.id"
            class="proof-img-item"
            @tap="previewImage(idx)"
          >
            <image class="proof-img" :src="img.url" mode="aspectFill" />
          </view>
        </view>
      </view>

      <!-- 视频凭证 -->
      <view v-if="videoProofs.length > 0" class="proof-section">
        <view class="proof-section-title">
          <text>视频凭证 ({{ videoProofs.length }}个，点击播放)</text>
        </view>
        <view class="proofs-grid">
          <view
            v-for="vid in videoProofs"
            :key="vid.id"
            class="proof-vid-item"
            @tap="playVideo(vid.url)"
          >
            <image
              v-if="vid.cover_url"
              class="proof-vid-cover"
              :src="vid.cover_url"
              mode="aspectFill"
            />
            <view v-else class="proof-vid-placeholder">
              <text class="vid-icon">🎬</text>
            </view>
            <view class="vid-play-btn">▶</view>
            <text class="vid-tag">视频</text>
          </view>
        </view>
      </view>
    </view>

    <!-- 退款详情 (若存在退款流水) -->
    <view v-if="detail.refund" class="card">
      <view class="card-title">退款进度</view>
      <view class="meta-row">
        <text class="meta-label">退款状态</text>
        <text class="meta-val highlight-val">{{ detail.refund.status_text }}</text>
      </view>
      <view class="meta-row">
        <text class="meta-label">退款金额</text>
        <text class="meta-val price-val">¥{{ detail.refund.refund_amount }}</text>
      </view>
      <view v-if="detail.refund.refunded_at" class="meta-row">
        <text class="meta-label">退款时间</text>
        <text class="meta-val">{{ formatDateTime(detail.refund.refunded_at) }}</text>
      </view>
    </view>

    <!-- 售后单基本信息 -->
    <view class="card">
      <view class="card-title">售后信息</view>
      <view class="meta-row">
        <text class="meta-label">售后单号</text>
        <view class="copy-val-row">
          <text class="meta-val">{{ detail.after_sale_no }}</text>
          <text class="copy-action" @tap="copyText(detail.after_sale_no)">复制</text>
        </view>
      </view>
      <view class="meta-row">
        <text class="meta-label">关联订单</text>
        <text class="meta-val link-val" @tap="goToOrder(detail.order_id)">查看关联订单 ›</text>
      </view>
      <view class="meta-row">
        <text class="meta-label">申请时间</text>
        <text class="meta-val">{{ formatDateTime(detail.applied_at || detail.created_at) }}</text>
      </view>
      <view class="meta-row">
        <text class="meta-label">退款金额</text>
        <text class="meta-val price-val">¥{{ detail.refund_amount }}</text>
      </view>
      <view class="meta-row">
        <text class="meta-label">申请原因</text>
        <text class="meta-val">{{ detail.reason }}</text>
      </view>
      <view v-if="detail.description" class="meta-row desc-row">
        <text class="meta-label">详细描述</text>
        <text class="meta-val desc-text">{{ detail.description }}</text>
      </view>
    </view>

    <!-- 协商流转日志 -->
    <view v-if="detail.logs && detail.logs.length > 0" class="card logs-card">
      <view class="card-title">协商历史</view>
      <view class="timeline">
        <view v-for="(log, idx) in detail.logs" :key="idx" class="timeline-item">
          <view class="timeline-dot"></view>
          <view class="timeline-body">
            <view class="timeline-header">
              <text class="timeline-action">{{ log.action }}</text>
              <text class="timeline-time">{{ formatDateTime(log.created_at) }}</text>
            </view>
            <text class="timeline-remark">{{ log.remark }}</text>
          </view>
        </view>
      </view>
    </view>

    <!-- 底部操作栏 -->
    <view v-if="canCancel || canReapply" class="bottom-bar">
      <button v-if="canCancel" class="cancel-btn" @tap="handleCancel">
        撤销售后申请
      </button>
      <button v-if="canReapply" class="reapply-btn" @tap="handleReapply">
        重新申请售后
      </button>
    </view>

    <!-- 视频播放弹窗 (controls 开启，禁止自动播放) -->
    <view v-if="activeVideoUrl" class="video-modal-mask" @tap="activeVideoUrl = ''">
      <view class="video-modal-content" @tap.stop>
        <video
          class="video-player"
          :src="activeVideoUrl"
          controls
          :autoplay="false"
        />
        <view class="video-close-btn" @tap="activeVideoUrl = ''">关闭视频</view>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref, computed, reactive } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import {
  getAfterSaleDetail,
  cancelAfterSale,
  fillReturnLogistics,
  type MiniappAfterSaleDetailResponse,
} from "@/api/aftersale"

const afterSaleID = ref<number>(0)
const detail = ref<MiniappAfterSaleDetailResponse | null>(null)
const activeVideoUrl = ref("")

const logisticsForm = reactive({
  company: "",
  trackingNo: "",
})
const logisticsSubmitting = ref(false)

const imageProofs = computed(() => {
  if (!detail.value?.proof_files) return []
  return detail.value.proof_files.filter((f) => f.file_type === "image" || !f.file_type)
})

const videoProofs = computed(() => {
  if (!detail.value?.proof_files) return []
  return detail.value.proof_files.filter((f) => f.file_type === "video")
})

const canCancel = computed(() => {
  const s = detail.value?.status
  return s === 10 || s === 20
})

const canReapply = computed(() => {
  const s = detail.value?.status
  return s === 60 || s === 70
})

const statusTip = computed(() => {
  switch (detail.value?.status) {
    case 10:
      return "您的售后申请已提交，等待商家审核处理"
    case 20:
      return "商家已同意退货申请，请按照商家地址退回商品并填写运单号"
    case 30:
      return "您已提交退货物流单号，等待商家收件确认"
    case 40:
      return "退款流程已启动，款项正在原路退回中"
    case 50:
      return "售后已圆满完成，退款已成功到账"
    case 60:
      return "商家已拒绝本次售后申请，如有疑问可联系客服"
    case 70:
      return "买家已撤销售后申请"
    case 80:
      return "退款处理异常，平台正在重试处理中"
    default:
      return ""
  }
})

onLoad((options) => {
  if (options && options.id) {
    afterSaleID.value = Number(options.id)
    void fetchDetail()
  }
})

async function fetchDetail() {
  if (!afterSaleID.value) return
  uni.showLoading({ title: "加载中..." })
  try {
    const res = await getAfterSaleDetail(afterSaleID.value)
    detail.value = res
  } catch (err: any) {
    uni.showToast({ title: err.message || "获取售后详情失败", icon: "none" })
  } finally {
    uni.hideLoading()
  }
}

function previewImage(index: number) {
  const urls = imageProofs.value.map((f) => f.url)
  uni.previewImage({
    urls,
    current: urls[index],
  })
}

function playVideo(url: string) {
  activeVideoUrl.value = url
}

function copyText(val: string) {
  if (!val) return
  uni.setClipboardData({
    data: val,
    success: () => uni.showToast({ title: "已复制到剪贴板", icon: "none" }),
  })
}

function copyReturnAddress() {
  if (!detail.value) return
  const fullText = `${detail.value.return_receiver} ${detail.value.return_phone} ${detail.value.return_address}`
  copyText(fullText)
}

async function submitLogistics() {
  if (!logisticsForm.company.trim()) {
    uni.showToast({ title: "请输入物流公司名称", icon: "none" })
    return
  }
  if (!logisticsForm.trackingNo.trim()) {
    uni.showToast({ title: "请输入物流单号", icon: "none" })
    return
  }

  logisticsSubmitting.value = true
  try {
    await fillReturnLogistics({
      after_sale_id: afterSaleID.value,
      company: logisticsForm.company.trim(),
      tracking_no: logisticsForm.trackingNo.trim(),
    })
    uni.showToast({ title: "物流信息提交成功", icon: "success" })
    void fetchDetail()
  } catch (err: any) {
    uni.showToast({ title: err.message || "提交失败", icon: "none" })
  } finally {
    logisticsSubmitting.value = false
  }
}

function handleCancel() {
  uni.showModal({
    title: "确认撤销",
    content: "撤销后该售后流程将关闭，是否确定撤销售后申请？",
    confirmColor: "#e02e24",
    success: async (res) => {
      if (res.confirm) {
        uni.showLoading({ title: "正在撤销..." })
        try {
          await cancelAfterSale(afterSaleID.value)
          uni.showToast({ title: "已成功撤销售后申请", icon: "success" })
          void fetchDetail()
        } catch (err: any) {
          uni.showToast({ title: err.message || "撤销失败", icon: "none" })
        } finally {
          uni.hideLoading()
        }
      }
    },
  })
}

function handleReapply() {
  if (!detail.value) return
  uni.navigateTo({
    url: `/pages/aftersale/apply?order_id=${detail.value.order_id}`,
  })
}

function goToOrder(orderId: number) {
  uni.navigateTo({
    url: `/pages/order/detail?id=${orderId}`,
  })
}

function formatDateTime(val?: any) {
  if (!val) return "-"
  const d = new Date(val)
  if (isNaN(d.getTime())) return String(val)
  const pad = (n: number) => (n < 10 ? "0" + n : n)
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}
</script>

<style lang="scss" scoped>
.aftersale-detail-page {
  min-height: 100vh;
  background-color: #f7f7f7;
  padding-bottom: 140rpx;
}

.status-banner {
  padding: 40rpx 32rpx;
  color: #ffffff;

  &.status-bg-10 {
    background: linear-gradient(135deg, #fa8c16, #fa541c);
  }
  &.status-bg-20 {
    background: linear-gradient(135deg, #1890ff, #096dd9);
  }
  &.status-bg-30 {
    background: linear-gradient(135deg, #13c2c2, #08979c);
  }
  &.status-bg-40 {
    background: linear-gradient(135deg, #2f54eb, #1d39c4);
  }
  &.status-bg-50 {
    background: linear-gradient(135deg, #52c41a, #389e0d);
  }
  &.status-bg-60 {
    background: linear-gradient(135deg, #f5222d, #cf1322);
  }
  &.status-bg-70 {
    background: linear-gradient(135deg, #8c8c8c, #595959);
  }
  &.status-bg-80 {
    background: linear-gradient(135deg, #eb2f96, #c41d7f);
  }

  .status-header {
    display: flex;
    align-items: center;
    gap: 16rpx;
    margin-bottom: 12rpx;

    .status-title {
      font-size: 36rpx;
      font-weight: 700;
    }

    .status-type-tag {
      font-size: 22rpx;
      background: rgba(255, 255, 255, 0.25);
      padding: 2rpx 14rpx;
      border-radius: 20rpx;
    }
  }

  .status-desc {
    font-size: 24rpx;
    opacity: 0.92;
    line-height: 34rpx;
  }
}

.reject-alert-card {
  margin: 20rpx 24rpx;
  padding: 24rpx;
  background: #fff1f0;
  border: 1rpx solid #ffa39e;
  border-radius: 12rpx;

  .alert-title {
    display: flex;
    align-items: center;
    gap: 8rpx;
    font-size: 26rpx;
    font-weight: 600;
    color: #cf1322;
    margin-bottom: 10rpx;
  }

  .alert-desc {
    font-size: 24rpx;
    color: #5c0011;
    line-height: 34rpx;
  }
}

.card {
  background: #ffffff;
  border-radius: 16rpx;
  padding: 28rpx;
  margin: 20rpx 24rpx;

  .card-title {
    font-size: 28rpx;
    font-weight: 600;
    color: #333333;
    margin-bottom: 20rpx;
  }

  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 20rpx;

    .card-title {
      margin-bottom: 0;
    }
  }
}

.return-address-card {
  .copy-address-btn {
    margin: 0;
    padding: 0 20rpx;
    height: 48rpx;
    line-height: 46rpx;
    font-size: 22rpx;
    color: #1890ff;
    border: 1rpx solid #91d5ff;
    background: #e6f7ff;
    border-radius: 24rpx;

    &::after {
      border: none;
    }
  }

  .address-content {
    background: #fafafa;
    padding: 20rpx;
    border-radius: 8rpx;

    .address-row {
      display: flex;
      margin-bottom: 10rpx;
      font-size: 24rpx;

      &:last-child {
        margin-bottom: 0;
      }

      .address-label {
        color: #888888;
        width: 130rpx;
        flex-shrink: 0;
      }

      .address-val {
        color: #333333;
        line-height: 34rpx;
      }
    }
  }
}

.return-logistics-card {
  .form-row {
    display: flex;
    align-items: center;
    padding: 16rpx 0;
    border-bottom: 1rpx solid #f0f0f0;

    .form-label {
      width: 140rpx;
      font-size: 26rpx;
      color: #666666;
    }

    .form-input {
      flex: 1;
      font-size: 26rpx;
      color: #333333;
    }
  }

  .submit-logistics-btn {
    margin-top: 24rpx;
    height: 72rpx;
    line-height: 72rpx;
    background: #1890ff;
    color: #ffffff;
    font-size: 26rpx;
    border-radius: 36rpx;
    border: none;

    &::after {
      border: none;
    }
  }
}

.goods-row {
  display: flex;
  gap: 16rpx;
  padding: 16rpx 0;
  border-bottom: 1rpx solid #f5f5f5;

  &:last-child {
    border-bottom: none;
    padding-bottom: 0;
  }

  .goods-img {
    width: 120rpx;
    height: 120rpx;
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
      line-height: 34rpx;
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

    .goods-meta {
      display: flex;
      justify-content: space-between;
      align-items: center;

      .goods-qty {
        font-size: 22rpx;
        color: #888888;
      }

      .goods-amount {
        font-size: 24rpx;
        color: #e02e24;
        font-weight: 600;
      }
    }
  }
}

.proof-section {
  margin-top: 16rpx;

  &:first-of-type {
    margin-top: 0;
  }

  .proof-section-title {
    font-size: 24rpx;
    color: #888888;
    margin-bottom: 12rpx;
  }

  .proofs-grid {
    display: flex;
    flex-wrap: wrap;
    gap: 16rpx;

    .proof-img-item {
      width: 140rpx;
      height: 140rpx;
      border-radius: 8rpx;
      overflow: hidden;
      background: #f0f0f0;

      .proof-img {
        width: 100%;
        height: 100%;
      }
    }

    .proof-vid-item {
      position: relative;
      width: 180rpx;
      height: 140rpx;
      border-radius: 8rpx;
      overflow: hidden;
      background: #1e293b;
      display: flex;
      align-items: center;
      justify-content: center;

      .proof-vid-cover {
        width: 100%;
        height: 100%;
        opacity: 0.7;
      }

      .proof-vid-placeholder {
        .vid-icon {
          font-size: 44rpx;
        }
      }

      .vid-play-btn {
        position: absolute;
        width: 44rpx;
        height: 44rpx;
        background: rgba(0, 0, 0, 0.65);
        color: #ffffff;
        border-radius: 50%;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 20rpx;
      }

      .vid-tag {
        position: absolute;
        bottom: 6rpx;
        left: 6rpx;
        background: rgba(0, 0, 0, 0.6);
        color: #ffffff;
        font-size: 18rpx;
        padding: 2rpx 8rpx;
        border-radius: 4rpx;
      }
    }
  }
}

.meta-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 14rpx 0;
  border-bottom: 1rpx solid #f9f9f9;
  font-size: 26rpx;

  &:last-child {
    border-bottom: none;
    padding-bottom: 0;
  }

  .meta-label {
    color: #888888;
  }

  .meta-val {
    color: #333333;
  }

  .copy-val-row {
    display: flex;
    align-items: center;
    gap: 12rpx;

    .copy-action {
      color: #1890ff;
      font-size: 22rpx;
      padding: 2rpx 10rpx;
      border: 1rpx solid #91d5ff;
      border-radius: 4rpx;
      background: #e6f7ff;
    }
  }

  .link-val {
    color: #1890ff;
  }

  .price-val {
    color: #e02e24;
    font-weight: 700;
  }

  .highlight-val {
    color: #52c41a;
    font-weight: 600;
  }

  &.desc-row {
    flex-direction: column;
    align-items: flex-start;
    gap: 8rpx;

    .desc-text {
      color: #666666;
      line-height: 36rpx;
      background: #fafafa;
      padding: 16rpx;
      border-radius: 8rpx;
      width: 100%;
      box-sizing: border-box;
    }
  }
}

.timeline {
  padding: 10rpx 0;

  .timeline-item {
    position: relative;
    padding-left: 36rpx;
    padding-bottom: 24rpx;
    border-left: 2rpx solid #e8e8e8;

    &:last-child {
      border-left-color: transparent;
      padding-bottom: 0;
    }

    .timeline-dot {
      position: absolute;
      left: -9rpx;
      top: 6rpx;
      width: 16rpx;
      height: 16rpx;
      background: #1890ff;
      border-radius: 50%;
    }

    .timeline-body {
      .timeline-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 6rpx;

        .timeline-action {
          font-size: 26rpx;
          font-weight: 600;
          color: #333333;
        }

        .timeline-time {
          font-size: 22rpx;
          color: #aaaaaa;
        }
      }

      .timeline-remark {
        font-size: 24rpx;
        color: #666666;
        line-height: 34rpx;
      }
    }
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
  justify-content: flex-end;
  padding: 0 32rpx;
  gap: 20rpx;
  box-shadow: 0 -4rpx 16rpx rgba(0, 0, 0, 0.05);
  z-index: 99;

  button {
    margin: 0;
    padding: 0 32rpx;
    height: 72rpx;
    line-height: 70rpx;
    font-size: 26rpx;
    border-radius: 36rpx;
    box-sizing: border-box;

    &::after {
      border: none;
    }

    &.cancel-btn {
      background: #ffffff;
      border: 2rpx solid #dddddd;
      color: #666666;
    }

    &.reapply-btn {
      background: #e02e24;
      color: #ffffff;
      border: 2rpx solid #e02e24;
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

    .video-player {
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
