# Skeleton 与 PageState 公共组件使用规范

## 1. Skeleton.vue (骨架屏)

用于在页面或模块初次数据加载期间提供占位结构，匹配实际内容尺寸，减少布局抖动。

### 属性 (Props)

| 属性名 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `type` | `"rect" \| "circle" \| "text"` | `"rect"` | 骨架形状，circle默认50%圆角，text默认28rpx高 |
| `width` | `string \| number` | `"100%"` | 宽度，数字自动追加 `rpx` |
| `height` | `string \| number` | 依类型 | 高度，数字自动追加 `rpx` |
| `radius` | `string \| number` | 依类型 | 圆角，数字自动追加 `rpx` |
| `animated` | `boolean` | `true` | 是否开启动画微光效果 (H5 下自动适配 prefers-reduced-motion) |

### 接入示例

```vue
<!-- 商品卡片骨架 -->
<view class="skeleton-card">
  <Skeleton type="rect" width="100%" height="340rpx" radius="16rpx" />
  <view class="skeleton-info" style="padding: 16rpx 0;">
    <Skeleton type="text" width="80%" height="32rpx" style="margin-bottom: 12rpx;" />
    <Skeleton type="text" width="40%" height="24rpx" style="margin-bottom: 16rpx;" />
    <Skeleton type="rect" width="50%" height="36rpx" />
  </view>
</view>
```

---

## 2. PageState.vue (页面三态与重试)

统一承载列表与详情的首次加载、真实空态与异常重试，不吞掉错误，不把失败显示为“暂无”。

### 属性与事件

| 属性/事件 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `state` | `"loading" \| "empty" \| "error" \| "content"` | `"content"` | 当前展示状态 |
| `emptyText` | `string` | `"暂无相关数据"` | 空态主文案 |
| `emptySubText`| `string` | `""` | 空态辅助文案 |
| `emptyActionText`| `string` | `""` | 空态按钮文案 (触发 `empty-action`) |
| `errorText` | `string` | `"网络连接异常，请重试"` | 错误主文案 |
| `errorSubText`| `string` | `""` | 错误辅助文案 |
| `retryText` | `string` | `"点击重试"` | 错误态重试按钮文案 |
| `@retry` | `Event` | - | 点击重试按钮时触发 |
| `@empty-action` | `Event` | - | 点击空态按钮时触发 |

### 接入示例

```vue
<PageState
  :state="pageState"
  :error-text="firstErrorMsg"
  empty-text="暂无相关订单"
  empty-action-text="去逛逛"
  @retry="loadData(true)"
  @empty-action="goToHome"
>
  <template #loading>
    <!-- 自定义首屏骨架结构 -->
    <MyPageSkeleton />
  </template>

  <!-- 实际列表内容 -->
  <view class="data-list">
    <view v-for="item in list" :key="item.id">...</view>
  </view>
</PageState>
```

---

## 3. NetworkBanner.vue & useNetwork (弱网与网络恢复)

在页面顶端提示网络断开，并支持在网络恢复时仅对当前**前台可见页面**触发只读刷新，严禁重放写请求。

### 接入示例

```vue
<template>
  <view class="page-container">
    <NetworkBanner @retry="loadData(true)" />
    <!-- 业务视图 -->
  </view>
</template>

<script setup lang="ts">
import NetworkBanner from "@/components/NetworkBanner.vue"
import { useNetwork } from "@/composables/useNetwork"

// 仅在当前页面处于前台且网络由断到连时触发只读刷新
useNetwork({
  onOnlineRefresh: () => {
    return loadData(true)
  }
})
</script>
```
