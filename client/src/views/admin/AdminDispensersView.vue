<script setup lang="ts">
import { onMounted, ref } from 'vue'

import {
  assignDispenserFuelType,
  listDispensers,
  listPriceVersions,
  type AdminDispenserView,
} from '@/api/admin.api'

const dispensers = ref<AdminDispenserView[]>([])
const availableFuelTypes = ref<string[]>([])
const isLoading = ref(true)
const loadError = ref('')
const savingId = ref<number | null>(null)
const saveErrors = ref<Record<number, string>>({})

function formatTimestamp(iso: string): string {
  if (!iso) return '—'
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return iso
  return date.toLocaleString('ru-RU', { hour12: false })
}

onMounted(async () => {
  try {
    const [dispenserList, versions] = await Promise.all([
      listDispensers(),
      listPriceVersions(),
    ])
    dispensers.value = dispenserList

    const latest = versions[0]
    if (latest?.items) {
      availableFuelTypes.value = latest.items.map((item) => item.fuelType)
    }
  } catch (e) {
    loadError.value = e instanceof Error ? e.message : String(e)
  } finally {
    isLoading.value = false
  }
})

async function onAssign(dispenser: AdminDispenserView, fuelType: string): Promise<void> {
  savingId.value = dispenser.id
  saveErrors.value = { ...saveErrors.value, [dispenser.id]: '' }
  try {
    const updated = await assignDispenserFuelType(dispenser.id, fuelType)
    const idx = dispensers.value.findIndex((d) => d.id === dispenser.id)
    if (idx !== -1) {
      dispensers.value[idx] = updated
    }
  } catch (e) {
    saveErrors.value = {
      ...saveErrors.value,
      [dispenser.id]: e instanceof Error ? e.message : String(e),
    }
  } finally {
    savingId.value = null
  }
}
</script>

<template>
  <section class="flex flex-col gap-6">
    <div v-if="isLoading" class="font-karla text-fuel-olive text-center py-12">
      Загрузка…
    </div>

    <div v-else-if="loadError" class="font-karla text-sm text-red-600 text-center py-12">
      {{ loadError }}
    </div>

    <template v-else>
      <div
        v-for="dispenser in dispensers"
        :key="dispenser.id"
        class="bg-white rounded-2xl border border-fuel-olive/20 p-8 flex flex-col gap-5 shadow-sm"
      >
        <div class="flex flex-col md:flex-row md:items-start md:justify-between gap-4">
          <div>
            <h3 class="font-rubik font-semibold text-xl text-fuel-forest mb-1">
              {{ dispenser.label }}
            </h3>
            <p class="font-karla text-sm text-fuel-olive">
              AZT-адрес: {{ dispenser.id }}
            </p>
          </div>

          <div
            v-if="dispenser.fuelType"
            class="inline-flex items-center gap-2 px-3 py-1.5 rounded-full bg-fuel-lime/15 border border-fuel-lime/40"
          >
            <span class="h-2.5 w-2.5 rounded-full bg-fuel-lime" aria-hidden="true" />
            <span class="font-karla text-sm text-fuel-forest">{{ dispenser.fuelType }}</span>
          </div>
          <div
            v-else
            class="inline-flex items-center gap-2 px-3 py-1.5 rounded-full bg-gray-100 border border-gray-300"
          >
            <span class="h-2.5 w-2.5 rounded-full bg-gray-400" aria-hidden="true" />
            <span class="font-karla text-sm text-gray-700">Не назначено</span>
          </div>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm font-karla">
          <div class="rounded-lg border border-fuel-olive/20 bg-fuel-cream/30 p-4">
            <p class="text-xs uppercase tracking-widest text-fuel-olive mb-1">Текущий вид топлива</p>
            <p class="text-fuel-forest font-medium">{{ dispenser.fuelType || '—' }}</p>
          </div>
          <div class="rounded-lg border border-fuel-olive/20 bg-fuel-cream/30 p-4">
            <p class="text-xs uppercase tracking-widest text-fuel-olive mb-1">Обновлено</p>
            <p class="text-fuel-forest font-medium">{{ formatTimestamp(dispenser.updatedAt) }}</p>
          </div>
        </div>

        <p v-if="saveErrors[dispenser.id]" class="font-karla text-sm text-red-600">
          {{ saveErrors[dispenser.id] }}
        </p>

        <div class="flex flex-col sm:flex-row sm:items-center gap-3">
          <label
            :for="`fuel-type-${dispenser.id}`"
            class="font-karla text-sm text-fuel-olive shrink-0"
          >
            Назначить вид топлива:
          </label>
          <select
            :id="`fuel-type-${dispenser.id}`"
            :disabled="savingId === dispenser.id"
            :value="dispenser.fuelType"
            class="font-karla text-sm text-fuel-forest border border-fuel-olive/30 rounded-lg px-3 py-2
                   bg-white focus:outline-none focus:ring-2 focus:ring-fuel-lime/50
                   disabled:opacity-60 disabled:cursor-not-allowed cursor-pointer"
            @change="(e) => onAssign(dispenser, (e.target as HTMLSelectElement).value)"
          >
            <option value="">— Не назначено —</option>
            <option
              v-for="ft in availableFuelTypes"
              :key="ft"
              :value="ft"
            >
              {{ ft }}
            </option>
          </select>
          <span
            v-if="savingId === dispenser.id"
            class="font-karla text-sm text-fuel-olive animate-pulse"
          >
            Сохранение…
          </span>
        </div>
      </div>

      <p v-if="dispensers.length === 0" class="font-karla text-fuel-olive text-center py-12">
        Колонки не найдены. Проверьте параметр FUEL_DISPENSER_COUNT.
      </p>
    </template>
  </section>
</template>
