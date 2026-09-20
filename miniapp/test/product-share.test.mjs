import assert from 'node:assert/strict';

// 引入生产源码
const {
  parsePositiveInteger,
  parseProductEntry,
  buildProductShare,
  buildH5ShareUrl,
  MAX_SCENE_LENGTH,
} = await import('../src/utils/product-share.ts');

console.log('--- Starting Product Share & Entry Parsing Tests ---');

// ----------------------------------------------------
// 测试 1: parsePositiveInteger 安全整数校验
// ----------------------------------------------------
console.log('Test 1: parsePositiveInteger 安全整数校验');
{
  assert.equal(parsePositiveInteger(1), 1);
  assert.equal(parsePositiveInteger(100), 100);
  assert.equal(parsePositiveInteger(99999999), 99999999);
  assert.equal(parsePositiveInteger('1'), 1);
  assert.equal(parsePositiveInteger(' 100 '), 100);
  assert.equal(parsePositiveInteger('99999999'), 99999999);

  // 非法输入防御
  assert.equal(parsePositiveInteger(0), null);
  assert.equal(parsePositiveInteger(-5), null);
  assert.equal(parsePositiveInteger(1.5), null);
  assert.equal(parsePositiveInteger('0'), null);
  assert.equal(parsePositiveInteger('-5'), null);
  assert.equal(parsePositiveInteger('012'), null); // 禁止前导 0
  assert.equal(parsePositiveInteger('12a'), null);
  assert.equal(parsePositiveInteger('abc'), null);
  assert.equal(parsePositiveInteger(''), null);
  assert.equal(parsePositiveInteger('   '), null);
  assert.equal(parsePositiveInteger(null), null);
  assert.equal(parsePositiveInteger(undefined), null);
}

// ----------------------------------------------------
// 测试 2: parseProductEntry 正常与边界解析
// ----------------------------------------------------
console.log('Test 2: parseProductEntry 正常与边界解析');
{
  // 1. 普通合法 id
  const res1 = parseProductEntry({ id: 101 });
  assert.deepEqual(res1, { valid: true, productId: 101 });

  const res2 = parseProductEntry({ id: '202' });
  assert.deepEqual(res2, { valid: true, productId: 202 });

  // 2. 扫码约定短场景 p=<id>
  const res3 = parseProductEntry({ scene: 'p=303' });
  assert.deepEqual(res3, { valid: true, productId: 303 });

  // 3. 编码后的场景 p%3D404
  const res4 = parseProductEntry({ scene: 'p%3D404' });
  assert.deepEqual(res4, { valid: true, productId: 404 });

  // 4. 多参数场景 p=505&from=poster
  const res5 = parseProductEntry({ scene: 'p=505&from=poster' });
  assert.deepEqual(res5, { valid: true, productId: 505 });

  // 5. 别名 id=606
  const res6 = parseProductEntry({ scene: 'id=606' });
  assert.deepEqual(res6, { valid: true, productId: 606 });

  // 6. id 与 scene 同时提供且一致
  const res7 = parseProductEntry({ id: 707, scene: 'p=707' });
  assert.deepEqual(res7, { valid: true, productId: 707 });

  // 7. 微信 App 启动场景值（数字 1001）不作为商品 scene，且有合法 id
  const res8 = parseProductEntry({ id: 808, scene: 1001 });
  assert.deepEqual(res8, { valid: true, productId: 808 });
}

