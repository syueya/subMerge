# React + TypeScript + Vite

此模板提供了一套最小配置，用于在 Vite 中运行支持热模块替换（HMR）的 React，并包含部分 Oxlint 规则。

目前有两个官方插件可用：

- [@vitejs/plugin-react](https://github.com/vitejs/vite-plugin-react/blob/main/packages/plugin-react) 使用 [Oxc](https://oxc.rs)
- [@vitejs/plugin-react-swc](https://github.com/vitejs/vite-plugin-react/blob/main/packages/plugin-react-swc) 使用 [SWC](https://swc.rs/)

## React 编译器

考虑到 React 编译器对开发和构建性能的影响，此模板默认未启用该功能。如需添加，请参阅[安装文档](https://react.dev/learn/react-compiler/installation)。

## 扩展 Oxlint 配置

开发生产应用时，建议安装 `oxlint-tsgolint` 并编辑 `.oxlintrc.json`，以启用类型感知的代码检查规则：

```json
{
  "$schema": "./node_modules/oxlint/configuration_schema.json",
  "plugins": ["react", "typescript", "oxc"],
  "options": {
    "typeAware": true
  },
  "rules": {
    "react/rules-of-hooks": "error",
    "react/only-export-components": ["warn", { "allowConstantExport": true }]
  }
}
```

完整的规则和分类列表请参阅 [Oxlint 规则文档](https://oxc.rs/docs/guide/usage/linter/rules)。
