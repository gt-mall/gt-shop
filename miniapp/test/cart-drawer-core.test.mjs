import assert from 'node:assert/strict';
import { nextTick } from 'vue';
import { createPinia, setActivePinia, defineStore } from 'pinia';

// 直接引入生产源码，确保测试的是真实实现
import {
  isItemCheckoutAvailable,
  calculateCartTotalFen,
  calculateCartTotalAmount,
  calculateSelectedValidCount,
  formatCartSpecs,
  filterCheckoutCartItems,
  validateCheckoutPrerequisites,
} from '../src/utils/cart-core.ts';

import { useCartManage } from '../src/composables/useCartManage.ts';
import { useMemberStore } from '../src/stores/member.ts';

console.log('--- Starting Cart Drawer & Selection Core Logic Tests ---');

// Mock uni global environment
const storage = new Map();
let navigateToUrl = '';
let showToastMsg = '';

globalThis.uni = {
  getStorageSync(key) {
    return storage.get(key);
  },
  setStorageSync(key, val) {
    storage.set(key, val);
  },
  removeStorageSync(key) {
    storage.delete(key);
  },
  showToast({ title }) {
    showToastMsg = title;
  },
  showModal({ success }) {
    if (success) success({ confirm: true });
  },
  showLoading() {},
  hideLoading() {},
  navigateTo({ url, complete }) {
    navigateToUrl = url;
    if (complete) complete();
  },
  switchTab() {},
  setTabBarBadge() {},
  removeTabBarBadge() {},
  stopPullDownRefresh() {},
};

// ----------------------------------------------------
// 测试 1: isItemCheckoutAvailable 判定
// ----------------------------------------------------
console.log('Test 1: isItemCheckoutAvailable 校验规则');
{
  const normalItem = { id: 1, available: true, quantity: 2, stock: 10 };
  assert.equal(isItemCheckoutAvailable(normalItem), true, '有效且库存充足应为可结算');

  const outOfStockItem = { id: 2, available: true, quantity: 5, stock: 3 };
  assert.equal(isItemCheckoutAvailable(outOfStockItem), false, '数量超过库存不可结算');

  const equalStockItem = { id: 3, available: true, quantity: 5, stock: 5 };
  assert.equal(isItemCheckoutAvailable(equalStockItem), true, '数量等于库存应可结算');

  const unavailableItem = { id: 4, available: false, quantity: 1, stock: 10 };
  assert.equal(isItemCheckoutAvailable(unavailableItem), false, '已下架/失效商品不可结算');

  const zeroQtyItem = { id: 5, available: true, quantity: 0, stock: 10 };
  assert.equal(isItemCheckoutAvailable(zeroQtyItem), false, '数量为0不可结算');

  assert.equal(isItemCheckoutAvailable(null), false, 'null 安全性');
  assert.equal(isItemCheckoutAvailable(undefined), false, 'undefined 安全性');
}
console.log('-> Test 1 PASSED');

// ----------------------------------------------------
// 测试 2: calculateCartTotalFen / calculateCartTotalAmount 精确金额计算（防浮点数陷阱）
// ----------------------------------------------------
console.log('Test 2: 购物车金额精确计算（分与元转换）');
{
  const testItems = [
    { id: 1, available: true, price: '2.09', quantity: 3, stock: 10 }, // 2.09 * 3 = 6.27 (避免 6.2700000000000005)
    { id: 2, available: true, price: '19.90', quantity: 2, stock: 5 }, // 19.90 * 2 = 39.80
    { id: 3, available: true, price: '99.00', quantity: 5, stock: 2 }, // 超库存，不计入金额
    { id: 4, available: false, price: '50.00', quantity: 1, stock: 10 }, // 失效，不计入金额
  ];

  const selectedAll = new Set([1, 2, 3, 4]);

  // 应该只计算 item 1 和 2: 627 + 3980 = 4607 分 -> 46.07 元
  const totalFen = calculateCartTotalFen(testItems, selectedAll);
  assert.equal(totalFen, 4607, '总分值应为 4607');

  const totalAmount = calculateCartTotalAmount(testItems, selectedAll);
  assert.equal(totalAmount, '46.07', '总金额应精确为 46.07 元');

  // 只勾选 item 1
  assert.equal(calculateCartTotalAmount(testItems, new Set([1])), '6.27');

  // 空选中
  assert.equal(calculateCartTotalAmount(testItems, new Set()), '0.00');

  // 种类统计
  assert.equal(calculateSelectedValidCount(testItems, selectedAll), 2, '仅 2 种商品符合结算条件');
}
console.log('-> Test 2 PASSED');

