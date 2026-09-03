---
name: go-module-structure
description: >
  Go 商城 internal 目录模块化规范。
  仅当新增、移动、重构 Go 后端业务模块，或需要决定代码应该放到
  product、sku、category、system 等哪个模块时使用。
  普通业务代码修改、Bug 修复、SQL 修改、前端开发时不要使用。
  单元测速放在 internal/test 目录下，不要放在业务模块中。
---

# Go 商城模块化目录 Skill

## 目标

项目 `internal` 采用：

```text
大模块
└── 小模块
```

按业务功能组织代码，不按技术层级组织。

例如：

```text
internal/
├── product/
│   ├── product/
│   ├── sku/
│   └── category/
│
├── system/
│   ├── admin/
│   ├── role/
│   └── permission/
│
└── auth/
```

## 核心规则

### 1. 大模块

大模块代表一个完整业务领域，例如：

```text
product
order
customer
system
marketing
payment
```

只有项目实际使用到时才创建，不提前建立大量空模块。

### 2. 小模块

小模块代表一个相对独立的业务功能。

例如：

```text
product/
├── product/
├── sku/
├── category/
└── brand/
```

一个小模块可以对应一张或多张数据库表。

不要按照“一张表一个目录”的方式拆分。

### 3. 小模块内部

默认保持简单：

```text
sku/
├── api.go
├── service.go
├── types.go
└── router.go
```

职责：

- `api.go`：Gin API / Handler
- `service.go`：业务逻辑
- `types.go`：Request、Response 等 API 结构体
- `router.go`：路由注册

不要默认增加：

```text
controller/
service/
repository/
dao/
dto/
vo/
```

只有模块明显复杂后才继续拆分。

### 4. 数据库代码

数据库反向生成代码统一放：

```text
internal/model/
internal/query/
```

其中：

```text
model/   # GORM Gen 数据库结构体
query/   # GORM Gen 查询代码
```

生成代码禁止手工修改。

业务模块直接使用 `model` 和 `query`。

### 5. 模块调用

小模块尽量不要互相循环调用。

例如不要：

```text
sku → category
category → sku
```

需要同时操作多个小模块时，由大模块负责业务编排：

```text
product/
├── product/
├── sku/
├── category/
└── service.go
```

例如创建商品需要：

```text
检查分类
创建商品
创建 SKU
初始化库存
```

统一由：

```text
product/service.go
```

协调。

### 6. 新增功能时

新增代码前先判断：

1. 属于哪个大模块？
2. 是否已有合适的小模块？
3. 如果已有，直接放入现有小模块。
4. 如果是独立业务功能，再创建新的小模块。
5. 不要因为新增一张数据库表就创建新目录。

## 示例

新增“商品品牌管理”：

```text
internal/product/brand/
├── api.go
├── service.go
├── types.go
└── router.go
```

新增 SKU 规格表：

```text
product_sku_specs
product_sku_spec_values
```

如果它们属于 SKU 功能，则继续放：

```text
internal/product/sku/
```

不要额外创建：

```text
sku_spec/
sku_spec_value/
```

## 原则

优先：

```text
业务清晰
目录少
容易定位
方便 AI 修改
避免循环依赖
```

不要为了“架构完整”而过度拆分代码。
