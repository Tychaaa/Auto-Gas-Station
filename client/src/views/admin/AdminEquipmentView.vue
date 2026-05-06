<script setup lang="ts">
import { computed, reactive, ref } from 'vue'

import {
  checkDispenser as checkDispenserApi,
  requestSystemReboot,
  type AdminDispenserCheckResult,
  type AdminSystemRebootMethod,
} from '@/api/admin.api'
import { useWatchdogStateStore } from '@/stores/watchdogState'

const watchdogStateStore = useWatchdogStateStore()

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

// --- Stub equipment check (KKT, Vendotek) ---

type StubStatus = 'idle' | 'checking' | 'no-data'

function useStubCheck() {
  const status = ref<StubStatus>('idle')

  function check() {
    status.value = 'checking'
    setTimeout(() => {
      status.value = 'no-data'
    }, 600)
  }

  return { status, check }
}

const kkt = useStubCheck()
const vendotek = useStubCheck()

// --- Dispenser real check ---

const dispenser = reactive({
  isChecking: false,
  result: null as AdminDispenserCheckResult | null,
  error: '',
})

async function checkDispenser(): Promise<void> {
  dispenser.isChecking = true
  dispenser.error = ''
  try {
    dispenser.result = await checkDispenserApi()
  } catch (e) {
    dispenser.error = e instanceof Error ? e.message : String(e)
  } finally {
    dispenser.isChecking = false
  }
}
</script>

