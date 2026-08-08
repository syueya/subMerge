import { Component, EventEmitter, Input, Output } from '@angular/core';

/**
 * 对话框通用页脚：分隔线 + 右侧按钮区。
 * 取消按钮由组件统一渲染（文本可配），确定/主操作按钮通过 <ng-content> 投影进去。
 * 默认取消按钮点击触发 mat-dialog-close；若关闭时需要携带返回值或自定义逻辑，
 * 传 customCancel=true 并通过 (cancelClick) 自行处理。
 */
@Component({
	selector: 'cm-dialog-footer',
	templateUrl: './dialog-footer.component.html',
	standalone: false,
})
export class CmDialogFooterComponent {
	/** 取消按钮文案 */
	@Input() cancelText = '取消';
	/** true 时取消按钮交由 (cancel) 事件处理，不默认触发 mat-dialog-close */
	@Input() customCancel = false;
	/** customCancel=true 时的取消点击回调 */
	@Output() cancel = new EventEmitter<void>();
}