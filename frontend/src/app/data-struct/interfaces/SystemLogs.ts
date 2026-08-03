/** 日志类型列表 */
export interface SystemLogsTypeData {
	files: SystemLogsType[];
}

export interface SystemLogsType {
	name: string; // 类型名称
	size: number; // 大小
	updatedAt: number; // 更新时间
}

/** 日志详情 */
export interface SystemLogsContent {
	items: SystemLogs[]; // 日志内容
}

export interface SystemLogs {
	timestamp: number; // 时间戳
	caller: string; // 调用者
	content: string; // 日志内容
	level: string; // 日志级别

	// 本地
	timestampStr: string; // 时间
	colorClass: string; // 颜色类
}
