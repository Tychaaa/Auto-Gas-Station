import { z } from 'zod'

import { fiscalStatuses, fuelingStatuses, orderModes, paymentStatuses, transactionStatuses } from '@/types/transaction'

// Схемы для строковых статусов и режимов заказа
export const transactionStatusSchema = z.enum(transactionStatuses)
export const paymentStatusSchema = z.enum(paymentStatuses)
export const fiscalStatusSchema = z.enum(fiscalStatuses)
export const fuelingStatusSchema = z.enum(fuelingStatuses)
export const orderModeSchema = z.enum(orderModes)

// Данные выбора топлива и способа заправки
export const selectionPayloadSchema = z.object({
  fuelType: z.string(),
  orderMode: orderModeSchema,
  amountRub: z.number(),
  liters: z.number(),
  preset: z.string(),
})

// Полная схема транзакции из API
export const transactionSchema = z.object({
  id: z.string(),
  fuelType: z.string(),
  orderMode: orderModeSchema,
  amountRub: z.number(),
  liters: z.number(),
  preset: z.string(),
  priceVersionId: z.number().int(),
  priceVersionTag: z.string(),
  unitPriceMinor: z.number().int(),
  computedAmountMinor: z.number().int(),
  currency: z.string(),
  pricingSnapshotAt: z.string(),
  priceLockedUntil: z.string(),
  priceWasRepriced: z.boolean(),
  status: transactionStatusSchema,
  paymentStatus: paymentStatusSchema,
  fiscalStatus: fiscalStatusSchema,
  paymentProvider: z.string(),
  paymentSessionID: z.string(),
  paymentError: z.string(),
  fiscalError: z.string(),
  receiptNumber: z.string(),
  createdAt: z.string(),
  updatedAt: z.string(),
  fuelingStatus: fuelingStatusSchema,
  fuelingError: z.string(),
  fuelingSessionID: z.string(),
  dispensedLiters: z.number(),
  dispenseComplete: z.boolean(),
  dispensePartial: z.boolean(),
})

export const fuelPriceSchema = z.object({
  fuelType: z.string(),
  name: z.string(),
  grade: z.string(),
  pricePerLiter: z.number(),
  currency: z.string(),
  priceVersionId: z.number().int(),
  versionTag: z.string(),
  effectiveFrom: z.string(),
})

export const fuelPricesResponseSchema = z.object({
  items: z.array(fuelPriceSchema),
})

// Схема ошибки от сервера
export const apiErrorSchema = z.object({
  error: z.string(),
  route: z.string().optional(),
})

export type TransactionSchema = z.infer<typeof transactionSchema>
export type SelectionPayloadSchema = z.infer<typeof selectionPayloadSchema>
export type ApiErrorSchema = z.infer<typeof apiErrorSchema>
export type FuelPriceSchema = z.infer<typeof fuelPriceSchema>

// Проверяет и приводит ответ с транзакцией
export function parseTransactionResponse(input: unknown): TransactionSchema {
  return transactionSchema.parse(input)
}

// Проверяет и приводит ответ с ошибкой
export function parseApiError(input: unknown): ApiErrorSchema {
  return apiErrorSchema.parse(input)
}

export function parseFuelPricesResponse(input: unknown): FuelPriceSchema[] {
  return fuelPricesResponseSchema.parse(input).items
}
