import { useQuery, useQueryClient } from "@tanstack/react-query"
import { FileText, PanelLeft, RefreshCw, Search } from "lucide-react"
import { useDeferredValue, useEffect, useMemo, useState } from "react"

import { PageHeader } from "@/components/page-header"
import { ResourceState } from "@/components/resource-state"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { api } from "@/lib/api"
import { queryKeys } from "@/lib/query-keys"
import type { LogEntry, LogFileInfo } from "@/lib/types"

type LogFilesResponse = { files: LogFileInfo[] }
type LogEntriesResponse = { items: LogEntry[] }
const lineOptions = [50, 100, 200, 500, 1000]
const levelOptions = ["all", "error", "warn", "info", "debug"] as const

function initialLines() {
  const value = Number(window.localStorage.getItem("wt_log_system_defaultLine"))
  return lineOptions.includes(value) ? value : 100
}

function formatBytes(value: number) {
  if (value < 1024) return `${value} B`
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`
  return `${(value / 1024 / 1024).toFixed(1)} MB`
}

function logTime(value: number) {
  if (!value) return ""
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? String(value) : date.toLocaleString("zh-CN", { hour12: false })
}

export function LogsPage() {
  const client = useQueryClient()
  const [search, setSearch] = useState("")
  const deferredSearch = useDeferredValue(search.trim())
  const [selected, setSelected] = useState("")
  const [lines, setLines] = useState(initialLines)
  const [level, setLevel] = useState<(typeof levelOptions)[number]>("all")
  const [sidebar, setSidebar] = useState(true)
  const files = useQuery({ queryKey: [...queryKeys.logs, "files", deferredSearch], queryFn: () => api.get<LogFilesResponse>("/logs", { params: { name: deferredSearch } }) })
  const activeName = files.data?.files.some((file) => file.name === selected) ? selected : files.data?.files[0]?.name ?? ""
  const details = useQuery({ queryKey: [...queryKeys.logs, "details", activeName, lines], queryFn: () => api.get<LogEntriesResponse>("/logs/details", { params: { name: activeName, line: lines }, timeoutMs: 15_000 }), enabled: !!activeName })
  const visible = useMemo(() => (details.data?.items ?? []).filter((item) => level === "all" || (level === "warn" ? item.level.toLowerCase() === "warn" || item.level.toLowerCase() === "warning" : item.level.toLowerCase() === level)), [details.data?.items, level])

  useEffect(() => { window.localStorage.setItem("wt_log_system_defaultLine", String(lines)) }, [lines])
  const refresh = async () => { await client.invalidateQueries({ queryKey: queryKeys.logs }) }

  return <div className="flex flex-col gap-4 md:gap-6"><PageHeader title="系统日志" description="按日期查看应用日志，并在当前已加载内容中筛选等级。" actions={<Button variant="outline" size="icon" aria-label="刷新日志" onClick={() => void refresh()}><RefreshCw className={files.isFetching || details.isFetching ? "animate-spin" : ""} /></Button>} />
    <Card className="gap-0 overflow-hidden py-0"><div className={`grid min-h-[calc(100vh-13rem)] ${sidebar ? "md:grid-cols-[17rem_minmax(0,1fr)]" : "md:grid-cols-1"}`}>
      <aside className={`${sidebar ? "flex" : "hidden"} min-h-0 flex-col border-b bg-muted/20 md:border-r md:border-b-0`}><div className="border-b p-3"><div className="relative"><Search className="absolute left-2.5 top-2.5 size-4 text-muted-foreground" /><Input value={search} onChange={(event) => setSearch(event.target.value)} placeholder="搜索日志名称" className="pl-8" /></div></div><ResourceState pending={files.isPending} error={files.error} empty={!files.data?.files.length} onRetry={() => void files.refetch()}><nav className="max-h-56 overflow-y-auto p-2 md:max-h-[calc(100vh-18rem)]">{files.data?.files.map((file) => <button key={file.name} type="button" onClick={() => { setSelected(file.name); if (window.innerWidth < 768) setSidebar(false) }} className={`mb-1 flex w-full items-center gap-2 rounded-md px-3 py-2 text-left text-sm transition-colors ${activeName === file.name ? "bg-accent text-accent-foreground" : "hover:bg-muted"}`}><FileText className="size-4 shrink-0" /><span className="min-w-0 flex-1 truncate">{file.name}</span><span className="text-xs text-muted-foreground">{formatBytes(file.size)}</span></button>)}</nav></ResourceState></aside>
      <main className="min-w-0"><div className="flex flex-wrap items-center gap-2 border-b p-2"><Button variant="ghost" size="icon" aria-label="切换日志文件列表" onClick={() => setSidebar(!sidebar)}><PanelLeft /></Button><Select value={String(lines)} onValueChange={(value) => setLines(Number(value))}><SelectTrigger className="w-28"><SelectValue /></SelectTrigger><SelectContent>{lineOptions.map((value) => <SelectItem key={value} value={String(value)}>{value} 行</SelectItem>)}</SelectContent></Select><Select value={level} onValueChange={(value: (typeof levelOptions)[number]) => setLevel(value)}><SelectTrigger className="w-28"><SelectValue /></SelectTrigger><SelectContent>{levelOptions.map((value) => <SelectItem key={value} value={value}>{value === "all" ? "全部等级" : value}</SelectItem>)}</SelectContent></Select><span className="min-w-0 flex-1 truncate text-right text-xs text-muted-foreground" title={activeName}>{activeName || "未选择日志"}</span></div>
        <ResourceState pending={!!activeName && details.isPending} error={details.error} empty={!activeName || !visible.length} onRetry={() => void details.refetch()}><div className="max-h-[calc(100vh-17rem)] overflow-auto divide-y">{visible.map((item, index) => <article key={`${item.timestamp}:${item.caller}:${index}`} className="p-3 [content-visibility:auto] [contain-intrinsic-size:64px]"><div className="mb-1 flex flex-wrap items-center gap-2 text-xs text-muted-foreground"><span>{logTime(item.timestamp)}</span><span className="font-mono">{item.caller}</span><LogLevel level={item.level} /></div><pre className="whitespace-pre-wrap break-words font-sans text-sm leading-5">{item.content}</pre></article>)}</div></ResourceState>
      </main>
    </div></Card>
  </div>
}

function LogLevel({ level }: { level: string }) {
  const normalized = level.toLowerCase()
  const variant = normalized === "error" ? "destructive" : "outline"
  const className = normalized === "warn" || normalized === "warning" ? "status-warning" : normalized === "info" ? "status-info" : ""
  return <Badge variant={variant} className={className}>{normalized || "info"}</Badge>
}