// ----------------------------------------------------
// 测试 3: formatCartSpecs 规格格式化
// ----------------------------------------------------
console.log('Test 3: formatCartSpecs 规格字符串拼接');
{
  const specs = [
    { spec_id: 1, spec_name: '颜色', value_id: 11, value_name: '远峰蓝' },
    { spec_id: 2, spec_name: '内存', value_id: 22, value_name: '512GB' },
  ];
  assert.equal(formatCartSpecs(specs), '远峰蓝 · 512GB');
  assert.equal(formatCartSpecs([]), '');
  assert.equal(formatCartSpecs(undefined), '');
}
console.log('-> Test 3 PASSED');

// ----------------------------------------------------
// 测试 4: validateCheckoutPrerequisites 结算前置校验
// ----------------------------------------------------
console.log('Test 4: validateCheckoutPrerequisites 结算阻断与校验');
{
  const items = [
    { id: 10, sku_id: 100, available: true, price: '10.00', quantity: 1, stock: 5 },
    { id: 20, sku_id: 200, available: true, price: '20.00', quantity: 5, stock: 2 }, // 超库存
    { id: 30, sku_id: 300, available: false, price: '30.00', quantity: 1, stock: 5 }, // 失效
  ];

  // 1. 未选任何商品
  const r1 = validateCheckoutPrerequisites(items, new Set());
  assert.equal(r1.ok, false);
  assert.equal(r1.error, '请选择要结算的商品');

  // 2. 勾选包含超库存商品
  const r2 = validateCheckoutPrerequisites(items, new Set([10, 20]));
  assert.equal(r2.ok, false);
  assert.equal(r2.error, '库存不足，请先调整购买数量');

  // 3. 勾选仅包含失效商品
  const r3 = validateCheckoutPrerequisites(items, new Set([30]));
  assert.equal(r3.ok, false);
  assert.equal(r3.error, '所选商品暂不可结算');

  // 4. 正常勾选有效商品
  const r4 = validateCheckoutPrerequisites(items, new Set([10]));
  assert.equal(r4.ok, true);
  assert.equal(r4.checkoutItems.length, 1);
  assert.equal(r4.checkoutItems[0].id, 10);
}
console.log('-> Test 4 PASSED');

