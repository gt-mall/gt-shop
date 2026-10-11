请先完整阅读项目根目录 `AGENTS.md`、`.agents/skills` 以及当前 `admin-server`、`admin-web` 的目录结构和编码规范，再开始修改。

目标：为当前 `GT-Shop` 项目创建商城用户端 `miniapp`，第一版使用 **uni-app + Vue3 + TypeScript + Vite** 开发，同时兼容：

* H5
* 微信小程序 MP-WEIXIN

本阶段只创建基础架构和 Hello World 联调，不开发商城业务功能。

## 一、总体架构要求

项目最终保持：

```text
gt-shop/
├── admin-server/
├── admin-web/
└── miniapp/
```

其中：

* `admin-web` 是后台管理端，保持不变。
* `miniapp` 是商城用户端，使用 uni-app。
* `admin-server` 继续作为 Go 服务。
* 小程序/H5 用户端必须拥有一套**独立 API**。
* miniapp **禁止调用现有后台管理 API**。
* miniapp API 与 admin API 必须在路由、Handler/API、DTO、Swagger 文档层面隔离。
* 可以合理复用：

  * database
  * model
  * pkg 公共工具
  * 通用底层 service/domain 能力
* 不允许为了复用而让 miniapp API 转调 admin API HTTP 接口。

API 路径统一使用：

```text
/api/miniapp/*
```

后台现有 API：

```text
/api/*
```

不要破坏现有后台 API。

---

# 二、创建 miniapp

在项目根目录创建：

```text
miniapp/
```

技术栈：

```text
uni-app
Vue 3
TypeScript
Vite
pnpm
```

必须使用 **CLI 工程**，不要依赖 HBuilderX GUI 才能开发或运行。

要求能够在 Linux SSH 环境运行：

```bash
pnpm install
pnpm dev:h5
pnpm build:h5
pnpm dev:mp-weixin
pnpm build:mp-weixin
```

优先使用当前稳定版依赖，不使用 alpha、beta、experimental 版本。

---

# 三、miniapp 第一版目录

先保持简单，不要过度设计。

建议：

```text
miniapp/
├── src/
│   ├── api/
│   │   └── common.ts
│   ├── components/
│   ├── pages/
│   │   └── index/
│   │       └── index.vue
│   ├── services/
│   │   └── request.ts
│   ├── types/
│   │   └── api.ts
│   ├── utils/
│   ├── App.vue
│   ├── main.ts
│   ├── manifest.json
│   ├── pages.json
│   └── uni.scss
├── .env.development
├── .env.production
├── .gitignore
├── index.html
├── package.json
├── tsconfig.json
├── vite.config.ts
└── README.md
```

空目录如果 Git 无法保存，可以暂时不创建 `.gitkeep`，没有实际需求不要制造无意义文件。

暂时：

* 不使用大型 UI 框架。
* 不安装 uView、ThorUI 等完整组件库。
* 不安装 lodash 全量包。
* 不安装日期处理大包。
* 不安装大型图标包。
* 不加入状态管理库，Hello World 阶段不需要 Pinia。
* 不引入无实际用途的第三方依赖。

优先使用：

```text
Vue
uni-app API
TypeScript
原生 CSS/SCSS
```

---

# 四、从项目第一天开始控制微信小程序包体积

这是强制要求。

设计原则：

1. 首页主包只保留启动必需代码。
2. 后续商城页面需要提前考虑分包：

   * 商品
   * 订单
   * 用户
   * 营销
3. 当前 Hello World 阶段暂时不用创建大量分包，但目录和代码不能阻碍以后分包。
4. 禁止把大量静态图片直接放进小程序代码包。
5. 商品图片未来全部从服务端/CDN加载。
6. 公共图片只保留真正需要的小图标。
7. SVG、iconfont、第三方字体谨慎使用。
8. 不因为开发方便引入完整大型组件库。
9. 第三方依赖必须按实际需要增加。
10. 禁止直接 import 整个工具库。
11. 所有新依赖都要考虑是否会进入最终 MP-WEIXIN 产物。
12. 页面组件按需加载。
13. 后续业务页面优先使用分包。
14. H5 专属代码和微信专属代码使用 uni-app 条件编译隔离。

例如：

```text
#ifdef H5
#endif

#ifdef MP-WEIXIN
#endif
```

但不要为了“以后可能有差异”提前制造大量空的平台适配文件。

增加简单的构建说明，让开发者可以查看：

