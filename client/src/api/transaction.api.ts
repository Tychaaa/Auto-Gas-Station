import { parseTransactionResponse, selectionPayloadSchema } from '@/schemas/transaction.schema'
import type {
  CreateTransactionRequest,
  CreateTransactionResponse,
  FuelingProgressResponse,
  FuelingStartRequest,
  FuelingStartResponse,
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

// Достает вложенную транзакцию из ответа fueling API
function parseFuelingEnvelopeTransaction(payload: unknown): FuelingStartResponse {
  if (!payload || typeof payload !== 'object') {
    throw new Error('Fueling payload must be an object')
  }

  const transactionPayload = (payload as Record<string, unknown>).transaction
  return parseNormalizedTransaction(transactionPayload)
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

// Запускает отпуск топлива для транзакции
export async function startFueling(transactionId: string, payload: FuelingStartRequest): Promise<FuelingStartResponse> {
  const response = await httpPost<unknown>(`/transactions/${encodeTransactionId(transactionId)}/fueling/start`, payload)
  return parseFuelingEnvelopeTransaction(response)
}

// Запрашивает прогресс отпуска топлива
export async function getFuelingProgress(transactionId: string): Promise<FuelingProgressResponse> {
  const response = await httpPost<unknown>(`/transactions/${encodeTransactionId(transactionId)}/fueling/progress`)
  return parseFuelingEnvelopeTransaction(response)
}