// ----------------------------------------------------
// 测试 5: useCartManage 响应式状态管理（Pinia 环境）
// ----------------------------------------------------
console.log('Test 5: useCartManage 状态同步、全选/反选与账号切换');
{
  const pinia = createPinia();
  setActivePinia(pinia);

  storage.set('token', 'token_user_A');

  const cartManage = useCartManage();
  const { cartStore } = cartManage;

  // 注入测试数据到 Store
  cartStore.items = [
    { id: 1, sku_id: 101, available: true, price: '10.00', quantity: 2, stock: 5 },
    { id: 2, sku_id: 102, available: true, price: '25.50', quantity: 1, stock: 10 },
    { id: 3, sku_id: 103, available: true, price: '15.00', quantity: 8, stock: 3 }, // 超库存
    { id: 4, sku_id: 104, available: false, price: '99.00', quantity: 1, stock: 5 }, // 失效
  ];

  // 1. syncSelection: 首次应默认勾选所有可结算商品 (id: 1, 2)
  cartManage.syncSelection();
  assert.equal(cartManage.selectedIds.value.has(1), true, '商品1默认被勾选');
  assert.equal(cartManage.selectedIds.value.has(2), true, '商品2默认被勾选');
  assert.equal(cartManage.selectedIds.value.has(3), false, '超库存商品不默认勾选');
  assert.equal(cartManage.selectedIds.value.has(4), false, '失效商品不默认勾选');
  assert.equal(cartManage.selectedValidCount.value, 2);
  // 合计：10.00*2 + 25.50*1 = 45.50
  assert.equal(cartManage.totalAmount.value, '45.50');

  // 2. 单选切换 toggleSelect
  cartManage.toggleSelect(1);
  assert.equal(cartManage.selectedIds.value.has(1), false, '商品1取消勾选');
  assert.equal(cartManage.totalAmount.value, '25.50');

  // 3. 全选 toggleSelectAll (普通模式：针对可结算商品 1, 2)
  cartManage.toggleSelectAll();
  assert.equal(cartManage.isAllSelected.value, true, '普通模式下可结算商品全选');
  assert.equal(cartManage.selectedIds.value.has(1), true);
  assert.equal(cartManage.selectedIds.value.has(2), true);
  assert.equal(cartManage.selectedIds.value.has(3), false);

  // 再次点击全选 -> 全部取消
  cartManage.toggleSelectAll();
  assert.equal(cartManage.isAllSelected.value, false);
  assert.equal(cartManage.selectedIds.value.size, 0);

  // 4. 编辑模式切换与全选
  cartManage.toggleEditMode();
  assert.equal(cartManage.isEditMode.value, true);
  cartManage.toggleSelectAll();
  assert.equal(cartManage.selectedIds.value.size, 4, '编辑模式下全选全部商品（含失效和超库存）');

  // 5. 账号切换清理测试
  storage.set('token', 'token_user_B');
  cartManage.syncSelection(cartStore.items);
  // Token 切换，旧账号选中的应该被重置
  assert.equal(cartManage.selectedIds.value.has(3), false, '账号切换后超库存/失效勾选已被清除');
  assert.equal(cartManage.selectedIds.value.has(4), false);
}
console.log('-> Test 5 PASSED');

// ----------------------------------------------------
// 测试 6: useCartManage 数量修改限制与防重
// ----------------------------------------------------
console.log('Test 6: changeQuantity 数量变更边界与防重');
{
  const pinia = createPinia();
  setActivePinia(pinia);

  storage.set('token', 'token_test_qty');
  const cartManage = useCartManage();
  const { cartStore } = cartManage;

  const item = { id: 99, sku_id: 999, available: true, price: '10.00', quantity: 3, stock: 5 };
  cartStore.items = [item];

  // 1. 正常递增
  let requestedData = null;
  cartStore.updateQuantity = async (data) => {
    requestedData = data;
    item.quantity = data.quantity;
    return { code: 1, message: 'ok', data: null };
  };

  const res1 = await cartManage.changeQuantity(item, 1);
  assert.equal(res1, true);
  assert.equal(requestedData.quantity, 4);
  assert.equal(item.quantity, 4);

  // 2. 超出库存递增拦截 (stock = 5, quantity = 5)
  item.quantity = 5;
  const res2 = await cartManage.changeQuantity(item, 1);
  assert.equal(res2, false, '达到最大库存后增加应拦截');
  assert.equal(showToastMsg, '已达最大可用库存');

  // 3. 超库存商品允许减少 (如 item.quantity = 8, stock = 5)
  item.quantity = 8;
  const res3 = await cartManage.changeQuantity(item, -1);
  assert.equal(res3, true, '库存不足时允许减少数量');
  assert.equal(requestedData.quantity, 7);

  // 4. 减到小于 1 拦截
  item.quantity = 1;
  const res4 = await cartManage.changeQuantity(item, -1);
  assert.equal(res4, false, '数量小于1应拦截');
}
console.log('-> Test 6 PASSED');

// ----------------------------------------------------
// 测试 7: proceedCheckout 结算参数构造与流程
// ----------------------------------------------------
console.log('Test 7: proceedCheckout 结算流程与 payload 存储');
{
  const pinia = createPinia();
  setActivePinia(pinia);

  storage.set('token', 'token_checkout_user');
  const cartManage = useCartManage();
  const { cartStore } = cartManage;

  cartStore.items = [
    { id: 101, sku_id: 1001, available: true, price: '88.00', quantity: 2, stock: 10 },
  ];
  cartManage.syncSelection();

  let beforeNavigateCalled = false;
  cartManage.proceedCheckout({
    redirectFrom: '/pages/product/detail?id=123',
    onBeforeNavigate: () => {
      beforeNavigateCalled = true;
    },
  });

  assert.equal(beforeNavigateCalled, true, '跳转前回调成功执行');
  assert.equal(navigateToUrl, '/pages/order/checkout', '正确导航到结算页');

  const payload = storage.get('checkout_payload');
  assert.ok(payload, '已写入 checkout_payload');
  assert.equal(payload.source, 'cart');
  assert.ok(payload.request_id, 'request_id 已生成');
  assert.deepEqual(payload.cart_item_ids, [101]);
  assert.deepEqual(payload.items, [{ sku_id: 1001, quantity: 2 }]);
}
console.log('-> Test 7 PASSED');

