<script setup lang="ts">
import { computed, ref } from 'vue'

import { setMaintenance } from '@/api/admin.api'
import { useKioskStateStore } from '@/stores/kioskState'
import MaintenanceConfirmDialog from '@/components/MaintenanceConfirmDialog.vue'
import { categorize, screenLabel, blockedWarningText } from '@/utils/kioskScreen'

const kioskStateStore = useKioskStateStore()

const reasonInput = ref('')
const isSubmitting = ref(false)
const submitError = ref<string | null>(null)
const showConfirmDialog = ref(false)

const isMaintenance = computed(() => kioskStateStore.maintenance)
const currentReason = computed(() => kioskStateStore.reason)
const updatedAt = computed(() => kioskStateStore.state?.updatedAt ?? '')
const currentScreen = computed(() => kioskStateStore.screen)
const screenCategory = computed(() => categorize(currentScreen.value))

async function applyMaintenance(reason: string): Promise<void> {
  isSubmitting.value = true
  submitError.value = null
  try {
    const next = await setMaintenance({ enabled: true, reason })
    kioskStateStore.applyState(next)
  } catch (error) {
    submitError.value =
      error instanceof Error
        ? error.message
        : 'Не удалось переключить режим. Проверьте соединение с сервером и авторизацию.'
  } finally {
    isSubmitting.value = false
  }
}

async function disableMaintenance(): Promise<void> {
  isSubmitting.value = true
  submitError.value = null
  try {
    const next = await setMaintenance({ enabled: false, reason: '' })
    kioskStateStore.applyState(next)
    reasonInput.value = ''
  } catch (error) {
    submitError.value =
      error instanceof Error
        ? error.message
        : 'Не удалось переключить режим. Проверьте соединение с сервером и авторизацию.'
  } finally {
    isSubmitting.value = false
  }
}

function handleToggleClick(): void {
  if (isMaintenance.value) {
    disableMaintenance()
    return
  }
  const category = screenCategory.value
  if (category === 'blocked') return
  if (category === 'confirm') {
    showConfirmDialog.value = true
    return
  }
  applyMaintenance(reasonInput.value.trim())
}

function onDialogConfirm(reason: string): void {
  showConfirmDialog.value = false
  applyMaintenance(reason || reasonInput.value.trim())
}

function onDialogCancel(): void {
  showConfirmDialog.value = false
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
          Текущее состояние АЗС
        </p>
        <div class="flex flex-wrap items-baseline gap-x-3 gap-y-1">
          <h2
            class="font-rubik font-bold text-3xl leading-tight"
            :class="isMaintenance ? 'text-amber-700' : 'text-fuel-forest'"
          >
            {{ isMaintenance ? 'Ведутся технические работы' : 'АЗС в работе' }}
          </h2>
          <span
            v-if="!isMaintenance"
            class="font-karla text-base text-fuel-olive"
          >
            · {{ screenLabel(currentScreen) }}
          </span>
        </div>
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
          Кнопка переключает режим тех работ.
        </p>
      </div>

      <label class="flex flex-col gap-2" :class="{ 'opacity-60': isMaintenance }">
        <span class="font-karla text-sm text-fuel-forest">
          Причина (необязательно — отобразится на экране АЗС)
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

      <!-- Warning при заблокированном состоянии -->
      <div
        v-if="!isMaintenance && screenCategory === 'blocked'"
        class="flex items-start gap-3 rounded-lg border border-amber-200 bg-amber-50 px-4 py-3"
        role="alert"
      >
        <svg class="mt-0.5 h-4 w-4 shrink-0 text-amber-600" fill="currentColor" viewBox="0 0 20 20" aria-hidden="true">
          <path fill-rule="evenodd" d="M8.485 2.495c.673-1.167 2.357-1.167 3.03 0l6.28 10.875c.673 1.167-.17 2.625-1.516 2.625H3.72c-1.347 0-2.189-1.458-1.515-2.625L8.485 2.495zM10 5a.75.75 0 01.75.75v3.5a.75.75 0 01-1.5 0v-3.5A.75.75 0 0110 5zm0 9a1 1 0 100-2 1 1 0 000 2z" clip-rule="evenodd" />
        </svg>
        <p class="font-karla text-sm text-amber-800">
          {{ blockedWarningText(currentScreen) }}
        </p>
      </div>

      <button
        type="button"
        :disabled="isSubmitting || (!isMaintenance && screenCategory === 'blocked')"
        class="font-rubik font-semibold text-lg px-8 py-4 rounded-xl transition-all duration-200
               focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-offset-white"
        :class="isMaintenance
          ? 'bg-fuel-lime text-white hover:bg-fuel-forest active:scale-95 shadow-md shadow-fuel-lime/25 focus-visible:ring-fuel-lime cursor-pointer disabled:opacity-60 disabled:cursor-not-allowed'
          : screenCategory === 'blocked'
            ? 'bg-gray-300 text-gray-500 cursor-not-allowed'
            : 'bg-amber-500 text-white hover:bg-amber-600 active:scale-95 shadow-md shadow-amber-400/25 focus-visible:ring-amber-500 cursor-pointer disabled:opacity-60 disabled:cursor-not-allowed'"
        @click="handleToggleClick"
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

  <!-- Диалог подтверждения при переводе с confirm-экранов -->
  <MaintenanceConfirmDialog
    v-if="showConfirmDialog"
    @confirm="onDialogConfirm"
    @cancel="onDialogCancel"
  />
</template>
