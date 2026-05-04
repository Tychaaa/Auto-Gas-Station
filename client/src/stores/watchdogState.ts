import { computed, ref } from 'vue'
import { defineStore } from 'pinia'

import { getWatchdogStatus, type AdminWatchdogStatus } from '@/api/admin.api'

const DEFAULT_POLL_INTERVAL_MS = 5000

// Стор состояния ESP32 watchdog для админки
// Сервер сам кэширует snapshot, поэтому поллим без боязни нагрузить serial-порт
export const useWatchdogStateStore = defineStore('watchdogState', () => {
  const state = ref<AdminWatchdogStatus | null>(null)
  const isLoading = ref(false)
  const loadError = ref<string | null>(null)

  let pollTimerId: ReturnType<typeof setInterval> | null = null

  const mode = computed(() => state.value?.mode ?? 'disabled')
  const isDisabled = computed(() => mode.value === 'disabled')
  const isOnline = computed(() => state.value?.online ?? false)
  const lastHeartbeatAgoMs = computed(() => state.value?.lastHeartbeatAgoMs ?? 0)
  const lastHeartbeatAt = computed(() => state.value?.lastHeartbeatAt ?? '')
  const espUptimeMs = computed(() => state.value?.espUptimeMs ?? 0)
  const lastError = computed(() => state.value?.lastError ?? '')

  async function refresh(): Promise<void> {
    isLoading.value = true
    try {
      state.value = await getWatchdogStatus()
      loadError.value = null
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Не удалось загрузить состояние watchdog'
      loadError.value = message
    } finally {
      isLoading.value = false
    }
  }

  function startPolling(intervalMs: number = DEFAULT_POLL_INTERVAL_MS): void {
    if (pollTimerId !== null) {
      return
    }
    void refresh()
    pollTimerId = setInterval(() => {
      void refresh()
    }, intervalMs)
  }

  function stopPolling(): void {
    if (pollTimerId === null) {
      return
    }
    clearInterval(pollTimerId)
    pollTimerId = null
  }

  return {
    state,
    mode,
    isDisabled,
    isOnline,
    lastHeartbeatAgoMs,
    lastHeartbeatAt,
    espUptimeMs,
    lastError,
    isLoading,
    loadError,
    refresh,
    startPolling,
    stopPolling,
  }
})
