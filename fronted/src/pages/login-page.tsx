import { zodResolver } from "@hookform/resolvers/zod"
import { LoaderCircle } from "lucide-react"
import { useEffect, useState } from "react"
import { useForm } from "react-hook-form"
import { Navigate, useLocation, useNavigate } from "react-router-dom"
import { toast } from "sonner"
import { z } from "zod"

import heroImage from "@/assets/login-control-plane.png"
import submergeLogo from "@/assets/submerge-logo.svg"
import { ThemeToggle } from "@/components/theme-toggle"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { useAuth } from "@/features/auth/auth-context"
import { ApiError, api } from "@/lib/api"
import { randomAvatarDataURL } from "@/lib/avatars"

const loginSchema = z.object({
  username: z.string().trim().min(1, "请输入用户名"),
  password: z.string().min(1, "请输入密码"),
  displayName: z.string().trim().max(64, "昵称最多 64 个字符").optional(),
  password2: z.string().optional(),
})

type LoginValues = z.infer<typeof loginSchema>

function loginErrorMessage(error: unknown) {
  if (!(error instanceof ApiError)) return "登录失败，请稍后重试"

  const messages: Record<string, string> = {
    invalid_credentials: "用户名或密码错误",
    needs_setup: "尚未创建管理员，请先完成初始化",
    setup_done: "管理员已创建，请使用账户登录",
    weak_password: "密码至少需要 10 位，且不能使用常见弱密码",
    bad_username: "用户名仅限字母、数字、_、-、.，最长 32 位",
    bad_request: "提交的信息无效，请检查后重试",
    timeout: "请求超时，请稍后重试",
  }
  return messages[error.code ?? ""] ?? (error.status === 0 ? "网络连接失败，请检查网络后重试" : "登录失败，请稍后重试")
}

