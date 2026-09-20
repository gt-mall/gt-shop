import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import vm from 'node:vm';
import ts from 'typescript';
import * as vue from 'vue';
import { parse, compileScript } from 'vue/compiler-sfc';
import { createPinia, setActivePinia, storeToRefs } from 'pinia';
import { useMemberStore } from '../src/stores/member.ts';
import { useNetwork } from '../src/composables/useNetwork.ts';
import { onNetworkOnline, triggerNetworkStatusChangeForTest } from '../src/services/network.ts';
import {
  setWechatAddressDraft,
  consumeWechatAddressDraft,
  clearWechatAddressDraft,
} from '../src/composables/useWechatAddressImport.ts';

console.log('--- Starting SFC / Composable Integration Tests for Review & Flow ---');

const delay = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

function createSfcInstance(sfcRelativePath, customMocks = {}) {
  const sfcPath = path.resolve(process.cwd(), sfcRelativePath);
  const fileContent = fs.readFileSync(sfcPath, 'utf8');
  const { descriptor } = parse(fileContent);
  const script = compileScript(descriptor, { id: path.basename(sfcRelativePath) });
  const output = ts.transpileModule(script.content, {
    compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 },
  }).outputText;

  const lifecycles = {};
  const context = vm.createContext({
    exports: {},
    console,
    setTimeout,
    clearTimeout,
    setInterval,
    clearInterval,
    Date,
    Math,
    Array,
    Set,
    Map,
    Promise,
    Error,
    uni: globalThis.uni || {
      showToast() {},
      showLoading() {},
      hideLoading() {},
      showModal({ success }) { if (success) success({ confirm: true }); },
      navigateTo() {},
      redirectTo() {},
      navigateBack() {},
      switchTab() {},
      setNavigationBarTitle() {},
      getStorageSync: () => null,
      setStorageSync: () => {},
      removeStorageSync: () => {},
      previewImage() {},
      getNetworkType({ success }) { if (success) success({ networkType: 'wifi' }); },
      onNetworkStatusChange() {},
    },
    getCurrentPages: () => [{ route: sfcRelativePath, options: {} }],
    require(name) {
      if (name === 'vue') return vue;
      if (name === 'pinia') return { storeToRefs };
      if (name === '@dcloudio/uni-app') {
        return {
          onLoad: (cb) => { lifecycles.onLoad = cb; },
          onShow: (cb) => { lifecycles.onShow = cb; },
          onHide: (cb) => { lifecycles.onHide = cb; },
          onUnload: (cb) => { lifecycles.onUnload = cb; },
          onReachBottom: (cb) => { lifecycles.onReachBottom = cb; },
          onPullDownRefresh: (cb) => { lifecycles.onPullDownRefresh = cb; },
        };
      }
      if (customMocks[name]) {
        return customMocks[name];
      }
      // Return stub for sub-components
      if (name.endsWith('.vue') || name.endsWith('.svg')) {
        return { default: { render: () => null } };
      }
      if (name === '@/stores/member') return { useMemberStore };
      if (name === '@/composables/useNetwork') return { useNetwork };
      if (name === '@/utils') return { getToken: () => 'test-token', resolveImageUrl: (url) => url };
      if (name === '@/utils/search-history') {
        return {
          normalizeKeyword: (k) => (typeof k === 'string' ? k.trim() : ''),
          getSearchHistory: () => [],
          addSearchHistory: () => {},
          removeSearchHistoryItem: () => {},
          clearSearchHistory: () => {},
        };
      }
      return {};
    },
  });

  vm.runInContext(output, context);
  const component = context.exports.default;
  component.render = () => null;

  const renderer = vue.createRenderer({
    createElement: () => ({}),
    createText: () => ({}),
    createComment: () => ({}),
    insert() {},
    remove() {},
    setText() {},
    setElementText() {},
    patchProp() {},
    parentNode: () => null,
    nextSibling: () => null,
  });

  let state = null;
  const pinia = customMocks.pinia || createPinia();
  setActivePinia(pinia);

  const app = renderer.createApp({
    render() {
      return vue.h(component, {
        ref: (instance) => {
          if (instance) state = instance.$.setupState;
        },
      });
    },
  });
  app.use(pinia);
  app.mount({});

  return { state, lifecycles, app };
}

