export interface ApiError {
	code: string;
	message: string;
	details?: string;
}

export interface ApiResponse<T> {
	ok: boolean;
	data?: T;
	error?: ApiError;
}

export interface ListResponse<T> {
	items: T[];
}

export interface HealthResponse {
	status: string;
	version: string;
	time: string;
}
