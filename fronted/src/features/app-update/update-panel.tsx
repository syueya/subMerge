import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { CheckCircle2, Download, RefreshCw, RotateCcw } from "lucide-react"
import { toast } from "sonner"

import { ConfirmAction } from "@/components/confirm-action"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Progress } from "@/components/ui/progress"
import { api } from "@/lib/api"
import { queryKeys } from "@/lib/query-keys"
import type { UpdateStatus } from "@/lib/types"
import { formatDateTime } from "@/lib/types"
import { updateBusy, updateProgress, waitForVersion } from "@/features/app-update/update-client"

const phaseText: Record<string, string> = { disabled: "未启用", idle: "已是最新", checking: "检查中", available: "可下载", downloading: "下载中", ready: "可以安装", installing: "安装中", restarting: "重启中", failed: "失败" }

export function UpdatePanel() {
  const client = useQueryClient()
  const status = useQuery({ queryKey: queryKeys.update, queryFn: () => api.get<UpdateStatus>("/update/status"), refetchInterval: (query) => updateBusy(query.state.data) ? 1_000 : 30_000 })
  const action = useMutation({
    mutationFn: async (kind: "check" | "download" | "install" | "rollback") => {
      const result = await api.post<UpdateStatus>(`/update/${kind}`)
      if (kind === "install" || kind === "rollback") {
        const expected = kind === "install" ? result.latestVersion : result.rollbackVersion
        if (!expected) throw new Error("服务器未返回重启后的目标版本")
        await waitForVersion(expected)
        window.location.reload()
      }
      return result
    },
    onSuccess: async (_, kind) => { if (kind !== "install" && kind !== "rollback") toast.success(kind === "check" ? "更新检查完成" : "更新包已就绪"); await client.invalidateQueries({ queryKey: queryKeys.update }) },
    onError: (error: Error) => { toast.error(error.message); void client.invalidateQueries({ queryKey: queryKeys.update }) },
  })
  const value = status.data
  if (status.isPending) return <div className="h-40 animate-pulse rounded-md bg-muted" />
  if (status.error) return <Alert variant="destructive"><AlertTitle>更新状态加载失败</AlertTitle><AlertDescription>{status.error.message}</AlertDescription></Alert>
  if (!value?.enabled) return <Alert><AlertTitle>在线更新未启用</AlertTitle><AlertDescription>当前开发构建没有嵌入可信发布公钥。正式签名构建会自动启用。</AlertDescription></Alert>
  const busy = updateBusy(value) || action.isPending
  const progress = updateProgress(value)
  return <div className="space-y-5">
    <section className="rounded-md border p-4"><div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between"><div><div className="flex items-center gap-2"><h2 className="text-base font-medium">SubMerge {value.currentVersion}</h2><Badge variant={value.phase === "failed" ? "destructive" : value.available ? "default" : "secondary"}>{phaseText[value.phase] ?? value.phase}</Badge></div><p className="mt-2 text-sm text-muted-foreground">{value.latestVersion && value.latestVersion !== value.currentVersion ? `最新版本 ${value.latestVersion}` : "当前版本已运行"}{value.checkedAt ? ` · 检查于 ${formatDateTime(value.checkedAt)}` : ""}</p></div><Button type="button" variant="outline" disabled={busy} onClick={() => action.mutate("check")}><RefreshCw className={value.phase === "checking" ? "animate-spin" : ""} />检查更新</Button></div>
      {value.error ? <Alert variant="destructive" className="mt-4"><AlertTitle>更新失败</AlertTitle><AlertDescription>{value.error}</AlertDescription></Alert> : null}
      {value.phase === "downloading" ? <div className="mt-4 space-y-2"><Progress value={progress} /><div className="text-right text-xs text-muted-foreground">{progress}%</div></div> : null}
    </section>
    {value.available ? <section className="rounded-md border p-4"><div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between"><div><h2 className="font-medium">版本 {value.latestVersion}</h2><p className="mt-1 text-sm text-muted-foreground">{value.notes || "查看发布页获取完整变更记录。"}</p>{value.releaseUrl ? <a className="mt-2 inline-block text-sm text-primary underline underline-offset-4" href={value.releaseUrl} target="_blank" rel="noreferrer">查看 GitHub Release</a> : null}</div>{value.phase === "ready" ? <ConfirmAction title={`安装 ${value.latestVersion}`} description="应用将短暂离线，完成二进制与数据库备份后由进程管理器自动重启。" confirmLabel="安装并重启" onConfirm={() => action.mutate("install")} trigger={<Button type="button" disabled={busy}><CheckCircle2 />安装并重启</Button>} /> : <Button type="button" disabled={busy} onClick={() => action.mutate("download")}><Download />下载更新</Button>}</div></section> : null}
    {value.rollbackAvailable ? <section className="flex flex-col gap-3 rounded-md border p-4 sm:flex-row sm:items-center sm:justify-between"><div><h2 className="font-medium">回滚到 {value.rollbackVersion}</h2><p className="mt-1 text-sm text-muted-foreground">恢复上次更新前的二进制和数据库备份。</p></div><ConfirmAction title={`回滚到 ${value.rollbackVersion}`} description="当前应用和数据库会被替换，随后自动重启。" confirmLabel="回滚并重启" destructive onConfirm={() => action.mutate("rollback")} trigger={<Button type="button" variant="outline" disabled={busy}><RotateCcw />回滚</Button>} /></section> : null}
  </div>
}
