import { BooleanInput, coerceBooleanProperty } from '@angular/cdk/coercion';
import { Component, EventEmitter, Input, Output } from '@angular/core';

/**
 * 可折叠面板：基于 mat-expansion-panel，统一项目内折叠卡片样式。
 *
 * 基础：
 * <cm-collapse-panel title="过滤规则" description="可选">
 *   ...内容
 * </cm-collapse-panel>
 *
 * 自定义标题 / 右侧操作：
 * <cm-collapse-panel [(expanded)]="open">
 *   <span cmCollapseTitle>标题</span>
 *   <button cmCollapseAction mat-stroked-button>操作</button>
 *   ...内容
 * </cm-collapse-panel>
 */
@Component({
  selector: 'cm-collapse-panel',
  templateUrl: './collapse-panel.component.html',
  styleUrl: './collapse-panel.component.scss',
  standalone: false,
  host: { class: 'd-block' }
})
export class CmCollapsePanelComponent {
  /** 标题文案（也可用 cmCollapseTitle 投影） */
  @Input() title = '';
  /** 标题旁次要说明（也可用 cmCollapseDescription 投影） */
  @Input() description = '';
  /** 是否展开 */
  @Input()
  get expanded(): boolean {
    return this._expanded;
  }
  set expanded(value: BooleanInput) {
    this._expanded = coerceBooleanProperty(value);
  }
  private _expanded = true;

  /** 是否禁用折叠 */
  @Input()
  get disabled(): boolean {
    return this._disabled;
  }
  set disabled(value: BooleanInput) {
    this._disabled = coerceBooleanProperty(value);
  }
  private _disabled = false;

  /**
   * 隐藏左侧 ▶/▼ 指示。
   * Material 默认右侧箭头始终关闭，改用项目统一的前置符号。
   */
  @Input()
  get hideToggle(): boolean {
    return this._hideToggle;
  }
  set hideToggle(value: BooleanInput) {
    this._hideToggle = coerceBooleanProperty(value);
  }
  private _hideToggle = false;

  @Output() readonly expandedChange = new EventEmitter<boolean>();

  onExpandedChange(open: boolean): void {
    this._expanded = open;
    this.expandedChange.emit(open);
  }
}
