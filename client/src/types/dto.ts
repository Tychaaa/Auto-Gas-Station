import type { FuelPrice, SelectionPayload, Transaction } from './transaction'

// Тело запросов для создания и обновления выбора
export type CreateTransactionRequest = SelectionPayload
export type UpdateSelectionRequest = SelectionPayload

// Ответы API с текущим состоянием транзакции
export type CreateTransactionResponse = Transaction
export type UpdateSelectionResponse = Transaction
export type GetTransactionResponse = Transaction
export type PaymentStartResponse = Transaction
export type PaymentStatusResponse = Transaction
export type FuelPricesResponse = FuelPrice[]

// Ошибка, которую может вернуть сервер
export interface ApiErrorResponse {
  error: string
  route?: string
}
