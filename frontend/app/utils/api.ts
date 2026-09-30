import type {
  ApiErrorBody, Comment, CommentChallenge, CommentInput, CommentLike, CommentSubmission,
  FeedPage, Health, IDResolution, Post, PostLike,
} from '../types/api.ts'

export class ApiError extends Error {
  status: number
  code: string
  retryAfter: number | null

  constructor(status: number, code: string, message: string, retryAfter: number | null = null) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.retryAfter = retryAfter
  }
}

function isErrorBody(value: unknown): value is ApiErrorBody {
  if (!value || typeof value !== 'object' || !('error' in value)) return false
  const detail = value.error
  return !!detail && typeof detail === 'object'
    && 'code' in detail && typeof detail.code === 'string'
    && 'message' in detail && typeof detail.message === 'string'
}

export interface RequestOptions { signal?: AbortSignal }
export interface FeedQuery { cursor?: string, limit?: number }

// Native fetch works in Nitro and browsers; useAsyncData owns SSR payload/deduplication.
export function createPublicApi(baseURL: string, fetcher: typeof fetch = globalThis.fetch) {
  const base = baseURL.replace(/\/+$/, '')

  async function request<T>(path: string, options: RequestOptions = {}, body?: object): Promise<T> {
    let response: Response
    try {
      response = await fetcher(`${base}${path}`, {
        method: body === undefined ? 'GET' : 'POST',
        headers: body === undefined ? { Accept: 'application/json' } : { Accept: 'application/json', 'Content-Type': 'application/json' },
        body: body === undefined ? undefined : JSON.stringify(body),
        credentials: 'same-origin',
        signal: options.signal ?? AbortSignal.timeout(15_000),
      })
    }
    catch (error) {
      if (options.signal?.aborted) throw error
      throw new ApiError(0, 'network_error', 'API is unreachable.')
    }

    const data: unknown = await response.json().catch((error: unknown) => {
      if (options.signal?.aborted) throw error
      return null
    })
    if (!response.ok) {
      const rawRetry = response.headers.get('Retry-After')
      const retry = rawRetry === null ? NaN : Number(rawRetry)
      throw new ApiError(
        response.status,
        isErrorBody(data) ? data.error.code : 'http_error',
        isErrorBody(data) ? data.error.message : 'API request failed.',
        Number.isFinite(retry) && retry >= 0 ? retry : null,
      )
    }
    if (data === null) throw new ApiError(502, 'invalid_response', 'API returned invalid JSON.')
    return data as T
  }

  return {
    health: (options?: RequestOptions) => request<Health>('/health', options),
    feed: (query: FeedQuery = {}, options?: RequestOptions) => {
      const params = new URLSearchParams()
      if (query.limit !== undefined) params.set('limit', String(query.limit))
      if (query.cursor !== undefined) params.set('cursor', query.cursor)
      return request<FeedPage>(`/posts${params.size ? `?${params}` : ''}`, options)
    },
    post: (slug: string, options?: RequestOptions) => request<Post>(`/posts/${encodeURIComponent(slug)}`, options),
    resolvePost: (id: number, options?: RequestOptions) => request<IDResolution>(`/posts/by-id/${id}`, options),
    comments: (id: number, options?: RequestOptions) => request<Comment[]>(`/posts/${id}/comments`, options),
    challenge: (id: number, options?: RequestOptions) => request<CommentChallenge>(`/posts/${id}/comments/challenge`, options),
    createComment: (id: number, input: CommentInput, options?: RequestOptions) => request<CommentSubmission>(`/posts/${id}/comments`, options, input),
    likePost: (id: number, options?: RequestOptions) => request<PostLike>(`/posts/${id}/likes`, options, {}),
    likeComment: (id: number, options?: RequestOptions) => request<CommentLike>(`/comments/${id}/likes`, options, {}),
  }
}
