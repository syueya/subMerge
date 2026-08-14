import type { ReactNode } from "react"

import { cn } from "@/lib/utils"

export function DataPanel({ title, description, actions, children, className }: { title: string; description?: ReactNode; actions?: ReactNode; children: ReactNode; className?: string }) {
  return (
    <section className={cn("overflow-hidden rounded-lg border bg-card", className)}>
      <header className="flex min-h-14 flex-col gap-2 border-b px-4 py-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="min-w-0">
          <h2 className="text-sm font-medium">{title}</h2>
          {description ? <div className="mt-0.5 text-xs leading-5 text-muted-foreground">{description}</div> : null}
        </div>
        {actions ? <div className="flex shrink-0 flex-wrap items-center gap-2">{actions}</div> : null}
      </header>
      <div className="overflow-x-auto">{children}</div>
    </section>
  )
}
