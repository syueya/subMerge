import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Database, Globe2, ListFilter, Plus, RefreshCw, Search, Server, ShieldCheck } from "lucide-react"
import { useDeferredValue, useState, type FormEvent } from "react"
import { toast } from "sonner"

import { ConfirmAction } from "@/components/confirm-action"
import { FormDialog } from "@/components/form-dialog"
import { PageHeader } from "@/components/page-header"
import { ResourceState } from "@/components/resource-state"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Checkbox } from "@/components/ui/checkbox"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Field, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { api } from "@/lib/api"
import { draftDependencies, queryKeys } from "@/lib/query-keys"
import type { GeoCategories, GeoQueryResult, GeoResourceStatus, IPGeoResult, ListResponse, ProxyGroup, Rule } from "@/lib/types"
import { formatDateTime } from "@/lib/types"

type GeoUpdateResponse = { items: Array<{ name: string; updated: boolean; error?: string }> }
type GeoEntry = { type: string; value: string; detail?: string }
type GeoEntriesResponse = { file: string; category?: string; field?: string; keyword?: string; items?: GeoEntry[]; total: number; limit: number; offset: number; message?: string }
type GeoIPDetails = IPGeoResult & { flag?: { img?: string; emoji?: string }; continent?: string; latitude?: number; longitude?: number }
type EntryContext = { title: string; file: string; category?: string; field?: string; keyword?: string; offset: number; limit: number; addRule?: { type: "GEOSITE" | "GEOIP"; payload: string } }
type RuleDraft = { type: "GEOSITE" | "GEOIP"; payload: string; target: string; category: string; note: string }

const emptyCategories: GeoCategories = { geosite: [], geoip: [], metadb: { file: "geoip.metadb", supportsReverse: false }, asn: { file: "GeoLite2-ASN.mmdb", supportsReverse: false } }
const resourceNames: Record<string, string> = { "geoip.dat": "GeoIP", "geosite.dat": "GeoSite", "geoip.metadb": "MetaDB", "GeoLite2-ASN.mmdb": "ASN" }

function formatBytes(value: number) {
  if (!value) return "-"
  const units = ["B", "KB", "MB", "GB"]
  let size = value
  let unit = 0
  while (size >= 1024 && unit < units.length - 1) { size /= 1024; unit += 1 }
  return `${size >= 10 || unit === 0 ? size.toFixed(0) : size.toFixed(1)} ${units[unit]}`
}

function looksLikeIP(value: string) {
  const input = value.trim()
  if (!input || input.includes("/") || /\s/.test(input)) return false
  if (/^\d{1,3}(\.\d{1,3}){3}$/.test(input)) return input.split(".").every((part) => Number(part) >= 0 && Number(part) <= 255)
  return input.includes(":") && /^[0-9a-fA-F:.%]+$/.test(input)
}

