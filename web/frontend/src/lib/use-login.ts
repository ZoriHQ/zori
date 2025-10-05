import { useMutation } from '@tanstack/react-query'
import Zoriapi from 'zorihq'
import { auth } from '@/lib/auth'

const API_BASE_URL =
  import.meta.env.VITE_API_BASE_URL || 'http://localhost:1323'

export function useLogin() {
  const zori = new Zoriapi({
    baseURL: API_BASE_URL,
    apiKey: '__empty__',
  })

  return useMutation({
    mutationKey: ['login'],
    mutationFn: async (data: {
      email: string
      password: string
    }): Promise<Zoriapi.V1.Auth.AuthResponse> => {
      const response = await zori.v1.auth.login({
        email: data.email,
        password: data.password,
      })

      auth.setAuthData({
        access_token: response.access_token!,
        refresh_token: response.refresh_token!,
        expires_in: response.expires_in!,
        account: response.account!,
        organization: response.organization!,
      })

      return response
    },
    onError: (error) => {
      auth.clearAuthData()
      console.error('Login failed:', error)
    },
  })
}