// ----------------------------------------------------
// 测试 3: parseProductEntry 异常、冲突与拦截
// ----------------------------------------------------
console.log('Test 3: parseProductEntry 异常、冲突与拦截');
{
  // 1. 参数缺失
  assert.equal(parseProductEntry(null).valid, false);
  assert.equal(parseProductEntry({}).valid, false);
  assert.equal(parseProductEntry({ id: '' }).valid, false);

  // 2. 只有微信数字启动场景，无商品 ID
  const resLaunch = parseProductEntry({ scene: 1001 });
  assert.equal(resLaunch.valid, false);
  assert.equal(resLaunch.error, '缺少商品ID参数');

  // 3. 参数冲突（id 与 scene 商品不一致）
  const resConflict = parseProductEntry({ id: 101, scene: 'p=202' });
  assert.equal(resConflict.valid, false);
  assert.equal(resConflict.error, '商品参数冲突');

  // 4. 非法 id
  const resBadId = parseProductEntry({ id: 'bad_id' });
  assert.equal(resBadId.valid, false);
  assert.equal(resBadId.error, '商品ID参数非法');

  // 5. 非法 scene 内容
  const resBadSceneVal = parseProductEntry({ scene: 'p=bad_val' });
  assert.equal(resBadSceneVal.valid, false);
  assert.equal(resBadSceneVal.error, '场景中的商品ID非法');

  // 6. 无商品标识的 scene
  const resNoKey = parseProductEntry({ scene: 'channel=wechat' });
  assert.equal(resNoKey.valid, false);
  assert.equal(resNoKey.error, '场景中未找到有效商品标识');

  // 7. 解码损坏的 scene
  const resCorrupted = parseProductEntry({ scene: '%E0%A4%A' });
  assert.equal(resCorrupted.valid, false);
  assert.equal(resCorrupted.error, '场景参数解码失败');

  // 8. 场景超长
  const resTooLong = parseProductEntry({ scene: 'p=' + '1'.repeat(MAX_SCENE_LENGTH + 1) });
  assert.equal(resTooLong.valid, false);
  assert.equal(resTooLong.error, '场景参数超出长度限制');
}

// ----------------------------------------------------
// 测试 4: buildProductShare 分享载荷拼装与下架保护
// ----------------------------------------------------
console.log('Test 4: buildProductShare 分享载荷拼装与下架保护');
{
  // 1. 正常商品
  const product = {
    id: 123,
    name: '经典复古机械键盘 87键',
    main_image_url: 'https://cdn.example.com/keyboard.jpg',
  };
  const shareData = buildProductShare({
    product,
    shopName: '我的测试商城',
  });
  assert.ok(shareData);
  assert.equal(shareData.title, '经典复古机械键盘 87键');
  assert.equal(shareData.path, '/pages/product/detail?id=123');
  assert.equal(shareData.query, 'id=123');
  assert.equal(shareData.imageUrl, 'https://cdn.example.com/keyboard.jpg');

  // 2. 标题缺省回退
  const productNoName = {
    id: 123,
    name: '   ',
    main_image_url: '',
  };
  const shareData2 = buildProductShare({
    product: productNoName,
    shopName: '官方旗舰店',
    defaultImageUrl: 'https://cdn.example.com/default.png',
  });
  assert.ok(shareData2);
  assert.equal(shareData2.title, '官方旗舰店');
  assert.equal(shareData2.imageUrl, 'https://cdn.example.com/default.png');

  // 3. 无效/未加载/下架商品，拒绝伪造分享
  assert.equal(buildProductShare({ product: null }), null);
  assert.equal(buildProductShare({ product: { id: 0, name: '下架商品' } }), null);
  assert.equal(buildProductShare({ product: { id: -1, name: '非法商品' } }), null);
}

// ----------------------------------------------------
// 测试 5: buildH5ShareUrl 动态链接生成
// ----------------------------------------------------
console.log('Test 5: buildH5ShareUrl 动态链接生成');
{
  // 1. Hash 路由模式
  const locHash = {
    origin: 'https://mall.example.com',
    pathname: '/shop/',
    hash: '#/pages/product/detail?id=999',
  };
  const url1 = buildH5ShareUrl(123, locHash);
  assert.equal(url1, 'https://mall.example.com/shop/#/pages/product/detail?id=123');

  // 2. History 路由模式
  const locHistory = {
    origin: 'https://shop.org',
    pathname: '/h5/pages/product/detail',
    hash: '',
  };
  const url2 = buildH5ShareUrl(456, locHistory);
  assert.equal(url2, 'https://shop.org/h5/pages/product/detail?id=456');

  // 3. 非法 ID
  assert.equal(buildH5ShareUrl(0, locHash), '');
  assert.equal(buildH5ShareUrl(-1, locHash), '');
}

console.log('All Product Share & Entry Parsing tests passed successfully!');
