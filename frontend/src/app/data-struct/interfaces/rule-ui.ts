import type { Rule } from './rule';

export interface CategoryOption { value: string; text: string }
export interface CategorySection { key: string; label: string; rules: Rule[] }
