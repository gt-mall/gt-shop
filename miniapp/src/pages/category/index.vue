<template>
  <view class="category-container">
    <!-- 顶部搜索入口 (点击跳转独立搜索页) -->
    <view class="search-bar" @tap="goToSearch">
      <view class="search-input-box">
        <text class="search-icon">🔍</text>
        <text class="search-placeholder-text">搜索商品名称、副标题</text>
      </view>
      <text class="search-action-btn">搜索</text>
    </view>

    <!-- 分类主体内容 -->
    <view class="category-body">
      <!-- 左侧分类侧边栏 -->
      <scroll-view scroll-y class="sidebar">
        <view
          v-for="cat in categoryList"
          :key="cat.id"
          class="sidebar-item"
          :class="{ active: selectedCategoryId === cat.id }"
          @tap="onSelectCategory(cat.id)"
        >
          <text class="item-text">{{ cat.name }}</text>
        </view>
      </scroll-view>

      <!-- 右侧商品区 -->
      <view class="right-pane">
        <!-- 排序工具栏 -->
        <view class="sort-bar">
          <view
            class="sort-tab"
            :class="{ active: currentSort === 'default' }"
            @tap="handleSortChange('default')"
          >
            <text class="sort-text">综合</text>
          </view>
          <view
            class="sort-tab"
            :class="{ active: currentSort === 'sales' }"
            @tap="handleSortChange('sales')"
          >
            <text class="sort-text">销量</text>
          </view>
          <view
            class="sort-tab price-tab"
            :class="{ active: currentSort === 'price_asc' || currentSort === 'price_desc' }"
            @tap="handleTogglePriceSort"
          >
            <text class="sort-text">价格</text>
            <view class="sort-arrows">
              <text class="arrow" :class="{ highlight: currentSort === 'price_asc' }">▲</text>
              <text class="arrow" :class="{ highlight: currentSort === 'price_desc' }">▼</text>
            </view>
          </view>
        </view>

        <!-- 商品滚动列表 -->
        <scroll-view
          scroll-y
          class="product-scroll"
          @scrolltolower="handleScrollToLower"
        >
          <!-- 首次加载状态 -->
          <view v-if="loading && productList.length === 0" class="state-box">
            <text class="state-text">正在加载商品...</text>
          </view>

          <!-- 加载失败状态 -->
          <view v-else-if="error && productList.length === 0" class="state-box">
            <text class="error-text">{{ error }}</text>
            <button class="retry-btn" @tap="loadProducts(false)">点击重试</button>
          </view>

          <!-- 空数据状态 -->
          <view v-else-if="productList.length === 0" class="state-box">
            <text class="state-text">暂无相关商品</text>
          </view>

          <!-- 真实商品列表 -->
          <view v-else class="product-list-wrapper">
            <ProductCard
              v-for="item in productList"
              :key="item.id"
              :item="item"
              layout="row"
            />

            <!-- 底部加载更多状态 -->
            <view class="load-more-bar">
              <text v-if="loadingMore" class="loading-more-text">加载中...</text>
              <view v-else-if="moreError" class="more-error-box" @tap="loadProducts(true)">
                <text class="more-error-text">{{ moreError }}，点击重试</text>
              </view>
              <text v-else-if="!hasMore" class="no-more-text">没有更多了</text>
            </view>
          </view>
        </scroll-view>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref, computed } from "vue"
import { onShow } from "@dcloudio/uni-app"
import { storeToRefs } from "pinia"
import { useAppStore } from "@/stores/app"
import {
  getProductList,
  type ProductListItem,
  type ProductSortType,
} from "@/api/product"
import type { BootstrapCategory } from "@/api/common"
import ProductCard from "@/components/ProductCard.vue"

const appStore = useAppStore()
const { bootstrapData } = storeToRefs(appStore)
const { fetchBootstrap } = appStore

// 搜索输入与当前生效的搜索词
const keywordInput = ref<string>("")
const activeKeyword = ref<string>("")

// 当前选中的分类 ID (0 代表全部)
const selectedCategoryId = ref<number>(0)

// 当前排序方式 (default, sales, price_asc, price_desc)
const currentSort = ref<ProductSortType>("default")

