import assert from 'node:assert/strict';
import { ref, computed } from 'vue';
import { createPinia, setActivePinia, defineStore } from 'pinia';
// 直接使用生产代码的会话一致性纯函数（Node 24 原生支持 type stripping），
// 保证测试验证的是真实 request/upload 会话判断逻辑，而不是测试自己复制的实现。
import { isCurrentSessionToken } from '../src/services/auth-session.ts';

console.log('--- Starting Miniapp Pinia Auth Consistency & Request Isolation Tests ---');

// 1. Mock uni environment
const storage = new Map();
const eventListeners = new Map();

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
  $on(event, cb) {
    if (!eventListeners.has(event)) {
      eventListeners.set(event, []);
    }
    eventListeners.get(event).push(cb);
  },
  $emit(event, ...args) {
    const cbs = eventListeners.get(event) || [];
    for (const cb of cbs) {
      cb(...args);
    }
  },
  setTabBarBadge() {},
  removeTabBarBadge() {},
};

// 2. Auth storage helpers (mirroring src/utils/auth-storage.ts)
const TOKEN_KEY = 'token';
const MEMBER_KEY = 'member_info';
function getToken() {
  return storage.get(TOKEN_KEY) || '';
}
function setToken(token) {
  storage.set(TOKEN_KEY, token);
}
function clearAuthStorage() {
  storage.delete(TOKEN_KEY);
  storage.delete(MEMBER_KEY);
}

// 3. Mock API responses
let mockCartResponse = null;
let mockMemberMeResponse = null;
let mockAddressesResponse = null;

async function getCartList() {
  if (typeof mockCartResponse === 'function') {
    return await mockCartResponse();
  }
  return mockCartResponse;
}

async function getMemberMe() {
  if (typeof mockMemberMeResponse === 'function') {
    return await mockMemberMeResponse();
  }
  return mockMemberMeResponse;
}

async function getMemberAddresses() {
  if (typeof mockAddressesResponse === 'function') {
    return await mockAddressesResponse();
  }
  return mockAddressesResponse;
}

// 4. Implement stores mirroring src/stores/cart.ts and src/stores/member.ts
const useCartStore = defineStore('cart', () => {
  const items = ref([]);
  const loading = ref(false);
  const loaded = ref(false);
  const error = ref(null);
  let fetchSeq = 0;

  async function fetchCart() {
    const requestToken = getToken();
    const seq = ++fetchSeq;

    if (!requestToken) {
      items.value = [];
      loaded.value = true;
      loading.value = false;
      return [];
    }

    loading.value = true;
    error.value = null;
    try {
      const res = await getCartList();
      if (requestToken !== getToken() || seq !== fetchSeq) {
        return items.value;
      }
      if (res && res.code === 1 && Array.isArray(res.data)) {
        items.value = res.data;
        loaded.value = true;
        return res.data;
      } else {
        error.value = res?.message || '获取购物车失败';
        return items.value;
      }
    } catch (err) {
      if (requestToken !== getToken() || seq !== fetchSeq) {
        return items.value;
      }
      error.value = err.message || '获取购物车失败';
      return items.value;
    } finally {
      if (requestToken === getToken() && seq === fetchSeq) {
        loading.value = false;
      }
    }
  }

  function reset() {
    fetchSeq++;
    items.value = [];
    loading.value = false;
    loaded.value = false;
    error.value = null;
  }

  return { items, loading, loaded, error, fetchCart, reset };
});

const useMemberStore = defineStore('member', () => {
  const token = ref(getToken());
  const member = ref(null);
  const addresses = ref([]);
  const loading = ref(false);
  const error = ref(null);

  const isLoggedIn = computed(() => !!token.value && !!member.value);

  let refreshSeq = 0;
  let addressSeq = 0;

  async function refreshMember() {
    const requestToken = getToken();
    if (!requestToken) return null;
    const seq = ++refreshSeq;
    try {
      const res = await getMemberMe();
      if (requestToken !== getToken() || seq !== refreshSeq) {
        return member.value;
      }
      if (res && res.code === 1 && res.data) {
        member.value = res.data;
        return member.value;
      }
      return member.value;
    } catch (err) {
      if (requestToken !== getToken() || seq !== refreshSeq) {
        return member.value;
      }
      return member.value;
    }
  }

  async function fetchAddresses() {
    const requestToken = getToken();
    if (!requestToken) {
      addresses.value = [];
      return [];
    }
    const seq = ++addressSeq;
    try {
      const res = await getMemberAddresses();
      if (requestToken !== getToken() || seq !== addressSeq) {
        return addresses.value;
      }
      if (res && res.code === 1 && res.data) {
        addresses.value = res.data;
        return res.data;
      }
      return addresses.value;
    } catch (err) {
      if (requestToken !== getToken() || seq !== addressSeq) {
        return addresses.value;
      }
      return addresses.value;
    }
  }

  function logout() {
    refreshSeq++;
    addressSeq++;
    token.value = '';
    member.value = null;
    addresses.value = [];
    loading.value = false;
    error.value = null;
    clearAuthStorage();
    uni.removeStorageSync('checkout_payload');

    try {
      const cartStore = useCartStore();
      cartStore.reset();
    } catch (err) {}
  }

  return { token, member, addresses, loading, error, isLoggedIn, refreshMember, fetchAddresses, logout };
});

