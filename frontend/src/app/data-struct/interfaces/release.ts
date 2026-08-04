import type { ReleaseStatus } from '../enums/release';

export interface DraftStatus {
	hasPublished: boolean;
	dirty: boolean;
	publishedHash?: string;
	draftHash?: string;
	publishedVersion?: number;
	buildError?: string;
	/** 草稿相对已发布配置的实体级变更列表（节点/策略组/规则） */
	changes?: DraftChange[];
}

/** 单条草稿变更（相对当前已发布版本） */
export interface DraftChange {
	/** proxy（节点）| group（策略组）| rule（分流规则） */
	kind: 'proxy' | 'group' | 'rule' | string;
	/** added（新增）| removed（删除）| modified（修改） */
	action: 'added' | 'removed' | 'modified' | string;
	/** 变更对象名称（规则用规则内容） */
	name: string;
	/** 变更细节，可空 */
	detail?: string;
}

export interface Release {
	id: number;
	version: number;
	status: ReleaseStatus;
	note?: string;
	proxyCount: number;
	ruleCount: number;
	configHash: string;
	publishedAt?: string | null;
	createdAt: string;
	createdBy: string;
}

/** 发布配置中的单条规则（历史查看 / 匹配测试） */
export interface ReleaseRuleLine {
	type: string;
	payload?: string;
	target: string;
	raw: string;
}

/** 发布版本详情 */
export interface ReleaseDetail extends Release {
	configYaml: string;
	rules: ReleaseRuleLine[];
	groups: string[];
}

export interface ReleasePreview {
	proxyCount: number;
	ruleCount: number;
	groups: string[];
	yamlPreview: string;
	warnings: string[];
}

export interface PublishResponse {
	release: Release;
	preview: ReleasePreview;
}
