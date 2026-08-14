import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { ArrowDown, ArrowUp, Beaker, ChevronDown, ChevronsUp, Edit3, FileInput, Plus, Search, Trash2 } from "lucide-react"
import { useMemo, useState, type FormEvent } from "react"
import { toast } from "sonner"

import { ConfirmAction } from "@/components/confirm-action"
import { FormDialog } from "@/components/form-dialog"
import { PageHeader } from "@/components/page-header"
import { ResourceState } from "@/components/resource-state"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Field, FieldDescription, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Textarea } from "@/components/ui/textarea"
import { api } from "@/lib/api"
import { draftDependencies, queryKeys } from "@/lib/query-keys"
import type { ListResponse, ProxyGroup, Rule } from "@/lib/types"

const types = ["DOMAIN", "DOMAIN-SUFFIX", "DOMAIN-KEYWORD", "GEOSITE", "GEOIP", "IP-CIDR", "IP-CIDR6", "MATCH"]
const typeLabels: Record<string, string> = { DOMAIN: "完整域名", "DOMAIN-SUFFIX": "域名后缀", "DOMAIN-KEYWORD": "域名关键词", GEOSITE: "GeoSite", GEOIP: "GeoIP", "IP-CIDR": "IPv4 网段", "IP-CIDR6": "IPv6 网段", MATCH: "最终匹配" }
const categoryOrder = ["系统分类", "海外AI", "国内AI", "流媒体", "影视元数据", "电报", "开发", "Google", "社交", "其它", "PT站"]
type RuleForm = Omit<Rule, "id"> & { id?: number }
type MatchResult = { input: string; host: string; kind: string; matched: boolean; fallbackMatch: boolean; rule?: { type: string; payload?: string; target: string }; skipped: number; note: string; resolveError?: string }
type BatchAction = { ids: number[] } & ({ kind: "target"; target: string } | { kind: "category"; category: string } | { kind: "enabled"; enabled: boolean } | { kind: "delete" })
type BatchResult = { updated: number } | { deleted: number }

function isSystemRule(rule: Pick<Rule, "type" | "payload">) {
  return rule.type === "MATCH" || (rule.type === "GEOIP" && rule.payload.toUpperCase() === "CN") || (rule.type === "GEOSITE" && rule.payload === "category-ads-all")
}

function ordered(rules: Rule[]) { return [...rules].sort((a, b) => a.sortOrder - b.sortOrder || a.id - b.id) }

