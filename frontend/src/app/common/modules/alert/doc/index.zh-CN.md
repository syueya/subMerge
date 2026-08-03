---
category: Components
subtitle: 警告提示
type: 反馈
title: Alert
cover: 'https://gw.alipayobjects.com/zos/alicdn/8emPa3fjl/Alert.svg'
description: 警告提示，展现需要关注的信息。
---


## 何时使用

- 当某个页面需要向用户显示警告的信息时。
- 非浮层的静态展现形式，始终展现，不会自动消失，用户可以点击关闭。


## API

### cm-alert

| 参数              | 说明                                                      | 类型                                          | 默认值   | 全局配置 |
| ----------------- | --------------------------------------------------------- | --------------------------------------------- | -------- | -------- |
| `[banner]`      | 是否用作顶部公告                                          | `boolean`                                     | `false`  |
| `[action]`      | 自定义操作项                                              | `string \| TemplateRef<void>`                 | -        |
| `[showClose]`   | 默认不显示关闭按钮                                        | `boolean`                                     | -        | ✅        |
| `[closeText]`   | 自定义关闭按钮                                            | `string \| TemplateRef<void>`                 | -        |
| `[description]` | 警告提示的辅助性文字介绍                                  | `string \| TemplateRef<void>`                 | -        |
| `[message]`     | 警告提示内容                                              | `string \| TemplateRef<void>`                 | -        |
| `[iconName]`    | 是否显示辅助图标，`banner` 模式下默认值为 `true`        | `boolean`                                     | `false`  | ✅        |
| `[type]`        | 指定警告提示的样式，`banner` 模式下默认值为 `'warning'` | `'success' \| 'info' \| 'warning' \| 'error'` | `'info'` |
| `[custonIcon]`        | 自定义图标，showIcon 为 true 时有效                       | `string \| TemplateRef<void>`                 | -        |
| `(closeEvent)`     | 关闭时触发的回调函数                                      | `EventEmitter<void>`                          | -        |
