import { version } from '../../version';
import { Environment } from '../app/common/interfaces/Environment';

export const environment = {
  production: false,
  mockBackend: true,
  backEndUrl: '',
  version: version
} as Environment;
