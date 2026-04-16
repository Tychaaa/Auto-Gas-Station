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

export const paymentStatuses = ['none', 'pending', 'approved', 'declined'] as const

export type PaymentStatus = (typeof paymentStatuses)[number]

export const fiscalStatuses = ['none', 'pending', 'done', 'failed'] as const

export type FiscalStatus = (typeof fiscalStatuses)[number]

export const fuelingStatuses = [
  'none',
  'starting',
  'dispensing',
  'completed_waiting_fiscal',
  'failed',
] as const

export type FuelingStatus = (typeof fuelingStatuses)[number]

export const orderModes = ['amount', 'liters', 'preset'] as const

export type OrderMode = (typeof orderModes)[number]

// Fuel type values are backend-configurable, so keep it open as string.
export type FuelType = string

export interface SelectionPayload {
  fuelType: FuelType
  orderMode: OrderMode
  amountRub: number
  liters: number
  preset: string
}

export interface Transaction {
  id: string
  fuelType: FuelType
  orderMode: OrderMode
  amountRub: number
  liters: number
  preset: string
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