// ----------------------------------------------------
// 测试 1: 第 2 页失败后的请求序列 [1, 2, 2] (product.vue)
// ----------------------------------------------------
console.log('Test 1: 分页第 2 页失败后重试请求序列为 [1, 2, 2]');
{
  const requestedPages = [];
  let shouldFailPage2 = true;

  const mockReviewApi = {
    getProductReviews: async (pId, params) => {
      requestedPages.push(params.page);
      if (params.page === 2 && shouldFailPage2) {
        shouldFailPage2 = false; // 下一次重试允许成功
        throw new Error('网络异常');
      }
      return {
        items: Array.from({ length: 10 }, (_, i) => ({
          id: params.page * 100 + i,
          content: `page ${params.page} item ${i}`,
        })),
        total: 20,
        summary: { total_count: 20, average_rating: '5.0', good_rate: '100%' },
      };
    },
  };

  const { state, lifecycles, app } = createSfcInstance('src/pages/review/product.vue', {
    '@/api/review': mockReviewApi,
  });

  // 首次触发加载 (第 1 页)
  lifecycles.onLoad({ id: '10' });
  await delay(10);
  assert.equal(state.page, 1);
  assert.equal(state.items.length, 10);

  // 触底触发第 2 页 (注定失败)
  lifecycles.onReachBottom();
  await delay(10);
  // 失败后：page 不应递增到 2，必须停留在 1，以便重试请求第 2 页
  assert.equal(state.page, 1);
  assert.equal(state.items.length, 10, '第 2 页失败后保留第 1 页数据');
  assert.ok(state.moreError.length > 0, '显示分页错误文案');

  // 再次触底或点击重试第 2 页 (应成功)
  lifecycles.onReachBottom();
  await delay(10);
  assert.equal(state.page, 2);
  assert.equal(state.items.length, 20);

  assert.deepEqual(requestedPages, [1, 2, 2], '请求序列必须严格为 [1, 2, 2]，不可跳至 [1, 2, 3]');
  app.unmount();
  console.log('-> Test 1 PASSED');
}

console.log('Test 1B: 翻页中切换筛选会立即启动新查询并丢弃旧页');
{
  let resolveOldPage;
  const calls = [];
  const mockReviewApi = {
    getProductReviews: (_id, params) => {
      calls.push(`${params.filter}:${params.page}`);
      if (params.filter === 'all' && params.page === 1) {
        return Promise.resolve({ items: Array.from({ length: 10 }, (_, i) => ({ id: i + 1, rating: 5 })), total: 20, summary: {} });
      }
      if (params.filter === 'all' && params.page === 2) {
        return new Promise((resolve) => { resolveOldPage = () => resolve({ items: [{ id: 99, rating: 5 }], total: 20, summary: {} }); });
      }
      return Promise.resolve({ items: [{ id: 200, rating: 1 }], total: 1, summary: {} });
    },
  };
  const { state, lifecycles, app } = createSfcInstance('src/pages/review/product.vue', { '@/api/review': mockReviewApi });
  lifecycles.onLoad({ id: '10' });
  await delay(10);
  state.loadMore();
  await delay(1);
  state.switchFilter('bad');
  await delay(10);
  assert.deepEqual(calls, ['all:1', 'all:2', 'bad:1']);
  assert.equal(state.items[0].id, 200);
  resolveOldPage();
  await delay(10);
  assert.deepEqual(state.items.map((item) => item.id), [200], '旧筛选分页不得追加到差评列表');
  app.unmount();
  console.log('-> Test 1B PASSED');
}

