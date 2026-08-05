export interface AppSettings {
  /** 浅色 light / 深色 dark */
  theme: string;
  /** 侧边栏是否打开 */
  sidenavOpened: boolean;
  /** 侧边栏是否收起为 mini */
  sidenavCollapsed: boolean;
  /** 主题色 class，对应 html.xxx_theme，默认 blue_theme */
  activeTheme: string;
}

export const defaults: AppSettings = {
  theme: 'light',
  sidenavOpened: true,
  sidenavCollapsed: false,
  activeTheme: 'blue_theme',
};
