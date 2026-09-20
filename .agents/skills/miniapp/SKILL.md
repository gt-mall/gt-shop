---
name: miniapp
description: Mall miniapp 的 UniApp、Vue 3、TypeScript 和 Pinia 开发规范。修改用户端页面、组件、API、登录状态、交易流程或 H5/微信跨端行为时使用。
---

# Miniapp

## 项目结构

- UniApp + Vue 3 + TypeScript + Pinia；支持 H5 和微信小程序。
- `src/pages/` 放页面与局部逻辑，`src/components/` 放实际跨页面复用的组件；新增页面同步注册 `src/pages.json`。
- `src/api/` 放业务 API 薄封装，`src/services/` 放请求与会话基础能力，`src/stores/` 放跨页面状态，`src/utils/` 放工具。
- 修改前沿当前页面 → API → Store/公共工具检查关联代码。按业务区域拆分复杂页面，避免为了减少行数引入额外层级；局部表单和弹窗状态留在页面。
- 用户提供的设计与明确边界优先；复用现有组件和样式，公共组件以 Props/Emits 传递数据。

## 接口与错误处理

- 以 `admin-server/docs-miniapp/` 的 Swagger 为契约，仅调用 `/api/miniapp/*`。接口变化先维护后端 DTO/注解、生成 Swagger，再同步 `src/api/` 与调用方；禁止猜字段或伪造成功数据。
- 请求经 `src/api/` 和 `src/services/request.ts`；页面不直接写 `uni.request`，也不重复实现 Token、Base URL、401 处理。
- 异步优先 `async/await`，前端尽量不写 `try/catch`，复用已有错误处理能力。当前请求层会 reject，不能直接删掉必要的异常处理或留下未处理的 Promise；确需处理时集中在操作边界，保证失败提示、Loading 复位和重试。
- 不把请求失败显示为成功，不静默吞掉保存、下单、支付等错误；普通刷新失败保留可用旧数据并提供反馈。

## 会话与状态

- 复用 Member、Cart、App Store 及 `utils/auth-storage`，不新增 Token Key 或第二份登录状态。
- 登录成功同步 Store 与持久化数据；退出/认证失效清理会员、购物车、地址和结算临时状态。
- 请求发出时捕获会话，响应写回前检查归属；同账号并行刷新还需处理响应乱序。旧请求的 401 不得清理新账号。
- 页面返回后的数据通过现有生命周期/Store 刷新，不依赖延时模拟同步。

## 交易约束

- 金额保持十进制字符串；计算复用 `utils/money.ts` 的精确金额能力，禁止用 JavaScript 浮点数直接计算交易金额。
- 价格、库存、运费、订单与退款状态以后端为准；保留幂等请求 ID、不可配送提示和重复提交保护。
- 当前付款为模拟付款，未经任务要求不扩展真实支付。测试工具页的搬迁不应改变正常下单付款流程。

## 页面与测试工具

- 使用 `<script setup lang="ts">`、`rpx`、flex 和 UniApp API；平台专属能力通过条件编译隔离。
- 列表区分首次加载、加载更多、空数据、失败和无更多；提交按钮明确 Loading/Disabled，未登录时给出登录入口。
- 测试昵称、测试地址、预设地区和诊断数据统一放在 `src/pages/test/index.vue`；个人中心只保留跳转按钮，不在首页、个人中心或地址等业务页面铺开测试面板。
- 测试页复用真实 API 和 Store；写入资料或地址时说明影响、校验登录、防止重复操作。诊断展示使用必要且脱敏的字段，不输出 Token、微信 Code、密码或完整认证响应。
- 自动化测试仍放 `miniapp/test/`，独立测试页面不能代替自动化验证；优先验证实际源码行为，不能把复制业务逻辑的模拟测试称为完整端到端测试。

## 验证

在 `miniapp/` 执行 `pnpm run type-check` 和 `pnpm run test`。页面、路由或跨端改动执行 `pnpm run build:h5`、`pnpm run build:mp-weixin`，并检查生成路由；仓库根目录执行 `git diff --check`。

记录未完成的浏览器、微信开发者工具、真机或外部服务检查；构建成功不代表真实交易链路已验收。