export function RulesPage() {
  const client = useQueryClient()
  const [editing, setEditing] = useState<RuleForm | null>(null)
  const [importOpen, setImportOpen] = useState(false)
  const [matchOpen, setMatchOpen] = useState(false)
  const [search, setSearch] = useState("")
  const [mode, setMode] = useState("category")
  const [selected, setSelected] = useState<Set<number>>(new Set())
  const [batchTarget, setBatchTarget] = useState("")
  const [batchCategory, setBatchCategory] = useState("")
  const [expanded, setExpanded] = useState<Set<string>>(new Set())
  const [importForm, setImportForm] = useState({ text: "", defaultType: "DOMAIN-SUFFIX", defaultTarget: "", defaultCategory: "", defaultNote: "", enabled: true })
  const [matchInput, setMatchInput] = useState("")
  const [matchResult, setMatchResult] = useState<MatchResult | null>(null)
  const rules = useQuery({ queryKey: queryKeys.rules, queryFn: () => api.get<ListResponse<Rule>>("/rules") })
  const groups = useQuery({ queryKey: queryKeys.groups, queryFn: () => api.get<ListResponse<ProxyGroup>>("/groups") })
  const invalidate = async () => { await client.invalidateQueries({ queryKey: queryKeys.rules }); for (const key of draftDependencies) await client.invalidateQueries({ queryKey: key }) }
  const save = useMutation({ mutationFn: (form: RuleForm) => { const body = { type: form.type, payload: form.type === "MATCH" ? "" : form.payload.trim(), target: form.target, enabled: form.enabled, note: form.note ?? "", category: form.category ?? "", sortOrder: form.sortOrder }; return form.id ? api.put<Rule>(`/rules/${form.id}`, body) : api.post<Rule>("/rules", body) }, onSuccess: async () => { toast.success("规则已保存"); setEditing(null); await invalidate() }, onError: (e: Error) => toast.error(e.message) })
  const remove = useMutation({ mutationFn: (id: number) => api.delete(`/rules/${id}`), onSuccess: async () => { toast.success("规则已删除"); await invalidate() }, onError: (e: Error) => toast.error(e.message) })
  const reorder = useMutation({ mutationFn: (items: Rule[]) => api.post("/rules/reorder", { orderedIds: items.map((item) => item.id) }), onSuccess: invalidate, onError: (e: Error) => { toast.error(e.message); void rules.refetch() } })
  const batch = useMutation<BatchResult, Error, BatchAction>({
    mutationFn: (action) => {
      if (action.kind === "target") return api.put<{ updated: number }>("/rules/batch-target", { ids: action.ids, target: action.target })
      if (action.kind === "category") return api.put<{ updated: number }>("/rules/batch-category", { ids: action.ids, category: action.category })
      if (action.kind === "enabled") return api.put<{ updated: number }>("/rules/batch-enabled", { ids: action.ids, enabled: action.enabled })
      return api.post<{ deleted: number }>("/rules/batch-delete", { ids: action.ids })
    },
    onSuccess: async (result) => { toast.success("deleted" in result ? `已删除 ${result.deleted} 条规则` : `已更新 ${result.updated} 条规则`); setSelected(new Set()); await invalidate() },
    onError: (e: Error) => toast.error(e.message),
  })
  const importRules = useMutation({ mutationFn: () => api.post<{ created: number; skipped: number; errors?: string[] }>("/rules/batch-import", importForm), onSuccess: async (result) => { toast.success(`导入 ${result.created} 条，跳过 ${result.skipped} 条`); if (result.errors?.length) toast.warning(result.errors.slice(0, 3).join("；")); setImportOpen(false); await invalidate() }, onError: (e: Error) => toast.error(e.message) })
  const match = useMutation({ mutationFn: () => api.post<MatchResult>("/rules/match", { input: matchInput, rules: ordered(rules.data?.items ?? []).map((item) => ({ type: item.type, payload: item.payload, target: item.target, enabled: item.enabled })), resolve: (rules.data?.items ?? []).some((item) => item.type === "GEOIP") }), onSuccess: setMatchResult, onError: (e: Error) => toast.error(e.message) })
  const items = ordered(rules.data?.items ?? [])
  const groupNames = (groups.data?.items ?? []).map((item) => item.name)
  const defaultTarget = groupNames.includes("直连") ? "直连" : groupNames.includes("PROXY") ? "PROXY" : groupNames[0] ?? ""
  const filtered = items.filter((item) => [item.payload, item.type, typeLabels[item.type], item.target, item.note, item.category].join("\n").toLowerCase().includes(search.toLowerCase()))
  const sections = useMemo(() => {
    const key = (item: Rule) => mode === "category" ? item.category || "未分类" : item.target || "未指定出口"
    const map = new Map<string, Rule[]>()
    for (const item of filtered) map.set(key(item), [...(map.get(key(item)) ?? []), item])
    const keys = mode === "category" ? [...categoryOrder.filter((item) => map.has(item)), ...[...map.keys()].filter((item) => !categoryOrder.includes(item)).sort()] : [...groupNames.filter((item) => map.has(item)), ...[...map.keys()].filter((item) => !groupNames.includes(item)).sort()]
    return keys.map((label) => ({ label, items: map.get(label) ?? [] }))
  }, [filtered, groupNames, mode])
  const move = (rule: Rule, direction: -1 | 1) => {
    if (isSystemRule(rule)) return toast.error("系统规则不可排序")
    const movable = items.filter((item) => !isSystemRule(item))
    const index = movable.findIndex((item) => item.id === rule.id)
    const target = index + direction
    if (target < 0 || target >= movable.length) return
    ;[movable[index], movable[target]] = [movable[target], movable[index]]
    let cursor = 0
    const next = items.map((item) => isSystemRule(item) ? item : movable[cursor++])
    reorder.mutate(next)
  }
  const moveToSectionTop = (rule: Rule) => {
    if (isSystemRule(rule)) return toast.error("系统规则不可排序")
    const key = (item: Rule) => mode === "category" ? item.category || "未分类" : item.target || "未指定出口"
    const movable = items.filter((item) => !isSystemRule(item))
    const index = movable.findIndex((item) => item.id === rule.id)
    const first = movable.findIndex((item) => key(item) === key(rule))
    if (index <= first || first < 0) return
    const [target] = movable.splice(index, 1)
    movable.splice(first, 0, target)
    let cursor = 0
    reorder.mutate(items.map((item) => isSystemRule(item) ? item : movable[cursor++]))
  }
  const toggleSection = (label: string) => setExpanded((before) => { const next = new Set(before); if (next.has(label)) next.delete(label); else next.add(label); return next })
  const allExpanded = sections.length > 0 && sections.every((section) => expanded.has(section.label))
  const toggleAll = () => setExpanded(allExpanded ? new Set() : new Set(sections.map((section) => section.label)))
  const submit = (event: FormEvent<HTMLFormElement>) => { event.preventDefault(); if (!editing?.target) return toast.error("请选择目标策略组"); if (editing.type !== "MATCH" && !editing.payload.trim()) return toast.error("请填写匹配内容"); save.mutate(editing) }
  return <div className="flex flex-col gap-4 md:gap-6">
    <PageHeader title="分流规则" description="规则按顺序匹配；广告、国内和最终匹配为系统锚点。" actions={<><Button variant="outline" onClick={() => setMatchOpen(true)}><Beaker />测试</Button><Button variant="outline" onClick={() => { setImportForm((f) => ({ ...f, defaultTarget: f.defaultTarget || defaultTarget })); setImportOpen(true) }}><FileInput />批量导入</Button><Button onClick={() => setEditing({ type: "DOMAIN-SUFFIX", payload: "", target: defaultTarget, enabled: true, sortOrder: (items.at(-1)?.sortOrder ?? 0) + 10, note: "", category: "" })}><Plus />新建规则</Button></>} />
    <section className="overflow-hidden rounded-lg border bg-card">
      <div className="flex flex-wrap items-center gap-2 border-b p-3"><div className="relative min-w-56 flex-1"><Search className="absolute left-2.5 top-2 size-4 text-muted-foreground" /><Input className="border-0 bg-muted/50 pl-8 shadow-none focus-visible:bg-background" value={search} onChange={(e) => setSearch(e.target.value)} placeholder="搜索类型、内容、出口、备注或分类" /></div><Tabs value={mode} onValueChange={setMode}><TabsList><TabsTrigger value="category">按分类</TabsTrigger><TabsTrigger value="target">按策略组</TabsTrigger></TabsList></Tabs><Button variant="outline" onClick={toggleAll}><ChevronDown className={allExpanded ? "rotate-180" : ""} />{allExpanded ? "全部折叠" : "全部展开"}</Button></div>
      {selected.size ? <div className="flex flex-wrap items-center gap-2 border-b bg-muted/30 p-2"><span className="px-1 text-sm text-muted-foreground">已选 {selected.size} 条</span><Button size="sm" variant="outline" onClick={() => setSelected(new Set())}>取消选择</Button><Select value={batchTarget} onValueChange={setBatchTarget}><SelectTrigger className="w-40"><SelectValue placeholder="选择出口" /></SelectTrigger><SelectContent>{groupNames.map((name) => <SelectItem key={name} value={name}>{name}</SelectItem>)}</SelectContent></Select><ConfirmAction title="批量修改出口" description={`将 ${selected.size} 条规则的出口改为「${batchTarget}」。`} confirmLabel="修改" disabled={!batchTarget || batch.isPending} onConfirm={() => batch.mutate({ kind: "target", ids: [...selected], target: batchTarget })} trigger={<Button size="sm" disabled={!batchTarget || batch.isPending}>改出口</Button>} /><Input className="w-40" value={batchCategory} onChange={(e) => setBatchCategory(e.target.value)} placeholder="分类，留空为未分类" list="rule-categories" /><ConfirmAction title="批量修改分类" description={`将 ${selected.size} 条规则移至「${batchCategory.trim() || "未分类"}」；系统规则会跳过。`} confirmLabel="修改" disabled={batch.isPending} onConfirm={() => batch.mutate({ kind: "category", ids: [...selected], category: batchCategory })} trigger={<Button size="sm" variant="outline" disabled={batch.isPending}>改分类</Button>} /><Button size="sm" variant="outline" disabled={batch.isPending} onClick={() => batch.mutate({ kind: "enabled", ids: [...selected], enabled: true })}>启用</Button><Button size="sm" variant="outline" disabled={batch.isPending} onClick={() => batch.mutate({ kind: "enabled", ids: [...selected], enabled: false })}>停用</Button><ConfirmAction title="批量删除规则" description={`将删除选中的 ${selected.size} 条业务规则；系统规则会跳过。`} confirmLabel="删除" destructive disabled={batch.isPending} onConfirm={() => batch.mutate({ kind: "delete", ids: [...selected] })} trigger={<Button size="sm" variant="destructive" disabled={batch.isPending}><Trash2 />删除</Button>} /></div> : null}
    <ResourceState pending={rules.isPending || groups.isPending} error={rules.error ?? groups.error} empty={!items.length} onRetry={() => void Promise.all([rules.refetch(), groups.refetch()])}>
      <div className="space-y-3 p-3">{sections.map((section) => {
        const open = expanded.has(section.label)
        return <section key={section.label} className="overflow-hidden rounded-md border bg-background">
          <div className="flex items-center justify-between bg-muted/40 px-3 py-2">
            <button type="button" className="flex min-w-0 flex-1 items-center gap-2 text-left font-medium" aria-expanded={open} onClick={() => toggleSection(section.label)}><ChevronDown className={`size-4 shrink-0 transition-transform ${open ? "" : "-rotate-90"}`} /><span className="truncate">{section.label}</span><span className="text-xs text-muted-foreground">{section.items.length}</span></button>
            <Checkbox checked={section.items.length > 0 && section.items.every((item) => selected.has(item.id))} onCheckedChange={(checked) => setSelected((before) => { const next = new Set(before); for (const item of section.items) { if (checked) next.add(item.id); else next.delete(item.id) } return next })} aria-label={`选择 ${section.label}`} />
          </div>
          {open ? <Table><TableHeader><TableRow><TableHead className="w-10" /><TableHead>规则</TableHead><TableHead>目标</TableHead><TableHead>状态</TableHead><TableHead>备注</TableHead><TableHead className="w-48" /></TableRow></TableHeader><TableBody>{section.items.map((item) => <TableRow key={item.id}>
            <TableCell><Checkbox checked={selected.has(item.id)} onCheckedChange={(checked) => setSelected((before) => { const next = new Set(before); if (checked) next.add(item.id); else next.delete(item.id); return next })} aria-label={`选择规则 ${item.payload || item.type}`} /></TableCell>
            <TableCell><div className="flex max-w-xl items-center gap-2"><Badge variant="outline">{item.type}</Badge><span className="truncate font-mono text-xs" title={item.payload}>{item.type === "MATCH" ? "最终兜底" : item.payload}</span>{isSystemRule(item) ? <Badge variant="secondary">系统</Badge> : null}</div></TableCell>
            <TableCell>{item.target}</TableCell><TableCell><Switch checked={item.enabled} onCheckedChange={(enabled) => save.mutate({ ...item, enabled })} /></TableCell><TableCell className="max-w-48 truncate text-muted-foreground" title={item.note}>{item.note || "-"}</TableCell>
            <TableCell><div className="flex justify-end"><Button variant="ghost" size="icon" disabled={isSystemRule(item) || reorder.isPending || section.items[0]?.id === item.id} onClick={() => moveToSectionTop(item)} aria-label="组内置顶"><ChevronsUp /></Button><Button variant="ghost" size="icon" disabled={isSystemRule(item) || reorder.isPending} onClick={() => move(item, -1)} aria-label="上移"><ArrowUp /></Button><Button variant="ghost" size="icon" disabled={isSystemRule(item) || reorder.isPending} onClick={() => move(item, 1)} aria-label="下移"><ArrowDown /></Button><Button variant="ghost" size="icon" onClick={() => setEditing({ ...item })} aria-label="编辑"><Edit3 /></Button>{!isSystemRule(item) ? <ConfirmAction title="删除规则" description={`将删除 ${item.type},${item.payload}。`} destructive confirmLabel="删除" onConfirm={() => remove.mutate(item.id)} trigger={<Button variant="ghost" size="icon" aria-label="删除"><Trash2 /></Button>} /> : null}</div></TableCell>
          </TableRow>)}</TableBody></Table> : null}
        </section>
      })}</div>
    </ResourceState>
    </section>
    <FormDialog open={!!editing} onOpenChange={(open) => !open && setEditing(null)} title={editing?.id ? "编辑规则" : "新建规则"} busy={save.isPending} onSubmit={submit}>
      {editing ? <><div className="grid gap-4 sm:grid-cols-2"><Field><FieldLabel>规则类型</FieldLabel><Select disabled={!!editing.id && isSystemRule(editing as Rule)} value={editing.type} onValueChange={(type) => setEditing({ ...editing, type, payload: type === "MATCH" ? "" : editing.payload })}><SelectTrigger className="w-full"><SelectValue /></SelectTrigger><SelectContent>{types.map((type) => <SelectItem key={type} value={type}>{typeLabels[type]}</SelectItem>)}</SelectContent></Select></Field><Field><FieldLabel>目标策略组</FieldLabel><Select value={editing.target} onValueChange={(target) => setEditing({ ...editing, target })}><SelectTrigger className="w-full"><SelectValue /></SelectTrigger><SelectContent>{editing.target && !groupNames.includes(editing.target) ? <SelectItem value={editing.target}>{editing.target}（已失效）</SelectItem> : null}{groupNames.map((name) => <SelectItem key={name} value={name}>{name}</SelectItem>)}</SelectContent></Select></Field></div>{editing.type !== "MATCH" ? <Field><FieldLabel htmlFor="rule-payload">{typeLabels[editing.type] ?? "匹配内容"}</FieldLabel><Input id="rule-payload" value={editing.payload} maxLength={512} onChange={(e) => setEditing({ ...editing, payload: e.target.value })} required disabled={!!editing.id && isSystemRule(editing as Rule)} /></Field> : null}<Field><FieldLabel htmlFor="rule-category">业务分类</FieldLabel><Input id="rule-category" value={editing.category ?? ""} maxLength={64} onChange={(e) => setEditing({ ...editing, category: e.target.value })} disabled={!!editing.id && isSystemRule(editing as Rule)} list="rule-categories" /><datalist id="rule-categories">{categoryOrder.map((item) => <option key={item}>{item}</option>)}</datalist></Field><Field><FieldLabel htmlFor="rule-note">备注</FieldLabel><Input id="rule-note" value={editing.note ?? ""} maxLength={255} onChange={(e) => setEditing({ ...editing, note: e.target.value })} /></Field><Field orientation="horizontal"><FieldLabel htmlFor="rule-enabled">启用</FieldLabel><Switch id="rule-enabled" checked={editing.enabled} onCheckedChange={(enabled) => setEditing({ ...editing, enabled })} /></Field></> : null}
    </FormDialog>
    <FormDialog open={importOpen} onOpenChange={setImportOpen} title="批量导入规则" description="每行可写 category,TYPE,payload,target[,note]，或只写 payload 使用默认值。" busy={importRules.isPending} submitLabel="导入" onSubmit={(e) => { e.preventDefault(); if (!importForm.text.trim()) return toast.error("请粘贴规则"); importRules.mutate() }} className="sm:max-w-2xl"><div className="grid gap-4 sm:grid-cols-2"><Field><FieldLabel>默认类型</FieldLabel><Select value={importForm.defaultType} onValueChange={(defaultType) => setImportForm({ ...importForm, defaultType })}><SelectTrigger className="w-full"><SelectValue /></SelectTrigger><SelectContent>{types.filter((type) => type !== "MATCH").map((type) => <SelectItem key={type} value={type}>{type}</SelectItem>)}</SelectContent></Select></Field><Field><FieldLabel>默认目标</FieldLabel><Select value={importForm.defaultTarget} onValueChange={(defaultTarget) => setImportForm({ ...importForm, defaultTarget })}><SelectTrigger className="w-full"><SelectValue /></SelectTrigger><SelectContent>{groupNames.map((name) => <SelectItem key={name} value={name}>{name}</SelectItem>)}</SelectContent></Select></Field></div><div className="grid gap-4 sm:grid-cols-2"><Field><FieldLabel>默认分类</FieldLabel><Input value={importForm.defaultCategory} onChange={(e) => setImportForm({ ...importForm, defaultCategory: e.target.value })} /></Field><Field><FieldLabel>默认备注</FieldLabel><Input value={importForm.defaultNote} onChange={(e) => setImportForm({ ...importForm, defaultNote: e.target.value })} /></Field></div><Field><FieldLabel>规则文本</FieldLabel><Textarea rows={12} value={importForm.text} onChange={(e) => setImportForm({ ...importForm, text: e.target.value })} required /><FieldDescription>空行和 # 注释会忽略；单行最大 4096 字符。</FieldDescription></Field><Field orientation="horizontal"><FieldLabel>导入后启用</FieldLabel><Switch checked={importForm.enabled} onCheckedChange={(enabled) => setImportForm({ ...importForm, enabled })} /></Field></FormDialog>
    <Dialog open={matchOpen} onOpenChange={setMatchOpen}><DialogContent className="sm:max-w-lg"><DialogHeader><DialogTitle>测试规则匹配</DialogTitle><DialogDescription>按当前启用规则从上到下由服务端模拟匹配。</DialogDescription></DialogHeader><form className="flex gap-2" onSubmit={(e) => { e.preventDefault(); match.mutate() }}><Input value={matchInput} onChange={(e) => setMatchInput(e.target.value)} placeholder="chat.openai.com 或 IP" required /><Button type="submit" disabled={match.isPending}>测试</Button></form>{matchResult ? <div className="rounded-md border p-4"><div className="flex items-center gap-2"><Badge variant={matchResult.matched ? "default" : matchResult.fallbackMatch ? "secondary" : "outline"}>{matchResult.matched ? "命中" : matchResult.fallbackMatch ? "兜底" : "未命中"}</Badge><span className="font-medium">{matchResult.rule?.target ?? "-"}</span></div><p className="mt-2 text-sm">{matchResult.note}</p>{matchResult.rule ? <code className="mt-2 block break-all text-xs text-muted-foreground">{matchResult.rule.type},{matchResult.rule.payload},{matchResult.rule.target}</code> : null}{matchResult.resolveError ? <p className="mt-2 text-xs text-destructive">{matchResult.resolveError}</p> : null}</div> : null}</DialogContent></Dialog>
  </div>
}
