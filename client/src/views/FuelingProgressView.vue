<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'

import StepIndicator from '@/components/StepIndicator.vue'
import { useTransactionFlowStore } from '@/stores'

const STEPS = ['Топливо', 'Параметры', 'Оплата', 'Заправка'] as const

const rubFormatter = new Intl.NumberFormat('ru-RU', {
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
})

type ScreenMode = 'ready' | 'no-transaction' | 'wrong-stage'

interface FuelingUiState {
  title: string
  description: string
  providerStatus: string
  dispensedLiters: number
  dispensedAmount: number
  pricePerLiter: number
  targetLiters: number
  progressPercent: number
  badgeLabel: string
  badgeClass: string
}

const router = useRouter()
const store = useTransactionFlowStore()
const doneRedirectTimerId = ref<number | null>(null)
const doneRedirectScheduled = ref(false)
const FUELING_DONE_REDIRECT_DELAY_MS = 5000

const transaction = computed(() => store.transaction)
const errorMessage = computed(() => transaction.value?.fuelingError || store.lastError?.message || '')

const FUELING_STATUS_LABELS: Record<string, string> = {
  none: 'Нет данных',
  starting: 'Подготовка к отпуску',
  dispensing: 'Идет отпуск топлива',
  completed_waiting_fiscal: 'Отпуск завершен',
  failed: 'Ошибка отпуска топлива',
}

