import assert from 'node:assert/strict';
import {
  parseDateToMs,
  useOrderCountdown,
  syncOrderCountdownClock,
  resetOrderCountdownTicker,
} from '../src/composables/useOrderCountdown.ts';
import { formatCountdown } from '../src/utils/date.ts';

console.log('--- Starting Order Countdown & Clock Skew Tests ---');

// ----------------------------------------------------
// 测试 1: parseDateToMs 时间解析与边界防崩
// ----------------------------------------------------
console.log('Test 1: parseDateToMs 安全解析');
{
  // RFC3339 字符串
  const rfcStr = '2026-09-20T12:00:00Z';
  assert.equal(parseDateToMs(rfcStr), new Date(rfcStr).getTime());

  // 带时区偏移
  const tzStr = '2026-09-20T12:00:00+08:00';
  assert.equal(parseDateToMs(tzStr), new Date(tzStr).getTime());

  // iOS 风格破折号替代
  const dashStr = '2026-09-20 12:00:00';
  assert.ok(parseDateToMs(dashStr) > 0);

  // 10位秒级时间戳
  assert.equal(parseDateToMs(1726833600), 1726833600000);

  // 13位毫秒级时间戳
  assert.equal(parseDateToMs(1726833600000), 1726833600000);

  // Date 对象
  const d = new Date('2026-09-20T12:00:00Z');
  assert.equal(parseDateToMs(d), d.getTime());

  // 异常/非法值输入返回 0，严禁产生 NaN
  assert.equal(parseDateToMs(null), 0);
  assert.equal(parseDateToMs(undefined), 0);
  assert.equal(parseDateToMs(''), 0);
  assert.equal(parseDateToMs('invalid-date-string'), 0);
  console.log('-> Test 1 PASSED');
}

// ----------------------------------------------------
// 测试 2: formatCountdown 倒计时分秒格式化
// ----------------------------------------------------
console.log('Test 2: formatCountdown 格式化与零/负值防御');
{
  // 30分钟整 (1800秒)
  const cd1800 = formatCountdown(1800);
  assert.equal(cd1800.minutes, '30');
  assert.equal(cd1800.seconds, '00');
  assert.equal(cd1800.isEnded, false);

  // 29分59秒
  const cd1799 = formatCountdown(1799);
  assert.equal(cd1799.minutes, '29');
  assert.equal(cd1799.seconds, '59');
  assert.equal(cd1799.isEnded, false);

  // 0 秒 (归零)
  const cd0 = formatCountdown(0);
  assert.equal(cd0.minutes, '00');
  assert.equal(cd0.seconds, '00');
  assert.equal(cd0.isEnded, true);

  // 负秒 (超时)
  const cdNeg = formatCountdown(-15);
  assert.equal(cdNeg.minutes, '00');
  assert.equal(cdNeg.seconds, '00');
  assert.equal(cdNeg.isEnded, true);
  console.log('-> Test 2 PASSED');
}

// ----------------------------------------------------
// 测试 3: useOrderCountdown 基础状态与有效性
// ----------------------------------------------------
console.log('Test 3: useOrderCountdown 状态与截止时间判断');
{
  const now = Date.now();
  const futureExpires = new Date(now + 600 * 1000).toISOString(); // 10分钟后
  const currentServerTime = new Date(now).toISOString();

  // A. 正常待付款状态 (status: 10)
  const { hasCountdown, isExpired, remainingSeconds, countdown } = useOrderCountdown(() => ({
    expiresAt: futureExpires,
    serverTime: currentServerTime,
    status: 10,
  }));

  assert.equal(hasCountdown.value, true, 'status=10且有有效截止时间应具有倒计时');
  assert.equal(isExpired.value, false, '未到期不应为 expired');
  assert.ok(remainingSeconds.value >= 595 && remainingSeconds.value <= 605, '剩余秒数应在 600 秒左右');
  assert.ok(countdown.value.minutes.length === 2 && !isNaN(Number(countdown.value.minutes)));

  // B. 非待付款状态 (status: 20 已付款)
  const paidCountdown = useOrderCountdown(() => ({
    expiresAt: futureExpires,
    serverTime: currentServerTime,
    status: 20,
  }));
  assert.equal(paidCountdown.hasCountdown.value, false, '已付款状态不展示倒计时');
  assert.equal(paidCountdown.isExpired.value, false);
  assert.equal(paidCountdown.remainingSeconds.value, 0);

  // C. 缺少截止时间
  const noExpiryCountdown = useOrderCountdown(() => ({
    expiresAt: null,
    serverTime: currentServerTime,
    status: 10,
  }));
  assert.equal(noExpiryCountdown.hasCountdown.value, false, '无截止时间不展示倒计时');
  assert.equal(noExpiryCountdown.remainingSeconds.value, 0);
  assert.equal(noExpiryCountdown.countdown.value.minutes, '00');
  assert.equal(noExpiryCountdown.countdown.value.seconds, '00');
  console.log('-> Test 3 PASSED');
}

// ----------------------------------------------------
// 测试 4: 设备时钟偏差（Clock Skew）校准能力
// ----------------------------------------------------
console.log('Test 4: 设备本地时钟快/慢时的服务端校准');
{
  const serverNowMs = Date.parse('2026-09-20T12:00:00Z');
  const serverExpiresMs = Date.parse('2026-09-20T12:30:00Z'); // 30 分钟后到期
  const serverTimeStr = '2026-09-20T12:00:00Z';
  const expiresAtStr = '2026-09-20T12:30:00Z';

  // 无论设备当前本地 Date.now() 是什么（即使比服务端快或慢 10 小时），
  // 经 server_time 校准后，从 serverNow 到 expires 必须精确是 1800 秒（30分钟）！
  const { remainingSeconds, countdown } = useOrderCountdown(() => ({
    expiresAt: expiresAtStr,
    serverTime: serverTimeStr,
    status: 10,
  }));

  assert.ok(
    remainingSeconds.value >= 1798 && remainingSeconds.value <= 1800,
    `校准后剩余秒数应约为 1800 秒，实际: ${remainingSeconds.value}`
  );
  assert.equal(countdown.value.minutes, '30');
  assert.equal(countdown.value.seconds, '00');
  console.log('-> Test 4 PASSED');
}

// ----------------------------------------------------
// 测试 5: 订单超时自动触发 onTimeout 且防重复重入
// ----------------------------------------------------
console.log('Test 5: 订单超时 onTimeout 单次触发');
{
  let timeoutCount = 0;
  const pastExpires = new Date(Date.now() - 5000).toISOString(); // 5秒前已到期

  const { isExpired, remainingSeconds } = useOrderCountdown(() => ({
    expiresAt: pastExpires,
    serverTime: new Date().toISOString(),
    status: 10,
    onTimeout: () => {
      timeoutCount++;
    },
  }));

  assert.equal(isExpired.value, true, '过期订单 isExpired 应为 true');
  assert.equal(remainingSeconds.value, 0, '过期订单 remainingSeconds 应为 0');
  assert.equal(timeoutCount, 1, 'onTimeout 回调应只触发 1 次');

  // 手动调用时钟更新，不应导致 onTimeout 二次触发
  syncOrderCountdownClock();
  assert.equal(timeoutCount, 1, '时钟 tick 不应造成 onTimeout 二次触发');
  console.log('-> Test 5 PASSED');
}

resetOrderCountdownTicker();
console.log('ALL ORDER COUNTDOWN TESTS PASSED SUCCESSFULLY!');
process.exit(0);
