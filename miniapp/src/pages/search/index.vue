<template>
  <view class="search-page-container">
    <NetworkBanner />
    <!-- 顶部固定搜索栏 -->
    <view class="search-header">
      <view class="search-input-bar">
        <view class="search-input-box">
          <text class="search-icon">🔍</text>
          <input
            v-model="keywordInput"
            class="search-input"
            type="text"
            confirm-type="search"
            maxlength="50"
            placeholder="搜索商品名称、副标题"
            @confirm="handleConfirmSearch"
          />
          <text v-if="keywordInput" class="clear-input-btn" @tap="handleClearInput">✕</text>
        </view>
        <button class="search-submit-btn" @tap="handleConfirmSearch">搜索</button>
      </view>

      <!-- 分类筛选标签（如果从分类页传入 category_id，清晰展示且可显式取消） -->
      <view v-if="selectedCategoryId > 0" class="category-filter-bar">
        <view class="category-filter-chip">
          <text class="category-chip-label">分类: {{ currentCategoryName }}</text>
          <text class="category-chip-close" @tap="handleClearCategory">✕</text>
        </view>
        <text class="category-filter-tip">在当前分类下搜索</text>
      </view>
    </view>

    <!-- 发现模式：未提交搜索词时展示历史搜索与热门搜索 -->
    <view v-if="!submittedKeyword" class="discover-container">
      <!-- 搜索历史 -->
      <view v-if="historyList.length > 0" class="tag-section">
        <view class="section-title-row">
          <text class="section-title">搜索历史</text>
          <text class="clear-history-btn" @tap="handleConfirmClearHistory">清空</text>
        </view>
        <view class="tag-list">
          <view
            v-for="(item, index) in historyList"
            :key="index"
            class="tag-item"
            @tap="handleTagClick(item)"
          >
            <text class="tag-text">{{ item }}</text>
            <text class="tag-delete-btn" @tap.stop="handleDeleteHistoryItem(item)">✕</text>
          </view>
        </view>
      </view>

      <!-- 热门搜索（后台配置，支持启停和权重排序） -->
      <view v-if="hotKeywords.length > 0" class="tag-section">
        <view class="section-title-row">
          <text class="section-title">热门搜索</text>
        </view>
        <view class="tag-list">
          <view
            v-for="(word, index) in hotKeywords"
            :key="index"
            class="tag-item hot-tag"
            @tap="handleTagClick(word)"
          >
            <text class="tag-text">{{ word }}</text>
          </view>
        </view>
      </view>

      <!-- 历史与热门词皆为空时的提示 -->
      <view v-if="historyList.length === 0 && hotKeywords.length === 0" class="discover-empty">
        <text class="empty-icon">🔍</text>
        <text class="empty-text">输入关键词开始搜索心仪好物</text>
      </view>
    </view>

    <!-- 结果模式：已提交搜索词时展示排序栏与商品瀑布流 -->
    <view v-else class="result-container">
      <!-- 排序与结果信息栏 -->
      <view class="toolbar-wrapper">
        <view class="sort-tabs">
          <view
            class="sort-tab-item"
            :class="{ active: currentSort === 'default' }"
            @tap="changeSort('default')"
          >
            <text class="tab-label">综合</text>
          </view>

          <view
            class="sort-tab-item"
            :class="{ active: currentSort === 'sales' }"
            @tap="changeSort('sales')"
          >
            <text class="tab-label">销量</text>
          </view>

          <view
            class="sort-tab-item"
            :class="{ active: currentSort === 'price_asc' || currentSort === 'price_desc' }"
            @tap="togglePriceSort"
          >
            <text class="tab-label">价格</text>
            <view class="price-arrows">
              <text
                class="arrow-up"
                :class="{ active: currentSort === 'price_asc' }"
              >▲</text>
              <text
                class="arrow-down"
                :class="{ active: currentSort === 'price_desc' }"
              >▼</text>
            </view>
          </view>
        </view>

        <view class="result-summary">
          <text class="summary-text">{{ effectiveKeyword ? `“${effectiveKeyword}” ` : "" }}共 {{ total }} 件商品</text>
        </view>
      </view>

      <!-- 正在首次加载 -->
      <view v-if="firstLoading" class="state-container loading-state">
        <text class="state-icon">⏳</text>
        <text class="state-text">正在搜索商品...</text>
      </view>

      <!-- 首次加载失败 -->
      <view v-else-if="firstError" class="state-container error-state">
        <text class="state-icon">⚠️</text>
        <text class="state-text">{{ firstError }}</text>
        <button class="retry-btn" @tap="retrySearch">重新加载</button>
      </view>

      <!-- 搜索结果为空 -->
      <view
        v-else-if="productList.length === 0"
        class="state-container empty-state"
      >
        <text class="state-icon">📦</text>
        <text class="state-title">未找到相关商品</text>
        <text class="state-desc">建议检查关键词输入是否正确，或尝试更换其他搜索词</text>
      </view>

      <!-- 商品列表（两列瀑布流卡片） -->
      <view v-else class="product-list-wrapper">
        <view class="product-grid">
          <view
            v-for="item in productList"
            :key="item.id"
            class="grid-item"
          >
            <ProductCard :item="item" layout="grid" />
          </view>
        </view>

        <!-- 分页状态提示 -->
        <view class="pagination-footer">
          <text v-if="loadingMore" class="footer-loading">正在加载更多商品...</text>
          <view v-else-if="moreError" class="footer-error" @tap="handleRetryNextPage">
            <text class="error-msg">{{ moreError }}，点击重试</text>
          </view>
          <text v-else-if="!hasMore" class="footer-nomore">没有更多了</text>
        </view>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref, computed } from "vue"
