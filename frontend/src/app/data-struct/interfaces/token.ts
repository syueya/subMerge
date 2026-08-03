import type { TokenGroupMode, TokenStatus } from '../enums/enums';

export interface TokenUpsertBody {
	name?: string;
	status?: TokenStatus;
	/** 传 [] 表示全部源；非空为指定源 */
	sourceIds?: number[];
	groupMode?: TokenGroupMode;
	/** custom 时的策略组名白名单 */
	groupNames?: string[];
}
