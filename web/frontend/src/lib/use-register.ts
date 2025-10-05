import { useMutation } from '@tanstack/react-query'
import { useApiClient } from '@/lib/api-client'
// eslint-disable-next-line
import { auth, type AuthResponse } from '@/lib/auth'

export interface RegisterData {
  email: string
  password: string
  first_name: string
  last_name: string
  organization_name: string
}

export function useRegister() {
  const apiClient = useApiClient()

  return useMutation({
    mutationKey: ['register'],
    mutationFn: async (data: RegisterData) => {
      const response = await apiClient.post<AuthResponse>(
        `/api/v1/auth/register`,
        data,
      )

      // Store the auth data in localStorage
      auth.setAuthData(response)

      return response
    },
    onSuccess: (data) => {
      // Optional: Redirect to dashboard or home page after successful registration
      // window.location.href = '/dashboard'
    },
    onError: (error) => {
      // Clear any existing auth data on registration error
      auth.clearAuthData()
      console.error('Registration failed:', error)
    },
  })
}
