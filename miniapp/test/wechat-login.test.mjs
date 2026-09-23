import assert from 'node:assert/strict';
import { createPinia, setActivePinia } from 'pinia';

const storage = new Map();
let loginCallback;
let requestHandler;
const requests = [];

globalThis.uni = {
  getStorageSync: (key) => storage.get(key),
  setStorageSync: (key, value) => storage.set(key, value),
  removeStorageSync: (key) => storage.delete(key),
  login: (options) => { loginCallback = options; },
  request: (options) => {
    requests.push(options);
    return requestHandler(options);
  },
  removeTabBarBadge: () => {},
  setTabBarBadge: () => {},
  $emit: () => {},
};

const { useMemberStore } = await import('../src/stores/member.ts');

function freshStore() {
  storage.clear();
  requests.length = 0;
  loginCallback = undefined;
  requestHandler = async () => ({ statusCode: 200, data: { code: 1, data: [] } });
  setActivePinia(createPinia());
  return useMemberStore();
}

function authReply(isNewMember = true) {
  return {
    statusCode: 200,
    data: {
      code: 1,
      data: {
        token: 'test-jwt',
        expires_at: '2030-01-01T00:00:00Z',
        is_new_member: isNewMember,
        member: {
          id: 8,
          nickname: '微信用户',
          avatar_id: null,
          avatar_url: '',
          mobile: null,
          status: 1,
          last_login_at: null,
          created_at: '2026-01-01T00:00:00Z',
        },
      },
    },
  };
}

console.log('WeChat one-tap login: first login, repeat login and logout');
{
  const store = freshStore();
  requestHandler = async (options) => options.url.endsWith('/auth/wechat-login')
    ? authReply(true)
    : { statusCode: 200, data: { code: 1, data: [] } };
  const pending = store.loginWithWechat();
  assert.equal(store.loading, true, 'uni.login starts the loading state');
  loginCallback.success({ code: 'wx-code' });
  const result = await pending;
  assert.equal(result.is_new_member, true);
  assert.equal(requests[0].data.code, 'wx-code');
  assert.equal(store.isLoggedIn, true);
  assert.equal(storage.get('token'), 'test-jwt');
  assert.equal(storage.get('member_info').id, 8);

  storage.set('user_avatar', 'old-avatar');
  storage.set('user_nickname', 'old-name');
  storage.set('checkout_payload', { sku_id: 1 });
  store.logout();
  assert.equal(store.isLoggedIn, false);
  for (const key of ['token', 'member_info', 'user_avatar', 'user_nickname', 'checkout_payload']) {
    assert.equal(storage.has(key), false, key + ' must be cleared');
  }

  requestHandler = async (options) => options.url.endsWith('/auth/wechat-login')
    ? authReply(false)
    : { statusCode: 200, data: { code: 1, data: [] } };
  const repeat = store.loginWithWechat();
  loginCallback.success({ code: 'wx-code-2' });
  assert.equal((await repeat).is_new_member, false);
  assert.equal(store.isLoggedIn, true);
}

console.log('WeChat one-tap login: failure and concurrent tap');
{
  const store = freshStore();
  const pending = store.loginWithWechat();
  await assert.rejects(store.loginWithWechat(), /登录进行中/);
  loginCallback.fail({ errMsg: 'cancel' });
  await assert.rejects(pending, /微信登录失败/);
  assert.equal(store.loading, false);
  assert.equal(store.isLoggedIn, false);
  assert.equal(requests.length, 0);
}

console.log('WeChat one-tap login: logout discards pending code and API response');
{
  const store = freshStore();
  const pendingCode = store.loginWithWechat();
  const callback = loginCallback;
  store.logout();
  callback.success({ code: 'stale-code' });
  await assert.rejects(pendingCode, /登录已取消/);
  assert.equal(requests.length, 0);
  assert.equal(store.isLoggedIn, false);

  let finishRequest;
  requestHandler = (options) => options.url.endsWith('/auth/wechat-login')
    ? new Promise((resolve) => { finishRequest = resolve; })
    : Promise.resolve({ statusCode: 200, data: { code: 1, data: [] } });
  const pendingResponse = store.loginWithWechat();
  loginCallback.success({ code: 'second-code' });
  await Promise.resolve();
  await Promise.resolve();
  assert.equal(typeof finishRequest, 'function');
  store.logout();
  finishRequest(authReply());
  await assert.rejects(pendingResponse, /登录已取消/);
  assert.equal(store.isLoggedIn, false);
  assert.equal(storage.has('token'), false);
}

console.log('All WeChat one-tap login Store tests passed');
