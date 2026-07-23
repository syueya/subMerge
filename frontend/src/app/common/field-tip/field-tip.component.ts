import { Component, Input } from '@angular/core';
import { MatTooltipModule } from '@angular/material/tooltip';

/** 表单字段旁的 ? 提示；样式见 styles.css .field-tip / .field-label */
@Component({
	selector: 'app-field-tip',
	standalone: true,
	imports: [MatTooltipModule],
	templateUrl: './field-tip.component.html',
	styleUrl: './field-tip.component.scss',
})
export class FieldTipComponent {
	@Input({ required: true }) text = '';
}
