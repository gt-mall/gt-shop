# 库存前端 AI 执行提示词

在 [后端提示词](product库存后端提示词.md) 实现并生成客户端之后执行。页面、字段和 API 的完整约定见 [库存方案](product库存方案.md)。

复制下面整段给前端 AI：

```text
请直接在 /home/shop/mall 实现商品库存管理前端，交付可连接真实后端的页面、权限、操作交互和构建验证。本段不新写后端业务，也不手改生成API。

【先读与范围】
读取AGENTS.md、.agents/skills/admin-web/SKILL.md及references/crud.md、references/permission.md；读取docs/product/product库存方案.md和当前后端Swagger、admin-web/src/framework/http/generated_api.ts。
后端应已实现/api/product/inventories/{list,detail,adjust,warning,logs/list}。先核对实际方法和DTO，任何缺失/不一致要记录明确阻塞点，继续完成独立页面部分；不能猜接口、伪造成功、手写平行API类型或用假数据替代交付。
本次仅单仓库SKU库存管理、人工入库/出库/盘点、预警阈值设置、只读流水。locked_stock只读，本期初始化为0；若后端返回非零锁定量，必须照实展示和计算，不能写死0。不做订单、仓库选择、采购、调拨、批次、批量调整或库存成本。
保留已有商品和SKU编辑功能、金额处理及回归修复，禁止重新生成/重置已保存SKU ID。

【目录、页面与复用】
新增admin-web/src/modules/product/inventory/index.tsx（库存列表）和log.tsx（流水列表），调整Modal复杂时再拆adjust-modal.tsx；预警可复用简单Modal + Form。不建立通用CRUD框架，不引入ProTable或新状态库。
使用普通antd Card + Form/Input/Select/Table + Modal + Form，保持现有ProLayout布局、主题和表格风格。
路由接入framework/router/index.tsx，菜单接入framework/layout/layout.tsx，在现有“商品管理”下添加：
- /product/inventory，库存管理，product:inventory:list。
- /product/inventory/log，库存流水，product:inventory:log。
流水页可独立授权，不嵌在必须有inventory:list权限的路由里。PermissionRoute使用现有声明式写法；菜单、按钮和路由共用framework/store权限，不解析JWT。
商品列表及已有商品编辑/详情增加有inventory:list才显示的“查看库存”链接，跳转/product/inventory?product_id=...。新商品未保存没有SKU ID，不显示库存入口。
商品编辑页不增加库存输入框，也不把stock等字段拼进商品Create/Update请求。

【接口和异步规范】
只使用@/framework/http导出的api和自动生成DTO，方法按实际生成结果：listProductInventories、getProductInventory、adjustProductInventory、setProductInventoryWarning、listProductInventoryLogs。
统一code===1才是成功，错误显示message，分页读取total与data。失败保留用户输入，不能伪造余额变化或成功消息。
手写异步只用async/await；禁止try/catch/finally和.then/.catch/.finally。网络异常由公共HTTP层转统一Response；本任务不改Axios。仅在确实需要修改公共HTTP时另读references/http.md。
未知、可空字段按Swagger处理，不能用类型断言掩盖契约差异。image_url直接展示，不手拼存储域名，不安装第二套图片组件。

【库存列表】
请求包含page、page_size、keyword、product_id、sku_id、product_status、sku_status、stock_filter，字段含义按方案。分页20，可选20/50/100；筛选或重置回到第1页，刷新保留当前筛选。
支持关键词（商品名称/SKU编码/条码）、商品和SKU上下架/启用状态、低库存、缺货、正常、未初始化筛选。状态0有效，不使用value || undefined丢掉0筛选。
默认同时显示禁用SKU与下架商品，库存管理不能只展示在售SKU。
展示商品图片和名称、SKU编码、规格组合、单位、商品/SKU状态、总库存stock、锁定locked_stock、可用available_stock、预警阈值和最近更新时间。数量不用金额格式化。
字段名称明确为“总库存”“锁定库存”“可用库存”；不能把stock显示为可用量。available_stock由后端提供，页面预览可按整数公式辅助计算。
默认SKU显示“默认规格”；其他规格使用后端名称，涉及ID匹配时用(spec_id,value_id)，不能只按value_id找名字。图片缺失使用现有占位。
initialized=false或可空余额为null时显示“未初始化/—”，保留行但禁止任何调整，不能当0库存处理。
库存预警使用后端is_low_stock、is_out_of_stock；warning_stock=0显示“未开启”，不是预警为0件。低库存和缺货可同时提示；不自动改商品状态。
表格操作：入库/出库/盘点要求product:inventory:adjust；设置预警要求product:inventory:warning；查看流水要求product:inventory:log。无权限按钮不展示。
不要先乐观扣减表格数量；成功后重新查询以获得服务端最新余额。过滤后空列表、加载、失败、重试都要有现有风格的反馈。

【调整弹窗】
点击入库/出库/盘点/预警时先调用detail获取最新余额及version，成功才打开可提交表单；接口失败/未初始化/记录删除时显示错误，不使用旧列表数据继续写。
展示商品、SKU、当前总量/锁定/可用，字段只读；表单包括操作数量和必填原因（trim后1..500字符）。预警也必须有原因。
- 入库manual_in：标签“本次入库数量”，正整数。
- 出库manual_out：标签“本次出库数量”，正整数，不超过最新可用量；前端校验不能代替后端并发校验。
- 盘点stocktake：标签“盘点后总库存”，非负整数，允许0，不能小于锁定库存；展示本次差额和预计可用量，避免误输成差额。
- 预警warning接口：标签“预警阈值”，非负整数，允许0关闭预警；请求quantity就是该阈值，服务端固定warning_set。
所有数量最大2147483647，用InputNumber和明确整数校验；禁止把小数四舍五入后悄悄提交。保留0，不写quantity || 默认值。盘点/阈值未变化前端提示无需保存，后端NO_CHANGE同样正确展示。
提交请求仅使用sku_id、request_id、expected_version、change_type（warning接口不传）、quantity、remark。不能提交operator_id、product_id、inventory_id、before/after或直接改locked_stock。
操作者身份由后端确定，提交记录只作为当前登录用户的待确认操作保存。

【重复提交与网络结果不确定】
每次用户确认一项新的业务操作时生成一个UUID request_id，使用crypto.randomUUID；检查部署安全上下文是否支持，缺少时复用项目可靠UUID能力，不使用时间戳或Math.random冒充UUID。
一次提交冻结完整payload（包括当时version、原因和数量），同步设置提交保护，loading时禁用输入和按钮；不要只靠React下一帧才更新的loading阻止同一事件内双击。
保留这份payload直到收到明确成功或明确业务拒绝。超时、断网、RESULT_UNKNOWN、RETRYABLE或返回缺少data.error_code的错误时，不假定库存未变，不生成新UUID；提供“确认上次操作结果/重试”，发送同一份原payload。
将一个待确认请求按当前用户ID保存在sessionStorage以支持F5恢复，不能存Token；恢复时核对用户，不让另一个账号重放。待确认时不允许编辑payload后继续使用旧UUID。重新打开页面/弹窗要优先恢复待确认操作，不静默丢弃后让用户重复入库。
成功code=1时从data.log读取本次结果；replayed=true也按成功处理。提示操作已完成，清除待确认状态，再刷新列表/详情及已打开的相关流水；不要把历史回放的stock_after覆盖成当前余额。
VERSION_CONFLICT：说明库存已变化，清除已明确拒绝的请求，重新读取detail，展示新余额，让用户检查数量和原因后再提交，使用新UUID。盘点不得自动换新version重试覆盖别人变更。
INSUFFICIENT_STOCK、NOT_INITIALIZED、NOT_FOUND、NO_CHANGE、INVALID_ARGUMENT按明确业务错误展示，保留适当输入并解除loading。
IDEMPOTENCY_CONFLICT提示本次请求标识异常，停止自动重试；不能偷偷换UUID继续入库来绕过冲突。
异常情况下不显示实现术语、hash或UUID作为日常操作步骤，用户只需要知道是否已完成、是否需检查新余额和如何重试。

【库存流水页】
路由/product/inventory/log，支持URL里的sku_id、product_id初始筛选，查询参数校验为正安全整数。库存列表“流水”链接带sku_id；商品入口可带product_id。
支持类型（人工入库/人工出库/盘点校正/设置预警）、时间范围、SKU/商品ID、操作者ID、request_id筛选；后两项可放高级筛选，界面分别标“操作者ID”“操作编号”。不为了筛选强制请求系统用户管理接口。
查询区间按后端RFC3339和[开始,结束)发送。若使用日期而非时分秒，结束日转换为下一日零点；明确时区，不能少查所选最后一天。
列至少有时间、商品/SKU历史快照、操作类型、总库存变化（正负号）、总库存前后、锁定前后、可用前后、操作者和原因；预警变化在warning_set行显示“原阈值→新阈值”，不要显示成一次0数量出入库。其他细节可用展开行展示。
只展示后端sku_snapshot/operator_name，不用当前商品详情覆盖历史名称；商品已删除也照常可读。页面只有查询/刷新，无修改/删除流水入口。
URL中product_id/sku_id与页面筛选保持一致，F5、深链接、返回和重置都正确；清空条件同步清理URL。没有product:product:list或system:user:list仍可查看被授权的流水。

【验证与交付】
验证：菜单/路由/按钮权限，只有查看权限不可写，流水可独立授权；列表含禁用/下架SKU；未初始化不显示0；默认SKU和不同维度同value_id名称正确。
验证：入库出库、盘点目标0、阈值0、负数/小数/溢出、无变化；双击只产生一个操作；版本冲突刷新后需用户确认；超时/提交后丢响应重试沿用相同UUID和payload；F5恢复同用户待确认请求；不能跨账号重放。
验证：库存操作后刷新余额和流水、时间范围末日、URL筛选/F5/重置、已删除商品历史流水、商品编辑不会覆盖库存。
使用真实接口或明确标注的隔离测试服务，不能把mock结果说成真实后端验证。不具备浏览器/后端环境时明确记录未执行项。
在admin-web执行pnpm run build和现有pnpm run test:product；为新库存状态/请求边界增加必要可重复验证，避免只测实现细节。根目录执行git diff --check。
最终交付改动文件、页面路由、使用接口/权限、关键交互和验证结果，不顺手改其他业务，不以静态页面代替真实库存流程。
```
