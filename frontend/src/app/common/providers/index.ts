
import {ErrorHandlerProvider} from './error-handler.provider';
import {PaginatorIntlProvider} from './paginatorIntl.provider';


export * from './paginatorIntl.provider';
export * from './error-handler.provider';

export const CmProviders = [
  PaginatorIntlProvider,
  ErrorHandlerProvider,
];
