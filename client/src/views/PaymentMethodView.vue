<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'

import StepIndicator from '@/components/StepIndicator.vue'
import { useOrderSummary } from '@/composables/useOrderSummary'
import { useTransactionFlowStore } from '@/stores'

const router = useRouter()
const store = useTransactionFlowStore()
const { orderFuelType, orderLiters, orderUnitPrice, orderTotalAmount } = useOrderSummary()

const STEPS = ['Топливо', 'Параметры', 'Оплата', 'Заправка'] as const
const isPreparingOrderSummary = ref(false)
const selectedPaymentMethod = ref<'cashless' | null>('cashless')
const canUseCashless = computed(
  () =>
    store.isSelectionDraftValid &&
    !isPreparingOrderSummary.value &&
    !store.isSubmittingSelection &&
    !store.isStartingPayment,
)
const errorMessage = computed(() => store.lastError?.message || '')
const isCashlessProcessing = computed(() => store.isStartingPayment)
const canProceedWithPayment = computed(
  () => selectedPaymentMethod.value === 'cashless' && canUseCashless.value,
)

async function prepareOrderSummary(): Promise<void> {
  if (!store.isSelectionDraftValid || store.isSubmittingSelection || store.isStartingPayment) {
    return
  }

  isPreparingOrderSummary.value = true
  try {
    await store.submitSelection()
  } finally {
    isPreparingOrderSummary.value = false
  }
}

async function handleCashless(): Promise<void> {
  if (!canProceedWithPayment.value) return

  const selectionTransaction = await store.submitSelection()
  if (!selectionTransaction) return

  const paymentTransaction = await store.startPaymentFlow()
  if (!paymentTransaction) return

  if (paymentTransaction.status === 'payment_pending') {
    await router.push('/payment/pending')
    return
  }

  if (paymentTransaction.status === 'paid' || paymentTransaction.status === 'failed') {
    await router.push('/payment/result')
  }
}

async function goBack(): Promise<void> {
  await router.push('/select/order')
}

function selectCashlessMethod(): void {
  if (!canUseCashless.value) {
    return
  }
  selectedPaymentMethod.value = 'cashless'
}

onMounted(() => {
  void prepareOrderSummary()
})
</script>

<template>
  <div class="min-h-screen flex flex-col bg-fuel-cream">
    <header class="bg-fuel-forest border-b border-fuel-olive/35 py-5 px-6 text-center shrink-0 shadow-sm sm:px-10">
      <p class="font-karla text-xs text-white/80 tracking-widest uppercase mb-1">
        Автоматизированная АЗС
      </p>
      <h1 class="font-rubik font-bold text-3xl text-white leading-tight">
        Выбор способа оплаты
      </h1>
    </header>

    <StepIndicator
      :steps="STEPS"
      :current="3"
    />

    <main class="flex-1 w-full px-4 py-6 sm:px-6 sm:py-8">
      <section class="mx-auto w-full max-w-5xl flex flex-col gap-5">
        <div class="grid grid-cols-1 gap-5 lg:grid-cols-2">
          <article class="rounded-2xl border border-fuel-olive/20 bg-white p-5 shadow-sm sm:p-6">
            <h2 class="font-rubik font-semibold text-2xl text-fuel-forest mb-4">
              Детали заказа
            </h2>
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
                  Итоговая сумма
                </dt>
                <dd class="font-rubik text-lg font-semibold text-fuel-forest">
                  {{ orderTotalAmount }}
                </dd>
              </div>
            </dl>
          </article>

          <div class="flex flex-col gap-4">
          <button
            type="button"
            :disabled="!canUseCashless"
            :aria-disabled="!canUseCashless"
            class="rounded-2xl border-2 px-5 py-6 text-left transition-all duration-200
                   focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-fuel-lime"
            :class="
              canUseCashless && selectedPaymentMethod === 'cashless'
                ? 'border-fuel-lime bg-white shadow-sm'
                : canUseCashless
                  ? 'border-fuel-olive/30 bg-white hover:border-fuel-forest hover:bg-fuel-cream/40 active:scale-[0.98] shadow-sm'
                : 'border-fuel-lime/30 bg-fuel-cream/60 text-fuel-olive/70 cursor-not-allowed'
            "
            @click="selectCashlessMethod"
          >
            <p class="font-rubik font-semibold text-xl text-fuel-forest">
              Безналичный расчет
            </p>
            <p class="mt-2 font-karla text-sm text-fuel-olive">
              Оплата банковской картой или СБП
            </p>
          </button>

          <button
            type="button"
            disabled
            aria-disabled="true"
            class="rounded-2xl border-2 border-fuel-olive/20 bg-fuel-olive/10 px-5 py-6 text-left
                   text-fuel-olive/60 cursor-not-allowed"
          >
            <p class="font-rubik font-semibold text-xl">
              Наличные
            </p>
            <p class="mt-2 font-karla text-sm">
              Оплата наличными через кассовый терминал
            </p>
          </button>

          <button
            type="button"
            disabled
            aria-disabled="true"
            class="rounded-2xl border-2 border-fuel-olive/20 bg-fuel-olive/10 px-5 py-6 text-left
                   text-fuel-olive/60 cursor-not-allowed"
          >
            <p class="font-rubik font-semibold text-xl">
              Предоплаченная карта
            </p>
            <p class="mt-2 font-karla text-sm">
              Списание с баланса предоплаченной топливной карты
            </p>
          </button>
          </div>
        </div>

        <div
          v-if="errorMessage"
          class="rounded-xl border border-red-200 bg-red-50 px-4 py-3"
        >
          <p class="font-karla text-sm text-red-700">
            {{ errorMessage }}
          </p>
        </div>

        <div class="mt-1 flex items-center gap-3">
          <button
            type="button"
            class="font-rubik font-medium text-base px-5 py-3 rounded-xl border border-fuel-olive/30
                   text-fuel-olive bg-white transition-all duration-200 hover:border-fuel-olive/50 hover:bg-fuel-cream/60 active:scale-[0.98]
                   focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-fuel-lime"
            @click="goBack"
          >
            Назад
          </button>

          <button
            type="button"
            :disabled="!canProceedWithPayment"
            :aria-disabled="!canProceedWithPayment"
            class="ml-auto font-rubik font-semibold text-lg px-8 py-3 rounded-xl
                   transition-all duration-200
                   focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-fuel-lime focus-visible:ring-offset-2 focus-visible:ring-offset-white"
            :class="
              canProceedWithPayment
                ? 'bg-fuel-lime text-white hover:bg-fuel-forest active:scale-[0.98] shadow-md shadow-fuel-lime/25'
                : 'bg-fuel-lime/35 text-fuel-olive/60 cursor-not-allowed'
            "
            @click="handleCashless"
          >
            {{
              isCashlessProcessing
                ? 'Запускаем оплату...'
                : 'Оплатить безналичным расчетом →'
            }}
          </button>
        </div>
      </section>
    </main>
  </div>
</template>
