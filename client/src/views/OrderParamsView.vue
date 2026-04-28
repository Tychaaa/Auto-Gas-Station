<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'

import StepIndicator from '@/components/StepIndicator.vue'
import { useTransactionFlowStore } from '@/stores'

interface PresetOption {
  value: number
  label: string
}

const router = useRouter()
const store = useTransactionFlowStore()

const STEPS = ['Топливо', 'Параметры', 'Оплата', 'Заправка'] as const

const amountPresets: readonly PresetOption[] = [
  { value: 500, label: '500 ₽' },
  { value: 1000, label: '1000 ₽' },
  { value: 1500, label: '1500 ₽' },
] as const

const litersPresets: readonly PresetOption[] = [
  { value: 10, label: '10 л' },
  { value: 20, label: '20 л' },
  { value: 40, label: '40 л' },
] as const

const selectedFuel = computed(() => store.selectionDraft.fuelType)
const selectedMode = computed(() => store.selectionDraft.orderMode)
const amountRub = computed(() => store.selectionDraft.amountRub)
const liters = computed(() => store.selectionDraft.liters)
const canContinue = computed(() => store.isSelectionDraftValid)
const isAmountPresetSelected = computed(
  () => selectedMode.value === 'amount' && amountPresets.some((preset) => preset.value === amountRub.value),
)
const isLitersPresetSelected = computed(
  () => selectedMode.value === 'liters' && litersPresets.some((preset) => preset.value === liters.value),
)
const isAnyPresetSelected = computed(() => isAmountPresetSelected.value || isLitersPresetSelected.value)

function applyAmountSelection(rawAmountRub: number): void {
  const nextAmountRub = Number.isFinite(rawAmountRub) && rawAmountRub > 0 ? Math.floor(rawAmountRub) : 0

  store.setSelectionDraft({
    orderMode: 'amount',
    amountRub: nextAmountRub,
    liters: 0,
  })
}

function applyLitersSelection(rawLiters: number): void {
  const nextLiters = Number.isFinite(rawLiters) && rawLiters > 0 ? Number(rawLiters.toFixed(2)) : 0

  store.setSelectionDraft({
    orderMode: 'liters',
    amountRub: 0,
    liters: nextLiters,
  })
}

function onAmountInput(event: Event): void {
  const inputElement = event.target as HTMLInputElement
  const parsedValue = Number(inputElement.value)
  applyAmountSelection(parsedValue)
}

function onLitersInput(event: Event): void {
  const inputElement = event.target as HTMLInputElement
  const parsedValue = Number(inputElement.value)
  applyLitersSelection(parsedValue)
}

function selectAmountPreset(amountPreset: number): void {
  applyAmountSelection(amountPreset)
}

function selectLitersPreset(litersPreset: number): void {
  applyLitersSelection(litersPreset)
}

async function handleContinue(): Promise<void> {
  if (!canContinue.value) return
  await router.push('/payment/method')
}

async function goBack(): Promise<void> {
  await router.push('/select/fuel')
}
</script>

