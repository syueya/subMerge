import { Construction } from "lucide-react"

import { PageHeader } from "@/components/page-header"
import { Card, CardContent } from "@/components/ui/card"

export function PlaceholderPage({ title }: { title: string }) {
  return (
    <div className="space-y-6">
      <PageHeader title={title} />
      <Card><CardContent className="flex min-h-64 flex-col items-center justify-center gap-3 text-center text-muted-foreground"><Construction className="size-8" /><p className="text-sm">此模块正在迁移到 React。</p></CardContent></Card>
    </div>
  )
}
