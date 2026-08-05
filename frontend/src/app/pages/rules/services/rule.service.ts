import { Injectable, inject } from '@angular/core';
import { Observable, tap } from 'rxjs';
import { ApiService } from '@common/net/api.service';
import { CachedRequest } from '@common/net/cached-request';
import { ListResponse, MatchableRule, ProxyGroup, Rule, RuleMatchResult } from '@data-struct';

@Injectable({ providedIn: 'root' })
export class RuleService {
	private readonly api = inject(ApiService);

	// 规则与策略组列表被概览/规则/策略组多页共用，做会话内缓存，写操作后失效。
	// force 时 bypass 会话 HTTP 缓存，避免工具栏刷新仍命中拦截器缓存。
	private readonly rulesCache = new CachedRequest<ListResponse<Rule>>((context) =>
		this.api.get('/rules', { context }),
	);
	private readonly groupsCache = new CachedRequest<ListResponse<ProxyGroup>>((context) =>
		this.api.get('/groups', { context }),
	);

	listRules(forceRefresh = false): Observable<ListResponse<Rule>> {
		return this.rulesCache.get(forceRefresh);
	}

	/** 使规则与策略组缓存失效（增删改规则/组后调用；改组也可能影响规则出口展示）。 */
	private invalidateRulesAndGroups(): void {
		this.rulesCache.invalidate();
		this.groupsCache.invalidate();
	}

	/** 包裹写请求：成功后失效缓存，保证下次列表拉到最新数据。 */
	private afterWrite<T>(req: Observable<T>): Observable<T> {
		return req.pipe(tap(() => this.invalidateRulesAndGroups()));
	}

	createRule(
		body: Partial<Rule> & { type: string; payload: string; target: string },
	): Observable<Rule> {
		return this.afterWrite(this.api.post('/rules', body));
	}

	/** 批量导入：一行一条 category,TYPE,payload,target[,note] 或仅 payload */
	batchImportRules(body: {
		text: string;
		defaultType?: string;
		defaultTarget?: string;
		defaultNote?: string;
		defaultCategory?: string;
		enabled?: boolean;
	}): Observable<{ created: number; skipped: number; items: Rule[]; errors?: string[] }> {
		return this.afterWrite(this.api.post('/rules/batch-import', body));
	}

	/** 批量修改目标出口 */
	batchUpdateTarget(ids: number[], target: string): Observable<{ updated: number }> {
		return this.afterWrite(this.api.put('/rules/batch-target', { ids, target }));
	}

	/** 批量启用/禁用 */
	batchUpdateEnabled(ids: number[], enabled: boolean): Observable<{ updated: number }> {
		return this.afterWrite(this.api.put('/rules/batch-enabled', { ids, enabled }));
	}

	/** 批量改业务分类 */
	batchUpdateCategory(ids: number[], category: string): Observable<{ updated: number }> {
		return this.afterWrite(this.api.put('/rules/batch-category', { ids, category }));
	}

	/** 批量删除 */
	batchDeleteRules(ids: number[]): Observable<{ deleted: number }> {
		return this.afterWrite(this.api.post('/rules/batch-delete', { ids }));
	}

	updateRule(
		id: number,
		body: Partial<Rule> & { type: string; payload: string; target: string },
	): Observable<Rule> {
		return this.afterWrite(this.api.put(`/rules/${id}`, body));
	}

	deleteRule(id: number): Observable<{ success: boolean }> {
		return this.afterWrite(this.api.delete(`/rules/${id}`));
	}

	reorder(orderedIds: number[]): Observable<{ success: boolean }> {
		return this.afterWrite(this.api.post('/rules/reorder', { orderedIds }));
	}

	/**
	 * 服务端模拟规则匹配（含 GEOSITE/GEOIP，使用面板已加载的 geo 数据）。
	 * rules 为当前草稿或某次发布快照，不读库。
	 */
	matchRules(
		input: string,
		rules: MatchableRule[],
		resolve = false,
	): Observable<RuleMatchResult> {
		return this.api.post<RuleMatchResult>('/rules/match', { input, rules, resolve });
	}

	listGroups(forceRefresh = false): Observable<ListResponse<ProxyGroup>> {
		return this.groupsCache.get(forceRefresh);
	}

	createGroup(body: {
		name: string;
		type: string;
		proxies: string[];
		url?: string;
		interval?: number;
		enabled?: boolean;
		sortOrder?: number;
	}): Observable<ProxyGroup> {
		return this.afterWrite(this.api.post('/groups', body));
	}

	updateGroup(
		id: number,
		body: {
			name: string;
			type: string;
			proxies: string[];
			url?: string;
			interval?: number;
			enabled?: boolean;
			sortOrder?: number;
		},
	): Observable<ProxyGroup> {
		return this.afterWrite(this.api.put(`/groups/${id}`, body));
	}

	deleteGroup(id: number, cascadeRules = false): Observable<{ success: boolean }> {
		const q = cascadeRules ? '?cascadeRules=1' : '';
		return this.afterWrite(this.api.delete(`/groups/${id}${q}`));
	}
}
