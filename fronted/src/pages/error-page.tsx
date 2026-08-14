import { CircleAlert } from "lucide-react"
import { Link } from "react-router-dom"

import { Button } from "@/components/ui/button"

export function ErrorPage() {
  return (
    <main className="grid min-h-svh place-items-center p-6 text-center">
      <div><CircleAlert className="mx-auto size-10 text-destructive" /><h1 className="mt-4 text-xl font-semibold">页面不存在</h1><p className="mt-2 text-sm text-muted-foreground">请求的地址无法匹配到 SubMerge 页面。</p><Button asChild className="mt-5"><Link to="/main/dashboard">返回概览</Link></Button></div>
    </main>
  )
}
