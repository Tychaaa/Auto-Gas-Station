<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'

import StepIndicator from '@/components/StepIndicator.vue'
import { useOrderSummary } from '@/composables/useOrderSummary'
import { useTransactionFlowStore } from '@/stores'

const STEPS = ['Топливо', 'Параметры', 'Оплата', 'Заправка'] as const
const SUCCESS_REDIRECT_DELAY_MS = 5000

const router = useRouter()
const store = useTransactionFlowStore()
const transaction = computed(() => store.transaction)
const { orderFuelType, orderLiters, orderUnitPrice, orderTotalAmount } = useOrderSummary()
const successRedirectTimerId = ref<number | null>(null)
const isRetrying = ref(false)
const isCancelling = ref(false)

const isPaid = computed(() => transaction.value?.status === 'paid')
const isFailed = computed(() => transaction.value?.status === 'failed')
const isActionBusy = computed(() => isRetrying.value || isCancelling.value || store.isSubmittingSelection || store.isStartingPayment)

const statusTitle = computed(() => {
  if (isPaid.value) {
    return 'Оплата прошла успешно'
  }
  if (isFailed.value) {
    return 'Оплата не прошла'
  }
  return 'Статус оплаты уточняется'
})

const statusDescription = computed(() => {
  if (isPaid.value) {
    return 'Платеж подтвержден. Начинаем заправку.'
  }
  if (isFailed.value) {
    return transaction.value?.paymentError || store.lastError?.message || 'Попробуйте оплатить еще раз.'
  }
  return 'Подождите несколько секунд, информация обновляется.'
})

const statusBadgeClass = computed(() => {
  if (isPaid.value) {
    return 'border-emerald-200 bg-emerald-50 text-emerald-800'
  }
  if (isFailed.value) {
    return 'border-red-200 bg-red-50 text-red-700'
  }
  return 'border-fuel-olive/30 bg-fuel-cream text-fuel-forest'
})

