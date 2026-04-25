import type { Transaction } from '@/types'

// Формат транзакции в ответе от Go-сервера
interface BackendTransaction {
  ID: string
  FuelType: string
  OrderMode: string
  AmountRub: number
  Liters: number
  PriceVersionID: number
  PriceVersionTag: string
  UnitPriceMinor: number
  ComputedAmountMinor: number
  Currency: string
  PricingSnapshotAt: string
  PriceLockedUntil: string
  PriceWasRepriced: boolean
  Status: string
  PaymentStatus: string
  FiscalStatus: string
  PaymentProvider: string
  PaymentSessionID: string
  PaymentError: string
  FiscalError: string
  ReceiptNumber: string
  CreatedAt: string
  UpdatedAt: string
  FuelingStatus: string
  FuelingError: string
  FuelingSessionID: string
  DispensedLiters: number
  DispenseComplete: boolean
  DispensePartial: boolean
}

// Проверяет, что в нормализатор пришел объект
function ensureRecord(input: unknown): Record<string, unknown> {
  if (!input || typeof input !== 'object') {
    throw new Error('Transaction payload must be an object')
  }
  return input as Record<string, unknown>
}

// Приводит поля сервера к формату клиента
export function normalizeTransactionResponse(payload: unknown): Omit<Transaction, 'orderMode' | 'status' | 'paymentStatus' | 'fiscalStatus' | 'fuelingStatus'> & {
  orderMode: string
  status: string
  paymentStatus: string
  fiscalStatus: string
  fuelingStatus: string
} {
  const raw = ensureRecord(payload) as unknown as BackendTransaction

  return {
    id: raw.ID,
    fuelType: raw.FuelType,
    orderMode: raw.OrderMode,
    amountRub: raw.AmountRub,
    liters: raw.Liters,
    priceVersionId: raw.PriceVersionID,
    priceVersionTag: raw.PriceVersionTag,
    unitPriceMinor: raw.UnitPriceMinor,
    computedAmountMinor: raw.ComputedAmountMinor,
    currency: raw.Currency,
    pricingSnapshotAt: raw.PricingSnapshotAt,
    priceLockedUntil: raw.PriceLockedUntil,
    priceWasRepriced: raw.PriceWasRepriced,
    status: raw.Status,
    paymentStatus: raw.PaymentStatus,
    fiscalStatus: raw.FiscalStatus,
    paymentProvider: raw.PaymentProvider,
    paymentSessionID: raw.PaymentSessionID,
    paymentError: raw.PaymentError,
    fiscalError: raw.FiscalError,
    receiptNumber: raw.ReceiptNumber,
    createdAt: raw.CreatedAt,
    updatedAt: raw.UpdatedAt,
    fuelingStatus: raw.FuelingStatus,
    fuelingError: raw.FuelingError,
    fuelingSessionID: raw.FuelingSessionID,
    dispensedLiters: raw.DispensedLiters,
    dispenseComplete: raw.DispenseComplete,
    dispensePartial: raw.DispensePartial,
  }
}