```text
dist/build/h5
dist/build/mp-weixin
```

构建后的大小。

README 中明确说明：

> 判断小程序包体积时以最终 `dist/build/mp-weixin` 产物为准，不以 `node_modules` 大小作为小程序包大小。

---

# 五、统一请求层

miniapp 不允许页面直接大量调用：

```ts
uni.request(...)
```

创建：

```text
src/services/request.ts
```

统一封装请求。

需要支持：

```text
baseURL
method
url
data
header
timeout
统一 Response 类型
HTTP 错误
业务错误
Token 预留
```

第一版暂时不做复杂自动重试。

必须保证请求行为稳定、简单、可追踪。

API 返回统一按项目目前风格：

```json
{
  "code": 1,
  "message": "ok",
  "data": {},
  "total": 0
}
```

TypeScript 建立：

```ts
interface ApiResponse<T> {
  code: number
  message: string
  data: T
  total?: number
}
```

页面不能自己解析一套新的返回规范。

---

# 六、环境配置

支持开发环境和生产环境 API 地址。

例如：

```text
.env.development
.env.production
```

统一通过：

```text
VITE_API_BASE_URL
```

读取。

开发环境可以先使用：

```text
http://127.0.0.1:8080
```

但代码中禁止硬编码 API 域名。

H5 开发时如果存在跨域问题，优先通过 Vite dev proxy 或后端开发环境 CORS 正确解决。

不要在业务代码中写临时 hack。

---

# 七、后端创建独立 miniapp API

在 `admin-server` 内创建独立的用户端 API 模块。

推荐结构：

```text
admin-server/
├── internal/
│   └── miniapp/
│       └── common/
│           ├── api.go
│           ├── service.go
│           └── types.go
│
├── routers/
│   └── miniapp.go
│
└── docs-miniapp/
```

注意：

`miniapp` 后端模块未来可能继续扩展为：

```text
internal/miniapp/
├── auth/
├── product/
├── cart/
├── order/
├── user/
└── common/
```

当前只创建 Hello World 真正需要的内容。

不要一次创建几十个空模块。

---

# 八、miniapp API 与后台 API 强制隔离

必须满足：

```text
后台管理 API
/api/...

商城用户端 API
/api/miniapp/...
```

禁止 miniapp 调用例如：

```text
/api/product/products/list
/api/product/categories/list
/api/system/...
```

未来商品接口应该重新提供：

```text
/api/miniapp/products/...
```

注意：

独立 API ≠ 重复所有业务代码。

允许：

```text
miniapp api
   ↓
共享 domain/service/repository
   ↓
database
```

禁止：

```text
miniapp api
   ↓ HTTP
admin api
```

也尽量避免：

```text
miniapp api
   ↓
直接调用 admin API handler
```

API 层保持独立。

---

# 九、独立 Swagger

miniapp API 必须拥有自己的 Swagger 文档。

不能把用户端 Swagger 与当前后台 Swagger 混在同一个输出文件里。

当前：

```text
admin-server/docs/
```

继续用于后台。

新增：

```text
admin-server/docs-miniapp/
```

用于商城用户端。

增加独立生成脚本，例如：

```text
admin-server/scripts/generate-miniapp-swagger.sh
```

生成：

```text
docs-miniapp/
├── docs.go
├── swagger.json
└── swagger.yaml
```

Swagger 只应包含：

```text
/api/miniapp/*
```

不能把后台 `/api/...` 接口全部混进去。

Swagger：

```text
Title: GT-Shop Miniapp API
BasePath: /api/miniapp
```

如果现有 swag 工具的实际生成机制要求采用其它合理目录或入口文件，可以根据项目当前 Swagger 实现调整，但必须保证最终做到：

**后台 Swagger 与 miniapp Swagger 分离。**

不要升级或替换现有 Swagger 工具，除非确实无法实现。

---

# 十、创建 Hello World API

后端创建一个最简单的接口：

```http
GET /api/miniapp/hello
```

或者如果当前项目 API 风格统一使用 POST，则遵循项目现有规范使用：

```http
POST /api/miniapp/hello
```

返回：

```json
{
  "code": 1,
  "message": "ok",
  "data": {
    "text": "Hello World"
  },
  "total": 0
}
```

对应类型：

```go
type HelloResponse struct {
    Text string `json:"text"`
}
```

必须加入 miniapp 独立 Swagger。

接口当前无需登录。

---

# 十一、miniapp 调用 Hello World

创建：

