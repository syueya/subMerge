import { useQuery } from "@tanstack/react-query"
import { Download } from "lucide-react"
import { useEffect } from "react"
import { Link } from "react-router-dom"

import { Button } from "@/components/ui/button"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { api } from "@/lib/api"
import { queryKeys } from "@/lib/query-keys"
import type { UpdateStatus } from "@/lib/types"

const checkedKey = "submerge-update-checked"

export function UpdateIndicator() {
  const status = useQuery({
    queryKey: queryKeys.update,
    queryFn: () => api.get<UpdateStatus>("/update/status"),
    refetchInterval: 60_000,
  })
  useEffect(() => {
    if (!status.data?.enabled || window.sessionStorage.getItem(checkedKey)) return
    window.sessionStorage.setItem(checkedKey, "1")
    const timer = window.setTimeout(() => void api.post<UpdateStatus>("/update/check").then(() => status.refetch()).catch(() => undefined), 2_000)
    return () => window.clearTimeout(timer)
  }, [status])
  if (!status.data?.enabled) return null
  const available = status.data.available || status.data.phase === "ready"
  return <Tooltip><TooltipTrigger asChild><Button variant="ghost" size="icon" asChild className={available ? "text-primary" : undefined}><Link to="/main/settings/setting-settings?tab=update" aria-label={available ? `发现新版本 ${status.data.latestVersion}` : "应用更新"}><Download />{available ? <span className="absolute right-1.5 top-1.5 size-2 rounded-full bg-primary" /> : null}</Link></Button></TooltipTrigger><TooltipContent>{available ? `发现 ${status.data.latestVersion}` : "应用更新"}</TooltipContent></Tooltip>
}
