<script setup lang="ts">
import { onBeforeUnmount, onMounted } from 'vue'
import { RouterLink, RouterView } from 'vue-router'

import { useKioskStateStore } from '@/stores/kioskState'
import { useWatchdogStateStore } from '@/stores/watchdogState'

const kioskStateStore = useKioskStateStore()
const watchdogStateStore = useWatchdogStateStore()

// В админке также подгружаем состояние киоска (без оверлея) для индикатора в шапке
// Это безопасно потому что App.vue в /admin/* не поллит сам
onMounted(() => {
  kioskStateStore.startPolling()
  watchdogStateStore.startPolling()
})

onBeforeUnmount(() => {
  kioskStateStore.stopPolling()
  watchdogStateStore.stopPolling()
})
</script>

<template>
  <div class="min-h-screen flex flex-col bg-fuel-cream">
    <!-- Шапка админки в стиле основных экранов -->
    <header class="bg-fuel-forest border-b border-fuel-olive/35 py-5 px-10 shrink-0 shadow-sm">
      <div class="max-w-6xl mx-auto relative flex items-center justify-center">
        <div class="text-admin text-center">
          <p class="font-karla text-xs text-white/80 tracking-widest uppercase mb-1">
            Администратор АЗС
          </p>
          <h1 class="font-rubik font-bold text-3xl text-white leading-tight">
            Панель управления
          </h1>
        </div>

        <!-- Индикатор текущего режима киоска -->
        <div class="absolute right-0 flex items-center gap-3 font-karla text-sm">
          <span
            class="inline-flex h-3 w-3 rounded-full"
            :class="kioskStateStore.maintenance ? 'bg-amber-400 animate-pulse' : 'bg-fuel-lime'"
            aria-hidden="true"
          />
          <span class="text-white/90">
            {{ kioskStateStore.maintenance ? 'Тех. работы' : 'В работе' }}
          </span>
        </div>
      </div>
    </header>

    <!-- Навигация в стиле ссылок на fuel-forest -->
    <nav class="bg-fuel-forest/95 border-b border-fuel-olive/25 shrink-0">
      <div class="max-w-6xl mx-auto flex gap-2 px-10">
        <RouterLink
          v-for="link in [
            { to: '/admin', label: 'Режим работы', exact: true },
            { to: '/admin/prices', label: 'Цены', exact: false },
            { to: '/admin/transactions', label: 'Транзакции', exact: false },
            { to: '/admin/equipment', label: 'Оборудование', exact: false },
          ]"
          :key="link.to"
          :to="link.to"
          :exact-active-class="link.exact ? 'bg-fuel-lime text-white' : ''"
          :active-class="link.exact ? '' : 'bg-fuel-lime text-white'"
          class="font-rubik font-medium text-sm tracking-wide px-5 py-3
                 text-white/80 hover:text-white hover:bg-fuel-olive/40 transition-colors"
        >
          {{ link.label }}
        </RouterLink>
      </div>
    </nav>

    <!-- Основное пространство страницы -->
    <main class="flex-1 px-8 py-10">
      <div class="max-w-6xl mx-auto">
        <RouterView />
      </div>
    </main>
  </div>
</template>