// 分页数据
const page = ref<number>(1)
const pageSize = 20
const total = ref<number>(0)
const hasMore = ref<boolean>(true)

const productList = ref<ProductListItem[]>([])
const loading = ref<boolean>(false)
const loadingMore = ref<boolean>(false)
const error = ref<string>("")
const moreError = ref<string>("")

// 分类列表
const categoryList = computed<BootstrapCategory[]>(() => {
  const cats = bootstrapData.value?.categories
  if (cats && cats.length > 0) {
    return cats
  }
  return [
    { id: 1, name: "数码电子", image_url: "", parent_id: 0, sort: 1 },
    { id: 2, name: "服装服饰", image_url: "", parent_id: 0, sort: 2 },
    { id: 3, name: "食品生鲜", image_url: "", parent_id: 0, sort: 3 },
    { id: 4, name: "家居日用", image_url: "", parent_id: 0, sort: 4 },
    { id: 5, name: "美妆个护", image_url: "", parent_id: 0, sort: 5 },
    { id: 6, name: "运动户外", image_url: "", parent_id: 0, sort: 6 },
    { id: 7, name: "母婴玩具", image_url: "", parent_id: 0, sort: 7 },
    { id: 8, name: "图书文具", image_url: "", parent_id: 0, sort: 8 },
  ]
})

// 加载商品列表 (isNextPage: 是否为分页加载更多)
async function loadProducts(isNextPage = false): Promise<void> {
  if (isNextPage) {
    if (loadingMore.value || !hasMore.value || loading.value) return
    loadingMore.value = true
    moreError.value = ""
    page.value += 1
  } else {
    if (loading.value) return
    loading.value = true
    error.value = ""
    page.value = 1
    hasMore.value = true
  }

  try {
    const res = await getProductList({
      page: page.value,
      page_size: pageSize,
      keyword: activeKeyword.value || undefined,
      category_id: selectedCategoryId.value > 0 ? selectedCategoryId.value : undefined,
      sort: currentSort.value,
    })

    const items = res.data || []
    total.value = res.total || 0

    if (isNextPage) {
      productList.value.push(...items)
    } else {
      productList.value = items
    }

    hasMore.value = productList.value.length < total.value
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : "加载商品失败"
    if (isNextPage) {
      page.value -= 1
      moreError.value = msg
    } else {
      error.value = msg
    }
  } finally {
    if (isNextPage) {
      loadingMore.value = false
    } else {
      loading.value = false
    }
  }
}

// 切换分类
function onSelectCategory(catId: number): void {
  if (selectedCategoryId.value === catId) return
  selectedCategoryId.value = catId
  resetAndReload()
}

// 切换排序
function handleSortChange(sort: "default" | "sales"): void {
  if (currentSort.value === sort) return
  currentSort.value = sort
  resetAndReload()
}

// 切换价格排序 (price_asc <-> price_desc)
function handleTogglePriceSort(): void {
  if (currentSort.value === "price_asc") {
    currentSort.value = "price_desc"
  } else {
    currentSort.value = "price_asc"
  }
  resetAndReload()
}

// 跳转独立搜索页（携带当前分类筛选参数）
function goToSearch(): void {
  const cid = selectedCategoryId.value
  const url = cid > 0 ? `/pages/search/index?category_id=${cid}` : "/pages/search/index"
  uni.navigateTo({
    url,
  })
}

// 触底加载更多
function handleScrollToLower(): void {
  if (hasMore.value && !loading.value && !loadingMore.value) {
    loadProducts(true)
  }
}

// 重置分页并重新请求
function resetAndReload(): void {
  page.value = 1
  productList.value = []
  hasMore.value = true
  loadProducts(false)
}

onShow(async () => {
  if (!bootstrapData.value) {
    await fetchBootstrap()
  }

  // 检查是否有从首页金刚区点击指定分类传递的 active_category_id
  let targetCatId: number | null = null
  try {
    const storedId = uni.getStorageSync("active_category_id")
    if (storedId !== undefined && storedId !== null && storedId !== "") {
      uni.removeStorageSync("active_category_id")
      targetCatId = Number(storedId)
    }
  } catch {}

  if (targetCatId !== null && targetCatId > 0) {
    selectedCategoryId.value = targetCatId
    resetAndReload()
  } else if (productList.value.length === 0 && !loading.value) {
    // 若尚未选定分类，默认选中第一个分类
    if (categoryList.value.length > 0 && selectedCategoryId.value === 0) {
      selectedCategoryId.value = categoryList.value[0].id
    }
    resetAndReload()
  }
})
</script>

