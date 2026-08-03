import { CommonModule } from '@angular/common';
import { ModuleWithProviders, NgModule } from '@angular/core';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';

import { LoadingComponent } from './loading/loading.component';
import { PendingRequestsInterceptorProvider } from './services/pending-requests-interceptor.service';


@NgModule({
  declarations: [LoadingComponent],
  imports: [
    CommonModule,
    MatProgressSpinnerModule,
  ],
  exports: [LoadingComponent]
})
export class LoadingModule {
  static forRoot(): ModuleWithProviders<LoadingModule> {
    return {
        ngModule: LoadingModule,
        providers: [
            PendingRequestsInterceptorProvider,
        ]
    };
}
}