// Setup App.vue listener
function setupApp(memberStore) {
  uni.$on('auth-expired', () => {
    memberStore.logout();
  });
}

// Request handler with handleAuthExpired mirroring src/services/request.ts.
// The session guard delegates to the real production helper imported above.
function handleAuthExpired(requestToken) {
  if (!isCurrentSessionToken(requestToken, getToken())) {
    return;
  }
  clearAuthStorage();
  uni.$emit('auth-expired');
}

async function fakeRequest(options) {
  // 请求创建阶段捕获 requestToken（模拟生产代码 request() 的 requestToken）
  const requestToken = getToken();
  const resp = await options.mockResult;
  if (resp.statusCode === 401) {
    handleAuthExpired(requestToken);
    throw new Error('请先登录或登录凭证已失效');
  }
  if (resp.data && (resp.data.code === -1 || (resp.data.message && resp.data.message.includes('authentication required')))) {
    handleAuthExpired(requestToken);
    throw new Error('请先登录或登录凭证已失效');
  }
  return resp.data;
}

async function fakeUpload(options) {
  // 请求创建阶段捕获 requestToken（模拟生产代码 http.upload() 的 requestToken）
  const requestToken = getToken();
  const res = await options.mockResult;
  if (res.statusCode === 401) {
    handleAuthExpired(requestToken);
    throw new Error('请先登录或登录凭证已失效');
  }
  const data = JSON.parse(res.data);
  if (data.code === -1 || (data.message && data.message.includes('authentication required'))) {
    handleAuthExpired(requestToken);
    throw new Error('请先登录或登录凭证已失效');
  }
  return data;
}

