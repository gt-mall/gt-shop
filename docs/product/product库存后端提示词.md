# 库存后端 AI 执行提示词

先执行本提示词。完成后再将 [前端提示词](product库存前端提示词.md) 发给另一个 AI。SQL 与完整契约在 [库存方案](product库存方案.md) 和 [product库存.sql](product库存.sql)。

复制下面整段给后端 AI：

```text
请直接在 /home/shop/mall 实现商品库存后端，完成数据库迁移文件、业务代码、商品生命周期接入、Swagger、生成的前端 API Client 和必要验证。本段不编写前端页面。

【先读与范围】
1. 读取 AGENTS.md、.agents/skills/go-module-structure/SKILL.md、docs/product/product库存方案.md、docs/product/product库存.sql。
2. 阅读当前 admin-server/internal/product/product、internal/product/category、internal/model/product.go、routers/product.go、internal/auth、pkg/response、internal/test/product_test.go、product_regression_test.go；检查项目当前的迁移和 API 生成脚本。
3. 既有商品三张表和商品功能已经完成，不重做商品 CRUD。旧 product提示词.md 中“库存不做”是上一阶段约束，本任务按新库存方案扩展。
4. 本期仅单仓库、整数件数、SKU 余额、人工入库/出库/盘点、预警阈值、只读流水。locked_stock 为未来订单预留，本期只初始化为0、人工操作原样保留已有值，不提供订单占用/释放接口，不新建订单、采购、仓库、调拨、批次或价格成本模块。
5. 保留用户已有修改。先只读核对真实表结构；当前文档规划不代表业务库已建表。连接不可用时依据文档实现并注明未验证项，不擅自重建或清空旧表，不输出密码、DSN或Token。

【迁移与模型】
以 docs/product/product库存.sql 为 SQL 依据，在 admin-server/migrations 新增下一个可用编号的迁移，当前已存在001..005，执行时再核对编号。文档的建表与回填可作为同一迁移，但必须说明 DDL 不受普通事务整体回滚保护。
目标 MySQL 8.4 / InnoDB，新增且只新增 product_sku_inventory、product_sku_inventory_log。不得在产品运行时用 AutoMigrate 覆盖正式表结构；测试环境可单独准备表。
余额：id、product_id、sku_id、stock、locked_stock、warning_stock、version、created_at、updated_at；sku_id唯一。可用量=stock-locked_stock，只在查询/响应计算，不持久化第三份数量。
流水字段、索引、CHECK、快照按 SQL；request_id全局唯一，(inventory_id,version_after)唯一。流水没有更新和删除接口，两张表都没有deleted_at。
quantity和库存字段的业务范围0..2147483647，delta允许负数；用int64检查加减与范围。version为INT UNSIGNED，允许0，溢出明确拒绝。ID使用Go int64；已有商品表的INT不自动迁移，不引入雪花ID。金额继续沿用DECIMAL元和精确转换，库存数量不使用浮点或DECIMAL。
当前是手写GORM模型就沿用，显式TableName映射单数表名；只有实际存在的生成文件才禁止手改，不强行启用GORM Gen。
存量回填包含禁用和软删除SKU，只补缺失的零余额，不覆盖原库存，不写0到0的假流水。真实期初量必须走入库接口留痕。识别孤儿SKU/归属错误并报告，不用INSERT IGNORE隐藏异常。
业务库是否实际应用迁移依本次执行授权与部署环境确定；始终先在隔离库验证SQL。交付明确区分“迁移文件已生成”和“业务库已应用”，不能将连接成功当作迁移完成。

【目录与依赖】
新增 admin-server/internal/product/inventory/{api.go,service.go,types.go}，流水也属于inventory，不另建inventory_log业务模块。不加Repository、DDD、消息队列或通用库存框架。
库存模型放internal/model，业务测试放internal/test。路由在外层routers/product.go注册。
product/product.Service可单向调用inventory的初始化方法，方法必须接收调用方tx；inventory只依赖模型和公共能力，不反向调用product服务。不为本次改造强拆当前已经工作的模块。

【写入规则】
人工调整change_type只接受manual_in、manual_out、stocktake：
- manual_in：quantity是本次增加数量，>0，stock增加。
- manual_out：quantity是本次减少数量，>0，不得超过available_stock，stock减少。
- stocktake：quantity是盘点后的目标总库存，可为0，不是变化量，也不是目标可用量；目标不能小于locked_stock。
warning接口由服务端固定change_type=warning_set，quantity是目标warning_stock，允许0表示关闭预警，stock和locked_stock不变。
所有操作原因remark去除首尾空白后1..500字符；操作类型、请求必填、数量、版本、UUID和溢出均需服务端验证。stocktake/预警目标未变化返回NO_CHANGE，不加版本不写日志。
所有成功变更都version+1且写一条完整流水，保存总量/锁定/阈值的前后值、delta、版本、原始quantity、request_id/hash、操作者和SKU快照。
操作者从auth.CurrentUser(c)派生，operator_name用Nickname或Username；前端不能提供操作者、product_id、inventory_id、before/after、delta或hash来驱动写库。
SKU快照使用明确Struct，格式遵循方案，规格必须按(spec_id,value_id)联合查找；默认SKU specs=[]。响应JSON返回对象/数组，不能返回数据库JSON文本字符串。

【事务与幂等：不可省略】
先校验当前用户权限，再查幂等记录。规范化request_hash必须包含operator_id、sku_id、change_type、quantity、expected_version和trim后的remark，使用固定Struct序列化后SHA256；request_id接受规范UUID并统一小写。
同一个已提交request_id+相同规范请求：返回原流水并replayed=true，不重复变更；request_id相同但数量/版本/原因/SKU/操作者不同：返回IDEMPOTENCY_CONFLICT。
没有命中时开启事务，统一锁product→product_sku→product_sku_inventory；使用FOR UPDATE，核对未删除、SKU归属、余额归属。商品下架或SKU禁用允许人工调整。
获得相关锁以后，再以当前读检查request_id，先幂等回放、后校验expected_version；不能因第一次提交已加版本而拒绝相同请求的重试。已提交请求的回放不受后来商品软删除影响。
用最新锁定余额计算，不信任前端的剩余数量。UPDATE可加version条件并检查RowsAffected=1；更新余额和插入流水同一事务，流水失败必须回滚余额及版本。
事务内部所有查询和写入使用同一个tx，不能调用持有外层db的图片、幂等或其他helper额外借连接。不能在事务外先查后写，不能只靠前端禁用按钮或进程mutex。
唯一键竞争时先回滚，再用新查询读取已提交幂等记录核验hash；精确区分请求唯一键、余额唯一键和版本唯一键冲突。死锁/锁等待做有界整事务重试或返回RETRYABLE，原请求参数和request_id不变。
COMMIT结果或响应不确定返回RESULT_UNKNOWN/允许原请求重试，不能保证“肯定没提交”。失败未提交的请求不占幂等键；幂等承诺针对已提交成功操作。

【接入商品与SKU】
新建商品各SKU、无规格默认SKU、编辑商品新增SKU时，在现有商品事务创建零库存。初始化失败整单回滚。
普通编辑商品、改价格/规格名/图片、禁用SKU、商品上下架都不改库存。商品Create/Update DTO不得增加可写库存字段。
SKU同组合恢复复用原ID和原库存，不reset；上线回填后仍出现已有SKU缺库存则明确报NOT_INITIALIZED，不在普通读取/恢复中悄悄填0。
商品删除改为先锁商品，再按ID升序锁全部SKU和库存（包括软删除SKU），拒绝缺库存、归属错误、非零stock或locked_stock。商品/SKU软删除后余额和流水仍保留，禁止级联物理删除或自动清零。
所有相关路径采用相同锁顺序，防止删除和调整并发穿透。已有商品回归测试fixture要补正确库存前置条件，不能删除旧断言来让测试通过。

【接口与权限】
沿用POST JSON和Response{code,message,total,data}，接口、@ID和DTO详见product库存方案.md第7节，必须实现：
- POST /api/product/inventories/list，@ID listProductInventories，product:inventory:list。
- POST /api/product/inventories/detail，@ID getProductInventory，product:inventory:list。
- POST /api/product/inventories/adjust，@ID adjustProductInventory，要求inventory:list及product:inventory:adjust。
- POST /api/product/inventories/warning，@ID setProductInventoryWarning，要求inventory:list及product:inventory:warning。
- POST /api/product/inventories/logs/list，@ID listProductInventoryLogs，仅要求product:inventory:log。
权限常量/权限定义接入现有Casbin和角色授权；没有库存写权限不能绕过商品Update改库存，不新增角色表或权限关系表。
库存列表从未删除SKU和商品LEFT JOIN库存，默认包含禁用和下架。返回缺库存SKU initialized=false、数量/version等为null，不能用0掩盖异常。detail缺库存明确失败。
列表筛选/nullable字段/库存状态按方案；分页20、最多100，总数与过滤一致，按SKU id DESC。keyword查当前商品名、SKU编码、条码；不要连接流水造成重复行，不出现N+1图片/商品/规格查询。
库存查询自行返回商品及规格展示信息，不要求调用者拥有product:product:list或system:file:list。图片只做业务引用解析，不扩大文件管理写权限。
流水查询以日志为主，按created_at DESC,id DESC；支持独立sku_id/product_id、request_id、类型、操作者、RFC3339时间区间[开始,结束)，包括已删除商品历史；不用软删除商品联表筛掉日志。
定义ProductInventoryRow、ProductInventoryLogRow、ProductInventorySKUSnapshot、ProductInventoryMutationResult及独立请求DTO，Swagger标记nullable和数组。
quantity、expected_version允许0且必须出现，使用指针/存在性校验，禁止binding:required误判0。
写成功data含log和replayed；业务错误code=0且data.error_code枚举按方案，message为可理解中文。不改公共Response结构；绑定失败只输出一次JSON。不能把底层SQL/连接信息直接回给前端。

【生成与验证】
先修改Go DTO和Swagger注解，再从根目录执行：
  bash scripts/3_generate_api.sh
  bash sw.sh
禁止手改generated_api.ts；本段负责生成客户端并确认前端可用。本段不实现前端页面。
新增必要测试放admin-server/internal/test，覆盖方案第9节，尤其：同请求并发只一条流水、旧版本冲突、并发扣减、流水失败回滚、提交后丢响应重试、合法0、非法小数/溢出、默认SKU初始化、存量回填保留余额、恢复保留库存、删除保护、单连接事务和接口权限。
在隔离MySQL验证CHECK、唯一键、行锁、回填和当前读；SQLite只能辅助验证，不能替代MySQL。不要使用真实业务库作为并发测试库。未具备依赖时清楚记录哪些没执行。
执行admin-server中的go test ./...、go build ./...；在admin-web执行pnpm run build及现有pnpm run test:product；根目录git diff --check。检查Swagger/client路径、类型和空值语义一致。
最终交付实际改动、迁移状态、接口和权限、入库/重试/版本冲突示例、验证结果与未执行的外部依赖检查。直接完成本段实现，不停留在方案，不扩大到订单和多仓库。
```
