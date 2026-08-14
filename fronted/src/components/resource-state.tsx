import { AlertCircle, Inbox, LoaderCircle } from "lucide-react"
import type { ReactNode } from "react"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty"

export function ResourceState({
  pending,
  error,
  empty,
  onRetry,
  children,
}: {
  pending: boolean
  error?: Error | null
  empty?: boolean
  onRetry?: () => void
  children: ReactNode
}) {
  if (pending) return <div className="grid min-h-52 place-items-center rounded-lg border bg-card"><LoaderCircle className="size-5 animate-spin text-muted-foreground" /></div>
  if (error) return (
    <Alert variant="destructive">
      <AlertCircle />
      <AlertTitle>加载失败</AlertTitle>
      <AlertDescription className="flex items-center justify-between gap-3"><span>{error.message}</span>{onRetry ? <Button size="sm" variant="outline" onClick={onRetry}>重试</Button> : null}</AlertDescription>
    </Alert>
  )
  if (empty) return <Empty className="min-h-52 rounded-lg border bg-card"><EmptyHeader><EmptyMedia variant="icon"><Inbox /></EmptyMedia><EmptyTitle>暂无数据</EmptyTitle><EmptyDescription>当前还没有可显示的记录。</EmptyDescription></EmptyHeader></Empty>
  return children
}
