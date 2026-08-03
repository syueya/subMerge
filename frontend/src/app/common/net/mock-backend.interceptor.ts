import { HttpEvent, HttpHandlerFn, HttpInterceptorFn, HttpRequest, HttpResponse } from '@angular/common/http';
import { environment } from '@env/environment';
import { Observable, of } from 'rxjs';

function ok<T>(data: T): Observable<HttpEvent<unknown>> {
  return of(new HttpResponse({
    status: 200,
    body: { ok: true, data }
  }));
}

export const mockBackendInterceptor: HttpInterceptorFn = (req: HttpRequest<unknown>, next: HttpHandlerFn): Observable<HttpEvent<unknown>> => {
  if (!environment.mockBackend) {
    return next(req);
  }

  const path = req.url.split('?')[0];
  switch (path) {
    case '/api/auth/setup-status':
      return ok({ needsSetup: false });
    case '/api/auth/login':
      return ok({
        user: {
          id: 1,
          username: 'admin',
          displayName: 'Admin',
          avatar: '',
          createdAt: new Date().toISOString()
        }
      });
    case '/api/auth/me':
      return ok({
        user: {
          id: 1,
          username: 'admin',
          displayName: 'Admin',
          avatar: '',
          createdAt: new Date().toISOString()
        }
      });
    case '/api/auth/logout':
      return ok({ success: true });
    default:
      return next(req);
  }
};
