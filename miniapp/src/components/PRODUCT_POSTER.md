# ProductPoster 商品分享海报组件

`ProductPoster.vue` 是面向微信小程序与 H5 双端的轻量商品分享海报生成与保存弹层组件。

## 功能特性

1. **真实小程序码与扫码落地**：
   - 依赖后端 `/api/miniapp/products/:id/share-code` 接口获取带有短场景参数 `p=<id>` 的真实小程序码。
   - 扫码后由 `miniapp/src/utils/product-share.ts` 的 `parseProductEntry` 自动解析落地到目标商品。
   - 后端使用持久化映射表 `product_share_code` 与微信 AccessToken 自动续期，高并发安全缓存。

2. **高清晰度 Canvas 动态排版**：
   - 支持根据设备像素比 (DPR, 1x~3x) 自适应缩放画布坐标系与导出分辨率，保证高清文字与清晰小程序码。
   - 标题智能多行折行截断（最多 2 行，超长使用 `...` 省略）。
   - 金额严格沿用元单位高精度字符串排版，不做浮点精度丢失计算。
   - 标注价格实时提示 `* 海报价格仅供参考，实际以商品页面为准`。

3. **异常处理与会话时序保护**：
   - 状态隔离：快速开关或切换商品时，自动丢弃旧异步绘制结果，避免串图或覆盖。
   - 鉴权保护：要求会员登录后生成，未登录时友好提示。
   - 缺图兜底：若商品无主图或网络加载失败，展示优雅灰色占位卡片，不阻断海报生成。
   - 严禁假码：若小程序码生成失败，直接给出重试错误提示，严禁输出不可扫的假二维码。

4. **保存与权限恢复**：
   - 微信小程序端：调用 `uni.saveImageToPhotosAlbum` 保存到手机系统相册；若用户首次拒绝相册权限，弹出引导对话框并通过 `uni.openSetting` 帮助用户恢复授权。
   - H5 端：提供下载链接与长按菜单保存提示。

---

## 属性与事件

### Props

| 属性名 | 类型 | 必填 | 默认值 | 说明 |
| :--- | :--- | :--- | :--- | :--- |
| `productId` | `number \| string` | 是 | - | 目标商品 ID |
| `visible` | `boolean` | 是 | `false` | 弹层是否可见，支持 `v-model:visible` |
| `product` | `ProductDetailData \| null` | 否 | `null` | 已加载的商品详情数据，传入可免去重复请求商品接口 |

### Emits

| 事件名 | 参数 | 说明 |
| :--- | :--- | :--- |
| `update:visible` | `(val: boolean)` | 控制弹层显示隐藏 |
| `close` | - | 弹层关闭事件 |

---

## 接入示例

在商品详情页 `miniapp/src/pages/product/detail.vue` 中接入：

```vue
<template>
  <view class="product-detail">
    <!-- 分享操作区 -->
    <view class="share-actions">
      <!-- 微信原生转发 -->
      <button class="share-btn" open-type="share">
        <text class="icon">📤</text>
        <text>分享</text>
      </button>

      <!-- 打开生成海报弹层 -->
      <button class="poster-btn" @tap="showPoster = true">
        <text class="icon">🖼️</text>
        <text>生成海报</text>
      </button>
    </view>

    <!-- 商品海报组件 -->
    <ProductPoster
      v-if="product"
      :product-id="product.id"
      :product="product"
      v-model:visible="showPoster"
    />
  </view>
</template>

<script setup lang="ts">
import { ref } from "vue"
import ProductPoster from "@/components/ProductPoster.vue"

const product = ref({ id: 123, name: "商品名称", price: "99.00", main_image_url: "..." })
const showPoster = ref<boolean>(false)
</script>
```
