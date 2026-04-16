import { z } from 'zod'

import { fiscalStatuses, fuelingStatuses, orderModes, paymentStatuses, transactionStatuses } from '@/types/transaction'

export const transactionStatusSchema = z.enum(transactionStatuses)
export const paymentStatusSchema = z.enum(paymentStatuses)
export const fiscalStatusSchema = z.enum(fiscalStatuses)
export const fuelingStatusSchema = z.enum(fuelingStatuses)
export const orderModeSchema = z.enum(orderModes)

export const selectionPayloadSchema = z.object({
  fuelType: z.string(),
  orderMode: orderModeSchema,
  amountRub: z.number(),
  liters: z.number(),
  preset: z.string(),
})

export const transactionSchema = z.object({
  id: z.string(),
  fuelType: z.string(),
  orderMode: orderModeSchema,
  amountRub: z.number(),
  liters: z.number(),
  preset: z.string(),
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

export const apiErrorSchema = z.object({
  error: z.string(),
  route: z.string().optional(),
})

export type TransactionSchema = z.infer<typeof transactionSchema>
export type SelectionPayloadSchema = z.infer<typeof selectionPayloadSchema>
export type ApiErrorSchema = z.infer<typeof apiErrorSchema>

export function parseTransactionResponse(input: unknown): TransactionSchema {
  return transactionSchema.parse(input)
}

export function parseApiError(input: unknown): ApiErrorSchema {
  return apiErrorSchema.parse(input)
}
