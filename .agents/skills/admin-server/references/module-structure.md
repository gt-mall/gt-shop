# Go 商城模块与金额规范

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
└── types.go
```

职责：

- `api.go`：Gin API / Handler
- `service.go`：业务逻辑
- `types.go`：Request、Response 等 API 结构体

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

单元测试按大模块归类放在 `internal/test/<大模块>/`，不要放在业务模块中。具体规范见 [test.md](test.md)。

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

### 7. 金额存储与计算

- 数据库金额以元为单位使用 `DECIMAL`；商品价格默认 `DECIMAL(12,2)`，其他金额按业务范围选择精度。元字段使用 `price`、`amount` 等名称，只有单位确为分时才使用 `_cent` 后缀。
- Go 读取及接收金额时保留十进制精度，使用十进制类型或原始十进制字符串。禁止先转成 `float32` / `float64` 再转分或转十进制类型；生成模型时配置正确的金额类型映射，不手改生成文件。
- 普通加减、整数数量乘法可先精确转换为 `int64` 分计算，写入数据库前再转回元。例如 `"19.90"` 元 → `1990` 分 → `"19.90"` 元。转换及计算检查数值范围，输入金额超过允许的小数位时返回校验错误，不直接截断。
- 折扣、除法和优惠分摊使用精确十进制运算，按业务规则在确定的结算节点舍入到分；明确舍入方式及尾差归属，保证明细合计等于总额。
- API 金额默认以元为单位，使用十进制字符串保留精度，并在 Go DTO 和 Swagger 中声明类型、单位及示例。对接要求整数分的外部接口时，在边界统一转换；修改已有接口或分字段时同步处理调用方和数据迁移，不只修改注释或字段名。
- 复用已有金额转换能力；需要新增时集中实现，避免各业务重复转换。金额逻辑的测试覆盖元分往返、舍入、分摊尾差、非法精度和溢出等实际边界。

## 示例

新增“商品品牌管理”：

```text
internal/product/brand/
├── api.go
├── service.go
└── types.go
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