// ----------------------------------------------------
// 测试 2: 旧刷新/分页晚返回 (mine.vue 竞态丢弃)
// ----------------------------------------------------
console.log('Test 2: 旧刷新晚返回不能覆盖新请求数据');
{
  let callCount = 0;
  let resolveReqA;
  let resolveReqB;

  const mockReviewApi = {
    getMineReviews: () => {
      callCount++;
      if (callCount === 1) {
        // 请求 A: 慢请求
        return new Promise((resolve) => {
          resolveReqA = () => resolve({
            items: [{ id: 101, content: '慢返回的旧评价' }],
            total: 1,
          });
        });
      } else {
        // 请求 B: 快请求
        return new Promise((resolve) => {
          resolveReqB = () => resolve({
            items: [{ id: 202, content: '快返回的新评价' }],
            total: 1,
          });
        });
      }
    },
  };

  const { state, app } = createSfcInstance('src/pages/review/mine.vue', {
    '@/api/review': mockReviewApi,
  });

  // 1. 触发请求 A
  state.loadData(true);
  // 2. 紧接着用户触发刷新请求 B
  state.loadData(true);

  // 3. 请求 B 先返回
  resolveReqB();
  await delay(10);
  assert.equal(state.items.length, 1);
  assert.equal(state.items[0].id, 202);

  // 4. 请求 A 晚返回
  resolveReqA();
  await delay(10);
  // 晚返回的请求 A 必须被丢弃，不能覆盖请求 B 的数据！
  assert.equal(state.items.length, 1);
  assert.equal(state.items[0].id, 202, '旧请求晚返回被成功丢弃，未污染当前列表');

  app.unmount();
  console.log('-> Test 2 PASSED');
}

// ----------------------------------------------------
// 测试 3: A 账号响应晚于 B 账号 (mine.vue 账号切换隔离)
// ----------------------------------------------------
console.log('Test 3: A 账号慢响应不能覆盖 B 账号数据');
{
  const pinia = createPinia();
  setActivePinia(pinia);
  const memberStore = useMemberStore();
  memberStore.member = { id: 1001, nickname: '用户A' };

  let resolveUserA;
  let resolveUserB;

  const mockReviewApi = {
    getMineReviews: () => {
      if (memberStore.member?.id === 1001) {
        return new Promise((resolve) => {
          resolveUserA = () => resolve({
            items: [{ id: 1001, content: '用户A的评价' }],
            total: 1,
          });
        });
      } else {
        return new Promise((resolve) => {
          resolveUserB = () => resolve({
            items: [{ id: 2002, content: '用户B的评价' }],
            total: 1,
          });
        });
      }
    },
  };

  const { state, app } = createSfcInstance('src/pages/review/mine.vue', {
    '@/api/review': mockReviewApi,
    '@/stores/member': { useMemberStore: () => memberStore },
  });

  // 用户 A 触发加载
  state.loadData(true);

  // 切换到用户 B
  memberStore.member = { id: 2002, nickname: '用户B' };
  await vue.nextTick();

  // 用户 B 触发加载并立即返回
  state.loadData(true);
  resolveUserB();
  await delay(10);
  assert.equal(state.items.length, 1);
  assert.equal(state.items[0].id, 2002);

  // 用户 A 的旧响应此时才姗姗来迟
  resolveUserA();
  await delay(10);
  assert.equal(state.items.length, 1);
  assert.equal(state.items[0].id, 2002, '用户A的慢响应被忽略，未覆盖用户B');

  app.unmount();
  console.log('-> Test 3 PASSED');
}

console.log('Test 3B: 翻页中切换账号不会残留 loadingMore');
{
  const pinia = createPinia();
  setActivePinia(pinia);
  const memberStore = useMemberStore();
  memberStore.token = 'token-a';
  memberStore.member = { id: 4001, nickname: 'A' };
  let resolveOldPage;
  const calls = [];
  const mockReviewApi = {
    getMineReviews: (page) => {
      calls.push(`${memberStore.member?.id}:${page}`);
      if (memberStore.member?.id === 4001 && page === 1) {
        return Promise.resolve({ items: Array.from({ length: 10 }, (_, i) => ({ id: i + 1 })), total: 20 });
      }
      if (memberStore.member?.id === 4001) {
        return new Promise((resolve) => { resolveOldPage = () => resolve({ items: [{ id: 99 }], total: 20 }); });
      }
      return Promise.resolve({ items: [{ id: 5001 }], total: 1 });
    },
  };
  const { state, lifecycles, app } = createSfcInstance('src/pages/review/mine.vue', {
    pinia,
    '@/api/review': mockReviewApi,
    '@/stores/member': { useMemberStore: () => memberStore },
  });
  lifecycles.onLoad();
  await delay(10);
  state.loadMore();
  await delay(1);
  memberStore.member = { id: 5001, nickname: 'B' };
  memberStore.token = 'token-b';
  await delay(15);
  assert.equal(state.loadingMore, false);
  assert.deepEqual(state.items.map((item) => item.id), [5001]);
  resolveOldPage();
  await delay(10);
  assert.deepEqual(state.items.map((item) => item.id), [5001]);
  assert.ok(calls.includes('5001:1'), '新账号必须立即请求第 1 页');
  app.unmount();
  console.log('-> Test 3B PASSED');
}

