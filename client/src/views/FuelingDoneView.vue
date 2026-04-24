<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'

import StepIndicator from '@/components/StepIndicator.vue'
import { useTransactionFlowStore } from '@/stores'

const AUTO_RETURN_DELAY_MS = 15000
const STEPS = ['Топливо', 'Параметры', 'Оплата', 'Заправка'] as const

const router = useRouter()
const store = useTransactionFlowStore()
const autoReturnTimerId = ref<number | null>(null)

function goToStart(): void {
  store.resetFlow()
  void router.replace('/select/fuel')
}

onMounted(() => {
  autoReturnTimerId.value = window.setTimeout(() => {
    goToStart()
  }, AUTO_RETURN_DELAY_MS)
})

onUnmounted(() => {
  if (autoReturnTimerId.value !== null) {
    window.clearTimeout(autoReturnTimerId.value)
    autoReturnTimerId.value = null
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

    <main class="flex-1 px-8 py-10 flex items-center justify-center">
      <section class="w-full max-w-2xl rounded-2xl bg-white p-8 shadow-sm border border-fuel-olive/20 text-center">
        <h2 class="font-rubik font-bold text-3xl text-fuel-forest mb-4">
          Заправка завершена
        </h2>
        <p class="font-karla text-fuel-olive mb-8">
          Спасибо, что выбрали нашу АЗС. Будем рады видеть вас снова. Приезжайте еще!
        </p>
        <button
          type="button"
          class="font-rubik font-semibold text-lg px-8 py-3 rounded-xl bg-fuel-lime text-white hover:bg-fuel-forest transition-all duration-200"
          @click="goToStart"
        >
          Новая заправка
        </button>
      </section>
    </main>
  </div>
</template>
