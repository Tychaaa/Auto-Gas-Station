import { computed, ref } from 'vue'
import { defineStore } from 'pinia'

import { API_BASE_URL } from '@/api/http'
import { getKioskState } from '@/api/kiosk.api'
import type { KioskState } from '@/types/kioskState'

export const useKioskStateStore = defineStore('kioskState', () => {
  const state = ref<KioskState | null>(null)
  const isLoading = ref(false)
  const loadError = ref<string | null>(null)

  let es: EventSource | null = null

  const maintenance = computed(() => state.value?.maintenance ?? false)
  const reason = computed(() => state.value?.reason ?? '')
  const screen = computed(() => state.value?.screen ?? '')

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

  function connect(): void {
    if (es !== null) return

    es = new EventSource(`${API_BASE_URL}/kiosk/events`)

    es.onmessage = (event) => {
      try {
        applyState(JSON.parse(event.data) as KioskState)
      } catch {
        // ignore malformed events
      }
    }

    es.onerror = () => {
      loadError.value = 'Ошибка SSE-соединения с сервером'
    }
  }

  function disconnect(): void {
    if (es === null) return
    es.close()
    es = null
  }

  function applyState(next: KioskState): void {
    state.value = next
    loadError.value = null
  }

  return {
    state,
    maintenance,
    reason,
    screen,
    isLoading,
    loadError,
    refresh,
    connect,
    disconnect,
    applyState,
  }
})
