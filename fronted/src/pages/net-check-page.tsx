import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Plus, RefreshCw, RotateCcw, Settings2, Trash2, Wifi, WifiOff } from "lucide-react"
import { useEffect, useRef, useState, type FormEvent } from "react"
import { toast } from "sonner"

import { ConfirmAction } from "@/components/confirm-action"
import { DataPanel } from "@/components/data-panel"
import { PageHeader } from "@/components/page-header"
import { ResourceState } from "@/components/resource-state"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Field, FieldDescription, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Switch } from "@/components/ui/switch"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { api } from "@/lib/api"
import { queryKeys } from "@/lib/query-keys"
import type { NetCheckConfig, NetCheckResult, NetCheckTarget } from "@/lib/types"

const fallbackConfig: NetCheckConfig = { timeout: 5, autoRefresh: 0, targets: [] }
const cloneConfig = (config: NetCheckConfig): NetCheckConfig => ({ ...config, targets: config.targets.map((target) => ({ ...target })) })
const formatMs = (value: number) => value < 1000 ? `${value} ms` : `${(value / 1000).toFixed(2)} s`

export function NetCheckPage() {
  const client = useQueryClient()
  const [proxyURL, setProxyURL] = useState("")
  const [proxyEnabled, setProxyEnabled] = useState(false)
  const [result, setResult] = useState<NetCheckResult | null>(null)
  const [settingsOpen, setSettingsOpen] = useState(false)
  const [draft, setDraft] = useState<NetCheckConfig>(fallbackConfig)
  const checking = useRef(false)
  const config = useQuery({ queryKey: queryKeys.netCheck, queryFn: () => api.get<NetCheckConfig>("/net-check/config") })
  const check = useMutation({
    mutationFn: (targets?: NetCheckTarget[]) => api.post<NetCheckResult>("/net-check/check", { proxy: { enabled: proxyEnabled, url: proxyURL.trim() }, timeout: config.data?.timeout ?? 5, autoRefresh: config.data?.autoRefresh ?? 0, targets: targets ?? config.data?.targets ?? [] }),
    onMutate: () => { checking.current = true },
    onSuccess: setResult,
    onError: (error: Error) => toast.error(error.message),
    onSettled: () => { checking.current = false },
  })
  const runCheck = check.mutate
  const save = useMutation({
    mutationFn: (body: NetCheckConfig) => api.put<NetCheckConfig>("/net-check/config", body),
    onSuccess: async (data) => { toast.success("检测设置已保存"); setSettingsOpen(false); client.setQueryData(queryKeys.netCheck, data); await client.invalidateQueries({ queryKey: queryKeys.netCheck }) },
    onError: (error: Error) => toast.error(error.message),
  })
  const reset = useMutation({
    mutationFn: () => api.post<NetCheckConfig>("/net-check/reset"),
    onSuccess: async (data) => { setDraft(cloneConfig(data)); client.setQueryData(queryKeys.netCheck, data); toast.success("已恢复默认检测目标") },
    onError: (error: Error) => toast.error(error.message),
  })

  useEffect(() => {
    const seconds = config.data?.autoRefresh ?? 0
    if (seconds <= 0) return
    const timer = window.setInterval(() => { if (!checking.current) runCheck(undefined) }, seconds * 1000)
    return () => window.clearInterval(timer)
  }, [config.data?.autoRefresh, runCheck])

  const openSettings = () => { setDraft(cloneConfig(config.data ?? fallbackConfig)); setSettingsOpen(true) }
  const setProxy = (value: string) => { setProxyURL(value); setProxyEnabled(!!value.trim()) }
  const failed = result?.results.filter((item) => item.status === "FAIL") ?? []

  return <div className="flex flex-col gap-4 md:gap-6"><PageHeader title="网络检测" description="检测服务端到目标站点的连通性；临时代理仅对当前页面有效。" actions={<Button onClick={openSettings}><Settings2 />检测设置</Button>} />
    <DataPanel title="检测参数" description="代理设置仅应用于本次页面会话"><form className="flex flex-col gap-3 p-4 lg:flex-row lg:items-end" onSubmit={(event) => { event.preventDefault(); check.mutate(undefined) }}><Field className="flex-1"><FieldLabel htmlFor="net-proxy">本次检测代理 URL</FieldLabel><Input id="net-proxy" value={proxyURL} onChange={(event) => setProxy(event.target.value)} placeholder="http://127.0.0.1:7890" /><FieldDescription>留空但启用时回退到系统全局代理；关闭则直连。</FieldDescription></Field><label className="flex h-9 items-center gap-2 text-sm"><Checkbox checked={proxyEnabled} onCheckedChange={(checked) => setProxyEnabled(checked === true)} />启用代理</label><Button type="submit" disabled={check.isPending || !config.data}><Wifi className={check.isPending ? "animate-pulse" : ""} />{check.isPending ? "检测中..." : "开始检测"}</Button></form></DataPanel>
    <ResourceState pending={config.isPending} error={config.error} onRetry={() => void config.refetch()}><DataPanel title="检测结果" description={result ? `共 ${result.summary.total} 个目标，耗时 ${formatMs(result.summary.durationMs)}` : "尚未开始检测"} actions={result ? <><Badge variant="secondary">{result.summary.ok} 成功</Badge><Badge variant={result.summary.fail ? "destructive" : "outline"}>{result.summary.fail} 失败</Badge>{result.proxyMode ? <Badge variant="outline">{result.proxyInfo || (result.proxyMode === "proxy" ? "代理" : "直连")}</Badge> : null}{failed.length ? <Button variant="outline" size="sm" disabled={check.isPending} onClick={() => check.mutate(failed.map((item) => ({ name: item.name, url: item.url, enabled: true })))}><RefreshCw />重试失败项</Button> : null}</> : null}>{!result ? <div className="grid min-h-52 place-items-center text-sm text-muted-foreground"><span className="flex items-center gap-2"><Wifi className="size-4" />暂无检测结果</span></div> : <Table><TableHeader><TableRow><TableHead>名称</TableHead><TableHead>URL</TableHead><TableHead>结果</TableHead><TableHead>状态码</TableHead><TableHead>总耗时</TableHead><TableHead>阶段耗时</TableHead><TableHead>错误信息</TableHead></TableRow></TableHeader><TableBody>{result.results.map((item) => <TableRow key={`${item.name}:${item.url}`}><TableCell className="font-medium">{item.name}</TableCell><TableCell className="max-w-64 truncate" title={item.http.effectiveUrl || item.url}>{item.url}</TableCell><TableCell>{item.status === "OK" ? <Badge variant="secondary"><Wifi />可达</Badge> : <Badge variant="destructive"><WifiOff />失败</Badge>}</TableCell><TableCell>{item.http.code || "-"}</TableCell><TableCell>{formatMs(item.http.timeMs)}</TableCell><TableCell className="whitespace-nowrap text-xs text-muted-foreground">连接 {formatMs(item.http.timing.connectMs)} · TLS {formatMs(item.http.timing.tlsMs)} · 首包 {formatMs(item.http.timing.firstByteMs)}</TableCell><TableCell className="max-w-64 text-xs text-destructive" title={item.http.error}>{item.http.error || "-"}</TableCell></TableRow>)}</TableBody></Table>}</DataPanel></ResourceState>
    <SettingsDialog open={settingsOpen} onOpenChange={setSettingsOpen} draft={draft} setDraft={setDraft} saving={save.isPending} resetting={reset.isPending} save={() => save.mutate(draft)} reset={() => reset.mutate()} />
  </div>
}

