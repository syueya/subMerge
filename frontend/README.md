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
