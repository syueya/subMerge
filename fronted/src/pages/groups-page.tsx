import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Edit3, Plus, RefreshCw, Trash2 } from "lucide-react"
import { useMemo, useState, type FormEvent } from "react"
import { toast } from "sonner"

import { FormDialog } from "@/components/form-dialog"
import { DataPanel } from "@/components/data-panel"
import { PageHeader } from "@/components/page-header"
import { ResourceState } from "@/components/resource-state"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Field, FieldDescription, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Textarea } from "@/components/ui/textarea"
import { api } from "@/lib/api"
import { draftDependencies, queryKeys } from "@/lib/query-keys"
import type { ListResponse, ProxyGroup, ProxyNode, RegionCatalogResponse, Rule, SubscriptionSource } from "@/lib/types"

type GroupForm = Omit<ProxyGroup, "id"> & { id?: number }
const emptyGroup: GroupForm = { name: "", type: "select", proxies: ["ALL"], url: "https://www.gstatic.com/generate_204", interval: 300, enabled: true, sortOrder: 0 }
const groupTypes: Record<string, string> = { select: "手动选择", "url-test": "自动测速", fallback: "故障转移", "load-balance": "负载均衡" }

export function GroupsPage() {
  const client = useQueryClient()
  const [editing, setEditing] = useState<GroupForm | null>(null)
  const [deleting, setDeleting] = useState<ProxyGroup | null>(null)
  const groups = useQuery({ queryKey: queryKeys.groups, queryFn: () => api.get<ListResponse<ProxyGroup>>("/groups") })
  const rules = useQuery({ queryKey: queryKeys.rules, queryFn: () => api.get<ListResponse<Rule>>("/rules") })
  const sources = useQuery({ queryKey: queryKeys.sources, queryFn: () => api.get<ListResponse<SubscriptionSource>>("/sources") })
  const proxies = useQuery({ queryKey: queryKeys.proxies(), queryFn: () => api.get<ListResponse<ProxyNode>>("/proxies") })
  const regions = useQuery({ queryKey: queryKeys.regions, queryFn: () => api.get<RegionCatalogResponse>("/regions"), staleTime: Infinity })
  const invalidate = async () => { await client.invalidateQueries({ queryKey: queryKeys.groups }); for (const key of draftDependencies) await client.invalidateQueries({ queryKey: key }) }
  const save = useMutation({
    mutationFn: (form: GroupForm) => {
      const body = { name: form.name.trim(), type: form.type, proxies: [...new Set(form.proxies.map((item) => item.trim()).filter(Boolean))], url: form.url?.trim() || "", interval: form.interval || undefined, enabled: form.enabled, sortOrder: form.sortOrder }
      return form.id ? api.put<ProxyGroup>(`/groups/${form.id}`, body) : api.post<ProxyGroup>("/groups", body)
    },
    onSuccess: async () => { toast.success("策略组已保存"); setEditing(null); await invalidate() },
    onError: (error: Error) => toast.error(error.message),
  })
  const remove = useMutation({
    mutationFn: ({ id, cascade }: { id: number; cascade: boolean }) => api.delete(`/groups/${id}`, { params: { cascadeRules: cascade } }),
    onSuccess: async () => { toast.success("策略组已删除"); setDeleting(null); await invalidate(); await client.invalidateQueries({ queryKey: queryKeys.rules }) },
    onError: (error: Error) => toast.error(error.message),
  })
  const items = useMemo(() => groups.data?.items ?? [], [groups.data?.items])
  const refs = useMemo(() => new Map(items.map((group) => [group.id, (rules.data?.items ?? []).filter((rule) => rule.target === group.name).length])), [items, rules.data?.items])
  const presets = useMemo(() => {
    const result = new Set(["ALL", "DIRECT", "REJECT"])
    for (const source of sources.data?.items ?? []) result.add(`SOURCE:${source.name}`)
    for (const region of regions.data?.items ?? []) if (!["UNK", "UNKNOWN"].includes(region.code.toUpperCase())) result.add(`REGION:${region.code.toUpperCase()}`)
    for (const proxy of proxies.data?.items ?? []) if (!['UNK', 'UNKNOWN'].includes(proxy.region.toUpperCase())) result.add(`REGION:${proxy.region.toUpperCase()}`)
    for (const proxy of proxies.data?.items ?? []) result.add(proxy.name)
    for (const group of items) result.add(group.name)
    return [...result]
  }, [items, proxies.data?.items, regions.data?.items, sources.data?.items])

  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!editing?.name.trim()) return toast.error("请填写策略组名称")
    if (!editing.proxies.length) return toast.error("至少选择一个成员")
    if (["url-test", "fallback"].includes(editing.type) && !editing.url?.match(/^https?:\/\//)) return toast.error("测速 URL 必须是 HTTP 或 HTTPS")
    if (["url-test", "fallback"].includes(editing.type) && (editing.interval ?? 0) < 30) return toast.error("检测间隔不能少于 30 秒")
    save.mutate(editing)
  }
  const toggleMember = (member: string, checked: boolean) => {
    if (!editing) return
    let next = checked ? [...editing.proxies, member] : editing.proxies.filter((item) => item !== member)
    if (checked && member !== "ALL") next = next.filter((item) => item !== "ALL")
    if (checked && member === "ALL") next = ["ALL"]
    setEditing({ ...editing, proxies: [...new Set(next)] })
  }
  return <div className="flex flex-col gap-4 md:gap-6">
    <PageHeader title="策略分组" description="将节点、订阅源或其它策略组合成规则的目标出口。" actions={<><Button variant="outline" size="icon" aria-label="刷新策略组" onClick={() => void Promise.all([groups.refetch(), rules.refetch(), sources.refetch(), proxies.refetch()])}><RefreshCw className={groups.isFetching ? "animate-spin" : ""} /></Button><Button onClick={() => setEditing({ ...emptyGroup, sortOrder: items.length })}><Plus />新建策略组</Button></>} />
    <ResourceState pending={groups.isPending} error={groups.error} empty={!items.length} onRetry={() => void groups.refetch()}>
      <DataPanel title="策略组列表" description={`共 ${items.length} 个策略组，${items.filter((item) => item.enabled).length} 个已启用`}><Table><TableHeader><TableRow><TableHead>名称</TableHead><TableHead>类型</TableHead><TableHead>成员</TableHead><TableHead>规则引用</TableHead><TableHead>检测</TableHead><TableHead className="w-28" /></TableRow></TableHeader><TableBody>{items.map((item) => <TableRow key={item.id}><TableCell><div className="flex items-center gap-2"><span className="font-medium">{item.name}</span>{!item.enabled ? <Badge variant="outline">停用</Badge> : null}</div></TableCell><TableCell><Badge variant="secondary">{groupTypes[item.type] ?? item.type}</Badge></TableCell><TableCell><div className="max-w-96 truncate" title={item.proxies.join("、")}>{item.proxies.join("、") || "-"}</div></TableCell><TableCell>{refs.get(item.id) ?? 0}</TableCell><TableCell className="text-muted-foreground">{["url-test", "fallback"].includes(item.type) ? `${item.interval ?? 300}s` : "-"}</TableCell><TableCell><div className="flex justify-end"><Button variant="ghost" size="icon" aria-label={`编辑 ${item.name}`} onClick={() => setEditing({ ...item })}><Edit3 /></Button><Button variant="ghost" size="icon" aria-label={`删除 ${item.name}`} onClick={() => setDeleting(item)}><Trash2 /></Button></div></TableCell></TableRow>)}</TableBody></Table></DataPanel>
    </ResourceState>
    <FormDialog open={!!editing} onOpenChange={(open) => !open && setEditing(null)} title={editing?.id ? "编辑策略组" : "新建策略组"} busy={save.isPending} onSubmit={submit} className="sm:max-w-2xl">
      {editing ? <><div className="grid gap-4 sm:grid-cols-2"><Field><FieldLabel htmlFor="group-name">名称</FieldLabel><Input id="group-name" value={editing.name} maxLength={64} onChange={(e) => setEditing({ ...editing, name: e.target.value })} required /></Field><Field><FieldLabel>类型</FieldLabel><Select value={editing.type} onValueChange={(type) => setEditing({ ...editing, type })}><SelectTrigger className="w-full"><SelectValue /></SelectTrigger><SelectContent>{Object.entries(groupTypes).map(([value, label]) => <SelectItem key={value} value={value}>{label}</SelectItem>)}</SelectContent></Select></Field></div>
      {["url-test", "fallback"].includes(editing.type) ? <div className="grid gap-4 sm:grid-cols-[1fr_10rem]"><Field><FieldLabel htmlFor="group-url">检测 URL</FieldLabel><Input id="group-url" type="url" value={editing.url ?? ""} onChange={(e) => setEditing({ ...editing, url: e.target.value })} required /></Field><Field><FieldLabel htmlFor="group-interval">间隔（秒）</FieldLabel><Input id="group-interval" type="number" min={30} max={86400} value={editing.interval ?? 300} onChange={(e) => setEditing({ ...editing, interval: Number(e.target.value) })} /></Field></div> : null}
      <Field><FieldLabel>常用成员</FieldLabel><div className="grid max-h-52 gap-2 overflow-y-auto rounded-md border p-3 sm:grid-cols-2">{presets.filter((member) => member !== editing.name).map((member) => <label className="flex min-w-0 items-center gap-2 text-sm" key={member}><Checkbox checked={editing.proxies.includes(member)} onCheckedChange={(checked) => toggleMember(member, !!checked)} /><span className="truncate" title={member}>{member}</span></label>)}</div><FieldDescription>支持 ALL、SOURCE:*、REGION:*、节点名和其它策略组；选择其它成员时会自动移除 ALL。</FieldDescription></Field>
      <Field><FieldLabel htmlFor="group-members">自定义成员</FieldLabel><Textarea id="group-members" rows={4} value={editing.proxies.filter((item) => !presets.includes(item)).join("\n")} onChange={(e) => setEditing({ ...editing, proxies: [...editing.proxies.filter((item) => presets.includes(item)), ...e.target.value.split(/\r?\n/).map((item) => item.trim()).filter(Boolean)] })} placeholder="每行一个节点名或 SOURCE:id:3" /></Field>
      <Field orientation="horizontal"><FieldLabel htmlFor="group-enabled">启用</FieldLabel><Switch id="group-enabled" checked={editing.enabled} onCheckedChange={(enabled) => setEditing({ ...editing, enabled })} /></Field></> : null}
    </FormDialog>
    <Dialog open={!!deleting} onOpenChange={(open) => !open && setDeleting(null)}><DialogContent><DialogHeader><DialogTitle>删除「{deleting?.name}」</DialogTitle><DialogDescription>{deleting && (refs.get(deleting.id) ?? 0) > 0 ? `有 ${refs.get(deleting.id)} 条规则引用该组。可以同时删除这些规则，或只删除组并保留失效规则。` : "删除后无法恢复。"}</DialogDescription></DialogHeader><DialogFooter>{deleting && (refs.get(deleting.id) ?? 0) > 0 ? <Button variant="outline" disabled={remove.isPending} onClick={() => remove.mutate({ id: deleting.id, cascade: false })}>保留规则</Button> : null}<Button variant="destructive" disabled={remove.isPending} onClick={() => deleting && remove.mutate({ id: deleting.id, cascade: (refs.get(deleting.id) ?? 0) > 0 })}>{(refs.get(deleting?.id ?? 0) ?? 0) > 0 ? "组和规则一起删除" : "删除"}</Button></DialogFooter></DialogContent></Dialog>
  </div>
}