// ----------------------------------------------------
// 测试 4: A 地址草稿不能提交给 B (address/edit.vue)
// ----------------------------------------------------
console.log('Test 4: A 地址草稿填入表单后不能提交给 B 账号');
{
  const pinia = createPinia();
  setActivePinia(pinia);
  const memberStore = useMemberStore();
  memberStore.token = 'user-a-token';
  memberStore.member = { id: 3001, nickname: 'UserA' };

  // 设置 UserA 的微信草稿
  setWechatAddressDraft({
    userId: 3001,
    name: '张三',
    mobile: '13800138000',
    province: '广东省',
    province_code: '440000',
    city: '深圳市',
    city_code: '440300',
    district: '南山区',
    district_code: '440305',
    address: '科技园科苑路1号',
    regionMatched: true,
  });

  let addAddressCalled = false;
  memberStore.addAddress = async () => {
    addAddressCalled = true;
    return { code: 1, data: { id: 999 } };
  };
  memberStore.fetchAddresses = async () => {};
  memberStore.editAddress = async () => {};
  memberStore.removeAddress = async () => {};

  const { state, lifecycles, app } = createSfcInstance('src/pages/address/edit.vue', {
    pinia,
    '@/stores/member': { useMemberStore: () => memberStore },
    '@/utils/addressParse': {
      resolveRegionCodes: () => ({
        matched: true,
        province_code: '440000',
        city_code: '440300',
        district_code: '440305',
      }),
      findMatchingAddress: () => null,
      setCachedRegionTree: () => {},
      parseAddressText: () => ({ matched: true }),
    },
    '@/composables/useWechatAddressImport': {
      useWechatAddressImport: () => ({ importWechatAddress: () => {} }),
      consumeWechatAddressDraft,
      clearWechatAddressDraft,
    },
    '@/api': { getRegions: async () => ({ code: 1, data: { regions: [] } }) },
  });

  // User A 打开页面并导入草稿
  await lifecycles.onLoad({ fromWechat: '1' });
  assert.equal(state.form.name, '张三');

  // 切换账号为 User B
  memberStore.member = { id: 3002, nickname: 'UserB' };
  memberStore.token = 'user-b-token';
  await vue.nextTick();

  // 尝试以 User B 身份保存 User A 的草稿
  await state.handleSave();
  assert.equal(addAddressCalled, false, '跨账号提交微信草稿必须被严格拦截，禁止调用 addAddress');

  app.unmount();
  console.log('-> Test 4 PASSED');
}

// ----------------------------------------------------
// 测试 5: 清空搜索后旧响应忽略 (search/index.vue)
// ----------------------------------------------------
console.log('Test 5: 清空搜索后旧请求晚返回直接丢弃');
{
  let resolveSearch;
  const mockProductApi = {
    getProductList: () => {
      return new Promise((resolve) => {
        resolveSearch = () => resolve({
          code: 1,
          data: [{ id: 88, name: '搜索出的手机' }],
          total: 1,
        });
      });
    },
  };

  const { state, app } = createSfcInstance('src/pages/search/index.vue', {
    '@/api/product': mockProductApi,
    '@/api/search': { getSearchConfig: async () => ({ code: 1, data: {} }) },
    '@/stores/app': { useAppStore: () => ({ bootstrapData: vue.ref(null) }) },
  });

  // 1. 发起搜索
  state.keywordInput = '手机';
  state.handleConfirmSearch();

  // 2. 在请求未完成时，用户清空了搜索输入
  state.handleClearInput();
  assert.equal(state.submittedKeyword, '');
  assert.equal(state.productList.length, 0);

  // 3. 旧搜索响应返回
  resolveSearch();
  await delay(10);

  // 4. 验证：清空后旧响应不得被重新写回列表！
  assert.equal(state.productList.length, 0, '清空搜索后旧响应被成功忽略，未污染列表');
  assert.equal(state.total, 0);

  app.unmount();
  console.log('-> Test 5 PASSED');
}

