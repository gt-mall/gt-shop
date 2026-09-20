<template>
  <view v-if="visible" class="poster-modal-mask" @tap.self="handleClose">
    <view class="poster-modal-content">
      <!-- 头部标题与关闭按钮 -->
      <view class="poster-modal-header">
        <text class="header-title">商品分享海报</text>
        <text class="close-icon" @tap="handleClose">✕</text>
      </view>

      <!-- 绘制/加载中状态 -->
      <view v-if="loading" class="poster-state-box loading-box">
        <text class="loading-spinner">⏳</text>
        <text class="loading-text">{{ loadingTip }}</text>
      </view>

      <!-- 绘制失败状态 -->
      <view v-else-if="error" class="poster-state-box error-box">
        <text class="error-icon">⚠️</text>
        <text class="error-text">{{ error }}</text>
        <button class="retry-btn" @tap="handleGeneratePoster">重试生成</button>
      </view>

      <!-- 海报预览与保存区 -->
      <view v-else class="poster-preview-box">
        <image
          v-if="posterImagePath"
          :src="posterImagePath"
          mode="widthFix"
          class="poster-preview-img"
          :show-menu-by-longpress="true"
        />

        <view class="poster-action-bar">
          <!-- #ifdef MP-WEIXIN -->
          <button
            class="save-btn primary-btn"
            :loading="saving"
            :disabled="saving"
            @tap="handleSaveToAlbum"
          >
            保存到相册
          </button>
          <!-- #endif -->

          <!-- #ifndef MP-WEIXIN -->
          <button class="save-btn primary-btn" @tap="handleH5Save">
            长按图片保存 / 点击下载
          </button>
          <!-- #endif -->

          <button class="save-btn cancel-btn" @tap="handleClose">关闭</button>
        </view>
        <text class="poster-bottom-tip">* 海报价格仅供参考，实际以商品页面为准</text>
      </view>
    </view>

    <!-- 离屏 Canvas (用于生成高分辨率海报，避免干扰视口展示) -->
    <view class="canvas-offscreen-wrap">
      <canvas
        canvas-id="productPosterCanvas"
        id="productPosterCanvas"
        class="offscreen-canvas"
        :style="{
          width: canvasWidth * dpr + 'px',
          height: canvasHeight * dpr + 'px',
        }"
      />
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref, computed, watch, getCurrentInstance, onBeforeUnmount, nextTick } from "vue"
import { useAppStore } from "@/stores/app"
import { useMemberStore } from "@/stores/member"
import { getProductDetail, getProductShareCode, type ProductDetailData } from "@/api/product"
import { resolveImageUrl } from "@/utils/format"

interface Props {
  productId: number | string
  visible: boolean
  product?: ProductDetailData | null
}

const props = withDefaults(defineProps<Props>(), {
  product: null,
})

const emit = defineEmits<{
  (e: "update:visible", val: boolean): void
  (e: "close"): void
}>()

const instance = getCurrentInstance()
const appStore = useAppStore()
const memberStore = useMemberStore()

const canvasWidth = 375
const canvasHeight = 580
const dpr = computed(() => Math.min(Math.max(uni.getSystemInfoSync().pixelRatio || 2, 1), 3))

const loading = ref<boolean>(false)
const loadingTip = ref<string>("正在准备海报...")
const error = ref<string | null>(null)
const saving = ref<boolean>(false)
const posterImagePath = ref<string>("")

let currentSessionSeq = 0
let isDisposed = false

let running = false
let regenerate = false
const downloads = new Set<() => void>()
const objectURLs = new Set<string>()

function invalidate() {
  currentSessionSeq++
  posterImagePath.value = ""
  error.value = null
  for (const cancel of downloads) cancel()
  downloads.clear()
}

watch(
  () => [props.visible, props.productId, props.product?.price, props.product?.name, props.product?.main_image_url, memberStore.token],
  () => {
    invalidate()
    if (props.visible) void handleGeneratePoster()
  },
  { immediate: true }
)

onBeforeUnmount(() => {
  isDisposed = true
  invalidate()
  releaseImages()
})

function releaseImages() {
  // #ifdef H5
  for (const url of objectURLs) URL.revokeObjectURL(url)
  // #endif
  objectURLs.clear()
}

function handleClose() {
  emit("update:visible", false)
  emit("close")
}

/**
 * 校验登录状态
 */
