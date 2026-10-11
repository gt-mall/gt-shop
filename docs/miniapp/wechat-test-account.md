# 微信小程序开发测试号配置说明

本项目微信小程序端（`miniapp`）开发与联调所使用的微信公众平台测试号配置如下：

| 配置项 | 详细值 | 说明 |
| :--- | :--- | :--- |
| **小程序 ID (AppID)** | `wxc5edb3e825639871` | 用于小程序编译标识、微信开发者工具身份认证 |
| **小程序密钥 (AppSecret)** | 仅在微信公众平台查看，不写入仓库 | 用于服务端通过 Code 换取用户 OpenID 与 SessionKey |

AppSecret 仅保存在服务端安全配置中，不能写入小程序源码或公开文档。本文曾包含明文 AppSecret；如果仍在使用，请在微信公众平台重置，并更新服务端配置。删除本文中的值不会清除 Git 历史。

---

## 工程配置与生效路径

1. **小程序端工程配置**：
   已写入 [`miniapp/src/manifest.json`](../../miniapp/src/manifest.json)：
   ```json
   "mp-weixin": {
     "appid": "wxc5edb3e825639871",
     "setting": {
       "urlCheck": false,
       "es6": true,
       "minified": true
     },
     "usingComponents": true
   }
   ```

2. **微信开发者工具运行说明**：
   - 导入构建目录：`miniapp/dist/build/mp-weixin`（或开发监听目录 `miniapp/dist/dev/mp-weixin`）；
   - 工具会自动识别项目 AppID 为 `wxc5edb3e825639871`；
   - 开发者使用微信扫码登录微信开发者工具，即可脱离“游客模式”，完整支持：
     - 原生微信头像选择器（`open-type="chooseAvatar"`）；
     - 微信昵称键盘一键填入（`type="nickname"`）；
     - 微信一键登录（`uni.login` 换取真实 Code）。
