# Architecture

项目使用单一 Git 仓库，不引入复杂 Monorepo、Repository、DDD、CQRS 或微服务结构。

## Backend

请求流向保持为：

```text
routers -> 业务模块 API -> Service -> GORM/MySQL
```

- `admin-server/routers/`：集中注册系统、商品、订单、会员、数据分析和小程序路由及鉴权中间件。
- `admin-server/internal/model/`：数据库模型。
- `admin-server/internal/system/`：登录、用户、角色、权限、文件和系统设置。
- `admin-server/internal/product/`：分类、商品、库存、运费模板和商品字典。
- `admin-server/internal/order/`：后台订单、售后与退款协调。
- `admin-server/internal/member/`：后台会员管理。
- `admin-server/internal/dashboard/`、`internal/analytics/`：数据看板和经营分析。
- `admin-server/internal/miniapp/`：用户端登录、会员、商品、购物车、订单、微信支付与售后 API。
- `admin-server/internal/test/<domain>/`：按业务领域组织后端测试。
- `admin-server/bootstrap/`：配置、数据库、Migration、Seed、Casbin 和缓存初始化。
- `admin-server/pkg/`：跨业务公共能力。

后台 API 与小程序 API 分别生成到 `admin-server/docs/` 和 `admin-server/docs-miniapp/`。小程序路由统一使用 `/api/miniapp/*`，不转调后台 HTTP API。

## Admin Web

- `admin-web/src/framework/`：HTTP、生成 Client、布局、路由、登录态和权限。
- `admin-web/src/modules/`：看板、分析、商品、订单、会员和系统等业务页面。
- `admin-web/src/shared/`：跨业务复用的 UI、Hooks、类型和工具。

页面通过 `React.lazy` 按路由加载；菜单、路由和按钮统一消费后端返回的权限 code。API 调用使用 `admin-web/src/framework/http/generated_api.ts`，禁止手工维护重复 DTO。

## Miniapp

- `miniapp/src/pages/`：首页、分类、商品、购物车、会员、地址、订单和售后页面。
- `miniapp/src/api/`：按业务领域维护的薄 API 封装。
- `miniapp/src/services/`：请求、认证会话和启动配置。
- `miniapp/src/stores/`：会员、购物车和公共配置状态。
- `miniapp/src/components/`、`src/utils/`：复用组件和工具。

当前支持 H5 与微信小程序。金额字段沿用后端十进制字符串，关键交易结果以后端为准。

## Database

- `admin-server/data/migrations/001_baseline.sql`：V1 基线。
- `admin-server/data/migrations/002_product_dict.sql`：商品服务保障与单位字典增量迁移。
- `admin-server/data/seeds/001_default_data.sql`：幂等默认数据。

项目已经进入增量 Migration 阶段。后续结构变化只能新增递增 Migration，不得修改已经发布的 Migration，也不得再使用 `align` 重置版本记录。