function checkAuth(): boolean {
  if (!memberStore.isLoggedIn) {
    error.value = "请先登录后生成商品海报"
    uni.showToast({ title: "请先登录", icon: "none" })
    return false
  }
  return true
}

/**
 * 下载远程图片到本地临时文件 (兼容小程序与 H5)
 */
async function downloadImageFile(url: string): Promise<string> {
  const resolvedUrl = resolveImageUrl(url)
  if (!resolvedUrl) throw new Error("图片地址为空")
  const maxSize = 10 * 1024 * 1024
  // #ifdef H5
  const controller = new AbortController()
  const abort = () => controller.abort()
  downloads.add(abort)
  const timer = setTimeout(abort, 12000)
  const [result] = await Promise.allSettled([(async () => {
    const res = await fetch(resolvedUrl, { mode: "cors", signal: controller.signal })
    if (!res.ok || Number(res.headers.get("content-length")) > maxSize) throw new Error("图片下载失败或超过 10MB")
    const blob = await res.blob()
    if (blob.size > maxSize || !blob.type.startsWith("image/")) throw new Error("图片大小或格式不支持")
    const path = URL.createObjectURL(blob)
    objectURLs.add(path)
    return path
  })()])
  clearTimeout(timer)
  downloads.delete(abort)
  if (result.status === "rejected") throw new Error("图片下载失败，请检查网络及图片跨域配置")
  return result.value
  // #endif

  // #ifndef H5
  return new Promise((resolve, reject) => {
    let done = false
    const finish = (path?: string, message = "图片下载失败") => {
      if (done) return
      done = true
      clearTimeout(timer)
      downloads.delete(abort)
      if (path) resolve(path)
      else reject(new Error(message))
    }
    const abort = () => { task.abort(); finish(undefined, "图片下载已取消") }
    const timer = setTimeout(abort, 12000)
    const task = uni.downloadFile({
      url: resolvedUrl,
      success(res) {
        if (res.statusCode !== 200 || !res.tempFilePath) { finish(); return }
        uni.getFileInfo({
          filePath: res.tempFilePath,
          success(info) { info.size <= maxSize ? finish(res.tempFilePath) : finish(undefined, "图片超过 10MB") },
          fail() { finish() },
        })
      },
      fail() { finish() },
    })
    downloads.add(abort)
  })
  // #endif
}

async function handleGeneratePoster() {
  if (isDisposed || !props.visible) return
  if (running) { regenerate = true; return }
  if (!props.productId || Number(props.productId) <= 0) { error.value = "商品信息无效"; return }
  if (!checkAuth()) return
  running = true
  do {
    regenerate = false
    const session = ++currentSessionSeq
    const id = props.productId
    const token = memberStore.token
    const active = () => session === currentSessionSeq && !isDisposed && props.visible && memberStore.token === token
    loading.value = true
    error.value = null
    posterImagePath.value = ""
    const [result] = await Promise.allSettled([(async () => {
      loadingTip.value = "正在加载最新商品信息..."
      const res = await getProductDetail(id)
      if (!active()) return
      if (res.code !== 1 || !res.data || res.data.id !== Number(id)) throw new Error("获取商品信息失败")
      const product = res.data
      loadingTip.value = "正在生成小程序码..."
      const code = await getProductShareCode(id)
      if (!active()) return
      if (code.code !== 1 || !code.data?.url) throw new Error("生成小程序码失败")
      loadingTip.value = "正在下载图片素材..."
      const images = await Promise.allSettled([
        product.main_image_url ? downloadImageFile(product.main_image_url) : Promise.resolve(""),
        downloadImageFile(code.data.url),
      ])
      if (!active()) return
      if (images[1].status === "rejected") throw images[1].reason
      const main = images[0].status === "fulfilled" ? images[0].value : ""
      loadingTip.value = "正在绘制海报..."
      await nextTick()
      if (!active()) return
      const path = await drawPosterToCanvas(product, main, images[1].value)
      if (active()) posterImagePath.value = path
    })()])
    releaseImages()
    if (active() && result.status === "rejected") {
      error.value = result.reason instanceof Error ? result.reason.message : "生成海报失败，请重试"
    }
  } while (regenerate && props.visible && !isDisposed)
  running = false
  loading.value = false
}

/**
 * 文本多行排版计算 (最多 2 行，超长以省略号截断)
 */
