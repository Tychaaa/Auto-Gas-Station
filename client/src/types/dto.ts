import type { SelectionPayload, Transaction } from './transaction'

export type CreateTransactionRequest = SelectionPayload
export type UpdateSelectionRequest = SelectionPayload

export type CreateTransactionResponse = Transaction
export type UpdateSelectionResponse = Transaction
export type GetTransactionResponse = Transaction
export type PaymentStartResponse = Transaction
export type PaymentStatusResponse = Transaction

export interface ApiErrorResponse {
  error: string
  route?: string
}
