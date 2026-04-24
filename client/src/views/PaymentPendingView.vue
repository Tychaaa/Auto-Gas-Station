<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'

import StepIndicator from '@/components/StepIndicator.vue'
import { useTransactionFlowStore } from '@/stores'

const STEPS = ['Топливо', 'Параметры', 'Оплата', 'Заправка'] as const

const router = useRouter()
const store = useTransactionFlowStore()

const isNavigatingToResult = ref(false)

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

      </section>
    </main>
  </div>
</template>
