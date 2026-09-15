# 后台 UI 规范

负责后台页面结构、视觉、布局、间距及列表页、详情页、编辑页的 UI 交互一致性。API、查询分页、增删改和状态管理见 [crud.md](crud.md)，权限实现见 [permission.md](permission.md)。

## 设计参考优先级

1. 用户提供的截图 / 设计稿 / 指定参考页面。
2. 本文 `admin-ui.md`。
3. 当前项目成熟页面。
4. Ant Design 默认样式。

页面视觉与布局统一以本文为准，并服从用户明确的视觉参考；历史页面只复用适用的业务组织和交互，不因旧页面结构把用户的参考改回原样。

## 已定稿视觉基线

商品列表（`modules/product/product/index.tsx`）是当前 List Page 的视觉母版，页面结构、标题比例、筛选区、TableToolbar、表格与分页均以它为准。订单、会员、库存等后续列表页对齐它，不重新设计独立视觉体系。

实现真源，公共视觉只维护一份：

- 统一 Theme：`shared/admin-ui/admin-theme.tsx` 的 `adminThemeConfig(isDark)` 是全站唯一一套 Theme，由 `framework/layout/layout.tsx` 的 `ConfigProvider` 直接使用（含 `algorithm` 与明暗取值），跟随 Layout 的主题开关切换；业务页面不包主题容器，也不创建 Product / Order / Member 各自独立的 Theme。
- 公共 List 样式：`shared/admin-ui/list-page.less`，类名 `admin-list-page`、`admin-list-header`、`admin-list-filters`、`admin-list-toolbar`、`admin-list-table`、`admin-list-actions`、`admin-list-pagination`；主题色变量由 `shared/admin-ui/list-page.ts` 的 `adminListVars(token)` 注入，页面不再自己映射颜色。
- 页面标题：`shared/admin-ui/page-header.tsx` 已内置定稿标题比例（约 19px、字重 650、描述弱化），页面不再各自覆写 `.xxx-header h1` 字号。
- 主内容容器：`shared/admin-ui/content-card.tsx` 默认无边框，与定稿页面一致，靠背景明度与留白区分层级。

规则：

- 新列表页优先复用上述 Theme 与 `admin-list-*` 类名，只补业务特例样式。
- 业务特例（特殊搜索框宽度、图片尺寸、字段布局、模块专属响应式）留在各 module 的 less 中，不提前抽到 shared；至少两个页面确实需要时再抽象。
- 不创建 `AdminButton`、`AdminInput`、`AdminTable`、`AdminSwitch` 等 Ant Design 包装组件，也不为“设计系统完整”批量新建 token、spacing、typography、variants 或 hooks。

## 视觉方向

- 目标是 Modern SaaS / Backoffice / Operations Dashboard：简洁、精致、克制，信息密度高但不拥挤，避免传统 Admin Template 的卡片拼装感。
- 用标题、摘要、正文和辅助信息建立层级；表格突出业务主体，编号、时间等次要信息弱化，不能所有字段同等抢眼。
- 页面背景、主内容区域和表格应有清晰层级：用背景明度、表头和区域留白区分，减少大面积白色 Card 层层堆叠，弱化边框感。
- 相关内容紧凑排列，不同业务分区适当留白；保持对齐和同级间距一致，不靠增加 Card 数量或大块空白制造层级。
- 主色只用于关键操作；状态色采用柔和表达并保留文字与可读对比度，辅助信息使用中性色。
- 不使用大渐变、大阴影、玻璃拟态、大圆角等营销页风格；最终效果不能只是“默认 Ant Design Demo”。

## 技术分工

| 工具        | 职责                                                                         |
| ----------- | ---------------------------------------------------------------------------- |
| Ant Design  | 表格、表单、按钮、弹窗等业务组件；普通 CRUD 使用 Table，不使用 ProTable      |
| Tailwind    | flex / grid / gap / padding / margin / 宽高 / 响应式等简单布局，优先使用     |
| Less/CSS    | 复杂或重复样式、组件级样式                                                   |
| Theme Token | 颜色、背景、边框、状态色，通过 `theme.useToken()` 等现有能力消费，避免硬编码 |

不要为了 Tailwind 强行覆盖 Ant Design 内部样式。优先复用 `src/shared/admin-ui/` 中适用的组件，但不受历史包装结构限制；至少两个页面确实需要时再抽取共享能力，不包装 `AdminButton` 等已有基础组件。

## List Page：一个完整的业务工作区

默认结构：

```text
PageHeader

ContentCard
├── FilterArea
├── TableToolbar
└── Table
```

这些名称表示页面区域，不要求创建同名组件或在每层嵌套 Card。

- PageHeader 放标题、必要说明和新增等页面主操作；避免在多个位置重复主操作。
- 筛选较少时与表格共用一个主内容区域，避免“搜索 Card + 表格 Card”两张白色卡片上下堆叠；只有筛选非常复杂、需要独立分组或展开空间时才使用 FilterCard。
- FilterArea 将条件与查询、重置放在一起，允许换行；TableToolbar 可放列表标题、总数量、刷新、列设置，密度设置按需提供，不为凑齐结构增加控件。
- 表格保留高频核心字段，低频信息放详情；操作列放右侧，低频操作收入“更多”，删除排在最后并使用危险操作语义。
- 表格与筛选共享对齐和间距节奏；窄屏优先保证业务主体和操作可读，宽表按需横向滚动。

## Detail Page：概要优先，按业务分区

- 使用 PageHeader → Summary → DetailSection；状态、关键金额、时间等放顶部概要，明细与操作记录按业务关系组织，不要求每段都包 Card。
- 查看内容使用 Descriptions、Table、Timeline 等，不用 disabled Form 代替详情；字段空值统一为 `—`。

## Edit Page：分组清楚，保存易达

- 使用 PageHeader → FormSection → ActionBar；简单表单可用弹窗，复杂编辑页按业务分组或使用 Tabs，避免简单表单过度分栏。
- 普通输入区限制阅读宽度，SKU、富文本、图片等按内容扩展；长页面提供固定或吸底的取消、保存操作栏，避免遮挡内容。
- 复杂表单考虑未保存修改提醒、错误定位和错误 Tab 切换，减少用户寻找问题的成本。

## 交互与适配

- 同类页面保持操作命名、位置与反馈一致；Loading / Empty / Error 使用明确的区域反馈，失败提供重试或返回入口，不能只弹提示后留下空白。
- 检查明暗主题及 1920、1440、1366px 宽度；尤其在 1366 × 768 下，筛选、表格和操作栏仍须清晰可用。

## UI 美化验收

任务明确为“美化 / UI 优化 / 页面视觉改造”时，组件抽取、Tailwind 替换、inline style 清理或代码整理不能单独算完成，必须有明显视觉变化。

至少实现以下 2～3 项改善，并说明前后差异：

- 页面信息层级更明显。
- Card 层级更合理。
- 留白和密度更舒服。
- 表格主次信息更清晰。
- 操作区更简洁。
- 颜色使用更克制。
- 筛选区和数据区关系更合理。
- 1366px 下依然清晰好用。

使用同视口、同主题和可比数据的修改前后截图验收；无法截图时明确标记视觉验证未完成。视觉调整保持现有 API、路由、权限与业务行为。

> 如果最终截图与修改前视觉几乎一致，则不能认为“UI 美化”已经完成。
