/**
 * 浏览器侧规则匹配（域名 / IP-CIDR；GEOSITE/GEOIP 会跳过）。
 * 面板「测试规则」主路径已改为 POST /api/rules/match；本文件保留给无后端场景或单测。
 */
import type { MatchableRule, RuleMatchResult } from '@data-struct';

export type { MatchableRule, RuleMatchResult };

export function normalizeHost(raw: string): { host: string; kind: RuleMatchResult['kind'] } {
	let s = raw.trim();
	if (!s) return { host: '', kind: 'empty' };

	if (!/^[a-zA-Z][a-zA-Z0-9+.-]*:\/\//.test(s) && s.includes('/')) {
		s = 'http://' + s;
	}

	try {
		if (/^[a-zA-Z][a-zA-Z0-9+.-]*:\/\//.test(s)) {
			const u = new URL(s);
			s = u.hostname || s;
		}
	} catch {
		/* fall through */
	}

	s = s.split('/')[0].split('?')[0].split('#')[0];
	if (s.startsWith('[')) {
		const end = s.indexOf(']');
		if (end > 0) s = s.slice(1, end);
	} else if (/^\d{1,3}(\.\d{1,3}){3}:\d+$/.test(s)) {
		s = s.replace(/:\d+$/, '');
	} else if (/^[a-zA-Z0-9.-]+:\d+$/.test(s)) {
		s = s.replace(/:\d+$/, '');
	}

	s = s.trim().toLowerCase().replace(/\.$/, '');
	if (!s) return { host: '', kind: 'empty' };

	if (isIPv4(s)) return { host: s, kind: 'ipv4' };
	if (isIPv6(s)) return { host: s, kind: 'ipv6' };
	if (
		!/^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)*$/i.test(s) &&
		s !== 'localhost'
	) {
		if (!/[a-z0-9]/i.test(s)) return { host: s, kind: 'invalid' };
	}
	return { host: s, kind: 'domain' };
}

function isIPv4(s: string): boolean {
	const m = /^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/.exec(s);
	if (!m) return false;
	return m.slice(1).every((p) => {
		const n = Number(p);
		return n >= 0 && n <= 255 && String(n) === String(Number(p));
	});
}

function isIPv6(s: string): boolean {
	return s.includes(':') && /^[0-9a-f:]+$/i.test(s);
}

function matchDomainSuffix(host: string, payload: string): boolean {
	const p = payload.trim().toLowerCase().replace(/^\./, '');
	if (!p) return false;
	return host === p || host.endsWith('.' + p);
}

function matchDomain(host: string, payload: string): boolean {
	return host === payload.trim().toLowerCase().replace(/\.$/, '');
}

function matchDomainKeyword(host: string, payload: string): boolean {
	const p = payload.trim().toLowerCase();
	return !!p && host.includes(p);
}

function ipv4ToInt(ip: string): number | null {
	if (!isIPv4(ip)) return null;
	const [a, b, c, d] = ip.split('.').map(Number);
	return (((a << 24) >>> 0) + (b << 16) + (c << 8) + d) >>> 0;
}

function matchIPv4CIDR(ip: string, cidr: string): boolean {
	const parts = cidr.trim().split('/');
	if (parts.length !== 2) {
		return isIPv4(parts[0]) && ip === parts[0];
	}
	const [base, bitsStr] = parts;
	const bits = Number(bitsStr);
	if (!isIPv4(base) || !Number.isInteger(bits) || bits < 0 || bits > 32) return false;
	const ipN = ipv4ToInt(ip);
	const baseN = ipv4ToInt(base);
	if (ipN === null || baseN === null) return false;
	if (bits === 0) return true;
	const mask = bits === 32 ? 0xffffffff : (~((1 << (32 - bits)) - 1)) >>> 0;
	return (ipN & mask) === (baseN & mask);
}

/**
 * 按规则列表自上而下匹配（仅启用项；顺序即数组顺序）。
 * GEOSITE / GEOIP 在浏览器侧跳过。
 */
export function runRuleMatch(rawInput: string, rules: MatchableRule[]): RuleMatchResult {
	const raw = rawInput;
	const { host, kind } = normalizeHost(raw);
if (kind === 'empty') {
			return {
				input: raw,
				host: '',
				kind,
				matched: false,
				fallbackMatch: false,
				rule: null,
				skipped: 0,
				note: '请输入域名或 URL，例如 chat.openai.com 或 https://www.google.com',
			};
		}
		if (kind === 'invalid') {
			return {
				input: raw,
				host,
				kind,
				matched: false,
				fallbackMatch: false,
				rule: null,
				skipped: 0,
				note: '无法解析输入，请检查格式',
			};
		}

	const list = rules.filter((r) => r.enabled !== false);
	let skipped = 0;

	for (const rule of list) {
		const typ = String(rule.type || '').toUpperCase();
		const payload = rule.payload || '';

if (typ === 'MATCH') {
				// MATCH 是兜底：业务上算「未命中具体规则」，仅落到最终出口
				return {
					input: raw,
					host,
					kind,
					matched: false,
					fallbackMatch: true,
					rule,
					skipped,
					note: '未命中其它规则，落入最终匹配 (MATCH)',
				};
			}

		if (typ === 'GEOIP' || typ === 'GEOSITE') {
			skipped++;
			continue;
		}

		if (kind === 'domain') {
if (typ === 'DOMAIN' && matchDomain(host, payload)) {
					return {
						input: raw,
						host,
						kind,
						matched: true,
						fallbackMatch: false,
						rule,
						skipped,
						note: '命中域名精确匹配',
					};
				}
				if (typ === 'DOMAIN-SUFFIX' && matchDomainSuffix(host, payload)) {
					return {
						input: raw,
						host,
						kind,
						matched: true,
						fallbackMatch: false,
						rule,
						skipped,
						note: '命中域名后缀',
					};
				}
				if (typ === 'DOMAIN-KEYWORD' && matchDomainKeyword(host, payload)) {
					return {
						input: raw,
						host,
						kind,
						matched: true,
						fallbackMatch: false,
						rule,
						skipped,
						note: '命中域名关键词',
					};
				}
			if (typ === 'IP-CIDR' || typ === 'IP-CIDR6') {
				skipped++;
				continue;
			}
			skipped++;
			continue;
		}

		if (kind === 'ipv4') {
if (typ === 'IP-CIDR' && matchIPv4CIDR(host, payload)) {
					return {
						input: raw,
						host,
						kind,
						matched: true,
						fallbackMatch: false,
						rule,
						skipped,
						note: '命中 IPv4 CIDR',
					};
				}
			skipped++;
			continue;
		}

		if (kind === 'ipv6') {
			if (typ === 'IP-CIDR6') {
				const p = payload.trim().toLowerCase().split('/')[0];
				if (p && (host === p || host.startsWith(p.replace(/:$/, '')))) {
return {
							input: raw,
							host,
							kind,
							matched: true,
							fallbackMatch: false,
							rule,
							skipped,
							note: '命中 IPv6 规则（简化匹配，正式以 Clash 为准）',
						};
				}
			}
			skipped++;
			continue;
		}
	}

return {
			input: raw,
			host,
			kind,
			matched: false,
			fallbackMatch: false,
			rule: null,
			skipped,
			note: '没有启用规则命中（请确认是否存在 MATCH 兜底规则）',
		};
	}
