import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Copy, Edit3, Eye, KeyRound, RefreshCw, ShieldX, Trash2 } from "lucide-react"
import { useState, type FormEvent } from "react"
import { toast } from "sonner"

import { ConfirmAction } from "@/components/confirm-action"
import { FormDialog } from "@/components/form-dialog"
import { ResourceState } from "@/components/resource-state"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Field, FieldDescription, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Switch } from "@/components/ui/switch"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Textarea } from "@/components/ui/textarea"
import { api } from "@/lib/api"
import { queryKeys } from "@/lib/query-keys"
import type { ApiKey, ListResponse } from "@/lib/types"
import { formatDateTime } from "@/lib/types"

type KeyForm = { id?: number; name: string; note: string; scopes: string[] }

const scopeLabels: Record<string, string> = {
  "*": "全部",
  read: "读取",
  write: "写入",
  publish: "发布",
}

export function ApiKeysPanel() {
  const client = useQueryClient()
  const [editingKey, setEditingKey] = useState<KeyForm | null>(null)
  const [secret, setSecret] = useState<{ name: string; key: string } | null>(null)
  const keys = useQuery({ queryKey: queryKeys.apiKeys, queryFn: () => api.get<ListResponse<ApiKey>>("/apikeys") })
  const invalidateKeys = () => client.invalidateQueries({ queryKey: queryKeys.apiKeys })
  const saveKey = useMutation({
    mutationFn: (form: KeyForm) => {
      const body = { name: form.name.trim(), scopes: form.scopes, note: form.note.trim() }
      return form.id ? api.put<ApiKey>(`/apikeys/${form.id}`, body) : api.post<ApiKey>("/apikeys", body)
    },
    onSuccess: async (item) => {
      toast.success("API 密钥已保存")
      setEditingKey(null)
      if (item.key) setSecret({ name: item.name, key: item.key })
      await invalidateKeys()
    },
    onError: (error: Error) => toast.error(error.message),
  })
  const keyAction = useMutation({
    mutationFn: ({ item, kind }: { item: ApiKey; kind: "enable" | "disable" | "secret" | "revoke" | "regenerate" | "delete" }) => {
      if (kind === "delete") return api.delete(`/apikeys/${item.id}`)
      if (kind === "secret") return api.get<{ id: number; key: string }>(`/apikeys/${item.id}/secret`)
      if (kind === "revoke" || kind === "regenerate") return api.post<ApiKey>(`/apikeys/${item.id}/${kind}`)
      return api.put<ApiKey>(`/apikeys/${item.id}`, { status: kind === "enable" ? "active" : "disabled" })
    },
    onSuccess: async (result, values) => {
      if (values.kind === "secret") setSecret({ name: values.item.name, key: (result as { key: string }).key })
      else if ((result as ApiKey)?.key) setSecret({ name: values.item.name, key: (result as ApiKey).key! })
      else toast.success("操作成功")
      await invalidateKeys()
    },
    onError: (error: Error) => toast.error(error.message),
  })

  const selectScope = (scope: string, checked: boolean) => {
    if (!editingKey) return
    if (checked && scope === "*") {
      setEditingKey({ ...editingKey, scopes: ["*"] })
      return
    }
    const scopes = editingKey.scopes.filter((item) => item !== "*" && item !== scope)
    if (checked) scopes.push(scope)
    setEditingKey({ ...editingKey, scopes })
  }

  return (
    <div className="space-y-4">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h2 className="font-medium">API 密钥</h2>
          <p className="mt-1 text-sm text-muted-foreground">管理用于自动化调用和外部集成的访问凭据。</p>
        </div>
        <Button type="button" onClick={() => setEditingKey({ name: "", note: "", scopes: ["read"] })}><KeyRound />创建密钥</Button>
      </div>

      <ResourceState pending={keys.isPending} error={keys.error} empty={!keys.data?.items.length} onRetry={() => void keys.refetch()}>
        <div className="overflow-x-auto rounded-lg border bg-card">
          <Table>
            <TableHeader><TableRow><TableHead>名称</TableHead><TableHead>密钥</TableHead><TableHead>权限</TableHead><TableHead>备注</TableHead><TableHead>最近使用</TableHead><TableHead className="w-56" /></TableRow></TableHeader>
            <TableBody>{keys.data?.items.map((item) => (
              <TableRow key={item.id}>
                <TableCell><div className="flex items-center gap-2"><span className="font-medium">{item.name}</span><Badge variant={item.status === "active" ? "secondary" : item.status === "revoked" ? "destructive" : "outline"}>{item.status}</Badge></div><div className="text-xs text-muted-foreground">{item.createdBy} · {formatDateTime(item.createdAt)}</div></TableCell>
                <TableCell><code className="text-xs">{item.keyMasked}</code></TableCell>
                <TableCell>{item.scopes.map((scope) => scopeLabels[scope] ?? scope).join("、")}</TableCell>
                <TableCell className="max-w-48 truncate" title={item.note}>{item.note || "-"}</TableCell>
                <TableCell>{formatDateTime(item.lastUsedAt)}</TableCell>
                <TableCell><div className="flex justify-end">
                  <Button type="button" variant="ghost" size="icon" aria-label="查看密钥" onClick={() => keyAction.mutate({ item, kind: "secret" })}><Eye /></Button>
                  <Button type="button" variant="ghost" size="icon" aria-label="编辑" onClick={() => setEditingKey({ id: item.id, name: item.name, note: item.note ?? "", scopes: item.scopes })}><Edit3 /></Button>
                  {item.status !== "revoked" ? <Button type="button" variant="ghost" size="icon" aria-label={item.status === "active" ? "停用" : "启用"} onClick={() => keyAction.mutate({ item, kind: item.status === "active" ? "disable" : "enable" })}><Switch checked={item.status === "active"} className="pointer-events-none scale-75" /></Button> : null}
                  <ConfirmAction title="作废 API 密钥" description="现有密钥会立即失效，记录保留。" destructive confirmLabel="作废" onConfirm={() => keyAction.mutate({ item, kind: "revoke" })} trigger={<Button type="button" variant="ghost" size="icon" aria-label="作废"><ShieldX /></Button>} />
                  <ConfirmAction title="重新生成 API 密钥" description="旧密钥立即失效，所有调用方都需要更新。" confirmLabel="重新生成" onConfirm={() => keyAction.mutate({ item, kind: "regenerate" })} trigger={<Button type="button" variant="ghost" size="icon" aria-label="重新生成"><RefreshCw /></Button>} />
                  <ConfirmAction title="永久删除 API 密钥" description="密钥和使用记录将无法恢复。" destructive confirmLabel="删除" onConfirm={() => keyAction.mutate({ item, kind: "delete" })} trigger={<Button type="button" variant="ghost" size="icon" aria-label="删除"><Trash2 /></Button>} />
                </div></TableCell>
              </TableRow>
            ))}</TableBody>
          </Table>
        </div>
      </ResourceState>

      <FormDialog open={!!editingKey} onOpenChange={(open) => !open && setEditingKey(null)} title={editingKey?.id ? "编辑 API 密钥" : "创建 API 密钥"} busy={saveKey.isPending} onSubmit={(event: FormEvent<HTMLFormElement>) => { event.preventDefault(); if (!editingKey?.name.trim()) return toast.error("请填写名称"); if (!editingKey.scopes.length) return toast.error("至少选择一个权限"); saveKey.mutate(editingKey) }}>
        {editingKey ? <><Field><FieldLabel htmlFor="key-name">名称</FieldLabel><Input id="key-name" value={editingKey.name} maxLength={128} onChange={(event) => setEditingKey({ ...editingKey, name: event.target.value })} required /></Field><Field><FieldLabel>权限</FieldLabel><div className="grid grid-cols-2 gap-2">{["*", "read", "write", "publish"].map((scope) => <label key={scope} className="flex items-center gap-2 rounded-md border p-2 text-sm"><Checkbox checked={editingKey.scopes.includes(scope)} onCheckedChange={(checked) => selectScope(scope, !!checked)} />{scopeLabels[scope]}</label>)}</div><FieldDescription>“全部”与读取、写入、发布细分权限互斥。</FieldDescription></Field><Field><FieldLabel htmlFor="key-note">备注</FieldLabel><Textarea id="key-note" value={editingKey.note} maxLength={512} onChange={(event) => setEditingKey({ ...editingKey, note: event.target.value })} /></Field></> : null}
      </FormDialog>

      <Dialog open={!!secret} onOpenChange={(open) => !open && setSecret(null)}><DialogContent><DialogHeader><DialogTitle>{secret?.name}</DialogTitle><DialogDescription>通过 HTTP `Authorization: Bearer &lt;key&gt;` 使用，可随时在列表再次查看。</DialogDescription></DialogHeader><div className="flex items-center gap-2"><Input readOnly value={secret?.key ?? ""} className="font-mono text-xs" /><Button type="button" size="icon" aria-label="复制密钥" onClick={() => void navigator.clipboard.writeText(secret?.key ?? "").then(() => toast.success("已复制"))}><Copy /></Button></div></DialogContent></Dialog>
    </div>
  )
}
