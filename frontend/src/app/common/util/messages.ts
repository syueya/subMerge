/**
 * 通用操作提示文案，统一管理避免重复。
 */

// --- 验证提示 ---
export const MSG_NAME_REQUIRED = '请填写名称';

// --- 复制提示 ---
export const MSG_COPIED = '已复制到剪贴板';
export const MSG_COPY_FAILED = '复制失败，请手动选择文本';

// --- 操作确认 ---
export const TITLE_CONFIRM_DELETE = '删除确认';
export const TITLE_CONFIRM_REVOKE = '作废确认';
export const TEXT_CONFIRM_DELETE = '删除';

// --- 操作成功 ---
export function msgDeleted(name: string): string { return `已删除「${name}」`; }
export function msgDisabled(name: string): string { return `已禁用「${name}」`; }
export function msgEnabled(name: string): string { return `已启用「${name}」`; }
export function msgRevoked(name: string): string { return `已作废「${name}」`; }

// --- 额外提示 ---
export const MSG_DELETE_IRREVERSIBLE = '删除后不可恢复。';