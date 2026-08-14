import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { RotateCcw, Save } from "lucide-react"
import { useEffect, useState, type FormEvent } from "react"
import { useSearchParams } from "react-router-dom"
import { toast } from "sonner"

import { ConfirmAction } from "@/components/confirm-action"
import { PageHeader } from "@/components/page-header"
import { ResourceState } from "@/components/resource-state"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Field, FieldDescription, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { ApiKeysPanel } from "@/features/api-keys/api-keys-panel"
import { UpdatePanel } from "@/features/app-update/update-panel"
import { api } from "@/lib/api"
import { queryKeys } from "@/lib/query-keys"

type SystemSettings = {
  sourceFetchUA: string
  sourceFetchTimeout: number
  sourceMaxBytes: number
  refreshInterval: number
  geoipUrl: string
  geositeUrl: string
  geodbUrl: string
  geoasnUrl: string
  ipGeoUrl: string
  ipGeoTimeout: number
  logOutput: "console" | "file" | "both" | "none"
  debugLogging: boolean
  logRetentionDays: number
  proxyEnabled: boolean
  proxyConfigured: boolean
  proxyUrl: string
  proxyMaskedUrl?: string
  publicBaseUrl: string
  trustedProxies: string
  cookieSecure: boolean
}
type SettingsView = { settings: SystemSettings; source: Record<string, string>; override: Record<string, boolean>; restartRequired: boolean }

