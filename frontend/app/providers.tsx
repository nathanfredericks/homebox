"use client";

import { useState, type ReactNode } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Toaster } from "sonner";
import { ThemeRegistry } from "~~/lib/theme/ThemeRegistry";
import { I18nProvider } from "~~/lib/i18n/I18nProvider";

/**
 * Client-side application providers. Data is fetched entirely client-side
 * against the Go API (parity with the old ssr:false SPA), so the QueryClient
 * lives here. Theme + i18n providers wrap the tree; sonner's Toaster is
 * mounted once at the root. The auth/dialog/confirm providers are layered in
 * by the shell agent inside the (app) segment.
 */
export function Providers({ children }: { children: ReactNode }) {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            refetchOnWindowFocus: false,
            retry: 1,
            staleTime: 30_000,
          },
        },
      })
  );

  return (
    <QueryClientProvider client={queryClient}>
      <I18nProvider>
        <ThemeRegistry>
          {children}
          <Toaster richColors closeButton position="bottom-right" />
        </ThemeRegistry>
      </I18nProvider>
    </QueryClientProvider>
  );
}
