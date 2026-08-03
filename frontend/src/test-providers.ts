import { EnvironmentProviders, importProvidersFrom, Provider } from '@angular/core';
import { MaterialModule } from '@common/material.module';
import { IconsModule } from '@common/modules/icons/icons.module';
import { CustomTablerIcons } from '@common/modules/icons/import-config';
import { provideTablerIcons } from '@luoxiao123/angular-tabler-icons';

const testProviders: Array<Provider | EnvironmentProviders> = [
  importProvidersFrom(MaterialModule, IconsModule),
  provideTablerIcons(CustomTablerIcons)
];

export default testProviders;
