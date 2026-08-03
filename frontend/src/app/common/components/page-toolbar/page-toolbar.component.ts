import { Component, Input } from '@angular/core';

/**
 * 列表页统一顶栏：左侧页面标题，右侧操作区（刷新/添加/导入等）
 * 用法：
 * <cm-page-toolbar title="化合物">
 *   <button mat-flat-button ...>刷新</button>
 * </cm-page-toolbar>
 *
 * 标题旁额外内容（如规则切换）：
 * <cm-page-toolbar title="规则配置">
 *   <div pageToolbarExtra>...</div>
 *   <button>...</button>
 * </cm-page-toolbar>
 */
@Component({
  selector: 'cm-page-toolbar',
  standalone: false,
  templateUrl: './page-toolbar.component.html',
  styleUrl: './page-toolbar.component.scss'
})
export class PageToolbarComponent {
  /** 页面标题（通常与菜单名一致） */
  @Input({ required: true }) title = '';
}
