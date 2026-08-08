/** 带中文文案的枚举项 */
export interface EnumOption<V extends string = string> {
	value: V;
	text: string;
}

export function enumText(
	options: readonly EnumOption[],
	value: string | null | undefined,
	fallback = value ?? '-',
): string {
	if ((value === null || value === undefined) || value === '') return String(fallback);
	return options.find((o) => o.value === value)?.text ?? String(fallback);
}

/** 状态徽标：复用 assets/scss 工具类，不引入自定义 badge-* */
export const BADGE_OK = 'bg-light-success text-success rounded f-s-12 f-w-600 p-x-8 p-y-2';
export const BADGE_WARN = 'bg-light-warning text-warning rounded f-s-12 f-w-600 p-x-8 p-y-2';
export const BADGE_ERR = 'bg-light-error text-error rounded f-s-12 f-w-600 p-x-8 p-y-2';
export const BADGE_MUTED = 'bg-light-primary text-primary rounded f-s-12 f-w-600 p-x-8 p-y-2';

export function enumBadgeClass(
	map: Record<string, string>,
	value: string | null | undefined,
	fallback = BADGE_MUTED,
): string {
	if (!value) return fallback;
	return map[value] ?? fallback;
}