function wrapText(
  ctx: any,
  text: string,
  maxWidth: number,
  maxLines = 2
): string[] {
  const lines: string[] = []
  let currentLine = ""

  for (let i = 0; i < text.length; i++) {
    const char = text[i]
    const testLine = currentLine + char
    const metrics = ctx.measureText(testLine)
    if (metrics.width > maxWidth && currentLine.length > 0) {
      lines.push(currentLine)
      currentLine = char
      if (lines.length === maxLines - 1) {
        // 最后一行进行省略号处理
        const remaining = text.substring(i)
        if (ctx.measureText(remaining).width <= maxWidth) {
          lines.push(remaining)
          return lines
        }
        let fitText = ""
        for (let j = 0; j < remaining.length; j++) {
          if (ctx.measureText(fitText + remaining[j] + "...").width <= maxWidth) {
            fitText += remaining[j]
          } else {
            break
          }
        }
        lines.push(fitText + "...")
        return lines
      }
    } else {
      currentLine = testLine
    }
  }

  if (currentLine) {
    lines.push(currentLine)
  }
  return lines.slice(0, maxLines)
}

/**
 * 使用 CanvasContext 绘制海报并在完成后导出图片临时路径
 */
function drawPosterToCanvas(
  product: ProductDetailData,
  mainImgPath: string,
  qrImgPath: string
): Promise<string> {
  return new Promise((resolve, reject) => {
    const scale = dpr.value
    // 创建指定组件实例下的 canvas context
    const ctx = uni.createCanvasContext("productPosterCanvas", instance?.proxy)
    if (!ctx) {
      return reject(new Error("无法初始化画布上下文"))
    }

    // 根据设备像素比缩放画布坐标系
    ctx.setTransform(1, 0, 0, 1, 0, 0)
    ctx.scale(scale, scale)

    // 1. 背景底色：圆角纯白卡片
    ctx.setFillStyle("#ffffff")
    ctx.fillRect(0, 0, canvasWidth, canvasHeight)

    // 2. 顶部商城名称
    const siteName =
      appStore.bootstrapData?.shop?.name ||
      appStore.bootstrapData?.mp?.name ||
      appStore.bootstrapData?.site?.title ||
      "商城好物精选"
    ctx.setFontSize(14)
    ctx.setFillStyle("#333333")
    ctx.fillText(siteName, 20, 30)

    ctx.setFontSize(11)
    ctx.setFillStyle("#999999")
    ctx.fillText("精选好物", 310, 30)

    // 3. 商品主图绘制 (335 × 335 矩形)
    const imgX = 20
    const imgY = 44
    const imgSize = 335

    if (mainImgPath) {
      ctx.drawImage(mainImgPath, imgX, imgY, imgSize, imgSize)
    } else {
      // 缺图占位
      ctx.setFillStyle("#f3f4f6")
      ctx.fillRect(imgX, imgY, imgSize, imgSize)
      ctx.setFontSize(14)
      ctx.setFillStyle("#9ca3af")
      ctx.fillText("暂无商品图片", imgX + 120, imgY + 175)
    }

    // 4. 商品名称（最多 2 行，超长省略）
    ctx.setFontSize(16)
    ctx.setFillStyle("#111827")
    const titleLines = wrapText(ctx, product.name, 335, 2)
    titleLines.forEach((line, idx) => {
      ctx.fillText(line, 20, 404 + idx * 22)
    })

    // 5. 价格与划线价
    const priceY = 460
    ctx.setFontSize(14)
    ctx.setFillStyle("#e02e24")
    ctx.fillText("¥", 20, priceY)

    ctx.setFontSize(22)
    ctx.setFillStyle("#e02e24")
    const priceStr = String(product.price || "0.00")
    ctx.fillText(priceStr, 34, priceY)

    // 6. 分割线
    ctx.setStrokeStyle("#f3f4f6")
    ctx.setLineWidth(1)
    ctx.beginPath()
    ctx.moveTo(20, 478)
    ctx.lineTo(355, 478)
    ctx.stroke()

    // 7. 底部小程序码与引导文案
    const qrSize = 72
    const qrX = 20
    const qrY = 492
    if (qrImgPath) {
      ctx.drawImage(qrImgPath, qrX, qrY, qrSize, qrSize)
    }

    ctx.setFontSize(13)
    ctx.setFillStyle("#1f2937")
    ctx.fillText("长按识别小程序码", 104, 524)

    ctx.setFontSize(11)
    ctx.setFillStyle("#9ca3af")
    ctx.fillText("扫码直达商品详情页选购", 104, 544)

    // 8. 提交绘制并导出临时文件
    ctx.draw(false, () => {
      setTimeout(() => {
        uni.canvasToTempFilePath(
          {
            canvasId: "productPosterCanvas",
            width: canvasWidth * scale,
            height: canvasHeight * scale,
            destWidth: canvasWidth * scale,
            destHeight: canvasHeight * scale,
            fileType: "png",
            quality: 1,
            success(res) {
              if (res.tempFilePath) {
                resolve(res.tempFilePath)
              } else {
                reject(new Error("导出海报文件为空"))
              }
            },
            fail(err) {
              reject(new Error(err?.errMsg || "导出海报图片失败"))
            },
          },
          instance?.proxy
        )
      }, 150)
    })
  })
}

