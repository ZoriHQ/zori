import { createRouter } from '@tanstack/react-router'
// eslint-disable-next-line
import { type QueryClient } from '@tanstack/react-query'
import { routeTree } from './routeTree.gen'
// eslint-disable-next-line
import { type AuthContextState, getAuthState } from './lib/auth-context'

export function createAppRouter(queryClient: QueryClient) {
  const router = createRouter({
    routeTree,
    context: {
      queryClient,
      auth: undefined as AuthContextState | undefined,
    },
    defaultPreload: 'intent',
    defaultPendingComponent: () => (
      <div className="flex items-center justify-center min-h-screen">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary"></div>
      </div>
    ),
    defaultErrorComponent: ({ error }) => (
      <div className="flex min-h-screen items-center justify-center">
        <div className="text-center">
          <h1 className="text-2xl font-bold mb-4">Something went wrong</h1>
          <p className="text-muted-foreground mb-4">{error.message}</p>
          <button
            onClick={() => window.location.reload()}
            className="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90"
          >
            Reload page
          </button>
        </div>
      </div>
    ),
    defaultNotFoundComponent: () => {
      console.log('404 Not Found - Current path:', window.location.pathname)
      console.log('Available routes:', router.state.matches)
      return (
        <div className="flex min-h-screen items-center justify-center">
          <div className="text-center">
            <h1 className="text-4xl font-bold mb-4">404</h1>
            <p className="text-xl text-muted-foreground mb-4">Page not found</p>
            <p className="text-sm text-muted-foreground mb-4">
              Path: {window.location.pathname}
            </p>
            <a
              href="/"
              className="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 inline-block"
            >
              Go home
            </a>
          </div>
        </div>
      )
    },
  })

  getAuthState().then((authState) => {
    router.update({
      context: {
        ...router.options.context,
        auth: authState,
      },
    })
  })

  return router
}

// Export router type for use in route definitions
export type Router = ReturnType<typeof createAppRouter>

// Create router context type
declare module '@tanstack/react-router' {
  interface Register {
    router: Router
  }

  // Define the context that will be available in all routes
  interface RouterContext {
    queryClient: QueryClient
    auth: AuthContextState | undefined
  }
}
