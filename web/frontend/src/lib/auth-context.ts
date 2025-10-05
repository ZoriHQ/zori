import { createContext, useContext } from 'react'
// eslint-disable-next-line
import { type AuthResponse, auth } from './auth'

export interface AuthContextState {
  isAuthenticated: boolean
  account: AuthResponse['account'] | null
  organization: AuthResponse['organization'] | null
  accessToken: string | null
}

export const AuthContext = createContext<AuthContextState | undefined>(
  undefined,
)

export function useAuthContext() {
  const context = useContext(AuthContext)
  if (!context) {
    throw new Error('useAuthContext must be used within an AuthProvider')
  }
  return context
}

// This function will be called by the router to get auth state
export async function getAuthState(): Promise<AuthContextState> {
  try {
    const token = auth.getAccessToken()

    if (!token) {
      return {
        isAuthenticated: false,
        account: null,
        organization: null,
        accessToken: null,
      }
    }

    // Check if token is expired
    if (auth.isTokenExpired()) {
      // Try to refresh the token
      const refreshedData = await auth.refreshAccessToken()

      if (refreshedData) {
        return {
          isAuthenticated: true,
          account: refreshedData.account || auth.getAccount(),
          organization: refreshedData.organization || auth.getOrganization(),
          accessToken: refreshedData.access_token,
        }
      } else {
        // Refresh failed
        return {
          isAuthenticated: false,
          account: null,
          organization: null,
          accessToken: null,
        }
      }
    }

    // Token is valid
    return {
      isAuthenticated: true,
      account: auth.getAccount(),
      organization: auth.getOrganization(),
      accessToken: token,
    }
  } catch (error) {
    console.error('Failed to get auth state:', error)
    return {
      isAuthenticated: false,
      account: null,
      organization: null,
      accessToken: null,
    }
  }
}
