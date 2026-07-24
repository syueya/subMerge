import {
	Component,
	EventEmitter,
	Input,
	OnChanges,
	Output,
	SimpleChanges,
	inject,
	signal,
} from '@angular/core';
import { FormsModule } from '@angular/forms';
import { concatMap, finalize } from 'rxjs';
import {
	DEFAULT_EXCLUDE_NAME_REGEX,
	DEFAULT_EXCLUDE_SERVERS,
	FALLBACK_REGION,
	Region,
	RegionCatalogEntry,
	RegionMode,
	SubscriptionSource,
	regionOptionText,
} from '../../common/types';
import { DialogService } from '../../common/dialog/dialog.service';
import { FieldTipComponent } from '../../common/field-tip/field-tip.component';
import { SourceService } from './source.service';
import { formatRefreshMsg } from './source-refresh.util';

@Component({
	selector: 'app-source-form-modal',
	standalone: true,
	imports: [FormsModule, FieldTipComponent],
	templateUrl: './source-form-modal.component.html',
})
export class SourceFormModalComponent implements OnChanges {
	private readonly svc = inject(SourceService);
	private readonly dialog = inject(DialogService);

	@Input({ required: true }) open = false;
	/** null = 新建 */
	@Input() editingSource: SubscriptionSource | null = null;
	@Input() regionCatalog: RegionCatalogEntry[] = [];
	@Input() fallbackRegion: string = FALLBACK_REGION;

	@Output() closed = new EventEmitter<void>();
	/** 保存/创建成功后通知父组件刷新列表 */
	@Output() saved = new EventEmitter<void>();

	busy = signal(false);
	formName = '';
	formRegion: Region = FALLBACK_REGION;
	formUrl = '';
	formEnabled = true;
	formRegionMode: RegionMode = 'auto';
	formExcludeName = DEFAULT_EXCLUDE_NAME_REGEX;
	formExcludeServers = DEFAULT_EXCLUDE_SERVERS;
	formIncludeName = '';
	showAdvanced = signal(false);

	readonly tip = {
		name: '备注名会拼到节点名末尾（如「良心云」→ …-良心云），请起简短可辨认的名字。',
		region:
			'固定模式：从目录选择地区，该源全部节点加此前缀。\n自动识别：回退建议选「未知 (UNKNOWN)」，仅当节点名识别失败时使用。',
		regionMode:
			'自动识别（机场推荐）：从节点名 emoji/关键词推断 JP/HK/US…，混合地区订阅一条即可；回退默认 UNKNOWN。\n固定地区：整源强制同一前缀（单地区订阅更合适）。',
		url: '上游 Clash 订阅地址。加密存储，界面只显示脱敏结果。编辑时留空表示不修改。',
		excludeName:
			'名称匹配此正则的节点会被丢弃（不区分大小写）。\n默认含：剩余流量、套餐到期、过滤掉… 等机场信息节点。\n拉取成功后弹窗会列出被过滤的名称。',
		excludeServers: 'server 黑名单，逗号/换行分隔。如 127.0.0.1,localhost',
		includeName: '可选。非空时只保留名称匹配的节点（白名单）。',
	};

	get editingId(): number | null {
		return this.editingSource?.id ?? null;
	}

	/** 固定模式下拉：不含 UNKNOWN */
	fixedRegionOptions(): RegionCatalogEntry[] {
		return this.regionCatalog.filter((r) => String(r.code).toUpperCase() !== FALLBACK_REGION);
	}

	/** 自动模式回退下拉：含 UNKNOWN，并置顶 */
	autoRegionOptions(): RegionCatalogEntry[] {
		const list = [...this.regionCatalog];
		list.sort((a, b) => {
			const au = String(a.code).toUpperCase() === FALLBACK_REGION ? 0 : 1;
			const bu = String(b.code).toUpperCase() === FALLBACK_REGION ? 0 : 1;
			if (au !== bu) return au - bu;
			return String(a.code).localeCompare(String(b.code));
		});
		return list;
	}

	ngOnChanges(changes: SimpleChanges): void {
		if (changes['open'] && this.open) {
			this.hydrateFromInputs();
		}
	}

