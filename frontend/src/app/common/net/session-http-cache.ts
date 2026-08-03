import {
  HttpContext,
  HttpContextToken,
  HttpEvent,
  HttpHandlerFn,
  HttpInterceptorFn,
  HttpRequest,
  HttpResponse
} from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable, of } from 'rxjs';
import { finalize, shareReplay, tap } from 'rxjs/operators';

/**
 * 请求级缓存策略（仅前端 HttpContext，不落网关头）：
 * - default：可读接口优先命中会话内存缓存
 * - bypass：强制出网；成功后仍写回缓存（轮询/手动刷新用，保证再进菜单也是较新数据）
 */
export type WtHttpCacheMode = 'default' | 'bypass';

export const WT_HTTP_CACHE = new HttpContextToken<WtHttpCacheMode>(() => 'default');

/** 轮询、手动刷新等需要最新数据时使用 */
export function withWtHttpCacheBypass(context?: HttpContext): HttpContext {
  return (context ?? new HttpContext()).set(WT_HTTP_CACHE, 'bypass');
}

interface CacheEntry {
  response: HttpResponse<unknown>;
  expireAt: number;
}

/**
 * 会话级 HTTP 内存缓存：菜单离开再进入时复用列表/详情读请求，减少重复打接口。
 * 写操作成功或登出时整表清空，避免脏读。
 */
@Injectable({ providedIn: 'root' })
export class SessionHttpCacheService {
  /** 默认 5 分钟；同会话内切菜单回来通常仍命中 */
  private readonly ttlMs = 5 * 60 * 1000;
  private readonly maxEntries = 80;
  private readonly store = new Map<string, CacheEntry>();
  private readonly inflight = new Map<string, Observable<HttpEvent<unknown>>>();

  clear(): void {
    this.store.clear();
    this.inflight.clear();
  }

  isBypass(req: HttpRequest<unknown>): boolean {
    return req.context.get(WT_HTTP_CACHE) === 'bypass';
  }

  /** 是否属于可读列表/详情类接口（与是否 bypass 无关） */
  isReadRequest(req: HttpRequest<unknown>): boolean {
    if (req.method !== 'GET' && req.method !== 'POST') {
      return false;
    }
    if (req.responseType && req.responseType !== 'json') {
      return false;
    }
    if (typeof FormData !== 'undefined' && req.body instanceof FormData) {
      return false;
    }

    const path = this.pathOnly(req.url);
    if (!path.startsWith('/api/')) {
      return false;
    }

    if (req.method === 'POST') {
      return this.isReadPostPath(path);
    }
    return this.isReadGetPath(path);
  }

  /** 可走缓存路径：读接口且未 bypass */
  isCacheable(req: HttpRequest<unknown>): boolean {
    return this.isReadRequest(req) && !this.isBypass(req);
  }

  isMutation(req: HttpRequest<unknown>): boolean {
    if (req.method === 'DELETE' || req.method === 'PUT' || req.method === 'PATCH') {
      return true;
    }
    if (req.method !== 'POST') {
      return false;
    }
    const path = this.pathOnly(req.url);
    if (!path.startsWith('/api/')) {
      return false;
    }
    return !this.isReadPostPath(path);
  }

  buildKey(req: HttpRequest<unknown>): string {
    const params = req.params
      .keys()
      .sort()
      .map(k => `${k}=${req.params.getAll(k)?.join(',') ?? ''}`)
      .join('&');
    const body = this.stableSerialize(req.body);
    return `${req.method}|${this.pathOnly(req.url)}|${params}|${body}`;
  }

  get(key: string): HttpResponse<unknown> | null {
    const entry = this.store.get(key);
    if (!entry) {
      return null;
    }
    if (Date.now() > entry.expireAt) {
      this.store.delete(key);
      return null;
    }
    return entry.response.clone();
  }

  set(key: string, response: HttpResponse<unknown>): void {
    if (response.status !== 200) {
      return;
    }
    const body = response.body as { code?: number; ok?: boolean } | null;
    if (body && typeof body === 'object') {
      if (typeof body.ok === 'boolean' && body.ok === false) {
        return;
      }
      if (typeof body.code === 'number' && body.code !== 20000) {
        return;
      }
    }
    if (this.store.size >= this.maxEntries) {
      const oldest = this.store.keys().next().value;
      if (oldest) {
        this.store.delete(oldest);
      }
    }
    this.store.set(key, {
      response: response.clone(),
      expireAt: Date.now() + this.ttlMs
    });
  }

  getInflight(key: string): Observable<HttpEvent<unknown>> | null {
    return this.inflight.get(key) ?? null;
  }

