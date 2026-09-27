const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const { test } = require('node:test');

const loginPath = path.resolve(__dirname, '../src/modules/login/index.tsx');
const loginSource = fs.readFileSync(loginPath, 'utf8');

test('login page complies with national ICP and public security filing regulations', () => {
  // 工信部 ICP 备案合规性要求：必须链接至工信部备案系统 (https://beian.miit.gov.cn/) 并新窗口打开
  assert.ok(
    loginSource.includes('https://beian.miit.gov.cn/'),
    'Login page must link to MIIT beian portal',
  );
  assert.ok(
    loginSource.includes('target="_blank"'),
    'Regulatory filing links must open in a new tab',
  );
  assert.ok(
    loginSource.includes('rel="noopener noreferrer"'),
    'External regulatory links must specify security rel attributes',
  );

  // 公安机关互联网站安全管理服务平台备案合规性要求：
  // 必须展示全国统一网安备案图标，并提供公安联网备案编号及平台查询链接
  assert.ok(
    loginSource.includes('beian.png') || loginSource.includes('beianIcon'),
    'Login page must display the official Cyber Police filing emblem icon',
  );
  assert.ok(
    loginSource.includes('beian.mps.gov.cn') || loginSource.includes('beian.gov.cn/portal/registerSystemInfo'),
    'Login page must link to National Public Security filing service platform',
  );
  assert.ok(
    loginSource.includes('policeRecord') || loginSource.includes('公网安备'),
    'Login page must display Public Security filing record number',
  );
});

test('login page uses unified theme token and avoids forbidden async keywords', () => {
  // 统一主题和 Design Token
  assert.ok(
    loginSource.includes('ConfigProvider') && loginSource.includes('adminThemeConfig'),
    'Login page must wrap with unified ConfigProvider and adminThemeConfig',
  );

  // 严格遵守项目规范：禁止 .then(), .catch(), try/catch
  assert.ok(!loginSource.includes('.then('), 'Forbidden .then() in handwritten code');
  assert.ok(!loginSource.includes('.catch('), 'Forbidden .catch() in handwritten code');
  assert.ok(!loginSource.includes('try {'), 'Forbidden try/catch in handwritten code');
});

test('filing numbers match system settings and default to empty without fake defaults', () => {
  // 必须与站点设置关联，读取 siteSetting.icp 与 siteSetting.police_record
  assert.ok(
    loginSource.includes('siteSetting?.icp'),
    'Login page must read ICP number from siteSetting',
  );
  assert.ok(
    loginSource.includes('siteSetting?.police_record'),
    'Login page must read police record from siteSetting',
  );

  // 绝不瞎写：不得包含硬编码的伪造默认备案号
  assert.ok(!loginSource.includes('DEFAULT_ICP'), 'Must not define hardcoded DEFAULT_ICP');
  assert.ok(!loginSource.includes('DEFAULT_POLICE_RECORD'), 'Must not define hardcoded DEFAULT_POLICE_RECORD');
  assert.ok(!loginSource.includes('2026008888'), 'Must not contain fake default ICP number');

  // 未填写默认为空：条件渲染底部备案区
  assert.ok(
    loginSource.includes('Boolean(icpNumber || policeRecord)'),
    'Must conditionally render footer-beian only when filing numbers are provided',
  );
});