const formattedResultTime = computed(() => {
  const raw = transaction.value?.updatedAt || transaction.value?.createdAt
  if (!raw) {
    return '—'
  }

  const date = new Date(raw)
  if (Number.isNaN(date.getTime())) {
    return '—'
  }

  return new Intl.DateTimeFormat('ru-RU', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
})

function clearSuccessRedirectTimer(): void {
  if (successRedirectTimerId.value !== null) {
    window.clearTimeout(successRedirectTimerId.value)
    successRedirectTimerId.value = null
  }
}

function scheduleSuccessRedirect(): void {
  clearSuccessRedirectTimer()
  successRedirectTimerId.value = window.setTimeout(() => {
    void router.push('/fueling/progress')
  }, SUCCESS_REDIRECT_DELAY_MS)
}

async function handleRetryPayment(): Promise<void> {
  if (!isFailed.value || isActionBusy.value) {
    return
  }

  isRetrying.value = true
  clearSuccessRedirectTimer()
  store.resetForPaymentRetry()

  try {
    const nextTransaction = await store.submitSelection()
    if (!nextTransaction) {
      await router.push('/select/order')
      return
    }
    await router.push('/payment/method')
  } finally {
    isRetrying.value = false
  }
}

async function handleCancelToStart(): Promise<void> {
  if (isActionBusy.value) {
    return
  }

  isCancelling.value = true
  clearSuccessRedirectTimer()

  try {
    store.resetFlow()
    await router.push('/select/fuel')
  } finally {
    isCancelling.value = false
  }
}

watch(
  isPaid,
  (paid) => {
    if (paid) {
      scheduleSuccessRedirect()
      return
    }
    clearSuccessRedirectTimer()
  },
  { immediate: true },
)

onUnmounted(() => {
  clearSuccessRedirectTimer()
})
</script>

<template>
  <div class="min-h-screen flex flex-col bg-fuel-cream">
    <header class="bg-fuel-forest border-b border-fuel-olive/35 py-5 px-6 text-center shrink-0 shadow-sm sm:px-10">
      <p class="font-karla text-xs text-white/80 tracking-widest uppercase mb-1">
        Автоматизированная АЗС
      </p>
      <h1 class="font-rubik font-bold text-3xl text-white leading-tight">
        Результат оплаты
      </h1>
    </header>

    <StepIndicator
      :steps="STEPS"
      :current="3"
    />

    <main class="flex-1 px-4 py-8 sm:px-6 sm:py-10">
      <section class="mx-auto max-w-3xl rounded-2xl bg-white p-6 shadow-sm border border-fuel-olive/20 sm:p-8">
        <div
          class="rounded-xl border px-4 py-4"
          :class="statusBadgeClass"
        >
          <p class="font-rubik text-xl font-semibold">
            {{ statusTitle }}
          </p>
          <p class="mt-2 font-karla text-sm">
            {{ statusDescription }}
          </p>
        </div>

        <article
          v-if="isPaid"
          class="mt-5 rounded-xl border border-fuel-olive/20 bg-fuel-cream/40 p-4 sm:p-5"
        >
          <div class="flex items-center justify-between gap-3 mb-4">
            <h2 class="font-rubik text-xl font-semibold text-fuel-forest">
              Чек заказа
            </h2>
            <p class="font-karla text-xs text-fuel-olive uppercase tracking-wide">
              Оплата подтверждена
            </p>
          </div>

          <dl class="space-y-3">
            <div class="flex items-center justify-between gap-3">
              <dt class="font-karla text-sm text-fuel-olive">
                Вид топлива
              </dt>
              <dd class="font-rubik text-base font-medium text-fuel-forest">
                {{ orderFuelType }}
              </dd>
            </div>
            <div class="flex items-center justify-between gap-3">
              <dt class="font-karla text-sm text-fuel-olive">
                Объем
              </dt>
              <dd class="font-rubik text-base font-medium text-fuel-forest">
                {{ orderLiters }}
              </dd>
            </div>
            <div class="flex items-center justify-between gap-3">
              <dt class="font-karla text-sm text-fuel-olive">
                Цена за литр
              </dt>
              <dd class="font-rubik text-base font-medium text-fuel-forest">
                {{ orderUnitPrice }}
              </dd>
            </div>
            <div class="h-px bg-fuel-olive/20" />
            <div class="flex items-center justify-between gap-3">
              <dt class="font-karla text-sm text-fuel-olive">
                Итого
              </dt>
              <dd class="font-rubik text-lg font-semibold text-fuel-forest">
                {{ orderTotalAmount }}
              </dd>
            </div>
            <div class="h-px bg-fuel-olive/20" />
            <div class="flex items-center justify-between gap-3">
              <dt class="font-karla text-sm text-fuel-olive">
                № транзакции
              </dt>
              <dd class="font-rubik text-sm font-medium text-fuel-forest break-all text-right">
                {{ transaction?.id ?? '—' }}
              </dd>
            </div>
            <div class="flex items-center justify-between gap-3">
              <dt class="font-karla text-sm text-fuel-olive">
                Время операции
              </dt>
              <dd class="font-rubik text-sm font-medium text-fuel-forest">
                {{ formattedResultTime }}
              </dd>
            </div>
          </dl>
        </article>

        <article
          v-else-if="isFailed"
          class="mt-5 rounded-xl border border-amber-200 bg-amber-50 p-4 sm:p-5"
        >
          <h2 class="font-rubik text-lg font-semibold text-amber-800">
            Рекомендации
          </h2>
          <ul class="mt-3 space-y-2 font-karla text-sm text-amber-800">
            <li>Проверьте карту и попробуйте оплатить еще раз.</li>
            <li>Если проблема повторяется, обратитесь к оператору АЗС.</li>
          </ul>
        </article>

        <div class="mt-6 flex flex-wrap items-center gap-3">
          <button
            v-if="isFailed"
            type="button"
            :disabled="isActionBusy"
            :aria-disabled="isActionBusy"
            class="font-rubik font-semibold text-base px-6 py-3 rounded-xl transition-all duration-200
                   focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-fuel-lime focus-visible:ring-offset-2 focus-visible:ring-offset-white"
            :class="
              isActionBusy
                ? 'bg-fuel-lime/35 text-fuel-olive/60 cursor-not-allowed'
                : 'bg-fuel-lime text-white hover:bg-fuel-forest active:scale-[0.98] shadow-md shadow-fuel-lime/25'
            "
            @click="handleRetryPayment"
          >
            {{ isRetrying ? 'Подготовка новой оплаты...' : 'Попробовать снова' }}
          </button>

          <button
            v-if="!isPaid"
            type="button"
            :disabled="isActionBusy"
            :aria-disabled="isActionBusy"
            class="font-rubik font-semibold text-base px-6 py-3 rounded-xl border transition-all duration-200
                   focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-fuel-lime focus-visible:ring-offset-2 focus-visible:ring-offset-white"
            :class="
              isActionBusy
                ? 'border-fuel-olive/20 bg-fuel-olive/10 text-fuel-olive/50 cursor-not-allowed'
                : 'border-fuel-olive/30 bg-white text-fuel-olive hover:border-fuel-olive/50 hover:bg-fuel-cream/60 active:scale-[0.98]'
            "
            @click="handleCancelToStart"
          >
            Отменить и на главный экран
          </button>
        </div>
      </section>
    </main>
  </div>
</template>
