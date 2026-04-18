import { computed, ref } from 'vue'
import { defineStore } from 'pinia'

import {
  ApiClientError,
  createTransaction,
  getPaymentStatus,
  getTransaction,
  startPayment,
  updateSelection,
} from '@/api'
import type { SelectionPayload, Transaction } from '@/types'

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
  preset: '',
}

export const useTransactionFlowStore = defineStore('transactionFlow', () => {
  // Основное состояние сценария транзакции
  const transaction = ref<Transaction | null>(null)
  const transactionId = ref<string | null>(null)
  const selectionDraft = ref<SelectionPayload>({ ...DEFAULT_SELECTION_DRAFT })

  // Флаги сетевых действий и поллинга
  const isSubmittingSelection = ref(false)
  const isStartingPayment = ref(false)
  const isPollingPayment = ref(false)
  const pollingTimerId = ref<number | null>(null)
  const isPollingRequestInFlight = ref(false)

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
      (draft.orderMode === 'amount' && draft.amountRub > 0 && draft.liters === 0 && draft.preset === '') ||
      (draft.orderMode === 'liters' && draft.liters > 0 && draft.amountRub === 0 && draft.preset === '') ||
      (draft.orderMode === 'preset' && draft.preset.trim().length > 0 && draft.amountRub === 0 && draft.liters === 0)

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
      preset: nextTransaction.preset,
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

  // Полностью сбрасывает сценарий транзакции
  function resetFlow(): void {
    stopPaymentPolling()
    transaction.value = null
    transactionId.value = null
    selectionDraft.value = { ...DEFAULT_SELECTION_DRAFT }
    isSubmittingSelection.value = false
    isStartingPayment.value = false
    clearError()
  }

  return {
    transaction,
    transactionId,
    selectionDraft,
    isSubmittingSelection,
    isStartingPayment,
    isPollingPayment,
    pollingTimerId,
    lastError,
    hasActiveTransaction,
    isPaymentPending,
    isPaid,
    isFailed,
    isSelectionDraftValid,
    canStartPayment,
    setSelectionDraft,
    submitSelection,
    refreshTransaction,
    startPaymentFlow,
    pollPaymentStatusOnce,
    startPaymentPolling,
    stopPaymentPolling,
    resetFlow,
  }
})
