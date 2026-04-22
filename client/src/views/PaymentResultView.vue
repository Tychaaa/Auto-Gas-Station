<script setup lang="ts">
import { computed } from 'vue'

import { useTransactionFlowStore } from '@/stores'

const store = useTransactionFlowStore()
const transaction = computed(() => store.transaction)
const isRepriced = computed(() => transaction.value?.priceWasRepriced === true)
</script>

<template>
  <main class="min-h-screen bg-fuel-cream px-6 py-10">
    <section class="mx-auto max-w-2xl rounded-2xl bg-white p-8 shadow-sm border border-fuel-olive/20">
      <h1 class="font-rubik font-bold text-2xl text-fuel-forest mb-4">
        Результат оплаты
      </h1>
      <p class="font-karla text-fuel-olive">
        Текущий статус: <strong>{{ transaction?.status ?? 'unknown' }}</strong>
      </p>
      <p
        v-if="isRepriced"
        class="mt-4 rounded-lg border border-amber-300 bg-amber-50 px-4 py-3 font-karla text-amber-900"
      >
        Цена была обновлена перед оплатой, потому что время фиксации истекло. Сумма списана по актуальной серверной цене.
      </p>
      <p
        v-else
        class="mt-4 font-karla text-fuel-olive"
      >
        Цена соответствовала ранее подтвержденному snapshot.
      </p>
    </section>
  </main>
</template>