export function GeoPage() {
  const client = useQueryClient()
  const [queryText, setQueryText] = useState("")
  const [resolveDNS, setResolveDNS] = useState(true)
  const [result, setResult] = useState<GeoQueryResult | null>(null)
  const [ipDetails, setIPDetails] = useState<Record<string, GeoIPDetails>>({})
  const [ipErrors, setIPErrors] = useState<Record<string, string>>({})
  const [entryContext, setEntryContext] = useState<EntryContext | null>(null)
  const [ruleDraft, setRuleDraft] = useState<RuleDraft | null>(null)

  const status = useQuery({ queryKey: [...queryKeys.geo, "status"], queryFn: () => api.get<GeoResourceStatus[]>("/geo/status") })
  const categories = useQuery({ queryKey: [...queryKeys.geo, "categories"], queryFn: () => api.get<GeoCategories>("/geo/categories") })
  const groups = useQuery({ queryKey: queryKeys.groups, queryFn: () => api.get<ListResponse<ProxyGroup>>("/groups"), enabled: !!ruleDraft })
  const query = useMutation({
    mutationFn: ({ domain, resolve }: { domain: string; resolve: boolean }) => api.post<GeoQueryResult>("/geo/query", { domain, resolve }),
    onSuccess: (data) => { setResult(data); setIPDetails({}); setIPErrors({}) },
    onError: (error: Error) => toast.error(error.message),
  })
  const ipGeo = useMutation({
    mutationFn: (ip: string) => api.post<GeoIPDetails>("/geo/ip-geo", { ip }),
    onSuccess: (data, ip) => { setIPDetails((before) => ({ ...before, [ip]: data })); setIPErrors((before) => { const next = { ...before }; delete next[ip]; return next }) },
    onError: (error: Error, ip) => setIPErrors((before) => ({ ...before, [ip]: error.message })),
  })
  const update = useMutation({
    mutationFn: () => api.post<GeoUpdateResponse>("/geo/update", undefined, { timeoutMs: 3 * 60_000 }),
    onSuccess: async (data) => {
      const ok = data.items.filter((item) => item.updated).length
      toast[data.items.some((item) => !item.updated) ? "error" : "success"](`Geo 数据更新完成：成功 ${ok}/${data.items.length}`)
      setResult(null); setEntryContext(null)
      await client.invalidateQueries({ queryKey: queryKeys.geo })
    },
    onError: (error: Error) => toast.error(error.message),
  })
  const saveRule = useMutation({
    mutationFn: (draft: RuleDraft) => api.post<Rule>("/rules", { ...draft, enabled: true }),
    onSuccess: async () => { toast.success("规则已保存（草稿，需发布后生效）"); setRuleDraft(null); await client.invalidateQueries({ queryKey: queryKeys.rules }); for (const key of draftDependencies) await client.invalidateQueries({ queryKey: key }) },
    onError: (error: Error) => toast.error(error.message),
  })

  const submitQuery = (event: FormEvent) => {
    event.preventDefault()
    const domain = queryText.trim()
    if (!domain) return toast.error("请输入域名或 IP")
    query.mutate({ domain, resolve: looksLikeIP(domain) ? false : resolveDNS })
  }

  const startRule = (context: { type: "GEOSITE" | "GEOIP"; payload: string }) => {
    const names = groups.data?.items.map((group) => group.name) ?? []
    const target = names.includes("直连") ? "直连" : names.includes("PROXY") ? "PROXY" : names[0] ?? ""
    setRuleDraft({ ...context, target, category: "GEO", note: "" })
    setEntryContext(null)
  }

  return <div className="flex flex-col gap-4 md:gap-6">
    <PageHeader title="Geo 数据" description="查询域名分类、IP 归属和本地 Geo 数据文件。" actions={<>
      <Button variant="outline" size="icon" aria-label="刷新元数据" onClick={() => void client.invalidateQueries({ queryKey: queryKeys.geo })}><RefreshCw className={status.isFetching || categories.isFetching ? "animate-spin" : ""} /></Button>
      <ConfirmAction title="更新 Geo 数据" description="将从系统设置中的上游地址下载并覆盖本地 Geo 数据文件。" confirmLabel="开始更新" onConfirm={() => update.mutate()} disabled={update.isPending} trigger={<Button disabled={update.isPending}><Database className={update.isPending ? "animate-pulse" : ""} />{update.isPending ? "更新中..." : "更新数据"}</Button>} />
    </>} />

    <ResourceState pending={status.isPending} error={status.error} empty={!status.data?.length} onRetry={() => void status.refetch()}>
      <section className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">{status.data?.map((item) => <Card key={item.name}>
        <CardHeader className="flex-row items-center justify-between pb-2"><CardTitle className="text-sm">{resourceNames[item.name] || item.name}</CardTitle><Badge variant={item.available ? "secondary" : "destructive"}>{item.available ? "可用" : "不可用"}</Badge></CardHeader>
        <CardContent className="space-y-1 text-xs text-muted-foreground"><p>{item.name} · {formatBytes(item.size)}</p><p>更新于 {formatDateTime(item.modifiedAt)}</p><p className="truncate" title={[item.databaseType, item.version, item.sha256].filter(Boolean).join("\n")}>{item.databaseType || item.version || "版本 -"}</p>{item.error ? <p className="truncate text-destructive" title={item.error}>{item.error}</p> : null}</CardContent>
      </Card>)}</section>
    </ResourceState>

    <section className="overflow-hidden rounded-lg border bg-card"><Tabs defaultValue="query">
      <div className="border-b px-4 pt-3"><TabsList variant="line"><TabsTrigger value="query"><Globe2 />域名 / IP</TabsTrigger><TabsTrigger value="categories"><ListFilter />分类搜索</TabsTrigger></TabsList></div>
      <TabsContent value="query" className="p-4"><DomainQuery queryText={queryText} setQueryText={setQueryText} resolveDNS={resolveDNS} setResolveDNS={setResolveDNS} result={result} pending={query.isPending} submit={submitQuery} ipDetails={ipDetails} ipErrors={ipErrors} ipPending={ipGeo.isPending} lookup={(ip) => ipGeo.mutate(ip)} openEntries={(category) => setEntryContext({ title: `GeoSite · ${category}`, file: "geosite", category, offset: 0, limit: 50, addRule: { type: "GEOSITE", payload: category } })} /></TabsContent>
      <TabsContent value="categories" className="p-4"><CategorySearch categories={categories.data ?? emptyCategories} pending={categories.isPending} error={categories.error} openEntries={setEntryContext} /></TabsContent>
    </Tabs></section>

    <EntriesDialog context={entryContext} onOpenChange={(open) => !open && setEntryContext(null)} onContextChange={setEntryContext} onAddRule={startRule} />
    <FormDialog open={!!ruleDraft} onOpenChange={(open) => !open && setRuleDraft(null)} title="添加 Geo 规则" description="规则将进入草稿，发布后生效。" busy={saveRule.isPending} submitLabel="添加规则" onSubmit={(event) => { event.preventDefault(); if (!ruleDraft?.target) return toast.error("请先创建可用策略组"); saveRule.mutate(ruleDraft) }}>
      {ruleDraft ? <><div className="grid gap-3 sm:grid-cols-2"><Field><FieldLabel>规则类型</FieldLabel><Input value={ruleDraft.type} disabled /></Field><Field><FieldLabel>匹配内容</FieldLabel><Input value={ruleDraft.payload} disabled /></Field></div><Field><FieldLabel>目标出口</FieldLabel><Select value={ruleDraft.target} onValueChange={(target) => setRuleDraft({ ...ruleDraft, target })}><SelectTrigger className="w-full"><SelectValue placeholder="请选择策略组" /></SelectTrigger><SelectContent>{groups.data?.items.map((group) => <SelectItem key={group.id} value={group.name}>{group.name}{group.enabled ? "" : "（已禁用）"}</SelectItem>)}</SelectContent></Select></Field><div className="grid gap-3 sm:grid-cols-2"><Field><FieldLabel>业务分类</FieldLabel><Input value={ruleDraft.category} maxLength={64} onChange={(event) => setRuleDraft({ ...ruleDraft, category: event.target.value })} /></Field><Field><FieldLabel>备注</FieldLabel><Input value={ruleDraft.note} maxLength={255} onChange={(event) => setRuleDraft({ ...ruleDraft, note: event.target.value })} /></Field></div></> : null}
    </FormDialog>
  </div>
}

