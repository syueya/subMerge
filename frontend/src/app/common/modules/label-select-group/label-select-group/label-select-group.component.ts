/* eslint-disable @typescript-eslint/no-explicit-any */
import { FocusMonitor } from '@angular/cdk/a11y';
import { Component, Input, TemplateRef, ElementRef, Output, EventEmitter, inject } from '@angular/core';
import { NgControl, FormBuilder } from '@angular/forms';
import { MatChipListboxChange } from '@angular/material/chips';
import { MatFormFieldControl } from '@angular/material/form-field';
import { CmMatContorlParentComponent } from '@common/parents/mat-contorl-parent/mat-contorl-parent.component';

@Component({
    selector: 'cm-label-select-group',
    templateUrl: './label-select-group.component.html',
    styleUrls: ['./label-select-group.component.scss'],
    providers: [{ provide: MatFormFieldControl, useExisting: LabelSelectGroupComponent }],
    host: {
        '[class.floating]': 'shouldLabelFloat',
        '[class.mat-form-field-invalid]': 'invalid',
        '[id]': 'id',
        '[attr.aria-describedby]': 'describedBy'
    },
    standalone: false
})
export class LabelSelectGroupComponent extends CmMatContorlParentComponent {
  protected override fm: FocusMonitor;
  protected override elRef: ElementRef<HTMLElement>;
  override ngControl: NgControl;
  private fb = inject(FormBuilder);


  // 默认显示的字符串
  @Input()
  renderProp = 'text';
  // 用来作为option value的对象属性
  @Input()
  valueProp = 'value';

  @Input('data')
  dataList: any[] = [];


  @Input()
  disabledIds: any[] = [];

  //  自动提示渲染内容，
  @Input()
  optionRender: TemplateRef<any> | null = null;

  @Input()
  fxLayout: 'row' | 'row wrap' | 'column' = 'column';


  /**
   * 最多显示的标签数量，大于此数量将以下拉形式呈现
   */
  @Input()
  maxLabelNum = 20;

  /**
   * 用户选择change事件
   */
  @Output()
  readonly selectionChange = new EventEmitter<any>();

  constructor() {
    const fm = inject(FocusMonitor);
    const elRef = inject<ElementRef<HTMLElement>>(ElementRef);
    const ngControl = inject(NgControl, { optional: true, self: true });

    super(fm, elRef, ngControl);
    this.fm = fm;
    this.elRef = elRef;
    this.ngControl = ngControl;


    this.id = `cm-label-select-group-${LabelSelectGroupComponent.nextId++}`;
  }

  override afterSetValue(value: any) {
    this.itemCtrl.setValue(value);
  }

  override afterFormValueChange(value: any) {
    const result = value;
    this.onChange(result);
  }

   /**
   * 用户选择change
   * @param $event
   */
   userSelectionChange(value:any) {
    this.itemCtrl.setValue(value);
    this.stateChanges.next();
    this.selectionChange.emit(value);
  }
}