	private hydrateFromInputs(): void {
		const item = this.editingSource;
		if (item) {
			this.formName = item.name;
			this.formRegionMode = item.regionMode || 'auto';
			this.formRegion =
				item.region ||
				(this.formRegionMode === 'auto'
					? this.fallbackRegion
					: this.fixedRegionOptions()[0]?.code || 'US');
			this.formUrl = '';
			this.formEnabled = item.enabled;
			this.formExcludeName = item.excludeNameRegex ?? DEFAULT_EXCLUDE_NAME_REGEX;
			this.formExcludeServers = item.excludeServers ?? DEFAULT_EXCLUDE_SERVERS;
			this.formIncludeName = item.includeNameRegex ?? '';
			this.showAdvanced.set(false);
			this.busy.set(false);
			return;
		}
		this.formName = '';
		this.formRegionMode = 'auto';
		this.formRegion = this.fallbackRegion;
		this.formUrl = '';
		this.formEnabled = true;
		this.formExcludeName = DEFAULT_EXCLUDE_NAME_REGEX;
		this.formExcludeServers = DEFAULT_EXCLUDE_SERVERS;
		this.formIncludeName = '';
		this.showAdvanced.set(false);
		this.busy.set(false);
	}

	regionOptionLabel(r: RegionCatalogEntry): string {
		return regionOptionText(r.code, r.name);
	}

	onRegionModeChange(mode: RegionMode): void {
		this.formRegionMode = mode;
		if (mode === 'auto') {
			// 自动模式默认回退 UNKNOWN
			if (!this.formRegion || this.formRegion === 'US') {
				this.formRegion = this.fallbackRegion;
			}
		} else if (mode === 'fixed') {
			// 固定模式不要默认 UNKNOWN
			if (!this.formRegion || String(this.formRegion).toUpperCase() === FALLBACK_REGION) {
				const first = this.fixedRegionOptions()[0];
				this.formRegion = first?.code || 'US';
			}
		}
	}

	close(): void {
		if (this.busy()) return;
		this.closed.emit();
	}

	save(): void {
		const region = String(this.formRegion || '')
			.trim()
			.toUpperCase();
		if (!this.formName.trim()) {
			void this.dialog.error('请填写名称');
			return;
		}
		if (!this.editingId && !this.formUrl.trim()) {
			void this.dialog.error('请填写订阅 URL');
			return;
		}
		if (!/^[A-Z0-9]{1,16}$/.test(region)) {
			void this.dialog.error('请选择有效地区（如 UNKNOWN、US、JP、HK）');
			return;
		}
		if (this.formRegionMode === 'fixed' && region === FALLBACK_REGION) {
			void this.dialog.error('固定地区模式请选择具体国家/地区，不要用 UNKNOWN');
			return;
		}

		const body = {
			name: this.formName.trim(),
			region,
			enabled: this.formEnabled,
			regionMode: this.formRegionMode,
			excludeNameRegex: this.formExcludeName,
			excludeServers: this.formExcludeServers,
			includeNameRegex: this.formIncludeName,
			...(this.formUrl.trim() ? { url: this.formUrl.trim() } : {}),
		};

		this.busy.set(true);
		if (this.editingId == null) {
			this.svc
				.create({ ...body, url: this.formUrl.trim() })
				.pipe(
					concatMap((source) => this.svc.refresh(source.id)),
					finalize(() => this.busy.set(false)),
				)
				.subscribe({
					next: (res) => {
						void this.dialog.success(formatRefreshMsg(res, '已创建并拉取完成'));
						this.saved.emit();
					},
					error: (err: Error) => {
						// 源可能已入库但拉取失败
						void this.dialog.error(
							`自动拉取失败：${err.message}\n若列表中已出现该源，可稍后点「拉取」重试。`,
						);
						this.saved.emit();
					},
				});
			return;
		}

		this.svc
			.update(this.editingId, body)
			.pipe(finalize(() => this.busy.set(false)))
			.subscribe({
				next: (src) => {
					void this.dialog.success(`已更新「${src.name}」`);
					this.saved.emit();
				},
				error: (err: Error) => void this.dialog.error(err.message),
			});
	}
}
