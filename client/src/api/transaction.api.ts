import { parseTransactionResponse, selectionPayloadSchema } from '@/schemas/transaction.schema'
import type {
  CreateTransactionRequest,
  CreateTransactionResponse,
  GetTransactionResponse,
  PaymentStartResponse,
  PaymentStatusResponse,
  UpdateSelectionRequest,
  UpdateSelectionResponse,
} from '@/types'

import { httpGet, httpPost, httpPut } from './http'
import { normalizeTransactionResponse } from './normalizers/transaction.normalizer'

// Нормализует ответ сервера и проверяет его схемой
function parseNormalizedTransaction(payload: unknown): CreateTransactionResponse {
  const normalized = normalizeTransactionResponse(payload)
  return parseTransactionResponse(normalized)
}

// Проверяет данные выбора перед отправкой на сервер
function validateSelectionPayload<TPayload extends CreateTransactionRequest | UpdateSelectionRequest>(payload: TPayload): TPayload {
  return selectionPayloadSchema.parse(payload) as TPayload
}

// Безопасно кодирует идентификатор для URL
function encodeTransactionId(transactionId: string): string {
  return encodeURIComponent(transactionId)
}

// Создает новую транзакцию
export async function createTransaction(payload: CreateTransactionRequest): Promise<CreateTransactionResponse> {
  const body = validateSelectionPayload(payload)
  const response = await httpPost<unknown>('/transactions', body)
  return parseNormalizedTransaction(response)
}

// Получает транзакцию по идентификатору
export async function getTransaction(transactionId: string): Promise<GetTransactionResponse> {
  const response = await httpGet<unknown>(`/transactions/${encodeTransactionId(transactionId)}`)
  return parseNormalizedTransaction(response)
}

// Обновляет выбранные параметры заправки
export async function updateSelection(
  transactionId: string,
  payload: UpdateSelectionRequest,
): Promise<UpdateSelectionResponse> {
  const body = validateSelectionPayload(payload)
  const response = await httpPut<unknown>(`/transactions/${encodeTransactionId(transactionId)}/selection`, body)
  return parseNormalizedTransaction(response)
}

// Запускает оплату для транзакции
export async function startPayment(transactionId: string): Promise<PaymentStartResponse> {
  const response = await httpPost<unknown>(`/transactions/${encodeTransactionId(transactionId)}/payment/start`)
  return parseNormalizedTransaction(response)
}

// Запрашивает текущий статус оплаты
export async function getPaymentStatus(transactionId: string): Promise<PaymentStatusResponse> {
  const response = await httpPost<unknown>(`/transactions/${encodeTransactionId(transactionId)}/payment/status`)
  return parseNormalizedTransaction(response)
}
