import assert from 'node:assert/strict';

console.log('--- Starting Product Review Client Logic & Eligibility Tests ---');

// ----------------------------------------------------
// 测试 1: 订单商品明细评价资格判定与状态分支
// ----------------------------------------------------
console.log('Test 1: 订单明细评价资格判定与状态分支');
{
  function evaluateItemReviewAction(orderStatus, item) {
    if (orderStatus !== 40) {
      return { action: 'none', reason: '订单未完成不可评价' };
    }
    if (item.can_review) {
      return { action: 'create', target: `/pages/review/create?order_id=${item.order_id}&order_item_id=${item.id}` };
    }
    if (item.review_id) {
      return {
        action: 'detail',
        target: `/pages/review/detail?id=${item.review_id}`,
        status: item.review_status,
      };
    }
    return {
      action: 'blocked',
      reason: item.review_block_reason || '当前明细不可评价',
    };
  }

  // 1. 已完成订单，未评价且符合资格
  const eligibleItem = {
    id: 101,
    order_id: 1,
    can_review: true,
  };
  const res1 = evaluateItemReviewAction(40, eligibleItem);
  assert.equal(res1.action, 'create');
  assert.equal(res1.target, '/pages/review/create?order_id=1&order_item_id=101');

  // 2. 已完成订单，已评价 (审核中)
  const reviewedItem = {
    id: 102,
    order_id: 1,
    can_review: false,
    review_id: 55,
    review_status: 10,
  };
  const res2 = evaluateItemReviewAction(40, reviewedItem);
  assert.equal(res2.action, 'detail');
  assert.equal(res2.target, '/pages/review/detail?id=55');
  assert.equal(res2.status, 10);

  // 3. 已完成订单，不可评价原因（如已退款/超过评价期）
  const blockedItem = {
    id: 103,
    order_id: 1,
    can_review: false,
    review_block_reason: '已发生全额退款，不可评价',
  };
  const res3 = evaluateItemReviewAction(40, blockedItem);
  assert.equal(res3.action, 'blocked');
  assert.equal(res3.reason, '已发生全额退款，不可评价');

  // 4. 未完成订单 (如待发货 status 20)，不可评价
  const pendingOrderRes = evaluateItemReviewAction(20, eligibleItem);
  assert.equal(pendingOrderRes.action, 'none');

  console.log('-> Test 1 PASSED');
}

// ----------------------------------------------------
// 测试 2: 评价表单校验与幂等 request_id 保持规则
// ----------------------------------------------------
console.log('Test 2: 评价表单校验与幂等 request_id 保持规则');
{
  function validateReviewForm({ rating, content, images }) {
    if (!rating || rating < 1 || rating > 5) {
      return { valid: false, message: '请选择1-5星评分' };
    }
    const trimmed = (content || '').trim();
    if (trimmed.length === 0) {
      return { valid: false, message: '请填写评价内容' };
    }
    if (trimmed.length > 500) {
      return { valid: false, message: '评价内容不能超过500字' };
    }
    if (images && images.some((img) => img.status === 'uploading')) {
      return { valid: false, message: '图片上传中，请稍候' };
    }
    if (images && images.some((img) => img.status === 'error')) {
      return { valid: false, message: '存在上传失败的图片，请重试或删除' };
    }
    return { valid: true };
  }

  // 校验评分范围
  assert.equal(validateReviewForm({ rating: 0, content: '好评', images: [] }).valid, false);
  assert.equal(validateReviewForm({ rating: 6, content: '好评', images: [] }).valid, false);
  assert.equal(validateReviewForm({ rating: 5, content: '好评', images: [] }).valid, true);

  // 校验正文字数
  assert.equal(validateReviewForm({ rating: 5, content: '   ', images: [] }).valid, false);
  assert.equal(validateReviewForm({ rating: 5, content: 'a'.repeat(501), images: [] }).valid, false);
  assert.equal(validateReviewForm({ rating: 5, content: 'a'.repeat(500), images: [] }).valid, true);

  // 校验图片上传状态
  assert.equal(
    validateReviewForm({
      rating: 5,
      content: '很不错',
      images: [{ status: 'uploading' }],
    }).valid,
    false
  );
  assert.equal(
    validateReviewForm({
      rating: 5,
      content: '很不错',
      images: [{ status: 'error' }],
    }).valid,
    false
  );
  assert.equal(
    validateReviewForm({
      rating: 5,
      content: '很不错',
      images: [{ status: 'success', fileId: 88 }],
    }).valid,
    true
  );

  // 幂等性 request_id：首次点击生成，重试不可重新生成盲重发
  class ReviewSubmissionState {
    constructor() {
      this.requestId = '';
    }

    getOrCreateRequestId() {
      if (!this.requestId) {
        this.requestId = `rev_req_${Date.now()}_test`;
      }
      return this.requestId;
    }
  }

  const subState = new ReviewSubmissionState();
  const id1 = subState.getOrCreateRequestId();
  const id2 = subState.getOrCreateRequestId();
  assert.ok(id1.length > 0);
  assert.equal(id1, id2, '重试时必须复用原 request_id 保证幂等');

  console.log('-> Test 2 PASSED');
}

