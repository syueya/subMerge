import { MessageConfig } from "./MessageConfig";

export interface NotificationConfig extends MessageConfig {
    cmTop?: string | number;
    cmBottom?: string | number;
    cmPlacement?: 'topLeft' | 'topRight' | 'bottomLeft' | 'bottomRight' | 'top' | 'bottom';
  }