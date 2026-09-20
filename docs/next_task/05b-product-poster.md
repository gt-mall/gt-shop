# 05B 商品海报与扫码落地

前置：[05A](05a-native-share.md) 的商品 id/scene 解析约定已完成。海报包含商品图、名称、真实价格、商城名称和可识别到该商品的小程序码；不能把通用商城二维码冒充商品码。

## 现有可复用能力

- `internal/system/setting` 已有 MpConfig.AppID/AppSecret，公开配置会剔除敏感字段。
- `internal/miniapp/auth/service.go` 有微信 Code2Session 客户端，但当前未找到服务端 access_token 管理或商品小程序码接口，不能误把用户登录 Token 当微信接口 access_token。
- `internal/system/file` 有存储适配、文件解析和 `SceneSystemGenerated`；miniapp 有 `resolveImageUrl` 及图片上传/预览模式，尚无海报 Canvas。

## 执行提示词

读取 README 和 05A。后端负责商品小程序码，前端负责 Canvas 排版，第一版固定一套模板，不引入在线设计器。

### 后端：商品小程序码

1. 新增小型分享服务和微信 API 适配器。建议 `POST /api/miniapp/products/:id/share-code`，生成操作需会员登录；扫码后的商品页仍公开。请求只接受商品 ID，不接受任意微信 page、任意下载 URL 或任意 AppID。
2. 先验证商品上架/未删除；固定 page 为现有商品详情注册路径，scene 使用 05A 的 `p=<id>`。微信接口需要的 page 是否带斜杠、scene 长度/字符集、env_version/check_path、已发布页面等按执行时官方契约核实。
3. AppSecret/access_token 只留在服务端，按 AppID 和有效期缓存 token，提前续期、并发合并；凭证变更后失效旧缓存。Redis 不可用时能安全降级，不能无限重试微信接口。
4. 微信返回可能是图片二进制，也可能是 JSON 错误；检测状态/MIME/实际内容、限制响应大小，错误不可保存成图片。超时、无效配置、频控、页面不存在返回可理解的错误；不把 Secret、完整 token 或原始响应打到日志。
5. 相同 AppID + 发布环境 + 商品 ID + 落地版本复用一个小程序码，控制生成频率与并发。持久化映射可用小型 `product_share_code` 表，唯一键约束上述身份并关联 file_id；需要表时新增下一编号 Migration，不把 token 存入表。仅缓存二维码，海报随最新商品信息重新生成。
6. 生成的二维码通过已有系统文件服务以 system_generated 场景保存；返回 `file_id/url/product_id`，不直接暴露文件系统路径。资源保存失败要补偿，不留成功映射指向不存在文件。
7. 注入可 Mock 的微信客户端，对 token 续期、二维码成功/JSON 错误/超时/并发复用写真实服务测试；完成 Swagger 与小程序薄 API。

### 小程序：绘图、预览与保存

1. 商品详情“生成海报”打开可复用 `ProductPoster` 弹层，输入 productId，通过真实商品接口/已加载商品数据校验与刷新，组件不依赖详情页私有变量。提供说明和实际接入示例。
2. 固定布局：商城名 → 商品图 → 最多两行商品名 → 价格 → 商品码及“扫码查看商品”。没有主图显示合适占位；二维码失败时不能输出看起来可扫描的假码，不保存不完整海报。
3. 使用 UniApp 支持的 Canvas 实现，微信与 H5 必要处适配；组件内 Canvas 查询使用正确实例作用域，等图片加载/绘制完成后再导出，按 DPR 设置绘图与导出尺寸，保证文字及二维码清晰。
4. 图片下载/临时文件须有 Loading、超时、失败重试和大小约束；微信配置 downloadFile 合法域名，H5 处理 CORS/画布污染。前端只能下载后端返回的可信资源，不增加无校验的服务器任意 URL 代理。
5. 分享码不能裁剪、拉伸或被装饰覆盖；名称自动换行，超长省略，价格显示复用精确金额格式，不做浮点重算。
6. 展示预览并由用户主动保存：微信保存至相册并处理拒绝权限/取消/H5 不支持情况；H5 使用可用的图片下载或长按提示。拒绝权限时保留预览，提供用户主动恢复方式。
7. 连续点击只生成一次；切商品/关闭/卸载后旧绘制结果不覆盖新海报，释放临时资源。成功提示仅在实际导出/保存成功后出现，不把分享视为已完成交易。
8. 海报展示价格可能过期，注明以商品页面为准；不附带营销收益、推荐人、用户身份和订单资料。

## 验收

- 已登录生成、未登录扫码打开商品的完整真机链路；开发/体验/正式环境分别验证，不把游客 AppID 验证当发布成功。
- 真实主图、缺图、中文长标题、极端价格长度、小屏、不同 DPR 均不遮挡二维码。
- token 失效、微信 JSON 错误、下载失败、Canvas 导出失败、拒绝相册权限及快速开关可恢复。
- 多次生成复用商品码，价格修改后海报更新；商品下架后扫码显示真实不可购状态。
- 执行 README 后端/小程序检查；缺 AppID 配置、合法域名或微信服务访问时列明，不能用假图片宣称联调完成。

官方资料：[Canvas 导出](https://uniapp.dcloud.net.cn/api/canvas/canvasToTempFilePath.html)、[微信商品码 API 文档入口](https://developers.weixin.qq.com/miniprogram/dev/server/API/qrcode-link/qr-code/api_getunlimitedqrcode.html)。微信文档本次未能抓取，执行时须重新查证参数与环境要求。
