<script setup lang="ts">
import { onMounted, ref } from 'vue'

import {
  listDispensers,
  listPriceVersions,
  updateDispenser,
  type AdminDispenserView,
} from '@/api/admin.api'

const dispensers = ref<AdminDispenserView[]>([])
const availableFuelTypes = ref<string[]>([])
const isLoading = ref(true)
const loadError = ref('')

interface Draft {
  fuelType: string
  enabled: boolean
}

const drafts = ref<Record<number, Draft>>({})
const savingId = ref<number | null>(null)
const saveErrors = ref<Record<number, string>>({})

function formatTimestamp(iso: string): string {
  if (!iso) return '—'
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return iso
  return date.toLocaleString('ru-RU', { hour12: false })
}

function isDirty(dispenser: AdminDispenserView): boolean {
  const draft = drafts.value[dispenser.id]
  if (!draft) return false
  return draft.fuelType !== dispenser.fuelType || draft.enabled !== dispenser.enabled
}

function initDraft(d: AdminDispenserView): void {
  drafts.value[d.id] = { fuelType: d.fuelType, enabled: d.enabled }
}

function setDraftFuelType(id: number, fuelType: string): void {
  const draft = drafts.value[id]
  if (draft) draft.fuelType = fuelType
}

function toggleDraftEnabled(id: number): void {
  const draft = drafts.value[id]
  if (draft) draft.enabled = !draft.enabled
}

onMounted(async () => {
  try {
    const [dispenserList, versions] = await Promise.all([listDispensers(), listPriceVersions()])
    dispensers.value = dispenserList
    for (const d of dispenserList) initDraft(d)
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

async function onSave(dispenser: AdminDispenserView): Promise<void> {
  const draft = drafts.value[dispenser.id]
  if (!draft) return
  savingId.value = dispenser.id
  saveErrors.value = { ...saveErrors.value, [dispenser.id]: '' }
  try {
    const updated = await updateDispenser(dispenser.id, draft)
    const idx = dispensers.value.findIndex((d) => d.id === dispenser.id)
    if (idx !== -1) {
      dispensers.value[idx] = updated
      initDraft(updated)
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
  <section class="flex flex-col gap-8">
    <div v-if="isLoading" class="font-karla text-fuel-olive text-center py-12">
      Загрузка…
    </div>

    <div v-else-if="loadError" class="font-karla text-sm text-red-600 text-center py-12">
      {{ loadError }}
    </div>

    <template v-else>
      <div class="grid grid-cols-2 lg:grid-cols-4 gap-5">
        <div
          v-for="dispenser in dispensers"
          :key="dispenser.id"
          class="flex flex-col rounded-2xl border-2 overflow-hidden transition-all duration-200"
          :class="[
            !drafts[dispenser.id]?.enabled
              ? 'border-gray-300 opacity-70'
              : isDirty(dispenser)
                ? 'border-amber-400 shadow-sm shadow-amber-200'
                : 'border-fuel-lime/50',
          ]"
        >
          <!-- Верхняя часть карточки -->
          <div
            class="flex flex-col items-center justify-center gap-3 py-8 px-5"
            :class="drafts[dispenser.id]?.enabled ? 'bg-white' : 'bg-gray-50'"
          >
            <span class="font-karla text-xs text-fuel-olive uppercase tracking-widest">
              {{ dispenser.label }} — Резервуар {{ dispenser.id }}
            </span>

            <span
              class="font-rubik font-bold text-4xl leading-none tracking-tight"
              :class="drafts[dispenser.id]?.enabled ? 'text-fuel-forest' : 'text-gray-400'"
            >
              {{ drafts[dispenser.id]?.fuelType || '—' }}
            </span>

            <!-- TODO(топливомер): показывать остаток и объём резервуара когда будет датчик уровня топлива -->
            <!-- <span class="font-karla text-sm text-fuel-olive/70">
              {{ dispenser.tankRemaining.toLocaleString('ru-RU') }} л / {{ dispenser.tankVolume.toLocaleString('ru-RU') }} л
            </span> -->

            <span
              class="font-karla text-xs font-semibold tracking-wide px-3 py-1 rounded-full"
              :class="drafts[dispenser.id]?.enabled
                ? 'bg-fuel-cream text-fuel-lime'
                : 'bg-gray-200 text-gray-500'"
            >
              {{ drafts[dispenser.id]?.enabled ? 'Работает' : 'Отключена' }}
            </span>
          </div>

          <!-- Нижняя часть — управление -->
          <div
            class="border-t flex flex-col gap-3 p-4"
            :class="drafts[dispenser.id]?.enabled
              ? 'bg-fuel-cream/40 border-fuel-olive/15'
              : 'bg-gray-50 border-gray-200'"
          >
            <p v-if="saveErrors[dispenser.id]" class="font-karla text-xs text-red-600">
              {{ saveErrors[dispenser.id] }}
            </p>

            <select
              :id="`fuel-type-${dispenser.id}`"
              :disabled="savingId === dispenser.id"
              :value="drafts[dispenser.id]?.fuelType"
              class="w-full font-karla text-sm text-fuel-forest border border-fuel-olive/30 rounded-lg px-3 py-2
                     bg-white focus:outline-none focus:ring-2 focus:ring-fuel-lime/50
                     disabled:opacity-60 disabled:cursor-not-allowed cursor-pointer"
              @change="(e) => setDraftFuelType(dispenser.id, (e.target as HTMLSelectElement).value)"
            >
              <option value="">— Не назначено —</option>
              <option v-for="ft in availableFuelTypes" :key="ft" :value="ft">{{ ft }}</option>
            </select>

            <div class="flex gap-2">
              <button
                type="button"
                :disabled="savingId === dispenser.id"
                class="flex-1 font-karla text-xs px-3 py-2 rounded-lg border transition-colors
                       disabled:opacity-60 disabled:cursor-not-allowed"
                :class="drafts[dispenser.id]?.enabled
                  ? 'border-red-300 text-red-600 hover:bg-red-50'
                  : 'border-fuel-lime/50 text-fuel-forest hover:bg-fuel-lime/10'"
                @click="toggleDraftEnabled(dispenser.id)"
              >
                {{ drafts[dispenser.id]?.enabled ? 'Отключить' : 'Включить' }}
              </button>

              <button
                type="button"
                :disabled="savingId === dispenser.id || !isDirty(dispenser)"
                class="flex-1 font-rubik font-semibold text-xs px-3 py-2 rounded-lg transition-all
                       bg-fuel-lime text-white hover:bg-fuel-forest active:scale-95
                       disabled:opacity-40 disabled:cursor-not-allowed disabled:hover:bg-fuel-lime"
                @click="onSave(dispenser)"
              >
                {{ savingId === dispenser.id ? '…' : 'Сохранить' }}
              </button>
            </div>

            <span class="font-karla text-xs text-fuel-olive/40">
              {{ formatTimestamp(dispenser.updatedAt) }}
            </span>
          </div>
        </div>
      </div>

      <p v-if="dispensers.length === 0" class="font-karla text-fuel-olive text-center py-12">
        Колонки не найдены. Проверьте FUEL_DISPENSER_ADDRESSES в .env
      </p>
    </template>
  </section>
</template>
