import React from 'react'
// eslint-disable-next-line
import ReactDOM, { type Container } from 'react-dom/client'
import { RouterProvider } from '@tanstack/react-router'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { ReactQueryDevtools } from '@tanstack/react-query-devtools'
import { createAppRouter } from './router'
import { getAuthState } from './lib/auth-context'
import { auth } from './lib/auth'
import './index.css'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 1000 * 60 * 5, // 5 minutes
      gcTime: 1000 * 60 * 10, // 10 minutes
      retry: (failureCount, error) => {
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

const router = createAppRouter(queryClient)

function RootApp() {
  return (
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
      {import.meta.env.DEV && <ReactQueryDevtools />}
    </QueryClientProvider>
  )
}

function setupTokenRefresh() {
  const checkAndRefreshToken = async () => {
    const expiry = localStorage.getItem('token_expiry')
    if (!expiry) return

    const expiryTime = parseInt(expiry, 10)
    const currentTime = Date.now()
    const timeUntilExpiry = expiryTime - currentTime

    const refreshThreshold = 5 * 60 * 1000 // 5 minutes

    if (timeUntilExpiry > 0 && timeUntilExpiry <= refreshThreshold) {
      await auth.refreshAccessToken()
    }
  }

  const intervalId = setInterval(checkAndRefreshToken, 60 * 1000)

  checkAndRefreshToken()

  return () => clearInterval(intervalId)
}

const rootElement = document.getElementById('root')
if (!rootElement) {
  throw new Error('Root element not found')
}

async function initApp() {
  const cleanup = setupTokenRefresh()

  const authState = await getAuthState()

  router.update({
    context: {
      ...router.options.context,
      auth: authState,
    },
  })

  window.addEventListener('beforeunload', () => {
    cleanup()
  })

  await router.load()

  ReactDOM.createRoot(rootElement as Container).render(
    <React.StrictMode>
      <RootApp />
    </React.StrictMode>,
  )
}

// Start the app
initApp().catch(console.error)
