<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'

import { getFuelPrices } from '@/api/transaction.api'
import {
  createPriceVersion,
  listPriceVersions,
  type AdminPriceVersion,
} from '@/api/admin.api'
import type { FuelPrice } from '@/types'

const FUEL_DISPLAY_ORDER = ['АИ-92', 'АИ-95', 'АИ-100', 'ДТ'] as const

const currentPrices = ref<FuelPrice[]>([])
const isLoadingCurrent = ref(false)
const currentLoadError = ref<string | null>(null)

const versions = ref<AdminPriceVersion[]>([])
const isLoadingVersions = ref(false)
const versionsLoadError = ref<string | null>(null)

const formState = reactive<{
  effectiveFrom: string
  versionTag: string
  prices: Record<string, string>
}>({
  effectiveFrom: '',
  versionTag: '',
  prices: Object.fromEntries(FUEL_DISPLAY_ORDER.map((fuelType) => [fuelType, ''])),
})

const isSubmitting = ref(false)
const submitError = ref<string | null>(null)
const submitSuccess = ref<string | null>(null)

function formatTimestamp(iso: string): string {
  if (!iso) {
    return '—'
  }
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) {
    return iso
  }
  return date.toLocaleString('ru-RU', { hour12: false })
}

async function loadCurrent(): Promise<void> {
  isLoadingCurrent.value = true
  currentLoadError.value = null
  try {
    currentPrices.value = await getFuelPrices()
  } catch (error) {
    currentLoadError.value =
      error instanceof Error ? error.message : 'Не удалось загрузить текущие цены'
    currentPrices.value = []
  } finally {
    isLoadingCurrent.value = false
  }
}

async function loadVersions(): Promise<void> {
  isLoadingVersions.value = true
  versionsLoadError.value = null
  try {
    versions.value = await listPriceVersions()
  } catch (error) {
    versionsLoadError.value =
      error instanceof Error ? error.message : 'Не удалось загрузить историю версий цен'
    versions.value = []
  } finally {
    isLoadingVersions.value = false
  }
}

function resetFormWithCurrent(): void {
  for (const fuelType of FUEL_DISPLAY_ORDER) {
    const current = currentPrices.value.find((price) => price.fuelType === fuelType)
    formState.prices[fuelType] = current ? current.pricePerLiter.toFixed(2) : ''
  }
  const now = new Date()
  const pad = (value: number) => value.toString().padStart(2, '0')
  formState.effectiveFrom = `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}T${pad(
    now.getHours(),
  )}:${pad(now.getMinutes())}`
  formState.versionTag = ''
}

async function handleSubmit(): Promise<void> {
  submitError.value = null
  submitSuccess.value = null

  if (!formState.effectiveFrom) {
    submitError.value = 'Укажите дату и время вступления цен в силу'
    return
  }

  const items = FUEL_DISPLAY_ORDER.map((fuelType) => {
    const raw = formState.prices[fuelType]?.trim() ?? ''
    const value = Number.parseFloat(raw.replace(',', '.'))
    return { fuelType, raw, value }
  })

  for (const entry of items) {
    if (!entry.raw || Number.isNaN(entry.value) || entry.value <= 0) {
      submitError.value = `Укажите корректную цену для ${entry.fuelType}`
      return
    }
  }

  isSubmitting.value = true
  try {
    const version = await createPriceVersion({
      versionTag: formState.versionTag.trim() || undefined,
      effectiveFrom: new Date(formState.effectiveFrom).toISOString(),
      items: items.map((entry) => ({ fuelType: entry.fuelType, pricePerLiter: entry.value })),
    })
    submitSuccess.value = `Версия ${version.versionTag} создана`
    await Promise.all([loadCurrent(), loadVersions()])
  } catch (error) {
    submitError.value =
      error instanceof Error
        ? error.message
        : 'Не удалось создать новую версию цен. Проверьте соединение и авторизацию.'
  } finally {
    isSubmitting.value = false
  }
}

