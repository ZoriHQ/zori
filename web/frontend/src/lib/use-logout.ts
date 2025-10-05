import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useApiClient } from '@/lib/api-client'
import { auth } from '@/lib/auth'

export function useLogout() {
  const apiClient = useApiClient()
  const queryClient = useQueryClient()
  const navigate = useNavigate()

  return useMutation({
    mutationKey: ['logout'],
    mutationFn: async () => {
      // Get refresh token for logout request
      const refreshToken = auth.getRefreshToken()

      try {
        // Call logout endpoint if available
        // This allows the backend to invalidate the refresh token
        if (refreshToken) {
          await apiClient.post('/api/v1/auth/logout', {
            refresh_token: refreshToken,
          })
        }
      } catch (error) {
        // Even if the logout API call fails, we still want to clear local data
        console.error('Logout API call failed:', error)
      }
    },
    onSettled: () => {
      // Clear all auth data from localStorage
      auth.clearAuthData()

      // Clear all cached queries
      queryClient.clear()

      // Redirect to login page using TanStack Router
      navigate({ to: '/login' })
    },
  })
}