export function SettingsPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const client = useQueryClient()
  const requestedTab = searchParams.get("tab")
  const activeTab = ["deployment", "fetch", "api-keys", "update"].includes(requestedTab ?? "") ? requestedTab! : "deployment"
  const [form, setForm] = useState<SystemSettings | null>(null)
  const settings = useQuery({ queryKey: queryKeys.settings, queryFn: () => api.get<SettingsView>("/system-settings") })
  useEffect(() => { if (settings.data) setForm({ ...settings.data.settings }) }, [settings.data])
  const save = useMutation({ mutationFn: () => api.put<SettingsView>("/system-settings", form), onSuccess: async (view) => { setForm({ ...view.settings }); toast.success(view.restartRequired ? "设置已保存，可信代理变更将在重启后生效" : "设置已保存"); await client.invalidateQueries({ queryKey: queryKeys.settings }) }, onError: (e: Error) => toast.error(e.message) })
  const reset = useMutation({ mutationFn: () => api.post<SettingsView>("/system-settings/reset"), onSuccess: async (view) => { setForm({ ...view.settings }); toast.success(view.restartRequired ? "已恢复默认设置，部分变更将在重启后生效" : "已恢复默认设置"); await client.invalidateQueries({ queryKey: queryKeys.settings }) }, onError: (e: Error) => toast.error(e.message) })
  const update = <K extends keyof SystemSettings>(key: K, value: SystemSettings[K]) => form && setForm({ ...form, [key]: value })
  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!form) return
    try { const publicURL = new URL(form.publicBaseUrl); if (!["http:", "https:"].includes(publicURL.protocol) || publicURL.search || publicURL.hash) throw new Error() } catch { return toast.error("公开访问地址必须是无查询和锚点的 HTTP/HTTPS URL") }
    if (form.proxyEnabled && form.proxyUrl && !/^(https?|socks5h?):\/\//i.test(form.proxyUrl)) return toast.error("代理地址协议必须是 HTTP、HTTPS、SOCKS5 或 SOCKS5H")
    if (form.sourceFetchTimeout < 1 || form.ipGeoTimeout < 1 || form.refreshInterval < 1 || form.refreshInterval > 720) return toast.error("超时和刷新间隔超出允许范围")
    save.mutate()
  }
  const settingsActions = activeTab === "deployment" || activeTab === "fetch" ? <><ConfirmAction title="恢复默认设置" description="所有网页保存的系统设置都会被清除并恢复默认值。" confirmLabel="恢复默认" destructive onConfirm={() => reset.mutate()} trigger={<Button type="button" variant="outline"><RotateCcw />恢复默认</Button>} /><Button type="submit" disabled={!form || save.isPending}><Save />{save.isPending ? "保存中..." : "保存设置"}</Button></> : undefined

  return <form className="flex flex-col gap-4 md:gap-6" onSubmit={submit}><PageHeader title="系统设置" description="管理部署参数、数据获取、API 密钥和应用更新。" actions={settingsActions} />
    <Tabs className="max-w-5xl" value={activeTab} onValueChange={(value) => setSearchParams(value === "deployment" ? {} : { tab: value }, { replace: true })}><TabsList variant="line" className="w-full justify-start border-b pb-1"><TabsTrigger className="flex-none px-3" value="deployment">部署与安全</TabsTrigger><TabsTrigger className="flex-none px-3" value="fetch">数据获取</TabsTrigger><TabsTrigger className="flex-none px-3" value="api-keys">API 密钥</TabsTrigger><TabsTrigger className="flex-none px-3" value="update">应用更新</TabsTrigger></TabsList>
    {activeTab === "deployment" || activeTab === "fetch" ? <ResourceState pending={settings.isPending} error={settings.error} onRetry={() => void settings.refetch()}>{form ? <>
      <TabsContent value="deployment" className="mt-5 space-y-5"><section className="grid gap-5 rounded-md border p-4 sm:grid-cols-2"><Field><FieldLabel htmlFor="public-url">公开访问地址</FieldLabel><Input id="public-url" type="url" value={form.publicBaseUrl} onChange={(e) => update("publicBaseUrl", e.target.value)} required /><FieldDescription>用于生成订阅链接，可包含端口和路径。</FieldDescription></Field><Field><FieldLabel htmlFor="trusted-proxies">可信代理</FieldLabel><Input id="trusted-proxies" value={form.trustedProxies} onChange={(e) => update("trustedProxies", e.target.value)} placeholder="10.0.0.0/8, 172.16.0.1" /><FieldDescription>逗号分隔 IP/CIDR，修改后需重启。</FieldDescription></Field><Field orientation="horizontal" className="rounded-md border p-3"><div className="flex-1"><FieldLabel htmlFor="cookie-secure">Secure 会话 Cookie</FieldLabel><FieldDescription>仅 HTTPS 部署开启。</FieldDescription></div><Switch id="cookie-secure" checked={form.cookieSecure} onCheckedChange={(value) => update("cookieSecure", value)} /></Field><Field orientation="horizontal" className="rounded-md border p-3"><div className="flex-1"><FieldLabel htmlFor="debug-log">调试日志</FieldLabel><FieldDescription>输出节点过滤和地区识别细节。</FieldDescription></div><Switch id="debug-log" checked={form.debugLogging} onCheckedChange={(value) => update("debugLogging", value)} /></Field></section>
      <section className="grid gap-5 rounded-md border p-4 sm:grid-cols-2"><Field><FieldLabel>日志输出</FieldLabel><Select value={form.logOutput} onValueChange={(value: SystemSettings["logOutput"]) => update("logOutput", value)}><SelectTrigger className="w-full"><SelectValue /></SelectTrigger><SelectContent><SelectItem value="both">控制台和文件</SelectItem><SelectItem value="console">仅控制台</SelectItem><SelectItem value="file">仅文件</SelectItem><SelectItem value="none">关闭</SelectItem></SelectContent></Select></Field><Field><FieldLabel htmlFor="log-retention">日志保留天数</FieldLabel><Input id="log-retention" type="number" min={0} max={365} value={form.logRetentionDays} onChange={(e) => update("logRetentionDays", Number(e.target.value))} /><FieldDescription>0 表示不自动清理。</FieldDescription></Field></section>
      {settings.data?.restartRequired ? <Alert><AlertTitle>需要重启</AlertTitle><AlertDescription>可信代理配置将在应用重启后生效。</AlertDescription></Alert> : null}</TabsContent>
      <TabsContent value="fetch" className="mt-5 space-y-5"><section className="space-y-4 rounded-md border p-4"><div className="flex items-center justify-between"><div><h2 className="font-medium">出站代理</h2><p className="mt-1 text-sm text-muted-foreground">用于订阅拉取、Geo 更新与 IP 位置查询。</p></div><Switch checked={form.proxyEnabled} onCheckedChange={(value) => update("proxyEnabled", value)} /></div><Field><FieldLabel htmlFor="proxy-url">代理地址</FieldLabel><Input id="proxy-url" value={form.proxyUrl} onChange={(e) => update("proxyUrl", e.target.value)} placeholder={form.proxyMaskedUrl || "socks5://127.0.0.1:1080"} /><FieldDescription>留空时关闭代理；包含认证信息的地址会加密存储。</FieldDescription></Field></section>
      <section className="grid gap-5 rounded-md border p-4 sm:grid-cols-2"><Field><FieldLabel htmlFor="fetch-ua">订阅请求 User-Agent</FieldLabel><Input id="fetch-ua" value={form.sourceFetchUA} onChange={(e) => update("sourceFetchUA", e.target.value)} required /></Field><Field><FieldLabel htmlFor="fetch-timeout">订阅超时（秒）</FieldLabel><Input id="fetch-timeout" type="number" min={1} value={form.sourceFetchTimeout} onChange={(e) => update("sourceFetchTimeout", Number(e.target.value))} /></Field><Field><FieldLabel htmlFor="fetch-size">最大响应（MB）</FieldLabel><Input id="fetch-size" type="number" min={1} value={Math.max(1, Math.round(form.sourceMaxBytes / 1048576))} onChange={(e) => update("sourceMaxBytes", Number(e.target.value) * 1048576)} /></Field><Field><FieldLabel htmlFor="refresh-hours">自动拉取间隔（小时）</FieldLabel><Input id="refresh-hours" type="number" min={1} max={720} value={form.refreshInterval} onChange={(e) => update("refreshInterval", Number(e.target.value))} /></Field></section>
      <section className="grid gap-5 rounded-md border p-4 sm:grid-cols-2"><Field><FieldLabel htmlFor="ipgeo-url">IP 位置 API</FieldLabel><Input id="ipgeo-url" value={form.ipGeoUrl} onChange={(e) => update("ipGeoUrl", e.target.value)} required /></Field><Field><FieldLabel htmlFor="ipgeo-timeout">IP 查询超时（秒）</FieldLabel><Input id="ipgeo-timeout" type="number" min={1} value={form.ipGeoTimeout} onChange={(e) => update("ipGeoTimeout", Number(e.target.value))} /></Field>{[["geoipUrl", "GeoIP 数据"], ["geositeUrl", "GeoSite 数据"], ["geodbUrl", "MetaDB 数据"], ["geoasnUrl", "ASN 数据"]].map(([key, label]) => <Field key={key}><FieldLabel htmlFor={key}>{label}</FieldLabel><Input id={key} type="url" value={form[key as keyof SystemSettings] as string} onChange={(e) => update(key as keyof SystemSettings, e.target.value as never)} required pattern="https://.*" /><FieldDescription>仅允许 HTTPS。</FieldDescription></Field>)}</section></TabsContent>
    </> : null}</ResourceState> : null}
      <TabsContent value="api-keys" className="mt-5"><ApiKeysPanel /></TabsContent>
      <TabsContent value="update" className="mt-5"><UpdatePanel /></TabsContent>
    </Tabs>
  </form>
}