<style scoped>
.category-container {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background-color: #f7f8fa;
  max-width: 1000rpx;
  margin: 0 auto;
}

/* 顶部搜索栏 */
.search-bar {
  display: flex;
  align-items: center;
  gap: 16rpx;
  padding: 16rpx 24rpx;
  background-color: #ffffff;
  border-bottom: 1rpx solid #ebeef5;
}

.search-input-box {
  flex: 1;
  display: flex;
  align-items: center;
  background-color: #f2f3f5;
  border-radius: 32rpx;
  padding: 10rpx 20rpx;
  gap: 12rpx;
}

.search-icon {
  font-size: 26rpx;
  color: #909399;
}

.search-placeholder-text {
  flex: 1;
  font-size: 26rpx;
  color: #909399;
  height: 48rpx;
  line-height: 48rpx;
}

.search-input {
  flex: 1;
  font-size: 26rpx;
  color: #303133;
  height: 48rpx;
  line-height: 48rpx;
}

.clear-icon {
  font-size: 24rpx;
  color: #c0c4cc;
  padding: 4rpx 10rpx;
}

.search-action-btn {
  font-size: 26rpx;
  color: #e02e24;
  background: none;
  padding: 0 10rpx;
  height: 48rpx;
  line-height: 48rpx;
  margin: 0;
}

.search-action-btn::after {
  border: none;
}

/* 分类主体 */
.category-body {
  display: flex;
  flex: 1;
  overflow: hidden;
}

/* 左侧分类栏 */
.sidebar {
  width: 180rpx;
  background-color: #f5f6f8;
  height: 100%;
}

.sidebar-item {
  padding: 30rpx 14rpx;
  text-align: center;
  font-size: 26rpx;
  color: #606266;
  border-left: 6rpx solid transparent;
  transition: all 0.2s;
}

.sidebar-item.active {
  background-color: #ffffff;
  color: #e02e24;
  font-weight: 600;
  border-left-color: #e02e24;
}

.item-text {
  word-break: break-all;
}

/* 右侧商品区 */
.right-pane {
  flex: 1;
  display: flex;
  flex-direction: column;
  height: 100%;
  background-color: #f7f8fa;
  min-width: 0;
}

/* 排序条 */
.sort-bar {
  display: flex;
  background-color: #ffffff;
  border-bottom: 1rpx solid #ebeef5;
  height: 76rpx;
  align-items: center;
}

.sort-tab {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  font-size: 26rpx;
  color: #606266;
  position: relative;
}

.sort-tab.active {
  color: #e02e24;
  font-weight: 600;
}

.price-tab {
  gap: 6rpx;
}

.sort-arrows {
  display: flex;
  flex-direction: column;
  font-size: 14rpx;
  line-height: 10rpx;
  color: #c0c4cc;
}

.sort-arrows .arrow.highlight {
  color: #e02e24;
  font-weight: bold;
}

/* 商品滚动区 */
.product-scroll {
  flex: 1;
  height: 0;
  padding: 16rpx 16rpx 20rpx;
  box-sizing: border-box;
}

.product-list-wrapper {
  display: flex;
  flex-direction: column;
}

/* 状态展示 */
.state-box {
  padding: 100rpx 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 20rpx;
}

.state-text {
  font-size: 26rpx;
  color: #909399;
}

.error-text {
  font-size: 26rpx;
  color: #f56c6c;
}

.retry-btn {
  font-size: 24rpx;
  color: #ffffff;
  background-color: #e02e24;
  border-radius: 30rpx;
  padding: 0 32rpx;
  height: 54rpx;
  line-height: 54rpx;
  margin: 0;
}

.retry-btn::after {
  border: none;
}

/* 底部状态 */
.load-more-bar {
  padding: 24rpx 0 36rpx;
  text-align: center;
}

.loading-more-text,
.no-more-text,
.more-error-text {
  font-size: 24rpx;
  color: #c0c4cc;
}

.more-error-text {
  color: #e02e24;
}
</style>