function DomainQuery({ queryText, setQueryText, resolveDNS, setResolveDNS, result, pending, submit, ipDetails, ipErrors, ipPending, lookup, openEntries }: { queryText: string; setQueryText: (value: string) => void; resolveDNS: boolean; setResolveDNS: (value: boolean) => void; result: GeoQueryResult | null; pending: boolean; submit: (event: FormEvent) => void; ipDetails: Record<string, GeoIPDetails>; ipErrors: Record<string, string>; ipPending: boolean; lookup: (ip: string) => void; openEntries: (category: string) => void }) {
  return <div className="mt-4 space-y-4"><form className="flex flex-col gap-2 sm:flex-row" onSubmit={submit}><Input value={queryText} onChange={(event) => setQueryText(event.target.value)} placeholder="www.example.com 或 1.1.1.1" className="flex-1" /><Button type="submit" disabled={pending}><Search className={pending ? "animate-pulse" : ""} />{pending ? "查询中..." : "查询"}</Button>{!looksLikeIP(queryText) ? <label className="flex items-center gap-2 px-1 text-sm text-muted-foreground"><Checkbox checked={resolveDNS} onCheckedChange={(checked) => setResolveDNS(checked === true)} />DNS 解析</label> : null}</form>
    {!result ? <div className="grid min-h-48 place-items-center border-y text-sm text-muted-foreground">输入域名或 IP 开始查询</div> : <>
      <div className="flex flex-wrap items-center gap-2">{result.resolveError ? <span className="text-sm text-destructive">DNS：{result.resolveError}</span> : result.resolveSkipped && result.inputType !== "ip" ? <span className="text-sm text-muted-foreground">未进行 DNS 解析</span> : null}{result.ips.map((ip) => <div key={ip} className="flex items-center gap-1"><Badge variant="outline">{ip}</Badge><Button variant="ghost" size="icon-sm" aria-label={`查询 ${ip} 归属`} disabled={ipPending} onClick={() => lookup(ip)}><Globe2 /></Button>{ipDetails[ip] ? <span className="text-xs text-muted-foreground">{ipDetails[ip].flag?.emoji} {ipDetails[ip].country} · {ipDetails[ip].region} · {ipDetails[ip].city}</span> : null}{ipErrors[ip] ? <span className="text-xs text-destructive" title={ipErrors[ip]}>查询失败</span> : null}</div>)}</div>
      <div className="grid gap-3 lg:grid-cols-2"><ResultPanel icon={Globe2} title="GeoSite" count={result.geosite.length} empty={result.inputType === "ip" ? "IP 查询不涉及 GeoSite" : "无命中"}>{result.geosite.map((hit) => <div key={`${hit.category}:${hit.type}:${hit.value}`} className="flex items-start justify-between gap-3 border-b py-2 last:border-0"><div className="min-w-0"><p className="font-medium">{hit.category}</p><p className="truncate text-xs text-muted-foreground">{hit.type} · {hit.value}</p></div><Button size="sm" variant="outline" onClick={() => openEntries(hit.category)}>查看条目</Button></div>)}</ResultPanel><ResultPanel icon={ShieldCheck} title="GeoIP" count={result.geoip.length}>{result.geoip.map((hit) => <div key={`${hit.ip}:${hit.cidr}`} className="border-b py-2 last:border-0"><p className="font-medium">{hit.category}</p><p className="text-xs text-muted-foreground">{hit.ip} · {hit.cidr}</p></div>)}</ResultPanel><ResultPanel icon={Database} title="MetaDB" count={result.metadb.length}>{result.metadb.map((hit) => <div key={`${hit.ip}:${hit.cidr}`} className="border-b py-2 last:border-0"><p className="font-medium">{hit.ip}</p><p className="text-xs text-muted-foreground">{hit.cidr} · {hit.record}</p></div>)}</ResultPanel><ResultPanel icon={Server} title="ASN" count={result.asn.length}>{result.asn.map((hit) => <div key={`${hit.ip}:${hit.cidr}`} className="border-b py-2 last:border-0"><p className="font-medium">AS{hit.asn}</p><p className="text-xs text-muted-foreground">{hit.ip} · {hit.cidr} · {hit.organization}</p></div>)}</ResultPanel></div>
    </>}
  </div>
}

