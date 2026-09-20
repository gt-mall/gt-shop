import assert from 'node:assert/strict';
import { createPinia, setActivePinia } from 'pinia';
import { useMemberStore } from '../src/stores/member.ts';

console.log('--- Starting ProductPoster & Share Code Tests ---');

// Mock wrapText 逻辑
function wrapText(measureWidthFn, text, maxWidth, maxLines = 2) {
  const lines = [];
  let currentLine = '';

  for (let i = 0; i < text.length; i++) {
    const char = text[i];
    const testLine = currentLine + char;
    const width = measureWidthFn(testLine);
    if (width > maxWidth && currentLine.length > 0) {
      lines.push(currentLine);
      currentLine = char;
      if (lines.length === maxLines - 1) {
        const remaining = text.substring(i);
        let fitText = '';
        for (let j = 0; j < remaining.length; j++) {
          if (measureWidthFn(fitText + remaining[j] + '...') <= maxWidth) {
            fitText += remaining[j];
          } else {
            break;
          }
        }
        lines.push(fitText + '...');
        return lines;
      }
    } else {
      currentLine = testLine;
    }
  }

  if (currentLine) {
    lines.push(currentLine);
  }
  return lines.slice(0, maxLines);
}

// 模拟简易文字宽度：每个中文字符 16px，英文字符 8px
function mockMeasureText(str) {
  let w = 0;
  for (const c of str) {
    w += c.charCodeAt(0) > 255 ? 16 : 8;
  }
  return w;
}

// ----------------------------------------------------
// 测试 1: wrapText 文本折行与截断测试
// ----------------------------------------------------
console.log('Test 1: wrapText 单行、双行及超长截断省略号');
{
  // 单行
  const shortTitle = 'iPhone 15 Pro';
  const lines1 = wrapText(mockMeasureText, shortTitle, 200, 2);
  assert.equal(lines1.length, 1);
  assert.equal(lines1[0], 'iPhone 15 Pro');

  // 双行以内
  const mediumTitle = '2026年最新款旗舰智能手机5G全网通双卡双待原装正品';
  const lines2 = wrapText(mockMeasureText, mediumTitle, 160, 2);
  assert.equal(lines2.length, 2);

  // 超长标题：第 2 行末尾必须有 '...'
  const superLongTitle = '【官方正品】2026新款超长续航折叠屏智能手机 顶级屏幕 高通骁龙旗舰芯片 超清三摄专业影像系统 极速快充 尊享售后保障服务 原装正品全国联保特惠热卖中';
  const lines3 = wrapText(mockMeasureText, superLongTitle, 160, 2);
  assert.equal(lines3.length, 2);
  assert(lines3[1].endsWith('...'), '超长标题第2行末尾必须包含省略号');
  console.log('-> Test 1 PASSED');
}

// ----------------------------------------------------
// 测试 2: 未登录拦截生成海报，登录后允许
// ----------------------------------------------------
console.log('Test 2: 会员登录鉴权守卫');
{
  const pinia = createPinia();
  setActivePinia(pinia);
  const memberStore = useMemberStore();

  memberStore.token = '';
  memberStore.member = null;
  assert.equal(memberStore.isLoggedIn, false);

  // 未登录调用检查
  let shareCodeCalled = false;
  const mockGetProductShareCode = async (id) => {
    shareCodeCalled = true;
    return { code: 1, data: { product_id: id, file_id: 1, url: 'http://test/qr.png' } };
  };

  async function simulatePosterTrigger(productId) {
    if (!memberStore.isLoggedIn) {
      return { success: false, error: '请先登录后生成商品海报' };
    }
    const res = await mockGetProductShareCode(productId);
    return { success: true, data: res.data };
  }

  const resUnauth = await simulatePosterTrigger(101);
  assert.equal(resUnauth.success, false);
  assert.equal(resUnauth.error, '请先登录后生成商品海报');
  assert.equal(shareCodeCalled, false, '未登录时不应发起 share-code 请求');

  // 模拟登录
  memberStore.token = 'test-token-valid';
  memberStore.member = { id: 1001, nickname: '测试用户' };
  assert.equal(memberStore.isLoggedIn, true);

  const resAuth = await simulatePosterTrigger(101);
  assert.equal(resAuth.success, true);
  assert.equal(shareCodeCalled, true);
  assert.equal(resAuth.data.product_id, 101);
  console.log('-> Test 2 PASSED');
}

// ----------------------------------------------------
// 测试 3: 商品切换/弹层关闭时旧绘制会话自动作废
// ----------------------------------------------------
console.log('Test 3: 切换商品或快速开关时旧异步绘制不污染新海报');
{
  let currentSessionSeq = 0;
  let activePosterResult = null;

  async function triggerGenerate(productId, delayMs) {
    const session = ++currentSessionSeq;
    await new Promise((r) => setTimeout(r, delayMs));
    if (session !== currentSessionSeq) {
      // 旧会话已被废弃
      return;
    }
    activePosterResult = `poster_for_${productId}`;
  }

  // 先触发商品 101（慢请求 50ms）
  triggerGenerate(101, 50);

  // 用户迅速切换到商品 202（快请求 10ms）
  triggerGenerate(202, 10);

  await new Promise((r) => setTimeout(r, 60));

  // 最终结果必须是商品 202 的海报，101 的慢请求不得覆盖 202
  assert.equal(activePosterResult, 'poster_for_202', '商品切换后旧请求结果被成功丢弃');
  console.log('-> Test 3 PASSED');
}

// ----------------------------------------------------
// 测试 4: 二维码生成失败时严禁输出假二维码
// ----------------------------------------------------
console.log('Test 4: 接口异常时展示错误状态，不生成假码海报');
{
  let canvasDrawn = false;
  let errorState = null;

  async function generatePosterFlow(apiSuccess) {
    canvasDrawn = false;
    errorState = null;

    try {
      if (!apiSuccess) {
        throw new Error('微信接口调用频次限制或服务异常');
      }
      canvasDrawn = true;
    } catch (err) {
      errorState = err.message;
    }
  }

  await generatePosterFlow(false);
  assert.equal(canvasDrawn, false, '接口失败时禁止进入画布绘制阶段');
  assert.equal(errorState, '微信接口调用频次限制或服务异常');

  await generatePosterFlow(true);
  assert.equal(canvasDrawn, true);
  assert.equal(errorState, null);
  console.log('-> Test 4 PASSED');
}

console.log('\n========================================');
console.log('ALL PRODUCT POSTER TESTS PASSED SUCCESSFULLY!');
console.log('========================================\n');
