import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Check, CloudDownload, Edit3, FilePlus2, MoreHorizontal, Plus, Power, RefreshCw, Search, Trash2 } from "lucide-react"
import { useMemo, useState, type FormEvent } from "react"
import { toast } from "sonner"

import { ConfirmAction } from "@/components/confirm-action"
import { FormDialog } from "@/components/form-dialog"
import { DataPanel } from "@/components/data-panel"
import { PageHeader } from "@/components/page-header"
import { ResourceState } from "@/components/resource-state"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu"
import { Field, FieldDescription, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Textarea } from "@/components/ui/textarea"
import { api } from "@/lib/api"
import { draftDependencies, queryKeys } from "@/lib/query-keys"
import type { ListResponse, ManualSourceImportResult, ProxyNode, RefreshAllResult, RefreshSourceResult, RegionCatalogResponse, SubscriptionSource } from "@/lib/types"
import { formatDateTime } from "@/lib/types"

type SourceForm = {
  id?: number
  kind: "remote" | "manual"
  name: string
  url: string
  content: string
  enabled: boolean
  regionMode: "auto" | "fixed"
  region: string
  excludeNameRegex: string
  excludeServers: string
  includeNameRegex: string
}

const emptyForm: SourceForm = { kind: "remote", name: "", url: "", content: "", enabled: true, regionMode: "auto", region: "UNK", excludeNameRegex: "", excludeServers: "", includeNameRegex: "" }

function sourceForm(item?: SubscriptionSource, kind: SourceForm["kind"] = "remote"): SourceForm {
  if (!item) return { ...emptyForm, kind }
  return { id: item.id, kind: item.kind, name: item.name, url: item.url ?? "", content: item.manualContent ?? "", enabled: item.enabled, regionMode: item.regionMode, region: item.region, excludeNameRegex: item.excludeNameRegex ?? "", excludeServers: item.excludeServers ?? "", includeNameRegex: item.includeNameRegex ?? "" }
}

function refreshSummary(result: RefreshSourceResult | ManualSourceImportResult) {
  const details = [`解析 ${result.parsed}，保留 ${result.kept}，新增 ${result.added}，移除 ${result.removed}，修改 ${result.modified}`]
  const parseDropped = Object.entries(result.parseDropped ?? {}).filter(([, count]) => count > 0).map(([reason, count]) => `${reason} ${count}`).join("、")
  if (parseDropped) details.push(`解析丢弃：${parseDropped}`)
  if ("filterDropped" in result) {
    const filtered = Object.entries(result.filterDropped ?? {}).filter(([, count]) => count > 0).map(([reason, count]) => `${reason} ${count}`).join("、")
    if (filtered) details.push(`规则过滤：${filtered}`)
  }
  const regions = Object.entries(result.regionCounts ?? {}).sort(([left], [right]) => left.localeCompare(right)).map(([region, count]) => `${region} ${count}`).join("、")
  if (regions) details.push(`地区：${regions}`)
  if (result.regionConflictTotal) details.push(`地区标记冲突 ${result.regionConflictTotal} 条，已按名称关键词归类`)
  return details.join("；")
}

function formatBytes(value: number) {
  if (!Number.isFinite(value) || value <= 0) return "0 B"
  const units = ["B", "KB", "MB", "GB", "TB"]
  const index = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1)
  return `${(value / 1024 ** index).toFixed(index === 0 ? 0 : 1)} ${units[index]}`
}

function trafficText(item: SubscriptionSource) {
  const used = Math.max(0, item.trafficUpload) + Math.max(0, item.trafficDownload)
  const usage = item.trafficTotal > 0 ? `${formatBytes(used)} / ${formatBytes(item.trafficTotal)}` : used > 0 ? `已用 ${formatBytes(used)}` : "-"
  if (item.trafficExpire <= 0) return usage
  const expires = new Date(item.trafficExpire * 1000)
  if (Number.isNaN(expires.getTime())) return usage
  return `${usage} · ${expires.getTime() < Date.now() ? "已过期" : "到期"} ${expires.toLocaleDateString("zh-CN")}`
}

