import type { KioskState } from '@/types/kioskState'

import { ApiClientError, httpGet, httpPost } from './http'

// Все admin-ручки защищены Basic Auth на сервере.
// На практике браузер может не показывать системный Basic Auth диалог для fetch-запросов,
// поэтому делаем fallback: при первом 401 просим логин/пароль и повторяем запрос.
let cachedAuthorizationHeader: string | null = null
let adminSessionVerified = false
let ensureAdminSessionPromise: Promise<boolean> | null = null

function encodeBasicAuth(username: string, password: string): string {
  return `Basic ${btoa(`${username}:${password}`)}`
}

function askAdminCredentials(): { username: string; password: string } | null {
  const username = window.prompt('Логин администратора')
  if (!username) {
    return null
  }
  const password = window.prompt('Пароль администратора')
  if (password === null) {
    return null
  }
  return { username: username.trim(), password }
}

async function adminGet<T>(path: string): Promise<T> {
  return adminRequest(() => httpGet<T>(path, authOptions()))
}

async function adminPost<T>(path: string, payload?: unknown): Promise<T> {
  return adminRequest(() => httpPost<T>(path, payload, authOptions()))
}

function authOptions(): { headers?: Record<string, string> } {
  if (!cachedAuthorizationHeader) {
    return {}
  }
  return {
    headers: {
      Authorization: cachedAuthorizationHeader,
    },
  }
}

async function adminRequest<T>(request: () => Promise<T>): Promise<T> {
  try {
    const result = await request()
    adminSessionVerified = true
    return result
  } catch (error) {
    const isUnauthorized = error instanceof ApiClientError && error.status === 401
    if (!isUnauthorized) {
      throw error
    }
    adminSessionVerified = false

    while (true) {
      const credentials = askAdminCredentials()
      if (!credentials) {
        throw new ApiClientError('Требуется авторизация администратора')
      }

      cachedAuthorizationHeader = encodeBasicAuth(credentials.username, credentials.password)
      try {
        const retryResult = await request()
        adminSessionVerified = true
        return retryResult
      } catch (retryError) {
        const retryUnauthorized = retryError instanceof ApiClientError && retryError.status === 401
        if (retryUnauthorized) {
          cachedAuthorizationHeader = null
          adminSessionVerified = false
          window.alert('Неверный логин или пароль. Попробуйте снова.')
          continue
        }
        throw retryError
      }
    }
  }
}

// ensureAdminSession блокирует вход в /admin до успешной авторизации.
// Если пользователь отменил ввод логина/пароля или ввел неверные данные,
// вернет false и роутер не пустит в админку.
export async function ensureAdminSession(): Promise<boolean> {
  if (adminSessionVerified) {
    return true
  }
  if (ensureAdminSessionPromise) {
    return ensureAdminSessionPromise
  }

  ensureAdminSessionPromise = (async () => {
    try {
      await adminGet<AdminPriceVersionsResponse>('/admin/prices/versions')
      adminSessionVerified = true
      return true
    } catch {
      adminSessionVerified = false
      return false
    } finally {
      ensureAdminSessionPromise = null
    }
  })()

  return ensureAdminSessionPromise
}

export interface AdminSetMaintenanceRequest {
  enabled: boolean
  reason?: string
}

// Переключает режим тех работ на киоске
// Возвращает актуальный KioskState чтобы UI сразу обновил индикатор
export async function setMaintenance(payload: AdminSetMaintenanceRequest): Promise<KioskState> {
  return adminPost<KioskState>('/admin/maintenance', payload)
}

export interface AdminPriceVersionItem {
  fuelType: string
  displayName: string
  grade: string
  pricePerLiter: number
  currency: string
}

export interface AdminPriceVersion {
  id: number
  versionTag: string
  effectiveFrom: string
  createdAt: string
  items: AdminPriceVersionItem[]
}

interface AdminPriceVersionsResponse {
  items: AdminPriceVersion[]
}

// Получает историю версий цен от самой свежей к старой
export async function listPriceVersions(): Promise<AdminPriceVersion[]> {
  const response = await adminGet<AdminPriceVersionsResponse>('/admin/prices/versions')
  return response.items ?? []
}

export interface AdminCreatePriceVersionItem {
  fuelType: string
  pricePerLiter: number
}

export interface AdminCreatePriceVersionRequest {
  versionTag?: string
  effectiveFrom: string
  items: AdminCreatePriceVersionItem[]
}

// Создает новую версию цен. Сервер проставит displayName/grade сам из справочника
export async function createPriceVersion(
  payload: AdminCreatePriceVersionRequest,
): Promise<AdminPriceVersion> {
  return adminPost<AdminPriceVersion>('/admin/prices/versions', payload)
}

export interface AdminTransactionView {
  id: string
  createdAt: string
  fuelType: string
  liters: number
  amountRub: number
  status: string
  paymentStatus: string
  fiscalStatus: string
  receiptNumber: string
  errorMessage: string
}

interface AdminTransactionsResponse {
  items: AdminTransactionView[]
}

// Возвращает список транзакций для админской таблицы
// Пока на сервере возвращаются захардкоженные примеры (см. TODO в admin_handlers.go)
export async function listAdminTransactions(): Promise<AdminTransactionView[]> {
  const response = await adminGet<AdminTransactionsResponse>('/admin/transactions')
  return response.items ?? []
}
