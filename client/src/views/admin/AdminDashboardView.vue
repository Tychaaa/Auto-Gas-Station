<script setup lang="ts">
import { computed, ref } from 'vue'

import { requestSystemReboot, setMaintenance, type AdminSystemRebootMethod } from '@/api/admin.api'
import { useKioskStateStore } from '@/stores/kioskState'
import { useWatchdogStateStore } from '@/stores/watchdogState'

const kioskStateStore = useKioskStateStore()
const watchdogStateStore = useWatchdogStateStore()

const reasonInput = ref('')
const isSubmitting = ref(false)
const submitError = ref<string | null>(null)

const isMaintenance = computed(() => kioskStateStore.maintenance)
const currentReason = computed(() => kioskStateStore.reason)
const updatedAt = computed(() => kioskStateStore.state?.updatedAt ?? '')

const watchdogMode = computed(() => watchdogStateStore.mode)
const isWatchdogDisabled = computed(() => watchdogStateStore.isDisabled)
const isWatchdogOnline = computed(() => watchdogStateStore.isOnline)
const watchdogLastError = computed(() => watchdogStateStore.lastError)
const watchdogLastHeartbeatAt = computed(() => watchdogStateStore.lastHeartbeatAt)
const watchdogLastHeartbeatAgoMs = computed(() => watchdogStateStore.lastHeartbeatAgoMs)
const watchdogEspUptimeMs = computed(() => watchdogStateStore.espUptimeMs)

const isRebootModalOpen = ref(false)
const isRebootSubmitting = ref(false)
const rebootError = ref<string | null>(null)
const rebootRequested = ref(false)
const rebootMethod = ref<AdminSystemRebootMethod>('soft')
const rebootCompletedKind = ref<AdminSystemRebootMethod | null>(null)

const isHardRebootSelected = computed(() => rebootMethod.value === 'hard')

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

// Форматирует длительность в мс в человекочитаемый вид (3 с / 12 мин / 2 ч)
function formatDurationMs(ms: number): string {
  if (!Number.isFinite(ms) || ms <= 0) {
    return '—'
  }
  const seconds = Math.floor(ms / 1000)
  if (seconds < 60) {
    return `${seconds} с`
  }
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) {
    return `${minutes} мин`
  }
  const hours = Math.floor(minutes / 60)
  if (hours < 24) {
    return `${hours} ч ${minutes % 60} мин`
  }
  const days = Math.floor(hours / 24)
  return `${days} д ${hours % 24} ч`
}

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

// Отправляет запрос на аппаратную перезагрузку и не закрывает модалку,
// чтобы пользователь увидел сообщение что страница станет недоступна
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

    <!-- Карточка состояния ESP32 watchdog + кнопка аппаратной перезагрузки -->
    <div class="bg-white rounded-2xl border border-fuel-olive/20 p-8 flex flex-col gap-5 shadow-sm">
      <div class="flex flex-col md:flex-row md:items-start md:justify-between gap-4">
        <div>
          <h3 class="font-rubik font-semibold text-xl text-fuel-forest mb-1">
            Watchdog ESP32
          </h3>
        </div>

        <div
          v-if="isWatchdogDisabled"
          class="inline-flex items-center gap-2 px-3 py-1.5 rounded-full bg-gray-100 border border-gray-300"
        >
          <span class="h-2.5 w-2.5 rounded-full bg-gray-400" aria-hidden="true" />
          <span class="font-karla text-sm text-gray-700">Не настроен</span>
        </div>
        <div
          v-else-if="isWatchdogOnline"
          class="inline-flex items-center gap-2 px-3 py-1.5 rounded-full bg-fuel-lime/15 border border-fuel-lime/40"
        >
          <span class="h-2.5 w-2.5 rounded-full bg-fuel-lime" aria-hidden="true" />
          <span class="font-karla text-sm text-fuel-forest">На связи</span>
        </div>
        <div
          v-else
          class="inline-flex items-center gap-2 px-3 py-1.5 rounded-full bg-red-50 border border-red-200"
        >
          <span class="h-2.5 w-2.5 rounded-full bg-red-500 animate-pulse" aria-hidden="true" />
          <span class="font-karla text-sm text-red-700">Нет связи</span>
        </div>
      </div>

      <div
        v-if="!isWatchdogDisabled"
        class="grid grid-cols-1 md:grid-cols-3 gap-4 text-sm font-karla"
      >
        <div class="rounded-lg border border-fuel-olive/20 bg-fuel-cream/30 p-4">
          <p class="text-xs uppercase tracking-widest text-fuel-olive mb-1">Режим</p>
          <p class="text-fuel-forest font-medium">{{ watchdogMode }}</p>
        </div>
        <div class="rounded-lg border border-fuel-olive/20 bg-fuel-cream/30 p-4">
          <p class="text-xs uppercase tracking-widest text-fuel-olive mb-1">Последний heartbeat</p>
          <p class="text-fuel-forest font-medium">
            {{ formatTimestamp(watchdogLastHeartbeatAt) }}
          </p>
          <p class="text-xs text-fuel-olive mt-1">
            {{ watchdogLastHeartbeatAt ? `${formatDurationMs(watchdogLastHeartbeatAgoMs)} назад` : '' }}
          </p>
        </div>
        <div class="rounded-lg border border-fuel-olive/20 bg-fuel-cream/30 p-4">
          <p class="text-xs uppercase tracking-widest text-fuel-olive mb-1">Аптайм ESP32</p>
          <p class="text-fuel-forest font-medium">{{ formatDurationMs(watchdogEspUptimeMs) }}</p>
        </div>
      </div>

      <p v-if="!isWatchdogDisabled && watchdogLastError" class="font-karla text-sm text-red-600">
        Последняя ошибка обмена: {{ watchdogLastError }}
      </p>

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
      <p v-if="isWatchdogDisabled" class="font-karla text-xs text-fuel-olive">
        Аварийная перезагрузка через ESP32 недоступна, пока не настроен serial-watchdog.
      </p>
    </div>

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
  </section>
</template>
