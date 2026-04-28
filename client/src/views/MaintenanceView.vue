<script setup lang="ts">
import { computed } from 'vue'

import { useKioskStateStore } from '@/stores/kioskState'

const kioskStateStore = useKioskStateStore()

const reason = computed(() => kioskStateStore.reason.trim())
</script>

<template>
  <div class="fixed inset-0 z-[9999] min-h-screen flex flex-col bg-fuel-cream">
    <!-- Шапка как у других экранов киоска -->
    <header class="bg-fuel-forest border-b border-fuel-olive/35 py-5 px-10 text-center shrink-0 shadow-sm">
      <p class="font-karla text-xs text-white/80 tracking-widest uppercase mb-1">
        Автоматизированная АЗС
      </p>
      <h1 class="font-rubik font-bold text-3xl text-white leading-tight">
        Ведутся технические работы
      </h1>
    </header>

    <!-- Контент на весь оставшийся экран, перехватываем любые тапы -->
    <main
      class="flex-1 flex flex-col items-center justify-center gap-6 px-8 py-10"
      role="alert"
      aria-live="assertive"
      @pointerdown.capture.stop.prevent
      @touchstart.capture.stop.prevent
      @click.capture.stop.prevent
    >
      <!-- Круглый индикатор в фирменном зеленом -->
      <div
        class="w-28 h-28 rounded-full bg-fuel-forest flex items-center justify-center shadow-lg shadow-fuel-olive/30"
        aria-hidden="true"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
          class="w-14 h-14 text-fuel-lime"
        >
          <path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z" />
        </svg>
      </div>

      <p class="font-rubik font-semibold text-3xl text-fuel-forest text-center max-w-2xl leading-tight">
        Заправка временно недоступна
      </p>
      <p class="font-karla text-lg text-fuel-olive text-center max-w-2xl">
        Приносим извинения за неудобства. Работы проводятся администратором.
      </p>

      <p
        v-if="reason"
        class="font-karla text-base text-fuel-forest/80 text-center max-w-2xl border-t border-fuel-olive/25 pt-4"
      >
        {{ reason }}
      </p>
    </main>

    <!-- Подвал в стиле кнопки "Далее" только как статус-бар -->
    <footer class="bg-fuel-forest/95 py-4 px-10 text-center shrink-0">
      <p class="font-karla text-sm text-white/70">
        Обратитесь к персоналу АЗС, если требуется срочная заправка.
      </p>
    </footer>
  </div>
</template>