<template>
  <section class="flex flex-col gap-8">
    <!-- ESP32 Watchdog -->
    <div class="bg-white rounded-2xl border border-fuel-olive/20 p-8 flex flex-col gap-5 shadow-sm">
      <div class="flex flex-col md:flex-row md:items-start md:justify-between gap-4">
        <div>
          <h3 class="font-rubik font-semibold text-xl text-fuel-forest mb-1">
            Watchdog ESP32
          </h3>
          <p class="font-karla text-sm text-fuel-olive">
            Автоматическая проверка каждые 5 секунд
          </p>
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

    <!-- Топливораздаточная колонка (АЗТ) -->
    <div class="bg-white rounded-2xl border border-fuel-olive/20 p-8 flex flex-col gap-5 shadow-sm">
      <div class="flex flex-col md:flex-row md:items-start md:justify-between gap-4">
        <div>
          <h3 class="font-rubik font-semibold text-xl text-fuel-forest mb-1">
            Топливораздаточная колонка
          </h3>
          <p class="font-karla text-sm text-fuel-olive">
            Подключение через протокол АЗТ (serial)
          </p>
        </div>

        <div
          v-if="dispenser.isChecking"
          class="inline-flex items-center gap-2 px-3 py-1.5 rounded-full bg-gray-100 border border-gray-300 animate-pulse"
        >
          <span class="h-2.5 w-2.5 rounded-full bg-gray-400" aria-hidden="true" />
          <span class="font-karla text-sm text-gray-700">Проверка…</span>
        </div>
        <div
          v-else-if="dispenser.result?.online"
          class="inline-flex items-center gap-2 px-3 py-1.5 rounded-full bg-fuel-lime/15 border border-fuel-lime/40"
        >
          <span class="h-2.5 w-2.5 rounded-full bg-fuel-lime" aria-hidden="true" />
          <span class="font-karla text-sm text-fuel-forest">На связи</span>
        </div>
        <div
          v-else-if="dispenser.result && !dispenser.result.online"
          class="inline-flex items-center gap-2 px-3 py-1.5 rounded-full bg-red-50 border border-red-200"
        >
          <span class="h-2.5 w-2.5 rounded-full bg-red-500 animate-pulse" aria-hidden="true" />
          <span class="font-karla text-sm text-red-700">Нет связи</span>
        </div>
        <div
          v-else
          class="inline-flex items-center gap-2 px-3 py-1.5 rounded-full bg-gray-100 border border-gray-300"
        >
          <span class="h-2.5 w-2.5 rounded-full bg-gray-400" aria-hidden="true" />
          <span class="font-karla text-sm text-gray-700">Не проверялось</span>
        </div>
      </div>

      <div
        v-if="dispenser.result"
        class="grid grid-cols-1 md:grid-cols-3 gap-4 text-sm font-karla"
      >
        <div class="rounded-lg border border-fuel-olive/20 bg-fuel-cream/30 p-4">
          <p class="text-xs uppercase tracking-widest text-fuel-olive mb-1">Статус колонки</p>
          <p class="text-fuel-forest font-medium">{{ dispenser.result.providerStatus || '—' }}</p>
        </div>
        <div class="rounded-lg border border-fuel-olive/20 bg-fuel-cream/30 p-4">
          <p class="text-xs uppercase tracking-widest text-fuel-olive mb-1">Код статуса / причины</p>
          <p class="text-fuel-forest font-medium font-mono">
            {{ dispenser.result.statusCode || '—' }} / {{ dispenser.result.reasonCode || '—' }}
          </p>
        </div>
        <div class="rounded-lg border border-fuel-olive/20 bg-fuel-cream/30 p-4">
          <p class="text-xs uppercase tracking-widest text-fuel-olive mb-1">Проверено</p>
          <p class="text-fuel-forest font-medium">{{ formatTimestamp(dispenser.result.checkedAt) }}</p>
        </div>
      </div>

      <p v-if="dispenser.result?.error" class="font-karla text-sm text-red-600">
        Ошибка: {{ dispenser.result.error }}
      </p>
      <p v-if="dispenser.error" class="font-karla text-sm text-red-600">
        {{ dispenser.error }}
      </p>

      <button
        type="button"
        :disabled="dispenser.isChecking"
        class="font-rubik font-semibold text-lg px-8 py-4 rounded-xl transition-all duration-200
               bg-fuel-lime text-white hover:bg-fuel-forest active:scale-95
               shadow-md shadow-fuel-lime/25
               focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-offset-2
               focus-visible:ring-offset-white focus-visible:ring-fuel-lime
               disabled:cursor-not-allowed disabled:opacity-60 cursor-pointer"
        @click="checkDispenser"
      >
        {{ dispenser.isChecking ? 'Проверка…' : 'Проверить' }}
      </button>
    </div>

    <!-- Онлайн-касса (KKT) -->
    <div class="bg-white rounded-2xl border border-fuel-olive/20 p-8 flex flex-col gap-5 shadow-sm">
      <div class="flex flex-col md:flex-row md:items-start md:justify-between gap-4">
        <div>
          <h3 class="font-rubik font-semibold text-xl text-fuel-forest mb-1">
            Онлайн-касса (KKT)
          </h3>
          <p class="font-karla text-sm text-fuel-olive">
            Фискальный регистратор PayOnline-01-ФА
          </p>
        </div>

        <div
          v-if="kkt.status.value === 'idle'"
          class="inline-flex items-center gap-2 px-3 py-1.5 rounded-full bg-gray-100 border border-gray-300"
        >
          <span class="h-2.5 w-2.5 rounded-full bg-gray-400" aria-hidden="true" />
          <span class="font-karla text-sm text-gray-700">Не проверялось</span>
        </div>
        <div
          v-else-if="kkt.status.value === 'checking'"
          class="inline-flex items-center gap-2 px-3 py-1.5 rounded-full bg-gray-100 border border-gray-300 animate-pulse"
        >
          <span class="h-2.5 w-2.5 rounded-full bg-gray-400" aria-hidden="true" />
          <span class="font-karla text-sm text-gray-700">Проверка…</span>
        </div>
        <div
          v-else
          class="inline-flex items-center gap-2 px-3 py-1.5 rounded-full bg-gray-100 border border-gray-300"
        >
          <span class="h-2.5 w-2.5 rounded-full bg-gray-400" aria-hidden="true" />
          <span class="font-karla text-sm text-gray-700">Нет данных</span>
        </div>
      </div>

      <p class="font-karla text-sm text-fuel-olive">
        Проверка состояния не реализована.
      </p>

      <button
        type="button"
        :disabled="kkt.status.value === 'checking'"
        class="font-rubik font-semibold text-lg px-8 py-4 rounded-xl transition-all duration-200
               bg-fuel-lime text-white hover:bg-fuel-forest active:scale-95
               shadow-md shadow-fuel-lime/25
               focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-offset-2
               focus-visible:ring-offset-white focus-visible:ring-fuel-lime
               disabled:cursor-not-allowed disabled:opacity-60 cursor-pointer"
        @click="kkt.check()"
      >
        Проверить
      </button>
    </div>

    <!-- Платёжный терминал (Vendotek) -->
    <div class="bg-white rounded-2xl border border-fuel-olive/20 p-8 flex flex-col gap-5 shadow-sm">
      <div class="flex flex-col md:flex-row md:items-start md:justify-between gap-4">
        <div>
          <h3 class="font-rubik font-semibold text-xl text-fuel-forest mb-1">
            Платёжный терминал
          </h3>
          <p class="font-karla text-sm text-fuel-olive">
            Vendotek — карточный эквайринг
          </p>
        </div>

        <div
          v-if="vendotek.status.value === 'idle'"
          class="inline-flex items-center gap-2 px-3 py-1.5 rounded-full bg-gray-100 border border-gray-300"
        >
          <span class="h-2.5 w-2.5 rounded-full bg-gray-400" aria-hidden="true" />
          <span class="font-karla text-sm text-gray-700">Не проверялось</span>
        </div>
        <div
          v-else-if="vendotek.status.value === 'checking'"
          class="inline-flex items-center gap-2 px-3 py-1.5 rounded-full bg-gray-100 border border-gray-300 animate-pulse"
        >
          <span class="h-2.5 w-2.5 rounded-full bg-gray-400" aria-hidden="true" />
          <span class="font-karla text-sm text-gray-700">Проверка…</span>
        </div>
        <div
          v-else
          class="inline-flex items-center gap-2 px-3 py-1.5 rounded-full bg-gray-100 border border-gray-300"
        >
          <span class="h-2.5 w-2.5 rounded-full bg-gray-400" aria-hidden="true" />
          <span class="font-karla text-sm text-gray-700">Нет данных</span>
        </div>
      </div>

      <p class="font-karla text-sm text-fuel-olive">
        Проверка состояния не реализована.
      </p>

      <button
        type="button"
        :disabled="vendotek.status.value === 'checking'"
        class="font-rubik font-semibold text-lg px-8 py-4 rounded-xl transition-all duration-200
               bg-fuel-lime text-white hover:bg-fuel-forest active:scale-95
               shadow-md shadow-fuel-lime/25
               focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-offset-2
               focus-visible:ring-offset-white focus-visible:ring-fuel-lime
               disabled:cursor-not-allowed disabled:opacity-60 cursor-pointer"
        @click="vendotek.check()"
      >
        Проверить
      </button>
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
