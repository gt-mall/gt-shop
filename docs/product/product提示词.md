## 第一段：后端执行提示词

```text
请直接在 /home/shop/mall 实现商品模块后端，完成代码、Swagger、前端 API Client 生成和必要验证。本段先执行，完成后再执行前端提示词。

【范围与依据】
只实现 docs/product/product.md 中的 product_category、product、product_sku 三张表对应的分类管理、商品管理和商品内 SKU 编辑。库存、库存流水、订单、支付、品牌等本次不做，不创建占位接口或空模块。
先读取 AGENTS.md、docs/product/product.md、.agents/skills/go-module-structure/SKILL.md，再参考 admin-server/internal/system/role、internal/system/file、routers 和 pkg/response 的当前实现。
后端 Skill 暂未完善，按任务、真实表结构和现有代码判断；不要为了套用 Skill 引入尚未使用的生成器、目录或复杂架构。
三张表已经创建：先只读核对实际字段、类型、索引和软删除规则，按现有表映射，不重建、清空或擅自改变字段。连接不可用时依据文档继续完成代码，明确记录未验证的数据库差异。保留用户已有改动，不输出连接密码或 Token。
文档第一段规格 JSON 的“黑色”“白色”后有尾逗号，示例使用时修正；这不改变 JSON 的业务结构。ID 在 Go 使用 int64，已建表的 INT 不因类型偏好而自动迁移。

【实现组织】
业务放在 admin-server/internal/product 下，按实际需要划分 product、category、sku 小模块。商品与 SKU 的联合保存由一个 service 统一编排，避免循环依赖。
沿用 api.go、service.go、types.go；查询复杂时再增加 dao.go，不引入 Repository、DDD 或通用 CRUD 框架。
路由集中放在 admin-server/routers/product.go，并接入现有路由注册和鉴权。
模型放 internal/model，显式映射单数表名 product、product_category、product_sku。检查 domain.go 的旧 Product/SKU 占位模型，处理命名及字段冲突，不再连接旧 products/skus 表，也不修改无关订单模型。
当前仓库未启用 GORM Gen 查询目录；若实施时仍如此，直接使用现有 GORM 写法和手写模型。只有实际存在的生成文件才禁止手改，不为本任务强行安装生成器。

【金额与数据格式】
保留 price、market_price、cost_price 的 DECIMAL(12,2)，单位元；API 请求及响应统一十进制字符串，例如 "199.90"，Swagger 也声明为 string。
读写及校验不能经过 float32/float64。复用已有精确金额能力；没有时选择简单的精确十进制实现。需要普通加减、数量乘法时可精确转换为 int64 分，保存前转回元。输入最多两位小数，校验非负和数据库范围，禁止静默截断；不提前实现订单优惠计算。
数据库可空字段明确清空语义：图片和重量可传 null；sku_code、barcode 去除首尾空白，未填写存 NULL，不能把所有空 sku_code 存成空字符串触发唯一索引。
specs、spec_values、gallery_file_ids 在 API 中返回真正的数组，空值统一 []，不返回转义后的 JSON 字符串。数据库原有 JSON NULL 读取时兼容。
禁止直接把 GORM 模型作为可任意写入的请求体；请求和响应使用明确 DTO，避免客户端覆盖 deleted_at、sales_count、spec_hash 等服务端字段。

【规格与 SKU 规则】
ProductSpec：id 为正整数，name 为字符串，values 为 SpecValue 数组；SpecValue：id 为正整数、name 为字符串。
SKUValue：spec_id、value_id 都为正整数。
商品 specs 示例：
[{"id":1,"name":"颜色","values":[{"id":1,"name":"黑色"},{"id":2,"name":"白色"}]},{"id":2,"name":"容量","values":[{"id":3,"name":"128GB"},{"id":4,"name":"256GB"}]}]
某个 SKU 的 spec_values 示例：
[{"spec_id":1,"value_id":1},{"spec_id":2,"value_id":4}]
规格 ID 在商品内唯一，选项 ID 在所属规格内唯一。ID 创建后稳定，不能用数组下标重新编号，不能复用仍被历史 SKU 引用的 ID 来表示另一种规格。
后端校验规格名及选项名非空、同层名称不重复；每个 SKU 对每个规格恰好选择一个有效选项，不接受漏选、多选、未知 ID 或重复组合。
spec_hash 只由后端计算：按数字 spec_id 升序排列，拼接成 1:1|2:4，再取 SHA256 的小写 64 位十六进制。不能直接对未经规范化的 JSON 文本计算，也不能信任客户端 hash。
无规格商品 specs=[]，仍须有且仅有一个默认 SKU，spec_values=[]，hash 固定对字符串 default 计算。
第一版限制最多 3 个规格维度、每维最多 20 个选项、每个商品最多 200 条未删除 SKU（含禁用）。后端限制实际提交的 SKU 数量，不要求保存完整笛卡尔积；有效组合可以只销售其中一部分。
已有 SKU 保留数据库 ID 和规格身份；改名、改排序不重建 SKU。允许新增选项和组合，已保存 SKU 在本期只启用/禁用，不提供单独删除入口。删除规格值或增删规格维度若导致已有 SKU 无效，应拒绝并给出明确原因。
兼容库内已有软删除 SKU：同一商品重新提交相同组合时恢复原记录及 ID，不能新插入撞唯一索引。sku_code 全局唯一且软删除后仍占用；与其他记录冲突时明确报错，不自动修改或释放旧编码。

【分类与商品业务】
分类支持层级、名称、图片、排序、启用/禁用及软删除。parent_id=0 为顶级；禁止父级指向自身或后代，父级必须存在且未删除；并发改父级时也要保持无环。
有未删除子分类或商品引用的分类不能删除；禁用不等于删除，不批量改写商品状态。
商品支持名称、副标题、分类、主图、轮播图片、有序规格、富文本详情、排序、上下架、虚拟销量，以及 SKU 全部业务字段。sales_count 为只读，新增默认 0，本期不模拟或递增实际销量。
下架商品允许 category_id=0 和暂缺图片；任何已保存商品至少有一个 SKU。上架要求分类有效且启用、主图有效、至少一个启用 SKU，且所有启用 SKU 的销售价格大于 0。商品状态只使用已有 0下架/1上架，不增加数据库未定义的草稿状态。
商品删除使用软删除，与关联 SKU 的软删除放在同一事务；不物理删除，也不删除引用图片文件。
新增、编辑商品时一次提交主表和 skus，事务内校验并保存，任何 SKU 编码/规格/价格出错都整体回滚。更新时锁定商品记录，校验所有传入 SKU ID 都属于该商品，拒绝跨商品写入。
编辑是完整聚合保存，必须提交所有未删除 SKU（包括禁用项）；漏传已有 SKU 报错，不把漏传解释成删除。已保存 SKU 通过 status 禁用，新建但未保存的行可不提交。
图片复用 system_file 和现有解析能力：入库保存 ID，轮播最多 10 张且去重保序；列表批量解析主图，详情和编辑可解析已被素材库软删除但仍受业务引用的旧图片。新增或替换图片要校验有效性，不因旧引用被素材库隐藏而禁止商品其他字段更新。
富文本 detail 使用服务端 HTML 白名单清洗，保留当前 RichEditor 所需的合理图文/表格结构，过滤脚本、事件属性和危险协议；优先复用现有能力，不用正则充当 HTML 清洗器。

【接口与前端交接契约】
沿用现有 POST + JSON 风格和 Response{code,message,total,data}；code=1 才是成功。下面路径作为本次约定，如实施时已有同等接口则复用，并在 Swagger 中统一实际契约。
分类接口：
- POST /api/product/categories/list：返回全部未删除分类的扁平数组，含 id、parent_id、name、image_id、image_url、sort、status、created_at、updated_at；前端组装树和过滤。
- POST /api/product/categories/options：返回轻量完整分类选项 id、parent_id、name、status、sort，包含禁用项以支持旧值回显。
- POST /api/product/categories/create、update、delete：新增/编辑/软删除；update 包含 id，delete 请求为 {id}，分类状态在 update 中修改。
商品接口：
- POST /api/product/products/list：请求 page、page_size、keyword、category_id、status；状态 0 必须生效。keyword 搜索商品名称/副标题及其未删除 SKU 编码/条码，分类筛选包含所选分类及未删除后代，未传分类表示不过滤，传 0 只查未分类。
- list 返回分页 total 和商品行数组，含主表列表字段、category_name、main_image_url、sku_count、min_price、max_price；价格范围只统计未删除且启用的 SKU，无启用 SKU 时两项为 null。sku_count 包含禁用但未删除的 SKU。不要在列表返回完整富文本和 SKU 集合，不制造关联查询重复行或 N+1 查询。
- POST /api/product/products/detail：请求 {id}，返回主表全部可展示字段和 skus。每个 SKU 含 id、spec_values、sku_code、barcode、image_id、price、market_price、cost_price、unit、weight_gram、sort、status；返回禁用但未删除的 SKU。
- POST /api/product/products/create、update：请求为商品可编辑字段加 skus，update 额外带商品 id；已有 SKU 必须带 id，新 SKU 不带 id。成功返回商品 id。新增商品使用 status=0 默认值，编辑不得重置实际销量。
- POST /api/product/products/status：请求 {id,status}，独立修改上下架状态，执行同样的上架校验，避免列表操作必须重传整个商品。
- POST /api/product/products/delete：请求 {id}，执行聚合软删除。
分页默认 20、最多 100，默认 sort ASC、id DESC；过滤条件、总数和列表范围保持一致。
定义清晰 Swagger 类型名，例如 ProductSpec、ProductSpecValue、ProductSKUValue、ProductRow、ProductDetail、ProductSKU、ProductCategoryRow。必填字段、可空字段、数组、价格字符串和状态值在生成客户端中必须准确。
注册 product:category:list/create/update/delete、product:product:list/create/update/delete 权限，复用现有登录与权限中间件。商品详情使用 list 权限，上下架和商品内 SKU 修改使用 product:product:update，不另建 SKU 权限体系。
分类 options 允许具有分类 list 或商品 list/create/update 中任一权限的用户读取，不能要求商品编辑者额外拥有分类管理权限；不扩大分类写权限。
权限进入现有权限定义与角色授权流程，菜单和前端按钮随后使用同一组 code。图库选择/上传继续使用现有 system:file 权限，不绕过或自动赋予全员。
先维护后端 DTO 和 Swagger 注解，从仓库根目录依次执行 bash scripts/3_generate_api.sh、bash sw.sh。当前 sw.sh 只生成前端 Client，不能省略前一步；不手改生成文件。本段允许更新生成的前端 Client，但不编写前端商品页面。

【验证与交付】
新增必要测试放 admin-server/internal/test，覆盖规范化 hash（顺序变化仍相同）、默认 SKU、非法规格/重复组合、金额精度/范围、SKU ID 保留及归属、软删除组合恢复、分类循环/引用删除、事务回滚和权限。
数据库相关测试使用隔离测试环境，不依赖或清空现有业务数据；SQLite 测试不能代替 MySQL 对 DECIMAL、JSON、唯一索引和事务行为的验证，MySQL 未执行的检查如实记录。
执行 admin-server 中的 go test ./...、go build ./...，生成 Swagger/Client 后在 admin-web 执行 pnpm run build，最后 git diff --check。只读核实实际接口响应及运行版本，避免把旧服务进程的结果当成新代码结果。
交付时列出改动文件、实际接口和权限、商品创建及详情响应示例、验证结果和未完成的外部依赖检查。直接完成范围内实现，不只交付方案，也不顺手改动无关模块。
```

