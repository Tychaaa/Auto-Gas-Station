<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'

import StepIndicator from '@/components/StepIndicator.vue'
import { useTransactionFlowStore } from '@/stores'

const STEPS = ['Топливо', 'Параметры', 'Оплата', 'Заправка'] as const

const router = useRouter()
const store = useTransactionFlowStore()

const isNavigatingToResult = ref(false)
const isCancelling = ref(false)
const showCancelConfirm = ref(false)

const transaction = computed(() => store.transaction)
const hasPendingStatus = computed(() => transaction.value?.status === 'payment_pending')
const errorMessage = computed(() => store.lastError?.message || transaction.value?.paymentError || '')

onMounted(() => {
  if (hasPendingStatus.value && !store.isPollingPayment) {
    store.startPaymentPolling()
  }
})

watch(
  () => transaction.value?.status,
  async (status) => {
    if (isNavigatingToResult.value) {
      return
    }

    if (status === 'paid' || status === 'failed') {
      isNavigatingToResult.value = true
      store.stopPaymentPolling()
      await router.push('/payment/result')
    }
  },
)

async function confirmCancel(): Promise<void> {
  showCancelConfirm.value = false
  isCancelling.value = true
  try {
    await store.cancelPaymentFlow()
    isNavigatingToResult.value = true
    await router.push('/payment/result')
  } finally {
    isCancelling.value = false
  }
}

onUnmounted(() => {
  store.stopPaymentPolling()
})
</script>

<template>
  <div class="min-h-screen flex flex-col bg-fuel-cream">
    <header class="bg-fuel-forest border-b border-fuel-olive/35 py-5 px-6 text-center shrink-0 shadow-sm sm:px-10">
      <p class="font-karla text-xs text-white/80 tracking-widest uppercase mb-1">
        Автоматизированная АЗС
      </p>
      <h1 class="font-rubik font-bold text-3xl text-white leading-tight">
        Ожидание оплаты
      </h1>
    </header>

    <StepIndicator
      :steps="STEPS"
      :current="3"
    />

    <main class="flex-1 w-full px-4 py-6 sm:px-6 sm:py-8">
      <section class="mx-auto w-full max-w-5xl flex flex-col gap-5">
        <article class="rounded-2xl border border-fuel-olive/20 bg-white p-5 shadow-sm sm:p-6">
          <div class="flex items-center justify-between gap-4">
            <div>
              <h2 class="font-rubik font-semibold text-2xl text-fuel-forest">
                Подтвердите оплату на терминале
              </h2>
              <p class="mt-2 font-karla text-fuel-olive">
                Ожидаем ответ эквайринга. Обычно это занимает несколько секунд.
              </p>
            </div>

            <div class="flex items-center gap-3 rounded-xl bg-fuel-cream px-4 py-3 border border-fuel-olive/20">
              <span class="relative flex h-3 w-3">
                <span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-fuel-lime opacity-75" />
                <span class="relative inline-flex h-3 w-3 rounded-full bg-fuel-lime" />
              </span>
              <p class="font-karla text-sm text-fuel-forest">
                {{ hasPendingStatus ? 'Ожидание подтверждения' : 'Статус обновлен' }}
              </p>
            </div>
          </div>
        </article>

        <div
          v-if="errorMessage"
          class="rounded-xl border border-red-200 bg-red-50 px-4 py-3"
        >
          <p class="font-karla text-sm text-red-700">
            {{ errorMessage }}
          </p>
        </div>

        <!-- Кнопка отмены -->
        <div v-if="!showCancelConfirm" class="flex justify-center">
          <button
            :disabled="isCancelling || !hasPendingStatus"
            class="font-karla text-sm text-fuel-olive/70 underline underline-offset-2 hover:text-fuel-forest disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
            @click="showCancelConfirm = true"
          >
            Отменить оплату
          </button>
        </div>

        <!-- Подтверждение отмены -->
        <div
          v-else
          class="rounded-xl border border-amber-200 bg-amber-50 px-5 py-4 flex flex-col gap-3"
        >
          <p class="font-karla text-sm text-amber-800 font-medium">
            Отменить оплату? Терминал будет освобождён.
          </p>
          <div class="flex gap-3">
            <button
              :disabled="isCancelling"
              class="flex-1 rounded-lg bg-red-600 px-4 py-2 font-karla text-sm text-white hover:bg-red-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              @click="confirmCancel"
            >
              {{ isCancelling ? 'Отменяем...' : 'Да, отменить' }}
            </button>
            <button
              :disabled="isCancelling"
              class="flex-1 rounded-lg border border-fuel-olive/30 px-4 py-2 font-karla text-sm text-fuel-forest hover:bg-fuel-cream/70 disabled:opacity-50 transition-colors"
              @click="showCancelConfirm = false"
            >
              Назад
            </button>
          </div>
        </div>

      </section>
    </main>
  </div>
</template>