export function SourcesPage() {
  const client = useQueryClient()
  const [editing, setEditing] = useState<SourceForm | null>(null)
  const [selected, setSelected] = useState<Set<number>>(new Set())
  const [proxySource, setProxySource] = useState<SubscriptionSource | null>(null)
  const sources = useQuery({ queryKey: queryKeys.sources, queryFn: () => api.get<ListResponse<SubscriptionSource>>("/sources") })
  const regions = useQuery({ queryKey: queryKeys.regions, queryFn: () => api.get<RegionCatalogResponse>("/regions"), staleTime: Infinity })

  const invalidate = async () => {
    await Promise.all([client.invalidateQueries({ queryKey: queryKeys.sources }), ...draftDependencies.map((key) => client.invalidateQueries({ queryKey: key }))])
  }
  const save = useMutation({
    mutationFn: async (form: SourceForm) => {
      const common = { name: form.name.trim(), enabled: form.enabled, regionMode: form.regionMode, region: form.region.toUpperCase() }
      if (form.kind === "manual") {
        const body = { ...common, content: form.content.trim() }
        return form.id ? api.put<ManualSourceImportResult>(`/sources/manual/${form.id}`, body) : api.post<ManualSourceImportResult>("/sources/manual", body)
      }
      const body = { ...common, ...(form.url.trim() ? { url: form.url.trim() } : {}), excludeNameRegex: form.excludeNameRegex, excludeServers: form.excludeServers, includeNameRegex: form.includeNameRegex }
      if (form.id) return api.put<SubscriptionSource>(`/sources/${form.id}`, body)
      const created = await api.post<SubscriptionSource>("/sources", body)
      return api.post<RefreshSourceResult>(`/sources/${created.id}/refresh`)
    },
    onSuccess: async (result) => { toast.success("订阅源已保存", { description: "source" in result ? refreshSummary(result) : undefined }); setEditing(null); await invalidate() },
    onError: (error: Error) => toast.error(error.message),
  })
  const refresh = useMutation({
    mutationFn: (id: number) => api.post<RefreshSourceResult>(`/sources/${id}/refresh`, undefined, { timeoutMs: 10 * 60_000 }),
    onSuccess: async (result) => { toast.success("订阅拉取完成", { description: refreshSummary(result) }); await invalidate(); await client.invalidateQueries({ queryKey: ["proxies"] }) },
    onError: (error: Error) => toast.error(error.message),
  })
  const refreshAll = useMutation({
    mutationFn: () => api.post<RefreshAllResult>("/sources/refresh-all", undefined, { timeoutMs: 10 * 60_000 }),
    onSuccess: async (result) => { const failures = result.results.filter((item) => !item.ok).map((item) => `${item.name}：${item.error || "未知错误"}`); const message = `成功 ${result.ok}，失败 ${result.failed}`; if (failures.length) toast.warning(`批量拉取完成：${message}`, { description: failures.slice(0, 4).join("；") }); else toast.success(`批量拉取完成：${message}`); await invalidate(); await client.invalidateQueries({ queryKey: ["proxies"] }) },
    onError: (error: Error) => toast.error(error.message),
  })
  const remove = useMutation({
    mutationFn: (ids: number[]) => ids.length === 1 ? api.delete(`/sources/${ids[0]}`) : api.post("/sources/batch-delete", { ids }),
    onSuccess: async () => { toast.success("已删除"); setSelected(new Set()); await invalidate() },
    onError: (error: Error) => toast.error(error.message),
  })
  const toggleSource = useMutation<unknown, Error, SubscriptionSource>({
    mutationFn: (item: SubscriptionSource) => item.kind === "manual"
      ? api.put<ManualSourceImportResult>(`/sources/manual/${item.id}`, { name: item.name, region: item.region, regionMode: item.regionMode, enabled: !item.enabled, content: item.manualContent ?? "" })
      : api.put<SubscriptionSource>(`/sources/${item.id}`, { enabled: !item.enabled }),
    onSuccess: async (_, item) => { toast.success(`已${item.enabled ? "停用" : "启用"}「${item.name}」`); await invalidate() },
    onError: (error: Error) => toast.error(error.message),
  })

  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!editing?.name.trim()) return toast.error("请填写名称")
    if (editing.kind === "remote" && !editing.id && !editing.url.trim()) return toast.error("请填写订阅 URL")
    if (editing.kind === "manual" && !editing.content.trim()) return toast.error("请粘贴节点分享链接")
    if (editing.regionMode === "fixed" && editing.region === "UNK") return toast.error("固定地区请选择具体地区")
    save.mutate(editing)
  }
  const items = sources.data?.items ?? []
  const allSelected = items.length > 0 && items.every((item) => selected.has(item.id))

  return (
    <div className="flex flex-col gap-4 md:gap-6">
      <PageHeader title="订阅源" description="维护远程订阅和手工节点，拉取后可逐个控制节点是否参与发布。" actions={<>
        <Button variant="outline" size="icon" aria-label="刷新列表" onClick={() => void sources.refetch()}><RefreshCw className={sources.isFetching ? "animate-spin" : ""} /></Button>
        <Button variant="outline" onClick={() => setEditing(sourceForm(undefined, "manual"))}><FilePlus2 />手工节点</Button>
        <Button onClick={() => setEditing(sourceForm())}><Plus />远程订阅</Button>
      </>} />
      <ResourceState pending={sources.isPending} error={sources.error} empty={!items.length} onRetry={() => void sources.refetch()}>
        <DataPanel title="订阅源列表" description={`共 ${items.length} 个源，${items.reduce((sum, item) => sum + item.proxyCount, 0)} 个节点`} actions={<>
          {selected.size ? <ConfirmAction title="批量删除订阅源" description={`将永久删除选中的 ${selected.size} 个订阅源及其节点。`} confirmLabel="删除" destructive onConfirm={() => remove.mutate([...selected])} trigger={<Button size="sm" variant="destructive"><Trash2 />删除 {selected.size}</Button>} /> : null}
          <ConfirmAction title="拉取全部订阅源" description="将从上游重新获取所有已启用远程订阅，并覆盖对应节点数据。" confirmLabel="开始拉取" disabled={refreshAll.isPending || !items.some((item) => item.kind === "remote" && item.enabled)} onConfirm={() => refreshAll.mutate()} trigger={<Button size="sm" variant="outline" disabled={refreshAll.isPending || !items.some((item) => item.kind === "remote" && item.enabled)}><CloudDownload className={refreshAll.isPending ? "animate-pulse" : ""} />全部拉取</Button>} />
        </>}>
          <Table>
            <TableHeader><TableRow><TableHead className="w-10"><Checkbox checked={allSelected} onCheckedChange={(checked) => setSelected(checked ? new Set(items.map((item) => item.id)) : new Set())} aria-label="全选" /></TableHead><TableHead>名称</TableHead><TableHead>类型 / 地区</TableHead><TableHead>流量</TableHead><TableHead>状态</TableHead><TableHead>节点</TableHead><TableHead>最近拉取</TableHead><TableHead className="w-16" /></TableRow></TableHeader>
            <TableBody>{items.map((item) => <TableRow key={item.id}>
              <TableCell><Checkbox checked={selected.has(item.id)} onCheckedChange={(checked) => setSelected((before) => { const next = new Set(before); if (checked) next.add(item.id); else next.delete(item.id); return next })} aria-label={`选择 ${item.name}`} /></TableCell>
              <TableCell><div className="max-w-72"><div className="truncate font-medium" title={item.name}>{item.name}</div><div className="truncate text-xs text-muted-foreground" title={item.urlMasked}>{item.kind === "manual" ? "本地手工节点" : item.urlMasked}</div></div></TableCell>
              <TableCell><div className="flex gap-1"><Badge variant="outline">{item.kind === "manual" ? "手工" : "远程"}</Badge><Badge variant="secondary">{item.regionMode === "auto" ? `自动/${item.region}` : item.region}</Badge></div></TableCell>
              <TableCell className="whitespace-nowrap text-xs text-muted-foreground" title={item.kind === "manual" ? "手工节点无上游流量信息" : `上传 ${formatBytes(item.trafficUpload)}\n下载 ${formatBytes(item.trafficDownload)}\n总量 ${item.trafficTotal > 0 ? formatBytes(item.trafficTotal) : "未知"}`}>{item.kind === "manual" ? "-" : trafficText(item)}</TableCell>
              <TableCell><Badge variant={item.enabled && item.refreshStatus !== "failed" ? "secondary" : "destructive"}>{!item.enabled ? "停用" : item.refreshStatus === "failed" ? "失败" : item.refreshStatus === "running" ? "拉取中" : "正常"}</Badge>{item.lastError ? <div className="mt-1 max-w-48 truncate text-xs text-destructive" title={item.lastError}>{item.lastError}</div> : null}</TableCell>
              <TableCell><Button variant="ghost" size="sm" onClick={() => setProxySource(item)}>{item.proxyCount}</Button></TableCell>
              <TableCell className="text-muted-foreground">{formatDateTime(item.lastRefreshAt)}</TableCell>
              <TableCell><DropdownMenu><DropdownMenuTrigger asChild><Button variant="ghost" size="icon" aria-label={`${item.name} 操作`}><MoreHorizontal /></Button></DropdownMenuTrigger><DropdownMenuContent align="end">
                {item.kind === "remote" ? <ConfirmAction title={`拉取「${item.name}」`} description="将重新从上游获取节点并覆盖本地数据。" confirmLabel="拉取" disabled={refresh.isPending} onConfirm={() => refresh.mutate(item.id)} trigger={<DropdownMenuItem onSelect={(event) => event.preventDefault()}><RefreshCw />拉取</DropdownMenuItem>} /> : null}<DropdownMenuItem disabled={toggleSource.isPending} onClick={() => toggleSource.mutate(item)}><Power />{item.enabled ? "停用" : "启用"}</DropdownMenuItem><DropdownMenuItem onClick={() => setProxySource(item)}><Search />查看节点</DropdownMenuItem><DropdownMenuItem onClick={() => setEditing(sourceForm(item))}><Edit3 />编辑</DropdownMenuItem><ConfirmAction title={`删除「${item.name}」`} description={`将永久删除该${item.kind === "manual" ? "手工节点源" : "订阅源"}及其 ${item.proxyCount} 个节点。`} confirmLabel="删除" destructive disabled={remove.isPending} onConfirm={() => remove.mutate([item.id])} trigger={<DropdownMenuItem variant="destructive" onSelect={(event) => event.preventDefault()}><Trash2 />删除</DropdownMenuItem>} />
              </DropdownMenuContent></DropdownMenu></TableCell>
            </TableRow>)}</TableBody>
          </Table>
        </DataPanel>
      </ResourceState>

      <FormDialog open={!!editing} onOpenChange={(open) => !open && setEditing(null)} title={editing?.id ? `编辑${editing.kind === "manual" ? "手工节点" : "订阅源"}` : `添加${editing?.kind === "manual" ? "手工节点" : "远程订阅"}`} busy={save.isPending} onSubmit={submit} className="sm:max-w-xl">
        {editing ? <>
          <Field><FieldLabel htmlFor="source-name">名称</FieldLabel><Input id="source-name" value={editing.name} maxLength={64} onChange={(e) => setEditing({ ...editing, name: e.target.value })} required /></Field>
          {editing.kind === "remote" ? <Field><FieldLabel htmlFor="source-url">订阅 URL</FieldLabel><Input id="source-url" type="url" value={editing.url} placeholder={editing.id ? "留空表示不修改" : "https://..."} onChange={(e) => setEditing({ ...editing, url: e.target.value })} required={!editing.id} /></Field> : <Field><FieldLabel htmlFor="manual-content">节点分享链接</FieldLabel><Textarea id="manual-content" rows={9} value={editing.content} onChange={(e) => setEditing({ ...editing, content: e.target.value })} placeholder="每行一个 vmess://、vless://、ss://、trojan:// 或 hysteria2:// 链接" required /><FieldDescription>保存时整体替换这批手工节点。</FieldDescription></Field>}
          <div className="grid gap-4 sm:grid-cols-2"><Field><FieldLabel>地区识别</FieldLabel><Select value={editing.regionMode} onValueChange={(value: "auto" | "fixed") => setEditing({ ...editing, regionMode: value, region: value === "auto" && !editing.region ? "UNK" : editing.region })}><SelectTrigger className="w-full"><SelectValue /></SelectTrigger><SelectContent><SelectItem value="auto">自动识别</SelectItem><SelectItem value="fixed">固定地区</SelectItem></SelectContent></Select></Field><Field><FieldLabel>默认地区</FieldLabel><Select value={editing.region} onValueChange={(region) => setEditing({ ...editing, region })}><SelectTrigger className="w-full"><SelectValue /></SelectTrigger><SelectContent>{(regions.data?.items ?? [{ code: "UNK", name: "未知" }]).filter((item) => editing.regionMode === "auto" || item.code !== "UNK").map((item) => <SelectItem key={item.code} value={item.code}>{item.code} · {item.name}</SelectItem>)}</SelectContent></Select></Field></div>
          {editing.kind === "remote" ? <div className="grid gap-4 rounded-md border p-3"><div className="text-sm font-medium">节点过滤</div><Field><FieldLabel>名称包含（正则）</FieldLabel><Input value={editing.includeNameRegex} onChange={(e) => setEditing({ ...editing, includeNameRegex: e.target.value })} /></Field><Field><FieldLabel>名称排除（正则）</FieldLabel><Input value={editing.excludeNameRegex} onChange={(e) => setEditing({ ...editing, excludeNameRegex: e.target.value })} /></Field><Field><FieldLabel>服务器排除</FieldLabel><Textarea rows={3} value={editing.excludeServers} onChange={(e) => setEditing({ ...editing, excludeServers: e.target.value })} /><FieldDescription>按现有服务规则填写要排除的服务器。</FieldDescription></Field></div> : null}
          <Field orientation="horizontal"><FieldLabel htmlFor="source-enabled">启用</FieldLabel><Switch id="source-enabled" checked={editing.enabled} onCheckedChange={(enabled) => setEditing({ ...editing, enabled })} /></Field>
        </> : null}
      </FormDialog>
      <ProxyDialog source={proxySource} onOpenChange={(open) => !open && setProxySource(null)} />
    </div>
  )
}

