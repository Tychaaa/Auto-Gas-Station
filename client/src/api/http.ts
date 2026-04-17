import { apiErrorSchema } from '@/schemas/transaction.schema'
import type { ApiErrorResponse } from '@/types'

// Базовый путь к API по умолчанию
const DEFAULT_API_BASE_URL = '/api/v1'

// Убирает лишний пробел и завершающий слеш
function normalizeBaseUrl(rawBaseUrl: string): string {
  const trimmed = rawBaseUrl.trim()
  if (trimmed.length === 0) {
    return DEFAULT_API_BASE_URL
  }
  return trimmed.endsWith('/') ? trimmed.slice(0, -1) : trimmed
}

export const API_BASE_URL = normalizeBaseUrl(import.meta.env.VITE_API_BASE_URL ?? DEFAULT_API_BASE_URL)

// Дополнительные данные для клиентской ошибки API
export interface ApiClientErrorOptions {
  status?: number
  serverError?: ApiErrorResponse
  cause?: unknown
}

// Ошибка для сетевых запросов и ответов сервера
export class ApiClientError extends Error {
  status?: number
  serverError?: ApiErrorResponse
  override cause?: unknown

  constructor(message: string, options?: ApiClientErrorOptions) {
    super(message)
    this.name = 'ApiClientError'
    this.status = options?.status
    this.serverError = options?.serverError
    this.cause = options?.cause
  }
}

export type HttpMethod = 'GET' | 'POST' | 'PUT'

// Общие параметры HTTP-запроса
export interface RequestOptions {
  body?: unknown
  signal?: AbortSignal
  headers?: Record<string, string>
}

// Собирает полный URL для запроса
function buildUrl(path: string): string {
  if (path.startsWith('http://') || path.startsWith('https://')) {
    return path
  }
  const normalizedPath = path.startsWith('/') ? path : `/${path}`
  return `${API_BASE_URL}${normalizedPath}`
}

// Пытается разобрать JSON и возвращает текст если не получилось
function tryParseJson(text: string): unknown {
  if (!text) {
    return undefined
  }
  try {
    return JSON.parse(text)
  } catch {
    return text
  }
}

// Приводит любую ошибку к одному типу
function toApiClientError(error: unknown): ApiClientError {
  if (error instanceof ApiClientError) {
    return error
  }
  if (error instanceof Error) {
    return new ApiClientError(error.message, { cause: error })
  }
  return new ApiClientError('Unknown network error', { cause: error })
}

// Выполняет HTTP-запрос и обрабатывает ошибки
export async function httpRequest<TResponse>(
  method: HttpMethod,
  path: string,
  options: RequestOptions = {},
): Promise<TResponse> {
  const { body, signal, headers } = options

  const requestInit: RequestInit = {
    method,
    signal,
    headers: {
      ...(body !== undefined ? { 'Content-Type': 'application/json' } : {}),
      ...headers,
    },
    body: body !== undefined ? JSON.stringify(body) : undefined,
  }

  try {
    const response = await fetch(buildUrl(path), requestInit)
    const rawText = await response.text()
    const parsedBody = tryParseJson(rawText)

    if (!response.ok) {
      const parsedError = apiErrorSchema.safeParse(parsedBody)
      const serverError = parsedError.success ? parsedError.data : undefined
      throw new ApiClientError(serverError?.error ?? `Request failed with status ${response.status}`, {
        status: response.status,
        serverError,
      })
    }

    return parsedBody as TResponse
  } catch (error) {
    throw toApiClientError(error)
  }
}

// Упрощенный GET-запрос
export function httpGet<TResponse>(path: string, options?: Omit<RequestOptions, 'body'>): Promise<TResponse> {
  return httpRequest<TResponse>('GET', path, options)
}

// Упрощенный POST-запрос
export function httpPost<TResponse>(path: string, body?: unknown, options?: Omit<RequestOptions, 'body'>): Promise<TResponse> {
  return httpRequest<TResponse>('POST', path, { ...options, body })
}

// Упрощенный PUT-запрос
export function httpPut<TResponse>(path: string, body?: unknown, options?: Omit<RequestOptions, 'body'>): Promise<TResponse> {
  return httpRequest<TResponse>('PUT', path, { ...options, body })
}
