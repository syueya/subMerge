# 前端开发约束

修改 `fronted/` 内的界面、样式、组件或交互前，必须先完整阅读 [`STYLE_GUIDE.md`](STYLE_GUIDE.md)，并按其中规范实现。

- 优先复用现有共享组件与语义样式 token，不做无关的视觉重构。
- 完成后至少运行 `npm run typecheck`；涉及交互时运行相关 `npm test`。
