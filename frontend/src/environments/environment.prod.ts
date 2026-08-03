// This file can be replaced during build by using the `fileReplacements` array.
// `ng build ---prod` replaces `environment.ts` with `environment.prod.ts`.
// The list of file replacements can be found in `angular.json`.


import { version } from '../../version';
import { Environment } from '../app/common/interfaces/Environment';

export const environment = {
  production: true,
  mockBackend: false,
  backEndUrl: '',
  version: version
} as Environment;
