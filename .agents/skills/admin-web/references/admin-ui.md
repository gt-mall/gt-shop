# 后台 UI 规范

适用于管理端列表页、详情页、编辑页及系统页面。

核心目标：

> 同类型页面保持统一，用户学会一个页面后，就能使用整个后台。

## 1. 技术使用原则

项目使用：

- React + TypeScript
- Ant Design / ProComponents
- Tailwind CSS
- Less / CSS
- Ant Design Theme Token

使用优先级：

```text
Ant Design：按钮、表格、表单、弹窗、Tag、Tabs 等组件

Tailwind：flex、grid、gap、padding、margin、宽高、定位、响应式等简单样式

Less/CSS：复杂布局、重复样式、组件级样式

Theme Token：颜色、背景、边框、状态色
```

简单布局优先 Tailwind，例如：

```tsx
<div className="flex items-center justify-between gap-4">
```

不要为了简单布局大量写：

```tsx
style={{
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'space-between',
}}
```

禁止为了使用 Tailwind 而重写 Ant Design 自带样式。

颜色优先：

```ts
theme.useToken();
```

不要随意新增：

```text
#888
#333
#f5222d
#4338CA
```

等硬编码颜色。

---

## 2. 页面统一分类

后台页面主要分为：

```text
List Page    列表页
Detail Page  详情页
Edit Page    编辑页
```

开发新页面前，先确定页面类型，并参考已有同类型标准页面。

参考：

```text
列表页：商品列表
详情页：订单详情
编辑页：商品编辑
授权页：角色授权
```

不要为每个模块重新设计页面结构。

---

## 3. 列表页

统一结构：

```text
PageHeader

FilterCard

TableCard
```

PageHeader：

```text
商品管理
管理商城商品及上下架状态                 + 新建商品
```

新增等主要操作放右上角。

筛选区统一：

```text
关键词 / 分类 / 状态 / 日期

查询
重置
```

规则：

- 查询按钮使用 `primary`
- 重置放在查询后面
- 筛选项允许换行
- 不同列表页保持相似高度和间距

Table 默认：

```text
pageSize = 20

10 / 20 / 50 / 100
```

提供：

- loading
- empty
- showTotal
- showSizeChanger
- 必要时横向滚动

操作列统一放右侧：

```text
查看 | 编辑 | 业务操作 | 删除
```

删除：

- 永远最后
- danger
- 必须二次确认

操作超过 3~4 个时，低频操作放入“更多”。

列表只展示高频核心字段，低频信息放详情页。

---

## 4. 详情页

详情页用于查看，不要直接使用 `disabled Form` 代替详情展示。

统一结构：

```text
PageHeader

Summary

DetailSection

DetailSection

操作记录 / Timeline
```

例如订单详情：

```text
订单概要
商品信息
支付信息
收货信息
物流信息
售后信息
操作记录
```

详情展示优先使用：

```text
Descriptions
Card
Table
Timeline
Tag
```

字段空值统一使用：

```text
—
```

核心状态、金额、时间等放在顶部 Summary。

---

## 5. 编辑页

简单编辑页：

```text
PageHeader

FormSection
FormSection

ActionBar
```

复杂编辑页可以使用 Tabs，例如商品：

```text
基本信息
SKU规格
商品参数
商品详情
```

不要简单表单也使用大量 Tabs。

长编辑页面必须提供固定或吸底操作栏：

```text
取消    保存
```

保存使用 `primary`。

复杂页面建议支持：

- 未保存修改提醒
- 表单错误自动定位
- Tab 内错误时自动跳到对应 Tab

表单不要横跨整个大屏。

普通输入区域控制在合理宽度，复杂 SKU、富文本、图片组件可以使用更宽区域。

---

## 6. 页面状态

所有主要页面必须考虑：

```text
Loading
Empty
Error
No Permission
```

加载失败不能只：

```ts
message.error(...)
```

然后留下空白页面。

重要页面应提供：

```text
加载失败
重新加载
返回
```

等恢复方式。

---

## 7. 权限

UI 根据权限隐藏不可执行操作。

例如：

```text
无 create → 不显示新增

无 update → 不显示编辑

无 delete → 不显示删除

无 ship → 不显示发货
```

不要显示按钮后让用户点击得到 403。

但前端权限只是 UI 控制，后端仍必须进行真实权限校验。

菜单、路由、快捷入口应尽量使用同一权限定义。

---

## 8. 统一视觉

后台风格保持：

```text
简洁
高信息密度
明确层级
少装饰
```

避免：

- 大量阴影
- 大圆角
- 渐变背景
- 花哨动画
- 不必要的大字号

简单布局优先 Tailwind。

复杂可复用样式放 Less/CSS。

颜色优先 Ant Design Token。

新增和修改页面时同时检查：

```text
Light Mode
Dark Mode
```

至少保证：

```text
1920 × 1080
1440 × 900
1366 × 768
```

下可以正常操作。

---

## 9. 公共组件

公共页面组件放：

```text
src/shared/admin-ui/
```

按实际需要逐步增加，例如：

```text
PageHeader
FilterCard
TableCard
DetailSection
FormSection
StickyActionBar
PageLoading
PageError
PageEmpty
```

不要一次性为了“架构完整”全部创建。

至少有两个页面真正需要相同能力时，再考虑抽公共组件。

也不要包装：

```text
AdminButton
AdminInput
AdminSelect
AdminCard
```

这类 Ant Design 已经提供的基础组件。

---

## 10. AI 修改页面时

创建或重构后台页面时必须检查：

```text
1. 属于 List / Detail / Edit 哪种页面
2. 是否有现成页面可以参考
3. 是否可以复用 shared/admin-ui
4. 页面结构是否与同类型页面一致
5. 权限是否正确
6. Loading / Empty / Error 是否完整
7. 是否存在不必要的 inline style
8. 简单布局是否适合使用 Tailwind
9. 是否新增了不必要的硬编码颜色
10. 是否破坏现有 API、路由、权限和业务逻辑
```

UI 重构原则上不要同时大规模修改业务逻辑。

---

## 最终原则

```text
一致性 > 个性化

操作效率 > 装饰效果

Ant Design 管组件

Tailwind 管简单布局

Less/CSS 管复杂样式

Theme Token 管颜色

复用 > 重复开发

简单实现 > 过度抽象
```
