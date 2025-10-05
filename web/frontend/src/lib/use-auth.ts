import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect } from 'react'
import { useRouter } from '@tanstack/react-router'
import { auth } from './auth'

import type Zoriapi from 'zorihq'

export interface AuthState {
  isAuthenticated: boolean
  isLoading: boolean
  account: Zoriapi.V1.Auth.Account | null
  organization: Zoriapi.V1.Auth.Organization | null
  accessToken: string | null
}

export function useAuth() {
  const router = useRouter()
  const queryClient = useQueryClient()

  const { data, isLoading } = useQuery({
    queryKey: ['auth-status'],
    queryFn: async (): Promise<AuthState> => {
      const token = auth.getAccessToken()

      if (!token) {
        return {
          isAuthenticated: false,
          isLoading: false,
          account: null,
          organization: null,
          accessToken: null,
        }
      }

      if (auth.isTokenExpired()) {
        const refreshedData = await auth.refreshAccessToken()

        if (refreshedData) {
          return {
            isAuthenticated: true,
            isLoading: false,
            account: refreshedData.account || auth.getAccount(),
            organization: refreshedData.organization || auth.getOrganization(),
            accessToken: refreshedData.access_token!,
          }
        } else {
          return {
            isAuthenticated: false,
            isLoading: false,
            account: null,
            organization: null,
            accessToken: null,
          }
        }
      }

      return {
        isAuthenticated: true,
        isLoading: false,
        account: auth.getAccount(),
        organization: auth.getOrganization(),
        accessToken: token,
      }
    },
    staleTime: 1000 * 60 * 5,
    gcTime: 1000 * 60 * 10,
    refetchInterval: 1000 * 60 * 5,
    refetchOnWindowFocus: true,
    retry: false,
  })

  const invalidateAuth = () => {
    queryClient.invalidateQueries({ queryKey: ['auth-status'] })
  }

  const checkAuth = () => {
    return data?.isAuthenticated ?? false
  }

  const requireAuth = () => {
    if (!checkAuth()) {
      router.navigate({ to: '/login' })
      return false
    }
    return true
  }

  const getUser = () => {
    return data?.account ?? null
  }

  const getOrganization = () => {
    return data?.organization ?? null
  }

  return {
    isAuthenticated: data?.isAuthenticated ?? false,
    isLoading,
    account: data?.account ?? null,
    organization: data?.organization ?? null,
    accessToken: data?.accessToken ?? null,
    checkAuth,
    requireAuth,
    getUser,
    getOrganization,
    invalidateAuth,
  }
}

// Hook for protecting routes
export function useAuthGuard(redirectTo = '/login') {
  const { isAuthenticated, isLoading } = useAuth()
  const router = useRouter()

  useEffect(() => {
    if (!isLoading && !isAuthenticated) {
      router.navigate({ to: redirectTo })
    }
  }, [isAuthenticated, isLoading, redirectTo, router])

  return { isAuthenticated, isLoading }
}

// Hook for redirecting authenticated users (e.g., from login page)
export function useGuestGuard(redirectTo = '/dashboard') {
  const { isAuthenticated, isLoading } = useAuth()
  const router = useRouter()

  useEffect(() => {
    if (!isLoading && isAuthenticated) {
      router.navigate({ to: redirectTo })
    }
  }, [isAuthenticated, isLoading, redirectTo, router])

  return { isAuthenticated, isLoading }
}
