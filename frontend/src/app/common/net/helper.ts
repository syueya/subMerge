/* eslint-disable @typescript-eslint/no-explicit-any */
import { HttpHeaders, HttpResponseBase } from '@angular/common/http';
import { Injector, inject } from '@angular/core';
import { Router } from '@angular/router';
import { CmNotificationService } from '@common/modules/notification';
import { AuthService } from '@common/services/auth.service';
import { REQUEST_NO_NEET_AUTH_CHECK } from '@common/util';

export interface ReThrowHttpError {
  body: any;
  _throw: true;
}

export const CODEMESSAGE: Record<number, string> = {
  200: '服务器成功返回请求的数据。',
  201: '新建或修改数据成功。',
  202: '一个请求已经进入后台排队（异步任务）。',
  204: '删除数据成功。',
  400: '发出的请求有错误，服务器没有进行新建或修改数据的操作。',
  401: '用户没有权限（令牌、用户名、密码错误）。',
  403: '用户得到授权，但是访问是被禁止的。',
  404: '发出的请求针对的是不存在的记录，服务器没有进行操作。',
  406: '请求的格式不可得。',
  410: '请求的资源被永久删除，且不会再得到的。',
  422: '当创建一个对象时，发生一个验证错误。',
  500: '后端服务发生错误，请检查后端服务状态。',
  502: '后端服务错误。',
  503: '后端服务不可用，服务器暂时过载或维护。',
  504: '后端服务超时。'
};

export function toLogin(injector: Injector): void {
  injector.get(CmNotificationService).error(`未登录或登录已过期，请重新登录。`, ``);
  injector.get(AuthService).clearSessionOnly();
  injector.get(Router).navigateByUrl('/auth/login');
}

/**
 * 处理请求头：
 * - SubMerge 使用 Cookie 会话，不注入 Authorization
 * - 仍剥离 no-auth / timeout 标记头
 */
export function processHeaders(headers?: HttpHeaders): HttpHeaders {
  if (!headers) {
    headers = new HttpHeaders();
  }

  if (headers.has(REQUEST_NO_NEET_AUTH_CHECK)) {
    headers = headers.delete(REQUEST_NO_NEET_AUTH_CHECK);
  }

  if (headers.has('timeout')) {
    headers = headers.delete('timeout');
  }

  // 避免误带模板遗留的 Authorization
  if (headers.has('Authorization')) {
    headers = headers.delete('Authorization');
  }

  return headers;
}

export function checkStatus(injector: Injector, ev: HttpResponseBase): void {
  // 403 可能是业务错误（如旧密码错误），不在此全局告警
  if ((ev.status >= 200 && ev.status < 300) || ev.status === 401 || ev.status === 403) {
    return;
  }
  if (ev.status === 0) {
    injector.get(CmNotificationService).error(`请求错误`, `接口： ${ev.url} 错误：未知错误，请检查网络或后端服务状态。`);
  } else {
    const errortext = CODEMESSAGE[ev.status] || ev.statusText || '未知错误';
    // 尝试提取后端 error.message
    const body = (ev as any)?.error;
    const detail = body?.error?.message || body?.message;
    injector
      .get(CmNotificationService)
      .error(`请求错误 ${ev.status}`, detail || `接口： ${ev.url}  错误：${errortext}`);
  }
}
