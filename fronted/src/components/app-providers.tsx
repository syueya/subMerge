import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { ThemeProvider } from "next-themes"
import { type PropsWithChildren, useState } from "react"
import { Toaster } from "sonner"

import { TooltipProvider } from "@/components/ui/tooltip"
import { AuthProvider } from "@/features/auth/auth-context"

export function AppProviders({ children }: PropsWithChildren) {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            staleTime: 5 * 60 * 1000,
            retry: 1,
            refetchOnWindowFocus: false,
          },
          mutations: { retry: false },
        },
      }),
  )

  return (
    <ThemeProvider attribute="class" defaultTheme="system" enableSystem storageKey="submerge-theme">
      <QueryClientProvider client={queryClient}>
        <AuthProvider>
          <TooltipProvider>{children}</TooltipProvider>
        </AuthProvider>
        <Toaster richColors closeButton position="top-center" />
      </QueryClientProvider>
    </ThemeProvider>
  )
}
