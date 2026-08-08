import { Component, Input, OnDestroy, OnInit, forwardRef } from '@angular/core';
import { ControlValueAccessor, FormControl, NG_VALUE_ACCESSOR } from '@angular/forms';
import { Subject, takeUntil } from 'rxjs';

export interface SelectWithCreateOption {
  value: string;
  text: string;
}

/**
 * 可搜索下拉 + 自由输入新建：
 * - 输入即过滤已有选项
 * - 也可直接输入新名称（不强制从列表选）
 * - 空串表示未分类（不填即可）
 * - CVA 输出最终业务字符串（已有 value 或新建名称）
 */
@Component({
  selector: 'cm-select-with-create',
  standalone: false,
  templateUrl: './select-with-create.component.html',
  styleUrl: './select-with-create.component.scss',
  providers: [
    {
      provide: NG_VALUE_ACCESSOR,
      useExisting: forwardRef(() => SelectWithCreateComponent),
      multi: true
    }
  ]
})
export class SelectWithCreateComponent implements ControlValueAccessor, OnInit, OnDestroy {
  /** 业务选项 */
  @Input() options: SelectWithCreateOption[] = [];
  /** 是否允许输入不在列表中的新值（默认允许） */
  @Input() allowCreate = true;
  /** 输入框占位 */
  @Input() placeholder = '不填则归入未分类';
  /** 无匹配时的提示文案前缀 */
  @Input() createHint = '使用新名称';

  inputCtrl = new FormControl<string>('', { nonNullable: true });

  /** 当前已提交的业务值 */
  private committedValue = '';
  private destroy$ = new Subject<void>();
  private onChange: (v: string) => void = () => undefined;
  private onTouched: () => void = () => undefined;
  private writing = false;

  ngOnInit(): void {
    this.inputCtrl.valueChanges.pipe(takeUntil(this.destroy$)).subscribe((raw) => {
      if (this.writing) return;
      this.emitFromInput(raw);
    });
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }

  writeValue(value: string | null): void {
    this.writing = true;
    const v = value === null || value === undefined ? '' : String(value);
    this.committedValue = v;
    this.inputCtrl.setValue(this.displayTextFor(v), { emitEvent: false });
    this.writing = false;
  }

  registerOnChange(fn: (v: string) => void): void {
    this.onChange = fn;
  }

  registerOnTouched(fn: () => void): void {
    this.onTouched = fn;
  }

  setDisabledState(isDisabled: boolean): void {
    if (isDisabled) {
      this.inputCtrl.disable({ emitEvent: false });
    } else {
      this.inputCtrl.enable({ emitEvent: false });
    }
  }

  markTouched(): void {
    this.onTouched();
  }

  /**
   * 过滤候选。
   * - 空输入：显示全部
   * - 输入正好等于「当前已选项」的展示文案：显示全部（避免选中后只剩一项）
   * - 否则按 text/value 模糊过滤
   */
  filteredOptions(): SelectWithCreateOption[] {
    const list = this.options || [];
    const q = this.inputCtrl.value.trim();
    if (!q) return list;

    const committedOpt = list.find((o) => o.value === this.committedValue);
    if (committedOpt) {
      const committedText = this.displayTextFor(committedOpt.value) || committedOpt.text;
      if (q === committedText || q === committedOpt.value) {
        return list;
      }
    }

    const lower = q.toLowerCase();
    return list.filter((o) => {
      const text = String(o.text || '').toLowerCase();
      const value = String(o.value || '').toLowerCase();
      return text.includes(lower) || value.includes(lower);
    });
  }

  /** 当前输入是否已精确匹配某个选项 */
  exactMatch(): SelectWithCreateOption | null {
    const q = this.inputCtrl.value.trim();
    if (!q) return null;
    return (
      (this.options || []).find(
        (o) => o.value === q || String(o.text || '').trim() === q
      ) || null
    );
  }

  /** 是否展示「使用新名称」提示行 */
  showCreateHint(): boolean {
    if (!this.allowCreate) return false;
    const q = this.inputCtrl.value.trim();
    if (!q) return false;
    return !this.exactMatch();
  }

  onOptionSelected(text: string): void {
    const opt = (this.options || []).find((o) => o.text === text || o.value === text);
    this.writing = true;
    if (opt) {
      this.committedValue = opt.value;
      this.inputCtrl.setValue(this.displayTextFor(opt.value), { emitEvent: false });
      this.writing = false;
      this.onChange(opt.value);
    } else {
      const name = String(text || '').trim();
      this.committedValue = name;
      this.inputCtrl.setValue(name, { emitEvent: false });
      this.writing = false;
      this.onChange(name);
    }
    this.markTouched();
  }

  private emitFromInput(raw: string): void {
    const q = String(raw ?? '').trim();
    if (!q) {
      // 空串 = 未分类
      this.committedValue = '';
      this.onChange('');
      return;
    }
    const opt = (this.options || []).find(
      (o) => o.value === q || String(o.text || '').trim() === q
    );
    if (opt) {
      this.committedValue = opt.value;
      this.onChange(opt.value);
      return;
    }
    if (this.allowCreate) {
      this.committedValue = q;
      this.onChange(q);
    }
  }

  /** 业务值 → 输入框展示；空值不填文案（未分类靠 placeholder 说明） */
  private displayTextFor(value: string): string {
    if (!value) return '';
    const opt = (this.options || []).find((o) => o.value === value);
    return opt ? opt.text : value;
  }
}
