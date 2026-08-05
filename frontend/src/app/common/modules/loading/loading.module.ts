import { CommonModule } from '@angular/common';
import { ModuleWithProviders, NgModule } from '@angular/core';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';

import { LoadingComponent } from './loading/loading.component';

@NgModule({
  declarations: [LoadingComponent],
  imports: [CommonModule, MatProgressSpinnerModule],
  exports: [LoadingComponent]
})
export class LoadingModule {
  static forRoot(): ModuleWithProviders<LoadingModule> {
    return {
      ngModule: LoadingModule,
      // pending 计数改由 functional interceptor 注册，不再 provide HTTP_INTERCEPTORS
      providers: []
    };
  }
}
