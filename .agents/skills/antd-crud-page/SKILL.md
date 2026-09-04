---
name: antd-crud-page
description: 为 Mall admin-web 新增或精简普通列表 CRUD 页面。适用于基于现有生成 API 实现查询、分页、新增、编辑、删除；不适用于复杂编辑器或图表页面。
---

# Antd CRUD Page

以 `admin-web/src/pages/UserPage.tsx` 为样本，保持页面短小、可直接复用。

## 规则

- 使用 antd `Card + Input.Search + Table + Modal + Form`，不要使用 `ProTable`。
- 使用生成的 `@/framework/http` API；禁止手改 `generated_api.ts`。
- 使用 `async/await`，不要使用 `try/catch` 或 `.then()`；请求失败可用 `await promise.catch(() => undefined)` 处理。
- 查询条件和分页放在一个 state；新增、编辑共用一个 Modal 和保存方法。
- 编辑优先复用行数据；只有列表字段不够时才请求详情。
- 删除使用 `Popconfirm`，成功后刷新；删除分页最后一条时回到上一页。
- 只保留必要字段、状态和 loading；不要提前拆组件或增加通用层。

## 验证

执行前端类型检查和构建，并运行 `git diff --check`。若失败，报告实际错误，不修改无关配置。