## 第二段：前端执行提示词

```text
请直接在 /home/shop/mall 实现商品模块管理后台页面。本段在商品后端和生成 API Client 已完成后执行；交付可连接真实后端的页面、交互和构建验证。

【范围与依据】
本次只围绕 docs/product/product.md 的 product_category、product、product_sku，实现商品分类、商品列表和商品新增/编辑。SKU 放在商品编辑页，不建独立 SKU 菜单；不做库存、订单、品牌等功能，不用假数据代替实际接口。
先读取 AGENTS.md、.agents/skills/admin-web/SKILL.md 及 references/crud.md、references/permission.md，检查后端 Swagger 和 admin-web/src/framework/http/generated_api.ts。后端 Skill 尚未完善，不要据此推翻真实 API Contract。
页面只消费现有 HTTP 能力，无需改 Axios；只有实际要改公共 HTTP 层时才读取 references/http.md。接口缺失或与需求不符时，明确列出阻塞点，先完成不依赖它的页面部分；不能猜接口、伪造成功或手写替代生成类型。
参考现有 modules/system/user、modules/system/role 的列表和表单风格；保持当前 ProLayout、主题、顶部大模块和左侧子菜单布局。

【目录与复用】
页面放 admin-web/src/modules/product/category 和 modules/product/product。商品列表用 product/index.tsx，新增/编辑共用 product/edit.tsx；规格及 SKU 编辑复杂时再拆 spec-editor.tsx、sku-table.tsx，不创建空目录或通用 CRUD 框架。
分类和商品列表使用普通 antd Card、Form、Input、Select、Table；分类新增/编辑共用 Modal + Form，不使用 ProTable。
商品内容较多，使用完整编辑页，按基本信息、图片与详情、规格与 SKU、其他设置组织 Card 或 Tabs；不要把商品编辑硬塞进小弹窗。
图片直接复用 @/modules/system/file 的 ImagePicker：分类图片、商品主图、SKU 图片单选；轮播使用 multiple，最多 10 张，提供简单上移/下移操作保存图片顺序，不为排序另装拖拽库。
富文本直接复用 @/shared/components 的 RichEditor，参考 modules/demo/editor；Form.Item 绑定 HTML 字符串，不新装第二套编辑器。图片预览复用已有文件解析能力，不在页面手拼存储域名。
路由接入 framework/router/index.tsx，菜单接入 framework/layout/layout.tsx；新增“商品管理”大模块，下设“商品列表”“商品分类”，不重构无关导航。

【接口与格式】
只使用 @/framework/http 导出的 api 和生成类型，实际方法名以 generated_api.ts 为准。默认后端接口如下：
- POST /api/product/categories/list、options、create、update、delete。
- POST /api/product/products/list、detail、create、update、status、delete。
统一读取 Response 的 code、message、data、total，code===1 才更新成功状态；列表请求使用 page、page_size、keyword、category_id、status。
价格 price、market_price、cost_price 都是单位元的十进制字符串，例如 "199.90"；使用 InputNumber 的 stringMode 或能保留字符串的等效控件，最多两位小数。禁止 Number、parseFloat、浮点乘 100 或先转浮点再格式化金额；空值校验与 0 值区分，提交前按字符串规则补齐两位小数。
specs、spec_values、gallery_file_ids 是数组；缺省数组使用 []。图片清空提交 null，SKU 编码/条码未填写按契约提交 null 或空白由后端归一化，不用 undefined 代替“明确清空”导致旧值残留。
沿用前端 Skill 的异步规则：手写代码只使用 async/await，禁止 .then/.catch/.finally 和 try/catch/finally。表单使用 onFinish/onFinishFailed，避免自行 await 可能 reject 的 validateFields；API 异常由公共 HTTP 层转换为 Response，业务依据 code/message 收尾 loading 和 saving。
提交失败保留表单输入，防止重复提交；详情加载失败展示错误与重试，不显示可误保存的空编辑表单。加载/保存状态在成功和业务失败分支均能恢复。

【分类页面】
分类 list 返回全部未删除分类扁平数组，在页面组成树形 Table，展示图片、名称、排序、启用状态和操作。
支持名称查询及状态筛选，过滤树时保留命中节点的祖先；查询清空后恢复完整树。
支持新增顶级分类、添加子分类、编辑、启用/禁用和删除。表单含 parent_id、name、image_id、sort、status，parent_id=0 表示顶级。
父级使用 TreeSelect，编辑时禁选自身和后代；有子分类或商品引用时展示后端拒绝原因，不做前端假删除。
表单编辑后再次打开新增要清空旧值。删除使用 Popconfirm，成功后刷新树。

【商品列表】
查询区含关键词、分类树选择、上下架状态、查询和重置。关键词交给后端搜索名称/副标题/SKU 编码/条码；分类筛选包含后代，由后端处理。
分类来自 categories/options，保留禁用分类用于历史记录筛选；“全部”和“未分类”分别对应不传 category_id、传 0，不混用。status=0 不能因真假值判断被丢弃。
展示主图、商品名称/副标题、分类、价格或价格区间、SKU 数量、实际销量、虚拟销量、状态、排序和更新时间。无启用 SKU 导致价格范围为 null 时显示“—”，不要显示成 0 元。
操作包含新增、查看、编辑、上下架和删除。上下架调用 status 接口；删除商品有确认；实际销量只展示不编辑。
查询和重置回到第一页，更改页大小回到第一页，删除当前页最后一条时调整分页。请求失败不清空已有列表，快速切换查询时避免旧响应覆盖新结果。
建议路由：/product/product 列表、/product/product/create 新增、/product/product/:id/edit 编辑、/product/product/:id 查看、/product/category 分类。查看复用编辑表单的只读模式，不显示保存操作。

【商品新增/编辑】
编辑必须调用 detail 回填完整商品和全部未删除 SKU，包含禁用行，不用列表行拼凑详情。
表单字段覆盖 name、subtitle、category_id、main_image_id、gallery_file_ids、detail、specs、sort、status、virtual_sales；sales_count 只读，更新不提交它。
下架允许未分类和暂缺图片；上架要求有效且启用分类、主图、至少一个启用 SKU，且启用 SKU 售价均大于 0。前端提示与后端校验一致，不新增“草稿”状态。
新增默认下架、无规格且有一条默认 SKU；新增商品可以切换多规格。已有商品切换规格模式或增删维度若使已保存 SKU 无效，应禁用操作并说明原因，不能偷偷重新生成 SKU ID。
编辑分类时可回显原有禁用分类，但不能把它当成可上架的有效分类。图库选择/上传受现有 system:file 权限控制；没有图库操作权限仍能预览和保留原图片。
编辑提交为完整商品加 skus，已有 SKU 带数据库 id，新 SKU 不带 id，所有未删除 SKU 都提交。一次保存调用 create 或 update，不循环调用多个请求分别保存商品和 SKU。
保存成功返回列表并刷新；保存失败保留用户输入。离开有未保存修改的编辑页时使用现有或简单确认交互，避免误丢编辑内容。

【规格与 SKU 编辑交互】
规格结构为 [{id,name,values:[{id,name}]}]，SKU 选择结构为 [{spec_id,value_id}]；ID 为稳定正整数，表格 key 不使用可变名称或数组下标。
最多 3 个规格维度、每维最多 20 个选项、每商品最多 200 条 SKU（含禁用）。新规格/选项分配稳定 ID，不能复用本次编辑删除的 ID，也不能修改已保存规格的身份。
无规格商品 specs=[]，只有一行“默认规格”，该行 spec_values=[]。
多规格提供规格名、选项增删和排序，以及明确的“生成/补充 SKU”操作。根据有效选项生成组合前先估算数量，超过 200 时阻止全量生成并提示缩减；已有稀疏组合可以正常编辑，不要求补齐全部排列。
合并 SKU 时按 spec_id 排序后的 ID 组合匹配，保留已有行的 id、价格、编码、图片、状态等输入；改显示名称或排序不会清空价格或重新创建行。
已保存 SKU 只提供启用/禁用；未保存的新组合可移除。删除规格值或维度会影响已保存 SKU 时阻止并提示，影响未保存行时确认后再移除相关行。
SKU 表格展示规格组合、图片、sku_code、barcode、price、market_price、cost_price、unit、weight_gram、sort、status。规格选择来自商品 specs，不允许输入自由 JSON。
支持给选中行批量设置售价、划线价、成本价、单位和启用状态；没有选中行时按钮禁用，批量动作明确作用范围，不覆盖其他列。
检查空规格名、空选项、重复名称、漏选或重复组合、SKU 编码重复、负价格、超两位小数、负重量等；后端仍负责最终校验。spec_hash 是后端内部去重字段，不在页面编辑或自行计算 SHA256。

【权限、验证与交付】
复用 framework/store 的 usePermission 和现有 PermissionRoute，菜单、路由及按钮使用 product:category:list/create/update/delete、product:product:list/create/update/delete；商品内 SKU 和上下架使用商品 update 权限。
商品页面统一要求 product:product:list，新增页再要求 create，编辑页再要求 update，查看页只要求 list；这与后端 detail 使用 list 权限一致。分类 options 不应额外要求分类管理权限，不在前端硬编码 super_admin 绕过权限。
角色配置按 list 加对应操作权限组合授权，缺少权限时使用现有无权限页面和提示，不能为了加载分类/详情而扩大写权限或绕过后端。
对照真实 Swagger 验证接口字段；有契约问题先反馈具体差异，不手改 generated_api.ts 或自行使用 any 绕过。
执行 admin-web 中的 pnpm run build，以及仓库根目录 git diff --check；有浏览器能力时实际检查分类 CRUD、默认/多规格商品、SKU 数据保留、价格字符串提交、图片/富文本回填、失败保留输入、上下架及无权限访问。
复用已获授权的测试环境与测试数据，不改动真实商品；没有浏览器或后端不可用时，如实记录未完成的联调检查，不能声称已全部验证。
交付改动文件、页面入口、已实现交互、构建和联调结果，以及仍需后端处理的具体问题。直接完成范围内页面，不只给方案。
```
