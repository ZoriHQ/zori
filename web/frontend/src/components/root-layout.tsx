'use client'

import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useEffect } from 'react'
import { setupTokenRefresh } from '@/lib/auth'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 1000 * 60 * 5, // 5 minutes
      gcTime: 1000 * 60 * 10, // 10 minutes
      retry: (failureCount, error) => {
        // Don't retry on 4xx errors
        if (error instanceof Error) {
          const message = error.message
          if (
            message.includes('401') ||
            message.includes('403') ||
            message.includes('404')
          ) {
            return false
          }
        }
        return failureCount < 3
      },
    },
  },
})

export function RootLayout({ children }: { children: React.ReactNode }) {
  useEffect(() => {
    // Set up automatic token refresh
    const cleanup = setupTokenRefresh()

    // Clean up on unmount
    return cleanup
  }, [])

  return (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  )
}
