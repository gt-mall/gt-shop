# Go 后端测试规范

## 目录与归类

- 新增的 Go 单元测试和集成测试统一放在 `admin-server/test/<被测模块>/<被测方法>/`，目录对应源码模块和被测方法；例如 `bootstrap.CollectStartupInfo` 放在 `test/bootstrap/collect_startup_info/`，`product/sku.CreateSKU` 放在 `test/product/sku/create_sku/`。
- 一个方法的不同场景放在同一方法目录下，按需要拆分测试文件；不要为每个场景单独建目录，也不要在 `internal/`、`bootstrap/`、`routers/`、`pkg/` 等源码目录新增 `*_test.go`。
- 本规则约束新测试；现有 `internal/test/` 和源码目录中的测试在单独迁移时处理，修改 Skill 不等于已迁移。

## 命名与包名

- 优先使用黑盒测试包 `package <方法目录>_test`，通过公开接口测试行为。
- 测试文件用 `{功能}_test.go`；回归测试用 `{功能}_regression_test.go`；依赖真实 MySQL 的测试用 `{功能}_mysql_test.go`。
- 测试函数名用 `Test<被测方法><场景>`，便于按方法或场景运行。

## 环境与数据规范

- 常规测试优先使用 SQLite 内存库（`glebarez/sqlite`），按用例隔离数据；涉及 InnoDB 事务、锁或 MySQL 特性的行为使用真实 MySQL 验证。
- 金额测试覆盖元分往返转换与非法精度拦截；禁止使用 `float32` / `float64` 计算金额。
- 并发与事务测试注意连接池配置，避免 SQLite 排他锁造成测试死锁。

## 执行命令

在 `admin-server/` 下执行：

- 新测试目录：`go test ./test/...`
- 单个方法目录：`go test -v ./test/<被测模块>/<被测方法>/...`
- 单个用例：`go test -v ./test/<被测模块>/<被测方法>/ -run <测试函数名>`
- 全量回归（包含尚未迁移的旧测试）：`go test ./...`