import { onLoad, onShow, onUnload, onReachBottom } from "@dcloudio/uni-app"
import { storeToRefs } from "pinia"
import { useMemberStore } from "@/stores/member"
import { useAppStore } from "@/stores/app"
import {
  getProductList,
  type ProductListItem,
  type ProductSortType,
} from "@/api/product"
import { getSearchConfig } from "@/api/search"
import ProductCard from "@/components/ProductCard.vue"
import NetworkBanner from "@/components/NetworkBanner.vue"
import { useNetwork } from "@/composables/useNetwork"
import {
  normalizeKeyword,
  getSearchHistory,
  addSearchHistory,
  removeSearchHistoryItem,
  clearSearchHistory,
} from "@/utils/search-history"

const memberStore = useMemberStore()
const appStore = useAppStore()
const { bootstrapData } = storeToRefs(appStore)

// 搜索输入与当前生效提交词（输入与提交状态分离）
const keywordInput = ref<string>("")
const submittedKeyword = ref<string>("")
const effectiveKeyword = ref<string>("")
let isUnmounted = false

// 筛选分类 ID
const selectedCategoryId = ref<number>(0)

// 排序方式
const currentSort = ref<ProductSortType>("default")

// 本地搜索历史与后台热门词
const historyList = ref<string[]>([])
const hotKeywords = ref<string[]>([])

// 列表与分页状态
const productList = ref<ProductListItem[]>([])
const page = ref<number>(1)
const pageSize = 20
const total = ref<number>(0)
const hasMore = ref<boolean>(true)

// 加载状态与错误隔离
const firstLoading = ref<boolean>(false)
const firstError = ref<string>("")
const loadingMore = ref<boolean>(false)
const moreError = ref<string>("")

// 请求并发与竞态控制序号
let requestSeq = 0

// 获取当前分类展示名称
const currentCategoryName = computed<string>(() => {
  if (selectedCategoryId.value <= 0) return ""
  const cats = bootstrapData.value?.categories || []
  const found = cats.find((c) => c.id === selectedCategoryId.value)
  return found ? found.name : "指定分类"
})

// 获取当前会员 ID（若未登录则返回 undefined 使用游客 key）
function getActiveMemberId(): number | undefined {
  return memberStore.isLoggedIn && memberStore.member?.id ? memberStore.member.id : undefined
}

// 刷新本地历史记录
function reloadHistory(): void {
  historyList.value = getSearchHistory(getActiveMemberId())
}