function normalize(value: string) { return value.replace(/[\u200B-\u200D\uFEFF]/g, "").normalize("NFKC").trim().toLowerCase() }

function ProxyDialog({ source, onOpenChange }: { source: SubscriptionSource | null; onOpenChange: (open: boolean) => void }) {
  const client = useQueryClient()
  const [search, setSearch] = useState("")
  const [region, setRegion] = useState("all")
  const [quality, setQuality] = useState("all")
  const proxies = useQuery({ queryKey: queryKeys.proxies(source?.id), queryFn: () => api.get<ListResponse<ProxyNode>>("/proxies", { params: { sourceId: source?.id } }), enabled: !!source })
  const update = useMutation<unknown, Error, { ids: number[]; enabled: boolean }>({
    mutationFn: ({ ids, enabled }: { ids: number[]; enabled: boolean }) => ids.length === 1 ? api.put<ProxyNode>(`/proxies/${ids[0]}`, { enabled }) : api.put<{ updated: number }>("/proxies/batch", { ids, enabled }),
    onSuccess: async () => { await client.invalidateQueries({ queryKey: queryKeys.proxies(source?.id) }); for (const key of draftDependencies) await client.invalidateQueries({ queryKey: key }) },
    onError: (error: Error) => toast.error(error.message),
  })
  const items = useMemo(() => proxies.data?.items ?? [], [proxies.data?.items])
  const filtered = useMemo(() => items.filter((item) => {
    if (region !== "all" && item.region !== region) return false
    if (quality === "ok" && !item.ok) return false
    if (quality === "bad" && item.ok) return false
    const q = normalize(search)
    return !q || [item.name, item.server, item.type, item.region, item.issue ?? "", String(item.port)].some((value) => normalize(value).includes(q))
  }), [items, quality, region, search])
  const regions = [...new Set(items.map((item) => item.region))].sort()
  const allEnabled = filtered.length > 0 && filtered.every((item) => item.enabled)
  return <Dialog open={!!source} onOpenChange={onOpenChange}><DialogContent className="sm:max-w-5xl"><DialogHeader><DialogTitle>{source?.name} · 节点</DialogTitle><DialogDescription>筛选、检查并控制进入发布配置的节点。</DialogDescription></DialogHeader>
    <div className="flex flex-wrap gap-2"><div className="relative min-w-48 flex-1"><Search className="absolute left-2.5 top-2 size-4 text-muted-foreground" /><Input className="pl-8" value={search} onChange={(e) => setSearch(e.target.value)} placeholder="搜索节点、服务器或类型" /></div><Select value={region} onValueChange={setRegion}><SelectTrigger><SelectValue /></SelectTrigger><SelectContent><SelectItem value="all">全部地区</SelectItem>{regions.map((item) => <SelectItem key={item} value={item}>{item}</SelectItem>)}</SelectContent></Select><Select value={quality} onValueChange={setQuality}><SelectTrigger><SelectValue /></SelectTrigger><SelectContent><SelectItem value="all">全部质量</SelectItem><SelectItem value="ok">识别正常</SelectItem><SelectItem value="bad">需要检查</SelectItem></SelectContent></Select><Button variant="outline" disabled={!filtered.length || update.isPending} onClick={() => update.mutate({ ids: filtered.map((item) => item.id), enabled: !allEnabled })}>{allEnabled ? "全部停用" : "全部启用"}</Button></div>
    <div className="max-h-[60vh] overflow-auto rounded-md border"><Table><TableHeader><TableRow><TableHead>启用</TableHead><TableHead>地区</TableHead><TableHead>名称</TableHead><TableHead>类型</TableHead><TableHead>服务器</TableHead><TableHead>识别</TableHead></TableRow></TableHeader><TableBody>{filtered.map((item) => <TableRow key={item.id}><TableCell><Switch checked={item.enabled} onCheckedChange={(enabled) => update.mutate({ ids: [item.id], enabled })} /></TableCell><TableCell>{item.region}</TableCell><TableCell className="max-w-64 truncate" title={item.name}>{item.name}</TableCell><TableCell>{item.type}</TableCell><TableCell>{item.server}:{item.port}</TableCell><TableCell>{item.ok ? <Check className="size-4 status-success" /> : <span className="text-xs text-destructive" title={item.issue}>{item.issue || "异常"}</span>}</TableCell></TableRow>)}</TableBody></Table></div>
    <div className="text-xs text-muted-foreground">显示 {filtered.length} / {items.length}，启用 {filtered.filter((item) => item.enabled).length}</div>
  </DialogContent></Dialog>
}
