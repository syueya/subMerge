import { HttpErrorResponse, HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { Router } from '@angular/router';
import { catchError, throwError } from 'rxjs';
import { AuthService } from '../features/auth/auth.service';

/**
 * 统一鉴权横切：
 * 1. 同源 /api 请求带上凭据（HttpOnly 会话 cookie，JS 读不到，抗 XSS 窃取）
 * 2. HTTP 401 清会话并跳登录（已在登录页则不重复跳转）
 *
 * 注意：业务错误（如旧密码错误）后端用 403，不会触发此处。
 */
export const authInterceptor: HttpInterceptorFn = (req, next) => {
	const auth = inject(AuthService);
	const router = inject(Router);

	const authReq = req.url.startsWith('/api')
		? req.clone({ withCredentials: true })
		: req;

	return next(authReq).pipe(
		catchError((err: unknown) => {
			if (err instanceof HttpErrorResponse && err.status === 401) {
				auth.clearSession();
				const path = router.url || '';
				if (!path.startsWith('/login')) {
					void router.navigateByUrl('/login');
				}
			}
			return throwError(() => err);
		}),
	);
};
