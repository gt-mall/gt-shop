# AI Shop 商城用户端 (Miniapp)

本项目为 `ai-shop` 独立商城用户端，采用 **uni-app + Vue 3 + TypeScript + Vite** 开发，同时兼容 H5 与微信小程序 (MP-WEIXIN)。

## 购物车抽屉组件

`src/components/CartDrawer.vue` 复用 Cart Store 和现有结算流程。商品详情页为实际接入示例；其他页面可按下方使用。微信端 `page-meta` 必须放在页面模板首位，用于锁住背景页面滚动。

```vue
<template>
  <!-- #ifdef MP-WEIXIN -->
  <page-meta :page-style="scrollLocked ? 'overflow: hidden;' : ''" />
  <!-- #endif -->
  <view>
    <button @tap="cartVisible = true">购物车</button>
    <CartDrawer v-model="cartVisible" @scroll-lock="scrollLocked = $event" />
  </view>
</template>
<script setup lang="ts">
import { ref } from 'vue'
import CartDrawer from '@/components/CartDrawer.vue'
const cartVisible = ref(false)
const scrollLocked = ref(false)
</script>
```

- Props：`modelValue` 控制开关；`height` 默认 `75vh`；`showClose` 默认 `true`。
- Events：`close` 表示开始关闭，`scroll-lock` 同步背景滚动锁，`checkout` 仅表示成功进入结算页（不是下单或支付成功）。
- H5 自动保存/恢复页面滚动样式；页面隐藏、组件卸载时释放锁。微信新接入页面需要绑定上例中的 `page-meta`，已有页面样式应合并保留。
- 勾选状态归各组件实例，购物车商品和写操作锁由 Store 共享。首次默认选中可结算商品，之后刷新保留用户选择；修改/删除期间禁止重复操作与结算。

---

## 1. 技术栈

* **框架**：[uni-app](https://uniapp.dcloud.io/) (Vue 3 组合式 API + `<script setup lang="ts">`)
* **开发语言**：TypeScript (Strict 模式，强类型约束)
* **构建工具**：Vite 5
* **包管理器**：pnpm
* **样式方案**：原生 SCSS / CSS 样式，无重型 UI 框架开销

---

## 2. 安装命令

在 `miniapp` 目录下执行：

```bash
pnpm install
```

---

## 3. H5 开发与构建命令

* **启动 H5 本地开发服务**：
  ```bash
  pnpm dev:h5
  ```
  本地服务默认运行在 `http://localhost:5100`，支持 Vite 代理转发 `/api` 请求至 Go 后端。

* **构建 H5 生产产物**：
  ```bash
  pnpm build:h5
  ```
  构建产物输出至：`dist/build/h5/`

---

## 4. 微信小程序开发与构建命令

* **启动微信小程序本地监听构建**：
  ```bash
  pnpm dev:mp-weixin
  ```
  监听产物输出至：`dist/dev/mp-weixin/`

* **构建微信小程序生产产物**：
  ```bash
  pnpm build:mp-weixin
  ```
  构建产物输出至：`dist/build/mp-weixin/`

---

## 5. 微信开发者工具导入目录

* **导入目录**：`miniapp/dist/build/mp-weixin`（或开发调试时导入 `miniapp/dist/dev/mp-weixin`）
* **微信 AppID 说明**：
  - 默认工程使用测试/游客号配置（`manifest.json` 中配置的 `appid`）。
  - 开发真实小程序时，请在 `src/manifest.json` -> `mp-weixin.appid` 中填入你的小程序 AppID。
  - **切勿**将敏感密钥（如 AppSecret）或私密配置提交到版本库。

---

## 6. API 环境变量说明

项目使用 Vite 标准环境变量文件进行 API 地址管理：

* `.env.development`：开发环境配置
  ```env
  VITE_API_BASE_URL=http://127.0.0.1:8100
  ```
* `.env.production`：生产环境配置
  ```env
  VITE_API_BASE_URL=https://api.example.com
  ```

网络请求层统一封装在 `src/services/request.ts` 中，优先读取 `VITE_API_BASE_URL`：
* **H5 环境**：开发模式下走相对路径触发 Vite Proxy (`/api` -> 后端端口)，杜绝浏览器跨域问题。
* **小程序环境**：使用完整的 HTTP/HTTPS 服务端根路径。

---

## 7. miniapp API 与 admin API 的边界原则

1. **路由与接口绝对隔离**：
   - 商城用户端 API 统一前缀为 `/api/miniapp/*`。
   - 后台管理端 API 统一前缀为 `/api/*`（如 `/api/system/*`, `/api/product/*`）。
   - **禁止** miniapp 直接调用现有的后台管理 API。
2. **Swagger 合约隔离**：
   - 用户端 Swagger 产物位于 `admin-server/docs-miniapp/`，BasePath 为 `/api/miniapp`。
   - 后台管理端 Swagger 位于 `admin-server/docs/`，两者互不污染。
   - 用户端 Swagger 生成脚本：`admin-server/scripts/generate-miniapp-swagger.sh`。
3. **后端代码复用规范**：
   - 允许复用底层的 database、model、pkg 公共工具以及公共领域/Service 能力。
   - 严禁 miniapp API 转调 admin API 的 HTTP 接口。

---

## 8. 包体积控制原则

微信小程序主包体积上限为 2MB，必须从第一天起严格控制体积：

> **重要准则**：判断小程序包体积时以最终 `dist/build/mp-weixin` 产物为准，不以 `node_modules` 大小作为小程序包大小。

1. **主包精简**：首页主包只保留启动和首页渲染必需的代码与依赖。
2. **提前规划分包**：后续商品、订单、购物车、用户中心等页面均采用 uni-app 分包加载。
3. **禁止提交大静态资源**：图片、插画、商品多媒体全部托管于服务端/对象存储 CDN，小程序包内只允许保留极少量核心矢量图标。
4. **拒绝全量重型库**：严禁引入全量 lodash、moment/dayjs 等大包，严禁未经裁剪引入重量级 UI 组件库；优先使用 uni-app 原生能力与 SCSS/CSS。
5. **平台差异条件编译**：仅在必要时通过 `#ifdef H5` 或 `#ifdef MP-WEIXIN` 进行平台隔离，杜绝空适配文件。

---

## 9. 工具类库与函数规范 (`src/utils`)

为杜绝小程序主包体积失控以及全局污染隐患，项目对工具函数实行集中自研收口治理：

1. **唯一出口原则**：
   - 所有页面与组件使用工具函数，**必须统一从 `@/utils` 按需导入**：
     ```ts
     import { formatMoney, debounce, formatDate } from '@/utils'
     ```
   - **严禁全局挂载**（如禁止在 `main.ts` 中通过 `app.config.globalProperties` 或 `uni.$xxx` 挂载全局工具），确保 TypeScript 类型推导准确并让 Vite Tree-shaking 彻底生效。
2. **核心工具自主手写原则**：
   - `src/utils/money.ts`：商城金额精确计算（分元互转、加减乘、展示格式化），严格防止 JS 浮点数失真。
   - `src/utils/tool.ts`：通用核心防抖 (`debounce`)、节流 (`throttle`)、轻量深拷贝 (`deepClone`)、判空 (`isEmpty`)。
   - `src/utils/date.ts`：轻量日期格式化 (`formatDate`)、订单支付倒计时 (`formatCountdown`)。
   - `src/utils/format.ts`：业务数据脱敏（手机号 `maskPhone`、姓名 `maskName`、URL 参数解析 `parseQuery`）。
   - `src/utils/storage.ts`：本地缓存安全包装，支持 TTL 过期时间控制。
