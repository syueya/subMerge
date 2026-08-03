import { MessageConfig } from '@common/interfaces/MessageConfig';
import { NotificationConfig } from '@common/interfaces/NotificationConfig';

export const DEFAULT_TIMEOUT = 30000; // 默认超时时间 30 秒

/**
 * 存储token的key
 */
export const STORAGE_TOKEN_KEY = 'wt_token';
/**
 * 存储用户信息的key
 */
export const STORAGE_USERINFO_KEY = 'wt_userinfo';
/** 无需登录的请求 */
export const REQUEST_NO_NEET_AUTH_CHECK = 'request_no_neet_auth_check';
/**
 * 通知提示组件 默认配置
 */
export const CM_NOTIFICATION_DEFAULT_CONFIG: Required<NotificationConfig> = {
  cmTop: '24px',
  cmBottom: '24px',
  cmPlacement: 'topRight',
  cmDuration: 8500,
  cmMaxStack: 8,
  cmPauseOnHover: true,
  cmAnimate: true
};

/**
 * 消息提示组件 默认配置
 */
export const CM_MESSAGE_DEFAULT_CONFIG: Required<MessageConfig> = {
  cmAnimate: true,
  cmDuration: 5000,
  cmMaxStack: 7,
  cmPauseOnHover: true,
  cmTop: 70
};

/** trackBy */
export const TrackById = <T>(_: number, item: { id: number | string } & T) => item.id;

export const trackByItem = <T>(_: number, item: T) => item;
