import { useMutation } from '@tanstack/react-query'
import { useApiClient } from '@/lib/api-client'
// eslint-disable-next-line
import { auth, type AuthResponse } from '@/lib/auth'

export function useLogin() {
  const apiClient = useApiClient()

  return useMutation({
    mutationKey: ['login'],
    mutationFn: async (data: { email: string; password: string }) => {
      const response = await apiClient.post<AuthResponse>(
        `/api/v1/auth/login`,
        data,
      )

      // Store the auth data in localStorage
      auth.setAuthData(response)

      return response
    },
    onSuccess: (data) => {
      // Optional: Redirect to dashboard or home page after successful login
      // window.location.href = '/dashboard'
    },
    onError: (error) => {
      // Clear any existing auth data on login error
      auth.clearAuthData()
      console.error('Login failed:', error)
    },
  })
}
