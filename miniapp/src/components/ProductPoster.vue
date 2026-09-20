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
import { ref, computed, watch, getCurrentInstance, onBeforeUnmount } from "vue"
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
const dpr = computed(() => {
  try {
    const info = uni.getSystemInfoSync()
    return Math.min(Math.max(info.pixelRatio || 2, 1), 3)
  } catch {
    return 2
  }
})

const loading = ref<boolean>(false)
const loadingTip = ref<string>("正在准备海报...")
const error = ref<string | null>(null)
const saving = ref<boolean>(false)
const posterImagePath = ref<string>("")

let currentSessionSeq = 0
let isDisposed = false

// 监听弹层可见性变化
watch(
  () => props.visible,
  (val) => {
    if (val) {
      if (!posterImagePath.value) {
        handleGeneratePoster()
      }
    } else {
      // 关闭时自增序号丢弃尚未完成的异步绘制
      currentSessionSeq++
    }
  },
  { immediate: true }
)

// 监听商品 ID 变化，商品切换时清空旧海报并重新生成
watch(
  () => props.productId,
  (newId, oldId) => {
    if (newId !== oldId) {
      posterImagePath.value = ""
      error.value = null
      if (props.visible) {
        handleGeneratePoster()
      }
    }
  }
)

onBeforeUnmount(() => {
  isDisposed = true
  currentSessionSeq++
})

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
function downloadImageFile(url: string, timeoutMs = 12000): Promise<string> {
  return new Promise((resolve, reject) => {
    const resolvedUrl = resolveImageUrl(url)
    if (!resolvedUrl) {
      return reject(new Error("图片地址为空"))
    }

    // #ifdef H5
    // H5 环境直接返回图片链接供 Canvas/Image 渲染
    return resolve(resolvedUrl)
    // #endif

    // #ifndef H5
    let isCompleted = false
    const timer = setTimeout(() => {
      if (!isCompleted) {
        isCompleted = true
        reject(new Error("下载图片超时"))
      }
    }, timeoutMs)

    uni.downloadFile({
      url: resolvedUrl,
      success(res) {
        if (isCompleted) return
        isCompleted = true
        clearTimeout(timer)
        if (res.statusCode === 200 && res.tempFilePath) {
          resolve(res.tempFilePath)
        } else {
          reject(new Error(`下载图片失败 (HTTP ${res.statusCode})`))
        }
      },
      fail(err) {
        if (isCompleted) return
        isCompleted = true
        clearTimeout(timer)
        reject(new Error(err?.errMsg || "下载图片失败"))
      },
    })
    // #endif
  })
}

/**
 * 核心流程：获取商品信息、小程序码并执行 Canvas 绘制
 */
async function handleGeneratePoster() {
  if (isDisposed) return
  if (!props.productId || Number(props.productId) <= 0) {
    error.value = "商品信息无效"
    return
  }

  if (!checkAuth()) {
    return
  }

  const session = ++currentSessionSeq
  loading.value = true
  error.value = null
  loadingTip.value = "正在加载商品与分享码..."

  try {
    // 1. 获取商品最新信息（如未提供已加载数据）
    let prodData: ProductDetailData | null = props.product
    if (!prodData || prodData.id !== Number(props.productId)) {
      const res = await getProductDetail(props.productId)
      if (session !== currentSessionSeq || isDisposed) return
      if (res && res.data) {
        prodData = res.data
      } else {
        throw new Error("获取商品信息失败")
      }
    }

    // 2. 获取商品小程序码 (需会员登录)
    loadingTip.value = "正在生成小程序码..."
    const shareCodeRes = await getProductShareCode(props.productId)
    if (session !== currentSessionSeq || isDisposed) return

    if (!shareCodeRes || !shareCodeRes.data || !shareCodeRes.data.url) {
      throw new Error("生成小程序码失败")
    }

    const qrCodeUrl = shareCodeRes.data.url
    const mainImageUrl = prodData.main_image_url || ""

    // 3. 并行下载商品主图和小程序码
    loadingTip.value = "正在下载图片素材..."
    let localMainImg = ""
    let localQrImg = ""

    const downloadTasks: Promise<any>[] = []

    if (mainImageUrl) {
      downloadTasks.push(
        downloadImageFile(mainImageUrl).then((path) => {
          localMainImg = path
        }).catch((err) => {
          console.warn("主图下载失败，使用占位图:", err)
        })
      )
    }

    downloadTasks.push(
      downloadImageFile(qrCodeUrl).then((path) => {
        localQrImg = path
      })
    )

    await Promise.all(downloadTasks)
    if (session !== currentSessionSeq || isDisposed) return

    if (!localQrImg) {
      throw new Error("下载小程序码素材失败")
    }

    // 4. 执行 Canvas 绘制
    loadingTip.value = "正在绘制海报..."
    const exportedFilePath = await drawPosterToCanvas(prodData, localMainImg, localQrImg)
    if (session !== currentSessionSeq || isDisposed) return

    posterImagePath.value = exportedFilePath
    loading.value = false
  } catch (err: any) {
    if (session !== currentSessionSeq || isDisposed) return
    loading.value = false
    error.value = err?.message || "生成海报失败，请稍后重试"
  }
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
            destWidth: canvasWidth * scale * 2,
            destHeight: canvasHeight * scale * 2,
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
