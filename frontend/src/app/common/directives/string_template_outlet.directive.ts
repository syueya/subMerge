/* eslint-disable @typescript-eslint/no-explicit-any */
/**
 * Use of this source code is governed by an MIT-style license that can be
 * found in the LICENSE file at https://github.com/NG-ZORRO/ng-zorro-antd/blob/master/LICENSE
 */

import { Directive, EmbeddedViewRef, Input, OnChanges, SimpleChange, SimpleChanges, TemplateRef, ViewContainerRef, inject } from '@angular/core';

@Directive({
  selector: '[cmStringTemplateOutlet]',
  exportAs: 'cmStringTemplateOutlet',
  standalone: true
})
export class CmStringTemplateOutletDirective<_T = unknown> implements OnChanges {
  private viewContainer = inject(ViewContainerRef);
  private templateRef = inject<TemplateRef<any>>(TemplateRef);

  private embeddedViewRef: EmbeddedViewRef<any> | null = null;
  private context = new CmStringTemplateOutletContext();
  @Input() cmStringTemplateOutletContext: any | null = null;
  @Input() cmStringTemplateOutlet: any | TemplateRef<any> = null;

  static ngTemplateContextGuard<T>(
    _dir: CmStringTemplateOutletDirective<T>,
    _ctx: any
  ): _ctx is CmStringTemplateOutletContext {
    return true;
  }

  private recreateView(): void {
    this.viewContainer.clear();
    const isTemplateRef = this.cmStringTemplateOutlet instanceof TemplateRef;
    const templateRef = (isTemplateRef ? this.cmStringTemplateOutlet : this.templateRef) as any;
    this.embeddedViewRef = this.viewContainer.createEmbeddedView(
      templateRef,
      isTemplateRef ? this.cmStringTemplateOutletContext : this.context
    );
  }

  private updateContext(): void {
    const isTemplateRef = this.cmStringTemplateOutlet instanceof TemplateRef;
    const newCtx = isTemplateRef ? this.cmStringTemplateOutletContext : this.context;
    const oldCtx = this.embeddedViewRef!.context as any;
    if (newCtx) {
      for (const propName of Object.keys(newCtx)) {
        oldCtx[propName] = newCtx[propName];
      }
    }
  }

  ngOnChanges(changes: SimpleChanges): void {
    const { cmStringTemplateOutletContext, cmStringTemplateOutlet } = changes;
    const shouldRecreateView = (): boolean => {
      let shouldOutletRecreate = false;
      if (cmStringTemplateOutlet) {
        if (cmStringTemplateOutlet.firstChange) {
          shouldOutletRecreate = true;
        } else {
          const isPreviousOutletTemplate = cmStringTemplateOutlet.previousValue instanceof TemplateRef;
          const isCurrentOutletTemplate = cmStringTemplateOutlet.currentValue instanceof TemplateRef;
          shouldOutletRecreate = isPreviousOutletTemplate || isCurrentOutletTemplate;
        }
      }
      const hasContextShapeChanged = (ctxChange: SimpleChange): boolean => {
        const prevCtxKeys = Object.keys(ctxChange.previousValue || {});
        const currCtxKeys = Object.keys(ctxChange.currentValue || {});
        if (prevCtxKeys.length === currCtxKeys.length) {
          for (const propName of currCtxKeys) {
            if (prevCtxKeys.indexOf(propName) === -1) {
              return true;
            }
          }
          return false;
        } else {
          return true;
        }
      };
      const shouldContextRecreate =
        cmStringTemplateOutletContext && hasContextShapeChanged(cmStringTemplateOutletContext);
      return shouldContextRecreate || shouldOutletRecreate;
    };

    if (cmStringTemplateOutlet) {
      this.context.$implicit = cmStringTemplateOutlet.currentValue;
    }

    const recreateView = shouldRecreateView();
    if (recreateView) {
      /** recreate view when context shape or outlet change **/
      this.recreateView();
    } else {
      /** update context **/
      this.updateContext();
    }
  }
}

export class CmStringTemplateOutletContext {
  public $implicit: any;
}
