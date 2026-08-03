import { Provider } from '@angular/core';
import { MatPaginatorIntl } from '@angular/material/paginator';
import { CustomPaginatorIntl } from '@common/util/CustomPaginator';


export const PaginatorIntlProvider: Provider = {
  provide: MatPaginatorIntl,
  useClass: CustomPaginatorIntl,
};
