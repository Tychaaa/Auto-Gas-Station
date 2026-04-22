import type { SelectionPayload, Transaction } from './transaction'

// Тело запросов для создания и обновления выбора
export type CreateTransactionRequest = SelectionPayload
export type UpdateSelectionRequest = SelectionPayload

// Ответы API с текущим состоянием транзакции
export type CreateTransactionResponse = Transaction
export type UpdateSelectionResponse = Transaction
export type GetTransactionResponse = Transaction
export type PaymentStartResponse = Transaction
export type PaymentStatusResponse = Transaction

// Тело запроса для старта отпуска топлива
export interface FuelingStartRequest {
  pumpId: string
  nozzleId: string
  scenario?: string
}

// Тело ответа старта отпуска топлива
export interface FuelingStartApiResponse {
  fuelingStarted: boolean
  providerStatus: string
  fuelingSessionId: string
  transaction: Transaction
}

// Тело ответа прогресса отпуска топлива
export interface FuelingProgressApiResponse {
  providerStatus: string
  transaction: Transaction
}

// Ответы клиентского API после нормализации
export type FuelingStartResponse = Transaction
export type FuelingProgressResponse = Transaction

// Ошибка, которую может вернуть сервер
export interface ApiErrorResponse {
  error: string
  route?: string
}
