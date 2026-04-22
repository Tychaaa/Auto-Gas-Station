<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'

import { getFuelPrices } from '@/api'
import { useTransactionFlowStore } from '@/stores'
import type { FuelPrice } from '@/types'
import StepIndicator from '@/components/StepIndicator.vue'
import FuelCard from '@/components/FuelCard.vue'

const router = useRouter()
const store = useTransactionFlowStore()

// Шаги индикатора в верхней части экрана
const STEPS = ['Топливо', 'Параметры', 'Оплата', 'Заправка'] as const
const FUEL_DISPLAY_ORDER = ['АИ-92', 'АИ-95', 'АИ-100', 'ДТ'] as const

const fuelPrices = ref<FuelPrice[]>([])
const isLoadingPrices = ref(false)
const pricesLoadError = ref('')

const selectedFuel = computed(() => store.selectionDraft.fuelType)
const hasFuelPrices = computed(() => fuelPrices.value.length > 0)
const orderedFuelPrices = computed(() => {
  const orderIndexMap = new Map<string, number>()
  FUEL_DISPLAY_ORDER.forEach((fuelType, index) => {
    orderIndexMap.set(fuelType, index)
  })

  return [...fuelPrices.value].sort((left, right) => {
    const leftIndex = orderIndexMap.get(left.fuelType) ?? Number.MAX_SAFE_INTEGER
    const rightIndex = orderIndexMap.get(right.fuelType) ?? Number.MAX_SAFE_INTEGER

    if (leftIndex !== rightIndex) {
      return leftIndex - rightIndex
    }
    return left.fuelType.localeCompare(right.fuelType, 'ru')
  })
})

// Сохраняет выбранный тип топлива в store
function selectFuel(fuelId: string): void {
  store.setSelectionDraft({ fuelType: fuelId })
}

// Переход к шагу выбора параметров
function handleNext(): void {
  if (!selectedFuel.value) return
  void router.push('/select/order')
}

async function loadFuelPrices(): Promise<void> {
  isLoadingPrices.value = true
  pricesLoadError.value = ''
  try {
    fuelPrices.value = await getFuelPrices()
  } catch (error) {
    const message = error instanceof Error ? error.message : 'Не удалось загрузить актуальные цены'
    pricesLoadError.value = message
    fuelPrices.value = []
  } finally {
    isLoadingPrices.value = false
  }
}

onMounted(() => {
  void loadFuelPrices()
})
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
        v-if="isLoadingPrices"
        class="font-karla text-base text-fuel-olive"
      >
        Загружаем актуальные цены...
      </div>

      <div
        v-else-if="pricesLoadError"
        class="flex flex-col items-center gap-3"
      >
        <p class="font-karla text-base text-red-600">
          {{ pricesLoadError }}
        </p>
        <button
          type="button"
          class="font-rubik font-medium px-4 py-2 rounded-lg bg-fuel-forest text-white hover:bg-fuel-olive transition-colors"
          @click="loadFuelPrices"
        >
          Повторить
        </button>
      </div>

      <div
        v-else
        class="grid grid-cols-4 gap-5 w-full max-w-5xl"
        role="group"
        aria-label="Виды топлива"
      >
        <div
          v-for="(fuel, index) in orderedFuelPrices"
          :key="fuel.fuelType"
          class="flex flex-col items-center gap-2"
        >
          <p class="font-karla text-sm text-fuel-olive">
            Колонка {{ index + 1 }}
          </p>
          <FuelCard
            :name="fuel.name"
            :grade="fuel.grade"
            :price-per-liter="fuel.pricePerLiter"
            :selected="selectedFuel === fuel.fuelType"
            @select="selectFuel(fuel.fuelType)"
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
        :disabled="!selectedFuel || !hasFuelPrices"
        :aria-disabled="!selectedFuel || !hasFuelPrices"
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
