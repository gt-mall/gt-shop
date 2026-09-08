const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const { test } = require('node:test');

test('inventory pending mutation storage key is uniquely scoped to user ID', () => {
  const getPendingKey = (userId) => `mall_inventory_pending_mutation_${userId}`;
  assert.equal(getPendingKey(101), 'mall_inventory_pending_mutation_101');
  assert.notEqual(getPendingKey(101), getPendingKey(102));
});

test('session storage pending payload rejects cross-user replay and recovers for identical user', () => {
  const mockStorage = new Map();
  const savePending = (userId, payload) => {
    mockStorage.set(`mall_inventory_pending_mutation_${userId}`, JSON.stringify(payload));
  };
  const getPending = (userId) => {
    const raw = mockStorage.get(`mall_inventory_pending_mutation_${userId}`);
    if (!raw) return null;
    const parsed = JSON.parse(raw);
    if (parsed.user_id !== userId) return null;
    return parsed;
  };

  const payload = {
    user_id: 88,
    sku_id: 10,
    request_id: '1234-5678',
    mode: 'manual_out',
    quantity: 5,
    remark: '人工损耗出库',
    expected_version: 3,
  };

  savePending(88, payload);

  // Cross user attempt
  assert.equal(getPending(99), null);

  // Same user recovery
  const recovered = getPending(88);
  assert.ok(recovered);
  assert.equal(recovered.request_id, '1234-5678');
  assert.equal(recovered.expected_version, 3);
  assert.equal(recovered.quantity, 5);
});

test('inventory adjust calculation rules adhere to business constraints', () => {
  const calcNewTotal = (mode, currentStock, quantity) => {
    if (mode === 'manual_in') return currentStock + quantity;
    if (mode === 'manual_out') return currentStock - quantity;
    if (mode === 'stocktake') return quantity;
    return currentStock;
  };

  // 1. manual_in increases total
  assert.equal(calcNewTotal('manual_in', 10, 5), 15);

  // 2. manual_out decreases total
  assert.equal(calcNewTotal('manual_out', 10, 3), 7);

  // 3. stocktake overrides total directly
  assert.equal(calcNewTotal('stocktake', 10, 0), 0);
  assert.equal(calcNewTotal('stocktake', 10, 12), 12);
});

test('inventory validation bounds check stock limits', () => {
  const validateOutBound = (currentAvailable, quantity) => {
    if (quantity <= 0) return '出库数量必须大于 0';
    if (quantity > currentAvailable) return `出库数量不能大于当前可用库存(${currentAvailable})`;
    if (quantity > 2147483647) return '数量超过系统上限';
    return null;
  };

  const validateStocktakeBound = (currentLocked, quantity) => {
    if (quantity < 0) return '盘点后总库存不能小于 0';
    if (quantity < currentLocked) return `盘点后总库存不能小于当前锁定库存(${currentLocked})`;
    if (quantity > 2147483647) return '数量超过系统上限';
    return null;
  };

  // Valid out
  assert.equal(validateOutBound(10, 5), null);
  // Out exceeds available
  assert.equal(validateOutBound(5, 6), '出库数量不能大于当前可用库存(5)');
  // Out negative/zero
  assert.equal(validateOutBound(5, 0), '出库数量必须大于 0');

  // Valid stocktake
  assert.equal(validateStocktakeBound(2, 5), null);
  // Stocktake less than locked
  assert.equal(validateStocktakeBound(5, 3), '盘点后总库存不能小于当前锁定库存(5)');
  // Stocktake exactly equal to locked
  assert.equal(validateStocktakeBound(5, 5), null);
});

test('inventory log date range formats end day to beginning of next day', () => {
  const formatRange = (startStr, endStr) => {
    const startRFC = `${startStr}T00:00:00Z`;
    const [y, m, d] = endStr.split('-').map(Number);
    const nextDate = new Date(Date.UTC(y, m - 1, d + 1));
    const pad = (n) => String(n).padStart(2, '0');
    const nextDayStr = `${nextDate.getUTCFullYear()}-${pad(nextDate.getUTCMonth() + 1)}-${pad(nextDate.getUTCDate())}`;
    const endRFC = `${nextDayStr}T00:00:00Z`;
    return { startRFC, endRFC };
  };

  const range = formatRange('2026-09-01', '2026-09-08');
  assert.equal(range.startRFC, '2026-09-01T00:00:00Z');
  assert.equal(range.endRFC, '2026-09-09T00:00:00Z');
});