// Run the 7 test scenarios
async function runTests() {
  setActivePinia(createPinia());
  const memberStore = useMemberStore();
  const cartStore = useCartStore();
  setupApp(memberStore);

  console.log('Running Test 1: Token 过期清理 Storage, memberStore, cartStore, checkout_payload...');
  {
    setToken('token_test_1');
    memberStore.token = 'token_test_1';
    memberStore.member = { id: 1, nickname: 'Alice' };
    memberStore.addresses = [{ id: 10 }];
    cartStore.items = [{ id: 100, quantity: 2 }];
    uni.setStorageSync('checkout_payload', { items: [100] });

    assert.equal(getToken(), 'token_test_1');
    assert.equal(memberStore.isLoggedIn, true);
    assert.equal(cartStore.items.length, 1);
    assert.ok(uni.getStorageSync('checkout_payload'));

    // Trigger token expiration via business code = -1
    await assert.rejects(
      async () => await fakeRequest({ mockResult: { statusCode: 200, data: { code: -1, message: 'authentication required' } } }),
      /请先登录或登录凭证已失效/
    );

    assert.equal(getToken(), '', 'Storage token must be cleared');
    assert.equal(memberStore.token, '', 'memberStore.token must be cleared');
    assert.equal(memberStore.member, null, 'memberStore.member must be null');
    assert.equal(memberStore.addresses.length, 0, 'memberStore.addresses must be empty');
    assert.equal(cartStore.items.length, 0, 'cartStore.items must be empty');
    assert.equal(uni.getStorageSync('checkout_payload'), undefined, 'checkout_payload must be removed');
    console.log('-> Test 1 PASSED');
  }

  console.log('Running Test 2: HTTP 401 同样触发全局 logout...');
  {
    setToken('token_test_2');
    memberStore.token = 'token_test_2';
    memberStore.member = { id: 2, nickname: 'Bob' };
    cartStore.items = [{ id: 200 }];
    uni.setStorageSync('checkout_payload', { items: [200] });

    await assert.rejects(
      async () => await fakeRequest({ mockResult: { statusCode: 401, data: 'Unauthorized' } }),
      /请先登录或登录凭证已失效/
    );

    assert.equal(getToken(), '');
    assert.equal(memberStore.token, '');
    assert.equal(memberStore.member, null);
    assert.equal(cartStore.items.length, 0);
    assert.equal(uni.getStorageSync('checkout_payload'), undefined);
    console.log('-> Test 2 PASSED');
  }

  console.log('Running Test 3: upload 返回认证失效同样触发 logout...');
  {
    setToken('token_test_3');
    memberStore.token = 'token_test_3';
    memberStore.member = { id: 3, nickname: 'Charlie' };

    await assert.rejects(
      async () => await fakeUpload({ mockResult: { statusCode: 200, data: JSON.stringify({ code: -1, message: 'authentication required' }) } }),
      /请先登录或登录凭证已失效/
    );

    assert.equal(getToken(), '');
    assert.equal(memberStore.token, '');
    assert.equal(memberStore.member, null);
    console.log('-> Test 3 PASSED');
  }

  console.log('Running Test 4: A 的 fetchCart 晚返回，B 已登录后不能覆盖 B 数据...');
  {
    setToken('token_user_A');
    memberStore.token = 'token_user_A';

    let resolveA;
    mockCartResponse = () => new Promise((resolve) => { resolveA = resolve; });

    // A launches fetchCart
    const fetchPromiseA = cartStore.fetchCart();

    // User A logs out
    memberStore.logout();
    assert.equal(cartStore.items.length, 0);

    // User B logs in
    setToken('token_user_B');
    memberStore.token = 'token_user_B';
    memberStore.member = { id: 20, nickname: 'User B' };

    // User B fetches cart and gets B's items
    mockCartResponse = { code: 1, data: [{ id: 999, name: 'B Product' }] };
    await cartStore.fetchCart();
    assert.equal(cartStore.items.length, 1);
    assert.equal(cartStore.items[0].id, 999);

    // Now A's old slow request returns
    resolveA({ code: 1, data: [{ id: 111, name: 'A Product' }] });
    await fetchPromiseA;

    // Verify: Cart MUST still be B's product, NOT overwritten by A
    assert.equal(cartStore.items.length, 1);
    assert.equal(cartStore.items[0].id, 999, "A's stale response must not overwrite B's cart");
    console.log('-> Test 4 PASSED');
  }

  console.log('Running Test 5: A 的 fetchAddresses 晚返回，B 已登录后不能覆盖 B 数据...');
  {
    setToken('token_user_A');
    memberStore.token = 'token_user_A';

    let resolveAddressesA;
    mockAddressesResponse = () => new Promise((resolve) => { resolveAddressesA = resolve; });

    const fetchAddressesA = memberStore.fetchAddresses();

    memberStore.logout();

    setToken('token_user_B');
    memberStore.token = 'token_user_B';

    mockAddressesResponse = { code: 1, data: [{ id: 500, detail: 'Address B' }] };
    await memberStore.fetchAddresses();
    assert.equal(memberStore.addresses.length, 1);
    assert.equal(memberStore.addresses[0].id, 500);

    // A's response returns
    resolveAddressesA({ code: 1, data: [{ id: 100, detail: 'Address A' }] });
    await fetchAddressesA;

    // Verify: Addresses must still be B's
    assert.equal(memberStore.addresses.length, 1);
    assert.equal(memberStore.addresses[0].id, 500, "A's stale response must not overwrite B's addresses");
    console.log('-> Test 5 PASSED');
  }

  console.log('Running Test 6: logout 后旧请求返回，Store 不能复活...');
  {
    setToken('token_user_A');
    memberStore.token = 'token_user_A';

    let resolveCartA;
    mockCartResponse = () => new Promise((resolve) => { resolveCartA = resolve; });

    const fetchPromise = cartStore.fetchCart();

    // User logs out
    memberStore.logout();
    assert.equal(cartStore.items.length, 0);

    // Old response returns
    resolveCartA({ code: 1, data: [{ id: 10, name: 'Ghost Item' }] });
    await fetchPromise;

    // Verify: Cart must remain empty!
    assert.equal(cartStore.items.length, 0, 'Store must not resurrect after logout');
    console.log('-> Test 6 PASSED');
  }

  console.log('Running Test 7: 同账号连续两个 fetchCart，旧请求晚返回不能覆盖新请求...');
  {
    setToken('token_user_A');
    memberStore.token = 'token_user_A';

    let resolveReq1;
    let resolveReq2;

    mockCartResponse = () => new Promise((resolve) => { resolveReq1 = resolve; });
    const p1 = cartStore.fetchCart(); // Request 1 (seq = 1, slow)

    mockCartResponse = () => new Promise((resolve) => { resolveReq2 = resolve; });
    const p2 = cartStore.fetchCart(); // Request 2 (seq = 2, fast)

    // Request 2 completes first
    resolveReq2({ code: 1, data: [{ id: 2, title: 'New Version' }] });
    await p2;
    assert.equal(cartStore.items[0].id, 2);

    // Request 1 completes later
    resolveReq1({ code: 1, data: [{ id: 1, title: 'Old Version' }] });
    await p1;

    // Verify: Items must still be 'New Version'
    assert.equal(cartStore.items.length, 1);
    assert.equal(cartStore.items[0].id, 2, 'Stale seq 1 must not overwrite newer seq 2');
    console.log('-> Test 7 PASSED');
  }

  console.log('Running Test 8: A 的旧 request 返回 HTTP 401，不能踢掉后登录的 B...');
  {
    setToken('token_user_A');
    memberStore.token = 'token_user_A';
    memberStore.member = { id: 10, nickname: 'User A' };
    cartStore.items = [{ id: 10, name: 'A Item' }];

    // A 的请求已发出（requestToken = A），但尚未返回
    let resolveOldRequest;
    const oldRequestPromise = fakeRequest({
      mockResult: new Promise((resolve) => { resolveOldRequest = resolve; }),
    });

    // A logout
    memberStore.logout();
    assert.equal(getToken(), '');

    // B login
    setToken('token_user_B');
    memberStore.token = 'token_user_B';
    memberStore.member = { id: 20, nickname: 'User B' };
    cartStore.items = [{ id: 200, name: 'B Item' }];
    uni.setStorageSync('checkout_payload', { items: [200] });

    // A 的旧请求返回 HTTP 401，旧请求自身仍然报错，但不能影响 B
    resolveOldRequest({ statusCode: 401, data: 'Unauthorized' });
    await assert.rejects(() => oldRequestPromise, /请先登录或登录凭证已失效/);

    assert.equal(getToken(), 'token_user_B', 'B token must survive stale 401');
    assert.equal(memberStore.token, 'token_user_B', 'memberStore B must survive stale 401');
    assert.deepEqual(memberStore.member, { id: 20, nickname: 'User B' });
    assert.equal(cartStore.items.length, 1, 'B cart must not be reset by stale 401');
    assert.equal(cartStore.items[0].id, 200, 'B cart must survive stale 401');
    assert.ok(uni.getStorageSync('checkout_payload'), 'B checkout_payload must survive stale 401');
    console.log('-> Test 8 PASSED');
  }

  console.log('Running Test 9: A 的旧 upload 返回 code=-1，不能踢掉后登录的 B...');
  {
    setToken('token_user_A');
    memberStore.token = 'token_user_A';
    memberStore.member = { id: 10, nickname: 'User A' };
    cartStore.items = [{ id: 10, name: 'A Item' }];

    // A 的 upload 已发出（requestToken = A），但尚未返回
    let resolveOldUpload;
    const oldUploadPromise = fakeUpload({
      mockResult: new Promise((resolve) => { resolveOldUpload = resolve; }),
    });

    memberStore.logout();

    setToken('token_user_B');
    memberStore.token = 'token_user_B';
    memberStore.member = { id: 20, nickname: 'User B' };
    cartStore.items = [{ id: 200, name: 'B Item' }];

    resolveOldUpload({ statusCode: 200, data: JSON.stringify({ code: -1, message: 'authentication required' }) });
    await assert.rejects(() => oldUploadPromise, /请先登录或登录凭证已失效/);

    assert.equal(getToken(), 'token_user_B', 'B token must survive stale upload auth failure');
    assert.equal(memberStore.token, 'token_user_B');
    assert.deepEqual(memberStore.member, { id: 20, nickname: 'User B' });
    assert.equal(cartStore.items.length, 1, 'B cart must not be reset by stale upload failure');
    assert.equal(cartStore.items[0].id, 200);
    console.log('-> Test 9 PASSED');
  }

  console.log('Running Test 10: 当前 Token 本身过期时，必须完整清理 Storage / memberStore / cart / checkout_payload...');
  {
    setToken('token_user_A');
    memberStore.token = 'token_user_A';
    memberStore.member = { id: 10, nickname: 'User A' };
    memberStore.addresses = [{ id: 1 }];
    cartStore.items = [{ id: 10, name: 'A Item' }];
    uni.setStorageSync('checkout_payload', { items: [10] });

    await assert.rejects(
      () => fakeRequest({ mockResult: { statusCode: 401, data: 'Unauthorized' } }),
      /请先登录或登录凭证已失效/,
    );

    assert.equal(getToken(), '', 'current session token must be cleared');
    assert.equal(memberStore.token, '', 'memberStore must be logged out');
    assert.equal(memberStore.member, null);
    assert.equal(memberStore.addresses.length, 0);
    assert.equal(cartStore.items.length, 0, 'cart must be reset on current-session 401');
    assert.equal(uni.getStorageSync('checkout_payload'), undefined, 'checkout_payload must be cleared');
    console.log('-> Test 10 PASSED');
  }

  console.log('\nALL 10 MINIAPP AUTH & REQUEST CONSISTENCY TESTS PASSED SUCCESSFULLY!');
}
runTests().catch((err) => {
  console.error('Test failed:', err);
  process.exit(1);
});
