import { useMutation } from "@tanstack/react-query"
import { useState } from "react"
import { useNavigate } from "react-router-dom"
import { toast } from "sonner"

import { PageHeader } from "@/components/page-header"
import { Button } from "@/components/ui/button"
import { Field, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { useAuth } from "@/features/auth/auth-context"
import { api } from "@/lib/api"
import { avatarDataURL, avatarURLs } from "@/lib/avatars"
import type { AdminUser } from "@/lib/types"

export function AccountPage() {
  const { user, setUser, logout } = useAuth()
  const navigate = useNavigate()
  const [profile, setProfile] = useState({ username: user?.username ?? "", displayName: user?.displayName ?? "" })
  const [password, setPassword] = useState({ oldPassword: "", newPassword: "", confirm: "" })
  const profileMutation = useMutation({
    mutationFn: (body: { username?: string; displayName?: string; avatar?: string }) => api.put<{ user: AdminUser }>("/auth/profile", body),
    onSuccess: ({ user: next }) => {
      setUser(next)
      setProfile({ username: next.username, displayName: next.displayName })
      toast.success("账户信息已更新")
    },
    onError: (error: Error) => toast.error(error.message),
  })
  const passwordMutation = useMutation({
    mutationFn: () => api.post("/auth/password", { oldPassword: password.oldPassword, newPassword: password.newPassword }),
    onSuccess: async () => {
      toast.success("密码已修改，请重新登录")
      await logout()
      navigate("/auth/login", { replace: true })
    },
    onError: (error: Error) => toast.error(error.message),
  })

  return (
    <div className="flex flex-col gap-4 md:gap-6">
      <PageHeader title="我的信息" description="管理当前管理员的个人资料和登录密码。" />
      <div className="grid max-w-6xl gap-5 xl:grid-cols-[20rem_minmax(0,1fr)]">
        <section className="space-y-4 rounded-lg border bg-card p-5">
          <div><h2 className="font-medium">头像</h2><p className="mt-1 text-sm text-muted-foreground">选择一个预设头像。</p></div>
          <div className="flex justify-center"><div className="grid size-24 place-items-center overflow-hidden rounded-full border bg-muted text-2xl font-semibold">{user?.avatar ? <img src={user.avatar} alt="当前头像" className="size-full object-cover" /> : (user?.displayName || user?.username || "S").slice(0, 1)}</div></div>
          <div className="grid grid-cols-6 gap-2">{avatarURLs.map((url) => <button key={url} type="button" className="aspect-square overflow-hidden rounded-full border transition hover:ring-2 hover:ring-ring" disabled={profileMutation.isPending} onClick={() => void avatarDataURL(url).then((avatar) => profileMutation.mutate({ avatar })).catch((error: Error) => toast.error(error.message))}><img src={url} alt="预设头像" className="size-full object-cover" /></button>)}</div>
          {user?.avatar ? <Button variant="outline" className="w-full" onClick={() => profileMutation.mutate({ avatar: "" })}>清除头像</Button> : null}
        </section>
        <div className="space-y-5">
          <form className="space-y-5 rounded-lg border bg-card p-5" onSubmit={(event) => { event.preventDefault(); profileMutation.mutate({ username: profile.username.trim(), displayName: profile.displayName.trim() }) }}>
            <div><h2 className="font-medium">基本信息</h2><p className="mt-1 text-sm text-muted-foreground">登录名用于认证，昵称用于页面展示。</p></div>
            <div className="grid gap-4 sm:grid-cols-2"><Field><FieldLabel htmlFor="profile-username">登录名</FieldLabel><Input id="profile-username" value={profile.username} maxLength={32} pattern="[A-Za-z0-9_.-]{1,32}" onChange={(event) => setProfile({ ...profile, username: event.target.value })} required /></Field><Field><FieldLabel htmlFor="profile-display">昵称</FieldLabel><Input id="profile-display" value={profile.displayName} maxLength={32} onChange={(event) => setProfile({ ...profile, displayName: event.target.value })} /></Field></div>
            <div className="flex justify-end"><Button type="submit" disabled={profileMutation.isPending}>保存资料</Button></div>
          </form>
          <form className="space-y-5 rounded-lg border bg-card p-5" onSubmit={(event) => { event.preventDefault(); if (password.newPassword.length < 10) return toast.error("新密码至少 10 位"); if (password.newPassword !== password.confirm) return toast.error("两次新密码不一致"); passwordMutation.mutate() }}>
            <div><h2 className="font-medium">修改密码</h2><p className="mt-1 text-sm text-muted-foreground">成功后所有会话会失效，需要重新登录。</p></div>
            <Field><FieldLabel htmlFor="old-password">当前密码</FieldLabel><Input id="old-password" type="password" value={password.oldPassword} onChange={(event) => setPassword({ ...password, oldPassword: event.target.value })} required /></Field>
            <div className="grid gap-4 sm:grid-cols-2"><Field><FieldLabel htmlFor="new-password">新密码</FieldLabel><Input id="new-password" type="password" minLength={10} value={password.newPassword} onChange={(event) => setPassword({ ...password, newPassword: event.target.value })} required /></Field><Field><FieldLabel htmlFor="confirm-password">确认新密码</FieldLabel><Input id="confirm-password" type="password" minLength={10} value={password.confirm} onChange={(event) => setPassword({ ...password, confirm: event.target.value })} required /></Field></div>
            <div className="flex justify-end"><Button type="submit" variant="destructive" disabled={passwordMutation.isPending}>修改密码并退出</Button></div>
          </form>
        </div>
      </div>
    </div>
  )
}