  setInflight(key: string, stream: Observable<HttpEvent<unknown>>): void {
    this.inflight.set(key, stream);
  }

  deleteInflight(key: string): void {
    this.inflight.delete(key);
  }

  private pathOnly(url: string): string {
    try {
      if (url.startsWith('http')) {
        return new URL(url).pathname;
      }
    } catch {
      // ignore
    }
    const q = url.indexOf('?');
    return q >= 0 ? url.slice(0, q) : url;
  }

  private isReadPostPath(path: string): boolean {
    return (
      path.endsWith('/page') ||
      path.endsWith('/list') ||
      path.endsWith('/taskPage') ||
      path.endsWith('/loginLogs')
    );
  }

  private isReadGetPath(path: string): boolean {
    if (/\/(login|bootstrap|logout|setup-status|register|initAdminStatus)(\/|$)/.test(path)) {
      return false;
    }
    // SubMerge 列表多为 GET /api/sources|/rules|/groups|/tokens|/releases|/proxies|/regions|/audit
    if (
      path === '/api/sources' ||
      path === '/api/rules' ||
      path === '/api/groups' ||
      path === '/api/tokens' ||
      path === '/api/releases' ||
      path === '/api/proxies' ||
      path === '/api/regions' ||
      path === '/api/audit' ||
      path === '/api/geo/status' ||
      path === '/api/geo/categories' ||
      path === '/api/auth/me' ||
      path === '/api/health'
    ) {
      return true;
    }
    return (
      path.endsWith('/list') ||
      path.endsWith('/details') ||
      path.endsWith('/detail') ||
      path.endsWith('/fileNames') ||
      path.endsWith('/taskDetail') ||
      path.endsWith('/resultList') ||
      path.endsWith('/blendResultList') ||
      path.endsWith('/blendDetail') ||
      path.endsWith('/feedbackList') ||
      path.endsWith('/tokens') ||
      path.endsWith('/info') ||
      path.endsWith('/byCompound')
    );
  }

  private stableSerialize(value: unknown): string {
    if (value == null || value === '') {
      return '';
    }
    if (typeof value === 'string') {
      return value;
    }
    try {
      return JSON.stringify(this.sortKeys(value));
    } catch {
      return String(value);
    }
  }

  private sortKeys(value: unknown): unknown {
    if (Array.isArray(value)) {
      return value.map(item => this.sortKeys(item));
    }
    if (value && typeof value === 'object' && !(value instanceof Date) && !(typeof FormData !== 'undefined' && value instanceof FormData)) {
      const obj = value as Record<string, unknown>;
      const sorted: Record<string, unknown> = {};
      for (const key of Object.keys(obj).sort()) {
        sorted[key] = this.sortKeys(obj[key]);
      }
      return sorted;
    }
    return value;
  }
}

/**
 * 会话读缓存拦截器：放在 defaultInterceptor 之后，命中则不再出网。
 */
export const sessionHttpCacheInterceptor: HttpInterceptorFn = (
  req: HttpRequest<unknown>,
  next: HttpHandlerFn
): Observable<HttpEvent<unknown>> => {
  const cache = inject(SessionHttpCacheService);

  if (cache.isMutation(req)) {
    return next(req).pipe(
      tap(event => {
        if (!(event instanceof HttpResponse) || event.status !== 200) {
          return;
        }
        const body = event.body as { code?: number; ok?: boolean } | null;
        if (body && typeof body === 'object') {
          if (typeof body.ok === 'boolean' && body.ok === false) {
            return;
          }
          if (typeof body.code === 'number' && body.code !== 20000) {
            return;
          }
        }
        cache.clear();
      })
    );
  }

  // 读接口：default 可命中缓存；bypass 强制出网但仍写回，供轮询/刷新
  if (!cache.isReadRequest(req)) {
    return next(req);
  }

  const key = cache.buildKey(req);
  const bypass = cache.isBypass(req);

  if (!bypass) {
    const hit = cache.get(key);
    if (hit) {
      return of(hit);
    }

    const inflight = cache.getInflight(key);
    if (inflight) {
      return inflight;
    }
  }

  const shared = next(req).pipe(
    tap(event => {
      if (event instanceof HttpResponse) {
        cache.set(key, event);
      }
    }),
    finalize(() => {
      if (!bypass) {
        cache.deleteInflight(key);
      }
    }),
    shareReplay({ bufferSize: 1, refCount: true })
  );

  if (!bypass) {
    cache.setInflight(key, shared);
  }
  return shared;
};