// 加载后台配置的热门搜索词
async function fetchHotKeywords(): Promise<void> {
  try {
    const res = await getSearchConfig()
    if (res && res.code === 1 && res.data && Array.isArray(res.data.hot_keywords)) {
      hotKeywords.value = res.data.hot_keywords
    }
  } catch (err) {
    // 降级为空，不阻断正常搜索体验
    hotKeywords.value = []
  }
}

useNetwork(() => {
  if (submittedKeyword.value && !isUnmounted) {
    void loadProducts(false)
  }
})

onUnload(() => {
  ++requestSeq
  isUnmounted = true
})

// 核心商品数据拉取方法
async function loadProducts(isNextPage = false): Promise<void> {
  const normKw = normalizeKeyword(submittedKeyword.value)
  if (!normKw) {
    // 空词回到发现态，严禁全商城拉取冒充搜索结果
    ++requestSeq
    productList.value = []
    total.value = 0
    effectiveKeyword.value = ""
    firstLoading.value = false
    firstError.value = ""
    return
  }

  // 先判断是否允许加载，再生成该次请求序号，避免无效调用使有效请求失效
  if (isNextPage) {
    if (loadingMore.value || !hasMore.value || firstLoading.value) return
    loadingMore.value = true
    moreError.value = ""
  } else {
    firstLoading.value = true
    firstError.value = ""
    page.value = 1
    hasMore.value = true
  }

  const currentSeq = ++requestSeq
  const queryPage = isNextPage ? page.value + 1 : 1

  try {
    const res = await getProductList({
      keyword: normKw,
      category_id: selectedCategoryId.value > 0 ? selectedCategoryId.value : undefined,
      sort: currentSort.value,
      page: queryPage,
      page_size: pageSize,
    })

    // 请求竞态防乱序：旧请求晚于新请求返回时直接丢弃
    if (currentSeq !== requestSeq || isUnmounted) {
      return
    }

    if (res && res.code === 1) {
      const items = res.data || []
      const returnedTotal = typeof res.total === "number" ? res.total : items.length

      if (isNextPage) {
        // 分页成功推进页码
        page.value = queryPage
        // 按商品 ID 去重追加
        const existingIds = new Set(productList.value.map((p) => p.id))
        const uniqueItems = items.filter((p) => !existingIds.has(p.id))
        productList.value.push(...uniqueItems)
      } else {
        page.value = 1
        productList.value = items
        total.value = returnedTotal
        effectiveKeyword.value = normKw
      }

      hasMore.value = productList.value.length < returnedTotal && items.length >= pageSize
    } else {
      throw new Error(res?.message || "搜索商品失败")
    }
  } catch (err: unknown) {
    if (currentSeq !== requestSeq || isUnmounted) return

    const msg = err instanceof Error ? err.message : "网络连接异常，请重试"
    if (isNextPage) {
      // 翻页失败：页码不推进，保留旧列表，提示错误支持原页重试
      moreError.value = msg
    } else {
      // 首次加载失败：保留或清空并提示重试
      firstError.value = msg
    }
  } finally {
    if (currentSeq === requestSeq) {
      firstLoading.value = false
      loadingMore.value = false
    }
  }
}

// 提交搜索词（触发检索、记录历史并切换至结果态）
function executeSearch(keyword: string): void {
  const normalized = normalizeKeyword(keyword)
  if (!normalized) {
    // 清空回到发现模式，使之前的请求失效
    ++requestSeq
    keywordInput.value = ""
    submittedKeyword.value = ""
    effectiveKeyword.value = ""
    productList.value = []
    total.value = 0
    firstLoading.value = false
    return
  }

  // 记录本地搜索历史（仅明确提交有效词时记录）
  historyList.value = addSearchHistory(normalized, getActiveMemberId())

  keywordInput.value = normalized
  submittedKeyword.value = normalized
  loadProducts(false)
}

// 点击键盘“搜索”或页面“搜索”按钮
function handleConfirmSearch(): void {
  executeSearch(keywordInput.value)
}

// 点击历史标签或热门词标签
function handleTagClick(keyword: string): void {
  executeSearch(keyword)
}

