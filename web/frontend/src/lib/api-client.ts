import { formatError } from 'zod'
import { formatApiError } from './utils'
import { auth } from './auth'

const API_BASE_URL =
  import.meta.env.VITE_API_BASE_URL || 'http://localhost:1323'

export class ApiError extends Error {
  constructor(
    message: string,
    public status: number,
    public data?: unknown,
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

export interface ApiResponse<T = unknown> {
  data: T
  success: boolean
  message?: string
  pagination?: {
    page: number
    limit: number
    total: number
    totalPages: number
  }
}

export interface RequestConfig extends RequestInit {
  timeout?: number
  retries?: number
  retryDelay?: number
}

export interface ApiClientConfig {
  baseURL: string
  timeout?: number
  retries?: number
  retryDelay?: number
}

type RequestInterceptor = (
  config: RequestConfig,
) => RequestConfig | Promise<RequestConfig>
type ResponseInterceptor = (response: Response) => Response | Promise<Response>
type ErrorInterceptor = (error: ApiError) => ApiError | Promise<ApiError>

export class ApiClient {
  private baseURL: string
  private getToken: () => Promise<string | null>
  private timeout: number
  private retries: number
  private retryDelay: number
  private requestInterceptors: Array<RequestInterceptor> = []
  private responseInterceptors: Array<ResponseInterceptor> = []
  private errorInterceptors: Array<ErrorInterceptor> = []
  private abortControllers = new Map<string, AbortController>()

  constructor(
    baseURL: string,
    getToken: () => Promise<string | null>,
    config: Partial<ApiClientConfig> = {},
  ) {
    this.baseURL = baseURL
    this.getToken = getToken
    this.timeout = config.timeout ?? 10000
    this.retries = config.retries ?? 3
    this.retryDelay = config.retryDelay ?? 1000
  }

  addRequestInterceptor(interceptor: RequestInterceptor): void {
    this.requestInterceptors.push(interceptor)
  }

  addResponseInterceptor(interceptor: ResponseInterceptor): void {
    this.responseInterceptors.push(interceptor)
  }

  addErrorInterceptor(interceptor: ErrorInterceptor): void {
    this.errorInterceptors.push(interceptor)
  }

  private async applyRequestInterceptors(
    config: RequestConfig,
  ): Promise<RequestConfig> {
    let finalConfig = config
    for (const interceptor of this.requestInterceptors) {
      finalConfig = await interceptor(finalConfig)
    }
    return finalConfig
  }

  private async applyResponseInterceptors(
    response: Response,
  ): Promise<Response> {
    let finalResponse = response
    for (const interceptor of this.responseInterceptors) {
      finalResponse = await interceptor(finalResponse)
    }
    return finalResponse
  }

  private async applyErrorInterceptors(error: ApiError): Promise<ApiError> {
    let finalError = error
    for (const interceptor of this.errorInterceptors) {
      finalError = await interceptor(finalError)
    }
    return finalError
  }

  private createAbortController(requestId: string): AbortController {
    const existingController = this.abortControllers.get(requestId)
    if (existingController) {
      existingController.abort()
    }

    const controller = new AbortController()
    this.abortControllers.set(requestId, controller)
    return controller
  }

  private async request<T>(
    endpoint: string,
    options: RequestConfig = {},
  ): Promise<T> {
    const requestId = `${options.method || 'GET'}-${endpoint}`
    const abortController = this.createAbortController(requestId)

    const token = await this.getToken()
    const url = `${this.baseURL}${endpoint}`

    let config: RequestConfig = {
      headers: {
        'Content-Type': 'application/json',
        ...(token && { Authorization: `Bearer ${token}` }),
        ...options.headers,
      },
      signal: abortController.signal,
      ...options,
    }

    config = await this.applyRequestInterceptors(config)

    const timeoutId = setTimeout(() => {
      abortController.abort()
    }, config.timeout ?? this.timeout)

    try {
      const response = await fetch(url, config)
      clearTimeout(timeoutId)

      const interceptedResponse = await this.applyResponseInterceptors(response)

      if (!interceptedResponse.ok) {
        const errorData = await interceptedResponse.text().catch(() => ({}))
        const error = new ApiError(
          formatApiError(errorData),
          interceptedResponse.status,
          errorData,
        )
        throw await this.applyErrorInterceptors(error)
      }

      if (interceptedResponse.status === 204) {
        return {} as T
      }

      const data = await interceptedResponse.json()
      return data
    } catch (error) {
      clearTimeout(timeoutId)
      this.abortControllers.delete(requestId)

      if (error instanceof ApiError) {
        throw await this.applyErrorInterceptors(error)
      }

      if (error instanceof TypeError && error.message.includes('fetch')) {
        const networkError = new ApiError(
          'API service is currently unavailable. Please try again later.',
          503,
          error,
        )
        throw await this.applyErrorInterceptors(networkError)
      }

      const genericError = new ApiError('Network error occurred', 0, error)
      throw await this.applyErrorInterceptors(genericError)
    }
  }

  private async requestWithRetry<T>(
    endpoint: string,
    options: RequestConfig = {},
  ): Promise<T> {
    const maxRetries = options.retries ?? this.retries
    const retryDelay = options.retryDelay ?? this.retryDelay

    for (let attempt = 0; attempt <= maxRetries; attempt++) {
      try {
        return await this.request<T>(endpoint, options)
      } catch (error) {
        if (
          attempt === maxRetries ||
          !(error instanceof ApiError) ||
          error.status < 500
        ) {
          throw error
        }

        await new Promise((resolve) =>
          setTimeout(resolve, retryDelay * Math.pow(2, attempt)),
        )
      }
    }

    throw new ApiError('Max retries exceeded', 0)
  }

  async get<T>(
    endpoint: string,
    params?: Record<string, unknown>,
    config?: RequestConfig,
  ): Promise<T> {
    const url = new URL(endpoint, this.baseURL)
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          url.searchParams.append(key, String(value))
        }
      })
    }

    return this.requestWithRetry<T>(url.pathname + url.search, config)
  }

  async post<T>(
    endpoint: string,
    data?: unknown,
    config?: RequestConfig,
  ): Promise<T> {
    return this.requestWithRetry<T>(endpoint, {
      method: 'POST',
      body: data ? JSON.stringify(data) : undefined,
      ...config,
    })
  }

  async put<T>(
    endpoint: string,
    data?: unknown,
    config?: RequestConfig,
  ): Promise<T> {
    return this.requestWithRetry<T>(endpoint, {
      method: 'PUT',
      body: data ? JSON.stringify(data) : undefined,
      ...config,
    })
  }

  async patch<T>(
    endpoint: string,
    data?: unknown,
    config?: RequestConfig,
  ): Promise<T> {
    return this.requestWithRetry<T>(endpoint, {
      method: 'PATCH',
      body: data ? JSON.stringify(data) : undefined,
      ...config,
    })
  }

  async delete<T>(endpoint: string, config?: RequestConfig): Promise<T> {
    return this.requestWithRetry<T>(endpoint, {
      method: 'DELETE',
      ...config,
    })
  }

  cancelAllRequests(): void {
    this.abortControllers.forEach((controller) => controller.abort())
    this.abortControllers.clear()
  }

  cancelRequest(requestId: string): void {
    const controller = this.abortControllers.get(requestId)
    if (controller) {
      controller.abort()
      this.abortControllers.delete(requestId)
    }
  }
}

export function useApiClient() {
  return new ApiClient(
    API_BASE_URL,
    async () => {
      // Check if token is expired and try to refresh it
      if (auth.isTokenExpired()) {
        const refreshed = await auth.refreshAccessToken()
        if (refreshed) {
          return refreshed.access_token
        }
      }
      return auth.getAccessToken()
    },
    {
      timeout: 4000000,
      retries: 3,
      retryDelay: 1000,
    },
  )
}
