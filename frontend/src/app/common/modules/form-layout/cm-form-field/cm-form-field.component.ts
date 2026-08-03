import { Component, Input, TemplateRef } from '@angular/core';

/**
 * 表单字段布局：标签在上，下方为控件。
 * 默认 margin-bottom: 12px；需要贴底时加 m-b-0。
 *
 * 基础：
 * <cm-form-field class="col-24" [help]="'说明'">
 *   <mat-label>名称</mat-label>
 *   <mat-form-field>...</mat-form-field>
 * </cm-form-field>
 *
 * 输入框 + 右侧操作按钮同一行（垂直居中）：
 * <cm-form-field>
 *   <mat-label>域名/IP</mat-label>
 *   <mat-form-field class="w-full">...</mat-form-field>
 *   <button formFieldAction mat-flat-button color="primary">测试</button>
 * </cm-form-field>
 */
@Component({
  selector: 'cm-form-field',
  templateUrl: './cm-form-field.component.html',
  styleUrls: ['./cm-form-field.component.scss'],
  standalone: false,
  // 宿主可吃 col-* / m-b-0 等工具类
  host: { class: 'd-block' }
})
export class CmFormFieldComponent {
  /** 帮助信息样式 class（预留） */
  @Input() helpClass: string | null = null;

  /** 帮助信息（问号 tooltip） */
  @Input() help: string | null = null;

  /** 标签行右侧额外按钮模板（旧用法，仍支持） */
  @Input()
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  buttonsTemplate!: TemplateRef<any>;

  constructor() {}
}