function SettingsDialog({ open, onOpenChange, draft, setDraft, saving, resetting, save, reset }: { open: boolean; onOpenChange: (open: boolean) => void; draft: NetCheckConfig; setDraft: (config: NetCheckConfig) => void; saving: boolean; resetting: boolean; save: () => void; reset: () => void }) {
  const updateTarget = (index: number, patch: Partial<NetCheckTarget>) => setDraft({ ...draft, targets: draft.targets.map((target, current) => current === index ? { ...target, ...patch } : target) })
  const submit = (event: FormEvent) => { event.preventDefault(); if (draft.timeout < 1 || draft.timeout > 120) return toast.error("超时必须在 1 到 120 秒之间"); if (draft.autoRefresh < 0 || draft.autoRefresh > 3600) return toast.error("自动刷新必须在 0 到 3600 秒之间"); if (draft.targets.some((target) => !target.name.trim() || !target.url.trim())) return toast.error("请填写完整的目标名称和 URL"); save() }
  return <Dialog open={open} onOpenChange={onOpenChange}><DialogContent className="sm:max-w-4xl"><DialogHeader><DialogTitle>检测设置</DialogTitle><DialogDescription>目标、超时和自动刷新会持久化；临时代理不会保存。</DialogDescription></DialogHeader><form className="contents" onSubmit={submit}><div className="grid max-h-[65vh] gap-4 overflow-y-auto"><div className="grid gap-3 sm:grid-cols-2"><Field><FieldLabel htmlFor="check-timeout">超时（秒）</FieldLabel><Input id="check-timeout" type="number" min={1} max={120} value={draft.timeout} onChange={(event) => setDraft({ ...draft, timeout: Number(event.target.value) })} /></Field><Field><FieldLabel htmlFor="check-auto">自动刷新（秒）</FieldLabel><Input id="check-auto" type="number" min={0} max={3600} value={draft.autoRefresh} onChange={(event) => setDraft({ ...draft, autoRefresh: Number(event.target.value) })} /><FieldDescription>0 表示关闭，离开页面也会停止。</FieldDescription></Field></div><div className="flex items-center justify-between"><h3 className="text-sm font-medium">检测目标 · {draft.targets.length}</h3><div className="flex gap-2"><ConfirmAction title="恢复默认检测目标" description="当前已保存的检测设置会被默认值覆盖。" confirmLabel="恢复默认" destructive onConfirm={reset} disabled={resetting} trigger={<Button type="button" variant="outline" size="sm" disabled={resetting}><RotateCcw />恢复默认</Button>} /><Button type="button" variant="outline" size="sm" onClick={() => setDraft({ ...draft, targets: [...draft.targets, { name: "自定义目标", url: "https://example.com/", enabled: true }] })}><Plus />添加目标</Button></div></div><div className="space-y-2">{draft.targets.length ? draft.targets.map((target, index) => <div key={index} className="grid gap-2 rounded-md border p-3 sm:grid-cols-[auto_1fr_2fr_auto] sm:items-center"><Switch checked={target.enabled} onCheckedChange={(enabled) => updateTarget(index, { enabled })} aria-label={`启用 ${target.name}`} /><Input value={target.name} maxLength={80} onChange={(event) => updateTarget(index, { name: event.target.value })} aria-label="目标名称" /><Input value={target.url} type="url" onChange={(event) => updateTarget(index, { url: event.target.value })} aria-label="目标 URL" /><Button type="button" variant="ghost" size="icon" aria-label="删除目标" onClick={() => setDraft({ ...draft, targets: draft.targets.filter((_, current) => current !== index) })}><Trash2 className="text-destructive" /></Button></div>) : <div className="grid min-h-28 place-items-center border-y text-sm text-muted-foreground">保存空列表时服务端将恢复默认目标</div>}</div></div><DialogFooter><Button type="button" variant="outline" onClick={() => onOpenChange(false)}>取消</Button><Button type="submit" disabled={saving}>{saving ? "保存中..." : "保存设置"}</Button></DialogFooter></form></DialogContent></Dialog>
}
