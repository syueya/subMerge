import type { TokenGroupMode, TokenStatus } from '../enums/token';

export interface ShareToken {
	id: number;
	name: string;
	token?: string;
	tokenMasked: string;
	status: TokenStatus;
	/** 允许的订阅源；空数组 = 全部源 */
	sourceIds: number[];
	/** 与 sourceIds 对应的源名（已失效源会带标记） */
	sourceNames?: string[];
	/** 策略组投影：auto / all / custom */
	groupMode: TokenGroupMode;
	/** custom 时的策略组白名单 */
	groupNames?: string[];
	accessCount: number;
	lastAccessAt?: string | null;
	createdAt: string;
	updatedAt: string;
	subscribeUrl?: string;
}

export interface TokenUpsertBody {
	name?: string;
	status?: TokenStatus;
	/** 传 [] 表示全部源；非空为指定源 */
	sourceIds?: number[];
	groupMode?: TokenGroupMode;
	/** custom 时的策略组名白名单 */
	groupNames?: string[];
}
