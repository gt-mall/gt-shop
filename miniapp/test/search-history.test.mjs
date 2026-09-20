import assert from 'node:assert/strict';

// 模拟 uni storage 环境
const mockStorage = new Map();
globalThis.uni = {
  getStorageSync(key) {
    return mockStorage.get(key) || '';
  },
  setStorageSync(key, val) {
    mockStorage.set(key, val);
  },
  removeStorageSync(key) {
    mockStorage.delete(key);
  },
  clearStorageSync() {
    mockStorage.clear();
  },
};

// 引入生产源码
const {
  normalizeKeyword,
  getSearchHistoryKey,
  getSearchHistory,
  addSearchHistory,
  removeSearchHistoryItem,
  clearSearchHistory,
  clearUserSearchHistory,
  MAX_SEARCH_HISTORY_COUNT,
  GUEST_SEARCH_HISTORY_KEY,
} = await import('../src/utils/search-history.ts');

console.log('--- Starting Search History Core Logic Tests ---');

// ----------------------------------------------------
// 测试 1: normalizeKeyword 规范化与截断
// ----------------------------------------------------
console.log('Test 1: normalizeKeyword 规范化与截断');
{
  assert.equal(normalizeKeyword('  iPhone 15   '), 'iPhone 15');
  assert.equal(normalizeKeyword('Apple    Watch    Ultra   2'), 'Apple Watch Ultra 2');
  assert.equal(normalizeKeyword('\t\n  华为   Mate 60 Pro \r\n '), '华为 Mate 60 Pro');
  assert.equal(normalizeKeyword(''), '');
  assert.equal(normalizeKeyword('   '), '');
  assert.equal(normalizeKeyword(null), '');
  assert.equal(normalizeKeyword(undefined), '');

  const longStr = 'A'.repeat(60);
  const normalizedLong = normalizeKeyword(longStr);
  assert.equal(normalizedLong.length, 50);
  assert.equal(normalizedLong, 'A'.repeat(50));
}

// ----------------------------------------------------
// 测试 2: 存储 Key 会员隔离与游客独立
// ----------------------------------------------------
console.log('Test 2: 存储 Key 会员隔离与游客独立');
{
  assert.equal(getSearchHistoryKey(), GUEST_SEARCH_HISTORY_KEY);
  assert.equal(getSearchHistoryKey(null), GUEST_SEARCH_HISTORY_KEY);
  assert.equal(getSearchHistoryKey(undefined), GUEST_SEARCH_HISTORY_KEY);
  assert.equal(getSearchHistoryKey(''), GUEST_SEARCH_HISTORY_KEY);
  assert.equal(getSearchHistoryKey(0), GUEST_SEARCH_HISTORY_KEY);

  assert.equal(getSearchHistoryKey(101), 'search_history_user_101');
  assert.equal(getSearchHistoryKey('202'), 'search_history_user_202');
}

// ----------------------------------------------------
// 测试 3: 搜索历史新增、去重与最近优先
// ----------------------------------------------------
console.log('Test 3: 搜索历史新增、去重与最近优先');
{
  clearSearchHistory(); // 清空游客
  const guestId = null;

  // 添加首个词
  let list = addSearchHistory('苹果手机', guestId);
  assert.deepEqual(list, ['苹果手机']);

  // 添加第二个词，排在前面
  list = addSearchHistory('机械键盘', guestId);
  assert.deepEqual(list, ['机械键盘', '苹果手机']);

  // 重复添加“苹果手机”，应提升至第一位且去重
  list = addSearchHistory('  苹果手机  ', guestId);
  assert.deepEqual(list, ['苹果手机', '机械键盘']);

  // 大小写去重
  list = addSearchHistory('iphone', guestId);
  assert.deepEqual(list, ['iphone', '苹果手机', '机械键盘']);
  list = addSearchHistory('iPhone', guestId);
  assert.deepEqual(list, ['iPhone', '苹果手机', '机械键盘']);

  // 空白词不应记录
  list = addSearchHistory('   ', guestId);
  assert.deepEqual(list, ['iPhone', '苹果手机', '机械键盘']);
}

// ----------------------------------------------------
// 测试 4: 上限 20 条自动截断
// ----------------------------------------------------
console.log('Test 4: 上限 20 条自动截断');
{
  clearSearchHistory();
  for (let i = 1; i <= 25; i++) {
    addSearchHistory(`搜索词_${i}`);
  }
  const list = getSearchHistory();
  assert.equal(list.length, MAX_SEARCH_HISTORY_COUNT);
  assert.equal(list[0], '搜索词_25');
  assert.equal(list[MAX_SEARCH_HISTORY_COUNT - 1], '搜索词_6');
}

// ----------------------------------------------------
// 测试 5: 单条删除与清空
// ----------------------------------------------------
console.log('Test 5: 单条删除与清空');
{
  clearSearchHistory();
  addSearchHistory('商品A');
  addSearchHistory('商品B');
  addSearchHistory('商品C');

  let list = removeSearchHistoryItem('商品B');
  assert.deepEqual(list, ['商品C', '商品A']);

  clearSearchHistory();
  assert.deepEqual(getSearchHistory(), []);
}

// ----------------------------------------------------
// 测试 6: 会员与游客历史隔离，登出清理会员历史
// ----------------------------------------------------
console.log('Test 6: 会员与游客历史隔离，登出清理会员历史');
{
  clearSearchHistory(); // 游客
  clearUserSearchHistory(1001); // 用户 1001
  clearUserSearchHistory(1002); // 用户 1002

  addSearchHistory('游客看电视', null);
  addSearchHistory('用户A买电脑', 1001);
  addSearchHistory('用户B买显卡', 1002);

  assert.deepEqual(getSearchHistory(null), ['游客看电视']);
  assert.deepEqual(getSearchHistory(1001), ['用户A买电脑']);
  assert.deepEqual(getSearchHistory(1002), ['用户B买显卡']);

  // 用户 1001 登出清理
  clearUserSearchHistory(1001);
  assert.deepEqual(getSearchHistory(1001), []);
  // 游客与用户 1002 历史不受影响
  assert.deepEqual(getSearchHistory(null), ['游客看电视']);
  assert.deepEqual(getSearchHistory(1002), ['用户B买显卡']);
}

// ----------------------------------------------------
// 测试 7: 存储异常容错降级
// ----------------------------------------------------
console.log('Test 7: 存储异常容错降级');
{
  // 模拟损坏数据
  mockStorage.set('search_history_guest', 'INVALID_JSON_CORRUPTED');
  const fallbackList = getSearchHistory(null);
  assert.ok(Array.isArray(fallbackList));

  // 添加词应依然能正常工作
  const rescuedList = addSearchHistory('新商品', null);
  assert.deepEqual(rescuedList, ['新商品']);
}

console.log('All Search History tests passed successfully!');
