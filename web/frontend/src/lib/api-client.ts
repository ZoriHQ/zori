import Zoriapi from 'zorihq'
import { auth } from './auth'

const API_BASE_URL =
  import.meta.env.VITE_API_BASE_URL || 'http://localhost:1323'

export function useApiClient() {
  const zclient = new Zoriapi({
    baseURL: API_BASE_URL,
    apiKey: auth.getAccessToken()!,
  })

  return zclient
}
