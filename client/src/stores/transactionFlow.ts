import { computed, ref } from 'vue'
import { defineStore } from 'pinia'

import {
  ApiClientError,
  createTransaction,
  getFuelingProgress,
  getPaymentStatus,
  getTransaction,
  startFueling,
  startPayment,
  updateSelection,
} from '@/api'
import type { FuelingStartRequest, OrderSummary, SelectionPayload, Transaction } from '@/types'

// Ошибка для отображения в UI
export interface TransactionFlowError {
  message: string
  status?: number
  route?: string
}

// Начальные значения выбора перед созданием транзакции
const DEFAULT_SELECTION_DRAFT: SelectionPayload = {
  fuelType: '',
  orderMode: 'amount',
  amountRub: 0,
  liters: 0,
}

type FuelingConfig = Required<FuelingStartRequest>

const DEFAULT_FUELING_CONFIG: FuelingConfig = {
  pumpId: '1',
  nozzleId: '1',
  scenario: '',
}

export const useTransactionFlowStore = defineStore('transactionFlow', () => {
  // Основное состояние сценария транзакции
  const transaction = ref<Transaction | null>(null)
  const transactionId = ref<string | null>(null)
  const selectionDraft = ref<SelectionPayload>({ ...DEFAULT_SELECTION_DRAFT })

  // Флаги сетевых действий и поллинга
  const isSubmittingSelection = ref(false)
  const isStartingPayment = ref(false)
  const isStartingFueling = ref(false)
  const isPollingPayment = ref(false)
  const isPollingFueling = ref(false)
  const pollingTimerId = ref<number | null>(null)
  const fuelingPollingTimerId = ref<number | null>(null)
  const isPollingRequestInFlight = ref(false)
  const isFuelingPollingRequestInFlight = ref(false)
  const fuelingConfig = ref<FuelingConfig>({ ...DEFAULT_FUELING_CONFIG })

  // Последняя ошибка для интерфейса
  const lastError = ref<TransactionFlowError | null>(null)

  // Короткие вычисляемые признаки состояния
  const hasActiveTransaction = computed(() => transactionId.value !== null)
  const isPaymentPending = computed(() => transaction.value?.status === 'payment_pending')
  const isPaid = computed(() => transaction.value?.status === 'paid')
  const isFailed = computed(() => transaction.value?.status === 'failed')

  // Проверяет корректность выбранных параметров
  const isSelectionDraftValid = computed(() => {
    const draft = selectionDraft.value
    const hasFuelType = draft.fuelType.trim().length > 0

    const modeMatchesValue =
      (draft.orderMode === 'amount' && draft.amountRub > 0 && draft.liters === 0) ||
      (draft.orderMode === 'liters' && draft.liters > 0 && draft.amountRub === 0)

    return hasFuelType && modeMatchesValue
  })

  // Можно ли запускать оплату прямо сейчас
  const canStartPayment = computed(() => {
    const hasSelectionState = transaction.value?.status === 'selection'
    return (
      hasSelectionState &&
      isSelectionDraftValid.value &&
      !isSubmittingSelection.value &&
      !isStartingPayment.value &&
      !isPollingPayment.value
    )
  })

  const canStartFueling = computed(() => {
    return (
      transaction.value?.status === 'paid' &&
      !isStartingFueling.value &&
      !isPollingFueling.value &&
      !isFuelingPollingRequestInFlight.value
    )
  })

  const orderSummary = computed<OrderSummary>(() => {
    const transactionFuelType = transaction.value?.fuelType.trim() ?? ''
    const draftFuelType = selectionDraft.value.fuelType.trim()
    const fuelType = transactionFuelType || draftFuelType || null

    const transactionLiters = transaction.value?.liters ?? 0
    const draftLiters = selectionDraft.value.liters
    const liters = transactionLiters > 0 ? transactionLiters : draftLiters > 0 ? draftLiters : null

    const unitPriceMinor = transaction.value?.unitPriceMinor ?? 0
    const unitPrice = unitPriceMinor > 0 ? unitPriceMinor / 100 : null

    const computedAmountMinor = transaction.value?.computedAmountMinor ?? 0
    let totalAmount: number | null = computedAmountMinor > 0 ? computedAmountMinor / 100 : null
    if (totalAmount === null) {
      if (selectionDraft.value.orderMode === 'amount' && selectionDraft.value.amountRub > 0) {
        totalAmount = selectionDraft.value.amountRub
      }
    }

    const calculatedLiters =
      liters === null && unitPrice !== null && totalAmount !== null && unitPrice > 0
        ? Math.ceil((totalAmount / unitPrice) * 100) / 100
        : null

    return {
      fuelType,
      liters: calculatedLiters ?? liters,
      unitPrice,
      totalAmount,
      isComplete: fuelType !== null && (calculatedLiters ?? liters) !== null && unitPrice !== null && totalAmount !== null,
    }
  })

  // Сбрасывает текущую ошибку
  function clearError(): void {
    lastError.value = null
  }

  // Приводит ошибку к единому формату store
  function setStoreError(error: unknown): TransactionFlowError {
    const normalized =
      error instanceof ApiClientError
        ? {
            message: error.serverError?.error ?? error.message,
            status: error.status,
            route: error.serverError?.route,
          }
        : error instanceof Error
          ? { message: error.message }
          : { message: 'Unknown store error' }

    lastError.value = normalized
    return normalized
  }

  // Сохраняет транзакцию и синхронизирует черновик выбора
  function applyTransaction(nextTransaction: Transaction): void {
    transaction.value = nextTransaction
    transactionId.value = nextTransaction.id
    selectionDraft.value = {
      fuelType: nextTransaction.fuelType,
      orderMode: nextTransaction.orderMode,
      amountRub: nextTransaction.amountRub,
      liters: nextTransaction.liters,
    }
  }

  // Обновляет часть черновика выбора
  function setSelectionDraft(patch: Partial<SelectionPayload>): void {
    selectionDraft.value = {
      ...selectionDraft.value,
      ...patch,
    }
    clearError()
  }

  // Обновляет конфигурацию топливной колонки для API
  function setFuelingConfig(patch: Partial<FuelingConfig>): void {
    fuelingConfig.value = {
      ...fuelingConfig.value,
      ...patch,
    }
  }

  // Создает транзакцию или обновляет существующую
  async function submitSelection(): Promise<Transaction | null> {
    isSubmittingSelection.value = true
    clearError()

    try {
      const currentId = transactionId.value
      const nextTransaction = currentId
        ? await updateSelection(currentId, selectionDraft.value)
        : await createTransaction(selectionDraft.value)

      applyTransaction(nextTransaction)
      return nextTransaction
    } catch (error) {
      setStoreError(error)
      return null
    } finally {
      isSubmittingSelection.value = false
    }
  }

  // Запрашивает актуальное состояние транзакции
  async function refreshTransaction(): Promise<Transaction | null> {
    const currentId = transactionId.value
    if (!currentId) {
      lastError.value = { message: 'Transaction id is not set' }
      return null
    }

    clearError()
    try {
      const nextTransaction = await getTransaction(currentId)
      applyTransaction(nextTransaction)
      return nextTransaction
    } catch (error) {
      setStoreError(error)
      return null
    }
  }

  // Один запрос статуса оплаты с проверками
  async function pollPaymentStatusOnce(): Promise<Transaction | null> {
    if (isPollingRequestInFlight.value) {
      return transaction.value
    }
    if (!transactionId.value) {
      lastError.value = { message: 'Transaction id is not set' }
      stopPaymentPolling()
      return null
    }
    if (transaction.value?.status !== 'payment_pending') {
      stopPaymentPolling()
      return transaction.value
    }

    isPollingRequestInFlight.value = true
    clearError()

    try {
      const nextTransaction = await getPaymentStatus(transactionId.value)
      applyTransaction(nextTransaction)

      if (nextTransaction.status !== 'payment_pending') {
        stopPaymentPolling()
      }
      return nextTransaction
    } catch (error) {
      setStoreError(error)
      stopPaymentPolling()
      return null
    } finally {
      isPollingRequestInFlight.value = false
    }
  }

  // Запускает периодический опрос статуса оплаты
  function startPaymentPolling(intervalMs = 2000): void {
    if (isPollingPayment.value || pollingTimerId.value !== null) {
      return
    }

    isPollingPayment.value = true
    pollingTimerId.value = window.setInterval(() => {
      void pollPaymentStatusOnce()
    }, intervalMs)
  }

  // Останавливает опрос и очищает связанные флаги
  function stopPaymentPolling(): void {
    if (pollingTimerId.value !== null) {
      window.clearInterval(pollingTimerId.value)
      pollingTimerId.value = null
    }
    isPollingPayment.value = false
    isPollingRequestInFlight.value = false
  }

  // Запускает оплату и включает поллинг при необходимости
  async function startPaymentFlow(): Promise<Transaction | null> {
    const currentId = transactionId.value
    if (!currentId) {
      lastError.value = { message: 'Transaction id is not set' }
      return null
    }

    stopPaymentPolling()
    isStartingPayment.value = true
    clearError()

    try {
      const nextTransaction = await startPayment(currentId)
      applyTransaction(nextTransaction)

      if (nextTransaction.status === 'payment_pending') {
        startPaymentPolling()
      } else {
        stopPaymentPolling()
      }

      return nextTransaction
    } catch (error) {
      setStoreError(error)
      stopPaymentPolling()
      return null
    } finally {
      isStartingPayment.value = false
    }
  }

  // Один запрос прогресса отпуска топлива с проверками
  async function pollFuelingProgressOnce(): Promise<Transaction | null> {
    if (isFuelingPollingRequestInFlight.value) {
      return transaction.value
    }
    if (!transactionId.value) {
      lastError.value = { message: 'Transaction id is not set' }
      stopFuelingPolling()
      return null
    }
    if (transaction.value?.status !== 'fueling') {
      stopFuelingPolling()
      return transaction.value
    }

    isFuelingPollingRequestInFlight.value = true
    clearError()

    try {
      const nextTransaction = await getFuelingProgress(transactionId.value)
      applyTransaction(nextTransaction)

      if (nextTransaction.status !== 'fueling') {
        stopFuelingPolling()
      }

      return nextTransaction
    } catch (error) {
      setStoreError(error)
      stopFuelingPolling()
      return null
    } finally {
      isFuelingPollingRequestInFlight.value = false
    }
  }

  // Запускает периодический опрос прогресса заправки
  function startFuelingPolling(intervalMs = 500): void {
    if (isPollingFueling.value || fuelingPollingTimerId.value !== null) {
      return
    }

    isPollingFueling.value = true
    fuelingPollingTimerId.value = window.setInterval(() => {
      void pollFuelingProgressOnce()
    }, intervalMs)
  }

  // Останавливает опрос заправки и очищает флаги
  function stopFuelingPolling(): void {
    if (fuelingPollingTimerId.value !== null) {
      window.clearInterval(fuelingPollingTimerId.value)
      fuelingPollingTimerId.value = null
    }
    isPollingFueling.value = false
    isFuelingPollingRequestInFlight.value = false
  }

  // Запускает этап отпуска топлива и включает поллинг
  async function startFuelingFlow(): Promise<Transaction | null> {
    const currentId = transactionId.value
    if (!currentId) {
      lastError.value = { message: 'Transaction id is not set' }
      return null
    }

    if (transaction.value?.status === 'fueling') {
      startFuelingPolling()
      return transaction.value
    }
    if (transaction.value?.status !== 'paid') {
      lastError.value = { message: 'Fueling can only be started from paid status' }
      return null
    }

    stopFuelingPolling()
    isStartingFueling.value = true
    clearError()

    try {
      const nextTransaction = await startFueling(currentId, fuelingConfig.value)
      applyTransaction(nextTransaction)

      if (nextTransaction.status === 'fueling') {
        startFuelingPolling()
      } else {
        stopFuelingPolling()
      }

      return nextTransaction
    } catch (error) {
      setStoreError(error)
      stopFuelingPolling()
      return null
    } finally {
      isStartingFueling.value = false
    }
  }

  // Полностью сбрасывает сценарий транзакции
  function resetFlow(): void {
    stopPaymentPolling()
    stopFuelingPolling()
    transaction.value = null
    transactionId.value = null
    selectionDraft.value = { ...DEFAULT_SELECTION_DRAFT }
    fuelingConfig.value = { ...DEFAULT_FUELING_CONFIG }
    isSubmittingSelection.value = false
    isStartingPayment.value = false
    isStartingFueling.value = false
    clearError()
  }

  // Сбрасывает текущую транзакцию для повторной оплаты, сохраняя выбор пользователя
  function resetForPaymentRetry(): void {
    stopPaymentPolling()
    stopFuelingPolling()
    transaction.value = null
    transactionId.value = null
    isSubmittingSelection.value = false
    isStartingPayment.value = false
    isStartingFueling.value = false
    clearError()
  }

  return {
    transaction,
    transactionId,
    selectionDraft,
    isSubmittingSelection,
    isStartingPayment,
    isStartingFueling,
    isPollingPayment,
    isPollingFueling,
    pollingTimerId,
    fuelingPollingTimerId,
    lastError,
    fuelingConfig,
    hasActiveTransaction,
    isPaymentPending,
    isPaid,
    isFailed,
    isSelectionDraftValid,
    canStartPayment,
    canStartFueling,
    orderSummary,
    setSelectionDraft,
    setFuelingConfig,
    submitSelection,
    refreshTransaction,
    startPaymentFlow,
    startFuelingFlow,
    pollPaymentStatusOnce,
    pollFuelingProgressOnce,
    startPaymentPolling,
    startFuelingPolling,
    stopPaymentPolling,
    stopFuelingPolling,
    resetForPaymentRetry,
    resetFlow,
  }
})