// ----------------------------------------------------
// 测试 3: 晒图文件限制与载荷提取（仅传已完成的 file_id）
// ----------------------------------------------------
console.log('Test 3: 晒图文件限制与载荷提取');
{
  const MAX_IMAGES = 6;
  const MAX_FILE_SIZE = 5 * 1024 * 1024; // 5MB

  function checkCanAddImages(currentCount, addingCount) {
    return currentCount + addingCount <= MAX_IMAGES;
  }

  function validateFileSize(sizeBytes) {
    return sizeBytes <= MAX_FILE_SIZE;
  }

  function extractSuccessFileIds(images) {
    return images
      .filter((img) => img.status === 'success' && typeof img.fileId === 'number' && img.fileId > 0)
      .map((img) => img.fileId);
  }

  assert.equal(checkCanAddImages(4, 2), true);
  assert.equal(checkCanAddImages(5, 2), false);

  assert.equal(validateFileSize(5 * 1024 * 1024), true);
  assert.equal(validateFileSize(5 * 1024 * 1024 + 1), false);

  const testImages = [
    { localPath: 'temp://a.jpg', status: 'success', fileId: 101 },
    { localPath: 'temp://b.jpg', status: 'uploading' },
    { localPath: 'temp://c.jpg', status: 'error', errorMsg: 'fail' },
    { localPath: 'temp://d.jpg', status: 'success', fileId: 104 },
  ];

  const fileIds = extractSuccessFileIds(testImages);
  assert.deepEqual(fileIds, [101, 104], '必须仅提取已上传成功的数字 fileId');
  assert.ok(!fileIds.some((f) => typeof f === 'string'), '严禁将本地临时路径写入服务端参数');

  console.log('-> Test 3 PASSED');
}

// ----------------------------------------------------
// 测试 4: 匿名昵称与头像脱敏
// ----------------------------------------------------
console.log('Test 4: 匿名昵称与头像脱敏');
{
  const DEFAULT_AVATAR = '/static/logo.png';

  function resolveDisplayInfo(review) {
    if (review.is_anonymous === 1) {
      const raw = review.user_nickname || '用户';
      let masked;
      if (raw.length <= 1) {
        masked = raw + '***';
      } else {
        masked = raw.charAt(0) + '***' + raw.charAt(raw.length - 1);
      }
      return {
        avatar: DEFAULT_AVATAR,
        nickname: masked,
        isAnonymous: true,
      };
    }
    return {
      avatar: review.user_avatar || DEFAULT_AVATAR,
      nickname: review.user_nickname || '买家',
      isAnonymous: false,
    };
  }

  const anonUser = {
    is_anonymous: 1,
    user_nickname: '小苹果',
    user_avatar: 'https://example.com/avatar.jpg',
  };
  const r1 = resolveDisplayInfo(anonUser);
  assert.equal(r1.avatar, DEFAULT_AVATAR, '匿名时必须使用默认头像');
  assert.equal(r1.nickname, '小***果', '匿名时昵称必须脱敏');
  assert.equal(r1.isAnonymous, true);

  const realUser = {
    is_anonymous: 0,
    user_nickname: '张三',
    user_avatar: 'https://example.com/zhangsan.jpg',
  };
  const r2 = resolveDisplayInfo(realUser);
  assert.equal(r2.avatar, 'https://example.com/zhangsan.jpg');
  assert.equal(r2.nickname, '张三');
  assert.equal(r2.isAnonymous, false);

  console.log('-> Test 4 PASSED');
}

// ----------------------------------------------------
// 测试 5: 筛选防乱序竞争与状态映射
// ----------------------------------------------------
console.log('Test 5: 筛选防乱序竞争与状态映射');
{
  class FilterSeqManager {
    constructor() {
      this.seq = 0;
      this.currentFilter = 'all';
    }

    startRequest(filter) {
      this.currentFilter = filter;
      return ++this.seq;
    }

    isLatest(reqSeq) {
      return reqSeq === this.seq;
    }
  }

  const manager = new FilterSeqManager();
  const seq1 = manager.startRequest('all');
  const seq2 = manager.startRequest('has_image');

  // 若较早的 seq1 慢返回，不能覆盖最新的 seq2
  assert.equal(manager.isLatest(seq1), false, '旧请求返回时必须被丢弃');
  assert.equal(manager.isLatest(seq2), true, '最新请求被允许渲染');

  // 状态映射验证
  function getStatusBadge(status) {
    switch (status) {
      case 10:
        return { text: '待审核', tag: 'warning' };
      case 20:
        return { text: '已公开', tag: 'success' };
      case 30:
        return { text: '审核未通过', tag: 'danger' };
      case 40:
        return { text: '已隐藏', tag: 'info' };
      default:
        return { text: '未知', tag: 'default' };
    }
  }

  assert.equal(getStatusBadge(10).text, '待审核');
  assert.equal(getStatusBadge(20).text, '已公开');
  assert.equal(getStatusBadge(30).text, '审核未通过');
  assert.equal(getStatusBadge(40).text, '已隐藏');

  console.log('-> Test 5 PASSED');
}

console.log('ALL PRODUCT REVIEW CLIENT TESTS PASSED SUCCESSFULLY!');
