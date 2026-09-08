# Go 后端测试规范

## 目录与归类

- 业务测试统一放在 `admin-server/internal/test/<大模块>/`，严格镜像 `internal/<大模块>/`（如 `product`、`system`、`auth`）。
- 禁止在业务小模块（如 `internal/product/sku/`）内写测试文件，保持业务目录纯净。
- `pkg/` 工具库（如 `pkg/util/`）允许同包就地单测（如 `decimal_test.go`）。

## 命名与包名

- **包名**：统一使用黑盒测试包名 `package <大模块>_test`（如 `package product_test`）。
- **文件命名**：
  - `{功能}_test.go`：常规业务逻辑、Service 与 API 边界测试。
  - `{功能}_regression_test.go`：Bug 回归、并发竞争与死锁防范。
  - `{功能}_mysql_test.go`：依赖真实 MySQL 环境的持久化集成测试。
- **函数命名**：`func Test<业务对象><场景>(t *testing.T)`。

## 环境与数据规范

- **常规测试优先 SQLite 内存库**（`glebarez/sqlite`），做到秒级执行与用例级数据隔离。
- **金额测试强制要求**：覆盖元分往返转换与非法精度（超小数位）拦截；禁止使用 `float32/float64`。
- **并发与事务**：单测试内模拟事务时注意连接池配置，避免排他锁引起测试死锁。

## 执行命令

- 全部测试：`go test ./internal/test/...`
- 单大模块：`go test -v ./internal/test/<大模块>/...`
- 单个用例：`go test -v ./internal/test/<大模块>/ -run <函数名>`
