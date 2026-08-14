import type { FormEvent, ReactNode } from "react"

import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"

export function FormDialog({
  open,
  onOpenChange,
  title,
  description,
  busy,
  submitLabel = "保存",
  children,
  onSubmit,
  className,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: string
  description?: string
  busy?: boolean
  submitLabel?: string
  children: ReactNode
  onSubmit: (event: FormEvent<HTMLFormElement>) => void
  className?: string
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className={className ?? "sm:max-w-lg"}>
        <DialogHeader><DialogTitle>{title}</DialogTitle>{description ? <DialogDescription>{description}</DialogDescription> : null}</DialogHeader>
        <form className="contents" onSubmit={onSubmit}>
          <div className="grid max-h-[65vh] gap-4 overflow-y-auto px-0.5">{children}</div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>取消</Button>
            <Button type="submit" disabled={busy}>{busy ? "处理中..." : submitLabel}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
