const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const { test } = require('node:test');
const ts = require('typescript');

const root = path.resolve(__dirname, '../src/modules/system/job');
const source = fs.readFileSync(path.join(root, 'status.ts'), 'utf8');
const output = ts.transpileModule(source, {
  compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.CommonJS },
}).outputText;
const context = vm.createContext({ exports: {} });
vm.runInContext(output, context);
const { canRetryJob, getJobStatusPresentation } = context.exports;

test('job status presentation covers idle, running, retrying, success, and failed states', () => {
  assert.deepEqual({ ...getJobStatusPresentation('idle') }, { label: '空闲', color: 'default' });
  assert.deepEqual({ ...getJobStatusPresentation('running') }, { label: '运行中', color: 'processing' });
  assert.deepEqual({ ...getJobStatusPresentation('retrying') }, { label: '等待重试', color: 'warning' });
  assert.deepEqual({ ...getJobStatusPresentation('success') }, { label: '成功', color: 'success' });
  assert.deepEqual({ ...getJobStatusPresentation('failed') }, { label: '最终失败', color: 'error' });
});

test('manual retry requires failed state and retry permission', () => {
  assert.equal(canRetryJob('failed', true), true);
  assert.equal(canRetryJob('failed', false), false);
  assert.equal(canRetryJob('running', true), false);
  assert.equal(canRetryJob('retrying', true), false);
  assert.equal(canRetryJob('success', true), false);
});

test('monitoring page uses generated API, confirms retry, and refreshes periodically', () => {
  const page = fs.readFileSync(path.join(root, 'index.tsx'), 'utf8');
  assert.ok(page.includes('api.api.listSystemJobs()'));
  assert.ok(page.includes('api.api.retrySystemJob(jobName)'));
  assert.ok(page.includes('<Popconfirm'));
  assert.ok(page.includes('window.setInterval'));
  assert.ok(page.includes('window.clearInterval'));
  assert.ok(page.includes('requestId !== latestRequest.current'));
  assert.ok(page.includes('latestRequest.current++'));
  assert.ok(!page.includes('.then('));
  assert.ok(!page.includes('.catch('));
});
