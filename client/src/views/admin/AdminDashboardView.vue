<script setup lang="ts">
import { computed, ref } from 'vue'

import { setMaintenance, requestSystemReboot, type AdminSystemRebootMethod } from '@/api/admin.api'
import { useKioskStateStore } from '@/stores/kioskState'
import { useWatchdogStateStore } from '@/stores/watchdogState'
import MaintenanceConfirmDialog from '@/components/MaintenanceConfirmDialog.vue'
import { categorize, screenLabel, blockedWarningText } from '@/utils/kioskScreen'

const kioskStateStore = useKioskStateStore()
const watchdogStateStore = useWatchdogStateStore()

const reasonInput = ref('')
const isSubmitting = ref(false)
const submitError = ref<string | null>(null)
const showConfirmDialog = ref(false)

const isWatchdogDisabled = computed(() => watchdogStateStore.isDisabled)

const isRebootModalOpen = ref(false)
const isRebootSubmitting = ref(false)
const rebootError = ref<string | null>(null)
const rebootRequested = ref(false)
const rebootMethod = ref<AdminSystemRebootMethod>('soft')
const rebootCompletedKind = ref<AdminSystemRebootMethod | null>(null)

const isHardRebootSelected = computed(() => rebootMethod.value === 'hard')

function openRebootModal(): void {
  rebootError.value = null
  rebootRequested.value = false
  rebootCompletedKind.value = null
  rebootMethod.value = 'soft'
  isRebootModalOpen.value = true
}

function closeRebootModal(): void {
  if (isRebootSubmitting.value) {
    return
  }
  isRebootModalOpen.value = false
}

async function confirmReboot(): Promise<void> {
  isRebootSubmitting.value = true
  rebootError.value = null
  try {
    await requestSystemReboot(rebootMethod.value)
    rebootCompletedKind.value = rebootMethod.value
    rebootRequested.value = true
  } catch (error) {
    rebootError.value =
      error instanceof Error
        ? error.message
        : 'Не удалось отправить команду перезагрузки. Проверьте состояние watchdog.'
  } finally {
    isRebootSubmitting.value = false
  }
}

const isMaintenance = computed(() => kioskStateStore.maintenance)
const currentReason = computed(() => kioskStateStore.reason)
const updatedAt = computed(() => kioskStateStore.state?.updatedAt ?? '')
const currentScreen = computed(() => kioskStateStore.screen)
const screenCategory = computed(() => categorize(currentScreen.value))

