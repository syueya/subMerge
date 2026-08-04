export interface AdminUser {
	id: number;
	username: string;
	displayName: string;
	avatar?: string;
	createdAt: string;
	lastLoginAt?: string | null;
}

export interface LoginRequest {
	username: string;
	password: string;
}

export interface LoginResponse {
	// 会话令牌改由 HttpOnly cookie 承载，不再出现在响应体中
	user: AdminUser;
}