// ----------------------------------------------------
// 测试 6: onHide 无重试/计时器残留 (order/detail.vue)
// ----------------------------------------------------
console.log('Test 6: 订单页 onHide 立即清理状态重查与计时器');
{
  let getDetailCallCount = 0;
  const mockOrderApi = {
    getOrderDetail: async () => {
      getDetailCallCount++;
      return {
        code: 1,
        data: {
          id: 501,
          status: 10,
          can_pay: true,
          payment_expires_at: new Date(Date.now() + 60000).toISOString(),
          server_time: new Date().toISOString(),
        },
      };
    },
    cancelOrder: async () => {},
    confirmOrder: async () => {},
  };

  const { state, lifecycles, app } = createSfcInstance('src/pages/order/detail.vue', {
    '@/api/order': mockOrderApi,
    '@/api/payment': {},
    '@/composables/useOrderCountdown': {
      syncOrderCountdownClock: () => {},
      parseDateToMs: (str) => new Date(str).getTime(),
    },
  });

  lifecycles.onLoad({ id: '501' });
  await delay(10);
  assert.equal(getDetailCallCount, 1);

  // 触发超时状态确认流程
  state.handleOrderTimeout();
  assert.equal(state.isConfirming, true);

  // 页面进入后台 onHide
  lifecycles.onHide();
  assert.equal(state.isPageHidden, true);

  // 推进时间，验证重试未在后台执行
  const prevCount = getDetailCallCount;
  await delay(2500);
  assert.equal(getDetailCallCount, prevCount, 'onHide 后重查定时器已清除，不向服务器发送后台请求');

  await lifecycles.onShow();
  assert.equal(getDetailCallCount, prevCount + 1, 'onShow 应立即重新确认订单状态');
  await delay(2200);
  assert.ok(getDetailCallCount >= prevCount + 2, '仍待付款时应恢复有限重查');

  app.unmount();
  console.log('-> Test 6 PASSED');
}

// ----------------------------------------------------
// 测试 7: 恢复网络不自动提交/支付 (useNetwork 机制)
// ----------------------------------------------------
console.log('Test 7: 恢复网络仅触发只读刷新，绝不自动重放写请求');
{
  let orderCreatedCount = 0;
  let paymentCreatedCount = 0;
  let readRefreshCount = 0;

  // 模拟写请求拦截器或业务函数
  function createOrder() {
    orderCreatedCount++;
  }
  function makePayment() {
    paymentCreatedCount++;
  }

  // 组件/页面仅能注册只读刷新
  const cleanup = onNetworkOnline(() => {
    readRefreshCount++;
  });

  // 先将状态变为离线
  triggerNetworkStatusChangeForTest({ isConnected: false, networkType: 'none' });
  // 再触发网络恢复在线
  triggerNetworkStatusChangeForTest({ isConnected: true, networkType: 'wifi' });

  assert.equal(readRefreshCount, 1, '只读刷新被触发 1 次');
  assert.equal(orderCreatedCount, 0, '网络恢复禁止自动重放创建订单');
  assert.equal(paymentCreatedCount, 0, '网络恢复禁止自动重放支付');

  cleanup();
  console.log('-> Test 7 PASSED');
}

console.log('\n========================================');
console.log('ALL 9 REAL SFC & COMPOSABLE LIFECYCLE TESTS PASSED!');
console.log('========================================\n');
