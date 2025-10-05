import { z } from 'zod'

// Define the structure of the auth response
export const AuthResponseSchema = z.object({
  access_token: z.string(),
  refresh_token: z.string(),
  expires_in: z.number(),
  account: z
    .object({
      id: z.string(),
      email: z.string(),
      first_name: z.string().optional(),
      last_name: z.string().optional(),
      created_at: z.string().optional(),
      updated_at: z.string().optional(),
    })
    .optional(),
  organization: z
    .object({
      id: z.string(),
      name: z.string(),
      created_at: z.string().optional(),
      updated_at: z.string().optional(),
    })
    .optional(),
})

export type AuthResponse = z.infer<typeof AuthResponseSchema>

// Storage keys
const AUTH_STORAGE_KEYS = {
  ACCESS_TOKEN: 'access_token',
  REFRESH_TOKEN: 'refresh_token',
  TOKEN_EXPIRY: 'token_expiry',
  ACCOUNT: 'account',
  ORGANIZATION: 'organization',
} as const

// Token management functions
export const auth = {
  // Store auth data
  setAuthData(data: AuthResponse) {
    try {
      // Store tokens
      localStorage.setItem(AUTH_STORAGE_KEYS.ACCESS_TOKEN, data.access_token)
      localStorage.setItem(AUTH_STORAGE_KEYS.REFRESH_TOKEN, data.refresh_token)

      // Calculate and store expiry time
      const expiryTime = Date.now() + data.expires_in * 1000
      localStorage.setItem(
        AUTH_STORAGE_KEYS.TOKEN_EXPIRY,
        expiryTime.toString(),
      )

      // Store account and organization data if available
      if (data.account) {
        localStorage.setItem(
          AUTH_STORAGE_KEYS.ACCOUNT,
          JSON.stringify(data.account),
        )
      }
      if (data.organization) {
        localStorage.setItem(
          AUTH_STORAGE_KEYS.ORGANIZATION,
          JSON.stringify(data.organization),
        )
      }
    } catch (error) {
      console.error('Failed to store auth data:', error)
      throw new Error('Failed to save authentication data')
    }
  },

  // Get access token
  getAccessToken(): string | null {
    try {
      const token = localStorage.getItem(AUTH_STORAGE_KEYS.ACCESS_TOKEN)

      // Check if token exists and is not expired
      if (token && !this.isTokenExpired()) {
        return token
      }

      return null
    } catch {
      return null
    }
  },

  // Get refresh token
  getRefreshToken(): string | null {
    try {
      return localStorage.getItem(AUTH_STORAGE_KEYS.REFRESH_TOKEN)
    } catch {
      return null
    }
  },

  // Check if token is expired
  isTokenExpired(): boolean {
    try {
      const expiry = localStorage.getItem(AUTH_STORAGE_KEYS.TOKEN_EXPIRY)
      if (!expiry) return true

      return Date.now() > parseInt(expiry, 10)
    } catch {
      return true
    }
  },

  // Get stored account data
  getAccount() {
    try {
      const accountStr = localStorage.getItem(AUTH_STORAGE_KEYS.ACCOUNT)
      if (!accountStr) return null

      return JSON.parse(accountStr)
    } catch {
      return null
    }
  },

  // Get stored organization data
  getOrganization() {
    try {
      const orgStr = localStorage.getItem(AUTH_STORAGE_KEYS.ORGANIZATION)
      if (!orgStr) return null

      return JSON.parse(orgStr)
    } catch {
      return null
    }
  },

  // Clear all auth data (logout)
  clearAuthData() {
    try {
      Object.values(AUTH_STORAGE_KEYS).forEach((key) => {
        localStorage.removeItem(key)
      })
    } catch (error) {
      console.error('Failed to clear auth data:', error)
    }
  },

  // Check if user is authenticated
  isAuthenticated(): boolean {
    return !!this.getAccessToken()
  },

  // Refresh the access token using refresh token
  async refreshAccessToken(): Promise<AuthResponse | null> {
    try {
      const refreshToken = this.getRefreshToken()
      if (!refreshToken) {
        throw new Error('No refresh token available')
      }

      const response = await fetch('/api/v1/auth/refresh', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ refresh_token: refreshToken }),
      })

      if (!response.ok) {
        throw new Error('Failed to refresh token')
      }

      const data = await response.json()
      const validatedData = AuthResponseSchema.parse(data)

      // Store the new auth data
      this.setAuthData(validatedData)

      return validatedData
    } catch (error) {
      console.error('Failed to refresh token:', error)
      // Clear auth data if refresh fails
      this.clearAuthData()
      return null
    }
  },

  // Update just the tokens (useful for token refresh)
  updateTokens(accessToken: string, refreshToken: string, expiresIn: number) {
    try {
      localStorage.setItem(AUTH_STORAGE_KEYS.ACCESS_TOKEN, accessToken)
      localStorage.setItem(AUTH_STORAGE_KEYS.REFRESH_TOKEN, refreshToken)

      const expiryTime = Date.now() + expiresIn * 1000
      localStorage.setItem(
        AUTH_STORAGE_KEYS.TOKEN_EXPIRY,
        expiryTime.toString(),
      )
    } catch (error) {
      console.error('Failed to update tokens:', error)
      throw new Error('Failed to update authentication tokens')
    }
  },
}

// Function to automatically refresh token when it's about to expire
export function setupTokenRefresh() {
  const checkAndRefreshToken = async () => {
    const expiry = localStorage.getItem(AUTH_STORAGE_KEYS.TOKEN_EXPIRY)
    if (!expiry) return

    const expiryTime = parseInt(expiry, 10)
    const currentTime = Date.now()
    const timeUntilExpiry = expiryTime - currentTime

    // Refresh token 5 minutes before it expires
    const refreshThreshold = 5 * 60 * 1000 // 5 minutes

    if (timeUntilExpiry > 0 && timeUntilExpiry <= refreshThreshold) {
      await auth.refreshAccessToken()
    }
  }

  // Check token status every minute
  const intervalId = setInterval(checkAndRefreshToken, 60 * 1000)

  // Also check immediately
  checkAndRefreshToken()

  // Return cleanup function
  return () => clearInterval(intervalId)
}
