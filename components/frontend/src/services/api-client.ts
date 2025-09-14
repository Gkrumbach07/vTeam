// Multi-tenant API client for namespace-scoped requests

import {
  Session,
  CreateSessionRequest,
  UpdateSessionRequest,
  SessionListResponse,
  UserNamespacesResponse,
} from '@/types/session'

export interface ApiClientConfig {
  baseUrl?: string
  authToken?: string
}

export interface ApiError extends Error {
  status: number
  code?: string
}

export interface PaginationOptions {
  pageToken?: string
  limit?: number
}

export interface SessionListOptions extends PaginationOptions {
  phase?: string
  framework?: string
  search?: string
}

class ApiClientError extends Error implements ApiError {
  status: number
  code?: string

  constructor(message: string, status: number, code?: string) {
    super(message)
    this.name = 'ApiClientError'
    this.status = status
    this.code = code
  }
}

export class ApiClient {
  private baseUrl: string
  private authToken?: string

  constructor(config: ApiClientConfig = {}) {
    this.baseUrl = config.baseUrl || '/api'
    this.authToken = config.authToken
  }

  /**
   * Set authentication token
   */
  setAuthToken(token: string) {
    this.authToken = token
  }

  /**
   * Get authentication token from localStorage if not provided
   */
  private getAuthToken(): string | null {
    if (this.authToken) {
      return this.authToken
    }

    if (typeof window !== 'undefined') {
      return localStorage.getItem('authToken')
    }

    return null
  }

  /**
   * Make an authenticated HTTP request
   */
  private async request<T>(
    path: string,
    options: RequestInit = {}
  ): Promise<T> {
    const token = this.getAuthToken()
    const url = `${this.baseUrl}${path}`

    const headers: HeadersInit = {
      'Content-Type': 'application/json',
      ...options.headers,
    }

    if (token) {
      headers.Authorization = `Bearer ${token}`
    }

    const config: RequestInit = {
      ...options,
      headers,
    }

    try {
      const response = await fetch(url, config)

      // Handle non-JSON responses (like 204 No Content)
      if (response.status === 204) {
        return {} as T
      }

      const data = await response.json()

      if (!response.ok) {
        throw new ApiClientError(
          data.message || `HTTP ${response.status}: ${response.statusText}`,
          response.status,
          data.code
        )
      }

      return data
    } catch (error) {
      if (error instanceof ApiClientError) {
        throw error
      }

      throw new ApiClientError(
        error instanceof Error ? error.message : 'Network error',
        0
      )
    }
  }

  /**
   * Build query string from parameters
   */
  private buildQueryString(params: Record<string, any>): string {
    const queryParams = new URLSearchParams()

    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined && value !== null && value !== '') {
        queryParams.set(key, String(value))
      }
    })

    const queryString = queryParams.toString()
    return queryString ? `?${queryString}` : ''
  }

  // Namespace APIs

  /**
   * Get user's accessible namespaces
   */
  async getUserNamespaces(): Promise<UserNamespacesResponse> {
    return this.request<UserNamespacesResponse>('/v1/user/namespaces')
  }

  // Session APIs

  /**
   * List sessions in a namespace
   */
  async listSessions(
    namespace: string,
    options: SessionListOptions = {}
  ): Promise<SessionListResponse> {
    const queryString = this.buildQueryString(options)
    return this.request<SessionListResponse>(
      `/v1/namespaces/${namespace}/sessions${queryString}`
    )
  }

  /**
   * Get a specific session
   */
  async getSession(namespace: string, sessionName: string): Promise<Session> {
    return this.request<Session>(
      `/v1/namespaces/${namespace}/sessions/${sessionName}`
    )
  }

  /**
   * Create a new session
   */
  async createSession(
    namespace: string,
    sessionData: CreateSessionRequest
  ): Promise<Session> {
    return this.request<Session>(
      `/v1/namespaces/${namespace}/sessions`,
      {
        method: 'POST',
        body: JSON.stringify(sessionData),
      }
    )
  }

  /**
   * Update a session
   */
  async updateSession(
    namespace: string,
    sessionName: string,
    updates: UpdateSessionRequest
  ): Promise<Session> {
    return this.request<Session>(
      `/v1/namespaces/${namespace}/sessions/${sessionName}`,
      {
        method: 'PATCH',
        body: JSON.stringify(updates),
      }
    )
  }

  /**
   * Delete a session
   */
  async deleteSession(namespace: string, sessionName: string): Promise<void> {
    return this.request<void>(
      `/v1/namespaces/${namespace}/sessions/${sessionName}`,
      {
        method: 'DELETE',
      }
    )
  }

  /**
   * Stop a running session
   */
  async stopSession(namespace: string, sessionName: string): Promise<void> {
    return this.request<void>(
      `/v1/namespaces/${namespace}/sessions/${sessionName}/stop`,
      {
        method: 'POST',
      }
    )
  }

  /**
   * Get session logs
   */
  async getSessionLogs(
    namespace: string,
    sessionName: string,
    options: { follow?: boolean; tail?: number } = {}
  ): Promise<{ logs: string[] }> {
    const queryString = this.buildQueryString(options)
    return this.request<{ logs: string[] }>(
      `/v1/namespaces/${namespace}/sessions/${sessionName}/logs${queryString}`
    )
  }

  /**
   * Get session artifacts
   */
  async getSessionArtifacts(
    namespace: string,
    sessionName: string
  ): Promise<{
    artifacts: Array<{
      name: string
      type: string
      size: string
      url: string
    }>
  }> {
    return this.request<{
      artifacts: Array<{
        name: string
        type: string
        size: string
        url: string
      }>
    }>(`/v1/namespaces/${namespace}/sessions/${sessionName}/artifacts`)
  }

  // Webhook APIs

  /**
   * Handle webhook events
   */
  async handleWebhook(
    source: string,
    payload: Record<string, any>
  ): Promise<void> {
    return this.request<void>(`/v1/webhooks/${source}`, {
      method: 'POST',
      body: JSON.stringify(payload),
    })
  }

  /**
   * Validate webhook configuration
   */
  async validateWebhook(
    source: string,
    config: Record<string, any>
  ): Promise<{ valid: boolean; errors?: string[] }> {
    return this.request<{ valid: boolean; errors?: string[] }>(
      `/v1/webhooks/${source}/validate`,
      {
        method: 'POST',
        body: JSON.stringify(config),
      }
    )
  }

  // Health and Status APIs

  /**
   * Check API health
   */
  async healthCheck(): Promise<{ status: string; version?: string }> {
    return this.request<{ status: string; version?: string }>('/health')
  }

  /**
   * Get API version information
   */
  async getVersion(): Promise<{ version: string; build?: string }> {
    return this.request<{ version: string; build?: string }>('/version')
  }
}

// Default singleton instance
let defaultApiClient: ApiClient

/**
 * Get the default API client instance
 */
export function getApiClient(config?: ApiClientConfig): ApiClient {
  if (!defaultApiClient || config) {
    defaultApiClient = new ApiClient(config)
  }
  return defaultApiClient
}

/**
 * Initialize API client with auth token
 */
export function initializeApiClient(authToken?: string): ApiClient {
  const client = getApiClient()

  if (authToken) {
    client.setAuthToken(authToken)
  } else if (typeof window !== 'undefined') {
    // Try to get token from localStorage
    const token = localStorage.getItem('authToken')
    if (token) {
      client.setAuthToken(token)
    }
  }

  return client
}

// Export types
export type { ApiClient }
export { ApiClientError }