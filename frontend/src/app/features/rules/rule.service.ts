import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { ApiService } from '../../core/api.service';
import { ListResponse, ProxyGroup, Rule } from '../../common/types';

@Injectable({ providedIn: 'root' })
export class RuleService {
 private readonly api = inject(ApiService);

 listRules(): Observable<ListResponse<Rule>> {
 return this.api.get('/rules');
 }

createRule(body: Partial<Rule> & { type: string; payload: string; target: string }): Observable<Rule> {
	 return this.api.post('/rules', body);
	 }

	/** 批量导入：一行一条 TYPE,payload,target[,note] 或仅 payload */
	batchImportRules(body: {
		text: string;
		defaultType?: string;
		defaultTarget?: string;
		defaultNote?: string;
		enabled?: boolean;
	}): Observable<{ created: number; skipped: number; items: Rule[]; errors?: string[] }> {
		return this.api.post('/rules/batch-import', body);
	}
	
	 updateRule(id: number, body: Partial<Rule> & { type: string; payload: string; target: string }): Observable<Rule> {
	 return this.api.put(`/rules/${id}`, body);
	 }

 deleteRule(id: number): Observable<{ success: boolean }> {
 return this.api.delete(`/rules/${id}`);
 }

 reorder(orderedIds: number[]): Observable<{ success: boolean }> {
 return this.api.post('/rules/reorder', { orderedIds });
 }

 listGroups(): Observable<ListResponse<ProxyGroup>> {
 return this.api.get('/groups');
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
 return this.api.post('/groups', body);
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
 return this.api.put(`/groups/${id}`, body);
 }

deleteGroup(id: number, cascadeRules = false): Observable<{ success: boolean }> {
		const q = cascadeRules ? '?cascadeRules=1' : '';
		return this.api.delete(`/groups/${id}${q}`);
	}
}
