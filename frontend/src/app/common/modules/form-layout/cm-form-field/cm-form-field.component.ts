import { Component, Input, TemplateRef } from '@angular/core';

@Component({
    selector: 'cm-form-field',
    templateUrl: './cm-form-field.component.html',
    styleUrls: ['./cm-form-field.component.scss'],
    standalone: false
})
export class CmFormFieldComponent {
  // 样式
  @Input()
  helpClass: string | null = null;

  // 帮助信息
  @Input()
  help: string | null = null;

  //按钮模板
  @Input()
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  buttonsTemplate!: TemplateRef<any>;

  constructor() {}

}
