# Plugin

第一版插件采用编译期注册：

1. 在 `admin_core/internal/plugin/<name>/` 实现 `plugin.Plugin`。
2. 在 `admin_core/internal/plugin/plugin.go` 的 `RegisterAll` 中显式加入插件。
3. 在 `admin_web/src/plugins/` 增加前端入口和页面。
4. 维护 Go 注解并重新生成 Swagger 与 `src/generated`。

当前 `hello` 插件提供 `GET /api/plugins/hello`。不使用 Go `.so`、微前端、Module Federation 或动态下载安装；未来扩展只需要增加编译期模块和注册项。
