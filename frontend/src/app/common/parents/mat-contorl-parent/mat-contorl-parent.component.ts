/* eslint-disable @typescript-eslint/no-explicit-any */
import { FocusMonitor } from '@angular/cdk/a11y';
import { coerceBooleanProperty } from '@angular/cdk/coercion';
import { AfterViewInit, Component, ElementRef, Input, OnDestroy, OnInit, inject } from '@angular/core';
import { ControlValueAccessor, NgControl, FormControl } from '@angular/forms';
import { MatFormFieldControl } from '@angular/material/form-field';
import { isNullOrUndefined } from '@common/util';
import { Subject, Subscription } from 'rxjs';

import { CmParentComponent } from '../parent/parent.component';

@Component({
    selector: 'cm-mat-contorl-parent',
    templateUrl: './mat-contorl-parent.component.html',
    standalone: false
})
export class CmMatContorlParentComponent extends CmParentComponent implements OnInit, ControlValueAccessor, MatFormFieldControl<any>, OnDestroy, AfterViewInit {
  protected fm = inject(FocusMonitor);
  protected elRef = inject<ElementRef<HTMLElement>>(ElementRef);
  ngControl = inject(NgControl, { optional: true, self: true });

  static nextId = 0;

  itemCtrl: FormControl;

  // 是否为select control
  isSelectControl = false;

  /**
   * 上一个值
   */
  prevValue: any;

  // 状态检测
  stateChanges = new Subject<void>();
  focused = false;
  // 错误状态
  get errorState(): boolean {
    return !!this.ngControl && !!this.ngControl.touched && !!this.ngControl.invalid;
  }

  id = `parent-control-${CmMatContorlParentComponent.nextId++}`;
  describedBy = '';

  onChange = (_: any) => {};
  onTouch = () => {};

  get empty() {
    return isNullOrUndefined(this.value) || this.value === '';
  }
  // 一直浮动
  get shouldLabelFloat(): boolean {
    return !this.empty || this.focused;
  }

  public get invalid() {
    return !!this.ngControl && this.ngControl.invalid;
  }

  private _placeholder = '';
  @Input()
  get placeholder(): string {
    return this._placeholder;
  }
  set placeholder(value: string) {
    this._placeholder = value;
    this.onTouch();
    this.stateChanges.next();
  }

  private _required = false;
  @Input()
  get required(): boolean {
    return this._required;
  }
  set required(value: boolean) {
    this._required = coerceBooleanProperty(value);
    this.stateChanges.next();
  }

  @Input()
  get disabled(): boolean {
    return this._disabled;
  }
  set disabled(value: boolean) {
    this._disabled = coerceBooleanProperty(value);
    this.stateChanges.next();
    if (value) {
      this.itemCtrl.disable();
    } else {
      this.itemCtrl.enable();
    }
  }
  private _disabled = false;

  @Input()
  get value(): any {
    //  如果每个值都存在，则有效
    return this.itemCtrl.value;
  }
  set value(v: any) {
    this.afterSetValue(v);
    this.stateChanges.next();
  }
  private valueChangeSubscriptions: Subscription[];
  constructor() {
    super();
    const fm = this.fm;
    const elRef = this.elRef;


    this.itemCtrl = new FormControl(null);

    this.valueChangeSubscriptions = [];

    // 初始化，并订阅事件
    // 见监控fm的点击事件
    fm.monitor(elRef, true).subscribe(origin => {
      if (this.focused && !origin) {
        this.onTouch();
      }
      this.focused = !!origin;
      this.stateChanges.next();
    });
    if (this.ngControl !== null) {
      this.ngControl.valueAccessor = this;
    }

    const sub = this.itemCtrl.valueChanges.subscribe(_ => {
      this._handleInput();
    });
    this.valueChangeSubscriptions.push(sub);
  }

  override ngOnDestroy() {
    super.ngOnDestroy();
    if (this.valueChangeSubscriptions) {
      this.valueChangeSubscriptions.forEach(s => s.unsubscribe());
    }
    this.stateChanges.complete();
    this.fm.stopMonitoring(this.elRef);
  }

  setDescribedByIds(ids: string[]) {
    this.describedBy = ids.join(' ');
  }

  /**
   * 点击后,自动展开第一个select
   *
   * @param {MouseEvent} event
   * @memberof SelectUser
   */
  onContainerClick(event: MouseEvent) {
    if ((event.target as Element).tagName.toLowerCase() !== 'mat-select') {
      // 用户点击后，
      this.onTouch();
    }
  }

  writeValue(v: any): void {
    this.value = v;
  }
  registerOnChange(fn: any): void {
    this.onChange = fn;
  }
  registerOnTouched(fn: any): void {
    this.onTouch = fn;
  }
  setDisabledState(isDisableed: boolean): void {
    this.disabled = isDisableed;
  }
  private _handleInput(): void {
    if (!this.ngControl || this.ngControl.touched || this.focused || this.isSelectControl) {
      // 那么判断用户是否选择了select，如果不再存在，则返回null
      const current = this.itemCtrl.value;
      const nextValue = current ? JSON.parse(JSON.stringify(current)) : current;
      if (!this.isEqual(this.prevValue, nextValue)) {
        this.afterFormValueChange(nextValue);
        this.prevValue = nextValue;
      }
    }
  }

  isEqual(val1: any, val2: any) {
    if (val1 && val2) {
      return JSON.stringify(val1) === JSON.stringify(val2);
    }
    return val1 === val2;
  }

  /**
   * set value后处理逻辑 由子级继承执行
   * @param value
   */
  afterSetValue(value: any) {}

  /**
   * 表单值改变 由子级继承执行
   * @param value
   */
  afterFormValueChange(value: any) {}
}
