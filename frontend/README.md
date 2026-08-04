# SubMerge Frontend

基于 [angular-template](../../angular-template)（Angular 22 + Material + SCSS）的 SubMerge 管理面板。

## 开发

```bash
npm ci
npm start                 # http://127.0.0.1:4200 ，proxy → backend :8080
```

另开终端启动后端：

```bash
cd ../backend && go run .
```

首次打开网页创建管理员。探活：`GET /api/health`。

## 构建

```bash
npm ci && npm run build
# 产物：dist/submerge/browser
# 后端 StaticDir / Docker 依赖此路径
```

## 主流程

登录 → 配置订阅源 → 拉取合并 → 改策略组 → 改分流规则 → 发布 → 创建分享 token → 客户端 `/subscribe/{token}`

## 目录

```text
src/app
├── common           模板公共能力（含 net/ApiService、error 页、util）
├── data-struct      枚举 / 接口类型（@data-struct）
├── layouts          Full(/main) · Blank(/auth)
└── pages            业务模块（懒加载，服务跟模块）
    ├── dashboard
    ├── sources/services
    ├── groups
    ├── rules/services
    ├── releases/services
    ├── tokens/services
    ├── geo/services
    └── account-setting
```

## 页面约定

与 [angular-template](../../angular-template) / 用户管理页一致：

1. **ts 与 html 同目录**：`pages/<module>/<feature>/<feature>.component.ts` + `.html`
2. **样式复用公共能力**：`cm-page-toolbar`、`cm-responsive-table-list`、`cm-form-field`、`cm-dialog-header`、Material 按钮/表格 + `assets/scss` 工具类
3. **弹窗**：`CmDialogOpenService`（不要手写 `dialog-overlay`）
4. **默认不加组件 scss**；仅 utilities 无法覆盖时才写
5. **领域服务**放在对应 `pages/<module>/services/`；**类型/枚举**放 `app/data-struct`

## 样式

- 全局入口 `src/assets/scss/style.scss`（顺序勿乱）  
- 目录：根下按职责平铺（`_variables` / `_theme` / `_buttons` / `_material` / `_layout` / `_base` / `_grid` / `_custom` / `_modules`）；原子工具类在 `utilities/`（spacing / borders / flex / text / colors）  
- 主题：`_theme.scss`（`html.*_theme` 色板 + `light-theme` / `dark-theme`）；语义 token 在 `_variables`；页面色用 `var(--mat-sys-*)` 或 `text-primary` / `bg-light-primary`  
- 断点唯一来源 `_grid.scss` 的 `$breakpoints`：`sm 600` / `md 960` / `lg 1280`；全局 `@media` 与 `col-*` 都在该文件，用 `bp-gt` / `bp-lt`  

- 工具类：`d-flex` `gap-*` `m-*` `p-*` `f-s-*` `w-full` `icon-*` `opacity-*` `row` `col-24` `col-md-*` `cardWithShadow`  
- 栅格用 `_grid.scss` 24 栏（优先 `col-12` / `col-md-12`，少用 `col-sm-*`）；页面样式写 component scss  
- 表单字段默认 `appearance: outline` + `subscriptSizing: dynamic`（无 error/hint 时不占底部空行）
