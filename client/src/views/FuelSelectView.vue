<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'

import { useTransactionFlowStore } from '@/stores'
import StepIndicator from '@/components/StepIndicator.vue'
import FuelCard from '@/components/FuelCard.vue'

const router = useRouter()
const store = useTransactionFlowStore()

// Шаги индикатора в верхней части экрана
const STEPS = ['Топливо', 'Параметры', 'Оплата', 'Заправка'] as const

// Доступные варианты топлива
const FUEL_TYPES = [
  { id: 'АИ-92', name: 'АИ-92', grade: 'Регулярный' },
  { id: 'АИ-95', name: 'АИ-95', grade: 'Улучшенный' },
  { id: 'АИ-98', name: 'АИ-98', grade: 'Премиум' },
  { id: 'ДТ', name: 'ДТ', grade: 'Дизель' },
] as const

const selectedFuel = computed(() => store.selectionDraft.fuelType)

// Сохраняет выбранный тип топлива в store
function selectFuel(fuelId: string): void {
  store.setSelectionDraft({ fuelType: fuelId })
}

// Переход к шагу выбора параметров
function handleNext(): void {
  if (!selectedFuel.value) return
  void router.push('/select/order')
}
</script>

<template>
  <div class="min-h-screen flex flex-col bg-zinc-100">
    <!-- Шапка экрана -->
    <header class="bg-white border-b border-zinc-200 py-5 px-10 text-center shrink-0">
      <p class="font-karla text-xs text-zinc-400 tracking-widest uppercase mb-1">
        Автоматизированная АЗС
      </p>
      <h1 class="font-rubik font-bold text-3xl text-zinc-900 leading-tight">
        Выберите вид топлива
      </h1>
    </header>

    <!-- Индикатор текущего шага -->
    <StepIndicator :steps="STEPS" :current="1" />

    <!-- Основная область с выбором топлива -->
    <main class="flex-1 flex flex-col items-center justify-center gap-8 px-8 py-10">
      <!-- Сетка карточек топлива -->
      <div
        class="grid grid-cols-4 gap-5 w-full max-w-4xl"
        role="group"
        aria-label="Виды топлива"
      >
        <FuelCard
          v-for="fuel in FUEL_TYPES"
          :key="fuel.id"
          :name="fuel.name"
          :grade="fuel.grade"
          :selected="selectedFuel === fuel.id"
          @select="selectFuel(fuel.id)"
        />
      </div>

      <!-- Подсказка для пользователя -->
      <p class="font-karla text-sm text-zinc-400 transition-all duration-300">
        {{
          selectedFuel
            ? `Выбрано: ${selectedFuel} — нажмите «Далее» для продолжения`
            : 'Нажмите на карточку, чтобы выбрать вид топлива'
        }}
      </p>

      <!-- Кнопка перехода к следующему шагу -->
      <button
        type="button"
        :disabled="!selectedFuel"
        :aria-disabled="!selectedFuel"
        class="font-rubik font-semibold text-lg px-14 py-4 rounded-xl
               transition-all duration-200
               focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-zinc-900 focus-visible:ring-offset-2"
        :class="
          selectedFuel
            ? 'bg-zinc-900 text-white hover:bg-zinc-700 active:scale-95 shadow-sm cursor-pointer'
            : 'bg-zinc-200 text-zinc-400 cursor-not-allowed'
        "
        @click="handleNext"
      >
        Далее →
      </button>
    </main>
  </div>
</template>
