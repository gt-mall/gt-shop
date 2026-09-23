# 小程序下一阶段任务提示词

研究日期：2026-09-20。代码基线：`641dd4f`。本目录是实施方案与 AI 执行提示词，尚未实现这些任务。执行时先复查工作树；以最新源码、Swagger 和实际数据库结构为准。

## 现状与任务拆分

| 任务 | 当前源码结论 | 实施范围 |
| --- | --- | --- |
| [01 订单倒计时](01-order-countdown.md) | 已有 30 分钟超时关单、每分钟扫描；缺倒计时契约与支付截止校验 | 后端、小程序 |
| [02 微信地址导入](02-wechat-address.md) | 地址页已有 `chooseAddress`，但猜测区划编码，缺完整导入确认与异常恢复 | 主要小程序，复用后端地址接口 |
| [03A 评价后端](03a-review-server.md) | 无评价业务表/接口；只有文件场景常量 `review`，上传白名单尚未支持 | 后端、Migration、Swagger |
| [03B 评价页面](03b-review-clients.md) | 无发表、展示、审核页面 | 小程序、管理端；依赖 03A |
| [04 独立搜索](04-search.md) | 分类页支持关键词和排序，首页入口跳分类；无独立搜索页、历史与热门词配置 | 三端 |
| [05A 原生分享](05a-native-share.md) | 未找到商品分享生命周期与分享入口 | 小程序 |
| [05B 商品海报](05b-product-poster.md) | 无 Canvas 海报、商品小程序码接口；现有配置中的二维码是通用静态码 | 后端、小程序；依赖 05A 路由约定 |
| [06 骨架与弱网](06-skeleton-network.md) | 已有部分 loading/错误/重试，无统一骨架；个人中心分页失败可能跳页 | 主要小程序 |

建议顺序：**01 → 02 → 06 → 04 → 03A → 03B → 05A → 05B**。各任务单独验收；03B 必须先有 03A 的真实契约，05B 与 05A 共用商品落地路径。06 的组件应被后续新页面复用。

这些任务都会接触 `pages.json`、商品/订单页面或 Swagger，按顺序实施能减少冲突。不要让多个任务各自固定同一个 Migration 编号；当前已存在 `001_baseline.sql` 和 `002_product_dict.sql`，创建新迁移时取最新下一个编号。

## 交给 AI 的通用开场提示词

```text
请在 /home/shop/mall 实施 docs/next_task/<指定任务文件>。
先完整读取本 README 的公共约束和指定任务，再复查最新源码与 git diff。
按指定任务直接完成代码、必要测试、Swagger/类型同步和简短交付说明。
本次只实施指定任务；相关前置任务未完成时，先核实并说明实际依赖。
保留用户已有改动。各文件中列出的“建议接口/新增文件”是待实现设计，不是已有能力。
真实外部服务不可用时完成可验证的代码和模拟边界测试，明确剩余配置与真机验收项，不伪造联调成功。
```

## 所有任务共同遵守

1. 读取根 `AGENTS.md` 和 `.agents/skills/miniapp/SKILL.md`；后端任务读取 `admin-server` Skill 与对应 module/test 参考；管理端任务读取 `admin-web` Skill 及 CRUD、UI、权限参考。前端尽量不使用 try/catch；管理端异步按其更严格规范执行，小程序保证 reject 被处理。
2. Go DTO/注解 → Swagger → 前端调用。生成命令：仓库根执行 `bash scripts/3_generate_api.sh`；影响管理端契约时执行 `bash sw.sh`。小程序 API 是 `miniapp/src/api/` 薄封装，禁止手改 `generated_api.ts`。统一响应 `code === 1` 成功，分页总数在响应 `total`。
3. 小程序只调用 `/api/miniapp/*`，复用 request、Member/Cart Store、会话归属与乱序保护。不重新实现 Token；用户 ID 从登录态取，后台操作必须服务端鉴权。
4. 本文档记录当时的规划边界；当前订单已接入微信小程序真实支付/退款。金额使用元字符串和整数分工具；不改变订单状态枚举、库存锁定与订单快照语义。
5. 新表和字段通过新增增量 Migration；禁止清库、重写历史迁移、运行 `align`。模型生成须遵循 Skill：当前未找到现成的 GORM Gen 入口，涉及新模型时先核实并补齐可复现的生成流程，在隔离测试库验证，不把手写生成文件伪装成生成结果。
6. 公共组件保持 Props/Emits 简单，提供使用说明和最小 Demo/真实接入示例。复用当前 `CartDrawer.vue`、`CartItemCard.vue`、`useCartManage.ts`、`OrderGoodsList.vue`、`ProductCard.vue`；不复制整页、不引入新的 UI 框架或复杂架构。
7. 测试业务真实源码；网络和微信接口可 Mock，但报告要区分源码测试、浏览器 Mock、真实后端、真实 MySQL、微信真机。缺少专用测试 DSN 时不得用业务库替代。

## 公共验收命令

按受影响端执行，并保留结果及未执行原因：

```bash
# 后端变化
cd /home/shop/mall/admin-server
go test ./...

# 管理端变化
cd /home/shop/mall/admin-web
pnpm run test
pnpm run build

# 小程序变化
cd /home/shop/mall/miniapp
pnpm run test
pnpm run type-check
pnpm run build:h5
pnpm run build:mp-weixin

cd /home/shop/mall
git diff --check
```

真实 MySQL 并发和迁移验收使用可丢弃测试库的 `TEST_DATABASE_DSN` / `ORDER_TEST_MYSQL_DSN` 等对应变量；SQLite 与构建成功不能替代这些验收。

## 平台文档依据

地址、分享、Canvas、网络能力已查阅 DCloud 官方文档，链接放在相关任务中。微信开发者文档本次抓取失败，涉及隐私声明、基础库兼容、朋友圈单页模式、小程序码参数与发布环境限制，执行时须再次核对微信官方最新文档并做真机验收，不能照抄旧教程的权限配置。