function formatProviderStatus(status: string): string {
  const normalized = status.trim().toLowerCase()
  if (!normalized) {
    return 'Нет данных'
  }
  if (FUELING_STATUS_LABELS[normalized]) {
    return FUELING_STATUS_LABELS[normalized]
  }

  // Фоллбек для новых/неожиданных статусов: "completed_waiting_fiscal" -> "Completed waiting fiscal".
  const humanized = normalized
    .replace(/[_-]+/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
  if (!humanized) {
    return 'Нет данных'
  }

  return humanized.charAt(0).toUpperCase() + humanized.slice(1)
}

const screenMode = computed<ScreenMode>(() => {
  if (!transaction.value) {
    return 'no-transaction'
  }

  const status = transaction.value.status
  
  if (status !== 'paid' && status !== 'fueling' && status !== 'fiscalizing' && status !== 'completed' && status !== 'failed') {
    return 'wrong-stage'
  }

  return 'ready'
})

const uiState = computed<FuelingUiState>(() => {
  const tx = transaction.value

  if (!tx) {
    return {
      title: 'Транзакция не найдена',
      description: 'Для просмотра прогресса сначала начните сценарий заправки.',
      providerStatus: 'none',
      dispensedLiters: 0,
      dispensedAmount: 0,
      pricePerLiter: 0,
      targetLiters: 0,
      progressPercent: 0,
      badgeLabel: 'Нет данных',
      badgeClass: 'bg-fuel-olive/15 text-fuel-forest border border-fuel-olive/30',
    }
  }

  const targetLiters = tx.liters > 0 ? tx.liters : Math.max(store.orderSummary.liters ?? 0, 0)
  const fuelingStatus = tx.fuelingStatus
  const providerStatusLabel = formatProviderStatus(fuelingStatus)
  const pricePerLiter = (tx.unitPriceMinor ?? 0) / 100
  const dispensedAmount = tx.dispensedLiters * pricePerLiter
  const progressByVolume =
    targetLiters > 0
      ? Math.min(100, Math.max(0, Math.round((tx.dispensedLiters / targetLiters) * 100)))
      : 0

  if (tx.status === 'failed' || fuelingStatus === 'failed') {
    return {
      title: 'Не удалось завершить заправку',
      description: errorMessage.value || 'Произошла ошибка топливного контура. Обратитесь к оператору.',
      providerStatus: providerStatusLabel,
      dispensedLiters: tx.dispensedLiters,
      dispensedAmount,
      pricePerLiter,
      targetLiters,
      progressPercent: progressByVolume,
      badgeLabel: 'Ошибка процесса',
      badgeClass: 'bg-red-100 text-red-700 border border-red-200',
    }
  }

  if (tx.status === 'completed' || tx.status === 'fiscalizing' || fuelingStatus === 'completed_waiting_fiscal') {
    return {
      title: 'Заправка завершена',
      description: 'Операция полностью завершена. Спасибо за использование сервиса.',
      providerStatus: providerStatusLabel,
      dispensedLiters: tx.dispensedLiters,
      dispensedAmount,
      pricePerLiter,
      targetLiters,
      progressPercent: 100,
      badgeLabel: 'Завершено',
      badgeClass: 'bg-fuel-lime text-white border border-fuel-lime',
    }
  }

  if (fuelingStatus === 'starting') {
    return {
      title: 'Подготовка к заправке',
      description: 'Колонка принимает команду. Пожалуйста, зафиксируйте пистолет в баке.',
      providerStatus: providerStatusLabel,
      dispensedLiters: tx.dispensedLiters,
      dispensedAmount,
      pricePerLiter,
      targetLiters,
      progressPercent: tx.dispensedLiters > 0 ? progressByVolume : 10,
      badgeLabel: 'Подготовка колонки',
      badgeClass: 'bg-fuel-olive/15 text-fuel-forest border border-fuel-olive/30',
    }
  }

  return {
    title: 'Идет отпуск топлива',
    description: 'Топливо подается. Следите за прогрессом заправки.',
    providerStatus: providerStatusLabel,
    dispensedLiters: tx.dispensedLiters,
    dispensedAmount,
    pricePerLiter,
    targetLiters,
    progressPercent: progressByVolume,
    badgeLabel: 'Отпуск топлива',
    badgeClass: 'bg-fuel-lime/20 text-fuel-forest border border-fuel-lime/40',
  }
})

const shouldRedirectToDone = computed(() => {
  const tx = transaction.value
  if (!tx) {
    return false
  }
  return tx.status === 'completed' || tx.status === 'fiscalizing' || tx.fuelingStatus === 'completed_waiting_fiscal'
})

function goToFuelSelect(): void {
  void router.push('/select/fuel')
}

function goToOrderParams(): void {
  void router.push('/select/order')
}

onMounted(() => {
  if (transaction.value?.status === 'paid') {
    void store.startFuelingFlow()
    return
  }

  if (transaction.value?.status === 'fueling') {
    store.startFuelingPolling()
  }
})

watch(
  shouldRedirectToDone,
  (isCompleted) => {
    if (!isCompleted || doneRedirectScheduled.value) {
      return
    }

    doneRedirectScheduled.value = true
    doneRedirectTimerId.value = window.setTimeout(() => {
      void router.replace('/fueling/done')
    }, FUELING_DONE_REDIRECT_DELAY_MS)
  },
  { immediate: true },
)

onUnmounted(() => {
  store.stopFuelingPolling()
  if (doneRedirectTimerId.value !== null) {
    window.clearTimeout(doneRedirectTimerId.value)
    doneRedirectTimerId.value = null
  }
})
</script>

<template>
  <div class="min-h-screen flex flex-col bg-fuel-cream">
    <header class="bg-fuel-forest border-b border-fuel-olive/35 py-5 px-10 text-center shrink-0 shadow-sm">
      <p class="font-karla text-xs text-white/80 tracking-widest uppercase mb-1">
        Автоматизированная АЗС
      </p>
      <h1 class="font-rubik font-bold text-3xl text-white leading-tight">
        Процесс заправки
      </h1>
    </header>

    <StepIndicator
      :steps="STEPS"
      :current="4"
    />

    <main class="flex-1 px-8 py-10">
      <section
        v-if="screenMode === 'ready'"
        class="w-full max-w-5xl mx-auto bg-white rounded-2xl border border-fuel-olive/20 shadow-sm p-8"
      >
        <div class="flex items-start justify-between gap-6 mb-8">
          <div class="space-y-2">
            <p class="font-karla text-sm text-fuel-olive tracking-wide uppercase">
              Текущий этап
            </p>
            <h2 class="font-rubik text-3xl font-bold text-fuel-forest">
              {{ uiState.title }}
            </h2>
            <p class="font-karla text-fuel-olive text-base max-w-2xl">
              {{ uiState.description }}
            </p>
          </div>

          <span
            class="font-karla text-xs font-semibold tracking-widest uppercase px-4 py-2 rounded-full whitespace-nowrap"
            :class="uiState.badgeClass"
          >
            {{ uiState.badgeLabel }}
          </span>
        </div>

        <div class="grid grid-cols-3 gap-4 mb-8">
          <article class="rounded-xl bg-fuel-cream border border-fuel-olive/20 p-4">
            <p class="font-karla text-xs uppercase tracking-widest text-fuel-olive/80 mb-1">
              Отпущено, ₽
            </p>
            <p class="font-rubik text-xl font-semibold text-fuel-forest">
              {{ rubFormatter.format(uiState.dispensedAmount) }} ₽
            </p>
          </article>
          <article class="rounded-xl bg-fuel-cream border border-fuel-olive/20 p-4">
            <p class="font-karla text-xs uppercase tracking-widest text-fuel-olive/80 mb-1">
              Отпущено, л
            </p>
            <p class="font-rubik text-xl font-semibold text-fuel-forest">
              {{ uiState.dispensedLiters.toFixed(2) }} л
            </p>
          </article>
          <article class="rounded-xl bg-fuel-cream border border-fuel-olive/20 p-4">
            <p class="font-karla text-xs uppercase tracking-widest text-fuel-olive/80 mb-1">
              Цена за литр
            </p>
            <p class="font-rubik text-xl font-semibold text-fuel-forest">
              {{ rubFormatter.format(uiState.pricePerLiter) }} ₽/л
            </p>
          </article>
        </div>

        <div class="mb-8">
          <div class="flex items-center justify-between mb-2">
            <p class="font-karla text-sm text-fuel-olive">
              Прогресс заправки
            </p>
            <p class="font-rubik text-sm font-semibold text-fuel-forest">
              {{ uiState.progressPercent }}%
            </p>
          </div>
          <div class="h-4 rounded-full bg-fuel-olive/15 overflow-hidden">
            <div
              class="h-full bg-fuel-lime transition-all duration-500"
              :style="{ width: `${uiState.progressPercent}%` }"
            />
          </div>
        </div>

        <div
          v-if="errorMessage"
          class="mb-6 rounded-xl border border-red-200 bg-red-50 px-4 py-3"
        >
          <p class="font-karla text-sm text-red-700">
            {{ errorMessage }}
          </p>
        </div>

      </section>

      <section
        v-else-if="screenMode === 'no-transaction'"
        class="w-full max-w-3xl mx-auto bg-white rounded-2xl border border-fuel-olive/20 shadow-sm p-8 text-center"
      >
        <h2 class="font-rubik text-2xl font-bold text-fuel-forest mb-2">
          Нет активной транзакции
        </h2>
        <p class="font-karla text-fuel-olive mb-6">
          Для отображения прогресса начните новую сессию заправки.
        </p>
        <button
          type="button"
          class="font-rubik font-semibold text-lg px-10 py-3 rounded-xl bg-fuel-lime text-white hover:bg-fuel-forest transition-all duration-200"
          @click="goToFuelSelect"
        >
          К выбору топлива
        </button>
      </section>

      <section
        v-else
        class="w-full max-w-3xl mx-auto bg-white rounded-2xl border border-fuel-olive/20 shadow-sm p-8 text-center"
      >
        <h2 class="font-rubik text-2xl font-bold text-fuel-forest mb-2">
          Этап заправки еще не начался
        </h2>
        <p class="font-karla text-fuel-olive mb-6">
          Текущий статус транзакции не относится к топливному контуру.
        </p>
        <div class="flex items-center justify-center gap-3">
          <button
            type="button"
            class="font-rubik font-semibold text-lg px-8 py-3 rounded-xl bg-fuel-lime text-white hover:bg-fuel-forest transition-all duration-200"
            @click="goToOrderParams"
          >
            К параметрам
          </button>
        </div>
      </section>
    </main>
  </div>
</template>
