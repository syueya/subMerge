import { Component, ViewChild } from '@angular/core';
import { FormGroup, FormGroupDirective } from '@angular/forms';
import { isString } from '@common/util';

import { CmParentComponent } from '../parent/parent.component';

@Component({
    selector: 'cm-parent-form',
    templateUrl: './parent-form.component.html',
    standalone: false
})
export class CmParentFormComponent extends CmParentComponent {
  editForm!: FormGroup;

  @ViewChild('formDirective', { static: false })
  formDirective!: FormGroupDirective;

  /**
   * 表单是否正在提交中(防止重复提交)
   */
  isSubmitting = false;

  constructor() {
    super();
  }

  /**
   * 是否可以提交
   *
   * @readonly
   * @type {boolean}
   * @memberof CmParentFormComponent
   */
  public get canSubmit(): boolean {
    return !this.isSubmitting && this.editForm.dirty && this.editForm.valid;
  }

  /**
   * 当前错误状态
   *
   * @param {string} prop
   * @returns {string}
   * @memberof CmParentFormComponent
   */
  public currentErrorStatus(prop: string, group: FormGroup = this.editForm): string {
    this.validatProp(prop);
    if (group !== null) {
      const control = group.controls[prop];
      if (control !== null && control.invalid) {
        return Object.keys(control.errors!)[0];
      }
    }
    // if (!!group && group.controls[prop].invalid) {
    //   return Object.keys(group.controls[prop].errors)[0];
    // }
    return '';
  }
  /**
   * 发生最大错误信息时，提示信息
   *
   * @param {string} prop
   * @returns {Observable<string>}
   * @memberof CmParentFormComponent
   */
  getMaxErrorInfo(prop: string, group: FormGroup = this.editForm): number {
    const { max } = group.controls[prop].errors!['max'];
    return max;
  }

  /**
   * 发生错误信息时，信息提示
   *
   * @param {string} prop
   * @returns {Observable<string>}
   * @memberof ParentFormComponent
   */
  getMinErrorInfo(prop: string, group: FormGroup = this.editForm): number {
    const { min } = group.controls[prop].errors!['min'];
    return min;
  }

  protected validatProp(prop: string) {
    if (!isString(prop)) {
      throw 'error parameter, need string';
    }
  }

  /**
   * 是否存在必填错误信息
   *
   * @param {string} prop
   * @returns {boolean}
   * @memberof CmParentFormComponent
   */
  hasErrorRequired(prop: string, group: FormGroup = this.editForm): boolean {
    this.validatProp(prop);

    const formContrl = group.controls[prop];
    if (formContrl) {
      return formContrl.hasError('required');
    }
    return false;
  }

  /**
   * 是否存在最大长度
   *
   * @param {string} prop
   * @returns {boolean}
   * @memberof CmParentFormComponent
   */
  hasErrorMaxlength(prop: string, group: FormGroup = this.editForm): boolean {
    this.validatProp(prop);
    const formContrl = group.controls[prop];
    if (formContrl) {
      return formContrl.hasError('maxlength');
    }
    return false;
  }

  /**
   * 得到最大长度
   *
   * @param {string} prop
   * @returns {number}
   * @memberof CmParentFormComponent
   */
  getMaxlength(prop: string, group: FormGroup = this.editForm): number {
    this.validatProp(prop);
    if (this.hasErrorMaxlength(prop, group)) {
      const formContrl = group.controls[prop];
      return formContrl.errors!['maxlength']['requiredLength'];
    }
    return 0;
  }

  /**
   * 是否存在电话格式错误
   *
   * @param {string} prop
   * @returns {boolean}
   * @memberof CmParentFormComponent
   */
  hasErrorTel(prop: string, group: FormGroup = this.editForm): boolean {
    this.validatProp(prop);
    const formContrl = group.controls[prop];
    if (formContrl) {
      return formContrl.hasError('tel');
    }
    return false;
  }

  /**
   * 是否存在邮箱错误
   *
   * @param {string} prop
   * @returns {boolean}
   * @memberof CmParentFormComponent
   */
  hasErrorEmail(prop: string, group: FormGroup = this.editForm): boolean {
    this.validatProp(prop);
    const formContrl = group.controls[prop];
    if (formContrl) {
      return formContrl.hasError('email');
    }
    return false;
  }

  hadError(prop: string, group: FormGroup = this.editForm): boolean {
    this.validatProp(prop);
    const formContrl = group.controls[prop];
    return !!formContrl.errors;
  }
}
