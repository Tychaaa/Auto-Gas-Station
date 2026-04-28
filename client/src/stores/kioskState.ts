import { computed, ref } from 'vue'
import { defineStore } from 'pinia'

import { getKioskState } from '@/api/kiosk.api'
import type { KioskState } from '@/types/kioskState'

const DEFAULT_POLL_INTERVAL_MS = 3000

// Стор состояния киоска: режим тех работ и причина
// Используется и оверлеем MaintenanceView (в киоске), и AdminDashboardView
export const useKioskStateStore = defineStore('kioskState', () => {
  const state = ref<KioskState | null>(null)
  const isLoading = ref(false)
  const loadError = ref<string | null>(null)

  let pollTimerId: ReturnType<typeof setInterval> | null = null

  const maintenance = computed(() => state.value?.maintenance ?? false)
  const reason = computed(() => state.value?.reason ?? '')

  async function refresh(): Promise<void> {
    isLoading.value = true
    try {
      state.value = await getKioskState()
      loadError.value = null
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Не удалось загрузить состояние киоска'
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

  function applyState(next: KioskState): void {
    state.value = next
    loadError.value = null
  }

  return {
    state,
    maintenance,
    reason,
    isLoading,
    loadError,
    refresh,
    startPolling,
    stopPolling,
    applyState,
  }
})
