import type { APIKeyScope, APIKeyStatus } from '../enums/api-key';

/** 管理端 API 密钥（agent / 自动化） */
export interface APIKey {
	id: number;
	name: string;
	/** 仅 create / regenerate / secret 返回 */
	key?: string;
	keyMasked: string;
	scopes: APIKeyScope[];
	status: APIKeyStatus;
	note?: string;
	expiresAt?: string | null;
	lastUsedAt?: string | null;
	createdBy: string;
	createdAt: string;
	updatedAt: string;
}

export interface APIKeySecret {
	id: number;
	key: string;
}

export interface APIKeyUpsertBody {
	name?: string;
	scopes?: APIKeyScope[];
	status?: APIKeyStatus;
	note?: string;
	/** 传 '' 清空过期；RFC3339 设置；省略不改 */
	expiresAt?: string | null;
}