function ResultPanel({ icon: Icon, title, count, empty = "无记录或文件不可用", children }: { icon: typeof Globe2; title: string; count: number; empty?: string; children: React.ReactNode }) {
  return <section className="rounded-md border p-3"><h2 className="mb-2 flex items-center gap-2 text-sm font-medium"><Icon className="size-4 text-muted-foreground" />{title}<Badge variant="secondary">{count}</Badge></h2>{count ? children : <p className="py-5 text-center text-sm text-muted-foreground">{empty}</p>}</section>
}

function CategorySearch({ categories, pending, error, openEntries }: { categories: GeoCategories; pending: boolean; error: Error | null; openEntries: (context: EntryContext) => void }) {
  const [file, setFile] = useState("geosite")
  const [keyword, setKeyword] = useState("")
  const [selected, setSelected] = useState("")
  const [field, setField] = useState("asn")
  const deferred = useDeferredValue(keyword.trim().toLowerCase())
  const list = file === "geoip" ? categories.geoip : categories.geosite
  const matches = list.filter((item) => !deferred || item.name.toLowerCase().includes(deferred))
  const reverseMode = file === "geosite" || file === "geoip"
  const changeFile = (value: string) => { setFile(value); setKeyword(""); setSelected(""); setField(value === "metadb" ? "code" : "asn") }
  return <div className="mt-4 space-y-4"><div className="grid gap-3 md:grid-cols-[12rem_1fr_auto]"><Select value={file} onValueChange={changeFile}><SelectTrigger className="w-full"><SelectValue /></SelectTrigger><SelectContent><SelectItem value="geosite">GeoSite</SelectItem><SelectItem value="geoip">GeoIP</SelectItem><SelectItem value="metadb">MetaDB</SelectItem><SelectItem value="asn">ASN</SelectItem></SelectContent></Select>{reverseMode ? <Input value={keyword} onChange={(event) => setKeyword(event.target.value)} placeholder="筛选分类，如 google、ads、cn" /> : <div className="grid gap-2 sm:grid-cols-[10rem_1fr]"><Select value={field} onValueChange={setField}><SelectTrigger className="w-full"><SelectValue /></SelectTrigger><SelectContent>{file === "asn" ? <><SelectItem value="asn">ASN 编号</SelectItem><SelectItem value="organization">组织名</SelectItem></> : <SelectItem value="code">国家/地区代码</SelectItem>}</SelectContent></Select><Input value={keyword} onChange={(event) => setKeyword(event.target.value)} placeholder={file === "asn" ? "15169 或 Google" : "CN、US"} /></div>}<Button disabled={pending || (reverseMode ? !selected : !keyword.trim())} onClick={() => openEntries(reverseMode ? { title: `${file === "geosite" ? "GeoSite" : "GeoIP"} · ${selected}`, file, category: selected, offset: 0, limit: 50, addRule: { type: file === "geosite" ? "GEOSITE" : "GEOIP", payload: selected } } : { title: `${file === "asn" ? "ASN" : "MetaDB"} · ${keyword.trim()}`, file, field, keyword: keyword.trim(), offset: 0, limit: 50 })}><Search />{reverseMode ? "查看条目" : "搜索记录"}</Button></div>
    <ResourceState pending={pending} error={error} empty={reverseMode && !list.length}>{reverseMode ? <><div className="flex max-h-56 flex-wrap content-start gap-2 overflow-y-auto rounded-md border p-3">{matches.slice(0, 30).map((item) => <Button key={item.name} variant={selected === item.name ? "default" : "outline"} size="sm" onClick={() => setSelected(item.name)}>{item.name}<span className="text-xs opacity-70">{item.count}</span></Button>)}</div><p className="text-xs text-muted-foreground">匹配 {matches.length} / 共 {list.length} 个分类{matches.length > 30 ? "，仅显示前 30 个" : ""}</p></> : <div className="grid min-h-40 place-items-center border-y text-sm text-muted-foreground">输入关键词搜索 {file === "asn" ? "ASN" : "MetaDB"} 记录</div>}</ResourceState>
  </div>
}