export function LoginPage() {
  const auth = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const [checking, setChecking] = useState(true)
  const [needsSetup, setNeedsSetup] = useState(false)
  const form = useForm<LoginValues>({
    resolver: zodResolver(loginSchema),
    defaultValues: { username: "", password: "", password2: "", displayName: "" },
  })

  useEffect(() => {
    api
      .get<{ needsSetup: boolean }>("/auth/setup-status")
      .then((result) => setNeedsSetup(result.needsSetup))
      .catch(() => setNeedsSetup(false))
      .finally(() => setChecking(false))
  }, [])

  if (!auth.loading && auth.user) return <Navigate replace to="/main/dashboard" />

  const submit = form.handleSubmit(async (values) => {
    if (needsSetup) {
      if (!/^[A-Za-z0-9_.-]{1,32}$/.test(values.username)) {
        form.setError("username", { message: "仅限字母、数字、_ - .，最长 32 位" })
        return
      }
      if (values.password.length < 10) {
        form.setError("password", { message: "密码至少 10 位" })
        return
      }
      if (values.password !== values.password2) {
        form.setError("password2", { message: "两次密码不一致" })
        return
      }
    }
    try {
      if (needsSetup) {
        const avatar = await randomAvatarDataURL()
        await auth.bootstrap({
          username: values.username.trim(),
          password: values.password,
          displayName: values.displayName?.trim() || values.username.trim(),
          avatar,
        })
        toast.success("管理员已创建")
      } else {
        await auth.login(values.username.trim(), values.password)
        toast.success("登录成功")
      }
      const from = (location.state as { from?: string } | null)?.from
      navigate(from?.startsWith("/main/") ? from : "/main/dashboard", { replace: true })
    } catch (error) {
      toast.error(loginErrorMessage(error))
    }
  })

  const busy = checking || auth.loading

  return (
    <main className="grid min-h-svh bg-background lg:grid-cols-2">
      <section className="flex min-h-svh flex-col p-6 md:p-10">
        <div className="relative flex h-8 shrink-0 items-center justify-end lg:h-auto lg:justify-between">
          <div className="absolute left-1/2 top-[7.5rem] flex -translate-x-1/2 items-center gap-2.5 whitespace-nowrap text-xl font-semibold lg:static lg:gap-2 lg:translate-x-0 lg:text-base">
            <img src={submergeLogo} alt="SubMerge" className="size-10 lg:size-8" />
            SubMerge
          </div>
          <ThemeToggle />
        </div>

        <div className="flex flex-1 items-center justify-center py-12">
          <div className="w-full max-w-sm">
            <div className="mb-8 space-y-2">
              <h1 className="text-2xl font-semibold">{needsSetup ? "创建管理员账户" : "欢迎回来"}</h1>
              <p className="text-sm leading-6 text-muted-foreground">
                {needsSetup ? "完成首次设置，开始管理你的订阅配置。" : "登录以继续管理订阅、策略与发布。"}
              </p>
            </div>

            {busy ? (
              <div className="flex h-48 items-center justify-center"><LoaderCircle className="size-5 animate-spin text-muted-foreground" /></div>
            ) : (
              <form className="space-y-5" onSubmit={submit} noValidate>
                <div className="space-y-2">
                  <Label htmlFor="username">用户名</Label>
                  <Input id="username" className="h-10" autoComplete="username" autoFocus {...form.register("username")} aria-invalid={!!form.formState.errors.username} />
                  {form.formState.errors.username ? <p className="text-xs text-destructive">{form.formState.errors.username.message}</p> : null}
                </div>
                {needsSetup ? (
                  <div className="space-y-2">
                    <Label htmlFor="displayName">显示名称</Label>
                    <Input id="displayName" className="h-10" autoComplete="nickname" {...form.register("displayName")} />
                  </div>
                ) : null}
                <div className="space-y-2">
                  <Label htmlFor="password">密码</Label>
                  <Input id="password" className="h-10" type="password" autoComplete={needsSetup ? "new-password" : "current-password"} {...form.register("password")} aria-invalid={!!form.formState.errors.password} />
                  {form.formState.errors.password ? <p className="text-xs text-destructive">{form.formState.errors.password.message}</p> : null}
                </div>
                {needsSetup ? (
                  <div className="space-y-2">
                    <Label htmlFor="password2">确认密码</Label>
                    <Input id="password2" className="h-10" type="password" autoComplete="new-password" {...form.register("password2")} aria-invalid={!!form.formState.errors.password2} />
                    {form.formState.errors.password2 ? <p className="text-xs text-destructive">{form.formState.errors.password2.message}</p> : null}
                  </div>
                ) : null}
                <Button className="h-10 w-full" type="submit" disabled={form.formState.isSubmitting}>
                  {form.formState.isSubmitting ? <LoaderCircle className="animate-spin" /> : null}
                  {needsSetup ? "创建管理员" : "登录"}
                </Button>
              </form>
            )}
          </div>
        </div>

        <p className="text-xs text-muted-foreground">自托管 Clash 订阅配置中心</p>
      </section>

      <section className="relative hidden overflow-hidden border-l border-[#274b6a] bg-[#17324a] p-8 text-white lg:flex lg:min-h-svh lg:flex-col xl:p-12">
        <div className="absolute inset-8 border border-white/10" />
        <div className="relative z-10 flex items-center gap-2 text-sm text-white/65"><span className="size-2 rounded-full bg-[#8fb8d7]" />SUBMERGE CONTROL PLANE</div>
        <div className="relative z-10 mx-auto grid min-h-0 w-full max-w-xl flex-1 place-items-center py-3">
          <img src={heroImage} alt="订阅源经过规则处理后统一发布到不同客户端" className="relative z-10 h-[min(52vh,32rem)] w-full object-contain" />
          <span className="absolute left-0 top-1/3 rounded-md border border-white/10 bg-white/5 px-3 py-2 text-xs text-white/60">SOURCE / 04</span>
          <span className="absolute right-0 top-1/2 rounded-md border border-white/10 bg-white/5 px-3 py-2 text-xs text-white/60">ROUTES / 257</span>
          <span className="absolute bottom-8 left-12 rounded-md border border-white/10 bg-white/5 px-3 py-2 text-xs text-white/60">RELEASE / READY</span>
        </div>
        <div className="relative z-10 max-w-md">
          <p className="text-3xl font-medium leading-tight">订阅在此汇流，配置由此发布。</p>
          <p className="mt-4 text-sm leading-6 text-white/60">统一管理来源、策略、规则与客户端下发。</p>
        </div>
      </section>
    </main>
  )
}
