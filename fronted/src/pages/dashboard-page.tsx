import { useQuery, useQueryClient } from "@tanstack/react-query"
import {
  ArrowRight,
  Boxes,
  CheckCircle2,
  CloudDownload,
  KeyRound,
  ListChecks,
  RefreshCw,
  Rocket,
  TriangleAlert,
} from "lucide-react"
import { Link } from "react-router-dom"
import { Bar, BarChart, CartesianGrid, ResponsiveContainer, Tooltip as RechartsTooltip, XAxis } from "recharts"

import { PageHeader } from "@/components/page-header"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { api } from "@/lib/api"
import { formatDateTime } from "@/lib/types"
import type { DraftStatus, ListResponse, ProxyGroup, ProxyNode, Release, Rule, ShareToken, SubscriptionSource } from "@/lib/types"

const dashboardQuery = ["dashboard"] as const

const workflow = [
  { number: "01", title: "订阅管理", description: "添加远程订阅或手工节点，并检查拉取与节点状态。", path: "/main/sources", action: "管理来源", icon: CloudDownload },
  { number: "02", title: "策略分组", description: "组织出口成员、测速方式与故障转移顺序。", path: "/main/groups", action: "配置分组", icon: Boxes },
  { number: "03", title: "分流规则", description: "按业务分类维护匹配顺序，并验证目标出口。", path: "/main/rules", action: "编辑规则", icon: ListChecks },
  { number: "04", title: "版本发布", description: "检查草稿变更，发布或回滚可供客户端使用的配置。", path: "/main/releases", action: "前往发布", icon: Rocket },
  { number: "05", title: "订阅链接", description: "为客户端生成独立链接并限制来源与策略组。", path: "/main/tokens", action: "管理链接", icon: KeyRound },
] as const

interface DashboardData {
  sources: SubscriptionSource[]
  proxies: ProxyNode[]
  groups: ProxyGroup[]
  rules: Rule[]
  releases: Release[]
  tokens: ShareToken[]
  draft: DraftStatus | null
  errors: string[]
}

async function loadDashboard(): Promise<DashboardData> {
  const errors: string[] = []
  const settle = async <T,>(label: string, request: Promise<T>, fallback: T) => request.catch((error: unknown) => {
    errors.push(`${label}：${error instanceof Error ? error.message : "加载失败"}`)
    return fallback
  })
  const [sources, proxies, groups, rules, releases, tokens, draft] = await Promise.all([
    settle("订阅源", api.get<ListResponse<SubscriptionSource>>("/sources"), { items: [] }),
    settle("节点", api.get<ListResponse<ProxyNode>>("/proxies"), { items: [] }),
    settle("策略组", api.get<ListResponse<ProxyGroup>>("/groups"), { items: [] }),
    settle("规则", api.get<ListResponse<Rule>>("/rules"), { items: [] }),
    settle("发布记录", api.get<ListResponse<Release>>("/releases"), { items: [] }),
    settle("订阅链接", api.get<ListResponse<ShareToken>>("/tokens"), { items: [] }),
    settle("草稿状态", api.get<DraftStatus>("/releases/draft-status"), null),
  ])
  return { sources: sources.items, proxies: proxies.items, groups: groups.items, rules: rules.items, releases: releases.items, tokens: tokens.items, draft, errors }
}

