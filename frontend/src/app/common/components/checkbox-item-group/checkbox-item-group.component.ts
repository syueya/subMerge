/* eslint-disable @typescript-eslint/no-explicit-any */
import { Component, Input, TemplateRef, inject } from '@angular/core';
import { FormBuilder } from '@angular/forms';
import { MatFormFieldControl } from '@angular/material/form-field';
import { CmMatContorlParentComponent } from '@common/parents/mat-contorl-parent/mat-contorl-parent.component';
import { isArray } from '@common/util';

@Component({
  selector: 'cm-checkbox-item-group',
  templateUrl: './checkbox-item-group.component.html',
  providers: [{ provide: MatFormFieldControl, useExisting: CmCheckboxItemGroupComponent }],
  host: {
    '[class.floating]': 'shouldLabelFloat',
    '[class.mat-form-field-invalid]': 'invalid',
    '[id]': 'id',
    '[attr.aria-describedby]': 'describedBy'
  },
  standalone: false
})
export class CmCheckboxItemGroupComponent extends CmMatContorlParentComponent {
  private fb = inject(FormBuilder);

  // id分隔符
  @Input()
  separator = ',';

  // 默认显示的字符串
  @Input()
  renderProp = 'text';
  // 用来作为option value的对象属性
  @Input()
  valueProp = 'value';

  @Input('data')
  checkboxList: any[] = [];

  /**
   * 返回数据格式化
   * @type {'string' | 'array'}
   */
  @Input()
  valueFormat: 'string' | 'array' = 'array';

  // 全选/全不选
  @Input()
  selectAll = true;

  @Input()
  disabledIds: any[] = [];

  //  自动提示渲染内容，
  @Input()
  optionRender!: TemplateRef<any>;

  @Input()
  fxLayout: 'row' | 'row wrap' | 'column' = 'column';
  constructor() {
    super();
    this.id = `cm-checkbox-item-group-${CmCheckboxItemGroupComponent.nextId++}`;
  }

  override afterSetValue(value: any) {
    if (!isArray(value)) {
      value = value
        ? (value as string)
            .split(this.separator)
            .filter(v => !!v)
            .map(value => parseInt(value, 10))
        : null;
    } else {
      value = value.filter((v: any) => !!v);
    }
    this.itemCtrl.setValue(value);
  }

  override afterFormValueChange(value: any) {
    let result = value;
    if (this.valueFormat === 'string') {
      result = (value || []).join(this.separator);
    }
    this.onChange(result);
  }

  get someComplete(): boolean {
    const value = this.itemCtrl.value || [];
    const selectLen = (this.checkboxList || []).filter(item => value.includes(item[this.valueProp])).length;
    return selectLen > 0 && selectLen < (this.checkboxList || []).length;
  }

  get allComplete(): boolean {
    const value = this.itemCtrl.value || [];
    const selectLen = (this.checkboxList || []).filter(item => value.includes(item[this.valueProp])).length;
    return selectLen > 0 && selectLen === (this.checkboxList || []).length;
  }

  /**
   * 全选全不选
   * @param completed
   */
  checkAllChange(completed: boolean) {
    if (completed) {
      this.itemCtrl.setValue((this.checkboxList || []).map(v => v[this.valueProp]));
    } else {
      this.itemCtrl.reset([]);
    }
  }

  /**
   * 选中某一个
   * @param completed
   * @param item
   */
  checkChange(completed: boolean, item: any) {
    const value = this.itemCtrl.value || [];
    const idx = value.indexOf(item[this.valueProp]);
    if (idx > -1) {
      value.splice(idx, 1);
    } else {
      value.push(item[this.valueProp]);
    }
    this.itemCtrl.setValue(value);
  }
}
