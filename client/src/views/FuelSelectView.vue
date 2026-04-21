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
  { id: 'АИ-92', name: 'АИ-92', grade: 'Регулярный', pricePerLiter: 61.53 },
  { id: 'АИ-95', name: 'АИ-95', grade: 'Улучшенный', pricePerLiter: 65.14 },
  { id: 'АИ-100', name: 'АИ-100', grade: 'Премиум', pricePerLiter: 87.80 },
  { id: 'ДТ', name: 'ДТ', grade: 'Дизель', pricePerLiter: 78.61 },
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
  <div class="min-h-screen flex flex-col bg-fuel-cream">
    <!-- Шапка экрана -->
    <header class="bg-fuel-forest border-b border-fuel-olive/35 py-5 px-10 text-center shrink-0 shadow-sm">
      <p class="font-karla text-xs text-white/80 tracking-widest uppercase mb-1">
        Автоматизированная АЗС
      </p>
      <h1 class="font-rubik font-bold text-3xl text-white leading-tight">
        Выберите вид топлива
      </h1>
    </header>

    <!-- Индикатор текущего шага -->
    <StepIndicator :steps="STEPS" :current="1" />

    <!-- Основная область с выбором топлива -->
    <main class="flex-1 flex flex-col items-center justify-center gap-8 px-8 py-10">
      <!-- Сетка карточек топлива -->
      <div
        class="grid grid-cols-4 gap-5 w-full max-w-5xl"
        role="group"
        aria-label="Виды топлива"
      >
        <div
          v-for="(fuel, index) in FUEL_TYPES"
          :key="fuel.id"
          class="flex flex-col items-center gap-2"
        >
          <p class="font-karla text-sm text-fuel-olive">
            Колонка {{ index + 1 }}
          </p>
          <FuelCard
            :name="fuel.name"
            :grade="fuel.grade"
            :price-per-liter="fuel.pricePerLiter"
            :selected="selectedFuel === fuel.id"
            @select="selectFuel(fuel.id)"
          />
        </div>
      </div>

      <!-- Подсказка для пользователя -->
      <p
        class="font-karla text-sm transition-all duration-300"
        :class="selectedFuel ? 'text-fuel-forest' : 'text-fuel-olive'"
      >
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
               focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-fuel-lime focus-visible:ring-offset-2 focus-visible:ring-offset-fuel-cream"
        :class="
          selectedFuel
            ? 'bg-fuel-lime text-white hover:bg-fuel-forest active:scale-95 shadow-md shadow-fuel-lime/25 cursor-pointer'
            : 'bg-fuel-lime/35 text-fuel-olive/60 cursor-not-allowed'
        "
        @click="handleNext"
      >
        Далее →
      </button>
    </main>
  </div>
</template>
