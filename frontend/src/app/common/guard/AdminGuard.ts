import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { CmNotificationService } from '@common/modules/notification';
import { UserRoleEnum } from '@data-struct';

import { AuthService } from '../services/auth.service';

/**
 * 管理员路由守卫: 仅允许管理员访问。
 * 与 authGuard 配合使用——authGuard 保证已登录,adminGuard 保证是管理员。
 * 普通用户直接访问管理员路由(如手敲 URL)会被重定向到首页并提示无权限。
 */
export const adminGuard: CanActivateFn = async () => {
  const authService = inject(AuthService);
  const router = inject(Router);
  const notification = inject(CmNotificationService);

  await authService.waitForState();
  const userInfo = authService.getUserInfo();
  if (userInfo && Number(userInfo.role) === UserRoleEnum.Admin) {
    return true;
  }
  notification.error('无访问权限', '该页面仅管理员可访问。');
  router.navigate(['/main']);
  return false;
};
