import { describe, expect, it } from 'vitest';
import { enumBadgeClass, enumText } from './enums';

describe('enum helpers', () => {
 it('returns the matching label and badge class', () => {
  const options = [{ value: 'active', text: '启用' }] as const;
  expect(enumText(options, 'active')).toBe('启用');
  expect(enumBadgeClass({ active: 'badge-ok' }, 'active')).toBe('badge-ok');
 });

 it('uses fallbacks for unknown values', () => {
  expect(enumText([], 'unknown')).toBe('unknown');
  expect(enumBadgeClass({}, 'unknown')).toBe('badge-muted');
 });
});
