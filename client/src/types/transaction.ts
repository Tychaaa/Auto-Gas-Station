// Возможные этапы жизненного цикла транзакции
export const transactionStatuses = [
  'selection',
  'payment_pending',
  'paid',
  'fueling',
  'fiscalizing',
  'completed',
  'failed',
] as const

export type TransactionStatus = (typeof transactionStatuses)[number]

// Состояния оплаты
export const paymentStatuses = ['none', 'pending', 'approved', 'declined'] as const

export type PaymentStatus = (typeof paymentStatuses)[number]

// Состояния фискализации
export const fiscalStatuses = ['none', 'pending', 'done', 'failed'] as const

export type FiscalStatus = (typeof fiscalStatuses)[number]

// Состояния процесса отпуска топлива
export const fuelingStatuses = [
  'none',
  'starting',
  'dispensing',
  'completed_waiting_fiscal',
  'failed',
] as const

export type FuelingStatus = (typeof fuelingStatuses)[number]

// Режимы оформления заправки
export const orderModes = ['amount', 'liters', 'preset'] as const

export type OrderMode = (typeof orderModes)[number]

// Тип топлива приходит с сервера, поэтому оставляем строку
export type FuelType = string

// Данные, которые пользователь выбирает перед оплатой
export interface SelectionPayload {
  fuelType: FuelType
  orderMode: OrderMode
  amountRub: number
  liters: number
  preset: string
}

// Полное состояние транзакции на клиенте
export interface Transaction {
  id: string
  fuelType: FuelType
  orderMode: OrderMode
  amountRub: number
  liters: number
  preset: string
  priceVersionId: number
  priceVersionTag: string
  unitPriceMinor: number
  computedAmountMinor: number
  currency: string
  pricingSnapshotAt: string
  priceLockedUntil: string
  priceWasRepriced: boolean
  status: TransactionStatus
  paymentStatus: PaymentStatus
  fiscalStatus: FiscalStatus
  paymentProvider: string
  paymentSessionID: string
  paymentError: string
  fiscalError: string
  receiptNumber: string
  createdAt: string
  updatedAt: string
  fuelingStatus: FuelingStatus
  fuelingError: string
  fuelingSessionID: string
  dispensedLiters: number
  dispenseComplete: boolean
  dispensePartial: boolean
}

export interface FuelPrice {
  fuelType: FuelType
  name: string
  grade: string
  pricePerLiter: number
  currency: string
  priceVersionId: number
  versionTag: string
  effectiveFrom: string
}