onMounted(async () => {
  await Promise.all([loadCurrent(), loadVersions()])
  resetFormWithCurrent()
})
</script>

<template>
  <section class="flex flex-col gap-10">
    <!-- Текущие цены читаем через публичную ручку /api/v1/fuel-prices -->
    <div class="bg-white rounded-2xl border border-fuel-olive/20 p-6 shadow-sm">
      <div class="flex items-center justify-between gap-4 mb-4">
        <h3 class="font-rubik font-semibold text-xl text-fuel-forest">
          Текущие цены
        </h3>
        <button
          type="button"
          class="font-karla text-sm text-fuel-forest underline-offset-4 hover:underline"
          @click="loadCurrent"
        >
          Обновить
        </button>
      </div>

      <p v-if="isLoadingCurrent" class="font-karla text-sm text-fuel-olive">
        Загружаем актуальные цены...
      </p>
      <p v-else-if="currentLoadError" class="font-karla text-sm text-red-600">
        {{ currentLoadError }}
      </p>
      <table v-else class="w-full text-left">
        <thead>
          <tr class="border-b border-fuel-olive/25">
            <th class="font-karla text-xs uppercase tracking-widest text-fuel-olive py-2">Топливо</th>
            <th class="font-karla text-xs uppercase tracking-widest text-fuel-olive py-2">Сорт</th>
            <th class="font-karla text-xs uppercase tracking-widest text-fuel-olive py-2">Цена, ₽/л</th>
            <th class="font-karla text-xs uppercase tracking-widest text-fuel-olive py-2">Версия</th>
            <th class="font-karla text-xs uppercase tracking-widest text-fuel-olive py-2">Действует с</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="price in currentPrices"
            :key="price.fuelType"
            class="border-b border-fuel-olive/10 last:border-b-0"
          >
            <td class="font-rubik font-medium text-fuel-forest py-3">{{ price.name }}</td>
            <td class="font-karla text-sm text-fuel-olive py-3">{{ price.grade }}</td>
            <td class="font-rubik font-semibold text-fuel-forest py-3">
              {{ price.pricePerLiter.toFixed(2) }}
            </td>
            <td class="font-karla text-sm text-fuel-olive py-3">{{ price.versionTag }}</td>
            <td class="font-karla text-sm text-fuel-olive py-3">
              {{ formatTimestamp(price.effectiveFrom) }}
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Форма создания новой версии -->
    <form
      class="bg-white rounded-2xl border border-fuel-olive/20 p-6 flex flex-col gap-5 shadow-sm"
      @submit.prevent="handleSubmit"
    >
      <div>
        <h3 class="font-rubik font-semibold text-xl text-fuel-forest mb-1">
          Новая версия цен
        </h3>
        <p class="font-karla text-sm text-fuel-olive">
          Укажите дату вступления в силу и цены за литр. Текущая версия останется в истории.
        </p>
      </div>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-5">
        <label class="flex flex-col gap-2">
          <span class="font-karla text-sm text-fuel-forest">Действует с</span>
          <input
            v-model="formState.effectiveFrom"
            type="datetime-local"
            required
            class="rounded-lg border border-fuel-olive/40 bg-fuel-cream/60 px-4 py-3
                   font-karla text-base text-fuel-forest
                   focus:outline-none focus:ring-2 focus:ring-fuel-lime focus:border-fuel-lime"
          />
        </label>
        <label class="flex flex-col gap-2">
          <span class="font-karla text-sm text-fuel-forest">Тег версии (необязательно)</span>
          <input
            v-model="formState.versionTag"
            type="text"
            placeholder="Автоматически, если пусто"
            class="rounded-lg border border-fuel-olive/40 bg-fuel-cream/60 px-4 py-3
                   font-karla text-base text-fuel-forest placeholder:text-fuel-olive/60
                   focus:outline-none focus:ring-2 focus:ring-fuel-lime focus:border-fuel-lime"
          />
        </label>
      </div>

      <div class="grid grid-cols-2 md:grid-cols-4 gap-5">
        <label
          v-for="fuelType in FUEL_DISPLAY_ORDER"
          :key="fuelType"
          class="flex flex-col gap-2"
        >
          <span class="font-karla text-sm text-fuel-forest">{{ fuelType }}, ₽/л</span>
          <input
            v-model="formState.prices[fuelType]"
            type="number"
            step="0.01"
            min="0.01"
            required
            class="rounded-lg border border-fuel-olive/40 bg-fuel-cream/60 px-4 py-3
                   font-rubik font-semibold text-lg text-fuel-forest
                   focus:outline-none focus:ring-2 focus:ring-fuel-lime focus:border-fuel-lime"
          />
        </label>
      </div>

      <div class="flex flex-col md:flex-row md:items-center gap-4 pt-2">
        <button
          type="submit"
          :disabled="isSubmitting"
          class="font-rubik font-semibold text-lg px-10 py-4 rounded-xl transition-all duration-200
                 bg-fuel-lime text-white hover:bg-fuel-forest active:scale-95 shadow-md shadow-fuel-lime/25
                 disabled:opacity-60 disabled:cursor-not-allowed
                 focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-fuel-lime focus-visible:ring-offset-2 focus-visible:ring-offset-white"
        >
          {{ isSubmitting ? 'Создаем...' : 'Создать версию' }}
        </button>
        <button
          type="button"
          class="font-karla text-sm text-fuel-olive underline-offset-4 hover:underline self-start"
          @click="resetFormWithCurrent"
        >
          Заполнить текущими ценами
        </button>
      </div>

      <p v-if="submitSuccess" class="font-karla text-sm text-fuel-forest">{{ submitSuccess }}</p>
      <p v-if="submitError" class="font-karla text-sm text-red-600">{{ submitError }}</p>
    </form>

    <!-- История версий -->
    <div class="bg-white rounded-2xl border border-fuel-olive/20 p-6 shadow-sm">
      <div class="flex items-center justify-between gap-4 mb-4">
        <h3 class="font-rubik font-semibold text-xl text-fuel-forest">
          История версий
        </h3>
        <button
          type="button"
          class="font-karla text-sm text-fuel-forest underline-offset-4 hover:underline"
          @click="loadVersions"
        >
          Обновить
        </button>
      </div>

      <p v-if="isLoadingVersions" class="font-karla text-sm text-fuel-olive">
        Загружаем историю версий...
      </p>
      <p v-else-if="versionsLoadError" class="font-karla text-sm text-red-600">
        {{ versionsLoadError }}
      </p>
      <p v-else-if="versions.length === 0" class="font-karla text-sm text-fuel-olive">
        Версий цен пока нет
      </p>
      <ul v-else class="flex flex-col gap-4">
        <li
          v-for="version in versions"
          :key="version.id"
          class="border border-fuel-olive/20 rounded-xl p-4 bg-fuel-cream/40"
        >
          <div class="flex items-center justify-between gap-4 mb-2">
            <p class="font-rubik font-semibold text-fuel-forest">
              {{ version.versionTag }}
            </p>
            <p class="font-karla text-xs text-fuel-olive">
              действует с {{ formatTimestamp(version.effectiveFrom) }}
            </p>
          </div>
          <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
            <div
              v-for="item in version.items"
              :key="item.fuelType"
              class="bg-white border border-fuel-olive/15 rounded-lg px-3 py-2"
            >
              <p class="font-karla text-xs text-fuel-olive">{{ item.fuelType }}</p>
              <p class="font-rubik font-semibold text-fuel-forest">
                {{ item.pricePerLiter.toFixed(2) }} ₽/л
              </p>
            </div>
          </div>
        </li>
      </ul>
    </div>
  </section>
</template>
