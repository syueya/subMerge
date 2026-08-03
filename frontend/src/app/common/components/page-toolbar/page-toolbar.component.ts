import { Component, Input } from '@angular/core';

/**
 * 列表页统一顶栏：左列标题+说明，右列操作按钮。
 *
 * 布局：
 *   [标题] [pageToolbarExtra?]     [操作按钮]
 *   [说明：description / pageToolbarDesc]
 *
 * 纯文本说明：
 *   <cm-page-toolbar title="概览" description="…">
 *     <button mat-flat-button>刷新</button>
 *   </cm-page-toolbar>
 *
 * 说明含链接（不能塞进 description 字符串）：
 *   <cm-page-toolbar title="策略组">
 *     <p pageToolbarDesc class="text-muted f-s-12 m-t-8 m-b-0">
 *       … <a routerLink="/main/rules">分流规则</a>
 *     </p>
 *     <button mat-flat-button>新增</button>
 *   </cm-page-toolbar>
 */
@Component({
  selector: 'cm-page-toolbar',
  standalone: false,
  templateUrl: './page-toolbar.component.html'
})
export class PageToolbarComponent {
  /** 页面标题（通常与菜单名一致） */
  @Input({ required: true }) title = '';

  /**
   * 纯文本说明，显示在标题下方。
   * 需要链接/内联 HTML 时改用 [pageToolbarDesc] 投影，不要两者同时用。
   */
  @Input() description = '';
}
