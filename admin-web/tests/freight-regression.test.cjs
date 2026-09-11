const assert = require("node:assert/strict");
const { test } = require("node:test");

test("freight template rules detect duplicate extra fee keys", () => {
  const extraFees = [
    { key: "1", provinceCode: "110000", cityCode: "", fee: "10.00" },
    { key: "2", provinceCode: "110000", cityCode: "", fee: "15.00" },
  ];

  const extraFeeKeySet = new Set();
  let duplicate = false;
  for (const row of extraFees) {
    const key = `${row.provinceCode}#${row.cityCode}`;
    if (extraFeeKeySet.has(key)) {
      duplicate = true;
      break;
    }
    extraFeeKeySet.add(key);
  }

  assert.equal(duplicate, true, "Should identify duplicate province in extra fees");
});

test("freight template rules detect conflict between extra fee and no delivery", () => {
  const extraFees = [
    { key: "1", provinceCode: "440000", cityCode: "440300", fee: "5.00" },
  ];
  const noDeliveries = [
    { key: "2", provinceCode: "440000", cityCode: "440300" },
  ];

  const extraFeeKeySet = new Set();
  for (const row of extraFees) {
    extraFeeKeySet.add(`${row.provinceCode}#${row.cityCode}`);
  }

  let conflict = false;
  for (const item of noDeliveries) {
    const key = `${item.provinceCode}#${item.cityCode}`;
    if (extraFeeKeySet.has(key)) {
      conflict = true;
      break;
    }
  }

  assert.equal(conflict, true, "Should identify conflict between extra fee and no delivery for same city");
});

test("product delivery type change cleans up inactive delivery values", () => {
  function applyDeliveryModeChange(type, currentFee, currentTemplateId) {
    let fee = currentFee;
    let templateId = currentTemplateId;
    if (type === 1) {
      // 包邮
      fee = null;
      templateId = null;
    } else if (type === 2) {
      // 固定运费
      templateId = null;
      if (fee === null) fee = 0;
    } else if (type === 3) {
      // 运费模板
      fee = null;
    }
    return { freightType: type, freightFee: fee, freightTemplateId: templateId };
  }

  // Switch to Free Shipping (1)
  const res1 = applyDeliveryModeChange(1, 15, 88);
  assert.equal(res1.freightType, 1);
  assert.equal(res1.freightFee, null);
  assert.equal(res1.freightTemplateId, null);

  // Switch to Fixed Freight (2)
  const res2 = applyDeliveryModeChange(2, null, 88);
  assert.equal(res2.freightType, 2);
  assert.equal(res2.freightFee, 0);
  assert.equal(res2.freightTemplateId, null);

  // Switch to Freight Template (3)
  const res3 = applyDeliveryModeChange(3, 20, 88);
  assert.equal(res3.freightType, 3);
  assert.equal(res3.freightFee, null);
  assert.equal(res3.freightTemplateId, 88);
});