// 清空输入框内容
function handleClearInput(): void {
  keywordInput.value = ""
  if (submittedKeyword.value) {
    // 同步清空已提交词，使旧请求失效，返回发现模式
    ++requestSeq
    submittedKeyword.value = ""
    effectiveKeyword.value = ""
    productList.value = []
    total.value = 0
    firstLoading.value = false
  }
}

// 取消选中的分类
function handleClearCategory(): void {
  selectedCategoryId.value = 0
  if (submittedKeyword.value) {
    // 已处于结果态时，取消分类后立即在全商城重新检索
    loadProducts(false)
  }
}

// 删除单条搜索历史
function handleDeleteHistoryItem(keyword: string): void {
  historyList.value = removeSearchHistoryItem(keyword, getActiveMemberId())
}

// 清空全部搜索历史
function handleConfirmClearHistory(): void {
  uni.showModal({
    title: "清空确认",
    content: "确定要清空全部搜索历史记录吗？",
    confirmText: "清空",
    confirmColor: "#e02e24",
    success: (res) => {
      if (res.confirm) {
        clearSearchHistory(getActiveMemberId())
        reloadHistory()
      }
    },
  })
}

// 切换排序方式
function changeSort(sort: ProductSortType): void {
  if (currentSort.value === sort) return
  currentSort.value = sort
  loadProducts(false)
}

// 价格排序升序/降序切换
function togglePriceSort(): void {
  if (currentSort.value === "price_asc") {
    currentSort.value = "price_desc"
  } else {
    currentSort.value = "price_asc"
  }
  loadProducts(false)
}

// 首次失败重新加载
function retrySearch(): void {
  loadProducts(false)
}

// 分页失败重试
function handleRetryNextPage(): void {
  loadProducts(true)
}

// 触底加载下一页
onReachBottom(() => {
  if (submittedKeyword.value && hasMore.value && !loadingMore.value && !firstLoading.value) {
    loadProducts(true)
  }
})

// 页面加载生命周期
onLoad((query?: Record<string, string>) => {
  if (query) {
    if (query.category_id) {
      const cid = Number(query.category_id)
      if (!Number.isNaN(cid) && cid > 0) {
        selectedCategoryId.value = cid
      }
    }
    if (query.keyword) {
      keywordInput.value = query.keyword
      executeSearch(query.keyword)
    }
  }

  // 异步获取热门词
  fetchHotKeywords()
})

// 页面每次显示时同步历史与可能的登录态切换
onShow(() => {
  reloadHistory()
  if (!appStore.bootstrapData) {
    appStore.fetchBootstrap()
  }
})
</script>

<style scoped>
.search-page-container {
  min-height: 100vh;
  background-color: #f7f8fa;
  padding-bottom: env(safe-area-inset-bottom);
  box-sizing: border-box;
}

/* 顶部搜索固定栏 */
.search-header {
  position: sticky;
  top: 0;
  z-index: 100;
  background-color: #ffffff;
  padding: 16rpx 24rpx;
  border-bottom: 1rpx solid #ebeef5;
}

.search-input-bar {
  display: flex;
  align-items: center;
  gap: 16rpx;
}

.search-input-box {
  flex: 1;
  display: flex;
  align-items: center;
  background-color: #f2f3f5;
  border-radius: 36rpx;
  padding: 12rpx 20rpx;
  gap: 12rpx;
}

.search-icon {
  font-size: 28rpx;
  color: #909399;
}

.search-input {
  flex: 1;
  font-size: 28rpx;
  color: #303133;
  height: 48rpx;
  line-height: 48rpx;
}

.clear-input-btn {
  font-size: 26rpx;
  color: #c0c4cc;
  padding: 4rpx 10rpx;
}

.search-submit-btn {
  font-size: 28rpx;
  color: #ffffff;
  background-color: #e02e24;
  border-radius: 32rpx;
  padding: 0 28rpx;
  height: 64rpx;
  line-height: 64rpx;
  margin: 0;
}

.search-submit-btn::after {
  border: none;
}

/* 分类筛选 Chip 提示栏 */
.category-filter-bar {
  display: flex;
  align-items: center;
  gap: 16rpx;
  margin-top: 14rpx;
}

