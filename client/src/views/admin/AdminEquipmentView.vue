<script setup lang="ts">
import { computed, reactive, ref } from 'vue'

import {
  checkDispenser as checkDispenserApi,
  type AdminDispenserCheckResult,
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

  </section>
</template>
