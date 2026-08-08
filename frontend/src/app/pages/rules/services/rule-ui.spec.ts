import { Rule } from '@data-struct';

import {
  canMoveRule,
  canMoveRuleGroup,
  canMoveRuleWithinGroup,
  moveRuleGroup,
  moveRuleOrder,
  moveRuleWithinGroup,
} from './rule-ui';

function rule(
  id: number,
  sortOrder: number,
  type = 'DOMAIN',
  category = 'A',
  target = 'G1',
): Rule {
  return {
    id,
    type,
    payload:
      type === 'MATCH'
        ? ''
        : type === 'GEOIP'
          ? 'CN'
          : type === 'GEOSITE'
            ? 'category-ads-all'
            : `rule-${id}.example.com`,
    target,
    enabled: true,
    sortOrder,
    category,
  };
}

describe('rule ordering', () => {
  it('moves business rules up and down without moving system anchors', () => {
    const rules = [rule(1, 10, 'GEOSITE'), rule(2, 20), rule(3, 30), rule(4, 40, 'GEOIP'), rule(5, 50, 'MATCH')];

    expect(moveRuleOrder(rules, 3, 'up').map((item) => item.id)).toEqual([1, 3, 2, 4, 5]);
    expect(moveRuleOrder(rules, 2, 'down').map((item) => item.id)).toEqual([1, 3, 2, 4, 5]);
    expect(moveRuleOrder(rules, 2, 'down').map((item) => item.sortOrder)).toEqual([10, 20, 30, 40, 50]);
  });

  it('places a business rule directly after the ad system rule', () => {
    const rules = [rule(1, 10, 'GEOSITE'), rule(2, 20), rule(3, 30), rule(4, 40, 'GEOIP'), rule(5, 50, 'MATCH')];

    expect(moveRuleOrder(rules, 3, 'top').map((item) => item.id)).toEqual([1, 3, 2, 4, 5]);
  });

  it('moves a rule to the top of its current category or target group', () => {
    const rules = [
      rule(1, 10, 'DOMAIN', 'A', 'G1'),
      rule(2, 20, 'DOMAIN', 'B', 'G2'),
      rule(3, 30, 'DOMAIN', 'A', 'G1'),
      rule(4, 40, 'GEOIP', '系统分类', '直连'),
      rule(5, 50, 'MATCH', '系统分类', '直连'),
    ];

    expect(moveRuleWithinGroup(rules, 3, 'category').map((item) => item.id)).toEqual([3, 1, 2, 4, 5]);
    expect(moveRuleWithinGroup(rules, 3, 'target').map((item) => item.id)).toEqual([3, 1, 2, 4, 5]);
    expect(canMoveRuleWithinGroup(rules, 3, 'category')).toBe(true);
    expect(canMoveRuleWithinGroup(rules, 1, 'category')).toBe(false);
  });

  it('moves an entire category or target group as a block', () => {
    const rules = [
      rule(1, 10, 'DOMAIN', 'A', 'G1'),
      rule(2, 20, 'DOMAIN', 'B', 'G2'),
      rule(3, 30, 'DOMAIN', 'A', 'G1'),
      rule(4, 40, 'GEOIP', '系统分类', '直连'),
      rule(5, 50, 'MATCH', '系统分类', '直连'),
    ];

    expect(moveRuleGroup(rules, 'A', 'category', 'down', ['A', 'B']).map((item) => item.id)).toEqual([2, 1, 3, 4, 5]);
    expect(moveRuleGroup(rules, 'G2', 'target', 'up', ['G1', 'G2']).map((item) => item.id)).toEqual([2, 1, 3, 4, 5]);
    expect(canMoveRuleGroup(rules, 'A', 'category', 'down', ['A', 'B'])).toBe(true);
    expect(canMoveRuleGroup(rules, 'B', 'category', 'down', ['A', 'B'])).toBe(false);
  });
  it('rejects system rules and movement at the business boundaries', () => {
    const rules = [rule(1, 10, 'GEOSITE'), rule(2, 20), rule(3, 30), rule(4, 40, 'GEOIP'), rule(5, 50, 'MATCH')];

    expect(canMoveRule(rules, 1, 'down')).toBe(false);
    expect(canMoveRule(rules, 2, 'up')).toBe(false);
    expect(canMoveRule(rules, 3, 'down')).toBe(false);
    expect(canMoveRule(rules, 4, 'up')).toBe(false);
    expect(canMoveRule(rules, 5, 'top')).toBe(false);
  });
});
