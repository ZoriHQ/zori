import { redirect } from '@tanstack/react-router'
import { getAuthState } from './auth-context'

export async function requireAuth({ location }: { location: any }) {
  const authState = await getAuthState()

  if (!authState.isAuthenticated) {
    // Store the attempted location for redirecting after login
    const redirectTo = location.href
    throw redirect({
      to: '/login',
      search: {
        redirect: redirectTo,
      },
    })
  }

  return authState
}

export async function requireGuest({ location }: { location: any }) {
  const authState = await getAuthState()

  if (authState.isAuthenticated) {
    throw redirect({
      to: '/',
    })
  }

  return authState
}

export async function optionalAuth() {
  const authState = await getAuthState()
  return authState
}

export async function requireRole(role: string) {
  const authState = await getAuthState()

  if (!authState.isAuthenticated) {
    throw redirect({
      to: '/login',
    })
  }

  const userRole = (authState.account as any)?.role

  if (userRole !== role) {
    throw redirect({
      to: '/login',
      statusCode: 403,
    })
  }

  return authState
}

export async function requireOrganization() {
  const authState = await getAuthState()

  if (!authState.isAuthenticated) {
    throw redirect({
      to: '/login',
    })
  }

  return authState
}

export async function requireAuthAndOrg({ location }: { location: any }) {
  const authState = await getAuthState()

  if (!authState.isAuthenticated) {
    const redirectTo = location.href
    throw redirect({
      to: '/login',
      search: {
        redirect: redirectTo,
      },
    })
  }

  return authState
}

export function getPostLoginRedirect(search: Record<string, any>): string {
  // eslint-disable-next-line
  const redirect = search?.redirect as string

  if (redirect && redirect.startsWith('/') && !redirect.startsWith('//')) {
    return redirect
  }

  return '/dashboard'
}
