// Kongtrol UI — cache + live feed for the whole app · v0.1.0 · 2026-11-18
import { useEffect, useState, type ReactNode } from 'react'
import { QueryClientProvider, type QueryClient } from '@tanstack/react-query'
import { createQueryClient } from '../data/queryClient'
import { startLiveFeed } from '../data/live'

/** One QueryClient and one WebSocket per page; pass `client` in tests. */
export function DataProvider({ children, client }: { children: ReactNode; client?: QueryClient }) {
  const [queryClient] = useState(() => client ?? createQueryClient())
  useEffect(() => startLiveFeed(queryClient), [queryClient])
  return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
}