.category-filter-chip {
  display: inline-flex;
  align-items: center;
  gap: 8rpx;
  background-color: #fef0f0;
  border: 1rpx solid #fde2e2;
  border-radius: 24rpx;
  padding: 6rpx 16rpx;
}

.category-chip-label {
  font-size: 24rpx;
  color: #e02e24;
}

.category-chip-close {
  font-size: 22rpx;
  color: #e02e24;
  padding: 0 4rpx;
}

.category-filter-tip {
  font-size: 22rpx;
  color: #909399;
}

/* 发现模式 */
.discover-container {
  padding: 24rpx 30rpx;
}

.tag-section {
  margin-bottom: 40rpx;
}

.section-title-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20rpx;
}

.section-title {
  font-size: 30rpx;
  font-weight: 600;
  color: #303133;
}

.clear-history-btn {
  font-size: 24rpx;
  color: #909399;
  padding: 6rpx 12rpx;
}

.tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: 16rpx;
}

.tag-item {
  display: inline-flex;
  align-items: center;
  gap: 8rpx;
  background-color: #ffffff;
  border: 1rpx solid #e4e7ed;
  border-radius: 28rpx;
  padding: 10rpx 24rpx;
}

.hot-tag {
  background-color: #fdf6ec;
  border-color: #faecd8;
}

.hot-tag .tag-text {
  color: #e6a23c;
  font-weight: 500;
}

.tag-text {
  font-size: 26rpx;
  color: #606266;
  max-width: 400rpx;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tag-delete-btn {
  font-size: 20rpx;
  color: #c0c4cc;
  padding: 0 4rpx;
}

.discover-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding-top: 160rpx;
  gap: 16rpx;
}

.discover-empty .empty-icon {
  font-size: 72rpx;
  color: #dcdfe6;
}

.discover-empty .empty-text {
  font-size: 28rpx;
  color: #909399;
}

/* 结果模式 */
.result-container {
  display: flex;
  flex-direction: column;
}

/* 排序工具栏 */
.toolbar-wrapper {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background-color: #ffffff;
  padding: 18rpx 30rpx;
  border-bottom: 1rpx solid #f2f3f5;
}

.sort-tabs {
  display: flex;
  align-items: center;
  gap: 48rpx;
}

.sort-tab-item {
  display: flex;
  align-items: center;
  gap: 6rpx;
  font-size: 28rpx;
  color: #606266;
  position: relative;
  padding: 6rpx 0;
}

.sort-tab-item.active {
  color: #e02e24;
  font-weight: 600;
}

.price-arrows {
  display: flex;
  flex-direction: column;
  font-size: 14rpx;
  line-height: 1;
  gap: 2rpx;
}

.arrow-up,
.arrow-down {
  color: #c0c4cc;
}

.arrow-up.active,
.arrow-down.active {
  color: #e02e24;
}

.result-summary {
  font-size: 24rpx;
  color: #909399;
}

/* 状态展示 */
.state-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 140rpx 40rpx;
  gap: 16rpx;
}

.state-icon {
  font-size: 72rpx;
}

.state-title {
  font-size: 30rpx;
  font-weight: 600;
  color: #303133;
}

.state-text,
.state-desc {
  font-size: 26rpx;
  color: #909399;
  text-align: center;
  line-height: 1.5;
}

.retry-btn {
  margin-top: 16rpx;
  font-size: 26rpx;
  color: #e02e24;
  border: 1rpx solid #e02e24;
  background-color: transparent;
  border-radius: 32rpx;
  padding: 8rpx 32rpx;
  line-height: 1.4;
}

.retry-btn::after {
  border: none;
}

/* 商品两列网格瀑布流 */
.product-list-wrapper {
  padding: 20rpx;
}

.product-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 16rpx;
}

.grid-item {
  width: calc(50% - 8rpx);
  box-sizing: border-box;
}

/* 分页尾部状态 */
.pagination-footer {
  padding: 30rpx 0 40rpx;
  display: flex;
  align-items: center;
  justify-content: center;
}

.footer-loading,
.footer-nomore {
  font-size: 24rpx;
  color: #909399;
}

.footer-error {
  padding: 6rpx 16rpx;
}

.error-msg {
  font-size: 24rpx;
  color: #e02e24;
}
</style>