console.log('ALL CART DRAWER & SELECTION CORE LOGIC TESTS PASSED SUCCESSFULLY!');

// 实际 composable：取消勾选后刷新不能重新全选；其他入口删除后清理选中 ID。
{
  setActivePinia(createPinia());
  storage.set('token', 'selection-session');
  const manage = useCartManage();
  manage.cartStore.items = [{ id: 1, sku_id: 1, price: '1.00', available: true, quantity: 1, stock: 5 }];
  manage.syncSelection();
  manage.toggleSelectAll();
  manage.syncSelection();
  assert.equal(manage.selectedIds.value.size, 0, '主动取消全选后刷新保持为空');
  manage.toggleSelect(1);
  manage.cartStore.items = [];
  await nextTick();
  assert.equal(manage.selectedIds.value.size, 0, '其他入口删除后清理勾选');
  manage.isEditMode.value = true;
  manage.selectedIds.value = new Set([99]);
  storage.set('token', 'new-session');
  useMemberStore().token = 'new-session';
  assert.equal(manage.selectedIds.value.size, 0, '账号变化立即清理勾选');
  assert.equal(manage.isEditMode.value, false);
}

function deferred() {
  let resolve;
  let reject;
  const promise = new Promise((yes, no) => { resolve = yes; reject = no; });
  return { promise, resolve, reject };
}
const response = (data = null) => ({ statusCode: 200, data: { code: 1, message: 'ok', data } });

// 实际 Store + 请求层：共享写锁、阻止更新时结算、过滤旧列表及跨账号写响应。
{
  setActivePinia(createPinia());
  storage.set('token', 'mutation-A');
  const manage = useCartManage();
  const store = manage.cartStore;
  store.items = [{ id: 1, sku_id: 1, price: '1.00', available: true, quantity: 1, stock: 5 }];
  manage.syncSelection();
  const oldList = deferred();
  const mutation = deferred();
  uni.request = ({ method }) => method === 'GET' ? oldList.promise : mutation.promise;
  const fetching = store.fetchCart();
  const writing = store.updateQuantity({ id: 1, quantity: 2 });
  const duplicate = await store.removeItem([1]);
  assert.equal(duplicate.code, 0, '更新时不能并行删除');
  storage.delete('checkout_payload');
  manage.proceedCheckout();
  assert.equal(storage.has('checkout_payload'), false, '更新期间不能构造旧数量结算单');
  mutation.resolve(response());
  await writing;
  oldList.resolve(response([{ ...store.items[0], quantity: 1 }]));
  await fetching;
  assert.equal(store.items[0].quantity, 2, '修改前列表晚返回不能覆盖新数量');

  for (const operation of ['update', 'delete']) {
    storage.set('token', 'mutation-A');
    const late = deferred();
    uni.request = () => late.promise;
    const pending = operation === 'update' ? store.updateQuantity({ id: 1, quantity: 4 }) : store.removeItem([1]);
    store.reset();
    storage.set('token', 'mutation-B');
    store.items = [{ id: 1, quantity: 3 }];
    late.resolve(response());
    const result = await pending;
    assert.equal(result.code, 0);
    assert.equal(store.items[0].quantity, 3, '旧账号响应不能修改或删除当前数据');
    assert.equal(store.mutating, false);
  }

  uni.request = async () => { throw new Error('test network failure'); };
  await assert.rejects(store.removeItem([1]), /test network failure/);
  assert.equal(store.mutating, false, '请求失败后释放写锁');
}
console.log('Cart selection, session, mutation and checkout regressions PASSED');