/**
 * 微信端保存海报到手机相册
 */
function handleSaveToAlbum() {
  if (!posterImagePath.value) return
  saving.value = true

  uni.saveImageToPhotosAlbum({
    filePath: posterImagePath.value,
    success() {
      saving.value = false
      uni.showToast({ title: "已保存至手机相册", icon: "success" })
    },
    fail(err) {
      saving.value = false
      const errMsg = err?.errMsg || ""
      if (errMsg.includes("auth deny") || errMsg.includes("auth denied")) {
        uni.showModal({
          title: "提示",
          content: "保存海报需要相册写入权限，请前往授权设置开启",
          confirmText: "去开启",
          success(modalRes) {
            if (modalRes.confirm) {
              uni.openSetting({})
            }
          },
        })
      } else if (!errMsg.includes("cancel")) {
        uni.showToast({ title: "保存相册失败", icon: "none" })
      }
    },
  })
}

/**
 * H5 端保存海报
 */
function handleH5Save() {
  if (!posterImagePath.value) return
  // #ifdef H5
  const a = document.createElement("a")
  a.href = posterImagePath.value
  a.download = `product_poster_${props.productId}.png`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  uni.showToast({ title: "已触发下载，也可长按图片保存", icon: "none" })
  // #endif
}
</script>

<style scoped>
.poster-modal-mask {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 24rpx;
  box-sizing: border-box;
}

.poster-modal-content {
  width: 100%;
  max-width: 640rpx;
  background-color: #ffffff;
  border-radius: 24rpx;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  box-shadow: 0 16rpx 48rpx rgba(0, 0, 0, 0.2);
}

.poster-modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 24rpx 32rpx;
  border-bottom: 1rpx solid #f3f4f6;
}

.header-title {
  font-size: 30rpx;
  font-weight: 600;
  color: #1f2937;
}

.close-icon {
  font-size: 32rpx;
  color: #9ca3af;
  padding: 8rpx;
}

.poster-state-box {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 480rpx;
  padding: 48rpx 32rpx;
}

.loading-spinner {
  font-size: 64rpx;
  margin-bottom: 24rpx;
}

.loading-text {
  font-size: 26rpx;
  color: #6b7280;
}

.error-icon {
  font-size: 64rpx;
  margin-bottom: 20rpx;
}

.error-text {
  font-size: 26rpx;
  color: #ef4444;
  text-align: center;
  margin-bottom: 32rpx;
}

.retry-btn {
  font-size: 26rpx;
  color: #ffffff;
  background-color: #e02e24;
  border-radius: 40rpx;
  padding: 12rpx 48rpx;
  line-height: 1.6;
}

.poster-preview-box {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 24rpx 32rpx 32rpx;
}

.poster-preview-img {
  width: 100%;
  max-height: 720rpx;
  border-radius: 16rpx;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.08);
  margin-bottom: 24rpx;
}

.poster-action-bar {
  display: flex;
  width: 100%;
  gap: 20rpx;
  margin-bottom: 16rpx;
}

.save-btn {
  flex: 1;
  font-size: 28rpx;
  font-weight: 500;
  border-radius: 44rpx;
  line-height: 80rpx;
  text-align: center;
}

.primary-btn {
  background-color: #e02e24;
  color: #ffffff;
}

.cancel-btn {
  background-color: #f3f4f6;
  color: #4b5563;
}

.poster-bottom-tip {
  font-size: 20rpx;
  color: #9ca3af;
  text-align: center;
}

.canvas-offscreen-wrap {
  position: fixed;
  left: -9999px;
  top: -9999px;
  opacity: 0;
  pointer-events: none;
}

.offscreen-canvas {
  display: block;
}
</style>
