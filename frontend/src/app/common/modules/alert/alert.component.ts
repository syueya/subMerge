/**
 * Use of this source code is governed by an MIT-style license that can be
 * found in the LICENSE file at https://github.com/NG-ZORRO/ng-zorro-antd/blob/master/LICENSE
 */

import { Direction, Directionality } from '@angular/cdk/bidi';
import { ChangeDetectionStrategy, ChangeDetectorRef, Component, EventEmitter, Input, OnChanges, OnDestroy, OnInit, Output, SimpleChanges, TemplateRef, ViewEncapsulation, booleanAttribute, inject } from '@angular/core';
import { CmStringTemplateOutletDirective } from '@common/directives/string_template_outlet.directive';
import { Subject } from 'rxjs';
import { takeUntil } from 'rxjs/operators';

import { IconsModule } from '../icons/icons.module';

const NZ_CONFIG_MODULE_NAME = 'alert';

@Component({
  selector: 'cm-alert',
  exportAs: 'cmAlert',
  imports: [IconsModule, CmStringTemplateOutletDirective],
  template: `
    @if (!closed) {
      <div
        class="ms-alert"
        [class.ms-alert-rtl]="dir === 'rtl'"
        [class.ms-alert-success]="type === 'success'"
        [class.ms-alert-info]="type === 'info'"
        [class.ms-alert-warning]="type === 'warning'"
        [class.ms-alert-error]="type === 'error'"
        [class.ms-alert-no-icon]="!showIcon"
        [class.ms-alert-banner]="banner"
        [class.ms-alert-closable]="showClose"
        [class.ms-alert-with-description]="!!description"
        [class.ms-alert-no-animation]="alertNoAnimation"
        animate.leave="ms-alert-leave"
        (animate.leave)="onFadeAnimationDone()"
      >
        @if (showIcon) {
          <div class="ms-alert-icon">
            @if (custonIcon) {
              <ng-container *cmStringTemplateOutlet="custonIcon"></ng-container>
            } @else {
              <i-tabler [name]="iconName" class="icon-20"></i-tabler>
            }
          </div>
        }

        @if (message || description) {
          <div class="ms-alert-content">
            @if (message) {
              <span class="ms-alert-message">
                <ng-container *cmStringTemplateOutlet="message">{{ message }}</ng-container>
              </span>
            }
            @if (description) {
              <span class="ms-alert-description">
                <ng-container *cmStringTemplateOutlet="description">{{ description }}</ng-container>
              </span>
            }
          </div>
        }

        @if (action) {
          <div class="ms-alert-action">
            <ng-container *cmStringTemplateOutlet="action">{{ action }}</ng-container>
          </div>
        }

        @if (showClose || closeText) {
          <button type="button" tabindex="0" class="ms-alert-close-icon" (click)="closeAlert()">
            @if (closeText) {
              <ng-container *cmStringTemplateOutlet="closeText">
                <span class="ms-alert-close-text">{{ closeText }}</span>
              </ng-container>
            } @else {
              <i-tabler name="x" class="icon-20"></i-tabler>
            }
          </button>
        }
      </div>
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
  encapsulation: ViewEncapsulation.None,
  preserveWhitespaces: false
})
export class CmAlertComponent implements OnChanges, OnDestroy, OnInit {
  private cdr = inject(ChangeDetectorRef);
  private directionality = inject(Directionality);

  readonly _alertModuleName = NZ_CONFIG_MODULE_NAME;

  @Input() action: string | TemplateRef<void> | null = null;
  @Input() closeText: string | TemplateRef<void> | null = null;
  @Input() iconName: string | null = null;
  @Input() message: string | TemplateRef<void> | null = null;
  @Input() description: string | TemplateRef<void> | null = null;
  @Input() type: 'success' | 'info' | 'warning' | 'error' = 'info';
  @Input({ transform: booleanAttribute }) showClose = false;
  @Input({ transform: booleanAttribute }) showIcon = false;
  @Input({ transform: booleanAttribute }) banner = false;
  @Input({ transform: booleanAttribute }) alertNoAnimation = false;
  @Input() custonIcon: string | TemplateRef<void> | null = null;
  @Output() readonly closeEvent = new EventEmitter<boolean>();
  closed = false;
  dir: Direction = 'ltr';
  private isTypeSet = false;
  private isShowIconSet = false;
  private destroy$ = new Subject<boolean>();

  ngOnInit(): void {
    this.directionality.change?.pipe(takeUntil(this.destroy$)).subscribe((direction: Direction) => {
      this.dir = direction;
      this.cdr.detectChanges();
    });

    this.dir = this.directionality.value;
  }

  closeAlert(): void {
    this.closed = true;
  }

  onFadeAnimationDone(): void {
    if (this.closed) {
      this.closeEvent.emit(true);
    }
  }

  ngOnChanges(changes: SimpleChanges): void {
    const { showIcon, description, type, banner } = changes;
    if (showIcon) {
      this.isShowIconSet = true;
    }
    if (type) {
      this.isTypeSet = true;
      switch (this.type) {
        case 'error':
          this.iconName = 'circle-x';
          break;
        case 'success':
          this.iconName = 'circle-check';
          break;
        case 'info':
          this.iconName = 'info-circle';
          break;
        case 'warning':
          this.iconName = 'exclamation-circle';
          break;
      }
    }

    if (banner) {
      if (!this.isTypeSet) {
        this.type = 'warning';
      }
      if (!this.isShowIconSet) {
        this.showIcon = true;
      }
    }
  }
  ngOnDestroy(): void {
    this.destroy$.next(true);
    this.destroy$.complete();
  }
}