export function DashboardPage() {
  const queryClient = useQueryClient()
  const { data, isPending, isFetching } = useQuery({ queryKey: dashboardQuery, queryFn: loadDashboard })
  const latest = data?.releases.find((release) => release.status === "published") ?? data?.releases[0]
  const enabledSources = data?.sources.filter((item) => item.enabled).length ?? 0
  const healthySources = data?.sources.filter((item) => item.enabled && item.refreshStatus !== "failed").length ?? 0
  const activeTokens = data?.tokens.filter((item) => item.status === "active").length ?? 0
  const enabledRules = data?.rules.filter((item) => item.enabled).length ?? 0
  const chartData = [
    { name: "节点", value: data?.proxies.length ?? 0 },
    { name: "策略组", value: data?.groups.length ?? 0 },
    { name: "规则", value: data?.rules.length ?? 0 },
    { name: "令牌", value: data?.tokens.length ?? 0 },
  ]
  const metrics = [
    { label: "可用节点", value: data?.proxies.filter((item) => item.enabled && item.ok).length ?? 0, total: `共 ${data?.proxies.length ?? 0} 个节点`, icon: CloudDownload },
    { label: "订阅源", value: enabledSources, total: `${healthySources}/${enabledSources || 0} 个运行正常`, icon: Boxes },
    { label: "已启用规则", value: enabledRules, total: `共 ${data?.rules.length ?? 0} 条规则`, icon: ListChecks },
    { label: "有效订阅链接", value: activeTokens, total: `累计 ${data?.tokens.reduce((sum, item) => sum + item.accessCount, 0) ?? 0} 次访问`, icon: KeyRound },
  ]

  return (
    <div className="flex flex-col gap-4 md:gap-6">
      <PageHeader
        title="概览"
        description="查看当前配置规模、发布状态和最近变更。"
        actions={<><Button variant="outline" size="icon" aria-label="刷新状态" disabled={isFetching} onClick={() => void queryClient.invalidateQueries({ queryKey: dashboardQuery })}><RefreshCw className={isFetching ? "animate-spin" : ""} /></Button><Button asChild><Link to="/main/releases"><Rocket />发布配置</Link></Button></>}
      />

      {data?.errors.length ? <Alert variant="destructive"><TriangleAlert /><AlertTitle>部分数据加载失败</AlertTitle><AlertDescription>{data.errors.join("；")}</AlertDescription></Alert> : null}

      <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {metrics.map((metric) => (
          <Card key={metric.label}>
            <CardHeader className="flex-row items-center justify-between pb-1">
              <CardDescription>{metric.label}</CardDescription>
              <metric.icon className="size-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              {isPending ? <Skeleton className="h-9 w-20" /> : <div className="text-3xl font-semibold tabular-nums">{metric.value}</div>}
              <p className="mt-3 text-xs text-muted-foreground">{metric.total}</p>
            </CardContent>
          </Card>
        ))}
      </section>

      <section className="grid gap-4 xl:grid-cols-[minmax(0,1.65fr)_minmax(300px,0.75fr)]">
        <Card>
          <CardHeader>
            <CardTitle>配置规模</CardTitle>
            <CardDescription>当前服务实例中的主要配置对象</CardDescription>
          </CardHeader>
          <CardContent className="h-72 pt-2">
            {isPending ? <Skeleton className="h-full w-full" /> : (
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={chartData} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
                  <CartesianGrid vertical={false} stroke="var(--border)" />
                  <XAxis dataKey="name" tickLine={false} axisLine={false} tick={{ fill: "var(--muted-foreground)", fontSize: 12 }} />
                  <RechartsTooltip cursor={{ fill: "var(--muted)" }} contentStyle={{ borderRadius: 10, border: "1px solid var(--border)", background: "var(--popover)", color: "var(--popover-foreground)" }} />
                  <Bar dataKey="value" name="数量" fill="var(--primary)" radius={[4, 4, 0, 0]} maxBarSize={72} />
                </BarChart>
              </ResponsiveContainer>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <div className="flex items-start justify-between gap-3">
              <div><CardTitle>发布状态</CardTitle><CardDescription>草稿与线上配置</CardDescription></div>
              {data?.draft?.dirty ? <Badge variant="secondary"><TriangleAlert />待发布</Badge> : <Badge variant="outline"><CheckCircle2 />已同步</Badge>}
            </div>
          </CardHeader>
          <CardContent className="space-y-6">
            <div>
              <p className="text-sm text-muted-foreground">当前版本</p>
              <div className="mt-2 text-3xl font-semibold">{latest ? `V${latest.version}` : "V0.0"}</div>
              <p className="mt-2 text-sm leading-6 text-muted-foreground">{data?.draft?.buildError || (data?.draft?.dirty ? "配置已有更改，发布后才会对客户端生效。" : data?.draft?.hasPublished ? "草稿与当前发布版本一致。" : "尚未发布第一个配置版本。")}</p>
            </div>
            <div className="grid grid-cols-2 gap-3 border-y py-4 text-sm">
              <div><p className="text-muted-foreground">策略组</p><p className="mt-1 font-medium tabular-nums">{data?.groups.length ?? 0}</p></div>
              <div><p className="text-muted-foreground">规则</p><p className="mt-1 font-medium tabular-nums">{data?.rules.length ?? 0}</p></div>
            </div>
            <Button variant="outline" className="w-full justify-between" asChild><Link to="/main/releases">查看发布中心<ArrowRight /></Link></Button>
          </CardContent>
        </Card>
      </section>

      <Card>
        <CardHeader className="flex-row items-start justify-between">
          <div><CardTitle>最近发布</CardTitle><CardDescription>最新的配置版本与节点规模</CardDescription></div>
          <Button variant="ghost" size="sm" asChild><Link to="/main/releases">查看全部<ArrowRight /></Link></Button>
        </CardHeader>
        <CardContent className="px-0">
          <Table>
            <TableHeader><TableRow><TableHead className="pl-4">版本</TableHead><TableHead>状态</TableHead><TableHead>节点</TableHead><TableHead>规则</TableHead><TableHead>发布时间</TableHead><TableHead className="pr-4 text-right">发布人</TableHead></TableRow></TableHeader>
            <TableBody>
              {(data?.releases ?? []).slice(0, 5).map((release) => <TableRow key={release.id}><TableCell className="pl-4 font-medium">V{release.version}</TableCell><TableCell><Badge variant={release.status === "published" ? "outline" : "secondary"}>{release.status === "published" ? "已发布" : release.status === "rolled_back" ? "已回滚" : "草稿"}</Badge></TableCell><TableCell>{release.proxyCount}</TableCell><TableCell>{release.ruleCount}</TableCell><TableCell className="text-muted-foreground">{formatDateTime(release.publishedAt ?? release.createdAt)}</TableCell><TableCell className="pr-4 text-right">{release.createdBy || "-"}</TableCell></TableRow>)}
              {!isPending && !data?.releases.length ? <TableRow><TableCell colSpan={6} className="h-28 text-center text-muted-foreground">还没有发布记录</TableCell></TableRow> : null}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>配置流程</CardTitle>
          <CardDescription>从来源整理到客户端下发的完整工作流</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-px overflow-hidden rounded-md border bg-border p-0 sm:grid-cols-2 xl:grid-cols-5">
          {workflow.map((step) => (
            <Link key={step.number} to={step.path} className="group flex min-h-40 flex-col bg-card p-4 transition-colors hover:bg-accent/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset">
              <div className="flex items-center justify-between">
                <span className="font-mono text-xs font-medium text-primary">{step.number}</span>
                <step.icon className="size-4 text-muted-foreground transition-colors group-hover:text-primary" />
              </div>
              <h3 className="mt-5 font-medium">{step.title}</h3>
              <p className="mt-2 flex-1 text-xs leading-5 text-muted-foreground">{step.description}</p>
              <span className="mt-4 flex items-center gap-1 text-xs font-medium text-primary">{step.action}<ArrowRight className="size-3.5 transition-transform group-hover:translate-x-0.5" /></span>
            </Link>
          ))}
        </CardContent>
      </Card>
    </div>
  )
}
