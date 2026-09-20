import assert from 'node:assert/strict';
import fs from 'node:fs';
import vm from 'node:vm';
import ts from 'typescript';
import * as vue from 'vue';
import { parse, compileScript } from 'vue/compiler-sfc';
import { createPinia, setActivePinia } from 'pinia';
import { useCartManage } from '../src/composables/useCartManage.ts';
import { useMemberStore } from '../src/stores/member.ts';
import * as utils from '../src/utils/index.ts';

// 挂载真实 SFC 的 setup，替换的仅是视图宿主、UniApp 生命周期和网络边界。
const storage = new Map();
let route = 'pages/product/detail';
let options = { id: '40' };
let navigation = '';
let hide;
globalThis.uni = {
  getStorageSync: (key) => storage.get(key),
  setStorageSync: (key, value) => storage.set(key, value),
  removeStorageSync: (key) => storage.delete(key),
  setTabBarBadge() {}, removeTabBarBadge() {}, showToast() {},
  navigateTo({ url }) { navigation = url; },
};
const document = { body: { style: { overflow: 'auto' } }, documentElement: { style: { overflow: 'scroll' } } };
const { descriptor } = parse(fs.readFileSync(new URL('../src/components/CartDrawer.vue', import.meta.url), 'utf8'));
const script = compileScript(descriptor, { id: 'cart-drawer-test' });
const output = ts.transpileModule(script.content, { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 } }).outputText;
const context = vm.createContext({
  exports: {}, document, setTimeout, clearTimeout,
  getCurrentPages: () => [{ route, options }],
  uni: globalThis.uni,
  require(name) {
    if (name === 'vue') return vue;
    if (name === '@dcloudio/uni-app') return { onHide: (callback) => { hide = callback; } };
    if (name === './CartItemCard.vue') return {};
    if (name === './Skeleton.vue') return {};
    if (name === '@/composables/useCartManage') return { useCartManage };
    if (name === '@/composables/useNetwork') return { useNetwork: () => ({}) };
    if (name === '@/stores/member') return { useMemberStore };
    if (name === '@/utils') return utils;
    throw new Error(`Unexpected import ${name}`);
  },
});
vm.runInContext(output, context);
const component = context.exports.default;
component.render = () => null;
const renderer = vue.createRenderer({
  createElement: () => ({}), createText: () => ({}), createComment: () => ({}),
  insert() {}, remove() {}, setText() {}, setElementText() {}, patchProp() {},
  parentNode: () => null, nextSibling: () => null,
});
const delay = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
const pinia = createPinia();
setActivePinia(pinia);
const model = vue.ref(true);
const locks = [];
let state;
const app = renderer.createApp({
  render() {
    return vue.h(component, {
      ref: (instance) => { if (instance) state = instance.$.setupState; },
      modelValue: model.value,
      'onUpdate:modelValue': (value) => { model.value = value; },
      onScrollLock: (value) => locks.push(value),
    });
  },
});
app.use(pinia);
app.mount({});
await vue.nextTick();
assert.equal(state.rendered, true);
assert.equal(document.body.style.overflow, 'hidden');

// 进入动画排队时关闭，回调不能将它重新打开。
model.value = false;
await vue.nextTick();
await delay(60);
assert.equal(state.visible, false);
// 退出动画尚未结束时重开，旧关闭计时器不能卸载新抽屉。
model.value = true;
await vue.nextTick();
await delay(350);
assert.equal(state.rendered, true);
assert.equal(state.visible, true);

state.handleClose();
await delay(350);
assert.equal(state.rendered, false);
assert.equal(document.body.style.overflow, 'auto');
assert.equal(document.documentElement.style.overflow, 'scroll');

// 数字前缀不能视为同一商品；相同商品但参数顺序不同仍能识别。
state.handleClickProduct(4);
assert.equal(navigation, '/pages/product/detail?id=4');
navigation = '';
options = { source: 'cart', id: '4' };
state.handleClickProduct(4);
assert.equal(navigation, '');

// 认证在抽屉打开期间失效时，UI 必须立即切换到未登录。
storage.set('token', 'drawer-session');
useMemberStore().token = 'drawer-session';
await vue.nextTick();
assert.equal(state.isLoggedIn, true);
storage.delete('token');
useMemberStore().token = '';
await vue.nextTick();
assert.equal(state.isLoggedIn, false);

model.value = true;
await vue.nextTick();
hide();
assert.equal(document.body.style.overflow, 'auto', '页面隐藏释放滚动锁');
app.unmount();
await delay(60);
assert.equal(document.body.style.overflow, 'auto');
assert.equal(locks.at(-1), false);
console.log('PASS actual CartDrawer setup: animation races, scroll cleanup, exact navigation and reactive auth');
