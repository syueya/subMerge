import { ChangeDetectionStrategy, Component, Input, ViewEncapsulation, inject } from '@angular/core';
import { AbstractControl, FormArrayName, FormGroupDirective, FormGroupName, NgControl, ValidationErrors } from '@angular/forms';
import { MatFormField } from '@angular/material/form-field';
import { Observable, of } from 'rxjs';

export enum ErrorsDefault {
  Required = 'required',
  Min = 'min',
  Max = 'max',
  MaxLength = 'maxlength',
  MinLength = 'minlength',
  Email = 'email',
  Duplicate = 'duplicate',
  InvalidChar = 'chars',
  ConfirmPasswordMismatch = 'confirmPasswordMismatch',
  userNameInvalid = 'userNameInvalid',
  CronMinIntervalInvalid = 'cronMinIntervalInvalid'
}

@Component({
    selector: 'cm-form-field-error',
    template: `
    @if (computedError) {

      {{ computedError | async }}
    
}
  `,
    encapsulation: ViewEncapsulation.None,
    changeDetection: ChangeDetectionStrategy.OnPush,
    standalone: false
})
export class FormFieldErrorComponent {
  readonly ngControl = inject(NgControl, { optional: true, self: true });
  readonly filed = inject(MatFormField, { optional: true });
  private readonly formArrayName = inject<FormArrayName | null>(FormArrayName, { optional: true, self: true });
  private readonly formGroupName = inject<FormGroupName | null>(FormGroupName, { optional: true, self: true });
  private readonly formGroup = inject<FormGroupDirective | null>(FormGroupDirective, { optional: true, self: true });

  static errorsMap = new Map<ErrorsDefault, string>([
    [ErrorsDefault.Required, '该字段为必填项'],
    [ErrorsDefault.Max, '最大值{{max}}'],
    [ErrorsDefault.Min, '最小值{{min}}'],
    [ErrorsDefault.MaxLength, '最多{{maxLength}}个字符'],
    [ErrorsDefault.MinLength, '至少{{minLength}}个字符'],
    [ErrorsDefault.Email, '邮箱无效'],
    [ErrorsDefault.Duplicate, '{ {{label}} } 已存在。'],
    [ErrorsDefault.InvalidChar, '必须是有效字符'],
    [ErrorsDefault.ConfirmPasswordMismatch, '两次输入的密码不一致'],
    [ErrorsDefault.userNameInvalid, '用户名必须为字母开头，包含大小写、数字至少一种'],
    [ErrorsDefault.CronMinIntervalInvalid, '最低执行间隔应该大于{{minInterval}}分钟'],
  ]);

  @Input() control?: AbstractControl;

  @Input() tip: string | ValidationErrors | null = null;

  get computedError(): Observable<string> | null {
    if (this.invalid && this.touched) {
      return this.error;
    }
    return null;
  }

  /**
   * 可以返回 ValidationErrors
   * 类似 { [prop: string]: string | Observable<string> }
   */
  get error(): Observable<string> {
    const tip = this.tip;

    if (typeof tip === 'string' && tip !== '') {
      return of(tip);
    }

    const errors = this.errors;

    const key = Object.keys(errors)[0] as ErrorsDefault;

    if (typeof tip === 'object' && tip !== null) {
      // eslint-disable-next-line no-prototype-builtins
      if ((tip as ValidationErrors).hasOwnProperty(key)) {
        const v = (tip as ValidationErrors)[key];
        if (typeof v === 'string') {
          return of(v);
        }
        if (v instanceof Observable) {
          return v;
        }
      }
    }
    if (Object.values(ErrorsDefault).includes(key)) {
      const value = FormFieldErrorComponent.errorsMap.get(key);
      if (value !== undefined) {
        if (key === ErrorsDefault.Required) {
          return of(this.getTranslationOrValue(value));
        }
        if (key === ErrorsDefault.Max) {
          return of(this.getTranslationOrValue(value, { max: this.max }));
        }
        if (key === ErrorsDefault.MaxLength) {
          return of(this.getTranslationOrValue(value, { maxLength: this.maxLength }));
        }
        if (key === ErrorsDefault.Min) {
          return of(this.getTranslationOrValue(value, { min: this.min }));
        }
        if (key === ErrorsDefault.MinLength) {
          return of(this.getTranslationOrValue(value, { minLength: this.minLength }));
        }
        if (key === ErrorsDefault.Email) {
          return of(this.getTranslationOrValue(value));
        }
        if (key === ErrorsDefault.Duplicate) {
          return of(this.getTranslationOrValue(value, { label: this.currentVal }));
        }
        if (key === ErrorsDefault.InvalidChar) {
          return of(this.getTranslationOrValue(value));
        }

        if (key === ErrorsDefault.ConfirmPasswordMismatch) {
          return of(this.getTranslationOrValue(value));
        }

        if (key === ErrorsDefault.userNameInvalid) {
          return of(this.getTranslationOrValue(value));
        }
        if (key === ErrorsDefault.CronMinIntervalInvalid) {
          return of(this.getTranslationOrValue(value, { minInterval: this.minInterval }));
        }
      } else {
        return of('字段未定义');
      }
    }

    const r = errors[key];
    if (typeof r === 'string') {
      return of(this.getTranslationOrValue(r));
    }
    if (r instanceof Observable) {
      return r;
    }

    return of('');
  }


  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  private get currentVal(): any {
    return !!this._control && this._control.value;
  }

  private get maxLength(): number {
    return this.errors[ErrorsDefault.MaxLength].requiredLength;
  }

  private get minLength(): number {
    return this.errors[ErrorsDefault.MinLength].requiredLength;
  }

  private get max(): number {
    return this.errors[ErrorsDefault.Max].max;
  }

  private get min(): number {
    return this.errors[ErrorsDefault.Min].min;
  }

  private get minInterval(): number {
    return this.errors[ErrorsDefault.CronMinIntervalInvalid].minInterval;
  }

  private get errors(): ValidationErrors {
    if (this._control && this._control.errors) {
      return this._control.errors;
    }
    return {};
  }

  private get invalid(): boolean {
    return !!this._control && this._control.invalid;
  }

  private get touched(): boolean {
    return !!this._control && this._control.touched;
  }

  private get _control(): AbstractControl | null {
    if (this.control) {
      return this.control;
    }

    if (this.ngControl) {
      return this.ngControl.control;
    }

    if (this.formArrayName) {
      return this.formArrayName.control;
    }

    if (this.formGroupName) {
      return this.formGroupName.control;
    }

    if (this.formGroup) {
      return this.formGroup.control;
    }

    if (this.filed) {
      const filedControl = this.filed._control.ngControl;
      if (filedControl) {
        return filedControl.control;
      }
    }

    return null;
  }

  constructor() {
    if (this.ngControl !== null) {
      this.ngControl.valueAccessor = this;
    }
  }

  writeValue() {}

  registerOnChange() {}

  registerOnTouched() {}

  setDisabledState() {}

  /**
   * 替换字符
   * @param key
   * @param replacements
   * @returns
   */
  private getTranslationOrValue(key: string, replacements?: Record<string, string | number>): string {
    // 假设这里并没有实际翻译，只是返回了翻译键
    let translation = key;

    // 如果有替换对象，遍历并替换翻译中的占位符
    if (replacements) {
      for (const [placeholder, value] of Object.entries(replacements)) {
        // 使用简单的字符串替换来模拟占位符替换（注意：这可能会引入问题，特别是当占位符重叠时）
        translation = translation.replace(new RegExp(`\\{\\{${placeholder}\\}\\}`, 'g'), value.toString());
      }
    }

    return translation;
  }
}