```text
src/api/common.ts
```

例如提供：

```ts
getHello()
```

`pages/index/index.vue` 不直接调用 `uni.request`。

调用链必须是：

```text
index.vue
   ↓
api/common.ts
   ↓
services/request.ts
   ↓
/api/miniapp/hello
```

首页显示：

```text
GT-Shop
Hello World
```

其中 `Hello World` 必须来自 Go API 返回的数据，而不是前端写死。

增加：

```text
loading
success
error
```

三个基本状态。

接口失败时页面显示简单友好的：

```text
数据加载失败
```

不要因为 Hello World 引入复杂状态管理。

---

# 十二、H5 验证

完成后实际执行：

```bash
cd miniapp
pnpm install
pnpm dev:h5
```

确认 H5 可以打开。

并执行：

```bash
pnpm build:h5
```

确保无 TypeScript/Vite 构建错误。

如果当前环境允许启动 Go 服务，则实际调用：

```text
/api/miniapp/hello
```

验证页面显示服务端返回的：

```text
Hello World
```

---

# 十三、微信小程序验证

执行：

```bash
pnpm build:mp-weixin
```

必须成功生成：

```text
miniapp/dist/build/mp-weixin/
```

确保该目录可以直接导入微信开发者工具。

如果没有真实微信 AppID：

使用测试号/游客模式可用配置，或者 README 明确告诉开发者在哪里填写 AppID。

不要把个人 AppID、Secret 等敏感信息提交 Git。

---

# 十四、不要做的事情

本任务严格控制范围。

不要：

* 开发登录
* 开发商品列表
* 开发分类
* 开发购物车
* 开发订单
* 开发支付
* 开发用户中心
* 引入 UI 框架
* 引入 Pinia
* 引入 Axios
* 引入大型第三方工具库
* 修改现有 admin-web
* 重构现有商品模块
* 修改现有后台 API 语义
* 为未来功能提前创建大量空目录
* 添加无意义 abstraction
* 创建重复的 response 系统
* 重复实现项目已经存在的通用后端工具

此次目标只是：

```text
基础工程
+
独立 miniapp API
+
独立 Swagger
+
统一 request
+
H5 Hello World
+
微信小程序 Hello World
```

---

# 十五、代码质量

遵循现有项目代码风格。

后端：

```text
Go
Gin
GORM
当前统一 response
当前错误处理风格
```

前端：

```text
Vue3
<script setup lang="ts">
TypeScript strict
组合式 API
```

禁止使用：

```text
any
```

除非第三方库确实无法避免，并注明原因。

不要出现：

```text
.then()
.catch()
.finally()
```

统一使用：

```ts
async / await
```

按照项目现有约定处理错误。

---

# 十六、README

`miniapp/README.md` 至少说明：

1. 技术栈
2. 安装命令
3. H5 开发命令
4. H5 构建命令
5. 微信小程序开发命令
6. 微信小程序构建命令
7. API 环境变量
8. 微信开发者工具导入目录
9. 包体积控制原则
10. miniapp API 与 admin API 的边界

---

# 十七、完成后的检查

修改完成后必须实际执行：

```bash
cd miniapp
pnpm install
pnpm build:h5
pnpm build:mp-weixin
```

后端执行项目现有：

```bash
go test ./...
```

如果项目有现有 Swagger 生成脚本，也不要破坏。

再执行新的：

```bash
./scripts/generate-miniapp-swagger.sh
```

确认独立 Swagger 可以生成。

检查：

```text
git status
```

确保没有提交：

```text
node_modules
dist
.env 中的私密配置
微信 Secret
临时日志
IDE缓存
```

---

# 十八、最终向我汇报

完成后不要只说“完成”。

请给我：

```text
1. 新增目录结构
2. 新增/修改文件列表
3. miniapp API 路径
4. Swagger 文档路径
5. H5 启动命令
6. 微信小程序构建命令
7. 微信开发者工具应该导入哪个目录
8. H5 build 是否成功
9. MP-WEIXIN build 是否成功
10. Go test 是否成功
11. 当前构建后的 mp-weixin 目录大小
12. 是否引入了第三方运行时依赖以及原因
```

如果执行过程中发现现有项目结构与上述建议存在冲突：

**优先遵循项目现有架构和 AGENTS/skills 规范，在不改变上述核心边界的情况下做最小调整，不要为了严格照搬目录而重构现有代码。**

现在直接检查项目并执行，不要只给方案。
