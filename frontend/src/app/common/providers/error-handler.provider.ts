import { ErrorHandler, Provider } from '@angular/core';
import { CmGlobalErrorHandler } from '@common/util';

export const ErrorHandlerProvider: Provider = {
  provide: ErrorHandler,
  useClass: CmGlobalErrorHandler,
};
