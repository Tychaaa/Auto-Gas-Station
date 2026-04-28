<script setup lang="ts">
import { computed, ref } from 'vue'

import { setMaintenance } from '@/api/admin.api'
import { useKioskStateStore } from '@/stores/kioskState'

const kioskStateStore = useKioskStateStore()

const reasonInput = ref('')
const isSubmitting = ref(false)
const submitError = ref<string | null>(null)

const isMaintenance = computed(() => kioskStateStore.maintenance)
const currentReason = computed(() => kioskStateStore.reason)
const updatedAt = computed(() => kioskStateStore.state?.updatedAt ?? '')

// Переключает режим на противоположный и обновляет стор актуальным состоянием
async function toggleMaintenance(): Promise<void> {
  isSubmitting.value = true
  submitError.value = null
  try {
    const nextEnabled = !isMaintenance.value
    const next = await setMaintenance({
      enabled: nextEnabled,
      reason: nextEnabled ? reasonInput.value.trim() : '',
    })
    kioskStateStore.applyState(next)
    if (!nextEnabled) {
      reasonInput.value = ''
    }
  } catch (error) {
    submitError.value =
      error instanceof Error
        ? error.message
        : 'Не удалось переключить режим. Проверьте соединение с сервером и авторизацию.'
  } finally {
    isSubmitting.value = false
  }
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
</script>

<template>
  <section class="flex flex-col gap-8">
    <!-- Текущий статус крупной плашкой в брендовых цветах -->
    <div
      class="rounded-2xl border p-8 flex flex-col md:flex-row md:items-center md:justify-between gap-6 shadow-sm"
      :class="isMaintenance
        ? 'bg-amber-50 border-amber-200'
        : 'bg-white border-fuel-olive/20'"
    >
      <div class="flex flex-col gap-2">
        <p class="font-karla text-xs uppercase tracking-widest text-fuel-olive">
          Текущее состояние киоска
        </p>
        <h2
          class="font-rubik font-bold text-3xl leading-tight"
          :class="isMaintenance ? 'text-amber-700' : 'text-fuel-forest'"
        >
          {{ isMaintenance ? 'Ведутся технические работы' : 'Киоск в работе' }}
        </h2>
        <p v-if="isMaintenance && currentReason" class="font-karla text-base text-fuel-forest/80">
          Причина: {{ currentReason }}
        </p>
        <p class="font-karla text-sm text-fuel-olive">
          Обновлено: {{ formatTimestamp(updatedAt) }}
        </p>
      </div>

      <div
        class="inline-flex h-16 w-16 rounded-full items-center justify-center shrink-0"
        :class="isMaintenance ? 'bg-amber-200' : 'bg-fuel-lime/25'"
        aria-hidden="true"
      >
        <span
          class="h-6 w-6 rounded-full"
          :class="isMaintenance ? 'bg-amber-500 animate-pulse' : 'bg-fuel-forest'"
        />
      </div>
    </div>

    <!-- Управление режимом: кнопка + причина -->
    <div class="bg-white rounded-2xl border border-fuel-olive/20 p-8 flex flex-col gap-5 shadow-sm">
      <div>
        <h3 class="font-rubik font-semibold text-xl text-fuel-forest mb-1">
          Управление режимом
        </h3>
        <p class="font-karla text-sm text-fuel-olive">
          Кнопка переключает режим тех работ. Киоск подхватит изменение в течение ~3 секунд.
        </p>
      </div>

      <label class="flex flex-col gap-2" :class="{ 'opacity-60': isMaintenance }">
        <span class="font-karla text-sm text-fuel-forest">
          Причина (необязательно — отобразится на экране киоска)
        </span>
        <input
          v-model="reasonInput"
          :disabled="isMaintenance || isSubmitting"
          type="text"
          placeholder="Например: замена картриджа с чеками"
          class="rounded-lg border border-fuel-olive/40 bg-fuel-cream/60 px-4 py-3
                 font-karla text-base text-fuel-forest placeholder:text-fuel-olive/60
                 focus:outline-none focus:ring-2 focus:ring-fuel-lime focus:border-fuel-lime
                 disabled:cursor-not-allowed"
          maxlength="200"
        />
      </label>

      <button
        type="button"
        :disabled="isSubmitting"
        class="font-rubik font-semibold text-lg px-8 py-4 rounded-xl transition-all duration-200
               focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-offset-white
               disabled:cursor-not-allowed disabled:opacity-60"
        :class="isMaintenance
          ? 'bg-fuel-lime text-white hover:bg-fuel-forest active:scale-95 shadow-md shadow-fuel-lime/25 focus-visible:ring-fuel-lime cursor-pointer'
          : 'bg-amber-500 text-white hover:bg-amber-600 active:scale-95 shadow-md shadow-amber-400/25 focus-visible:ring-amber-500 cursor-pointer'"
        @click="toggleMaintenance"
      >
        {{ isSubmitting
          ? 'Применяем...'
          : isMaintenance
            ? 'Вернуть в работу'
            : 'Перевести в тех. работы' }}
      </button>

      <p v-if="submitError" class="font-karla text-sm text-red-600">
        {{ submitError }}
      </p>
    </div>
  </section>
</template>
