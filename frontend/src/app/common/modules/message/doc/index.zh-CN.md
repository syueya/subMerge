---
category: Components
subtitle: 全局提示
type: 反馈
noinstant: true
title: Message
cover: https://gw.alipayobjects.com/zos/alicdn/hAkKTIW0K/Message.svg
---

全局展示操作反馈信息。

## 何时使用

- 可提供成功、警告和错误等反馈信息。
- 顶部居中显示并自动消失，是一种不打断用户操作的轻量级提示方式。

```ts
import { CmMessageModule } from 'ng-zorro-antd/message';
```

## API

### CmMessageService

组件提供了一些服务方法，使用方式和参数如下：

- `CmMessageService.success(content, [options])`
- `CmMessageService.error(content, [options])`
- `CmMessageService.info(content, [options])`
- `CmMessageService.warning(content, [options])`
- `CmMessageService.loading(content, [options])`

| 参数 | 说明 | 类型 | 默认值 |
| --- | --- | --- | --- |
| content | 提示内容 | `string \| TemplateRef<void>` | - |
| options | 支持设置针对当前提示框的参数，见下方表格 | `object` | - |

`options` 支持设置的参数如下：

| 参数 | 说明 | 类型 |
| --- | --- | --- |
| cmDuration | 持续时间(毫秒)，当设置为0时不消失 | `number` |
| cmPauseOnHover | 鼠标移上时禁止自动移除 | `boolean` |
| cmAnimate | 开关动画效果 | `boolean` |

还提供了全局销毁方法：

- `CmMessageService.remove(id)` // 移除特定id的消息，当id为空时，移除所有消息（该消息id通过上述方法返回值中得到）

### 配置

| 参数 | 说明 | 类型 | 默认值 |
| --- | --- | --- | --- |
| cmDuration | 持续时间(毫秒)，当设置为 0 时不消失 | `number` | `3000` |
| cmMaxStack | 同一时间可展示的最大提示数量 | `number` | `7` |
| cmPauseOnHover | 鼠标移上时禁止自动移除 | `boolean` | `true` |
| cmAnimate | 开关动画效果 | `boolean` | `true` |
| cmTop | 消息距离顶部的位置 | `number \| string` | `24` |

### CmMessageRef

当你调用 `CmMessageService.success` 或其他方法时会返回该对象。

```ts
export interface CmMessageRef {
  messageId: string;
  onClose: Subject<false>; // 当 message 关闭时它会派发一个事件
}
```