function EntriesDialog({ context, onOpenChange, onContextChange, onAddRule }: { context: EntryContext | null; onOpenChange: (open: boolean) => void; onContextChange: (context: EntryContext) => void; onAddRule: (context: { type: "GEOSITE" | "GEOIP"; payload: string }) => void }) {
  const entries = useQuery({ queryKey: [...queryKeys.geo, "entries", context], queryFn: () => context?.category ? api.post<GeoEntriesResponse>("/geo/reverse", { file: context.file, category: context.category, limit: context.limit, offset: context.offset }) : api.post<GeoEntriesResponse>("/geo/search", { file: context!.file, field: context!.field, keyword: context!.keyword, limit: context!.limit, offset: context!.offset }), enabled: !!context })
  const data = entries.data
  const page = context ? Math.floor(context.offset / context.limit) + 1 : 1
  const pages = context && data ? Math.max(1, Math.ceil(data.total / context.limit)) : 1
  return <Dialog open={!!context} onOpenChange={onOpenChange}><DialogContent className="sm:max-w-4xl"><DialogHeader><DialogTitle>{context?.title}</DialogTitle><DialogDescription>{data?.message || (data ? `共 ${data.total} 条` : "正在读取条目")}</DialogDescription></DialogHeader><ResourceState pending={entries.isPending} error={entries.error} empty={!data?.items?.length} onRetry={() => void entries.refetch()}><div className="max-h-[55vh] overflow-auto rounded-md border"><Table><TableHeader><TableRow><TableHead>类型</TableHead><TableHead>值</TableHead><TableHead>详情</TableHead></TableRow></TableHeader><TableBody>{data?.items?.map((item, index) => <TableRow key={`${item.type}:${item.value}:${index}`}><TableCell><Badge variant="outline">{item.type}</Badge></TableCell><TableCell className="break-all font-mono text-xs">{item.value}</TableCell><TableCell className="text-xs text-muted-foreground">{item.detail || "-"}</TableCell></TableRow>)}</TableBody></Table></div></ResourceState><DialogFooter className="items-center sm:justify-between"><div className="flex items-center gap-2"><Button variant="outline" size="sm" disabled={!context || page <= 1 || entries.isFetching} onClick={() => context && onContextChange({ ...context, offset: Math.max(0, context.offset - context.limit) })}>上一页</Button><span className="text-xs text-muted-foreground">{page} / {pages}</span><Button variant="outline" size="sm" disabled={!context || page >= pages || entries.isFetching} onClick={() => context && onContextChange({ ...context, offset: context.offset + context.limit })}>下一页</Button></div>{context?.addRule ? <Button disabled={!data?.items?.length} onClick={() => onAddRule(context.addRule!)}><Plus />添加规则</Button> : null}</DialogFooter></DialogContent></Dialog>
}
