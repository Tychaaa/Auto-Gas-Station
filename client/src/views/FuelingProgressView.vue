<script setup lang="ts">
import { computed, ref } from 'vue'

import StepIndicator from '@/components/StepIndicator.vue'

const STEPS = ['Топливо', 'Параметры', 'Оплата', 'Заправка'] as const

type FuelingUiStateKey = 'starting' | 'dispensing' | 'completed_waiting_fiscal' | 'failed'

interface FuelingUiState {
  title: string
  description: string
  providerStatus: string
  dispensedLiters: number
  targetLiters: number
  badgeClass: string
}

const STATE_SEQUENCE: FuelingUiStateKey[] = ['starting', 'dispensing', 'completed_waiting_fiscal', 'failed']

const STATE_LABELS: Record<FuelingUiStateKey, string> = {
  starting: 'Подготовка колонки',
  dispensing: 'Отпуск топлива',
  completed_waiting_fiscal: 'Ожидание чека',
  failed: 'Ошибка процесса',
}

const UI_STATES: Record<FuelingUiStateKey, FuelingUiState> = {
  starting: {
    title: 'Подготовка к заправке',
    description: 'Колонка принимает команду. Пожалуйста, зафиксируйте пистолет в баке.',
    providerStatus: 'starting',
    dispensedLiters: 0,
    targetLiters: 30,
    badgeClass: 'bg-fuel-olive/15 text-fuel-forest border border-fuel-olive/30',
  },
  dispensing: {
    title: 'Идет отпуск топлива',
    description: 'Топливо подается. Следите за индикатором, процесс обновляется в реальном времени.',
    providerStatus: 'dispensing',
    dispensedLiters: 18.6,
    targetLiters: 30,
    badgeClass: 'bg-fuel-lime/20 text-fuel-forest border border-fuel-lime/40',
  },
  completed_waiting_fiscal: {
    title: 'Заправка завершена',
    description: 'Отпуск топлива завершен. Ожидаем подтверждение и формирование чека.',
    providerStatus: 'completed_waiting_fiscal',
    dispensedLiters: 30,
    targetLiters: 30,
    badgeClass: 'bg-fuel-lime text-white border border-fuel-lime',
  },
  failed: {
    title: 'Не удалось завершить заправку',
    description: 'Произошла ошибка топливного контура. Обратитесь к оператору для продолжения.',
    providerStatus: 'failed',
    dispensedLiters: 7.4,
    targetLiters: 30,
    badgeClass: 'bg-red-100 text-red-700 border border-red-200',
  },
}

const activeStateKey = ref<FuelingUiStateKey>('dispensing')

const activeState = computed(() => UI_STATES[activeStateKey.value])

const progressPercent = computed(() => {
  const { dispensedLiters, targetLiters } = activeState.value
  if (targetLiters <= 0) return 0
  const normalized = Math.round((dispensedLiters / targetLiters) * 100)
  return Math.min(100, Math.max(0, normalized))
})

function selectState(nextState: FuelingUiStateKey): void {
  activeStateKey.value = nextState
}
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
      <section class="w-full max-w-5xl mx-auto bg-white rounded-2xl border border-fuel-olive/20 shadow-sm p-8">
        <div class="flex items-start justify-between gap-6 mb-8">
          <div class="space-y-2">
            <p class="font-karla text-sm text-fuel-olive tracking-wide uppercase">
              Текущий этап
            </p>
            <h2 class="font-rubik text-3xl font-bold text-fuel-forest">
              {{ activeState.title }}
            </h2>
            <p class="font-karla text-fuel-olive text-base max-w-2xl">
              {{ activeState.description }}
            </p>
          </div>

          <span
            class="font-karla text-xs font-semibold tracking-widest uppercase px-4 py-2 rounded-full whitespace-nowrap"
            :class="activeState.badgeClass"
          >
            {{ STATE_LABELS[activeStateKey] }}
          </span>
        </div>

        <div class="grid grid-cols-3 gap-4 mb-8">
          <article class="rounded-xl bg-fuel-cream border border-fuel-olive/20 p-4">
            <p class="font-karla text-xs uppercase tracking-widest text-fuel-olive/80 mb-1">
              Статус колонки
            </p>
            <p class="font-rubik text-xl font-semibold text-fuel-forest">
              {{ activeState.providerStatus }}
            </p>
          </article>
          <article class="rounded-xl bg-fuel-cream border border-fuel-olive/20 p-4">
            <p class="font-karla text-xs uppercase tracking-widest text-fuel-olive/80 mb-1">
              Отпущено
            </p>
            <p class="font-rubik text-xl font-semibold text-fuel-forest">
              {{ activeState.dispensedLiters.toFixed(1) }} л
            </p>
          </article>
          <article class="rounded-xl bg-fuel-cream border border-fuel-olive/20 p-4">
            <p class="font-karla text-xs uppercase tracking-widest text-fuel-olive/80 mb-1">
              План
            </p>
            <p class="font-rubik text-xl font-semibold text-fuel-forest">
              {{ activeState.targetLiters.toFixed(1) }} л
            </p>
          </article>
        </div>

        <div class="mb-8">
          <div class="flex items-center justify-between mb-2">
            <p class="font-karla text-sm text-fuel-olive">
              Прогресс заправки
            </p>
            <p class="font-rubik text-sm font-semibold text-fuel-forest">
              {{ progressPercent }}%
            </p>
          </div>
          <div class="h-4 rounded-full bg-fuel-olive/15 overflow-hidden">
            <div
              class="h-full bg-fuel-lime transition-all duration-500"
              :style="{ width: `${progressPercent}%` }"
            />
          </div>
        </div>

        <div class="flex flex-wrap items-center gap-3 mb-8">
          <p class="font-karla text-sm text-fuel-olive mr-1">
            Демо-состояние:
          </p>
          <button
            v-for="state in STATE_SEQUENCE"
            :key="state"
            type="button"
            class="font-karla text-xs font-semibold uppercase tracking-widest px-4 py-2 rounded-full border transition-all duration-200
                   focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-fuel-lime focus-visible:ring-offset-2 focus-visible:ring-offset-white"
            :class="
              activeStateKey === state
                ? 'bg-fuel-olive text-white border-fuel-olive'
                : 'bg-white text-fuel-olive border-fuel-olive/30 hover:border-fuel-lime hover:text-fuel-forest'
            "
            @click="selectState(state)"
          >
            {{ STATE_LABELS[state] }}
          </button>
        </div>

        <div class="flex items-center gap-4">
          <button
            type="button"
            disabled
            aria-disabled="true"
            class="font-rubik font-semibold text-lg px-10 py-3 rounded-xl bg-fuel-lime/35 text-fuel-olive/60 cursor-not-allowed"
          >
            Обновить
          </button>
          <button
            type="button"
            disabled
            aria-disabled="true"
            class="font-rubik font-semibold text-lg px-10 py-3 rounded-xl bg-fuel-olive/25 text-fuel-olive/60 cursor-not-allowed"
          >
            Завершить
          </button>
        </div>
      </section>
    </main>
  </div>
</template>
