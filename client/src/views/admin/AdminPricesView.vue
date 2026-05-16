<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'

import { getFuelPrices } from '@/api/transaction.api'
import {
  createPriceVersion,
  deletePriceVersion,
  listPriceVersions,
  type AdminPriceVersion,
  type AdminPriceVersionItem,
} from '@/api/admin.api'
import type { FuelPrice } from '@/types'

const FUEL_DISPLAY_ORDER = ['АИ-92', 'АИ-95', 'АИ-100', 'ДТ'] as const
const VERSIONS_PREVIEW_COUNT = 3

const currentPrices = ref<FuelPrice[]>([])
const isLoadingCurrent = ref(false)
const currentLoadError = ref<string | null>(null)

const versions = ref<AdminPriceVersion[]>([])
const isLoadingVersions = ref(false)
const versionsLoadError = ref<string | null>(null)
const showAllVersions = ref(false)

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

const deletingId = ref<number | null>(null)
const deleteError = ref<string | null>(null)

const displayedVersions = computed(() =>
  showAllVersions.value ? versions.value : versions.value.slice(0, VERSIONS_PREVIEW_COUNT),
)
const hasHiddenVersions = computed(() => versions.value.length > VERSIONS_PREVIEW_COUNT)

function sortedItems(items: AdminPriceVersionItem[]): AdminPriceVersionItem[] {
  return [...items].sort(
    (a, b) => FUEL_DISPLAY_ORDER.indexOf(a.fuelType as (typeof FUEL_DISPLAY_ORDER)[number]) - FUEL_DISPLAY_ORDER.indexOf(b.fuelType as (typeof FUEL_DISPLAY_ORDER)[number]),
  )
}

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

function copyVersionToForm(version: AdminPriceVersion): void {
  for (const fuelType of FUEL_DISPLAY_ORDER) {
    const item = version.items.find((i) => i.fuelType === fuelType)
    formState.prices[fuelType] = item ? item.pricePerLiter.toFixed(2) : ''
  }
}

async function handleDeleteVersion(id: number): Promise<void> {
  if (!confirm('Удалить эту версию цен?')) return
  deletingId.value = id
  deleteError.value = null
  try {
    await deletePriceVersion(id)
    await Promise.all([loadCurrent(), loadVersions()])
  } catch (error) {
    deleteError.value = error instanceof Error ? error.message : 'Не удалось удалить версию'
  } finally {
    deletingId.value = null
  }
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
    <!-- Текущие цены -->
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
            type="text"
            inputmode="decimal"
            required
            placeholder="0.00"
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
      <template v-else>
        <p v-if="deleteError" class="font-karla text-sm text-red-600 mb-3">{{ deleteError }}</p>
        <ul class="flex flex-col gap-4">
          <li
            v-for="version in displayedVersions"
            :key="version.id"
            class="border border-fuel-olive/20 rounded-xl p-4 bg-fuel-cream/40"
          >
            <div class="flex items-center justify-between gap-4 mb-3">
              <p class="font-rubik font-semibold text-fuel-forest">
                {{ version.versionTag }}
              </p>
              <div class="flex items-center gap-3">
                <p class="font-karla text-xs text-fuel-olive">
                  действует с {{ formatTimestamp(version.effectiveFrom) }}
                </p>
                <!-- Копировать в форму -->
                <button
                  type="button"
                  title="Скопировать цены в форму"
                  class="flex items-center gap-1 font-karla text-xs text-fuel-forest/70 hover:text-fuel-forest transition-colors"
                  @click="copyVersionToForm(version)"
                >
                  <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <rect x="9" y="9" width="13" height="13" rx="2" ry="2"/>
                    <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>
                  </svg>
                  <span class="hidden sm:inline">Скопировать</span>
                </button>
                <!-- Удалить -->
                <button
                  type="button"
                  title="Удалить версию"
                  :disabled="deletingId === version.id"
                  class="flex items-center gap-1 font-karla text-xs text-red-400 hover:text-red-600 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                  @click="handleDeleteVersion(version.id)"
                >
                  <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <polyline points="3 6 5 6 21 6"/>
                    <path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/>
                    <path d="M10 11v6"/>
                    <path d="M14 11v6"/>
                    <path d="M9 6V4a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2"/>
                  </svg>
                  <span class="hidden sm:inline">{{ deletingId === version.id ? 'Удаляем...' : 'Удалить' }}</span>
                </button>
              </div>
            </div>
            <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
              <div
                v-for="item in sortedItems(version.items)"
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

        <!-- Кнопка свернуть/развернуть -->
        <div v-if="hasHiddenVersions" class="flex justify-center mt-4">
          <button
            type="button"
            class="flex items-center gap-1 text-fuel-olive hover:text-fuel-forest transition-colors"
            :title="showAllVersions ? 'Свернуть' : `Показать ещё ${versions.length - VERSIONS_PREVIEW_COUNT}`"
            @click="showAllVersions = !showAllVersions"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              class="w-5 h-5 transition-transform duration-200"
              :class="{ 'rotate-180': showAllVersions }"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <polyline points="6 9 12 15 18 9"/>
            </svg>
          </button>
        </div>
      </template>
    </div>
  </section>
</template>
