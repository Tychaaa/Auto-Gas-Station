<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'

import StepIndicator from '@/components/StepIndicator.vue'
import { useTransactionFlowStore } from '@/stores'

const STEPS = ['Топливо', 'Параметры', 'Оплата', 'Заправка'] as const

type ScreenMode = 'ready' | 'no-transaction' | 'wrong-stage'

interface FuelingUiState {
  title: string
  description: string
  providerStatus: string
  dispensedLiters: number
  targetLiters: number
  progressPercent: number
  badgeLabel: string
  badgeClass: string
}

const router = useRouter()
const store = useTransactionFlowStore()
const isPreparingDemoTransaction = ref(false)
const doneRedirectTimerId = ref<number | null>(null)
const doneRedirectScheduled = ref(false)
const FUELING_DONE_REDIRECT_DELAY_MS = 5000

const DEV_SELECTION_DRAFT = {
  fuelType: 'АИ-95',
  orderMode: 'liters',
  amountRub: 0,
  liters: 5,
  preset: '',
} as const

const DEV_FUELING_CONFIG = {
  pumpId: '1',
  nozzleId: '1',
  scenario: '',
} as const

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
  
  // if (status !== 'fueling' && status !== 'fiscalizing' && status !== 'completed' && status !== 'failed') {
  // TODO(release): убрать временный допуск paid в ready-state после завершения полного fueling flow.
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
      targetLiters: 0,
      progressPercent: 0,
      badgeLabel: 'Нет данных',
      badgeClass: 'bg-fuel-olive/15 text-fuel-forest border border-fuel-olive/30',
    }
  }

  const targetLiters = tx.liters > 0 ? tx.liters : Math.max(store.orderSummary.liters ?? 0, 0)
  const fuelingStatus = tx.fuelingStatus
  const providerStatusLabel = formatProviderStatus(fuelingStatus)
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
      targetLiters,
      progressPercent: tx.dispensedLiters > 0 ? progressByVolume : 10,
      badgeLabel: 'Подготовка колонки',
      badgeClass: 'bg-fuel-olive/15 text-fuel-forest border border-fuel-olive/30',
    }
  }

  return {
    title: 'Идет отпуск топлива',
    description: 'Топливо подается. Следите за индикатором налива.',
    providerStatus: providerStatusLabel,
    dispensedLiters: tx.dispensedLiters,
    targetLiters,
    progressPercent: progressByVolume,
    badgeLabel: 'Отпуск топлива',
    badgeClass: 'bg-fuel-lime/20 text-fuel-forest border border-fuel-lime/40',
  }
})

const canStartFuelingManually = computed(() => store.canStartFueling)
const canCreateTransactionManually = computed(() => !isPreparingDemoTransaction.value)
const shouldRedirectToDone = computed(() => {
  const tx = transaction.value
  if (!tx) {
    return false
  }
  return tx.status === 'completed' || tx.status === 'fiscalizing' || tx.fuelingStatus === 'completed_waiting_fiscal'
})

// TODO(release): удалить временные служебные кнопки после завершения сквозного UI flow.
async function handleManualTransactionCreate(): Promise<void> {
  if (!canCreateTransactionManually.value) {
    return
  }

  isPreparingDemoTransaction.value = true

  try {
    store.resetFlow()
    store.setSelectionDraft(DEV_SELECTION_DRAFT)
    store.setFuelingConfig(DEV_FUELING_CONFIG)

    const createdTransaction = await store.submitSelection()
    if (!createdTransaction) {
      return
    }

    let paymentTransaction = await store.startPaymentFlow()
    if (!paymentTransaction) {
      return
    }

    // Временный dev-helper: дожидаемся автофинализации mock-платежа,
    // чтобы экран можно было тестировать без ручного заполнения Pinia.
    for (let attempt = 0; attempt < 10 && paymentTransaction.status === 'payment_pending'; attempt += 1) {
      await new Promise((resolve) => window.setTimeout(resolve, 500))
      paymentTransaction = await store.pollPaymentStatusOnce()
      if (!paymentTransaction) {
        return
      }
    }
  } finally {
    isPreparingDemoTransaction.value = false
  }
}

async function handleManualFuelingStart(): Promise<void> {
  if (!canStartFuelingManually.value) {
    return
  }

  await store.startFuelingFlow()
}

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
    <!-- TODO(release): удалить временные служебные кнопки для ручного dev-тестирования. -->
    <div class="fixed top-4 left-4 z-20 flex items-center gap-2">
      <button
        type="button"
        :disabled="!canCreateTransactionManually"
        :aria-disabled="!canCreateTransactionManually"
        class="font-rubik font-semibold text-sm px-4 py-2 rounded-lg border transition-all duration-200
              focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-fuel-lime focus-visible:ring-offset-2 focus-visible:ring-offset-fuel-cream"
        :class="
          canCreateTransactionManually
            ? 'bg-white text-fuel-forest border-fuel-olive/30 hover:bg-fuel-cream hover:border-fuel-lime shadow-sm'
            : 'bg-white/80 text-fuel-olive/60 border-fuel-olive/20 cursor-not-allowed'
        "
        @click="handleManualTransactionCreate"
      >
        {{ isPreparingDemoTransaction ? 'Служебно: готовим...' : 'Служебно: создать транзакцию' }}
      </button>

      <button
        type="button"
        :disabled="!canStartFuelingManually"
        :aria-disabled="!canStartFuelingManually"
        class="font-rubik font-semibold text-sm px-4 py-2 rounded-lg border transition-all duration-200
              focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-fuel-lime focus-visible:ring-offset-2 focus-visible:ring-offset-fuel-cream"
        :class="
          canStartFuelingManually
            ? 'bg-fuel-lime text-white border-fuel-lime hover:bg-fuel-forest hover:border-fuel-forest shadow-md shadow-fuel-lime/20'
            : 'bg-white/80 text-fuel-olive/60 border-fuel-olive/20 cursor-not-allowed'
        "
        @click="handleManualFuelingStart"
      >
        Служебно: начать налив
      </button>
    </div>

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
              Статус колонки
            </p>
            <p class="font-rubik text-xl font-semibold text-fuel-forest">
              {{ uiState.providerStatus }}
            </p>
          </article>
          <article class="rounded-xl bg-fuel-cream border border-fuel-olive/20 p-4">
            <p class="font-karla text-xs uppercase tracking-widest text-fuel-olive/80 mb-1">
              Отпущено
            </p>
            <p class="font-rubik text-xl font-semibold text-fuel-forest">
              {{ uiState.dispensedLiters.toFixed(2) }} л
            </p>
          </article>
          <article class="rounded-xl bg-fuel-cream border border-fuel-olive/20 p-4">
            <p class="font-karla text-xs uppercase tracking-widest text-fuel-olive/80 mb-1">
              План
            </p>
            <p class="font-rubik text-xl font-semibold text-fuel-forest">
              {{ uiState.targetLiters.toFixed(2) }} л
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