<template>
  <div class="min-h-screen flex flex-col bg-fuel-cream">
    <header class="bg-fuel-forest border-b border-fuel-olive/35 py-5 px-6 text-center shrink-0 shadow-sm sm:px-10">
      <p class="font-karla text-xs text-white/80 tracking-widest uppercase mb-1">
        Автоматизированная АЗС
      </p>
      <h1 class="font-rubik font-bold text-3xl text-white leading-tight">
        Параметры заправки
      </h1>
    </header>

    <StepIndicator
      :steps="STEPS"
      :current="2"
    />

    <main class="flex-1 w-full px-4 py-6 sm:px-6 sm:py-8">
      <section class="mx-auto w-full max-w-3xl flex flex-col gap-5">
        <p
          v-if="selectedFuel"
          class="font-karla text-sm text-fuel-forest/80"
        >
          Выбрано топливо: {{ selectedFuel }}
        </p>

        <div class="grid grid-cols-2 gap-5">
          <article
            class="rounded-2xl border-2 bg-white p-5 shadow-sm transition-all duration-200 sm:p-6"
            :class="
              selectedMode === 'amount'
                ? 'border-fuel-lime shadow-fuel-lime/20 shadow-md'
                : 'border-fuel-lime/30 hover:border-fuel-lime/50'
            "
          >
            <div class="flex items-center justify-between gap-3 mb-3">
              <h2 class="font-rubik font-semibold text-xl text-fuel-forest">
                По сумме
              </h2>
              <span
                class="font-karla text-xs font-semibold tracking-wide uppercase px-3 py-1 rounded-full transition-colors duration-200"
                :class="selectedMode === 'amount' ? 'bg-fuel-lime text-white' : 'bg-fuel-cream text-fuel-olive'"
              >
                ₽
              </span>
            </div>
            <label class="block">
              <span class="sr-only">Введите сумму в рублях</span>
              <input
                type="number"
                min="0"
                step="100"
                inputmode="numeric"
                placeholder="Например, 1500"
                class="w-full rounded-xl border border-fuel-olive/25 bg-fuel-cream/70 px-4 py-3
                       font-rubik text-lg text-fuel-forest transition-all duration-200
                       placeholder:text-fuel-olive/60
                       focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-fuel-lime focus-visible:border-fuel-lime"
                :value="amountRub || ''"
                @input="onAmountInput"
              >
            </label>
          </article>

          <article
            class="rounded-2xl border-2 bg-white p-5 shadow-sm transition-all duration-200 sm:p-6"
            :class="
              selectedMode === 'liters'
                ? 'border-fuel-lime shadow-fuel-lime/20 shadow-md'
                : 'border-fuel-lime/30 hover:border-fuel-lime/50'
            "
          >
            <div class="flex items-center justify-between gap-3 mb-3">
              <h2 class="font-rubik font-semibold text-xl text-fuel-forest">
                По литрам
              </h2>
              <span
                class="font-karla text-xs font-semibold tracking-wide uppercase px-3 py-1 rounded-full transition-colors duration-200"
                :class="selectedMode === 'liters' ? 'bg-fuel-lime text-white' : 'bg-fuel-cream text-fuel-olive'"
              >
                л
              </span>
            </div>
            <label class="block">
              <span class="sr-only">Введите количество литров</span>
              <input
                type="number"
                min="0"
                step="0.1"
                inputmode="decimal"
                placeholder="Например, 20"
                class="w-full rounded-xl border border-fuel-olive/25 bg-fuel-cream/70 px-4 py-3
                       font-rubik text-lg text-fuel-forest transition-all duration-200
                       placeholder:text-fuel-olive/60
                       focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-fuel-lime focus-visible:border-fuel-lime"
                :value="liters || ''"
                @input="onLitersInput"
              >
            </label>
          </article>
        </div>

        <article class="rounded-2xl border-2 border-fuel-lime/30 bg-white p-5 shadow-sm sm:p-6">
          <div class="flex items-center justify-between gap-3 mb-4">
            <h2 class="font-rubik font-semibold text-xl text-fuel-forest">
              Готовые варианты
            </h2>
            <span
              class="font-karla text-xs font-semibold tracking-wide uppercase px-3 py-1 rounded-full transition-colors duration-200"
              :class="isAnyPresetSelected ? 'bg-fuel-lime text-white' : 'bg-fuel-cream text-fuel-olive'"
            >
              Быстрый выбор
            </span>
          </div>

          <div class="grid grid-cols-2 gap-3">
            <div
              role="group"
              aria-label="Готовые варианты по сумме"
              class="flex flex-col gap-3"
            >
              <button
                v-for="preset in amountPresets"
                :key="preset.value"
                type="button"
                class="rounded-xl border px-3 py-3 text-left transition-all duration-200
                       focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-fuel-lime"
                :class="
                  selectedMode === 'amount' && amountRub === preset.value
                    ? 'border-fuel-lime bg-fuel-lime text-white shadow-md shadow-fuel-lime/20'
                    : 'border-fuel-olive/25 bg-fuel-cream/60 text-fuel-forest hover:border-fuel-lime/60 hover:bg-white active:scale-[0.98]'
                "
                :aria-pressed="selectedMode === 'amount' && amountRub === preset.value"
                @click="selectAmountPreset(preset.value)"
              >
                <p class="font-rubik font-semibold text-lg leading-tight">
                  {{ preset.label }}
                </p>
              </button>
            </div>

            <div
              role="group"
              aria-label="Готовые варианты по литрам"
              class="flex flex-col gap-3"
            >
              <button
                v-for="preset in litersPresets"
                :key="preset.value"
                type="button"
                class="rounded-xl border px-3 py-3 text-left transition-all duration-200
                       focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-fuel-lime"
                :class="
                  selectedMode === 'liters' && liters === preset.value
                    ? 'border-fuel-lime bg-fuel-lime text-white shadow-md shadow-fuel-lime/20'
                    : 'border-fuel-olive/25 bg-fuel-cream/60 text-fuel-forest hover:border-fuel-lime/60 hover:bg-white active:scale-[0.98]'
                "
                :aria-pressed="selectedMode === 'liters' && liters === preset.value"
                @click="selectLitersPreset(preset.value)"
              >
                <p class="font-rubik font-semibold text-lg leading-tight">
                  {{ preset.label }}
                </p>
              </button>
            </div>
          </div>
        </article>

        <div class="mt-1 flex items-center gap-3">
          <button
            type="button"
            class="font-rubik font-medium text-base px-5 py-3 rounded-xl border border-fuel-olive/30
                   text-fuel-olive bg-white transition-all duration-200 hover:border-fuel-olive/50 hover:bg-fuel-cream/60 active:scale-[0.98]
                   focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-fuel-lime"
            @click="goBack"
          >
            Назад
          </button>

          <button
            type="button"
            :disabled="!canContinue"
            :aria-disabled="!canContinue"
            class="ml-auto font-rubik font-semibold text-lg px-8 py-3 rounded-xl
                   transition-all duration-200
                   focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-fuel-lime focus-visible:ring-offset-2 focus-visible:ring-offset-white"
            :class="
              canContinue
                ? 'bg-fuel-lime text-white hover:bg-fuel-forest active:scale-[0.98] shadow-md shadow-fuel-lime/25'
                : 'bg-fuel-lime/35 text-fuel-olive/60 cursor-not-allowed'
            "
            @click="handleContinue"
          >
          {{
            'Перейти к оплате →'
          }}
          </button>
        </div>
      </section>
    </main>
  </div>
</template>
