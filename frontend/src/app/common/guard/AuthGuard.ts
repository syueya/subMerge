import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { CmNotificationService } from '@common/modules/notification';

import { AuthService } from '../services/auth.service';

export const authGuard: CanActivateFn = async () => {
  const authService = inject(AuthService);
  const router = inject(Router);
  const notification = inject(CmNotificationService);

  await authService.waitForState();
  if (authService.isLoggedIn()) {
    return true;
  }
  router.navigate(['/auth/login']);
  notification.error('登录已过期', '未登录或登录已过期，请重新登录。');
  return false;
};