const screenBadgeClass = computed(() => {
  switch (screenCategory.value) {
    case 'free':
      return 'bg-fuel-lime/15 border-fuel-lime/50 text-fuel-forest'
    case 'confirm':
      return 'bg-amber-50 border-amber-200 text-amber-700'
    case 'blocked':
      return 'bg-red-50 border-red-200 text-red-700'
    default:
      return 'bg-gray-100 border-gray-200 text-gray-500'
  }
})

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
        <div class="flex flex-wrap items-center gap-x-3 gap-y-1">
          <h2
            class="font-rubik font-bold text-3xl leading-tight"
            :class="isMaintenance ? 'text-amber-700' : 'text-fuel-forest'"
          >
            {{ isMaintenance ? 'Ведутся технические работы' : 'АЗС в работе' }}
          </h2>
          <span
            v-if="!isMaintenance"
            class="inline-block rounded-full border px-3 py-0.5 font-karla text-sm font-medium"
            :class="screenBadgeClass"
          >
            {{ screenLabel(currentScreen) }}
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

    <!-- Перезагрузка терминала -->
    <div class="bg-white rounded-2xl border border-fuel-olive/20 p-8 flex flex-col gap-5 shadow-sm">
      <div>
        <h3 class="font-rubik font-semibold text-xl text-fuel-forest mb-1">
          Перезагрузка терминала
        </h3>
        <p class="font-karla text-sm text-fuel-olive">
          Стандартная перезагрузка через ОС или аварийная через ESP32 watchdog.
        </p>
      </div>

      <button
        type="button"
        class="font-rubik font-semibold text-lg px-8 py-4 rounded-xl transition-all duration-200
               bg-red-600 text-white hover:bg-red-700 active:scale-95
               shadow-md shadow-red-400/25
               focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-offset-2
               focus-visible:ring-offset-white focus-visible:ring-red-500
               cursor-pointer"
        @click="openRebootModal"
      >
        Перезагрузить терминал
      </button>
    </div>

  </section>

  <!-- Диалог подтверждения при переводе с confirm-экранов -->
  <MaintenanceConfirmDialog
    v-if="showConfirmDialog"
    @confirm="onDialogConfirm"
    @cancel="onDialogCancel"
  />

  <!-- Модалка подтверждения перезагрузки -->
  <div
    v-if="isRebootModalOpen"
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 px-6"
    role="dialog"
    aria-modal="true"
  >
    <div class="bg-white rounded-2xl shadow-xl max-w-lg w-full p-8 flex flex-col gap-5">
      <div>
        <h3 class="font-rubik font-bold text-2xl text-fuel-forest mb-2">
          Перезагрузить терминал?
        </h3>
        <template v-if="!rebootRequested">
          <p class="font-karla text-base text-fuel-forest/80 mb-4">
            Выберите способ перезагрузки: обычный через команду ОС или аварийный через ESP32.
          </p>

          <div class="flex flex-col gap-3 font-karla text-sm">
            <label
              class="flex gap-3 items-start rounded-xl border p-4 cursor-pointer transition-colors"
              :class="
                rebootMethod === 'soft'
                  ? 'border-fuel-lime bg-fuel-lime/10'
                  : 'border-fuel-olive/25 hover:border-fuel-olive/50'
              "
            >
              <input
                v-model="rebootMethod"
                type="radio"
                value="soft"
                class="mt-1 h-4 w-4 accent-fuel-forest shrink-0"
              />
              <span>
                <span class="font-rubik font-semibold text-fuel-forest block">Обычная (команда ОС)</span>
                <span class="text-fuel-olive">
                  Использует стандартную команду перезагрузки на сервере.
                </span>
              </span>
            </label>

            <label
              class="flex gap-3 items-start rounded-xl border p-4 transition-colors"
              :class="
                isWatchdogDisabled
                  ? 'border-fuel-olive/15 bg-gray-50 opacity-60 cursor-not-allowed'
                  : rebootMethod === 'hard'
                    ? 'border-red-300 bg-red-50 cursor-pointer'
                    : 'border-fuel-olive/25 hover:border-red-200 cursor-pointer'
              "
            >
              <input
                v-model="rebootMethod"
                type="radio"
                value="hard"
                :disabled="isWatchdogDisabled"
                class="mt-1 h-4 w-4 accent-red-600 shrink-0 disabled:cursor-not-allowed"
              />
              <span>
                <span class="font-rubik font-semibold text-fuel-forest block">Аварийная (ESP32)</span>
                <span class="text-fuel-olive">
                  Принудительный reset через watchdog.
                </span>
              </span>
            </label>
          </div>

          <p v-if="isHardRebootSelected && !isWatchdogDisabled" class="font-karla text-xs text-red-700 mt-3">
            Внимание: это аварийный аппаратный сброс.
          </p>
        </template>

        <p v-else-if="rebootCompletedKind === 'soft'" class="font-karla text-base text-fuel-forest/80">
          Команда обычной перезагрузки отправлена. Сервер скоро уйдёт в перезагрузку,
          страница станет недоступна на короткое время.
        </p>
        <p v-else class="font-karla text-base text-fuel-forest/80">
          Команда аварийной перезагрузки через ESP32 отправлена.
          Страница станет недоступна на время перезапуска.
        </p>
      </div>

      <p v-if="rebootError" class="font-karla text-sm text-red-600">
        {{ rebootError }}
      </p>

      <div class="flex flex-col-reverse md:flex-row md:justify-end gap-3">
        <button
          type="button"
          :disabled="isRebootSubmitting"
          class="font-rubik font-medium text-base px-6 py-3 rounded-lg
                 border border-fuel-olive/40 text-fuel-forest
                 hover:bg-fuel-cream/60 transition-colors
                 disabled:cursor-not-allowed disabled:opacity-60 cursor-pointer"
          @click="closeRebootModal"
        >
          {{ rebootRequested ? 'Закрыть' : 'Отмена' }}
        </button>
        <button
          v-if="!rebootRequested"
          type="button"
          :disabled="isRebootSubmitting || (isHardRebootSelected && isWatchdogDisabled)"
          class="font-rubik font-semibold text-base px-6 py-3 rounded-lg
                 transition-all duration-200
                 disabled:cursor-not-allowed disabled:opacity-60 cursor-pointer"
          :class="
            isHardRebootSelected
              ? 'bg-red-600 text-white hover:bg-red-700 shadow-md shadow-red-400/25'
              : 'bg-fuel-forest text-white hover:bg-fuel-olive shadow-md shadow-fuel-forest/20'
          "
          @click="confirmReboot"
        >
          {{ isRebootSubmitting ? 'Отправляем...' : 'Выполнить перезагрузку' }}
        </button>
      </div>
    </div>
  </div>
</template>
