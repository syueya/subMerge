import type { Rule } from './rule';

export type CategoryOption = { value: string; text: string };
export type CategorySection = { key: string; label: string; rules: Rule[] };
